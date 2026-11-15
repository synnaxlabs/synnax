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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/icd/versions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
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
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("icd")
	v.NotNil("db", c.DB)
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
		Name:            tableName,
		Indexes:         s.indexes.all(),
		Migrations:      versions.Migrations,
		Instrumentation: s.cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if err = s.cfg.ImEx.RegisterImportExporter(s); !ok(err, nil) {
		return nil, err
	}
	if s.cfg.Signals == nil {
		return s, nil
	}
	if err = s.cfg.Signals.PublishActions(ctx, signals.ActionConfig[Key, Action]{
		Name:     "sy_icd_action_set",
		Actions:  s.state,
		Resource: ResourceID,
		Table:    tableName,
	}); !ok(err, nil) {
		return nil, err
	}
	if err = s.cfg.Signals.PublishTable(ctx, signals.TableConfig[Key, ICD]{
		Changes:     s.table.Changes,
		Resource:    ResourceID,
		SetDisabled: new(true),
	}); !ok(err, nil) {
		return nil, err
	}
	return s, nil
}

// Close releases the resources held by the service. Close is not safe to call
// concurrently with any other service method.
func (s *Service) Close() error { return s.closer.Close() }

// OnAction calls handler with each action batch that a transaction attaches, after the
// transaction commits. handler runs as a reader of gorp.DB.Observe does, so it must not
// commit on its own goroutine. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// NewWriter opens a Writer that creates, edits, and deletes ICDs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: tx, table: s.table, dispatcher: s.state.Dispatcher()}
}

// NewRetrieve opens a query for ICDs.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}
