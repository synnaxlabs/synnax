// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a symbol service.
type ServiceConfig struct {
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// DB is the database that the symbol service will store symbols in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Group creates and deletes the groups that hold symbols.
	//
	// [REQUIRED]
	Group *group.Service
	// Signals is used to propagate changes to symbols throughout the cluster.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// ImEx is the import/export registry the symbol service registers itself with as
	// the exporter for symbol resources during OpenService.
	//
	// [REQUIRED]
	ImEx *imex.Service
	// Enumeration holds the enumerations that symbol states show.
	//
	// [REQUIRED]
	Enumeration *enumeration.Service
	// Actuators returns the keys of the symbols that a custom_actuator node draws, for
	// the migration that gives them the commands of an actuator.
	//
	// [REQUIRED]
	Actuators func(ctx context.Context, tx gorp.Tx) (set.Set[Key], error)
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	c.Enumeration = override.Nil(c.Enumeration, other.Enumeration)
	c.Actuators = override.Nil(c.Actuators, other.Actuators)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("symbol")
	v.NotNil("db", c.DB)
	v.NotNil("group", c.Group)
	v.NotNil("imex", c.ImEx)
	v.NotNil("enumeration", c.Enumeration)
	v.NotNil("actuators", c.Actuators)
	return v.Error()
}

// Service is the primary service for retrieving and modifying symbols from Synnax.
type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Symbol]
	indexes indexes
	state   *actions.State[Key, Action]
}

// OpenService instantiates a new symbol service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the Config struct for information on which fields should be set.
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
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Symbol]{
		DB:         cfg.DB,
		Name:       tableName,
		Indexes:    s.indexes.all(),
		References: s.indexes.references(),
		Migrations: versions.NewMigrations(versions.MigrationsConfig{
			Intern: func(tx gorp.Tx) v5.Intern {
				return cfg.Enumeration.NewWriter(tx).NewInterner().Intern
			},
			Actuators: cfg.Actuators,
		}),
		// A named color delete or an enumeration retarget rewrites the symbol outside
		// any symbol write, so the service sends it as a create.
		OnRewrite: func(_ context.Context, tx gorp.Tx, sym Symbol) error {
			s.state.Dispatcher().
				Attach(tx, sym.Key, NewCreateAction(CreatePayload{Symbol: sym}))
			return nil
		},
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if err = cfg.ImEx.RegisterImportExporter(s); !ok(err, nil) {
		return nil, err
	}
	if cfg.Signals != nil {
		if err = cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, Symbol]{
			Changes:    s.table.Changes,
			Resource:   ResourceID,
			SetName:    "sy_schematic_symbol_set",
			DeleteName: "sy_schematic_symbol_delete",
		}); !ok(err, nil) {
			return nil, err
		}
		if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
			Name:     "sy_schematic_symbol_action_set",
			Actions:  s.state,
			Resource: ResourceID,
		}); !ok(err, nil) {
			return nil, err
		}
	}
	return s, nil
}

// NewWriter opens a new writer for creating, updating, and deleting symbols in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// NewRetrieve opens a new query build for retrieving symbols from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}

// Close closes the symbol service, shutting down signal publishers.
func (s *Service) Close() error { return s.closer.Close() }
