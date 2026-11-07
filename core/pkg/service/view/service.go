// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/view/versions"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	// DB is the underlying database that the service will use to store Views.
	DB *gorp.DB
	// Signals is used to publish signals when views are created, updated, or deleted.
	Signals *signals.Provider
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config
func (c ServiceConfig) Validate() error {
	v := validate.New("view.service")
	v.NotNil("db", c.DB)
	return v.Error()
}

// Service is the main entrypoint for managing views within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting views. It also provides
// mechanisms for listening to changes in views.
type Service struct {
	cfg     ServiceConfig
	table   *gorp.Table[Key, View]
	indexes indexes
	closer  xio.MultiCloser
}

// OpenService opens a new Service with the provided configuration. If error is nil, the
// service is ready for use and must be closed by calling Close to prevent resource
// leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	s = &Service{indexes: newIndexes()}
	if s.cfg, err = config.New(ServiceConfig{}, cfgs...); err != nil {
		return nil, err
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, View]{
		DB:              s.cfg.DB,
		Indexes:         s.indexes.all(),
		Migrations:      versions.Migrations,
		Instrumentation: s.cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if s.cfg.Signals == nil {
		return s, nil
	}
	var sig io.Closer
	if sig, err = s.cfg.Signals.PublishFromGorp(
		ctx,
		signals.GorpPublisherConfigUUID(s.table.Observe()),
	); !ok(err, sig) {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases any resources that it may have acquired. Close
// is not safe to call concurrently with any other service methods (including Writer(s)
// and Retrieve(s)).
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a new Writer to create, update, and delete views. If tx is not nil,
// the writer will use it to execute all operations. If tx is nil, the writer will
// execute all operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    gorp.OverrideTx(s.cfg.DB, tx),
		table: s.table,
	}
}

// NewRetrieve opens a new Retrieve query to fetch views from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		baseTX:  s.cfg.DB,
		indexes: s.indexes,
	}
}
