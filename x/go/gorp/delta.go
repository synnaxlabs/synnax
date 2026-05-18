// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"sync"

	"github.com/synnaxlabs/x/set"
)

// deltaEntry is the per-key staged state of an index under an open
// transaction.
type deltaEntry[V comparable] struct {
	// value is the indexed field's staged value for the key. Zero when
	// deleted is true.
	value V
	// deleted reports whether the key was staged for deletion.
	deleted bool
}

// delta is the per-transaction overlay for a single secondary index.
// SK is the storable key type (must be comparable for use as a map
// key). For comparable-keyed indexes SK is the entry's primary key
// type directly; for byte-keyed indexes SK is string (because []byte
// is not comparable), and the storage backend converts at the boundary.
type delta[SK comparable, V comparable] struct {
	// state is the authoritative per-tx view: if a key is in state,
	// the committed index's view of that key is ignored during resolve.
	state map[SK]deltaEntry[V]
	// forward is the value-to-keys inverse, mirroring non-deleted
	// entries in state for O(1) resolve.
	forward map[V]set.Set[SK]
}

//nolint:unused
func newDelta[SK comparable, V comparable]() *delta[SK, V] {
	return &delta[SK, V]{
		state:   make(map[SK]deltaEntry[V]),
		forward: make(map[V]set.Set[SK]),
	}
}

func (d *delta[SK, V]) isEmpty() bool { return len(d.state) == 0 }

//nolint:unused
func (d *delta[SK, V]) stageSet(key SK, value V) {
	if prev, ok := d.state[key]; ok && !prev.deleted {
		d.removeFromForward(key, prev.value)
	}
	d.state[key] = deltaEntry[V]{value: value}
	d.addToForward(key, value)
}

//nolint:unused
func (d *delta[SK, V]) stageDelete(key SK) {
	if prev, ok := d.state[key]; ok && !prev.deleted {
		d.removeFromForward(key, prev.value)
	}
	var zero V
	d.state[key] = deltaEntry[V]{value: zero, deleted: true}
}

//nolint:unused
func (d *delta[SK, V]) addToForward(key SK, value V) {
	bucket, ok := d.forward[value]
	if !ok {
		bucket = set.New[SK]()
		d.forward[value] = bucket
	}
	bucket.Add(key)
}

//nolint:unused
func (d *delta[SK, V]) removeFromForward(key SK, value V) {
	bucket, ok := d.forward[value]
	if !ok {
		return
	}
	delete(bucket, key)
	if len(bucket) == 0 {
		delete(d.forward, value)
	}
}

// merge overlays the delta onto a committed result set for a set of
// query values, returning the effective keys under read-your-own-writes
// semantics.
func (d *delta[SK, V]) merge(committedKeys []SK, values []V) []SK {
	if d.isEmpty() {
		return committedKeys
	}
	result := set.New(committedKeys...)
	valueSet := set.New(values...)
	for k, entry := range d.state {
		if entry.deleted {
			result.Remove(k)
			continue
		}
		if !valueSet.Contains(entry.value) {
			result.Remove(k)
		}
	}
	for _, v := range values {
		for k := range d.forward[v] {
			result.Add(k)
		}
	}
	return result.Slice()
}

// deltaOverlay tracks per-tx delta state for a single secondary index
// and routes mutations either to the per-tx buffer or, when tx has no
// per-tx identity, directly to committed state via commitSet /
// commitDelete. flush runs after a successful commit with the final
// delta state; no flush runs on rollback.
type deltaOverlay[SK comparable, V comparable] struct {
	// deltaMu guards txDeltas.
	deltaMu sync.Mutex
	// txDeltas holds the per-tx staging buffer, keyed by tx identity.
	txDeltas map[*txState]*delta[SK, V]
	// commitSet applies a set directly to committed state. Invoked for
	// writes against a tx with no per-tx identity.
	commitSet func(key SK, value V)
	// commitDelete applies a delete directly to committed state.
	// Invoked for writes against a tx with no per-tx identity.
	commitDelete func(key SK)
	// flush promotes a committed tx's delta into committed state.
	// Must apply every entry atomically; no flush runs on rollback.
	flush func(*delta[SK, V])
}

//nolint:unused
func (o *deltaOverlay[SK, V]) stage(tx Tx, key SK, value V) {
	state := tx.txIdentity()
	if state == nil {
		o.commitSet(key, value)
		return
	}
	o.loadOrCreate(state).stageSet(key, value)
}

//nolint:unused
func (o *deltaOverlay[SK, V]) unstage(tx Tx, key SK) {
	state := tx.txIdentity()
	if state == nil {
		o.commitDelete(key)
		return
	}
	o.loadOrCreate(state).stageDelete(key)
}

// resolve merges committed keys with any per-tx delta, returning the
// effective key set. Always returns a non-nil slice so callers can
// distinguish "no matches" (empty) from "unbounded" (nil).
func (o *deltaOverlay[SK, V]) resolve(
	tx Tx,
	committed []SK,
	values []V,
) []SK {
	if committed == nil {
		committed = []SK{}
	}
	state := tx.txIdentity()
	if state == nil {
		return committed
	}
	o.deltaMu.Lock()
	d, ok := o.txDeltas[state]
	o.deltaMu.Unlock()
	if !ok || d.isEmpty() {
		return committed
	}
	return d.merge(committed, values)
}

//nolint:unused
func (o *deltaOverlay[SK, V]) loadOrCreate(state *txState) *delta[SK, V] {
	o.deltaMu.Lock()
	defer o.deltaMu.Unlock()
	if o.txDeltas == nil {
		o.txDeltas = make(map[*txState]*delta[SK, V])
	}
	if d, ok := o.txDeltas[state]; ok {
		return d
	}
	d := newDelta[SK, V]()
	o.txDeltas[state] = d
	state.onCleanup(func(committed bool) {
		o.deltaMu.Lock()
		d := o.txDeltas[state]
		delete(o.txDeltas, state)
		o.deltaMu.Unlock()
		if committed && d != nil && !d.isEmpty() && o.flush != nil {
			o.flush(d)
		}
	})
	return d
}
