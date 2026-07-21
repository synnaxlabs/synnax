// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"
	stdio "io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/types/v5"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/types/v6"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a line plot service.
type ServiceConfig struct {
	// Instrumentation for logging, tracing, and metrics.
	alamos.Instrumentation
	// DB is the database that the line plot service will store line plots in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between line plots and other entities in
	// the Synnax resource graph.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Search is the search index for fuzzy searching line plots.
	// [REQUIRED]
	Search *search.Index
	// Signals is the optional cluster signals provider. When set, every
	// successful Writer.Dispatch broadcasts a ScopedAction onto the
	// sy_lineplot_set channel, and deletes flow through sy_lineplot_delete.
	Signals *signals.Provider
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a line plot
	// service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("lineplot")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

// Service is the primary service for retrieving and modifying line plots from Synnax.
type Service struct {
	ServiceConfig
	closer io.MultiCloser
	table  *gorp.Table[Key, LinePlot]
	state  *actions.State[Key, Action]
}

// OpenService instantiates a new line plot service using the provided configurations.
// Each configuration will be used as an override for the previous configuration in the
// list. See the Config struct for information on which fields should be set.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{ServiceConfig: cfg, state: actions.NewState[Key, Action]()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable[Key, LinePlot](ctx, gorp.TableConfig[Key, LinePlot]{
		DB: cfg.DB,
		Migrations: []migrate.Migration{
			gorp.CodecMigration[Key, v5.LinePlot]("msgpack_to_orc"),
			migrate.WithAddedDeps(
				gorp.NewEntryMigration[Key, Key, v5.LinePlot, LinePlot](
					"v55_lift_typed_lineplot",
					v6.MigrateLinePlot,
				),
				"msgpack_to_orc",
			),
		},
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if cfg.Signals != nil {
		var sig stdio.Closer
		if sig, err = actions.PublishSignals(ctx, actions.SignalsConfig[Key, Action]{
			Provider: cfg.Signals,
			State:    s.state,
			Name:     "lineplot",
		}); !ok(err, sig) {
			return nil, err
		}
		deleteCfg := signals.GorpPublisherConfigUUID(s.table.Observe())
		deleteCfg.DisableSet = true
		if sig, err = signals.PublishFromGorp(ctx, cfg.Signals, deleteCfg); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the line plot service and releases any resources that it may
// have acquired.
func (s *Service) Close() error { return s.closer.Close() }

// OnAction subscribes the given handler to the action stream emitted by
// Writer.Dispatch. The handler runs synchronously inside Dispatch after the
// underlying transaction commits. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnAction(handler)
}

// NewWriter opens a new writer for creating, updating, and deleting line plots in Synnax. If
// tx is nil, the writer will perform operations directly against the underlying gorp.DB provided
// to the line plot service. If tx is provided, the writer will use that transaction.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.DB, tx)
	return Writer{
		tx:         tx,
		otg:        s.Ontology.NewWriter(tx),
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new query builder for retrieving line plots from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   s.table.NewRetrieve(),
		baseTX: s.DB,
	}
}
