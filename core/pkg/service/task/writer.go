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
	"reflect"
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	svc        *Service
	tx         gorp.Tx
	status     status.Writer
	statuses   *status.Service
	table      *gorp.Table[Key, Task]
	configs    config.Registry
	dispatcher actions.Dispatcher[Key, Action]
	racks      rack.Writer
	guardRack  gorp.GuardFunc[rack.Key, rack.Rack]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// inTx calls f with w, or with a Writer on a transaction of its own that keeps the
// dispatch key of w when w has no transaction.
func (w Writer) inTx(ctx context.Context, f func(Writer) error) error {
	db, ok := w.tx.(*gorp.DB)
	if !ok {
		return f(w)
	}
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		bound := w.svc.NewWriter(tx)
		bound.dispatcher = w.dispatcher
		return f(bound)
	})
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
			Owner:   new(t.ResourceID()),
			Time:    telem.Now(),
			Name:    t.Name,
			Message: message,
			Variant: variant,
		}
	}
	provided.Owner = new(t.ResourceID())
	provided.Name = t.Name
	return provided
}

// healStatus returns the task's effective status, writing stat first if no row exists.
// Tasks are re-created on every scan cycle, so on a no-op update the placeholder must
// not overwrite a status the driver has already reported. An existing row takes the
// task's current name.
func (w Writer) healStatus(ctx context.Context, stat *Status) (*Status, error) {
	var existing Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](*stat.Owner)).
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
// config hash. An existing snapshot keeps its config and range. Create returns a
// validation error when a task that is not a snapshot has a range. A provided status is
// persisted as given. Without one, a new task gets a "not deployed" placeholder, and an
// existing task keeps its reported status (healed to "status unknown" if the row is
// missing). After the transaction commits, Create broadcasts a create action that
// carries the stored task.
func (w Writer) Create(ctx context.Context, t *Task) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, t) })
}

func (w Writer) create(ctx context.Context, t *Task) error {
	if t.Key == uuid.Nil() {
		t.Key = uuid.New()
	}
	if err := t.Validate(); err != nil {
		return err
	}
	if err := validateRange(t.Snapshot, t.Range); err != nil {
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
	var record *config.Record
	if existed && existing.Snapshot {
		t.Range = existing.Range
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
		rec, err := store.Prepare(ctx, w.tx, t.Key, t.Config)
		if err != nil {
			return err
		}
		t.Config = rec.Data()
		if t.ConfigHash, err = hashConfig(configContent(t.Config)); err != nil {
			return err
		}
		record = &rec
	}
	cfg := t.Config // Restored after the row write; rows do not store config.
	providedStatus := t.Status
	t.Config, t.Status = nil, nil
	err := w.table.NewCreate().Entry(t).Exec(ctx, w.tx)
	t.Config = cfg
	if err != nil {
		return err
	}
	if record != nil {
		if err := record.Write(ctx, w.tx); err != nil {
			return err
		}
		// Records of different types live in different tables, so the same key never
		// collides.
		if existed && existing.Type != t.Type {
			if oldStore, ok := w.configs.Store(existing.Type); ok {
				if err := oldStore.Delete(ctx, w.tx, t.Key); err != nil {
					return err
				}
			}
		}
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
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		t.Key,
		NewCreateAction(CreatePayload{Task: *t}),
	)
	return nil
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

func (w Writer) renameStatus(ctx context.Context, key Key, name string) error {
	// A missing status row is valid: the next Create heals it.
	return errors.Skip(
		w.status.RenameOwned(ctx, ResourceID(key), name),
		query.ErrNotFound,
	)
}

// apply applies the actions to the task with the given key and stores the result. It
// returns the actions that only the Core computes: a set of the stored config when
// storing changed it, such as by filling a default, then a set of the new config hash.
func (w Writer) apply(ctx context.Context, key Key, acts []Action) ([]Action, error) {
	var prev Task
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Task](key)).
		Entry(&prev).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	if prev.Snapshot && !renamesOnly(acts) {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"task %s is a snapshot, so only its name can change",
			key,
		)
	}
	store, ok := w.configs.Store(prev.Type)
	if !ok {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"unknown task type %q",
			prev.Type,
		)
	}
	var err error
	if prev.Config, err = store.Read(ctx, w.tx, key); err != nil {
		return nil, err
	}
	next, err := Reduce(prev, acts...)
	if err != nil {
		return nil, err
	}
	if err = next.Validate(); err != nil {
		return nil, err
	}
	rec, err := store.PrepareStrict(ctx, w.tx, key, next.Config)
	if err != nil {
		return nil, err
	}
	if err = rec.Write(ctx, w.tx); err != nil {
		return nil, err
	}
	stored := rec.Data()
	if next.ConfigHash, err = hashConfig(configContent(stored)); err != nil {
		return nil, err
	}
	computed := []Action{NewSetAction(SetPayload{
		Path:  []string{"config_hash"},
		Value: next.ConfigHash,
	})}
	if !reflect.DeepEqual(stored, next.Config) {
		computed = slices.Insert(computed, 0, NewSetAction(SetPayload{
			Path:  []string{"config"},
			Value: stored,
		}))
	}
	next.Config = nil
	if err = w.table.NewCreate().Entry(&next).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	if next.Name != prev.Name {
		if err = w.renameStatus(ctx, key, next.Name); err != nil {
			return nil, err
		}
	}
	return computed, nil
}

