// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// lookupDeltaEntry is the per-key staged state of a Lookup or Sorted index
// under an open transaction. value is the indexed field's current staged
// value for the key; deleted indicates the key was staged for deletion (in
// which case value is ignored).
type lookupDeltaEntry[V comparable] struct {
	value   V
	deleted bool
}

// lookupDelta is the per-transaction overlay for a single Lookup or Sorted
// index. It maintains two maps that together describe every pending
// mutation staged against the index within an open tx.
//
// state maps each touched key to the latest staged entry. Its presence
// is authoritative: if a key is in state, the committed index's view of
// that key is NOT trusted during resolve — the delta is the source of
// truth. If state marks the key as deleted, it is excluded from all
// result sets. If state holds a non-deleted value V, the key is a member
// of the V bucket and of no other bucket, regardless of what the
// committed index says.
//
// forward is the indexable view: a value V maps to the set of staged
// keys whose latest staged state is {value: V}. It exists so resolveTx
// can find newly-staged matches in O(1) per queried value instead of
// scanning the full state map. forward stays in sync with state: every
// mutation updates both.
//
// Both maps are typed end-to-end (K, V are concrete Go types at every
// call site). There is no sync.Map, no any, no reflection. Per-tx
// deltas are accessed only through the owning tx's goroutine, so no
// internal locking is needed; coarser locking on the owning index's
// txDeltas map handles cross-tx contention.
type lookupDelta[K comparable, V comparable] struct {
	state   map[K]lookupDeltaEntry[V]
	forward map[V]map[K]struct{}
}

func newLookupDelta[K comparable, V comparable]() *lookupDelta[K, V] {
	return &lookupDelta[K, V]{
		state:   make(map[K]lookupDeltaEntry[V]),
		forward: make(map[V]map[K]struct{}),
	}
}

// isEmpty reports whether no mutations have been staged. Used by
// resolveTx to short-circuit the merge when there's nothing to overlay.
func (d *lookupDelta[K, V]) isEmpty() bool { return len(d.state) == 0 }

// stageSet records that key now maps to value in the open tx. If the
// key was previously staged (set or deleted), its prior forward-bucket
// membership is removed first so the forward view stays in sync.
func (d *lookupDelta[K, V]) stageSet(key K, value V) {
	if prev, ok := d.state[key]; ok && !prev.deleted {
		d.removeFromForward(key, prev.value)
	}
	d.state[key] = lookupDeltaEntry[V]{value: value}
	d.addToForward(key, value)
}

// stageDelete records that key is deleted in the open tx. If the key
// had a prior non-deleted staged value, its forward-bucket entry is
// removed.
func (d *lookupDelta[K, V]) stageDelete(key K) {
	if prev, ok := d.state[key]; ok && !prev.deleted {
		d.removeFromForward(key, prev.value)
	}
	var zero V
	d.state[key] = lookupDeltaEntry[V]{value: zero, deleted: true}
}

func (d *lookupDelta[K, V]) addToForward(key K, value V) {
	bucket, ok := d.forward[value]
	if !ok {
		bucket = make(map[K]struct{})
		d.forward[value] = bucket
	}
	bucket[key] = struct{}{}
}

