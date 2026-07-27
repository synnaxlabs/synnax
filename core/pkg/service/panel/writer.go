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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx         gorp.Tx
	otg        ontology.Writer
	table      *gorp.Table[Key, Panel]
	dispatcher actions.Dispatcher[Key, Action]
}

// Create creates a new panel. If the panel's key is uuid.Nil, a new key is generated.
// The panel is registered with the ontology.
//
// Project-vs-draft ownership is enforced by the caller: set p.Parent to attach the
// panel to a project (project panel) or to a user (draft). When Parent is nil or
// zero, the panel has no parent in the ontology. Parent is not persisted on the
// record; parenthood lives in the ontology graph.
func (w Writer) Create(
	ctx context.Context,
	p *Panel,
) (err error) {
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	}
	// Default a freshly-created panel to a single empty leaf so action dispatchers
	// always operate against a well-formed tree.
	if p.Root.Variant == nil {
		p.Root = Node{Variant: NodeLeaf{Leaf: Leaf{Tabs: []Tab{}}}}
	}
	if err := validateTree(p.Root); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return
	}
	w.dispatcher.Notify(
		ctx, p.Key, "", []Action{NewCreateAction(CreatePayload{Panel: *p})},
	)
	otgID := p.OntologyID()
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

// Dispatch applies a sequence of actions atomically to the panel with the given key.
// After a successful update the actions are notified to the service-level dispatcher
// so subscribers (cluster signals) can broadcast them. dispatchKey identifies the
// originating batch so the originating client can recognize and skip its own echo.
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	acts []Action,
) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Panel](key)).
		ChangeErr(func(_ gorp.Context, p Panel) (Panel, error) {
			next, err := Reduce(p, acts...)
			if err != nil {
				return next, err
			}
			return next, validateTree(next.Root)
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Notify(ctx, key, dispatchKey, acts)
	return nil
}

// Delete deletes the panels with the given keys and removes their ontology entries.
func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	if err := w.table.NewDelete().Where(gorp.MatchKeys[Key, Panel](keys...)).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, OntologyIDs(keys)...)
}
