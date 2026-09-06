// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"
	stdio "io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a table service.
type ServiceConfig struct {
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// DB is the database that the table service will store tables in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between tables and other entities in the
	// Synnax resource graph.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Search is the search index for fuzzy searching tables.
	//
	// [REQUIRED]
	Search *search.Index
	// Signals is used to propagate changes to tables throughout the cluster. When nil,
	// the service does not broadcast action sequences and gorp delete events are not
	// published. Dispatch still applies actions to local state.
	//
	// [OPTIONAL] - Defaults to nil, which disables signal broadcasting.
	Signals *signals.Provider
	// ImEx is the import/export registry the table service registers itself with as the
	// exporter for table resources during OpenService.
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
	v := validate.New("table")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("search", c.Search)
	v.NotNil("imex", c.ImEx)
	return v.Error()
}

// Service is the primary service for retrieving and modifying tables from Synnax.
type Service struct {
	cfg    ServiceConfig
	closer io.MultiCloser
	table  *gorp.Table[Key, Table]
	state  *actions.State[Key, Action]
}

// OpenService instantiates a new table service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the Config struct for information on which fields should be set.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg, state: actions.NewState[Key, Action](cfg.DB)}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Table]{
		DB:              cfg.DB,
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if err = cfg.ImEx.RegisterImportExporter(s); !ok(err, nil) {
		return nil, err
	}
	if cfg.Signals != nil {
		var sig stdio.Closer
		if sig, err = actions.PublishSignals(ctx, actions.SignalsConfig[Key, Action]{
			Provider: cfg.Signals,
			State:    s.state,
			Name:     "table",
		}); !ok(err, sig) {
			return nil, err
		}
		deleteCfg := signals.GorpPublisherConfigUUID(s.table.Observe())
		deleteCfg.DisableSet = true
		if sig, err = cfg.Signals.PublishFromGorp(ctx, deleteCfg); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the table service and releases any resources.
func (s *Service) Close() error {
	return s.closer.Close()
}

// OnAction subscribes the given handler to the action stream emitted by Dispatch. The
// handler runs synchronously inside Dispatch after the underlying transaction commits.
// The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnAction(handler)
}

// Dispatch applies a sequence of actions atomically to the table with the given key.
// Dispatches for the same table run one at a time, each in its own transaction
// committed before the actions are notified, so two concurrent dispatches can never
// overwrite each other's edits. dispatchKey is a client-generated identifier carried
// verbatim onto the broadcast so the originating client can recognize its own echo.
func (s *Service) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	acts []Action,
) error {
	return s.state.Dispatch(ctx, key, dispatchKey, acts, func(tx gorp.Tx) error {
		return s.table.NewUpdate().Where(gorp.MatchKeys[Key, Table](key)).
			ChangeErr(func(_ gorp.Context, t Table) (Table, error) {
				return Reduce(t, acts...)
			}).Exec(ctx, tx)
	})
}

// NewWriter opens a new writer for creating, updating, and deleting tables in Synnax.
// If tx is provided, the writer will use that transaction. If tx is nil, the Writer
// will execute the operations directly on the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:         tx,
		otgWriter:  s.cfg.Ontology.NewWriter(tx),
		otg:        s.cfg.Ontology,
		tbl:        s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new query build for retrieving tables from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   s.table.NewRetrieve(),
		baseTX: s.cfg.DB,
	}
}
