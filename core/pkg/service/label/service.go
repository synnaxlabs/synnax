// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the label service. ServiceConfig is provided
// to the OpenService method.
type ServiceConfig struct {
	// DB specifies the database that the label service will use to store and retrieve
	// labels.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is the ontology service that the label service will use to manage
	// resources and relationships between other objects.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create and manage a root group for holding all labels.
	//
	// [REQUIRED]
	Group *group.Service
	// Instrumentation for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("label")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("group", c.Group)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Service is the main entry point for managing labels within Synnax. It provides
// mechanisms for creating, deleting, retrieving, and listening to changes on labels.
type Service struct {
	cfg    ServiceConfig
	closer xio.MultiCloser
	table  *gorp.Table[Key, Label]
	search *gorp.SearchIndex[Key, Label]
	state  *actions.State[Key, Action]
}

// OpenService opens a new label service using the provided configuration. If error
// is nil, the service is ready for use and must be closed by calling Close in order
// to prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:    cfg,
		search: newSearchIndex(),
		state:  actions.NewState[Key, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Label]{
		DB:              cfg.DB,
		Indexes:         []gorp.Index[Key, Label]{s.search},
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

// Close closes the label service and releases any resources that it may have acquired.
// Close must be called when the service is no longer needed to prevent resource leaks.
func (s *Service) Close() error { return s.closer.Close() }

// Observe returns an observable that notifies callers of changes to label entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Label]] {
	return s.table.Observe()
}

// ObserveActions returns an observable of the actions that Dispatch and Writer emit.
// Handlers run synchronously after the transaction that applied the actions commits.
func (s *Service) ObserveActions() observe.Observable[actions.Scoped[Key, Action]] {
	return s.state
}

// Dispatch applies the actions atomically to the label with the given key. Dispatches
// for the same label run one at a time, each in its own transaction, and the actions
// are broadcast after the commit. dispatchKey is carried onto the broadcast so the
// sender recognizes its echo. Dispatch returns an error wrapping validate.ErrValidation
// when an action is a create or the result fails to decode or validate, and an error
// wrapping query.ErrNotFound when no label has the key.
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

// NewRetrieve opens a new Retrieve query to fetch labels.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		baseTx: s.cfg.DB,
		gorp:   s.table.NewRetrieve(),
		search: s.search,
	}
}

// NewWriter opens a new Writer to create, update, and delete labels. If tx is not nil
// the writer will use it, otherwise it will execute operations directly against the
// underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:         tx,
		otg:        s.cfg.Ontology.NewWriter(tx),
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}
