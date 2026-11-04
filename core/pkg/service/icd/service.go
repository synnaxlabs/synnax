// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/icd/versions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening an ICD Service.
type ServiceConfig struct {
	// DB is the database the service stores ICDs in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology defines ICDs as resources.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// ImEx is the import/export registry the service registers itself with.
	//
	// [REQUIRED]
	ImEx *imex.Service
	// Signals publishes the actions that edit ICDs on sy_icd_action_set and ICD deletes
	// on sy_icd_delete.
	//
	// [OPTIONAL] - Defaults to nil.
	Signals *signals.Provider
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("icd")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("imex", c.ImEx)
	return v.Error()
}

// Service creates, edits, retrieves, and deletes ICDs.
type Service struct {
	cfg     ServiceConfig
	table   *gorp.Table[Key, ICD]
	indexes indexes
	state   *actions.State[Key, Action]
	closer  xio.MultiCloser
}

// OpenService opens a Service with the provided configuration. The service must be
// closed by calling Close.
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
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, ICD]{
		DB:              s.cfg.DB,
		Indexes:         s.indexes.all(),
		Migrations:      versions.Migrations,
		Instrumentation: s.cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	s.cfg.Ontology.RegisterService(s)
	if err = s.cfg.ImEx.RegisterImportExporter(s); !ok(err, nil) {
		return nil, err
	}
	if s.cfg.Signals == nil {
		return s, nil
	}
	var sig io.Closer
	if sig, err = s.cfg.Signals.PublishActions(
		ctx,
		signals.ActionPublisherConfig[Key, Action]{
			Observable: s.state,
			Name:       "icd_action",
		},
	); !ok(err, sig) {
		return nil, err
	}
	deleteCfg := signals.GorpPublisherConfigUUID(s.table.Observe())
	deleteCfg.DisableSet = true
	if sig, err = s.cfg.Signals.PublishFromGorp(ctx, deleteCfg); !ok(err, sig) {
		return nil, err
	}
	return s, nil
}

// Close releases the resources held by the service. Close is not safe to call
// concurrently with any other service method.
func (s *Service) Close() error { return s.closer.Close() }

// ObserveActions returns an observable of the actions that Dispatch and Writer emit.
// Handlers run synchronously after the transaction that applied the actions commits.
func (s *Service) ObserveActions() observe.Observable[actions.Scoped[Key, Action]] {
	return s.state
}

// Dispatch applies the actions atomically to the ICD with the given key. Dispatches for
// the same ICD run one at a time, each in its own transaction, and the actions are
// broadcast after the commit. dispatchKey is carried onto the broadcast so the sender
// recognizes its echo. Problems that Check finds do not stop a dispatch. Dispatch
// returns an error wrapping validate.ErrValidation when an action is a create or the
// ICD cannot hold its value, a path-scoped validation error when the result breaks a
// rule of the ICD's schema, and an error wrapping query.ErrNotFound when no ICD has the
// key.
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

// NewWriter opens a Writer that executes against tx, or directly against the database
// when tx is nil.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:         tx,
		otgWriter:  s.cfg.Ontology.NewWriter(tx),
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

// NewRetrieve opens a query for ICDs.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		baseTX:  s.cfg.DB,
		indexes: s.indexes,
	}
}
