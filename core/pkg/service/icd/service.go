// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/library/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a library Service.
type ServiceConfig struct {
	// DB is the database the service stores libraries in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology defines libraries as resources and finds the tasks that use them.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Task rewrites the tasks that use a library when the library changes.
	//
	// [REQUIRED]
	Task *task.Service
	// Search is the search index for fuzzy searching libraries.
	//
	// [REQUIRED]
	Search *search.Index
	// ImEx is the import/export registry the service registers itself with.
	//
	// [REQUIRED]
	ImEx *imex.Service
	// Signals publishes library changes to clients.
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
	c.Task = override.Nil(c.Task, other.Task)
	c.Search = override.Nil(c.Search, other.Search)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("library")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("task", c.Task)
	v.NotNil("search", c.Search)
	v.NotNil("imex", c.ImEx)
	return v.Error()
}

// Service creates, retrieves, and deletes libraries.
type Service struct {
	cfg    ServiceConfig
	table  *gorp.Table[Key, Library]
	closer xio.MultiCloser
}

// OpenService opens a Service with the provided configuration. The service must be
// closed by calling Close.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	s = &Service{}
	if s.cfg, err = config.New(ServiceConfig{}, cfgs...); err != nil {
		return nil, err
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Library]{
		DB:              s.cfg.DB,
		Migrations:      versions.Migrations,
		Instrumentation: s.cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	s.cfg.Ontology.RegisterService(s)
	s.cfg.Search.RegisterService(s)
	if err = s.cfg.ImEx.RegisterImportExporter(s); !ok(err, nil) {
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

// Close releases the resources held by the service. Close is not safe to call
// concurrently with any other service method.
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a Writer that executes against tx, or directly against the database
// when tx is nil.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:        tx,
		otg:       s.cfg.Ontology,
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		table:     s.table,
		tasks:     s.cfg.Task,
	}
}

// NewRetrieve opens a query for libraries.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   s.table.NewRetrieve(),
		baseTX: s.cfg.DB,
		search: s.cfg.Search,
	}
}
