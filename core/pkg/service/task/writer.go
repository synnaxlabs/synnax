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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
	status    status.Writer[StatusDetails]
	table     *gorp.Table[Key, Task]
	configs   config.Registry
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

// healStatus returns the task's effective status, writing stat first if no row exists.
// Tasks are re-created on every scan cycle, so on a no-op update the placeholder must
// not overwrite a status the driver has already reported.
func (w Writer) healStatus(ctx context.Context, stat *Status) (*Status, error) {
	var existing Status
	err := gorp.NewRetrieve[string, Status]().
		Where(gorp.MatchKeys[string, Status](stat.Key)).
		Entry(&existing).
		Exec(ctx, w.tx)
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	if err := w.status.Set(ctx, stat); err != nil {
		return nil, err
	}
	return stat, nil
}

// Create creates or updates a task. The task's config is stored as a record keyed by
// the task's key in its type's config store; the task row itself carries only the
// config hash. A provided status is persisted as given. Without one, a new task gets
// a "has not been deployed" placeholder, and an existing task keeps its reported
// status (healed to "status unknown" if the row is missing).
func (w Writer) Create(ctx context.Context, t *Task) error {
	if t.Key == uuid.Nil {
		t.Key = uuid.New()
	}
	store, ok := w.configs.Store(t.Type)
	if !ok {
		return errors.Wrapf(validate.ErrValidation, "unknown task type %q", t.Type)
	}
	var existing Task
	existed := true
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Task](t.Key)).
		Entry(&existing).
		Exec(ctx, w.tx); err != nil {
		if !errors.Is(err, query.ErrNotFound) {
			return err
		}
		existed = false
	}
	otgID := t.OntologyID()
	if existed && existing.Snapshot {
		t.ConfigHash = existing.ConfigHash
		existingStore, ok := w.configs.Store(existing.Type)
		if !ok {
			return errors.Wrapf(
				validate.ErrValidation, "unknown task type %q", existing.Type,
			)
		}
		data, err := existingStore.Read(ctx, w.tx, t.Key)
		if err != nil {
			if !errors.Is(err, query.ErrNotFound) {
				return err
			}
		} else {
			t.Config = data
		}
	} else {
		if err := store.Write(ctx, w.tx, t.Key, t.Config); err != nil {
			return err
		}
		// Read the record back so the returned config and its hash reflect the
		// canonical stored shape, defaults included.
		canonical, err := store.Read(ctx, w.tx, t.Key)
		if err != nil {
			return err
		}
		t.Config = canonical
		if t.ConfigHash, err = hashConfig(configContent(canonical)); err != nil {
			return err
		}
		// Delete the old type's record only after the new config is stored, so a
		// rejected config cannot destroy the prior one. Records of different types
		// live in different tables, so the same key never collides.
		if existed && existing.Type != t.Type {
			if oldStore, ok := w.configs.Store(existing.Type); ok {
				if err := oldStore.Delete(ctx, w.tx, t.Key); err != nil {
					return err
				}
			}
		}
	}
	cfg := t.Config // Restored after the row write; rows do not store config.
	providedStatus := t.Status
	t.Config, t.Status = nil, nil
	err := w.table.NewCreate().Entry(t).Exec(ctx, w.tx)
	t.Config = cfg
	if err != nil {
		return err
	}
	stat := resolveStatus(t, providedStatus, existed)
	if providedStatus != nil {
		if err := w.status.Set(ctx, stat); err != nil {
			return err
		}
	} else if stat, err = w.healStatus(ctx, stat); err != nil {
		return err
	}
	t.Status = stat
	exists, err := w.otg.NewRetrieve().WhereIDs(otgID).Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if !exists {
		if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
			return err
		}
	}
	// Internal tasks get no group parent, keeping them out of the resource tree.
	if t.Internal || exists {
		return nil
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

// Delete deletes the task with the given key, its config record, and its associated
// status.
func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	var t Task
	found := true
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Task](key)).
		Entry(&t).
		Exec(ctx, w.tx); err != nil {
		if !errors.Is(err, query.ErrNotFound) {
			return err
		}
		found = false
	}
	if found {
		if store, ok := w.configs.Store(t.Type); ok {
			if err := store.Delete(ctx, w.tx, key); err != nil {
				return err
			}
		}
	}
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

// DeleteByRacks deletes every task attached to the given racks, along with each task's
// config record and status. It is the caller's job to reject the delete when a rack
// still holds tasks the user must remove first.
func (w Writer) DeleteByRacks(ctx context.Context, keys ...rack.Key) error {
	var tasks []Task
	if err := w.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, t *Task) (bool, error) {
			return lo.Contains(keys, t.Rack), nil
		})).
		Entries(&tasks).
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	for _, t := range tasks {
		if err := w.Delete(ctx, t.Key, true); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) Copy(
	ctx context.Context,
	key Key,
	name string,
	snapshot bool,
) (Task, error) {
	newKey := uuid.New()
	var res Task
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Task](key)).
		Change(func(_ gorp.Context, t Task) Task {
			t.Key = newKey
			t.Name = name
			t.Snapshot = snapshot
			res = t
			return t
		}).
		Exec(ctx, w.tx); err != nil {
		return Task{}, err
	}
	if err := w.status.Set(ctx, resolveStatus(&res, nil, false)); err != nil {
		return Task{}, err
	}
	if err := w.otgWriter.DefineResources(ctx, OntologyID(newKey)); err != nil {
		return Task{}, err
	}
	store, ok := w.configs.Store(res.Type)
	if !ok {
		return res, nil
	}
	if err := store.Copy(ctx, w.tx, key, newKey); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return res, nil
		}
		return Task{}, err
	}
	data, err := store.Read(ctx, w.tx, newKey)
	if err != nil {
		return Task{}, err
	}
	res.Config = data
	return res, nil
}
