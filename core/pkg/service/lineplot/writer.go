// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, LinePlot]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given line plot, assigning a new key when lp.Key is zero. The line
// plot must belong to a project. An existing line plot keeps its group and project. A
// new line plot in a group belongs to the project that holds the group.
func (w Writer) Create(ctx context.Context, lp *LinePlot) error {
	var (
		exists bool
		err    error
	)
	if lp.Key == uuid.Nil() {
		lp.Key = uuid.New()
	} else {
		var stored LinePlot
		err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, LinePlot](lp.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			lp.Group, lp.Project = stored.Group, stored.Project
		}
	}
	if !exists && lp.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, lp.Group, resource.TypeLineplot)
		if err != nil {
			return err
		}
		lp.Project = proj
	}
	// Materialize lines for any channel/range bindings supplied at creation so a plot
	// created with channels and ranges but no lines is fully populated.
	lp.Lines = v8.ReconcileLines(lp.Key, lp.Channels, lp.Lines)
	lp.ApplyDefaults()
	if err := lp.Validate(); err != nil {
		return err
	}
	if err = project.ValidateOwner(lp.Project); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(lp).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		lp.Key,
		NewCreateAction(CreatePayload{LinePlot: *lp}),
	)
	return nil
}

// SetGroup moves the line plots with keys into g, or out of their group when g is nil.
// A line plot in a group belongs to the project that holds the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	p, err := project.GroupProject(ctx, w.tx, g, resource.TypeLineplot)
	if err != nil {
		return err
	}
	if err = setGroup(w.table, g, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.attachGroup(g, keys)
	if p == nil {
		return nil
	}
	return w.setProject(ctx, *p, keys...)
}

// SetProject moves the line plots with keys out of their group and into the project p.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	if err := setGroup(w.table, nil, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.attachGroup(nil, keys)
	return w.setProject(ctx, p, keys...)
}

// setProject moves the line plots with keys that are not in the project p into p, and
// attaches a set project action for each one.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.And(
			gorp.MatchKeys[Key, LinePlot](keys...),
			gorp.Match(func(_ gorp.Context, e *LinePlot) (bool, error) {
				return e.Project == nil || *e.Project != p, nil
			}),
		)).
		Change(func(_ gorp.Context, e LinePlot) LinePlot {
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

// attachGroup attaches a set of the group of each line plot with keys, or a remove of
// the group when g is nil.
func (w Writer) attachGroup(g *group.Key, keys []Key) {
	for _, k := range keys {
		a := NewRemoveAction(RemovePayload{Key: "group"})
		if g != nil {
			a = NewSetAction(SetPayload{Path: []string{"group"}, Value: *g})
		}
		w.dispatcher.Attach(w.tx, k, a)
	}
}

// CreateMany creates the given line plots. If line plots with the same key already
// exist, they will be overwritten.
func (w Writer) CreateMany(ctx context.Context, plots *[]LinePlot) error {
	for i := range *plots {
		if err := w.Create(ctx, &(*plots)[i]); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, LinePlot](keys...)).
		Exec(ctx, w.tx)
}

// Apply applies a sequence of actions to the line plot with the given key and attaches
// them to the transaction of w, so its commit broadcasts them with the dispatch key of
// w. Returns a validation error, and stores nothing, if the actions leave the line plot
// invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, LinePlot](key)).
		ChangeErr(func(_ gorp.Context, p LinePlot) (LinePlot, error) {
			reduced, err := Reduce(p, acts...)
			if err != nil {
				return LinePlot{}, err
			}
			if err = reduced.Validate(); err != nil {
				return LinePlot{}, err
			}
			return reduced, nil
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}
