// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// DB stores the named colors.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Group creates the palette of the built-in named colors.
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
	v := validate.New("color")
	v.NotNil("db", c.DB)
	v.NotNil("group", c.Group)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Service stores named colors.
type Service struct {
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Named]
	indexes indexes
	state   *actions.State[Key, Action]
}

// OpenService opens a Service and creates each built-in named color that is missing.
// When the error is nil, the caller must close the service.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{
		indexes: newIndexes(),
		state:   actions.NewState[Key, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Named]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if err = s.createBuiltins(ctx, cfg.DB, cfg.Group); err != nil {
		return nil, err
	}
	return s, nil
}

// Close releases the resources of the service.
func (s *Service) Close() error { return s.closer.Close() }

// Changes returns the changes that c made to named colors.
func (s *Service) Changes(c gorp.Commit) gorp.TxReader[Key, Named] {
	return s.table.Changes(c)
}

// Actions returns the action batches that writers attach to their transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// NewRetrieve opens a query for named colors.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}

// NewWriter opens a Writer that changes named colors in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: tx, table: s.table, dispatcher: s.state.Dispatcher()}
}
