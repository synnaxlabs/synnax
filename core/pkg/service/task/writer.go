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
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
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

// resolveStatus returns the status to persist for t. Without a provided status, a new
// task has no live instance, so it gets a quiet "not deployed" placeholder. The owner
// selects the status, so a provided key is ignored.
func resolveStatus(t *Task, provided *Status) *Status {
	if provided == nil {
		return &Status{
			Owner:   new(t.ResourceID()),
			Time:    telem.Now(),
			Name:    t.Name,
			Message: "Task not deployed",
			Variant: status.VariantDisabled,
		}
	}
	provided.Key = ""
	provided.Owner = new(t.ResourceID())
	provided.Name = t.Name
	return provided
}

// Create creates a task. The task's config is stored as a record keyed by the task's
// key in its type's config store; the task row itself carries only the config hash.
// Create returns a validation error when a task that is not a snapshot has a range, and
// an error wrapping validate.ErrValidation when a task with the key exists. A provided
// status is persisted as given. Without one, the task gets a "not deployed"
// placeholder. After the transaction commits, Create broadcasts a create action that
// carries the stored task.
func (w Writer) Create(ctx context.Context, t *Task) error {
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
	record, err := store.Prepare(ctx, w.tx, t.Key, t.Config)
	if err != nil {
		return err
	}
	if t.Config, err = record.Data(); err != nil {
		return err
	}
	t.ConfigHash = record.Hash()
	cfg := t.Config // Restored after the row write; rows do not store config.
	providedStatus := t.Status
	t.Config, t.Status = nil, nil
	err = w.table.NewCreate().Exclusive().Entry(t).Exec(ctx, w.tx)
	t.Config = cfg
	if err != nil {
		return err
	}
	if err = record.Write(ctx, w.tx); err != nil {
		return err
	}
	t.Status = resolveStatus(t, providedStatus)
	if err = w.status.Set(ctx, t.Status); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		t.Key,
		NewCreateAction(CreatePayload{Task: *t}),
	)
	return nil
}

// CreateMany creates the given tasks. It returns an error wrapping
// validate.ErrValidation when a task with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, tasks *[]Task) error {
	for i := range *tasks {
		if err := w.Create(ctx, &(*tasks)[i]); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) renameStatus(ctx context.Context, key Key, name string) error {
	// The status API can delete the status of a task that still exists.
	return errors.Skip(
		w.status.RenameOwned(ctx, ResourceID(key), name),
		query.ErrNotFound,
	)
}

// Apply applies the actions to the task with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w, then a
// set that replaces the config with the stored one when storing changed it, such as by
// filling a default, then an update of the new config hash. The config store of the
// task's type merges the config of an update. An update replaces the config of a task
// whose type has no store. Apply returns every action that it attaches. It returns an
// error wrapping validate.ErrValidation when an action is a create, an action that
// changes more than the name targets a snapshot, the config takes both path actions and
// updates, or the result fails to decode or validate. It returns an error wrapping
// patch.ErrStale when a guard does not hold or an update names a missing list entry,
// and query.ErrNotFound when no task has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	added, err := w.apply(ctx, key, acts)
	if err != nil {
		return nil, err
	}
	applied := slices.Concat(acts, added)
	w.dispatcher.Attach(w.tx, key, applied...)
	return applied, nil
}

// apply applies the actions to the task with the given key and stores the result. It
// returns the actions that only the Core computes for an edit of the config: a set
// that replaces the config with the stored one when storing changed it, then an update
// of the new config hash.
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
	_, stored := w.configs.Store(prev.Type)
	others, cursors, updates, err := splitConfig(acts, stored)
	if err != nil {
		return nil, err
	}
	next, err := Reduce(prev, others...)
	if err != nil {
		return nil, err
	}
	if err = next.Validate(); err != nil {
		return nil, err
	}
	var computed []Action
	if len(cursors) > 0 || len(updates) > 0 {
		if computed, err = w.editConfig(ctx, &next, cursors, updates); err != nil {
			return nil, err
		}
	}
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

// configField is the field of a task that its config store edits.
const configField = "config"

