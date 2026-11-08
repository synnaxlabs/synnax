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

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a line plot service.
type ServiceConfig struct {
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// DB is the database that the line plot service will store line plots in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Signals is the cluster signals provider. When set, every successful
	// Writer.Dispatch broadcasts a ScopedAction onto the sy_lineplot_set channel, and
	// deletes flow through sy_lineplot_delete.
	//
	// [OPTIONAL] - Defaults to nil, which disables signal broadcasting.
	Signals *signals.Provider
	// ImEx is the import/export registry the line plot service registers itself with as
	// the exporter for line plot resources during OpenService.
	//
	// [REQUIRED]
	ImEx *imex.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("lineplot")
	v.NotNil("db", c.DB)
	v.NotNil("imex", c.ImEx)
	return v.Error()
}

// Service is the primary service for retrieving and modifying line plots from Synnax.
type Service struct {
	cfg     ServiceConfig
	closer  io.MultiCloser
	table   *gorp.Table[Key, LinePlot]
	indexes indexes
	state   *actions.State[Key, Action]
}

// OpenService instantiates a new line plot service using the provided configurations.
// Each configuration will be used as an override for the previous configuration in the
// list. See the Config struct for information on which fields should be set.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:     cfg,
		state:   actions.NewState[Key, Action](cfg.DB),
		indexes: newIndexes(),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, LinePlot]{
		DB:      cfg.DB,
		Name:    tableName,
		Indexes: s.indexes.all(),
		References: append(
			s.indexes.references(),
			channelsReference(),
		),
		Migrations: newMigrations(),
		// A channel delete detaches the channel from the line plot, which no line plot
		// write does, so the service sends the line plot as a create.
		OnDetach: func(_ context.Context, tx gorp.Tx, p LinePlot) error {
			s.state.Dispatcher().
				Attach(tx, p.Key, NewCreateAction(CreatePayload{LinePlot: p}))
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
		if err = cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
			Name:    "sy_lineplot_set",
			Actions: s.state,
			Table:   tableName,
		}); !ok(err, nil) {
			return nil, err
		}
		if err = cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, LinePlot]{
			Changes:     s.table.Changes,
			SetDisabled: new(true),
		}); !ok(err, nil) {
			return nil, err
		}
	}
	return s, nil
}

// Close closes the line plot service and releases any resources that it may
// have acquired.
func (s *Service) Close() error { return s.closer.Close() }

// OnAction calls handler with each action batch that a transaction attaches, after the
// transaction commits. handler runs as a reader of gorp.DB.Observe does, so it must not
// commit on its own goroutine. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// Dispatch applies a sequence of actions atomically to the line plot with the given
// key. Dispatches for the same line plot run one at a time, each in its own transaction
// committed before the actions are notified, so two concurrent dispatches can never
// overwrite each other's edits. dispatchKey is a client-generated identifier carried
// verbatim onto the broadcast so the originating client can recognize its own echo.
// Returns a validation error, and stores nothing, if the actions leave the line plot
// invalid.
func (s *Service) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	acts []Action,
) error {
	return s.state.Dispatch(
		ctx,
		key,
		dispatchKey,
		acts,
		func(tx gorp.Tx) ([]Action, error) {
			return nil, s.table.NewUpdate().Where(gorp.MatchKeys[Key, LinePlot](key)).
				ChangeErr(func(_ gorp.Context, p LinePlot) (LinePlot, error) {
					reduced, err := Reduce(p, acts...)
					if err != nil {
						return LinePlot{}, err
					}
					if err = reduced.Validate(); err != nil {
						return LinePlot{}, err
					}
					return reduced, nil
				}).Exec(ctx, tx)
		},
	)
}

// NewWriter opens a new writer for creating, updating, and deleting line plots in
// Synnax. If tx is provided, the writer uses that transaction; if tx is nil, it
// executes operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		svc:        s,
		tx:         tx,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new query builder for retrieving line plots from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		baseTX:  s.cfg.DB,
		indexes: s.indexes,
	}
}

// MoveGroups moves the line plots in groups into the project top, in tx, after those
// groups move under top. It returns a validation error when one of the groups holds a
// line plot and top is not a project.
func (s *Service) MoveGroups(
	ctx context.Context,
	tx gorp.Tx,
	groups []group.Key,
	top *resource.ID,
) error {
	var moved []LinePlot
	if err := s.NewRetrieve().
		Where(MatchGroups(groups...)).
		Entries(&moved).
		Exec(ctx, tx); err != nil || len(moved) == 0 {
		return err
	}
	p, err := project.TopParentProject(top, resource.TypeLineplot)
	if err != nil {
		return err
	}
	keys := lo.Map(moved, func(e LinePlot, _ int) Key { return e.Key })
	return s.NewWriter(tx).setProject(ctx, *p, keys...)
}
