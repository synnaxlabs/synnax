// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"
	"slices"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create, update, and delete tables within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method.
type Writer struct {
	tx         gorp.Tx
	tbl        *gorp.Table[Key, Table]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given table, assigning a new key when t.Key is zero. The table
// must belong to a project. A table in a group belongs to the project that holds the
// group. Create returns an error wrapping validate.ErrValidation when a table with
// t.Key exists.
func (w Writer) Create(ctx context.Context, t *Table) error {
	if t.Key == uuid.Nil() {
		t.Key = uuid.New()
	}
	if t.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, t.Group, resource.TypeTable)
		if err != nil {
			return err
		}
		t.Project = proj
	}
	if err := t.Validate(); err != nil {
		return err
	}
	if err := project.ValidateOwner(t.Project); err != nil {
		return err
	}
	if err := validateUniqueKeys(*t); err != nil {
		return err
	}
	if err := w.tbl.NewCreate().Exclusive().Entry(t).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		t.Key,
		NewCreateAction(CreatePayload{Table: *t}),
	)
	return nil
}

// setProject moves the tables with keys that are not in the project p into p, and
// attaches a set project action for each one.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.tbl.NewUpdate().
		Where(gorp.And(
			gorp.MatchKeys[Key, Table](keys...),
			gorp.Match(func(_ gorp.Context, e *Table) (bool, error) {
				return e.Project == nil || *e.Project != p, nil
			}),
		)).
		Change(func(_ gorp.Context, e Table) Table {
			moved = append(moved, e.Key)
			e.Project = &p
			return e
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range moved {
		w.dispatcher.Attach(
			w.tx,
			k,
			NewSetAction(SetPayload{Path: []string{"project"}, Value: p}),
		)
	}
	return nil
}

// CreateMany creates the given tables. It returns an error wrapping
// validate.ErrValidation when a table with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, tables *[]Table) error {
	for i := range *tables {
		if err := w.Create(ctx, &(*tables)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the tables with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.tbl.NewDelete().
		Where(gorp.MatchKeys[Key, Table](keys...)).Exec(ctx, w.tx)
}

func validateUniqueKeys(t Table) error {
	if k, ok := duplicateKey(t.Rows, func(r Row) string { return r.Key }); ok {
		return errors.Wrapf(validate.ErrValidation, "duplicate row key %q", k)
	}
	if k, ok := duplicateKey(t.Columns, func(c Column) string { return c.Key }); ok {
		return errors.Wrapf(validate.ErrValidation, "duplicate column key %q", k)
	}
	return nil
}

func duplicateKey[T any](items []T, keyOf func(T) string) (string, bool) {
	seen := make(set.Set[string], len(items))
	for _, item := range items {
		k := keyOf(item)
		if seen.Contains(k) {
			return k, true
		}
		seen.Add(k)
	}
	return "", false
}

// Apply applies a sequence of actions to the table with the given key and attaches them
// to the transaction of w, so its commit broadcasts them with the dispatch key of w. A
// move into a group also moves the table into the project that holds the group, and a
// move into another project moves it out of its group. Apply attaches the actions of
// each such move after acts and returns them. It returns a validation error, and stores
// nothing, if the actions leave the table invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var added []Action
	if err := w.tbl.NewUpdate().Where(gorp.MatchKeys[Key, Table](key)).
		ChangeErr(func(_ gorp.Context, prev Table) (Table, error) {
			next, err := Reduce(prev, acts...)
			if err != nil {
				return Table{}, err
			}
			var placed project.Placement
			if placed, added, err = project.Place(
				ctx,
				w.tx,
				resource.TypeTable,
				project.Placement{Group: prev.Group, Project: prev.Project},
				project.Placement{Group: next.Group, Project: next.Project},
				newSetAction,
				newRemoveAction,
			); err != nil {
				return Table{}, err
			}
			next.Group, next.Project = placed.Group, placed.Project
			if err = next.Validate(); err != nil {
				return Table{}, err
			}
			if err = validateUniqueKeys(next); err != nil {
				return Table{}, err
			}
			return next, project.ValidateOwner(next.Project)
		}).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, slices.Concat(acts, added)...)
	return added, nil
}
