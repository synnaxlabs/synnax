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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type Writer struct {
	svc        *Service
	tx         gorp.Tx
	otg        ontology.Writer
	table      *gorp.Table[Key, LinePlot]
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

// Create creates the given line plot, assigning a new key when lp.Key is zero. The line
// plot must belong to exactly one of a project and a range. An existing line plot keeps
// its group, project, and range. A new line plot in a group belongs to the project that
// holds the group.
func (w Writer) Create(ctx context.Context, lp *LinePlot) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, lp) })
}

func (w Writer) create(ctx context.Context, lp *LinePlot) error {
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
			lp.Group, lp.Project, lp.Range = stored.Group, stored.Project, stored.Range
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
	if err = project.ValidateOwner(lp.Project, lp.Range); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(lp).Exec(ctx, w.tx); err != nil {
		return err
	}
	if !exists {
		otgID := lp.ResourceID()
		if err := w.otg.DefineResources(ctx, otgID); err != nil {
			return err
		}
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		lp.Key,
		NewCreateAction(CreatePayload{LinePlot: *lp}),
	)
	return nil
}

// SetGroup moves the line plots with keys into g, or out of their group when g is nil.
// A line plot in a group belongs to the project that holds the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		p, err := project.GroupProject(ctx, w.tx, g, resource.TypeLineplot)
		if err != nil {
			return err
		}
		if err = setGroup(w.table, g, keys...).Exec(ctx, w.tx); err != nil {
			return err
		}
		w.notifyGroup(ctx, g, keys)
		if p == nil {
			return nil
		}
		return w.setProject(ctx, *p, keys...)
	})
}

// SetProject moves the line plots with keys out of their group and into the project p.
// It returns a validation error when a line plot belongs to a range.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		if err := setGroup(w.table, nil, keys...).Exec(ctx, w.tx); err != nil {
			return err
		}
		w.notifyGroup(ctx, nil, keys)
		return w.setProject(ctx, p, keys...)
	})
}

// setProject moves the line plots with keys into the project p, and notifies a project
// change for each one that was not in p.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, LinePlot](keys...)).
		ChangeErr(func(_ gorp.Context, e LinePlot) (LinePlot, error) {
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
	if err := w.table.NewDelete().Where(gorp.MatchKeys[Key, LinePlot](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, ResourceIDs(keys)...)
}
