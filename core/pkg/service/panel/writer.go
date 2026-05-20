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
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
	table *gorp.Table[Key, Panel]
	// actionObserver is notified after a successful Dispatch so the cluster
	// signals subsystem can broadcast the action vector on the panel channel.
	// Nil when the service is opened without a Signals provider.
	actionObserver observe.Observer[ScopedAction]
	// seq points at the service-level monotonic sequence counter. Each Dispatch
	// increments it once and stamps the value onto the emitted ScopedAction so
	// clients can dedupe echoes by ordering rather than session identity.
	seq *atomic.Uint64
}

// Create creates a new panel. If the panel's key is uuid.Nil, a new key is generated.
// The panel is registered with the ontology and parented to the panel group.
//
// Project-vs-draft ownership is enforced by the caller: pass parentID to attach the
// panel to a project (project panel) or to a user (draft). When parentID is the
// zero value, the panel is parented only to the root panel group.
func (w Writer) Create(
	ctx context.Context,
	p *Panel,
	parentID ontology.ID,
) (err error) {
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	}
	// Default a freshly-created panel to a single empty leaf so action dispatchers
	// always operate against a well-formed tree.
	if p.Root.Leaf == nil && p.Root.Split == nil {
		p.Root = Node{Leaf: &Leaf{Tabs: []Tab{}}}
	}
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(p.Key)
	if err := w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	if err := w.otg.DefineRelationship(
		ctx,
		w.group.OntologyID(),
		ontology.RelationshipTypeParentOf,
		otgID,
	); err != nil {
		return err
	}
	if (parentID != ontology.ID{}) {
		if err := w.otg.DefineRelationship(
			ctx,
			parentID,
			ontology.RelationshipTypeParentOf,
			otgID,
		); err != nil {
			return err
		}
	}
	return nil
}

// Rename renames the panel to the given name.
func (w Writer) Rename(
	ctx context.Context,
	key Key,
	name string,
) error {
	return w.table.NewUpdate().Where(gorp.MatchKeys[Key, Panel](key)).
		Change(func(_ gorp.Context, p Panel) Panel {
			p.Name = name
			return p
		}).Exec(ctx, w.tx)
}

// Dispatch applies a sequence of actions atomically to the panel with the given key.
// After a successful update the actions are notified to the service-level observer
// so subscribers (cluster signals) can broadcast them. sessionKey identifies the
// originating client so subscribers can self-dedup.
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	sessionKey string,
	actions []Action,
) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Panel](key)).
		ChangeErr(func(_ gorp.Context, p Panel) (Panel, error) {
			return Reduce(p, actions...)
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	if w.actionObserver != nil {
		w.actionObserver.Notify(ctx, ScopedAction{
			Key:        key,
			SessionKey: sessionKey,
			Seq:        w.seq.Add(1),
			Actions:    actions,
		})
	}
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
	for _, key := range keys {
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}
