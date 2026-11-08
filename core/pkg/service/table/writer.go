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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create, update, and delete tables within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly on
// the database.
type Writer struct {
	svc        *Service
	tx         gorp.Tx
	tbl        *gorp.Table[Key, Table]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the create
// broadcasts it emits, so the client that sent the create recognizes its own echo.
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

// Create creates the given table, assigning a new key when t.Key is zero. The table
// must belong to exactly one of a project and a range. An existing table keeps its
// group, project, and range. A new table in a group belongs to the project that holds
// the group.
func (w Writer) Create(ctx context.Context, t *Table) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, t) })
}

func (w Writer) create(ctx context.Context, t *Table) error {
	var (
		exists bool
		err    error
	)
	if t.Key == uuid.Nil() {
		t.Key = uuid.New()
	} else {
		var stored Table
		err = w.tbl.NewRetrieve().
			Where(gorp.MatchKeys[Key, Table](t.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			t.Group, t.Project, t.Range = stored.Group, stored.Project, stored.Range
		}
	}
	if !exists && t.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, t.Group, resource.TypeTable)
		if err != nil {
			return err
		}
		t.Project = proj
	}
	if err = t.Validate(); err != nil {
		return err
	}
	if err = project.ValidateOwner(t.Project, t.Range); err != nil {
		return err
	}
	if err = validateUniqueKeys(*t); err != nil {
		return err
	}
	if err = w.tbl.NewCreate().Entry(t).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		t.Key,
		NewCreateAction(CreatePayload{Table: *t}),
	)
	return nil
}

// SetGroup moves the tables with keys into g, or out of their group when g is nil. A
// table in a group belongs to the project that holds the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		p, err := project.GroupProject(ctx, w.tx, g, resource.TypeTable)
		if err != nil {
			return err
		}
		if err = setGroup(w.tbl, g, keys...).Exec(ctx, w.tx); err != nil {
			return err
		}
		w.notifyGroup(ctx, g, keys)
		if p == nil {
			return nil
		}
		return w.setProject(ctx, *p, keys...)
	})
}

// SetProject moves the tables with keys out of their group and into the project p. It
// returns a validation error when a table belongs to a range.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		if err := setGroup(w.tbl, nil, keys...).Exec(ctx, w.tx); err != nil {
			return err
		}
		w.notifyGroup(ctx, nil, keys)
		return w.setProject(ctx, p, keys...)
	})
}

// setProject moves the tables with keys into the project p, and notifies a project
// change for each one that was not in p.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.tbl.NewUpdate().
		Where(gorp.MatchKeys[Key, Table](keys...)).
		ChangeErr(func(_ gorp.Context, e Table) (Table, error) {
			if e.Project == nil || *e.Project != p {
				moved = append(moved, e.Key)
			}
			e.Project = &p
			return e, project.ValidateOwner(e.Project, e.Range)
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range moved {
		w.dispatcher.NotifyOnCommit(
			ctx,
			w.tx,
			k,
			NewSetProjectAction(SetProjectPayload{Project: p}),
		)
	}
	return nil
}

func (w Writer) notifyGroup(ctx context.Context, g *group.Key, keys []Key) {
	for _, k := range keys {
		w.dispatcher.NotifyOnCommit(
			ctx,
			w.tx,
			k,
			NewSetGroupAction(SetGroupPayload{Group: g}),
		)
	}
}

// CreateMany creates the given tables. If tables with the same key already exist, they
// will be overwritten.
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
