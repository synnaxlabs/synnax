// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
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
	// Signals is used to publish signals when statuses are created, updated, or
	// deleted.
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
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "label", c.Label)
	return v.Error()
}

// Service is the main entrypoint for managing statuses within Synnax. It provides
// mechanisms for creating, retrieving, updating, and deleting statuses. It also
// provides mechanisms for listening to changes in statuses.
type Service struct {
	cfg             ServiceConfig
	group           group.Group
	shutdownSignals io.Closer
}

// OpenService opens a new status.Service with the provided configuration. If error is
// nil, the service is ready for use and must be closed by calling Close to prevent
// resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, "Statuses", ontology.RootID)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals == nil {
		return s, nil
	}
	signalsCfg := signals.GorpPublisherConfigString[Status[any]](cfg.DB)
	signalsCfg.SetName = "sy_status_set"
	signalsCfg.DeleteName = "sy_status_delete"
	if s.shutdownSignals, err = signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signalsCfg,
	); err != nil {
		return nil, err
	}
	return s, nil
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

// NewWriter opens a new Writer to create, update, and delete statuses. If tx is not
// nil, the writer will use it to execute all operations. If tx is nil, the writer will
// execute all operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer[any] { return NewWriter[any](s, tx) }

// NewRetrieve opens a new Retrieve query to fetch statuses from the database.
func (s *Service) NewRetrieve() Retrieve[any] { return NewRetrieve[any](s) }

func NewWriter[D any](s *Service, tx gorp.Tx) Writer[D] {
	return Writer[D]{
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otg:       s.cfg.Ontology,
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		group:     s.group,
	}
}

func NewRetrieve[D any](s *Service) Retrieve[D] {
	return Retrieve[D]{
		gorp:   gorp.NewRetrieve[string, Status[D]](),
		baseTX: s.cfg.DB,
		otg:    s.cfg.Ontology,
		label:  s.cfg.Label,
	}
}
