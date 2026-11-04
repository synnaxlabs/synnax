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
	"crypto/sha256"
	"crypto/subtle"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
	// committing spans each Create and Redeem through its commit: a Gorp transaction
	// does not stop a stale delete from removing a newer write.
	committing sync.Mutex
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

// Create stores a new invite for subject in place of any earlier one, and returns it
// with its token, the only copy of its secret. It commits on its own and runs one at a
// time with Redeem on a Core, so a redeem of the earlier invite cannot delete the new
// one. It makes no identity check: the caller authorizes the change. It returns a
// validation error when subject is invalid.
func (s *Service) Create(
	ctx context.Context,
	subject ontology.ID,
) (Invite, string, error) {
	i := Invite{
		Key:     keyOf(subject),
		Subject: subject,
		Expiry:  telem.NewTimeStamp(s.cfg.Clock.Now().Add(Lifetime)),
	}
	if err := i.Validate(); err != nil {
		return Invite{}, "", err
	}
	token, hash, err := auth.NewToken(prefix, i.Key)
	if err != nil {
		return Invite{}, "", err
	}
	i.Hash = hash
	s.committing.Lock()
	defer s.committing.Unlock()
	if err = s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return s.table.NewCreate().Entry(&i).Exec(ctx, tx)
	}); err != nil {
		return Invite{}, "", err
	}
	return i, token, nil
}

// keyOf is the same for every invite to subject, so a new invite overwrites the last
// one, even when two are created at once.
func keyOf(subject ontology.ID) Key {
	sum := sha256.Sum256([]byte(subject.String()))
	var k Key
	copy(k[:], sum[:16])
	k[6] = k[6]&0x0f | 0x80
	k[8] = k[8]&0x3f | 0x80
	return k
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
	s.committing.Lock()
	defer s.committing.Unlock()
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
