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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

// Writer is used to create, update, and delete arcs within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx         gorp.Tx
	otg        ontology.Writer
	task       task.Writer
	table      *gorp.Table[Key, Arc]
	dispatcher actions.Dispatcher[Key, Action]
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
		exists, err = w.table.NewRetrieve().Where(gorp.MatchKeys[Key, Arc](a.Key)).Exists(ctx, w.tx)
		if err != nil {
			return err
		}
	}
	if err = a.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(a).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := OntologyID(a.Key)
	if !exists {
		if err = w.otg.DefineResource(ctx, otgID); err != nil {
			return err
		}
	}
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
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	actions []Action,
) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Arc](key)).
		ChangeErr(func(_ gorp.Context, a Arc) (Arc, error) {
			return Reduce(a, actions...)
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Notify(ctx, key, dispatchKey, actions)
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

func (w Writer) deleteChildTasks(ctx context.Context, key Key) error {
	var children []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(ontology.ResourceTypeTask).
		Entries(&children).
		ExcludeFieldData(true).
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	if len(children) == 0 {
		return nil
	}
	taskKeys, err := task.KeysFromOntologyIDs(ontology.ResourceIDs(children))
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
