// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	alamos.Instrumentation
	DB      *gorp.DB
	Signals *signals.Provider
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("policy")
	v.NotNil("db", c.DB)
	return v.Error()
}

type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Role]
	indexes indexes
}

func (s *Service) Close() error { return s.closer.Close() }

func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg, indexes: newIndexes()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Role]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if cfg.Signals != nil {
		var sig io.Closer
		if sig, err = cfg.Signals.PublishFromGorp(
			ctx,
			signals.GorpPublisherConfigUUID(s.table.Observe()),
		); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) NewWriter(tx gorp.Tx, allowInternal bool) Writer {
	return Writer{
		tx:            gorp.OverrideTx(s.cfg.DB, tx),
		allowInternal: allowInternal,
		table:         s.table,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{baseTX: s.cfg.DB, gorp: s.table.NewRetrieve(), indexes: s.indexes}
}
