// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

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
)

type Writer struct {
	svc        *Service
	tx         gorp.Tx
	table      *gorp.Table[Key, Panel]
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

// Create creates a new panel. If the panel's key is uuid.Nil(), a new key is generated.
// The panel must belong to exactly one of a project and a range. An existing panel
// keeps its group, project, and range. A new panel in a group belongs to the project
// that holds the group.
func (w Writer) Create(ctx context.Context, p *Panel) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, p) })
}

func (w Writer) create(
	ctx context.Context,
	p *Panel,
) (err error) {
	var exists bool
	if p.Key == uuid.Nil() {
		p.Key = uuid.New()
	} else {
		var stored Panel
		err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Panel](p.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			p.Group, p.Project, p.Range = stored.Group, stored.Project, stored.Range
		}
	}
	if !exists && p.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, p.Group, resource.TypePanel)
		if err != nil {
			return err
		}
		p.Project = proj
	}
	// Default a freshly-created panel to a single empty leaf so action dispatchers
	// always operate against a well-formed tree.
	if p.Root.Variant == nil {
		p.Root = Node{Variant: LeafNode{Tabs: []Tab{}}}
	}
	if err := validateTree(p.Root); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if err := project.ValidateOwner(p.Project, p.Range); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		p.Key,
		NewCreateAction(CreatePayload{Panel: *p}),
	)
	return nil
}

// SetGroup moves the panels with keys into g, or out of their group when g is nil. A
// panel in a group belongs to the project that holds the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		p, err := project.GroupProject(ctx, w.tx, g, resource.TypePanel)
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

// SetProject moves the panels with keys out of their group and into the project p. It
// returns a validation error when a panel belongs to a range.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		if err := setGroup(w.table, nil, keys...).Exec(ctx, w.tx); err != nil {
			return err
		}
		w.notifyGroup(ctx, nil, keys)
		return w.setProject(ctx, p, keys...)
	})
}

// setProject moves the panels with keys into the project p, and notifies a project
// change for each one that was not in p.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Panel](keys...)).
		ChangeErr(func(_ gorp.Context, e Panel) (Panel, error) {
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

// CreateMany creates the given panels, applying the same key assignment, tree
// defaulting, and validation as Create to each.
func (w Writer) CreateMany(ctx context.Context, ps *[]Panel) error {
	for i := range *ps {
		if err := w.Create(ctx, &(*ps)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the panels with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Panel](keys...)).
		Exec(ctx, w.tx)
}
