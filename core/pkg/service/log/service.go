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
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
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
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("log")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("imex", c.ImEx)
	return v.Error()
}

// Service is the primary service for retrieving and modifying logs from Synnax.
type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Log]
	indexes indexes
	state   *actions.State[Key, Action]
}

// OpenService instantiates a new log service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the Config struct for information on which fields should be set.
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
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Log]{
		DB:      cfg.DB,
		Name:    tableName,
		Indexes: s.indexes.all(),
		References: append(
			s.indexes.references(),
			gorp.NewReference(channelsReference()),
		),
		Migrations: append(
			slices.Clone(versions.Migrations),
			migrate.DropDangling(
				"v61_log_dangling_channels",
				cfg.DB,
				[]gorp.ReferenceConfig[Key, Log]{channelsReference()},
			),
		),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	if err = cfg.ImEx.RegisterImportExporter(s); !ok(err, nil) {
		return nil, err
	}
	if cfg.Signals != nil {
		var sig io.Closer
		if sig, err = cfg.Signals.PublishActions(
			ctx,
			signals.ActionPublisherConfig[Key, Action]{
				Observable: s.state,
				Name:       "log",
			},
		); !ok(err, sig) {
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

// Close closes the log service and releases any resources.
func (s *Service) Close() error { return s.closer.Close() }

// OnAction subscribes the given handler to the action stream emitted by Dispatch. The
// handler runs synchronously inside Dispatch after the underlying transaction commits.
// The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// Dispatch applies a sequence of actions atomically to the log with the given key.
// Dispatches for the same log run one at a time, each in its own transaction committed
// before the actions are notified, so two concurrent dispatches can never overwrite
// each other's edits. dispatchKey is a client-generated identifier carried verbatim
// onto the broadcast so the originating client can recognize its own echo.
// Returns a validation error, and stores nothing, if the actions leave the log invalid.
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
			return nil, s.table.NewUpdate().Where(gorp.MatchKeys[Key, Log](key)).
				ChangeErr(func(_ gorp.Context, l Log) (Log, error) {
					reduced, err := Reduce(l, acts...)
					if err != nil {
						return Log{}, err
					}
					if err = reduced.Validate(); err != nil {
						return Log{}, err
					}
					return reduced, nil
				}).Exec(ctx, tx)
		},
	)
}

// NewWriter opens a new writer for creating, updating, and deleting logs in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer will
// execute the operations directly on the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		svc:        s,
		tx:         tx,
		otgWriter:  s.cfg.Ontology.NewWriter(tx),
		otg:        s.cfg.Ontology,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new query build for retrieving logs from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		baseTX:  s.cfg.DB,
		indexes: s.indexes,
	}
}

// MoveGroups moves the logs in groups into the project top, in tx, after those groups
// move under top. It returns a validation error when one of the groups holds a log and
// top is not a project.
func (s *Service) MoveGroups(
	ctx context.Context,
	tx gorp.Tx,
	groups []group.Key,
	top *resource.ID,
) error {
	var moved []Log
	if err := s.NewRetrieve().
		Where(MatchGroups(groups...)).
		Entries(&moved).
		Exec(ctx, tx); err != nil || len(moved) == 0 {
		return err
	}
	p, err := project.TopParentProject(top, resource.TypeLog)
	if err != nil {
		return err
	}
	keys := lo.Map(moved, func(e Log, _ int) Key { return e.Key })
	return s.NewWriter(tx).setProject(ctx, *p, keys...)
}
