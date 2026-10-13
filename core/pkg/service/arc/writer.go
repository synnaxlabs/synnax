// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"encoding/json/v2"
	"uuid"

	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	taskversions "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/debounce"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// TaskType is the task type for tasks that execute an Arc module.
const TaskType = "arc"

// Writer is used to create, update, and delete arcs within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx         gorp.Tx
	otgWriter  ontology.Writer
	otg        *ontology.Ontology
	tasks      *task.Service
	status     *status.Service
	table      *gorp.Table[Key, Arc]
	dispatcher actions.Dispatcher[Key, Action]
	sweeper    textSweeper
	taskSync   *debounce.Keyed[Key]
}

// Create creates the given Arc. If the Arc does not have a key,
// a new key will be generated.
func (w Writer) Create(ctx context.Context, a *Arc) error {
	var (
		exists bool
		err    error
	)
	if a.Key == uuid.Nil() {
		a.Key = uuid.New()
	} else {
		exists, err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Arc](a.Key)).
			Exists(ctx, w.tx)
		if err != nil {
			return err
		}
	}
	if len(a.Text.Doc.Inserts) == 0 && a.Text.Raw != "" {
		a.Text.Doc = text.Create(a.Text.Raw)
	}
	if err = a.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(a).Exec(ctx, w.tx); err != nil {
		return err
	}
	if !exists {
		if err = w.otgWriter.DefineResources(ctx, a.OntologyID()); err != nil {
			return err
		}
	} else if err = w.syncTask(ctx, *a); err != nil {
		return err
	}
	// Notify last: a create rejected by ontology validation must not be broadcast.
	w.dispatcher.Notify(
		ctx, a.Key, "", []Action{NewCreateAction(CreatePayload{Arc: *a})},
	)
	return nil
}

// CreateMany creates the given Arcs. If Arcs with the same key already exist, they will
// be overwritten.
func (w Writer) CreateMany(ctx context.Context, arcs *[]Arc) error {
	for i := range *arcs {
		if err := w.Create(ctx, &(*arcs)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the Arcs with the given keys. If the Arc has child tasks, those tasks
// will also be deleted.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	for _, key := range keys {
		if err := w.deleteChildTasks(ctx, key); err != nil {
			return err
		}
	}
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Arc](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otgWriter.DeleteResources(ctx, OntologyID(key)); err != nil {
			return err
		}
		w.sweeper.forget(key)
		w.taskSync.Forget(key)
	}
	return nil
}

func (w Writer) deleteChildTasks(ctx context.Context, key Key) error {
	taskKeys, err := w.childTaskKeys(ctx, key)
	if err != nil {
		return err
	}
	tw := w.tasks.NewWriter(w.tx)
	for _, taskKey := range taskKeys {
		if err = tw.Delete(ctx, taskKey, false); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) childTaskKeys(ctx context.Context, key Key) ([]task.Key, error) {
	var children []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(ontology.ResourceTypeTask).
		Entries(&children).
		ExcludeFieldData(true).
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	if len(children) == 0 {
		return nil, nil
	}
	return task.KeysFromOntologyIDs(ontology.ResourceIDs(children))
}

// TaskUpdate holds the fields of an arc's task to change. A nil field stays as it is.
type TaskUpdate struct {
	// Rack binds the arc to the rack, creating its task or moving the existing one. A
	// zero rack unbinds the arc, deleting its task.
	Rack *rack.Key
	// Performance sets how closely the task's runtime holds timer deadlines.
	Performance *taskversions.Performance
}

// UpdateTask changes the task of the arc with the given key. It returns the task, or
// nil after an unbind. It returns an error wrapping validate.ErrValidation when
// unbinding a running task, when unbinding and setting the performance in one update,
// or when setting the performance of an arc with no task.
func (w Writer) UpdateTask(
	ctx context.Context,
	key Key,
	update TaskUpdate,
) (*task.Task, error) {
	existing, err := w.childTaskKeys(ctx, key)
	if err != nil {
		return nil, err
	}
	if update.Rack != nil && *update.Rack == 0 {
		if update.Performance != nil {
			return nil, errors.Wrap(
				validate.ErrValidation,
				"cannot set the performance of an arc while unbinding it",
			)
		}
		return nil, w.clearRack(ctx, existing)
	}
	var (
		a   Arc
		cfg taskversions.Config
		tsk task.Task
	)
	if err = w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Arc](key)).
		Entry(&a).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		if tsk, cfg, err = w.retrieveTask(ctx, existing[0]); err != nil {
			return nil, err
		}
	} else if update.Rack == nil {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"cannot set the performance of an arc with no rack",
		)
	}
	if update.Rack != nil {
		tsk.Rack = *update.Rack
	}
	if update.Performance != nil {
		cfg.Performance = *update.Performance
	}
	if cfg.Hash, err = Hash(a); err != nil {
		return nil, err
	}
	return w.writeTask(ctx, a, tsk.Rack, existing, cfg)
}

