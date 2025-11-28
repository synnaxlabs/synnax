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
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for creating a Service.
type Config struct {
	alamos.Instrumentation
	// DB is the gorp database that tasks will be stored in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between tasks and other resources
	// in the Synnax cluster.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create task related groups of ontology resources.
	// [REQUIRED]
	Group *group.Service
	// Rack is used to manage rack-related operations for tasks.
	// [REQUIRED]
	Rack *rack.Service
	// Status is used to define and process statuses for tasks.
	// [REQUIRED]
	Status *status.Service
	// Signals is used to propagate task changes through the Synnax signals' channel
	// communication mechanism.
	// [OPTIONAL]
	Signals *signals.Provider
	// Channel is used to create channels related to task operations.
	// [OPTIONAL]
	Channel channel.Writeable
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
	c.Status = override.Nil(c.Status, other.Status)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("task")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "rack", c.Rack)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

type Service struct {
	cfg                           Config
	shutdownSignals               io.Closer
	group                         group.Group
	disconnectSuspectRackObserver observe.Disconnect
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
	if cfg.Channel != nil {
		if err = cfg.Channel.Create(ctx, &channel.Channel{
			Name:     "sy_task_cmd",
			DataType: telem.JSONT,
			Virtual:  true,
			Internal: true,
		}); err != nil {
			return nil, err
		}
	}
	s.disconnectSuspectRackObserver = cfg.Rack.OnSuspect(s.onSuspectRack)
	if cfg.Signals == nil {
		return
	}
	cdcS, err := signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signals.GorpPublisherConfigPureNumeric[Key, Task](cfg.DB, telem.Uint64T),
	)
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
	s.disconnectSuspectRackObserver()
	if s.shutdownSignals != nil {
		return s.shutdownSignals.Close()
	}
	return nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:     tx,
		otg:    s.cfg.Ontology.NewWriter(tx),
		rack:   s.cfg.Rack.NewWriter(tx),
		group:  s.group,
		status: status.NewWriter[StatusDetails](s.cfg.Status, tx),
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:    s.cfg.Ontology,
		baseTX: s.cfg.DB,
		gorp:   gorp.NewRetrieve[Key, Task](),
	}
}

func (s *Service) onSuspectRack(ctx context.Context, rackStat rack.Status) {
	var tasks []Task
	if err := s.NewRetrieve().WhereRacks(rackStat.Details.Rack).
		Entries(&tasks).
		Exec(ctx, nil); err != nil {
		s.cfg.L.Error("failed to retrieve tasks on suspect rack", zap.Error(err))
	}
	statuses := make([]Status, len(tasks))
	for i, tsk := range tasks {
		statuses[i] = Status{
			Key:         OntologyID(tsk.Key).String(),
			Time:        telem.Now(),
			Name:        tsk.Name,
			Variant:     rackStat.Variant,
			Message:     rackStat.Message,
			Description: rackStat.Description,
			Details:     StatusDetails{Task: tsk.Key},
		}
	}
	if err := status.NewWriter[StatusDetails](s.cfg.Status, nil).
		SetMany(ctx, &statuses); err != nil {
		s.cfg.L.Error("failed to set statuses on suspect rack", zap.Error(err))
	}
}
