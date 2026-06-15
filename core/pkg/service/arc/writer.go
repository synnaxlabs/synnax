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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// TaskType is the task type used for tasks that run an Arc module. The Arc runtime
// factory configures tasks of this type.
const TaskType = "arc"

// Writer is used to create, update, and delete arcs within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	task  task.Writer
	table *gorp.Table[Key, Arc]
}

// Create persists the Arc described by n and returns it. If n.Key is unset, a new key
// is generated. If n.Rack is set, the Arc is deployed by creating a task on that rack
// and associating it with the Arc; the returned Arc's Task is populated with the
// resulting task key.
func (w Writer) Create(ctx context.Context, n New) (Arc, error) {
	a := Arc{Key: n.Key, Name: n.Name, Mode: n.Mode, Graph: n.Graph, Text: n.Text}
	exists := false
	if a.Key == uuid.Nil {
		a.Key = uuid.New()
	} else {
		var err error
		if exists, err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Arc](a.Key)).
			Exists(ctx, w.tx); err != nil {
			return Arc{}, err
		}
	}
	if err := w.table.NewCreate().Entry(&a).Exec(ctx, w.tx); err != nil {
		return Arc{}, err
	}
	if !exists {
		if err := w.otg.DefineResource(ctx, OntologyID(a.Key)); err != nil {
			return Arc{}, err
		}
	}
	if n.Rack != nil {
		if err := w.associateTask(ctx, &a, *n.Rack); err != nil {
			return Arc{}, err
		}
	}
	return a, nil
}

// associateTask ensures the Arc has exactly one task deployed on rackKey. A task already
// deployed on a different rack is torn down first. a.Task is set to the key of the
// associated task.
func (w Writer) associateTask(ctx context.Context, a *Arc, rackKey rack.Key) error {
	existing, err := w.childTaskKeys(ctx, a.Key)
	if err != nil {
		return err
	}
	taskKey := task.NewKey(rackKey, 0)
	for _, k := range existing {
		if k.Rack() == rackKey {
			taskKey = k
			continue
		}
		if err = w.otg.DeleteRelationship(
			ctx,
			OntologyID(a.Key),
			ontology.RelationshipTypeParentOf,
			task.OntologyID(k),
		); err != nil {
			return err
		}
		if err = w.task.Delete(ctx, k, false); err != nil {
			return err
		}
	}
	tsk := task.Task{
		Key:    taskKey,
		Name:   a.Name,
		Type:   TaskType,
		Config: msgpack.EncodedJSON{"arc_key": a.Key.String()},
	}
	if err = w.task.Create(ctx, &tsk); err != nil {
		return err
	}
	if err = w.otg.DefineRelationship(
		ctx,
		OntologyID(a.Key),
		ontology.RelationshipTypeParentOf,
		task.OntologyID(tsk.Key),
	); err != nil {
		return err
	}
	a.Task = &tsk.Key
	return nil
}

// CreateMany persists each Arc described by news and returns the created Arcs.
func (w Writer) CreateMany(ctx context.Context, news []New) ([]Arc, error) {
	arcs := make([]Arc, len(news))
	for i, n := range news {
		a, err := w.Create(ctx, n)
		if err != nil {
			return nil, err
		}
		arcs[i] = a
	}
	return arcs, nil
}

// Delete deletes the Arcs with the given keys. If the Arc has child tasks, those tasks
// will also be deleted.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	for _, key := range keys {
		if err := w.deleteChildTasks(ctx, key); err != nil {
			return err
		}
	}
	if err := w.table.NewDelete().Where(gorp.MatchKeys[Key, Arc](keys...)).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}

// childTaskKeys returns the keys of the tasks parented under the Arc with the given key.
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
	return task.KeysFromOntologyIDs(ontology.ResourceIDs(children))
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
