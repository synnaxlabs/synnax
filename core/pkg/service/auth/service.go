// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/crypto/bcrypt"
)

type SecureCredentials struct {
	Username string
	Password []byte
}

// GorpKey implements gorp.Entry.
func (s SecureCredentials) GorpKey() string { return s.Username }

// SetOptions implements gorp.Entry.
func (SecureCredentials) SetOptions() []any { return nil }

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// DB is the [gorp.DB] to use for credential storage.
	//
	// [REQUIRED]
	DB *gorp.DB
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
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("auth.service")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

// Service is a Gorp-backed authenticator. Credentials are persisted as
// [SecureCredentials] entries in a single Gorp table. [SecureCredentials] is exported
// solely so that Gorp's type-name-derived key prefix remains stable across releases —
// renaming or unexporting it would orphan every existing on-disk credential row, so
// callers should treat it as an internal type. All [gorp.Tx] values passed to
// [Service.NewWriter] must be spawned from the same [gorp.DB] used to open the service.
type Service struct {
	cfg   ServiceConfig
	table *gorp.Table[string, SecureCredentials]
}

// OpenService opens a new [Service] with the given configurations.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[string, SecureCredentials]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases any resources.
func (s *Service) Close() error { return s.table.Close() }

// Authenticate validates the identity of the entity with the given credentials. It
// returns [ErrInvalidCredentials] only when creds.Username has no stored row or
// creds.Password does not match the stored hash. Any other failure (e.g. a storage
// error during the retrieve) is returned verbatim so callers can distinguish a
// transient system failure from a credential mismatch.
func (s *Service) Authenticate(ctx context.Context, tx gorp.Tx, creds Credentials) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	var stored SecureCredentials
	if err := s.table.NewRetrieve().
		Where(gorp.MatchKeys[string, SecureCredentials](creds.Username)).
		Entry(&stored).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return ErrInvalidCredentials
		}
		return err
	}
	if err := bcrypt.
		CompareHashAndPassword(stored.Password, []byte(creds.Password)); err != nil {
		return errors.Combine(ErrInvalidCredentials, err)
	}
	return nil
}

// NewWriter opens a new [Writer] using the provided transaction.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{service: s, tx: gorp.OverrideTx(s.cfg.DB, tx)}
}