// writeTask creates or overwrites the arc's task on rackKey with cfg, stamping the
// arc's key into it. cfg.Hash is the arc's semantic hash, so the task's config hash
// tracks the arc's content and the task drift mechanism reports arc content drift with
// no extra machinery.
func (w Writer) writeTask(
	ctx context.Context,
	a Arc,
	rackKey rack.Key,
	existing []task.Key,
	cfg taskversions.Config,
) (*task.Task, error) {
	cfg.ArcKey = a.Key
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var encoded msgpack.EncodedJSON
	if err = json.Unmarshal(b, &encoded); err != nil {
		return nil, err
	}
	tsk := task.Task{Rack: rackKey, Name: a.Name, Type: TaskType, Config: encoded}
	if len(existing) > 0 {
		tsk.Key = existing[0]
	}
	if err := w.tasks.NewWriter(w.tx).Create(ctx, &tsk); err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		if err := w.otgWriter.DefineRelationships(
			ctx,
			OntologyID(a.Key),
			ontology.RelationshipTypeParentOf,
			tsk.OntologyID(),
		); err != nil {
			return nil, err
		}
	}
	return &tsk, nil
}

// retrieveTask returns the task with the given key and its decoded config.
func (w Writer) retrieveTask(
	ctx context.Context,
	key task.Key,
) (task.Task, taskversions.Config, error) {
	var (
		tsk task.Task
		cfg taskversions.Config
	)
	if err := w.tasks.NewRetrieve().
		Where(task.MatchKeys(key)).
		Entry(&tsk).
		Exec(ctx, w.tx); err != nil {
		return tsk, cfg, err
	}
	return tsk, cfg, tsk.Config.Unmarshal(&cfg)
}

// syncTask rewrites the arc's task in place when an edit changed the arc's semantic
// hash, keeping the task config in step with the arc's content. A no-op for arcs with
// no rack bound and for edits, like graph layout moves, that hash equally.
func (w Writer) syncTask(ctx context.Context, a Arc) error {
	existing, err := w.childTaskKeys(ctx, a.Key)
	if err != nil || len(existing) == 0 {
		return err
	}
	tsk, cfg, err := w.retrieveTask(ctx, existing[0])
	if err != nil {
		return err
	}
	hash, err := Hash(a)
	if err != nil || cfg.Hash == hash {
		return err
	}
	cfg.Hash = hash
	_, err = w.writeTask(ctx, a, tsk.Rack, existing, cfg)
	return err
}

func (w Writer) clearRack(ctx context.Context, taskKeys []task.Key) error {
	for _, taskKey := range taskKeys {
		var stat task.Status
		err := w.status.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](task.OntologyID(taskKey).String())).
			Entry(&stat).
			Exec(ctx, w.tx)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
		if err == nil && stat.Details.Running {
			return errors.Wrap(
				validate.ErrValidation,
				"cannot clear the rack of a running arc; stop it first",
			)
		}
		if err = w.tasks.NewWriter(w.tx).Delete(ctx, taskKey, false); err != nil {
			return err
		}
	}
	return nil
}
