// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package grant stores grants. A [Grant] lets a subject read secrets once, until it
// expires.
package grant

import (
	"context"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

// DefaultTTL is the default [ServiceConfig.TTL].
const DefaultTTL = time.Minute

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// DB stores grants.
	//
	// [REQUIRED]
	DB *gorp.DB
	// TTL is how long a grant works when no read uses it.
	//
	// [OPTIONAL] - Defaults to [DefaultTTL].
	TTL time.Duration
	// Clock tells the time, for expiry.
	//
	// [OPTIONAL] - Defaults to the system clock.
	Clock xtime.Clock
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
	c.TTL = override.Numeric(c.TTL, other.TTL)
	c.Clock = override.Nil(c.Clock, other.Clock)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("grant")
	v.NotNil("db", c.DB)
	v.Positive("ttl", c.TTL)
	v.NotNil("clock", c.Clock)
	return v.Error()
}

// Service stores grants.
type Service struct {
	cfg   ServiceConfig
	table *gorp.Table[Key, Grant]
}

// OpenService opens a [Service] with the given configurations. If the error is nil, the
// Service must be closed with Close after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(
		ServiceConfig{TTL: DefaultTTL, Clock: xtime.Real},
		cfgs...,
	)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Grant]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// Close releases the service's resources.
func (s *Service) Close() error { return s.table.Close() }

// NewWriter opens a [Writer] that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer { return Writer{svc: s, tx: tx} }

// Redeem returns the grant with the given key and deletes it, so no later call can use
// it. Redeem runs in a transaction of its own, so two calls cannot both use one grant,
// and the caller must not hold a transaction or snapshot of the DB. It returns an error
// wrapping access.ErrDenied when no grant has the key, the grant has expired, or the
// grant is for a subject other than subject. A grant for another subject stays in
// place, and an expired grant is deleted.
func (s *Service) Redeem(
	ctx context.Context,
	key Key,
	subject resource.ID,
) (Grant, error) {
	var (
		g       Grant
		expired bool
	)
	if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		if err := s.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Grant](key)).
			Entry(&g).
			Exec(ctx, tx); err != nil {
			if errors.Is(err, query.ErrNotFound) {
				return errEnded(key)
			}
			return err
		}
		expired = g.ExpiresAt.Before(s.now())
		if !expired && g.Subject != subject {
			return errors.Wrapf(access.ErrDenied, "grant is not for %s", subject)
		}
		return s.table.NewDelete().
			Where(gorp.MatchKeys[Key, Grant](key)).
			Exec(ctx, tx)
	}); err != nil {
		return Grant{}, err
	}
	if expired {
		return Grant{}, errEnded(key)
	}
	return g, nil
}

func errEnded(key Key) error {
	return errors.Wrapf(access.ErrDenied, "grant %s has ended or does not exist", key)
}

func (s *Service) now() telem.TimeStamp { return telem.NewTimeStamp(s.cfg.Clock.Now()) }
