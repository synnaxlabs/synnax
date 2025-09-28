// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	alamos.Instrumentation
	// DB is the underlying database that the service will use to store Statuses.
	DB *gorp.DB
	// Ontology will be used to create relationships between statuses (parent-child) and
	// with other resources within the Synnax cluster.
	Ontology *ontology.Ontology
	// Group is used to create the top level "Statuses" group that will be the default
	// parent of all statuses.
	Group *group.Service
	// Signals is used to publish signals when statuses are created, updated, or deleted.
	Signals *signals.Provider
	Label   *label.Service
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
	c.Label = override.Nil(c.Label, other.Label)
	return c
}

// Validate implements config.Config
func (c ServiceConfig) Validate() error {
	v := validate.New("status.service")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	validate.NotNil(v, "Group", c.Group)
	validate.NotNil(v, "Label", c.Label)
	return v.Error()
}

// Service is the main entrypoint for managing statuses within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting statuses. It also
// provides mechanisms for listening to changes in statuses.
type Service struct {
	cfg             ServiceConfig
	group           group.Group
	shutdownSignals io.Closer
	mu              struct {
		sync.RWMutex
		statuses map[string]status.Status[any]
	}
}

const groupName = "Statuses"

// OpenService opens a new status.Service with the provided configuration. If error
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
	s.mu.statuses = make(map[string]status.Status[any])
	cfg.Ontology.RegisterService(s)
	if cfg.Signals == nil {
		return
	}
	statusSignals, err := signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signals.GorpPublisherConfigString[Status](cfg.DB),
	)
	if err != nil {
		return
	}
	s.shutdownSignals = xio.MultiCloser{statusSignals}
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

// NewWriter opens a new Writer to create, update, and delete statuses. If tx is not nil,
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

// NewRetrieve opens a new Retrieve query to fetch statuses from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[string, Status](),
		baseTX: s.cfg.DB,
		otg:    s.cfg.Ontology,
		label:  s.cfg.Label,
	}
}
