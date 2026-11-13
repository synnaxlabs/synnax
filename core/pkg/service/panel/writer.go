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

// Create creates a new panel. If the panel's key is uuid.Nil(), a new key is generated.
// The panel must belong to a project. An existing panel keeps its group and project. A
// new panel in a group belongs to the project that holds the group.
func (w Writer) Create(
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
			p.Group, p.Project = stored.Group, stored.Project
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
	if err := project.ValidateOwner(p.Project); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		p.Key,
		NewCreateAction(CreatePayload{Panel: *p}),
	)
	return nil
}

// SetGroup moves the panels with keys into g, or out of their group when g is nil. A
// panel in a group belongs to the project that holds the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	p, err := project.GroupProject(ctx, w.tx, g, resource.TypePanel)
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

// SetProject moves the panels with keys out of their group and into the project p.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	if err := setGroup(w.table, nil, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.attachGroup(nil, keys)
	return w.setProject(ctx, p, keys...)
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

// attachGroup attaches a set of the group of each panel with keys, or a remove of the
// group when g is nil.
func (w Writer) attachGroup(g *group.Key, keys []Key) {
	for _, k := range keys {
		a := NewRemoveAction(RemovePayload{Key: "group"})
		if g != nil {
			a = NewSetAction(SetPayload{Path: []string{"group"}, Value: *g})
		}
		w.dispatcher.Attach(w.tx, k, a)
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

// Apply applies a sequence of actions to the panel with the given key and attaches them
// to the transaction of w, so its commit broadcasts them with the dispatch key of w.
// Returns a validation error, and stores nothing, if the actions leave the panel
// invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Panel](key)).
		ChangeErr(func(_ gorp.Context, p Panel) (Panel, error) {
			reduced, err := Reduce(p, acts...)
			if err != nil {
				return Panel{}, err
			}
			if err = validateTree(reduced.Root); err != nil {
				return Panel{}, err
			}
			if err = reduced.Validate(); err != nil {
				return Panel{}, err
			}
			return reduced, nil
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}
