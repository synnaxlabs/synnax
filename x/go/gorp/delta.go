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
// transaction. value is the indexed field's current staged value for
// the key; deleted indicates the key was staged for deletion.
type deltaEntry[V comparable] struct {
	value   V
	deleted bool
}

// delta is the per-transaction overlay for a single secondary index.
// SK is the storable key type (must be comparable for use as a map
// key). For Lookup and Sorted, SK is the entry's primary key type
// directly. For BytesLookup, SK is string (because []byte is not
// comparable), and the BytesLookup wrapper converts at the boundary.
//
// state is authoritative: if a key is in state, the committed index's
// view of that key is not trusted during resolve. forward is the
// value-to-key inverse for fast resolve.
type delta[SK comparable, V comparable] struct {
	state   map[SK]deltaEntry[V]
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

// deltaOverlay tracks per-tx delta state for a single secondary index.
// stage / unstage record pending mutations against the open
// transaction; resolve merges the staged state into a committed result
// set under read-your-own-writes semantics.
//
// flush, when non-nil, runs after a successful commit with the final
// delta state. Owners can use it to promote staged mutations into
// committed storage; no flush runs on rollback.
type deltaOverlay[SK comparable, V comparable] struct {
	deltaMu  sync.Mutex
	txDeltas map[*txState]*delta[SK, V]
	flush    func(*delta[SK, V])
}

//nolint:unused
func (o *deltaOverlay[SK, V]) stage(tx Tx, key SK, value V) {
	state := tx.txIdentity()
	if state == nil {
		return
	}
	o.loadOrCreate(state).stageSet(key, value)
}

//nolint:unused
func (o *deltaOverlay[SK, V]) unstage(tx Tx, key SK) {
	state := tx.txIdentity()
	if state == nil {
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
