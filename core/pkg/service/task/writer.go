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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
	status    status.Writer[StatusDetails]
	table     *gorp.Table[Key, Task]
}

// resolveStatus returns the status to persist for t. Without a provided status, a
// task created for the first time provably has no live instance, so it gets a quiet
// "has not been deployed" placeholder; an existing task may be running, so its
// placeholder stays a "status unknown" warning.
func resolveStatus(t *Task, provided *Status, existed bool) *Status {
	if provided == nil {
		message := fmt.Sprintf("%s status unknown", t.Name)
		variant := status.VariantWarning
		if !existed {
			message = fmt.Sprintf("%s has not been deployed", t.Name)
			variant = status.VariantDisabled
		}
		return &Status{
			Key:     t.OntologyID().String(),
			Time:    telem.Now(),
			Name:    t.Name,
			Message: message,
			Variant: variant,
			Details: StatusDetails{Task: t.Key},
		}
	}
	provided.Key = t.OntologyID().String()
	provided.Details.Task = t.Key
	provided.Name = t.Name
	return provided
}

// healStatus restores a task's status row if it has gone missing (e.g. deleted
// out-of-band) without clobbering a live one. Tasks are re-created on every scan cycle,
// so on a no-op update the placeholder status must not overwrite a status the
// driver has already reported; it is only written when no row exists.
func (w Writer) healStatus(ctx context.Context, stat *Status) error {
	if exists, err := gorp.NewRetrieve[string, Status]().
		Where(gorp.MatchKeys[string, Status](stat.Key)).
		Exists(ctx, w.tx); err != nil || exists {
		return err
	}
	return w.status.Set(ctx, stat)
}

// Create creates or updates a task. A provided status is persisted as given. Without
// one, a new task gets a "has not been deployed" placeholder, and an existing task
// keeps its reported status (healed to "status unknown" if the row is missing).
func (w Writer) Create(ctx context.Context, t *Task) error {
	if t.Key == uuid.Nil {
		t.Key = uuid.New()
	}
	var err error
	if t.ConfigHash, err = hashConfig(t.Config); err != nil {
		return err
	}
	providedStatus := t.Status // Preserve before clearing for Gorp
	t.Status = nil             // Status stored separately, not in Gorp
	existed := false
	if err := w.table.NewCreate().
		MergeExisting(func(_ gorp.Context, creating, existing Task) (Task, error) {
			existed = true
			if existing.Snapshot {
				creating.Config = existing.Config
				creating.ConfigHash = existing.ConfigHash
			}
			return creating, nil
		}).
		Entry(t).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	stat := resolveStatus(t, providedStatus, existed)
	if providedStatus != nil {
		if err := w.status.Set(ctx, stat); err != nil {
			return err
		}
	} else if err := w.healStatus(ctx, stat); err != nil {
		return err
	}
	// We don't create ontology resources for internal tasks.
	if t.Internal {
		return nil
	}
	otgID := t.OntologyID()
	exists, err := w.otg.NewRetrieve().WhereIDs(otgID).Exists(ctx, w.tx)
	if err != nil || exists {
		return err
	}
	if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
		return err
	}
	return w.otgWriter.DefineRelationships(
		ctx,
		w.group.OntologyID(),
		ontology.RelationshipTypeParentOf,
		otgID,
	)
}

// CreateMany creates the given tasks. If tasks with the same key already exist, they
// will be overwritten.
func (w Writer) CreateMany(ctx context.Context, tasks *[]Task) error {
	for i := range *tasks {
		if err := w.Create(ctx, &(*tasks)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the task with the given key and its associated status.
func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Task](key)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.otgWriter.DeleteResources(ctx, OntologyID(key)); err != nil {
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
	newKey := uuid.New()
	var res Task
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Task](key)).Change(func(_ gorp.Context, t Task) Task {
		t.Key = newKey
		t.Name = name
		t.Snapshot = snapshot
		res = t
		return t
	}).Exec(ctx, w.tx); err != nil {
		return Task{}, err
	}
	if err := w.status.Set(ctx, resolveStatus(&res, nil, false)); err != nil {
		return Task{}, err
	}
	if err := w.otgWriter.DefineResources(ctx, OntologyID(newKey)); err != nil {
		return Task{}, err
	}
	return res, nil
}
