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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type Writer struct {
	tx         gorp.Tx
	otg        ontology.Writer
	table      *gorp.Table[Key, Panel]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the create
// broadcasts it emits, so the client that sent the create recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates a new panel. If the panel's key is uuid.Nil(), a new key is generated.
// An existing panel keeps its group. The panel is registered with the ontology.
//
// Set p.Parent to attach the panel to a parent resource in the ontology; when nil or
// zero, the panel has no parent. Parent is not persisted on the record; parenthood
// lives in the ontology graph.
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
			p.Group = stored.Group
		}
	}
	if !exists {
		if err := group.CheckMember(
			ctx,
			w.tx,
			p.Group,
			resource.TypePanel,
			nil,
		); err != nil {
			return err
		}
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
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := p.ResourceID()
	if err := w.otg.DefineResources(ctx, otgID); err != nil {
		return err
	}
	if p.Parent != nil && !p.Parent.IsZero() {
		if err := w.otg.DefineRelationships(
			ctx,
			*p.Parent,
			ontology.RelationshipTypeParentOf,
			otgID,
		); err != nil {
			return err
		}
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		p.Key,
		NewCreateAction(CreatePayload{Panel: *p}),
	)
	return nil
}

// SetGroup moves the panels with keys into g, or out of their group when g is nil.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	if err := group.CheckMember(ctx, w.tx, g, resource.TypePanel, nil); err != nil {
		return err
	}
	if err := setGroup(w.table, g, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range keys {
		w.dispatcher.NotifyOnCommit(
			ctx,
			w.tx,
			k,
			NewSetGroupAction(SetGroupPayload{Group: g}),
		)
	}
	return nil
}

// CreateMany creates the given panels, applying the same key assignment, tree
// defaulting, validation, and parenting as Create to each.
func (w Writer) CreateMany(ctx context.Context, ps *[]Panel) error {
	for i := range *ps {
		if err := w.Create(ctx, &(*ps)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the panels with the given keys and removes their ontology entries.
func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Panel](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, ResourceIDs(keys)...)
}
