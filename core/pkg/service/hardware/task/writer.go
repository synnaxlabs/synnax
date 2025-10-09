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

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	rack  rack.Writer
	group group.Group
}

func (w Writer) Create(ctx context.Context, t *Task) (err error) {
	if !t.Key.IsValid() {
		localKey, err := w.rack.NewTaskKey(ctx, t.Rack())
		if err != nil {
			return err
		}
		t.Key = NewKey(t.Rack(), localKey)
	}
	t.Status = nil
	// We don't create ontology resources for internal tasks.
	if err = gorp.NewCreate[Key, Task]().
		MergeExisting(func(creating Task, existing Task) (Task, error) {
			if !existing.Snapshot {
				return creating, nil
			}
			creating.Config = existing.Config
			return creating, nil
		}).
		Entry(t).
		Exec(ctx, w.tx); err != nil || t.Internal {
		return
	}
	otgID := OntologyID(t.Key)
	exists, err := w.otg.HasResource(ctx, otgID)
	if err != nil || exists {
		return err
	}
	if err = w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	return w.otg.DefineRelationship(ctx, w.group.OntologyID(), ontology.ParentOf, otgID)
}

func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	q := gorp.NewDelete[Key, Task]().WhereKeys(key)
	if err := q.Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResource(ctx, OntologyID(key))
}

func (w Writer) Copy(
	ctx context.Context,
	key Key,
	name string,
	snapshot bool,
) (Task, error) {
	localKey, err := w.rack.NewTaskKey(ctx, key.Rack())
	if err != nil {
		return Task{}, err
	}
	newKey := NewKey(key.Rack(), localKey)
	var res Task
	if err = gorp.NewUpdate[Key, Task]().WhereKeys(key).Change(func(t Task) Task {
		t.Key = newKey
		t.Name = name
		t.Snapshot = snapshot
		res = t
		return t
	}).Exec(ctx, w.tx); err != nil {
		return res, err
	}
	if err := w.otg.DefineResource(ctx, OntologyID(newKey)); err != nil {
		return Task{}, err
	}
	return res, err
}
