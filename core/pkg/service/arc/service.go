// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	stdio "io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/stl"
	arcsymbol "github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	arcv54 "github.com/synnaxlabs/synnax/pkg/service/arc/migrations/v54"
	arcv56 "github.com/synnaxlabs/synnax/pkg/service/arc/migrations/v56"
	"github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Arc service.
type ServiceConfig struct {
	// DB is the database that the Arc service will store Arcs in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between Arcs and other entities in the
	// Synnax resource graph.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Channel is used for retrieving channel information from the cluster.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Task is used for deleting tasks associated with Arcs when Arcs are deleted.
	//
	// [REQUIRED]
	Task *task.Service
	// Search is the search index for fuzzy searching Arcs.
	//
	// [REQUIRED]
	Search *search.Index
	// Signals is used to broadcast collaborative-edit actions to the cluster.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// Instrumentation is used for logging, tracing, and metrics.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Search = override.Nil(c.Search, other.Search)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Task = override.Nil(c.Task, other.Task)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("arc")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "task", c.Task)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

// Service is the primary service for retrieving and modifying arcs from Synnax.
type Service struct {
	table  *gorp.Table[Key, Arc]
	closer xio.MultiCloser
	cfg    ServiceConfig
	// state owns the observer that collaborative-edit dispatches are broadcast through.
	state *actions.State[Key, Action]
}

// NewChannelResolver returns the dynamic resolver that the analyzer consults for
// cluster channels not statically known to the program.
func (s *Service) NewChannelResolver(tx gorp.Tx) arc.SymbolResolver {
	return s.cfg.Channel.NewArcSymbolResolver(tx)
}

// NewSymbolResolver is the dynamic resolver attached to a program root's
// GlobalResolver. It resolves cluster channels by name or numeric key. Static prelude
// symbols (STL, status module) are attached to the ambient by NewRoot rather than
// chained behind this resolver.
func (s *Service) NewSymbolResolver(tx gorp.Tx) arc.SymbolResolver {
	return s.NewChannelResolver(tx)
}

// NewRoot builds a program root populated with STL + status module + the cluster
// channel resolver attached as the dynamic resolver. This is the production analysis
// root: tx is consulted for channel lookups, nil means "use the service DB directly."
func (s *Service) NewRoot(tx gorp.Tx) *arcsymbol.Symbol {
	stlSyms := stl.NewSymbols()
	statusSyms := status.NewSymbols()
	syms := make([]*arcsymbol.Symbol, 0, len(stlSyms)+len(statusSyms))
	syms = append(syms, stlSyms...)
	syms = append(syms, statusSyms...)
	return arcsymbol.NewRoot(s.NewSymbolResolver(tx), syms)
}

func (s *Service) NewLSP() (*lsp.Server, error) {
	return lsp.New(lsp.Config{
		Instrumentation: s.cfg.Child("lsp"),
		NewRoot:         func() *arcsymbol.Symbol { return s.NewRoot(nil) },
		OnRename: func(
			ctx context.Context,
			sym *arcsymbol.Symbol,
			oldName,
			newName string,
		) error {
			if sym.Kind != arcsymbol.KindChannel {
				return nil
			}
			return s.cfg.Channel.NewWriter(nil).
				Rename(ctx, channel.Key(sym.ID), newName, false)
		},
		OnExternalChange: observe.Translator[gorp.TxReader[channel.Key, channel.Channel], struct{}]{
			Observable: s.cfg.Channel.Observe(),
			Translate: func(
				context.Context,
				gorp.TxReader[channel.Key, channel.Channel],
			) (struct{}, bool) {
				return struct{}{}, true
			},
		},
	})
}

func (s *Service) Close() error { return s.closer.Close() }

// CompileProgram retrieves an Arc program by key and compiles its Module. The returned
// Arc has its Module field populated with the compiled module.
func (s *Service) CompileProgram(ctx context.Context, key Key) (Arc, error) {
	var entry Arc
	err := s.NewRetrieve().Where(MatchKeys(key)).Entry(&entry).Exec(ctx, nil)
	if err != nil {
		return Arc{}, err
	}
	var prog arc.Program
	if entry.Mode == ModeText {
		prog, err = arc.CompileText(ctx, entry.Text, s.NewRoot(nil))
	} else {
		prog, err = arc.CompileGraph(ctx, entry.Graph, s.NewRoot(nil))
	}
	if err != nil {
		return Arc{}, err
	}
	entry.Program = &prog
	return entry, nil
}

// OpenService instantiates a new Arc service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the ConfigValues struct for information on which fields should be set.
func OpenService(ctx context.Context, configs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg, state: actions.NewState[Key, Action]()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Arc]{
		DB: cfg.DB,
		Migrations: []migrate.Migration{
			gorp.CodecMigration[Key, arcv54.Arc]("msgpack_to_orc"),
			migrate.WithAddedDeps(
				gorp.NewEntryMigration("v54_drop_program_status", arcv56.MigrateArc),
				"msgpack_to_orc",
			),
			migrate.WithAddedDeps(
				gorp.NewEntryMigration("v55_rename_set_status", arcv56.RenameSetStatus),
				"v54_drop_program_status",
			),
			migrate.WithAddedDeps(
				gorp.NewEntryMigration("v56_to_live", MigrateArc),
				"v55_rename_set_status",
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
			Name:     "arc",
		}); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

// Observe returns an observable that notifies callers of changes to Arc entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Arc]] {
	return s.table.Observe()
}

// NewWriter opens a new writer for creating, updating, and deleting Arcs in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer will
// execute the operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         gorp.OverrideTx(s.cfg.DB, tx),
		otg:        s.cfg.Ontology.NewWriter(tx),
		task:       s.cfg.Task.NewWriter(tx),
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new query builder for retrieving Arcs from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   s.table.NewRetrieve(),
		baseTX: s.cfg.DB,
		search: s.cfg.Search,
	}
}