func renamesOnly(acts []Action) bool {
	return lo.EveryBy(acts, func(a Action) bool {
		return a.Type == ActionTypeSet &&
			a.Set != nil &&
			slices.Equal(a.Set.Path, []string{"name"})
	})
}

// Delete deletes the task with the given key, its config record, and its associated
// status.
func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Task](key)).
		Exec(ctx, w.tx)
}

func validateRange(snapshot bool, rng *ranger.Key) error {
	if rng == nil || snapshot {
		return nil
	}
	return validate.PathedError(errors.Wrap(
		validate.ErrValidation,
		"only a snapshot can sit in a range",
	), "range")
}

// DeleteRacks deletes the racks with keys, their statuses, and the internal tasks they
// hold. It returns a validation error when a rack is embedded, holds a device, or holds
// a task that is not internal.
func (w Writer) DeleteRacks(ctx context.Context, keys ...rack.Key) error {
	return w.racks.Delete(ctx, w.guardRack, keys...)
}

// Copy stores a copy of the task with the given key under a new key and name, with its
// config. A snapshot copy sits in rng, or in no range when rng is nil. After the
// transaction commits, Copy broadcasts a create action that carries the copy. It
// returns query.ErrNotFound if no task with the key exists, and a validation error when
// rng is set for a copy that is not a snapshot.
func (w Writer) Copy(
	ctx context.Context,
	key Key,
	name string,
	snapshot bool,
	rng *ranger.Key,
) (Task, error) {
	var res Task
	err := w.inTx(ctx, func(w Writer) (err error) {
		res, err = w.copy(ctx, key, name, snapshot, rng)
		return err
	})
	return res, err
}

func (w Writer) copy(
	ctx context.Context,
	key Key,
	name string,
	snapshot bool,
	rng *ranger.Key,
) (Task, error) {
	if err := validateRange(snapshot, rng); err != nil {
		return Task{}, err
	}
	newKey := uuid.New()
	var res Task
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Task](key)).
		ChangeErr(func(_ gorp.Context, t Task) (Task, error) {
			t.Key = newKey
			t.Name = name
			t.Snapshot = snapshot
			t.Range = rng
			res = t
			return t, t.Validate()
		}).
		Exec(ctx, w.tx); err != nil {
		return Task{}, err
	}
	if err := w.status.Set(ctx, resolveStatus(&res, nil, false)); err != nil {
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
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		newKey,
		NewCreateAction(CreatePayload{Task: res}),
	)
	return res, nil
}
