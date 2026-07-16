// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v55 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v55"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a log service.
type ServiceConfig struct {
	// Instrumentation for logging, tracing, and metrics.
	alamos.Instrumentation
	// DB is the database that the log service will store logs in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between logs and other entities in the
	// Synnax resource graph.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Search is the search index for fuzzy searching logs.
	//
	// [REQUIRED]
	Search *search.Index
	// Signals is the optional cluster signals provider. When set, every successful
	// Writer.Dispatch broadcasts a ScopedAction onto the sy_log_set channel, and
	// deletes flow through sy_log_delete.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// ImEx is the import/export registry that the log service registers itself with as
	// the importer/exporter for log resources during OpenService.
	//
	// [REQUIRED]
	ImEx *imex.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("log")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "search", c.Search)
	validate.NotNil(v, "imex", c.ImEx)
	return v.Error()
}

// Service is the primary service for retrieving and modifying logs from Synnax.
type Service struct {
	cfg    ServiceConfig
	closer xio.MultiCloser
	table  *gorp.Table[Key, Log]
	state  *actions.State[Key, Action]
}

// OpenService instantiates a new log service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the Config struct for information on which fields should be set.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg, state: actions.NewState[Key, Action]()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Log]{
		DB: cfg.DB,
		Migrations: []migrate.Migration{
			gorp.CodecMigration[Key, v55.Log]("msgpack_to_orc"),
			migrate.WithAddedDeps(
				gorp.NewEntryMigration("v55_lift_typed_log", MigrateLog),
				"msgpack_to_orc",
			),
		},
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	cfg.ImEx.RegisterImportExporter(s)
	if cfg.Signals != nil {
		var sig io.Closer
		if sig, err = actions.PublishSignals(ctx, actions.SignalsConfig[Key, Action]{
			Provider: cfg.Signals,
			State:    s.state,
			Name:     "log",
		}); !ok(err, sig) {
			return nil, err
		}
		deleteCfg := signals.GorpPublisherConfigUUID(s.table.Observe())
		deleteCfg.DisableSet = true
		if sig, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			deleteCfg,
		); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the log service and releases any resources.
func (s *Service) Close() error { return s.closer.Close() }

// OnAction subscribes the given handler to the action stream emitted by
// Writer.Dispatch. The handler runs synchronously inside Dispatch after the underlying
// transaction commits. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnAction(handler)
}

// NewWriter opens a new writer for creating, updating, and deleting logs in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer will
// execute the operations directly on the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:         tx,
		otgWriter:  s.cfg.Ontology.NewWriter(tx),
		otg:        s.cfg.Ontology,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new query build for retrieving logs from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), baseTX: s.cfg.DB}
}
