// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/migrations/v0"
	v54 "github.com/synnaxlabs/synnax/pkg/service/task/migrations/v54"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
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
	Channel *channel.Service
	// Search is the search index for fuzzy searching tasks.
	// [REQUIRED]
	Search *search.Index
	alamos.Instrumentation
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Status = override.Nil(c.Status, other.Status)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Search = override.Nil(c.Search, other.Search)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("task")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "rack", c.Rack)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

type Service struct {
	cfg               ServiceConfig
	closer            xio.MultiCloser
	group             group.Group
	table             *gorp.Table[Key, Task]
	commandChannelKey channel.Key
}

// Observe returns an observable that notifies callers of changes to task entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Task]] {
	return s.table.Observe()
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	v0Mig := v0.Migration(v0.MigrationConfig{Status: cfg.Status})
	if s.table, err = gorp.OpenTable[Key, Task](ctx, gorp.TableConfig[Task]{
		DB: cfg.DB,
		Migrations: []migrate.Migration{
			v0Mig,
			gorp.CodecMigration[v54.Key, v54.Task]("msgpack_to_orc", v0Mig.Key()),
			migrate.WithAddedDeps(
				gorp.NewEntryMigration[v54.Key, Key, v54.Task, Task](
					"v54_drop_status",
					MigrateTask,
				),
				"msgpack_to_orc",
			),
		},
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(ctx, "Tasks", ontology.RootID); !ok(err, nil) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	s.cleanupInternalOntologyResources(ctx)
	if cfg.Channel != nil {
		cmdCh := channel.Channel{
			Name:     "sy_task_cmd",
			DataType: telem.JSONT,
			Virtual:  true,
			Internal: true,
		}
		if err = cfg.Channel.Create(
			ctx,
			&cmdCh,
			channel.RetrieveIfNameExists(),
		); !ok(err, nil) {
			return nil, err
		}
		s.commandChannelKey = cmdCh.Key()
	}
	disconnect := cfg.Rack.OnSuspect(s.onSuspectRack)
	ok(nil, xio.NoFailCloserFunc(disconnect))
	if cfg.Signals == nil {
		return s, nil
	}
	var sig io.Closer
	if sig, err = signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signals.GorpPublisherConfigPureNumeric[Key, Task](s.table.Observe(), telem.Uint64T),
	); !ok(err, sig) {
		return nil, err
	}
	return s, nil
}

func (s *Service) CommandChannelKey() channel.Key {
	return s.commandChannelKey
}

// cleanupInternalOntologyResources purges existing internal task resources from the ontology.
// we want to hide internal tasks from the user.
func (s *Service) cleanupInternalOntologyResources(ctx context.Context) {
	var tasks []Task
	if err := s.NewRetrieve().Where(MatchInternal(true)).Entries(&tasks).Exec(ctx, nil); err != nil {
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

func (s *Service) Close() error { return s.closer.Close() }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:     tx,
		otg:    s.cfg.Ontology.NewWriter(tx),
		rack:   s.cfg.Rack.NewWriter(tx),
		group:  s.group,
		status: status.NewWriter[StatusDetails](s.cfg.Status, tx),
		table:  s.table,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		search: s.cfg.Search,
		baseTX: s.cfg.DB,
		gorp:   s.table.NewRetrieve(),
	}
}

func (s *Service) onSuspectRack(ctx context.Context, rackStat rack.Status) {
	var tasks []Task
	if err := s.NewRetrieve().Where(MatchRacks(rackStat.Details.Rack)).
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
			Details:     StatusDetails{Task: tsk.Key, Running: false},
		}
	}
	if err := status.NewWriter[StatusDetails](s.cfg.Status, nil).
		SetMany(ctx, &statuses); err != nil {
		s.cfg.L.Error("failed to set statuses on suspect rack", zap.Error(err))
	}
}
