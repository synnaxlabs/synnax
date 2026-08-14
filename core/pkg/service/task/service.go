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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/task/versions"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
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
	//
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between tasks and other resources in the
	// Synnax cluster.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create task related groups of ontology resources.
	//
	// [REQUIRED]
	Group *group.Service
	// Rack is used to manage rack-related operations for tasks.
	//
	// [REQUIRED]
	Rack *rack.Service
	// Status is used to define and process statuses for tasks.
	//
	// [REQUIRED]
	Status *status.Service
	// Channel is used to create channels related to task operations.
	//
	// [OPTIONAL]
	Channel *channel.Service
	// Search is the search index for fuzzy searching tasks.
	//
	// [REQUIRED]
	Search *search.Index
	// Signals is used to propagate task changes through the Synnax signals' channel
	// communication mechanism.
	//
	// [OPTIONAL]
	Signals *signals.Provider
	// ImEx is the import/export registry the task service registers itself with as the
	// exporter for task resources during OpenService.
	//
	// [REQUIRED]
	ImEx *imex.Service
	// Configs routes task types to the store that owns their configuration records.
	//
	// [REQUIRED]
	Configs config.Registry
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ xconfig.Config[ServiceConfig] = ServiceConfig{}

// Override implements xconfig.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Status = override.Nil(c.Status, other.Status)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	c.Configs = override.Zero(c.Configs, other.Configs)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("task")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "rack", c.Rack)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "search", c.Search)
	validate.NotNil(v, "imex", c.ImEx)
	v.Ternary("configs", c.Configs.IsZero(), "must be non-zero")
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

func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (s *Service, err error) {
	cfg, err := xconfig.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Task]{
		DB: cfg.DB,
		Migrations: versions.NewMigrations(versions.MigrationsConfig{
			Status:   cfg.Status,
			Ontology: cfg.Ontology,
			Configs:  cfg.Configs,
		}),
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	if s.group, err = cfg.Group.CreateOrRetrieve(
		ctx,
		"Tasks",
		ontology.RootID,
	); !ok(
		err,
		nil,
	) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	cfg.ImEx.RegisterExporter(s)
	// Task files carry the fine-grained type ("opc_read") while export routes under
	// the coarse "task" ontology type, so the importer registers per config type.
	for _, t := range cfg.Configs.Types() {
		cfg.ImEx.RegisterImporter(string(t), s)
	}
	// Retired types register too so an old export fails with a retirement message
	// instead of "no importer registered".
	for t := range retiredTypes {
		cfg.ImEx.RegisterImporter(t, s)
	}
	if cfg.Channel != nil {
		cmdCh := channel.Channel{
			Name:     "sy_task_cmd",
			DataType: telem.JSONT,
			Virtual:  true,
			Internal: true,
		}
		if err = cfg.Channel.NewWriter(nil).Create(
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
	if cfg.Signals != nil {
		pubCfg := signals.GorpPublisherConfigUUID[Task](s.table.Observe())
		pubCfg.MarshalSet = func(t Task) ([]byte, error) {
			t.Config, t.Status = nil, nil
			return signals.MarshalJSON[Key, Task](t)
		}
		var sig io.Closer
		if sig, err = signals.PublishFromGorp(ctx, cfg.Signals, pubCfg); !ok(err, sig) {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) CommandChannelKey() channel.Key {
	return s.commandChannelKey
}

func (s *Service) Close() error { return s.closer.Close() }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:        tx,
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		otg:       s.cfg.Ontology,
		group:     s.group,
		status:    status.NewWriter[StatusDetails](s.cfg.Status, tx),
		table:     s.table,
		configs:   s.cfg.Configs,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		search:  s.cfg.Search,
		baseTX:  s.cfg.DB,
		gorp:    s.table.NewRetrieve(),
		otg:     s.cfg.Ontology,
		configs: s.cfg.Configs,
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
	keys := make([]string, len(tasks))
	for i, tsk := range tasks {
		keys[i] = tsk.OntologyID().String()
	}
	// A silent rack does not undo the deploy the Driver reported, so its config hash
	// and rack are carried across rather than rebuilt.
	var reported []Status
	if err := status.NewRetrieve[StatusDetails](s.cfg.Status).
		Where(status.MatchKeys[StatusDetails](keys...)).
		Entries(&reported).
		Exec(ctx, nil); err != nil {
		s.cfg.L.Error("failed to retrieve statuses on suspect rack", zap.Error(err))
	}
	deployed := make(map[string]StatusDetails, len(reported))
	for _, stat := range reported {
		deployed[stat.Key] = stat.Details
	}
	for i, tsk := range tasks {
		details := StatusDetails{Task: tsk.Key, Running: false}
		if prev, ok := deployed[keys[i]]; ok {
			details.ConfigHash = prev.ConfigHash
			details.Rack = prev.Rack
		}
		statuses[i] = Status{
			Key:         keys[i],
			Time:        telem.Now(),
			Name:        tsk.Name,
			Variant:     rackStat.Variant,
			Message:     rackStat.Message,
			Description: rackStat.Description,
			Details:     details,
		}
	}
	if err := status.NewWriter[StatusDetails](s.cfg.Status, nil).
		SetMany(ctx, &statuses); err != nil {
		s.cfg.L.Error("failed to set statuses on suspect rack", zap.Error(err))
	}
}
