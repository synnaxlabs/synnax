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
	"iter"
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/auth/versions"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/crypto/bcrypt"
)

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
	v.NotNil("db", c.DB)
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
	table *gorp.Table[Key, SecureCredentials]
}

// OpenService opens a new [Service] with the given configurations.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, SecureCredentials]{
		DB:              cfg.DB,
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases any resources.
func (s *Service) Close() error { return s.table.Close() }

// Authenticate returns the key of the user whose credentials match creds. It returns
// [ErrInvalidCredentials] only when no credentials hold creds.Username or
// creds.Password
// does not match the stored hash. Any other failure (e.g. a storage error during the
// retrieve) is returned verbatim so callers can distinguish a transient system failure
// from a credential mismatch.
func (s *Service) Authenticate(
	ctx context.Context,
	tx gorp.Tx,
	creds Credentials,
) (Key, error) {
	if err := creds.Validate(); err != nil {
		return uuid.Nil(), err
	}
	var stored SecureCredentials
	if err := s.table.NewRetrieve().
		Where(matchUsernames(creds.Username)).
		Entry(&stored).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return uuid.Nil(), ErrInvalidCredentials
		}
		return uuid.Nil(), err
	}
	if err := bcrypt.
		CompareHashAndPassword(stored.Password, []byte(creds.Password)); err != nil {
		return uuid.Nil(), errors.Combine(ErrInvalidCredentials, err)
	}
	return stored.Key, nil
}

// Usernames returns the username of every user that holds credentials.
func (s *Service) Usernames(ctx context.Context, tx gorp.Tx) (map[Key]string, error) {
	var stored []SecureCredentials
	if err := s.table.NewRetrieve().
		Entries(&stored).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil {
		return nil, err
	}
	return usernamesOf(stored), nil
}

// UsernamesByKey returns the username of each user in keys that holds credentials.
func (s *Service) UsernamesByKey(
	ctx context.Context,
	tx gorp.Tx,
	keys ...Key,
) (map[Key]string, error) {
	if len(keys) == 0 {
		return map[Key]string{}, nil
	}
	var stored []SecureCredentials
	// A key without credentials is a user that does not sign in with a password.
	if err := s.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, SecureCredentials](keys...)).
		Entries(&stored).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil &&
		!errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	return usernamesOf(stored), nil
}

func usernamesOf(stored []SecureCredentials) map[Key]string {
	usernames := make(map[Key]string, len(stored))
	for _, c := range stored {
		usernames[c.Key] = c.Username
	}
	return usernames
}

// KeysByUsername returns the keys of the users whose credentials hold any of the given
// usernames. Usernames that no credentials hold are skipped.
func (s *Service) KeysByUsername(
	ctx context.Context,
	tx gorp.Tx,
	usernames ...string,
) ([]Key, error) {
	var stored []SecureCredentials
	if err := s.table.NewRetrieve().
		Where(matchUsernames(usernames...)).
		Entries(&stored).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil {
		return nil, err
	}
	return lo.Map(stored, func(c SecureCredentials, _ int) Key { return c.Key }), nil
}

// OnChange calls f after each commit with the keys of the users whose credentials the
// commit set.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[Key]),
) observe.Disconnect {
	return s.table.Observe().OnChange(
		func(ctx context.Context, reader gorp.TxReader[Key, SecureCredentials]) {
			f(ctx, func(yield func(Key) bool) {
				for ch := range reader {
					if ch.Variant == change.VariantSet && !yield(ch.Key) {
						return
					}
				}
			})
		},
	)
}

func matchUsernames(usernames ...string) gorp.Filter[Key, SecureCredentials] {
	return gorp.Match(func(_ gorp.Context, c *SecureCredentials) (bool, error) {
		return slices.Contains(usernames, c.Username), nil
	})
}

// NewWriter opens a new [Writer] using the provided transaction.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{service: s, tx: gorp.OverrideTx(s.cfg.DB, tx)}
}
