// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	alamos.Instrumentation
	Signals  *signals.Provider
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Group    *group.Service
	Search   *search.Index
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("workspace")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

type Service struct {
	cfg    ServiceConfig
	closer xio.MultiCloser
	table  *gorp.Table[uuid.UUID, Workspace]
	group  group.Group
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable[uuid.UUID, Workspace](ctx, gorp.TableConfig[Workspace]{
		DB:              cfg.DB,
		Migrations:      WorkspaceMigrations(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(ctx, "Workspaces", ontology.RootID); !ok(err, nil) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if cfg.Signals == nil {
		return s, nil
	}
	var sig io.Closer
	if sig, err = signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signals.GorpPublisherConfigUUID[Workspace](s.table.Observe()),
	); !ok(err, sig) {
		return nil, err
	}
	return s, nil
}

func (s *Service) Close() error { return s.closer.Close() }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    gorp.OverrideTx(s.cfg.DB, tx),
		otg:   s.cfg.Ontology.NewWriter(tx),
		group: s.group,
		table: s.table,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		search: s.cfg.Search,
		baseTX: s.cfg.DB,
		gorp:   s.table.NewRetrieve(),
	}
}
