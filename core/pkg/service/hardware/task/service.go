// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for creating a Service.
type Config struct {
	alamos.Instrumentation
	// DB is the gorp database that tasks will be stored in.
	DB           *gorp.DB
	Ontology     *ontology.Ontology
	Group        *group.Service
	Rack         *rack.Service
	Signals      *signals.Provider
	HostProvider cluster.HostProvider
	Channel      *channel.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("workspace")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "rack", c.Rack)
	validate.NotNil(v, "hostProvider", c.HostProvider)
	return v.Error()
}

type Service struct {
	cfg             Config
	shutdownSignals io.Closer
	group           group.Group
}

const groupName = "Tasks"

func OpenService(ctx context.Context, configs ...Config) (s *Service, err error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	if err != nil {
		return
	}
	s = &Service{cfg: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	s.cleanupInternalOntologyResources(ctx)
	if cfg.Signals == nil {
		return
	}
	cdcS, err := signals.PublishFromGorp(ctx, cfg.Signals, signals.GorpPublisherConfigPureNumeric[Key, Task](cfg.DB, telem.Uint64T))
	if err != nil {
		return
	}
	s.shutdownSignals = cdcS

	return
}

// cleanupInternalOntologyResources purges existing internal task resources from the ontology.
// we want to hide internal tasks from the user.
func (s *Service) cleanupInternalOntologyResources(ctx context.Context) {
	var tasks []Task
	if err := s.NewRetrieve().WhereInternal(true).Entries(&tasks).Exec(ctx, nil); err != nil {
		s.cfg.L.Warn("unable to retrieve internal tasks for cleanup", zap.Error(err))
	}
	ids := make([]ontology.ID, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, OntologyID(t.Key))
	}
	if err := s.cfg.Ontology.NewWriter(nil).DeleteManyResources(ctx, ids); err != nil {
		s.cfg.L.Warn("unable to delete internal task resources", zap.Error(err))
	}
}

func (s *Service) Close() error {
	if s.shutdownSignals != nil {
		return s.shutdownSignals.Close()
	}
	return nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    gorp.OverrideTx(s.cfg.DB, tx),
		otg:   s.cfg.Ontology.NewWriter(tx),
		rack:  s.cfg.Rack.NewWriter(tx),
		group: s.group,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:    s.cfg.Ontology,
		baseTX: s.cfg.DB,
		gorp:   gorp.NewRetrieve[Key, Task](),
	}
}
