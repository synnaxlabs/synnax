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
	"github.com/synnaxlabs/synnax/pkg/service/label/versions"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
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
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

// Service is the main entry point for managing labels within Synnax. It provides
// mechanisms for creating, deleting, retrieving, and listening to changes on labels.
type Service struct {
	cfg     ServiceConfig
	closer  xio.MultiCloser
	table   *gorp.Table[Key, Label]
	indexes indexes
	state   *actions.State[Key, Action]
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
		cfg:     cfg,
		indexes: newIndexes(),
		state:   actions.NewState[Key, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Label]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	return s, nil
}

// Close closes the label service and releases any resources that it may have acquired.
// Close must be called when the service is no longer needed to prevent resource leaks.
func (s *Service) Close() error { return s.closer.Close() }

// Changes returns the changes that c made to labels.
func (s *Service) Changes(
	ctx context.Context,
	c gorp.Commit,
) gorp.TxReader[Key, Label] {
	return s.table.Changes(ctx, c)
}

// Actions returns the action batches that Dispatch and Writer attach to their
// transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

// NewRetrieve opens a new Retrieve query to fetch labels.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		baseTX:  s.cfg.DB,
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}

// NewWriter opens a new Writer to create, update, and delete labels in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}