// splitConfig returns the actions that do not edit the task's config, then the cursors
// of the path actions that do, each past the segment that names the config, then the
// update of the config that each update holds when the config is stored. An update in
// the returned actions then holds no config and no guard on it.
func splitConfig(
	acts []Action,
	stored bool,
) ([]Action, []patch.Cursor, []patch.Update[msgpack.EncodedJSON], error) {
	var (
		others  []Action
		cursors []patch.Cursor
		updates []patch.Update[msgpack.EncodedJSON]
	)
	for _, a := range acts {
		if u, ok := a.Variant.(UpdatePayload); ok && stored {
			rest, cfg := splitUpdate(u)
			others = append(others, NewUpdateAction(rest))
			if cfg.Value != nil || len(cfg.Guards) > 0 {
				updates = append(updates, cfg)
			}
			continue
		}
		c, ok, err := a.cursor()
		if err != nil {
			return nil, nil, nil, err
		}
		if ok && c.Segment() == configField {
			cursors = append(cursors, c.Next())
		} else {
			others = append(others, a)
		}
	}
	return others, cursors, updates, nil
}

// splitUpdate returns u without its config or its guards on the config, then the update
// of the config, whose guards are the guards of u on the config, each past the segment
// that names the config.
func splitUpdate(u UpdatePayload) (UpdatePayload, patch.Update[msgpack.EncodedJSON]) {
	cfg := patch.Update[msgpack.EncodedJSON]{Value: u.Value.Config}
	var guards []patch.Guard
	for _, g := range u.Guards {
		if len(g.Path) > 0 && g.Path[0] == configField {
			g.Path = g.Path[1:]
			cfg.Guards = append(cfg.Guards, g)
		} else {
			guards = append(guards, g)
		}
	}
	u.Value.Config, u.Guards = nil, guards
	return u, cfg
}

// editConfig applies the cursors or the updates to the config of t, stores it, and
// sets the config hash of t. It returns a set that replaces the config with the stored
// one when storing changed it, then an update of the hash.
func (w Writer) editConfig(
	ctx context.Context,
	t *Task,
	cursors []patch.Cursor,
	updates []patch.Update[msgpack.EncodedJSON],
) ([]Action, error) {
	store, ok := w.configs.Store(t.Type)
	if !ok {
		return nil, errors.Wrapf(validate.ErrValidation, "unknown task type %q", t.Type)
	}
	var (
		rec     config.Record
		changed bool
		err     error
	)
	switch {
	case len(cursors) > 0 && len(updates) > 0:
		return nil, errors.Wrap(
			validate.ErrValidation,
			"a dispatch cannot edit the config with both path actions and updates",
		)
	case len(updates) > 0:
		rec, changed, err = store.Update(ctx, w.tx, t.Key, updates)
	default:
		rec, changed, err = store.Edit(ctx, w.tx, t.Key, cursors)
	}
	if err != nil {
		return nil, err
	}
	if err = rec.Write(ctx, w.tx); err != nil {
		return nil, err
	}
	t.ConfigHash = rec.Hash()
	var computed []Action
	if changed {
		stored, err := rec.Data()
		if err != nil {
			return nil, err
		}
		computed = append(computed, replaceConfig(stored))
	}
	return append(computed, setConfigHash(t.ConfigHash)), nil
}

// replaceConfig returns the set that replaces the config of a task with data.
func replaceConfig(data msgpack.EncodedJSON) Action {
	return NewSetAction(SetPayload{Path: []string{configField}, Value: data})
}

// setConfigHash returns the update that sets the config hash of a task to hash.
func setConfigHash(hash string) Action {
	return NewUpdateAction(UpdatePayload{Value: Update{ConfigHash: &hash}})
}

// syncConfig stores the hash of the config stored for the task with the given key, and
// attaches a set that replaces the config with the stored one, then an update of the
// hash.
func (w Writer) syncConfig(ctx context.Context, key Key) error {
	var t Task
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Task](key)).
		Entry(&t).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	store, ok := w.configs.Store(t.Type)
	if !ok {
		return errors.Wrapf(validate.ErrValidation, "unknown task type %q", t.Type)
	}
	stored, err := store.Read(ctx, w.tx, key)
	if err != nil {
		return err
	}
	if t.ConfigHash, err = store.Hash(ctx, w.tx, key); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(&t).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, replaceConfig(stored), setConfigHash(t.ConfigHash))
	return nil
}

// renamesOnly reports whether each action is a set of the name, or an update that sets
// only the name.
func renamesOnly(acts []Action) bool {
	return lo.EveryBy(acts, func(a Action) bool {
		switch v := a.Variant.(type) {
		case SetPayload:
			return slices.Equal(v.Path, []string{"name"})
		case UpdatePayload:
			rest := v.Value
			rest.Name = nil
			return v.Value.Name != nil && reflect.DeepEqual(rest, Update{})
		}
		return false
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
	if err := w.status.Set(ctx, resolveStatus(&res, nil)); err != nil {
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
	w.dispatcher.Attach(
		w.tx,
		newKey,
		NewCreateAction(CreatePayload{Task: res}),
	)
	return res, nil
}
