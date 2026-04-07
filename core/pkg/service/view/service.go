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

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	// DB is the underlying database that the service will use to store Views.
	DB *gorp.DB
	// Ontology will be used to create relationships between views (parent-child) and
	// with other resources within the Synnax cluster.
	Ontology *ontology.Ontology
	// Group is used to create the top level "Views" group that will be the default
	// parent of all views.
	Group *group.Service
	// Signals is used to publish signals when views are created, updated, or deleted.
	Signals *signals.Provider
	// Search is the search index for fuzzy searching views.
	// [REQUIRED]
	Search *search.Index
	alamos.Instrumentation
}

var (
	_                    config.Config[ServiceConfig] = (*ServiceConfig)(nil)
	DefaultServiceConfig                              = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config
func (c ServiceConfig) Validate() error {
	v := validate.New("view.service")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

// Service is the main entrypoint for managing views within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting views. It also provides
// mechanisms for listening to changes in views.
type Service struct {
	cfg    ServiceConfig
	group  group.Group
	table  *gorp.Table[uuid.UUID, View]
	closer xio.MultiCloser
}

// OpenService opens a new Service with the provided configuration. If error is nil, the
// service is ready for use and must be closed by calling Close to prevent resource
// leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	s = &Service{}
	if s.cfg, err = config.New(DefaultServiceConfig, cfgs...); err != nil {
		return nil, err
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.group, err = s.cfg.Group.CreateOrRetrieve(ctx, "Views", ontology.RootID); !ok(err, nil) {
		return nil, err
	}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[View]{
		DB:              s.cfg.DB,
		Instrumentation: s.cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	s.cfg.Ontology.RegisterService(s)
	s.cfg.Search.RegisterService(s)
	if s.cfg.Signals == nil {
		return s, nil
	}
	var sig io.Closer
	if sig, err = signals.PublishFromGorp(
		ctx,
		s.cfg.Signals,
		signals.GorpPublisherConfigUUID[View](s.table.Observe()),
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
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otg:       s.cfg.Ontology,
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		group:     s.group,
		table:     s.table,
	}
}

// NewRetrieve opens a new Retrieve query to fetch views from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   s.table.NewRetrieve(),
		baseTX: s.cfg.DB,
		search: s.cfg.Search,
	}
}
