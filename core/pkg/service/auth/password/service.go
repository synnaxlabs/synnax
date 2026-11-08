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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
	return Writer{svc: s, tx: gorp.OverrideTx(s.cfg.DB, tx)}
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

func identityOf(p Password) auth.Identity {
	return auth.Identity{Subject: p.Subject, Credential: p.Key, Version: p.Created}
}

// DeleteFor implements [auth.Authenticator].
func (s *Service) DeleteFor(
	ctx context.Context,
	tx gorp.Tx,
	subjects ...resource.ID,
) error {
	return s.NewWriter(tx).DeleteFor(ctx, subjects...)
}

// Version implements [auth.Authenticator]. A password's version changes each time the
// password is set.
func (s *Service) Version(
	ctx context.Context,
	tx gorp.Tx,
	credential uuid.UUID,
) (telem.TimeStamp, error) {
	var p Password
	err := s.NewRetrieve().Where(MatchKeys(credential)).Entry(&p).Exec(ctx, tx)
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
