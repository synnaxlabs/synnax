// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ranger implements a service for managing ranges in a Synnax cluster. A range
// is a user defined region of time in a Synnax cluster. They act as a method for
// labeling and categorizing data.
package ranger

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/versions"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the ranger.Service.
type ServiceConfig struct {
	// DB is the underlying database that the service will use to store Ranges.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Group is used by a migration that folds legacy range groups into parent ranges.
	//
	// [REQUIRED]
	Group *group.Service
	// Signals is used to publish signals on channels when ranges are created, updated,
	// or deleted.
	//
	// [OPTIONAL] - Defaults to nil.
	Signals *signals.Provider
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("service.ranger")
	v.NotNil("db", c.DB)
	v.NotNil("group", c.Group)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Service is the main entrypoint for managing ranges within Synnax. It provides
// mechanisms for creating, deleting, and listening to changes in ranges.
type Service struct {
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Range]
	indexes indexes
	state   *actions.State[Key, Action]
	cfg     ServiceConfig
}

// OpenService opens a new ranger.Service with the provided configuration. If error is
// nil, the services is ready for use and must be closed by calling Close to prevent
// resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
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
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Range]{
		DB:         cfg.DB,
		Name:       tableName,
		Indexes:    s.indexes.all(),
		References: s.indexes.references(),
		Migrations: versions.NewMigrations(versions.MigrationsConfig{
			Group:           cfg.Group,
			Instrumentation: cfg.Instrumentation,
		}),
		// A label delete detaches the label from each range that holds it, which no
		// range write does, so the service sets the labels that the range keeps.
		OnDetach: func(_ context.Context, tx gorp.Tx, r Range) error {
			s.state.Dispatcher().Attach(tx, r.Key, labelsAction(r.Labels))
			return nil
		},
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if cfg.Signals == nil {
		return s, nil
	}
	if err = cfg.Signals.PublishTable(
		ctx,
		signals.TableConfig[Key, Range]{
			Changes:  s.table.Changes,
			Resource: ResourceID,
		},
	); !ok(err, nil) {
		return nil, err
	}
	if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
		Name:     "sy_range_action_set",
		Actions:  s.state,
		Resource: ResourceID,
	}); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

// OnAction calls handler with each action batch that a transaction attaches, after the
// transaction commits. handler runs as a reader of gorp.DB.Observe does, so it must not
// commit on its own goroutine. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// Close closes the service and releases any resources that it may have acquired. Close
// is not safe to call concurrently with any other Service methods (including Writer(s)
// and Retrieve(s)).
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a new Writer to create, update, and delete ranges in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new Retrieve query to fetch ranges from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}
