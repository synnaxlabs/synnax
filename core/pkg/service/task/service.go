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
	"fmt"
	"io"
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/task/versions"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
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
	// ImExExcluded lists the task types with no file form. Export refuses them and no
	// importer is registered for them.
	//
	// [OPTIONAL]
	ImExExcluded []string
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
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	c.Configs = override.Zero(c.Configs, other.Configs)
	c.ImExExcluded = override.Slice(c.ImExExcluded, other.ImExExcluded)
	return c
}

// Validate implements xconfig.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("task")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("group", c.Group)
	v.NotNil("rack", c.Rack)
	v.NotNil("status", c.Status)
	v.NotNil("imex", c.ImEx)
	v.Ternary("configs", c.Configs.IsZero(), "must be non-zero")
	return v.Error()
}

type Service struct {
	cfg               ServiceConfig
	closer            xio.MultiCloser
	group             group.Group
	table             *gorp.Table[Key, Task]
	indexes           indexes
	state             *actions.State[Key, Action]
	commandChannelKey channel.Key
	actionChannelKey  channel.Key
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
	s = &Service{
		cfg:     cfg,
		indexes: newIndexes(),
		state:   actions.NewState[Key, Action](cfg.DB),
	}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Task]{
		DB:      cfg.DB,
		Name:    tableName,
		Indexes: s.indexes.all(),
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
	if err = cfg.ImEx.RegisterExporter(s); !ok(err, nil) {
		return nil, err
	}
	// Task files carry the fine-grained type ("opc_read") while export routes under
	// the coarse "task" resource type, so the importer registers per config type.
	for _, t := range cfg.Configs.Types() {
		if slices.Contains(cfg.ImExExcluded, t) {
			continue
		}
		if err = cfg.ImEx.RegisterImporter(t, s); !ok(err, nil) {
			return nil, err
		}
	}
	// Retired types register too so an old export fails with a retirement message
	// instead of "no importer registered".
	for t := range retiredTypes {
		if err = cfg.ImEx.RegisterImporter(t, s); !ok(err, nil) {
			return nil, err
		}
	}
	if cfg.Channel != nil {
		cmdCh := channel.Channel{
			Name:     "sy_task_cmd",
			DataType: telem.JSONT,
			Virtual:  true,
			Internal: true,
		}
		if err = cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return cfg.Channel.NewWriter(tx).Create(
				ctx,
				&cmdCh,
				channel.RetrieveIfNameExists(),
			)
		}); !ok(err, nil) {
			return nil, err
		}
		s.commandChannelKey = cmdCh.Key()
	}
	disconnect := cfg.Rack.OnSuspect(s.onSuspectRack)
	ok(nil, xio.NoFailCloserFunc(disconnect))
	if cfg.Signals != nil {
		pubCfg := signals.GorpPublisherConfigUUID(s.table.Observe())
		pubCfg.MarshalSet = func(t Task) ([]byte, error) {
			t.Config, t.Status = nil, nil
			return signals.MarshalJSON(t)
		}
		var sig io.Closer
		if sig, err = cfg.Signals.PublishFromGorp(ctx, pubCfg); !ok(err, sig) {
			return nil, err
		}
		// The Driver reads sy_task_set as task metadata, so actions take their own
		// channel.
		if sig, err = cfg.Signals.PublishActions(
			ctx,
			signals.ActionPublisherConfig[Key, Action]{
				Observable: s.state,
				Name:       "task_action",
			},
		); !ok(err, sig) {
			return nil, err
		}
		var actionCh channel.Channel
		if err = cfg.Channel.NewRetrieve().
			Where(channel.MatchNames("sy_task_action_set")).
			Entry(&actionCh).
			Exec(ctx, nil); !ok(err, nil) {
			return nil, err
		}
		s.actionChannelKey = actionCh.Key()
	}
	return s, nil
}

// OnAction subscribes the given handler to the actions that Dispatch and Writer emit.
// The handler runs synchronously after the transaction that applied the actions
// commits. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnChange(handler)
}

// Dispatch applies the actions atomically to the task with the given key. Dispatches
// for the same task run one at a time, each in its own transaction. After the commit,
// Dispatch broadcasts the actions, then a set of the stored config when storing changed
// it, such as by filling a default, then a set of the new config hash. Dispatch returns
// the sets it added. dispatchKey is carried onto the broadcast so the sender recognizes
// its echo. Dispatch returns an error wrapping validate.ErrValidation when an action is
// a create, an action other than a set of the name targets a snapshot, or the result
// fails to decode or validate, and an error wrapping query.ErrNotFound when no task has
// the key.
func (s *Service) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	acts []Action,
) ([]Action, error) {
	var added []Action
	if err := s.state.Dispatch(
		ctx,
		key,
		dispatchKey,
		acts,
		func(tx gorp.Tx) (_ []Action, err error) {
			added, err = s.NewWriter(tx).apply(ctx, key, acts)
			return added, err
		},
	); err != nil {
		return nil, err
	}
	return added, nil
}

func (s *Service) CommandChannelKey() channel.Key {
	return s.commandChannelKey
}

// ActionChannelKey returns the key of the channel that broadcasts task actions. Its
// samples carry task configs, so only subjects that can read a task may receive them.
func (s *Service) ActionChannelKey() channel.Key {
	return s.actionChannelKey
}

func (s *Service) Close() error { return s.closer.Close() }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:         tx,
		otgWriter:  s.cfg.Ontology.NewWriter(tx),
		otg:        s.cfg.Ontology,
		group:      s.group,
		status:     s.cfg.Status.NewWriter(tx),
		statuses:   s.cfg.Status,
		table:      s.table,
		configs:    s.cfg.Configs,
		dispatcher: s.state.Dispatcher(),
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		indexes: s.indexes,
		baseTX:  s.cfg.DB,
		gorp:    s.table.NewRetrieve(),
		otg:     s.cfg.Ontology,
		configs: s.cfg.Configs,
	}
}

func (s *Service) onSuspectRack(ctx context.Context, rackStat rack.Status) error {
	rackKey, err := rack.KeyFromResourceID(*rackStat.Owner)
	if err != nil {
		return err
	}
	var tasks []Task
	if err := s.NewRetrieve().Where(MatchRacks(rackKey)).
		Entries(&tasks).
		Exec(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to retrieve tasks on suspect rack")
	}
	reported, err := s.cfg.Status.Join[StatusDetails](
		ctx,
		nil,
		ResourceIDsFromTasks(tasks),
	)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve statuses on suspect rack")
	}
	statuses := make([]Status, 0, len(tasks))
	for i, tsk := range tasks {
		// A task without a status has never reported running, so it gets no copy.
		prev := reported[i]
		if prev == nil || !prev.Details.Running {
			continue
		}
		// A silent rack does not undo the deploy the Driver reported, so the config
		// hash is carried across.
		statuses = append(statuses, Status{
			Key:     prev.Key,
			Owner:   prev.Owner,
			Time:    telem.Now(),
			Name:    tsk.Name,
			Variant: rackStat.Variant,
			Message: "Task stopped responding",
			Description: fmt.Sprintf(
				"The Core lost contact with the Driver on %s while the task was "+
					"running.",
				rackStat.Name,
			),
			Details: StatusDetails{
				ConfigHash: prev.Details.ConfigHash,
				Rack:       rackKey,
			},
		})
	}
	if len(statuses) == 0 {
		return nil
	}
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return s.cfg.Status.NewWriter(tx).SetMany(ctx, &statuses)
	})
}