func (d *lookupDelta[K, V]) removeFromForward(key K, value V) {
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
// semantics. The input committedKeys is the result of calling Get on
// the committed index for the same values, captured under the index's
// read lock and released before this call.
//
// The algorithm is:
//  1. Build a set from committedKeys.
//  2. For every (k, entry) in state:
//     - If entry.deleted, remove k from the set (the tx erased it).
//     - Else if entry.value is NOT among query values, remove k (the
//     tx moved it to a different bucket than the one we queried).
//  3. For every v in query values, union forward[v] into the set (any
//     newly staged keys under this value become visible).
//  4. Return the set as a slice.
//
// Ordering is not guaranteed. This matches the behavior of the
// committed index's Get, which also returns keys in an unspecified
// (map iteration) order for multi-value queries.
func (d *lookupDelta[K, V]) merge(committedKeys []K, values []V) []K {
	if d.isEmpty() {
		return committedKeys
	}
	result := make(map[K]struct{}, len(committedKeys))
	for _, k := range committedKeys {
		result[k] = struct{}{}
	}
	valueSet := make(map[V]struct{}, len(values))
	for _, v := range values {
		valueSet[v] = struct{}{}
	}
	for k, entry := range d.state {
		if entry.deleted {
			delete(result, k)
			continue
		}
		if _, wanted := valueSet[entry.value]; !wanted {
			delete(result, k)
		}
	}
	for _, v := range values {
		for k := range d.forward[v] {
			result[k] = struct{}{}
		}
	}
	out := make([]K, 0, len(result))
	for k := range result {
		out = append(out, k)
	}
	return out
}

// bytesLookupDeltaEntry is the byte-keyed analogue of lookupDeltaEntry.
// It stores the original []byte key alongside the staged value so
// resolveTx can return the caller's bytes without a second copy pass
// through the state map.
type bytesLookupDeltaEntry[V comparable] struct {
	key     []byte
	value   V
	deleted bool
}

// bytesLookupDelta is the per-transaction overlay for a single
// BytesLookup index. The internal maps are keyed by string([]byte)
// because []byte is not strictly comparable. Semantics mirror
// lookupDelta exactly: state is authoritative per key, forward is the
// value-to-key inverse.
//
// On stage and resolve, the conversion string([]byte) is a no-alloc
// cast in modern Go. The original []byte is stored in the entry so we
// can return it verbatim without rebuilding it from the map key.
type bytesLookupDelta[V comparable] struct {
	state   map[string]bytesLookupDeltaEntry[V]
	forward map[V]map[string]struct{}
}

func newBytesLookupDelta[V comparable]() *bytesLookupDelta[V] {
	return &bytesLookupDelta[V]{
		state:   make(map[string]bytesLookupDeltaEntry[V]),
		forward: make(map[V]map[string]struct{}),
	}
}

func (d *bytesLookupDelta[V]) isEmpty() bool { return len(d.state) == 0 }

func (d *bytesLookupDelta[V]) stageSet(key []byte, value V) {
	sk := string(key)
	if prev, ok := d.state[sk]; ok && !prev.deleted {
		d.removeFromForward(sk, prev.value)
	}
	keyCopy := append([]byte(nil), key...)
	d.state[sk] = bytesLookupDeltaEntry[V]{key: keyCopy, value: value}
	d.addToForward(sk, value)
}

func (d *bytesLookupDelta[V]) stageDelete(key []byte) {
	sk := string(key)
	if prev, ok := d.state[sk]; ok && !prev.deleted {
		d.removeFromForward(sk, prev.value)
	}
	var zero V
	keyCopy := append([]byte(nil), key...)
	d.state[sk] = bytesLookupDeltaEntry[V]{key: keyCopy, value: zero, deleted: true}
}

func (d *bytesLookupDelta[V]) addToForward(sk string, value V) {
	bucket, ok := d.forward[value]
	if !ok {
		bucket = make(map[string]struct{})
		d.forward[value] = bucket
	}
	bucket[sk] = struct{}{}
}

func (d *bytesLookupDelta[V]) removeFromForward(sk string, value V) {
	bucket, ok := d.forward[value]
	if !ok {
		return
	}
	delete(bucket, sk)
	if len(bucket) == 0 {
		delete(d.forward, value)
	}
}

// merge overlays the delta onto a committed result set. See
// lookupDelta.merge for the algorithm; this is the byte-keyed
// counterpart. Returned slices carry fresh []byte copies that the
// caller owns.
func (d *bytesLookupDelta[V]) merge(committedKeys [][]byte, values []V) [][]byte {
	if d.isEmpty() {
		return committedKeys
	}
	result := make(map[string][]byte, len(committedKeys))
	for _, k := range committedKeys {
		result[string(k)] = k
	}
	valueSet := make(map[V]struct{}, len(values))
	for _, v := range values {
		valueSet[v] = struct{}{}
	}
	for sk, entry := range d.state {
		if entry.deleted {
			delete(result, sk)
			continue
		}
		if _, wanted := valueSet[entry.value]; !wanted {
			delete(result, sk)
		}
	}
	for _, v := range values {
		for sk := range d.forward[v] {
			if _, alreadyIn := result[sk]; alreadyIn {
				continue
			}
			result[sk] = append([]byte(nil), d.state[sk].key...)
		}
	}
	out := make([][]byte, 0, len(result))
	for _, k := range result {
		out = append(out, k)
	}
	return out
}
