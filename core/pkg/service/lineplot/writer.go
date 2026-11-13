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
	"slices"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
// plot must belong to a project. A line plot in a group belongs to the project that
// holds the group. Create returns an error wrapping validate.ErrValidation when a line
// plot with lp.Key exists.
func (w Writer) Create(ctx context.Context, lp *LinePlot) error {
	if lp.Key == uuid.Nil() {
		lp.Key = uuid.New()
	}
	if lp.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, lp.Group, resource.TypeLineplot)
		if err != nil {
			return err
		}
		lp.Project = proj
	}
	// Materialize lines for any channel/range bindings supplied at creation so a plot
	// created with channels and ranges but no lines is fully populated.
	lp.Lines = v10.ReconcileLines(lp.Key, lp.Channels, lp.Lines)
	if err := lp.Validate(); err != nil {
		return err
	}
	if err := project.CheckMember(lp.Project); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(lp).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		lp.Key,
		NewCreateAction(CreatePayload{LinePlot: *lp}),
	)
	return nil
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

// CreateMany creates the given line plots. It returns an error wrapping
// validate.ErrValidation when a line plot with one of their keys exists.
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
// w. A move into a group also moves the line plot into the project that holds the
// group, and a move into another project moves it out of its group. Apply attaches the
// actions of each such move after acts and returns them. It returns a validation error,
// and stores nothing, if the actions leave the line plot invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var added []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, LinePlot](key)).
		ChangeErr(func(_ gorp.Context, prev LinePlot) (LinePlot, error) {
			next, err := Reduce(prev, acts...)
			if err != nil {
				return LinePlot{}, err
			}
			var placed project.Placement
			if placed, added, err = project.Place(
				ctx,
				w.tx,
				resource.TypeLineplot,
				project.Placement{Group: prev.Group, Project: prev.Project},
				project.Placement{Group: next.Group, Project: next.Project},
				newSetAction,
				newRemoveAction,
			); err != nil {
				return LinePlot{}, err
			}
			next.Group, next.Project = placed.Group, placed.Project
			if err = next.Validate(); err != nil {
				return LinePlot{}, err
			}
			return next, project.CheckMember(next.Project)
		}).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, slices.Concat(acts, added)...)
	return added, nil
}
