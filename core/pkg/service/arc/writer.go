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

	"github.com/google/uuid"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/crdt"
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
	task       task.Writer
	status     *status.Service
	table      *gorp.Table[Key, Arc]
	dispatcher actions.Dispatcher[Key, Action]
	sweeper    textSweeper
}

// Create creates the given Arc. If the Arc does not have a key,
// a new key will be generated.
func (w Writer) Create(ctx context.Context, a *Arc) error {
	var (
		exists bool
		err    error
	)
	if a.Key == uuid.Nil {
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

// Dispatch applies a sequence of actions atomically to the Arc with the given
// key. After a successful update the actions are notified to the service-level
// observer so subscribers (cluster signals) can broadcast them. dispatchKey is
// a client-generated identifier carried verbatim onto the broadcast so the
// originating client can match its own echo against the set of outstanding
// local replays and skip a redundant reduce when no foreign action interleaved.
//
// When the arc's text has gone quiet, Dispatch also reclaims the space held by
// tombstoned characters: it forgets the characters that were already dead before this
// dispatch and broadcasts that sweep as a separate frame with an empty dispatchKey, so
// every editor (including the originator, which skips its own echo) applies it.
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	actions []Action,
) error {
	var sweep []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Arc](key)).
		ChangeErr(func(_ gorp.Context, a Arc) (Arc, error) {
			sweep = nil
			var dead []crdt.ID
			if w.sweeper.quiet(key) {
				dead = w.sweeper.forgettable(a.Text.Doc)
			}
			a, err := Reduce(a, actions...)
			if err != nil {
				return a, err
			}
			if len(dead) > 0 {
				sweep = []Action{NewForgetCharsAction(ForgetCharsPayload{IDs: dead})}
				if a, err = Reduce(a, sweep...); err != nil {
					return a, err
				}
			}
			if containsTextEdit(actions) {
				w.sweeper.recordEdit(key)
			}
			return a, nil
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Notify(ctx, key, dispatchKey, actions)
	if len(sweep) > 0 {
		w.dispatcher.Notify(ctx, key, "", sweep)
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
	}
	return nil
}

func (w Writer) deleteChildTasks(ctx context.Context, key Key) error {
	taskKeys, err := w.childTaskKeys(ctx, key)
	if err != nil {
		return err
	}
	for _, taskKey := range taskKeys {
		if err = w.task.Delete(ctx, taskKey, false); err != nil {
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

// Deploy binds the arc with the given key to rackKey by creating its task on that
// rack, or moving the existing one, stamping the arc's current semantic hash into the
// task config. A zero rackKey undeploys: the task is deleted, stopping it on its rack.
// It returns the deployed task, or nil after an undeploy. It returns an error wrapping
// validate.ErrValidation when undeploying a running task.
func (w Writer) Deploy(
	ctx context.Context,
	key Key,
	rackKey rack.Key,
) (*task.Task, error) {
	existing, err := w.childTaskKeys(ctx, key)
	if err != nil {
		return nil, err
	}
	if rackKey == 0 {
		return nil, w.undeploy(ctx, existing)
	}
	var a Arc
	if err = w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Arc](key)).
		Entry(&a).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	hash, err := Hash(a)
	if err != nil {
		return nil, err
	}
	tsk := task.Task{
		Rack:   rackKey,
		Name:   a.Name,
		Type:   TaskType,
		Config: msgpack.EncodedJSON{"arc_key": key.String(), "hash": hash},
	}
	if len(existing) > 0 {
		tsk.Key = existing[0]
	}
	if err = w.task.Create(ctx, &tsk); err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		if err = w.otgWriter.DefineRelationships(
			ctx,
			OntologyID(key),
			ontology.RelationshipTypeParentOf,
			tsk.OntologyID(),
		); err != nil {
			return nil, err
		}
	}
	return &tsk, nil
}

func (w Writer) undeploy(ctx context.Context, taskKeys []task.Key) error {
	for _, taskKey := range taskKeys {
		var stat task.Status
		err := status.NewRetrieve[task.StatusDetails](w.status).
			Where(status.MatchKeys[task.StatusDetails](task.OntologyID(taskKey).String())).
			Entry(&stat).
			Exec(ctx, w.tx)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
		if err == nil && stat.Details.Running {
			return errors.Wrap(
				validate.ErrValidation,
				"cannot undeploy a running arc; stop it first",
			)
		}
		if err = w.task.Delete(ctx, taskKey, false); err != nil {
			return err
		}
	}
	return nil
}
