// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package grant stores grants. A [Grant] lets a rack read secrets that nothing on the
// rack uses yet, and ends when the session that created it ends.
package grant

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// DB stores grants.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Auth holds the sessions that grants end with.
	//
	// [REQUIRED]
	Auth *auth.Service
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
	c.Auth = override.Nil(c.Auth, other.Auth)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("grant")
	v.NotNil("db", c.DB)
	v.NotNil("auth", c.Auth)
	return v.Error()
}

// Service stores grants, and deletes the grants of each session that ends.
type Service struct {
	cfg        ServiceConfig
	table      *gorp.Table[Key, Grant]
	indexes    indexes
	disconnect observe.Disconnect
}

// OpenService opens a [Service] with the given configurations. If the error is nil, the
// Service must be closed with Close after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, indexes: newIndexes()}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Grant]{
		DB:              cfg.DB,
		Indexes:         s.indexes.all(),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	s.disconnect = cfg.Auth.Observe().OnChange(s.endWithSessions)
	return s, nil
}

// Close stops deleting the grants of ended sessions and releases the service's
// resources.
func (s *Service) Close() error {
	s.disconnect()
	return s.table.Close()
}

// NewRetrieve opens a query for grants.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), baseTX: s.cfg.DB, indexes: s.indexes}
}

// Get returns the grant with the given key. It returns query.ErrNotFound when no grant
// has the key or the session that created the grant has ended.
func (s *Service) Get(ctx context.Context, tx gorp.Tx, key Key) (Grant, error) {
	var g Grant
	if err := s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&g).
		Exec(ctx, tx); err != nil {
		return Grant{}, err
	}
	live, err := s.cfg.Auth.NewRetrieve().
		Where(auth.MatchKeys(g.Session)).
		Exists(ctx, tx)
	if err != nil {
		return Grant{}, err
	}
	if !live {
		return Grant{}, errors.Wrapf(
			query.ErrNotFound,
			"grant %s ended with its session",
			key,
		)
	}
	return g, nil
}

// NewWriter opens a [Writer] that runs in tx. If tx is nil, the Writer writes directly
// to the DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: gorp.OverrideTx(s.cfg.DB, tx)}
}

func (s *Service) endWithSessions(
	ctx context.Context,
	changes gorp.TxReader[auth.SessionKey, auth.Session],
) {
	var ended []auth.SessionKey
	for ch := range changes {
		if ch.Variant == change.VariantDelete {
			ended = append(ended, ch.Key)
		}
	}
	if len(ended) == 0 {
		return
	}
	if err := s.table.NewDelete().
		Where(MatchSessions(ended...)(s.NewRetrieve())).
		Exec(ctx, s.cfg.DB); err != nil {
		s.cfg.L.Error("failed to delete the grants of ended sessions", zap.Error(err))
	}
}
