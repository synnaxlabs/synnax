// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	alamos.Instrumentation
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
}

var (
	_             config.Config[ServiceConfig] = (*ServiceConfig)(nil)
	DefaultConfig                              = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config
func (c ServiceConfig) Validate() error {
	v := validate.New("view.service")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	validate.NotNil(v, "Group", c.Group)
	return v.Error()
}

// Service is the main entrypoint for managing views within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting views. It also
// provides mechanisms for listening to changes in views.
type Service struct {
	cfg             ServiceConfig
	group           group.Group
	shutdownSignals io.Closer
}

const groupName = "Views"

// OpenService opens a new view.Service with the provided configuration. If error
// is nil, the service is ready for use and must be closed by calling Close to
// prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	if err != nil {
		return nil, err
	}
	s = &Service{
		cfg:   cfg,
		group: g,
	}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals == nil {
		return
	}
	viewSignals, err := signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signals.GorpPublisherConfigUUID[View](cfg.DB),
	)
	if err != nil {
		return
	}
	s.shutdownSignals = xio.MultiCloser{viewSignals}
	return
}

// Close closes the service and releases any resources that it may have acquired. Close
// is not safe to call concurrently with any other Service methods (including Writer(s)
// and Retrieve(s)).
func (s *Service) Close() error {
	if s.shutdownSignals != nil {
		return s.shutdownSignals.Close()
	}
	return nil
}

// NewWriter opens a new Writer to create, update, and delete views. If tx is not nil,
// the writer will use it to execute all operations. If tx is nil, the writer will execute
// all operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otg:       s.cfg.Ontology,
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		group:     s.group,
	}
}

// NewRetrieve opens a new Retrieve query to fetch views from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, View](),
		baseTX: s.cfg.DB,
		otg:    s.cfg.Ontology,
	}
}
