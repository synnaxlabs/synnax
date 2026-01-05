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

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

type Writer struct {
	tx     gorp.Tx
	otg    ontology.Writer
	rack   rack.Writer
	group  group.Group
	status status.Writer[StatusDetails]
}

func resolveStatus(t *Task, provided *Status) *Status {
	if provided == nil {
		return &Status{
			Key:     OntologyID(t.Key).String(),
			Time:    telem.Now(),
			Name:    t.Name,
			Message: fmt.Sprintf("%s status unknown", t.Name),
			Variant: xstatus.VariantWarning,
			Details: StatusDetails{Task: t.Key},
		}
	}
	provided.Key = OntologyID(t.Key).String()
	provided.Details.Task = t.Key
	provided.Name = t.Name
	return provided
}

// Create creates or updates a task. If a status is provided on the task,
// it will be used instead of the default "unknown" status.
func (w Writer) Create(ctx context.Context, t *Task) error {
	if !t.Key.IsValid() {
		localKey, err := w.rack.NewTaskKey(ctx, t.Rack())
		if err != nil {
			return err
		}
		t.Key = NewKey(t.Rack(), localKey)
	}
	providedStatus := t.Status // Preserve before clearing for gorp
	t.Status = nil             // Status stored separately, not in gorp
	if err := gorp.NewCreate[Key, Task]().
		MergeExisting(func(_ gorp.Context, creating, existing Task) (Task, error) {
			if existing.Snapshot {
				creating.Config = existing.Config
			}
			return creating, nil
		}).
		Entry(t).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	stat := resolveStatus(t, providedStatus)
	if err := w.status.Set(ctx, stat); err != nil {
		return err
	}
	// We don't create ontology resources for internal tasks.
	if t.Internal {
		return nil
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

// Delete deletes the task with the given key and its associated status.
func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	q := gorp.NewDelete[Key, Task]().WhereKeys(key)
	if err := q.Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
		return err
	}
	return w.status.Delete(ctx, OntologyID(key).String())
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
	if err = gorp.NewUpdate[Key, Task]().WhereKeys(key).Change(func(_ gorp.Context, t Task) Task {
		t.Key = newKey
		t.Name = name
		t.Snapshot = snapshot
		res = t
		return t
	}).Exec(ctx, w.tx); err != nil {
		return res, err
	}
	if err = w.status.Set(ctx, resolveStatus(&res, nil)); err != nil {
		return Task{}, err
	}
	if err = w.otg.DefineResource(ctx, OntologyID(newKey)); err != nil {
		return Task{}, err
	}
	return res, err
}
