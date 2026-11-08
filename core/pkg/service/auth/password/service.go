// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package password authenticates subjects by username and password. It stores one
// [Password] per subject and implements [auth.Authenticator] for
// [auth.MethodPassword].
package password

import (
	"bytes"
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// DB stores passwords.
	//
	// [REQUIRED]
	DB *gorp.DB
	// FIPS restricts password hashing to PBKDF2, the one password hash NIST approves.
	// With it on, the service refuses to check a bcrypt or Argon2id hash.
	//
	// [OPTIONAL] - Defaults to false.
	FIPS *bool
	// Instrumentation is for logging, tracing, metrics, etc.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.FIPS = override.Nil(c.FIPS, other.FIPS)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("password")
	v.NotNil("db", c.DB)
	v.NotNil("fips", c.FIPS)
	return v.Error()
}

// Service stores passwords and checks proofs of them.
type Service struct {
	cfg      ServiceConfig
	table    *gorp.Table[Key, Password]
	indexes  indexes
	current  hasher
	accepted []verifier
}

var _ auth.Authenticator = (*Service)(nil)

// OpenService opens a [Service] with the given configurations.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{FIPS: new(false)}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, indexes: newIndexes()}
	s.current, s.accepted = hashers(*cfg.FIPS)
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Password]{
		DB:              cfg.DB,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases its resources.
func (s *Service) Close() error { return s.table.Close() }

// NewRetrieve opens a query for stored passwords.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), baseTX: s.cfg.DB, indexes: s.indexes}
}

// NewWriter opens a [Writer] that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: tx}
}

// Authenticate implements [auth.Authenticator]. An unknown username and a wrong
// password both return [auth.ErrInvalidCredentials]. A hash made by an algorithm the
// service does not accept also fails with it, and names the algorithm. A password
// whose hash is not current is stored again with the current algorithm.
func (s *Service) Authenticate(
	ctx context.Context,
	proof auth.Proof,
) (auth.Identity, error) {
	creds, ok := proof.Variant.(auth.PasswordProof)
	if !ok {
		return auth.Identity{}, errors.Wrapf(
			validate.ErrValidation,
			"password authenticator cannot check a %T proof",
			proof.Variant,
		)
	}
	p, h, err := s.verify(ctx, nil, creds)
	if err != nil {
		return auth.Identity{}, err
	}
	if h != s.current {
		if err := s.rehash(ctx, p.Key, p.Hash, creds.Password); err != nil {
			return auth.Identity{}, err
		}
	}
	return identityOf(p), nil
}

// Verify returns the identity that creds prove, reading through tx. It returns the
// errors of Authenticate, and it never stores a new hash, so it can run in an open
// transaction.
func (s *Service) Verify(
	ctx context.Context,
	tx gorp.Tx,
	creds auth.PasswordProof,
) (auth.Identity, error) {
	p, _, err := s.verify(ctx, tx, creds)
	return identityOf(p), err
}

func (s *Service) verify(
	ctx context.Context,
	tx gorp.Tx,
	creds auth.PasswordProof,
) (Password, verifier, error) {
	if err := creds.Validate(); err != nil {
		return Password{}, nil, err
	}
	var p Password
	if err := s.NewRetrieve().
		Where(MatchUsernames(creds.Username)).
		Entry(&p).
		Exec(ctx, tx); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return Password{}, nil, auth.ErrInvalidCredentials
		}
		return Password{}, nil, err
	}
	h, ok := s.verifierFor(p.Hash)
	if !ok {
		return Password{}, nil, errors.Wrap(
			auth.ErrInvalidCredentials,
			"the stored hash uses an algorithm this Core does not accept",
		)
	}
	if err := h.verify(p.Hash, creds.Password); err != nil {
		return Password{}, nil, err
	}
	return p, h, nil
}

// Hash is a password hash made outside any write transaction, since hashing is slow.
// Only Service.Hash and Service.HashProven make one.
type Hash struct {
	value []byte
	// proven is the identity whose password a proof matched before the hash, or nil.
	proven *auth.Identity
}

// Proven returns the identity whose password the caller proved before the hash, and
// false when no proof came before it.
func (h Hash) Proven() (auth.Identity, bool) {
	if h.proven == nil {
		return auth.Identity{}, false
	}
	return *h.proven, true
}

// check returns auth.ErrInvalidCredentials when a proof came before h and stored is no
// longer the password it proved.
func (h Hash) check(stored Password) error {
	if h.proven == nil ||
		(h.proven.Credential == stored.Key && h.proven.Version == stored.Created) {
		return nil
	}
	return errors.Wrap(
		auth.ErrInvalidCredentials,
		"the password changed after its proof",
	)
}

// Hash hashes plaintext with the current algorithm. It returns a validation error when
// plaintext is empty.
func (s *Service) Hash(plaintext string) (Hash, error) {
	if plaintext == "" {
		return Hash{}, errors.Wrap(validate.ErrValidation, "password: required")
	}
	value, err := s.current.hash(plaintext)
	return Hash{value: value}, err
}

// HashProven checks proof against the stored passwords, then hashes plaintext. A write
// of the hash returns auth.ErrInvalidCredentials when the proven password changes
// first. HashProven returns the errors of Verify and Hash.
func (s *Service) HashProven(
	ctx context.Context,
	proof auth.PasswordProof,
	plaintext string,
) (Hash, error) {
	identity, err := s.Verify(ctx, nil, proof)
	if err != nil {
		return Hash{}, err
	}
	h, err := s.Hash(plaintext)
	if err != nil {
		return Hash{}, err
	}
	h.proven = &identity
	return h, nil
}

func identityOf(p Password) auth.Identity {
	return auth.Identity{Subject: p.Subject, Credential: p.Key, Version: p.Created}
}

// Version implements [auth.Authenticator]. A password's version changes each time the
// password is set.
func (s *Service) Version(
	ctx context.Context,
	snap gorp.Snapshot,
	credential uuid.UUID,
) (telem.TimeStamp, error) {
	var p Password
	err := s.NewRetrieve().Where(MatchKeys(credential)).Entry(&p).Exec(ctx, snap)
	return p.Created, err
}

func (s *Service) verifierFor(hash []byte) (verifier, bool) {
	for _, h := range s.accepted {
		if h.made(hash) {
			return h, true
		}
	}
	return nil, false
}

// rehash leaves the record alone when its hash changed after the check.
func (s *Service) rehash(
	ctx context.Context,
	key Key,
	checked []byte,
	plaintext string,
) error {
	hash, err := s.current.hash(plaintext)
	if err != nil {
		return err
	}
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return s.table.NewUpdate().
			Where(gorp.MatchKeys[Key, Password](key)).
			Change(func(_ gorp.Context, p Password) Password {
				if bytes.Equal(p.Hash, checked) {
					p.Hash = hash
				}
				return p
			}).
			Exec(ctx, tx)
	})
}
