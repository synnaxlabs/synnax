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
// plot must belong to exactly one of a project and a range. A line plot in a group
// belongs to the project that holds the group. Create returns an error wrapping
// validate.ErrValidation when a line plot with lp.Key exists.
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
	lp.Lines = v8.ReconcileLines(lp.Key, lp.Channels, lp.Lines)
	lp.ApplyDefaults()
	if err := lp.Validate(); err != nil {
		return err
	}
	if err := project.ValidateOwner(lp.Project, lp.Range); err != nil {
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
// It returns a validation error when a line plot belongs to a range.
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
		ChangeErr(func(_ gorp.Context, e LinePlot) (LinePlot, error) {
			moved = append(moved, e.Key)
			e.Project = &p
			return e, project.ValidateOwner(e.Project, e.Range)
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range moved {
		w.dispatcher.Attach(
			w.tx,
			k,
			NewSetProjectAction(SetProjectPayload{Project: p}),
		)
	}
	return nil
}

func (w Writer) attachGroup(g *group.Key, keys []Key) {
	for _, k := range keys {
		w.dispatcher.Attach(
			w.tx,
			k,
			NewSetGroupAction(SetGroupPayload{Group: g}),
		)
	}
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
