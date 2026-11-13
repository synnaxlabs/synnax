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
	"slices"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Panel]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates a new panel, assigning a new key when p.Key is uuid.Nil(). The panel
// must belong to a project. A panel in a group belongs to the project that holds the
// group. Create returns an error wrapping validate.ErrValidation when a panel with
// p.Key exists.
func (w Writer) Create(ctx context.Context, p *Panel) error {
	if p.Key == uuid.Nil() {
		p.Key = uuid.New()
	}
	if p.Group != nil {
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
	if err := project.ValidateOwner(p.Project); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		p.Key,
		NewCreateAction(CreatePayload{Panel: *p}),
	)
	return nil
}

// setProject moves the panels with keys that are not in the project p into p, and
// attaches a set project action for each one.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.And(
			gorp.MatchKeys[Key, Panel](keys...),
			gorp.Match(func(_ gorp.Context, e *Panel) (bool, error) {
				return e.Project == nil || *e.Project != p, nil
			}),
		)).
		Change(func(_ gorp.Context, e Panel) Panel {
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

// CreateMany creates the given panels, applying the same key assignment, tree
// defaulting, and validation as Create to each. It returns an error wrapping
// validate.ErrValidation when a panel with one of their keys exists.
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

// Apply applies a sequence of actions to the panel with the given key and attaches them
// to the transaction of w, so its commit broadcasts them with the dispatch key of w. A
// move into a group also moves the panel into the project that holds the group, and a
// move into another project moves it out of its group. Apply attaches the actions of
// each such move after acts and returns them. It returns a validation error, and stores
// nothing, if the actions leave the panel invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var added []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Panel](key)).
		ChangeErr(func(_ gorp.Context, prev Panel) (Panel, error) {
			next, err := Reduce(prev, acts...)
			if err != nil {
				return Panel{}, err
			}
			var placed project.Placement
			if placed, added, err = project.Place(
				ctx,
				w.tx,
				resource.TypePanel,
				project.Placement{Group: prev.Group, Project: prev.Project},
				project.Placement{Group: next.Group, Project: next.Project},
				newSetAction,
				newRemoveAction,
			); err != nil {
				return Panel{}, err
			}
			next.Group, next.Project = placed.Group, placed.Project
			if err = validateTree(next.Root); err != nil {
				return Panel{}, err
			}
			if err = next.Validate(); err != nil {
				return Panel{}, err
			}
			return next, project.ValidateOwner(next.Project)
		}).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, slices.Concat(acts, added)...)
	return added, nil
}
