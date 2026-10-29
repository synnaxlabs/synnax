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
	"io"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/versions"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the ranger.Service.
type ServiceConfig struct {
	// DB is the underlying database that the service will use to store Ranges.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology will be used to create relationships between ranges (parent-child) and
	// with other resources within the Synnax cluster.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create the top level "Ranges" group that will be the default
	// parent of all ranges.
	//
	// [REQUIRED]
	Group *group.Service
	// Signals is used to publish signals on channels when ranges are created, updated,
	// or deleted.
	//
	// [OPTIONAL] - Defaults to nil.
	Signals *signals.Provider
	// Label is the label service used to attach, remove, and query labels related to
	// changes.
	//
	// [REQUIRED]
	Label *label.Service
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
	v.NotNil("ontology", c.Ontology)
	v.NotNil("group", c.Group)
	v.NotNil("label", c.Label)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Label = override.Nil(c.Label, other.Label)
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
		DB:      cfg.DB,
		Indexes: s.indexes.all(),
		Migrations: versions.NewMigrations(versions.MigrationsConfig{
			Ontology:        cfg.Ontology,
			Group:           cfg.Group,
			Instrumentation: cfg.Instrumentation,
		}),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals == nil {
		return s, nil
	}
	var sig io.Closer
	if sig, err = cfg.Signals.PublishFromGorp(
		ctx,
		signals.GorpPublisherConfigUUID(s.table.Observe()),
	); !ok(err, sig) {
		return nil, err
	}
	if sig, err = cfg.Signals.PublishActions(
		ctx,
		signals.ActionPublisherConfig[Key, Action]{
			Observable: s.state,
			Name:       "range_action",
		},
	); !ok(err, sig) {
		return nil, err
	}
	return s, nil
}

// OnAction subscribes the given handler to the actions that Dispatch and Writer emit.
// The handler runs synchronously after the transaction that applied the actions
// commits. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// Dispatch applies the actions atomically to the range with the given key. Dispatches
// for the same range run one at a time, each in its own transaction, and the actions
// are broadcast after the commit. dispatchKey is carried onto the broadcast so the
// sender recognizes its echo. Dispatch returns an error wrapping validate.ErrValidation
// when an action is a create or the result fails to decode or validate, and an error
// wrapping query.ErrNotFound when no range has the key.
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
			return nil, s.NewWriter(tx).apply(ctx, key, acts)
		},
	)
}

// Close closes the service and releases any resources that it may have acquired. Close
// is not safe to call concurrently with any other Service methods (including Writer(s)
// and Retrieve(s)).
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a new Writer to create, update, and delete ranges. If tx is not nil,
// the writer will use it to execute all operations. If tx is nil, the writer will
// execute all operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         gorp.OverrideTx(s.cfg.DB, tx),
		otg:        s.cfg.Ontology,
		otgWriter:  s.cfg.Ontology.NewWriter(tx),
		label:      s.cfg.Label,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a new Retrieve query to fetch ranges from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		baseTX:  s.cfg.DB,
		indexes: s.indexes,
		label:   s.cfg.Label,
	}
}

// RetrieveParentKey returns the parent range key for the given range key. Returns
// query.ErrNotFound if the range has no parent.
func (s *Service) RetrieveParentKey(
	ctx context.Context,
	key Key,
	tx gorp.Tx,
) (Key, error) {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	var resources []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.ParentsTraverser).
		WhereTypes(ontology.ResourceTypeRange).
		ExcludeFieldData(true).
		Entries(&resources).
		Exec(ctx, tx); err != nil {
		return uuid.Nil(), err
	}
	if len(resources) == 0 {
		return uuid.Nil(), errors.Wrapf(
			query.ErrNotFound,
			"range %s has no parent",
			key,
		)
	}
	return KeyFromOntologyID(resources[0].ID)
}
