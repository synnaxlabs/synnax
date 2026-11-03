// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package invite stores one-time tokens that let a provider account link to a user
// without a session. A token is shown once, and only its hash is stored.
package invite

import (
	"context"
	"crypto/subtle"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

// Lifetime is how long an invite works after its creation.
const Lifetime = 24 * time.Hour

const prefix = "syi_"

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// DB stores invites.
	//
	// [REQUIRED]
	DB *gorp.DB
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
	c.Clock = override.Nil(c.Clock, other.Clock)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("invite")
	v.NotNil("db", c.DB)
	v.NotNil("clock", c.Clock)
	return v.Error()
}

// Service stores invites and redeems their tokens.
type Service struct {
	cfg     ServiceConfig
	table   *gorp.Table[Key, Invite]
	indexes indexes
	// redeeming spans the commit: a Gorp transaction does not stop two others from
	// reading and deleting one invite.
	redeeming sync.Mutex
}

// OpenService opens a [Service] with the given configurations.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{Clock: xtime.Real}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, indexes: newIndexes()}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Invite]{
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

// NewRetrieve opens a query for stored invites.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), baseTX: s.cfg.DB, indexes: s.indexes}
}

// NewWriter opens a [Writer] that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: gorp.OverrideTx(s.cfg.DB, tx)}
}

// MatchLive returns a filter for invites that have not expired.
func (s *Service) MatchLive() Filter {
	now := s.now()
	return Match(func(_ gorp.Context, _ Retrieve, i *Invite) (bool, error) {
		return i.Expiry > now, nil
	})
}

// Redeem deletes the invite that token names and runs f with it, in one transaction
// that commits when f returns nil. Redeems run one at a time on a Core, so a token
// works once. It returns [auth.ErrInvalidCredentials] for a malformed, unknown, used,
// or expired token, and otherwise any error from f, which leaves the invite in place.
func (s *Service) Redeem(
	ctx context.Context,
	token string,
	f func(tx gorp.Tx, i Invite) error,
) error {
	key, hash, err := auth.ParseToken(prefix, token)
	if err != nil {
		return auth.ErrInvalidCredentials
	}
	s.redeeming.Lock()
	defer s.redeeming.Unlock()
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var i Invite
		if err := s.NewRetrieve().
			Where(MatchKeys(key)).
			Entry(&i).
			Exec(ctx, tx); err != nil {
			if errors.Is(err, query.ErrNotFound) {
				return auth.ErrInvalidCredentials
			}
			return err
		}
		if subtle.ConstantTimeCompare(i.Hash, hash) != 1 || i.Expiry <= s.now() {
			return auth.ErrInvalidCredentials
		}
		if err := s.NewWriter(tx).Delete(ctx, key); err != nil {
			return err
		}
		return f(tx, i)
	})
}

func (s *Service) now() telem.TimeStamp { return telem.NewTimeStamp(s.cfg.Clock.Now()) }
