// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/observe"
	"go.uber.org/zap"
)

// docSeedReplica is the replica the server uses to seed a document from its persisted
// text. Clients bootstrap from the server snapshot rather than seeding themselves, so
// this is the sole author of seed characters and never collides with a client's edits.
// Clients must choose a replica other than this one.
const docSeedReplica = 1

// collab maintains the authoritative CRDT document for every arc under collaborative
// edit. It observes dispatched operations, integrates them into the per-arc document,
// and serves snapshots that bootstrap joining clients into the same id space so their
// edits compose with everyone else's. The documents are held only in memory; durable
// persistence and materialization back into Arc.Text.Raw are a follow-on concern.
// collab is safe for concurrent use.
type collab struct {
	svc        *Service
	mu         sync.Mutex
	docs       map[Key]*crdt.Text
	disconnect observe.Disconnect
}

// openCollab starts the collaborative session for svc, subscribing to dispatched
// actions.
func openCollab(svc *Service) *collab {
	c := &collab{svc: svc, docs: make(map[Key]*crdt.Text)}
	c.disconnect = svc.state.OnAction(c.handle)
	return c
}

// Close stops observing actions.
func (c *collab) Close() error {
	c.disconnect()
	return nil
}

// ensure returns the authoritative document for key, lazily seeding it from the arc's
// persisted text on first use. The caller must hold c.mu.
func (c *collab) ensure(ctx context.Context, key Key) (*crdt.Text, error) {
	if doc, ok := c.docs[key]; ok {
		return doc, nil
	}
	var entry Arc
	if err := c.svc.NewRetrieve().Where(MatchKeys(key)).Entry(&entry).Exec(ctx, nil); err != nil {
		return nil, err
	}
	doc := crdt.New(docSeedReplica)
	doc.Insert(0, entry.Text.Raw)
	c.docs[key] = doc
	return doc, nil
}

// handle integrates a dispatched action batch into the arc's authoritative document.
func (c *collab) handle(ctx context.Context, scoped actions.Scoped[Key, Action]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	doc, err := c.ensure(ctx, scoped.Key)
	if err != nil {
		c.svc.cfg.L.Error("seed collaborative document", zap.Error(err))
		return
	}
	for _, a := range scoped.Actions {
		switch a.Type {
		case ActionTypeInsertChar:
			if a.InsertChar != nil {
				doc.ApplyInsert(a.InsertChar.Op)
			}
		case ActionTypeDeleteChar:
			if a.DeleteChar != nil {
				doc.ApplyDelete(a.DeleteChar.Op)
			}
		}
	}
}

// snapshot returns the operations that reconstruct the arc's authoritative document,
// seeding it from persisted text if no session is yet open.
func (c *collab) snapshot(ctx context.Context, key Key) ([]crdt.Insert, []crdt.Delete, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	doc, err := c.ensure(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	inserts, deletes := doc.Snapshot()
	return inserts, deletes, nil
}
