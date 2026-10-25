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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx          gorp.Tx
	otgWriter   ontology.Writer
	otg         *ontology.Ontology
	group       group.Group
	status      status.Writer
	table       *gorp.Table[Key, Task]
	configs     config.Registry
	dispatcher  actions.Dispatcher[Key, Action]
	dispatchKey string
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatchKey = dispatchKey
	return w
}

// notify broadcasts the actions on the task with the given key after the transaction
// commits.
func (w Writer) notify(ctx context.Context, key Key, acts ...Action) {
	gorp.OnCommit(w.tx, func() { w.dispatcher.Notify(ctx, key, w.dispatchKey, acts) })
}

// resolveStatus returns the status to persist for t. Without a provided status, a task
// created for the first time provably has no live instance, so it gets a quiet "not
// deployed" placeholder; an existing task can be running, so its placeholder stays a
// "status unknown" warning.
func resolveStatus(t *Task, provided *Status, existed bool) *Status {
	if provided == nil {
		message := "Task status unknown"
		variant := status.VariantWarning
		if !existed {
			message = "Task not deployed"
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
// not overwrite a status the driver has already reported. An existing row takes the
// task's current name.
func (w Writer) healStatus(ctx context.Context, stat *Status) (*Status, error) {
	var existing Status
	err := gorp.NewRetrieve[string, Status]().
		Where(gorp.MatchKeys[string, Status](stat.Key)).
		Entry(&existing).
		Exec(ctx, w.tx)
	if err == nil {
		if existing.Name != stat.Name {
			if err = w.status.Rename(ctx, existing.Key, stat.Name); err != nil {
				return nil, err
			}
			existing.Name = stat.Name
		}
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
// a "not deployed" placeholder, and an existing task keeps its reported status (healed
// to "status unknown" if the row is missing). After the transaction commits, Create
// broadcasts a create action that carries the stored task.
func (w Writer) Create(ctx context.Context, t *Task) error {
	if t.Key == uuid.Nil() {
		t.Key = uuid.New()
	}
	if err := t.Validate(); err != nil {
		return err
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
		stored, err := store.Write(ctx, w.tx, t.Key, t.Config)
		if err != nil {
			return err
		}
		t.Config = stored
		if t.ConfigHash, err = hashConfig(configContent(stored)); err != nil {
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
	w.notify(ctx, t.Key, NewCreateAction(CreatePayload{Task: *t}))
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

// Rename renames the task with the given key and its status, keeping every other field,
// including its config. After the transaction commits, Rename broadcasts a set of the
// task's name. It returns query.ErrNotFound if no task with the key exists and a
// validation error if name is empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	v := validate.New("task.task")
	v.NotEmptyString("name", name)
	if err := v.Error(); err != nil {
		return err
	}
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Task](key)).
		Change(func(_ gorp.Context, t Task) Task {
			t.Name = name
			return t
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.renameStatus(ctx, key, name); err != nil {
		return err
	}
	w.notify(ctx, key, NewSetAction(SetPayload{Path: []string{"name"}, Value: name}))
	return nil
}

func (w Writer) renameStatus(ctx context.Context, key Key, name string) error {
	// A missing status row is valid: the next Create heals it.
	return errors.Skip(
		w.status.Rename(ctx, OntologyID(key).String(), name),
		query.ErrNotFound,
	)
}

// apply applies the actions to the task with the given key, stores the result, and
// returns the hash of the task's new config.
func (w Writer) apply(ctx context.Context, key Key, acts []Action) (string, error) {
	var prev Task
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Task](key)).
		Entry(&prev).
		Exec(ctx, w.tx); err != nil {
		return "", err
	}
	if prev.Snapshot {
		return "", errors.Wrapf(
			validate.ErrValidation, "task %s is a snapshot and cannot change", key,
		)
	}
	store, ok := w.configs.Store(prev.Type)
	if !ok {
		return "", errors.Wrapf(
			validate.ErrValidation,
			"unknown task type %q",
			prev.Type,
		)
	}
	var err error
	if prev.Config, err = store.Read(ctx, w.tx, key); err != nil {
		return "", err
	}
	next, err := Reduce(prev, acts...)
	if err != nil {
		return "", err
	}
	if err = next.Validate(); err != nil {
		return "", err
	}
	stored, err := store.WriteStrict(ctx, w.tx, key, next.Config)
	if err != nil {
		return "", err
	}
	if next.ConfigHash, err = hashConfig(configContent(stored)); err != nil {
		return "", err
	}
	next.Config = nil
	if err = w.table.NewCreate().Entry(&next).Exec(ctx, w.tx); err != nil {
		return "", err
	}
	if next.Name != prev.Name {
		if err = w.renameStatus(ctx, key, next.Name); err != nil {
			return "", err
		}
	}
	return next.ConfigHash, nil
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

// Copy stores a copy of the task with the given key under a new key and name, with its
// config. After the transaction commits, Copy broadcasts a create action that carries
// the copy. It returns query.ErrNotFound if no task with the key exists.
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
		ChangeErr(func(_ gorp.Context, t Task) (Task, error) {
			t.Key = newKey
			t.Name = name
			t.Snapshot = snapshot
			res = t
			return t, t.Validate()
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
	if store, ok := w.configs.Store(res.Type); ok {
		if err := store.Copy(ctx, w.tx, key, newKey); err == nil {
			if res.Config, err = store.Read(ctx, w.tx, newKey); err != nil {
				return Task{}, err
			}
		} else if !errors.Is(err, query.ErrNotFound) {
			return Task{}, err
		}
	}
	w.notify(ctx, newKey, NewCreateAction(CreatePayload{Task: res}))
	return res, nil
}
