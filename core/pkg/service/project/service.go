// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/project/versions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	alamos.Instrumentation
	Signals *signals.Provider
	DB      *gorp.DB
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("project")
	v.NotNil("db", c.DB)
	return v.Error()
}

type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Project]
	indexes indexes
	state   *actions.State[Key, Action]
}

func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:     cfg,
		indexes: newIndexes(),
		state:   actions.NewState[Key, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Project]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if cfg.Signals == nil {
		return s, nil
	}
	if err = cfg.Signals.PublishTable(
		ctx,
		signals.TableConfig[Key, Project]{
			Changes:  s.table.Changes,
			Resource: ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
		Name:     "sy_project_action_set",
		Actions:  s.state,
		Resource: ResourceID,
	}); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

func (s *Service) Close() error { return s.closer.Close() }

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// Children returns the ID of every record that references the project with key, as seen
// by snap: the groups directly under it and every visualization it holds.
func (s *Service) Children(
	ctx context.Context,
	snap gorp.Snapshot,
	key Key,
) ([]resource.ID, error) {
	return resource.Dependents(ctx, snap, s.table, key)
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		indexes: s.indexes,
		gorp:    s.table.NewRetrieve(),
	}
}
