// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package invite stores one-time tokens that let a user link a provider account or set
// a first password without a session. A token is shown once, and only its hash is
// stored.
package invite

import (
	"context"
	"crypto/subtle"
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

// lifetime is how long an invite works after its creation.
const lifetime = 24 * time.Hour

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
		References:      s.indexes.references(),
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
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}

// NewWriter opens a [Writer] that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: tx}
}

// MatchLive returns a filter for invites that have not expired.
func (s *Service) MatchLive() Filter {
	now := s.now()
	return Match(func(_ gorp.Context, _ Retrieve, i *Invite) (bool, error) {
		return i.Expiry > now, nil
	})
}

// Check returns [auth.ErrInvalidCredentials] when token is malformed, unknown, used,
// or expired, and nil when Redeem would accept it now. It writes nothing.
func (s *Service) Check(ctx context.Context, token string) error {
	return s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		_, err := s.find(ctx, snap, token)
		return err
	})
}

// Redeem deletes the invite that token names and runs f with it, in one transaction
// that commits when f returns nil. Gorp transactions run one at a time, so a token
// works once. It returns [auth.ErrInvalidCredentials] for a malformed, unknown, used,
// or expired token, and otherwise any error from f, which leaves the invite in place.
func (s *Service) Redeem(
	ctx context.Context,
	token string,
	f func(tx gorp.Tx, i Invite) error,
) error {
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		i, err := s.find(ctx, tx, token)
		if err != nil {
			return err
		}
		if err := s.NewWriter(tx).Delete(ctx, i.Key); err != nil {
			return err
		}
		return f(tx, i)
	})
}

func (s *Service) find(
	ctx context.Context,
	snap gorp.Snapshot,
	token string,
) (Invite, error) {
	key, hash, err := auth.ParseToken(prefix, token)
	if err != nil {
		return Invite{}, auth.ErrInvalidCredentials
	}
	var i Invite
	if err := s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&i).
		Exec(ctx, snap); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return Invite{}, auth.ErrInvalidCredentials
		}
		return Invite{}, err
	}
	if subtle.ConstantTimeCompare(i.Hash, hash) != 1 || i.Expiry <= s.now() {
		return Invite{}, auth.ErrInvalidCredentials
	}
	return i, nil
}

func (s *Service) now() telem.TimeStamp { return telem.NewTimeStamp(s.cfg.Clock.Now()) }
