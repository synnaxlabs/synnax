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
	"context"
	"sync"
)

// BytesLookup is the byte-keyed analogue of Lookup. It exists for tables
// whose primary key is []byte (and therefore not strictly comparable, so it
// cannot satisfy IndexKey). The shape and semantics mirror Lookup exactly:
// an in-memory exact-match index on a field of type V extracted from
// entries of type E. Construct with NewBytesLookup and register on a Table
// via TableConfig.Indexes.
//
// Membership probing uses string([]byte) under the hood, which costs a
// no-alloc conversion in modern Go but loses the strict-comparability
// guarantee that Lookup preserves on its happy path. Use Lookup whenever K
// is comparable; reach for BytesLookup only when the table key is genuinely
// []byte (e.g. composite keys encoded inline).
type BytesLookup[E Entry[[]byte], V comparable] struct {
	name    string
	extract func(e *E) V

	mu      sync.RWMutex
	storage *bytesLookupStorage[V]
	reverse map[string]V

	// deltaMu guards txDeltas. See Lookup.deltaMu for rationale.
	deltaMu  sync.Mutex
	txDeltas map[*txState]*bytesLookupDelta[V]
}

// NewBytesLookup constructs a BytesLookup index with the given display name
// and extract function. The returned index is empty; register it on a Table
// through TableConfig.Indexes to populate it from the existing table
// contents and keep it in sync with future writes.
func NewBytesLookup[E Entry[[]byte], V comparable](
	name string,
	extract func(e *E) V,
) *BytesLookup[E, V] {
	return &BytesLookup[E, V]{
		name:    name,
		extract: extract,
		storage: newBytesLookupStorage[V](),
		reverse: make(map[string]V),
	}
}

// Name implements Index.
func (l *BytesLookup[E, V]) Name() string { return l.name }

func (l *BytesLookup[E, V]) populate() (error, func(E), func()) {
	l.mu.Lock()
	insert := func(entry E) {
		key := entry.GorpKey()
		value := l.extract(&entry)
		l.storage.put(key, value)
		l.reverse[string(key)] = value
	}
	finish := func() { l.mu.Unlock() }
	return nil, insert, finish
}

func (l *BytesLookup[E, V]) set(key []byte, entry E) {
	l.mu.Lock()
	defer l.mu.Unlock()
	newValue := l.extract(&entry)
	if oldValue, existed := l.reverse[string(key)]; existed {
		if oldValue == newValue {
			return
		}
		l.storage.remove(key, oldValue)
	}
	l.storage.put(key, newValue)
	l.reverse[string(key)] = newValue
}

func (l *BytesLookup[E, V]) delete(key []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	skey := string(key)
	oldValue, existed := l.reverse[skey]
	if !existed {
		return
	}
	l.storage.remove(key, oldValue)
	delete(l.reverse, skey)
}

// stageSet records a pending insert or update against the open
// transaction. See Lookup.stageSet for semantics.
func (l *BytesLookup[E, V]) stageSet(tx Tx, key []byte, entry E) {
	state := tx.txIdentity()
	if state == nil {
		return
	}
	value := l.extract(&entry)
	l.loadOrCreateDelta(state).stageSet(key, value)
}

// stageDelete records a pending deletion against the open transaction.
func (l *BytesLookup[E, V]) stageDelete(tx Tx, key []byte) {
	state := tx.txIdentity()
	if state == nil {
		return
	}
	l.loadOrCreateDelta(state).stageDelete(key)
}

func (l *BytesLookup[E, V]) loadOrCreateDelta(state *txState) *bytesLookupDelta[V] {
	l.deltaMu.Lock()
	defer l.deltaMu.Unlock()
	if l.txDeltas == nil {
		l.txDeltas = make(map[*txState]*bytesLookupDelta[V])
	}
	if d, ok := l.txDeltas[state]; ok {
		return d
	}
	d := newBytesLookupDelta[V]()
	l.txDeltas[state] = d
	state.onCleanup(func() {
		l.deltaMu.Lock()
		delete(l.txDeltas, state)
		l.deltaMu.Unlock()
	})
	return d
}

// resolveTx computes the effective key set for an exact-match query
// under a given transaction. See Lookup.resolveTx for semantics.
func (l *BytesLookup[E, V]) resolveTx(tx Tx, values []V) [][]byte {
	committed := l.Get(values...)
	state := tx.txIdentity()
	if state == nil {
		return committed
	}
	l.deltaMu.Lock()
	d, ok := l.txDeltas[state]
	l.deltaMu.Unlock()
	if !ok || d.isEmpty() {
		return committed
	}
	return d.merge(committed, values)
}

// Get returns the primary keys of entries whose indexed field matches any of
// the provided values. Returned keys are owned by the caller and may be
// freely retained.
func (l *BytesLookup[E, V]) Get(values ...V) [][]byte {
	if len(values) == 0 {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(values) == 1 {
		src := l.storage.get(values[0])
		out := make([][]byte, len(src))
		for i, k := range src {
			out[i] = append([]byte(nil), k...)
		}
		return out
	}
	var out [][]byte
	for _, v := range values {
		for _, k := range l.storage.get(v) {
			out = append(out, append([]byte(nil), k...))
		}
	}
	return out
}

// Filter returns a Filter[[]byte, E] whose Keys are resolved at
// Retrieve.Exec time against the passed transaction. See
// Lookup.Filter for the read-your-own-writes semantics.
func (l *BytesLookup[E, V]) Filter(values ...V) Filter[[]byte, E] {
	captured := append([]V(nil), values...)
	return Filter[[]byte, E]{
		resolve: func(_ context.Context, tx Tx) ([][]byte, func([][]byte) keyMembership[[]byte], error) {
			return l.resolveTx(tx, captured), bytesIndexedKeyMembership, nil
		},
	}
}

// GetTx is the tx-aware counterpart to Get. See Lookup.GetTx for
// semantics. Used by graph-traversal helpers (e.g. ontology's
// parentsByIndex) that need to probe the index for candidate keys
// directly without going through Retrieve.Exec.
func (l *BytesLookup[E, V]) GetTx(tx Tx, values ...V) [][]byte {
	if len(values) == 0 {
		return nil
	}
	return l.resolveTx(tx, values)
}

// bytesLookupStorage is the byte-keyed analogue of mapLookupStorage. The
// forward map is keyed by V (the indexed field) and stores slices of
// []byte primary keys. Removal is identity-by-equality on the byte slice.
type bytesLookupStorage[V comparable] struct {
	forward map[V][][]byte
}

func newBytesLookupStorage[V comparable]() *bytesLookupStorage[V] {
	return &bytesLookupStorage[V]{forward: make(map[V][][]byte)}
}

func (s *bytesLookupStorage[V]) put(key []byte, value V) {
	s.forward[value] = append(s.forward[value], append([]byte(nil), key...))
}

func (s *bytesLookupStorage[V]) remove(key []byte, value V) {
	keys := s.forward[value]
	for i, k := range keys {
		if string(k) == string(key) {
			keys = append(keys[:i], keys[i+1:]...)
			break
		}
	}
	if len(keys) == 0 {
		delete(s.forward, value)
		return
	}
	s.forward[value] = keys
}

func (s *bytesLookupStorage[V]) get(value V) [][]byte {
	return s.forward[value]
}
