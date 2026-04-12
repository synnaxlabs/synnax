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
	"cmp"
	"context"
	"sync"
)

// Index is a registered secondary index on a Table. Implementations are
// provided by gorp (Lookup, Sorted) and constructed via NewLookup / NewSorted.
// The interface methods are unexported so external code cannot substitute
// custom implementations; callers should use the provided generic types.
type Index[K Key, E Entry[K]] interface {
	// Name returns the human-readable name of the index, used in diagnostics.
	Name() string
	// populate transitions the index into the populated state and returns
	// (init error, insert closure, finish closure). The caller invokes insert
	// once for every existing entry in the table, then invokes finish exactly
	// once after the last insert. The implementation may hold a write lock
	// across the entire populate phase, so finish is mandatory; failing to
	// call it leaks the lock.
	populate() (error, func(entry E), func())
	// set is invoked by the Table observer when an entry is created or
	// updated. The index extracts the new indexed value from entry, removes
	// any stale mapping for key, and inserts the new one.
	set(key K, entry E)
	// delete is invoked by the Table observer when an entry is deleted. The
	// index uses its reverse map to locate and remove the stale mapping.
	delete(key K)
	// stageSet is invoked by a table-bound Writer after a tx-scoped Set,
	// before the tx commits. It records the pending insert or update in a
	// per-tx delta keyed off tx.txIdentity(). If the tx has no identity (DB
	// acting as Tx), it is a no-op. The committed index is not touched.
	stageSet(tx Tx, key K, entry E)
	// stageDelete is the delete analogue of stageSet. It records a pending
	// deletion in the per-tx delta. The committed index is not touched.
	stageDelete(tx Tx, key K)
}

// Lookup is an in-memory exact-match index on a field of type V extracted
// from entries of type E. Construct with NewLookup and register on a Table
// via TableConfig.Indexes.
type Lookup[K IndexKey, E Entry[K], V comparable] struct {
	name    string
	extract func(e *E) V
	mu      sync.RWMutex
	storage lookupStorage[K, V]
	reverse map[K]V

	// deltaMu guards txDeltas. It is held independently of mu so tx-scoped
	// staging writers never block committed-index readers, and vice versa.
	// Contention on deltaMu is bounded by the number of concurrent write
	// transactions touching this index, which is typically 1.
	deltaMu  sync.Mutex
	txDeltas map[*txState]*lookupDelta[K, V]
}

// NewLookup constructs a Lookup index with the given display name and extract
// function. The returned index is empty; register it on a Table through
// TableConfig.Indexes to populate it from the existing table contents and
// keep it in sync with future writes.
func NewLookup[K IndexKey, E Entry[K], V comparable](
	name string,
	extract func(e *E) V,
) *Lookup[K, E, V] {
	return &Lookup[K, E, V]{
		name:    name,
		extract: extract,
		storage: newLookupStorage[K, V](),
		reverse: make(map[K]V),
	}
}

// Name implements Index.
func (l *Lookup[K, E, V]) Name() string { return l.name }

func (l *Lookup[K, E, V]) populate() (error, func(E), func()) {
	l.mu.Lock()
	insert := func(entry E) {
		key := entry.GorpKey()
		value := l.extract(&entry)
		l.storage.put(key, value)
		l.reverse[key] = value
	}
	finish := func() { l.mu.Unlock() }
	return nil, insert, finish
}

func (l *Lookup[K, E, V]) set(key K, entry E) {
	l.mu.Lock()
	defer l.mu.Unlock()
	newValue := l.extract(&entry)
	if oldValue, existed := l.reverse[key]; existed {
		if oldValue == newValue {
			return
		}
		l.storage.remove(key, oldValue)
	}
	l.storage.put(key, newValue)
	l.reverse[key] = newValue
}

func (l *Lookup[K, E, V]) delete(key K) {
	l.mu.Lock()
	defer l.mu.Unlock()
	oldValue, existed := l.reverse[key]
	if !existed {
		return
	}
	l.storage.remove(key, oldValue)
	delete(l.reverse, key)
}

// stageSet records a pending insert or update against the open
// transaction. Called from the table-bound Writer after a successful
// kv.Tx.Set. The committed index is untouched. If tx has no identity
// (DB acting as Tx, or a Tx implementation without per-tx scoping), the
// call is a no-op — writes go straight to the store and the global
// index is updated post-commit by the DB observer.
func (l *Lookup[K, E, V]) stageSet(tx Tx, key K, entry E) {
	state := tx.txIdentity()
	if state == nil {
		return
	}
	value := l.extract(&entry)
	l.loadOrCreateDelta(state).stageSet(key, value)
}

// stageDelete records a pending deletion against the open transaction.
// Called from the table-bound Writer after a successful kv.Tx.Delete.
func (l *Lookup[K, E, V]) stageDelete(tx Tx, key K) {
	state := tx.txIdentity()
	if state == nil {
		return
	}
	l.loadOrCreateDelta(state).stageDelete(key)
}

// loadOrCreateDelta returns the per-tx delta for this index, creating
// it lazily on first stage. On creation it registers a cleanup hook on
// the tx state that drops the delta when the tx commits or closes.
// This ensures overlays are dropped deterministically, regardless of
// whether the tx committed or rolled back.
func (l *Lookup[K, E, V]) loadOrCreateDelta(state *txState) *lookupDelta[K, V] {
	l.deltaMu.Lock()
	defer l.deltaMu.Unlock()
	if l.txDeltas == nil {
		l.txDeltas = make(map[*txState]*lookupDelta[K, V])
	}
	if d, ok := l.txDeltas[state]; ok {
		return d
	}
	d := newLookupDelta[K, V]()
	l.txDeltas[state] = d
	state.onCleanup(func() {
		l.deltaMu.Lock()
		delete(l.txDeltas, state)
		l.deltaMu.Unlock()
	})
	return d
}

// resolveTx computes the effective key set for an exact-match query
// under a given transaction. If the tx has no per-tx scoping or no
// staged mutations for this index, it returns the committed result
// unchanged. Otherwise it overlays the per-tx delta via delta.merge.
// This is the core read-your-own-writes hook for indexed Retrieve
// queries inside an open write tx.
func (l *Lookup[K, E, V]) resolveTx(tx Tx, values []V) []K {
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
// the provided values.
func (l *Lookup[K, E, V]) Get(values ...V) []K {
	if len(values) == 0 {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(values) == 1 {
		// Single-value fast path: avoid the append-grow loop. Allocate the
		// result slice with the exact size and copy directly. This is the
		// dominant case for index-backed exact-match queries.
		src := l.storage.get(values[0])
		out := make([]K, len(src))
		copy(out, src)
		return out
	}
	var out []K
	for _, v := range values {
		out = append(out, l.storage.get(v)...)
	}
	return out
}

// Filter returns a Filter[K, E] whose Keys are resolved at
// Retrieve.Exec time against the passed transaction. The resolver
// overlays any per-tx delta staged against this index on top of the
// committed index state, so an indexed Retrieve inside the same write
// tx that created / updated / deleted an entry sees those pending
// changes (read-your-own-writes). When the tx has no per-tx scoping
// (DB passed directly) or no staged mutations, the resolver falls
// through to the committed-only path and returns the same keys that
// the pre-overlay Filter would have returned.
//
// The returned Filter carries `resolve` instead of an eager `Keys`
// field. Retrieve.Exec / Exists / Count invoke the resolver before
// dispatch and assign the result to `Keys` + `membership` on the
// local filter copy, after which the rest of the pipeline works
// unchanged (execKeys fast path, composition with And/Or, etc.).
func (l *Lookup[K, E, V]) Filter(values ...V) Filter[K, E] {
	captured := append([]V(nil), values...)
	return Filter[K, E]{
		resolve: func(_ context.Context, tx Tx) ([]K, func([]K) keyMembership[K], error) {
			return l.resolveTx(tx, captured), indexedKeyMembership[K], nil
		},
	}
}

// GetTx is the tx-aware counterpart to Get. It returns the primary
// keys of entries whose indexed field matches any of the provided
// values, merging committed index state with any per-tx delta staged
// against the open transaction. When tx is a DB (no per-tx scoping)
// or has no staged mutations for this index, it returns the same
// result as Get. Use GetTx when consuming keys directly outside of
// Retrieve — e.g. graph traversal helpers that probe the index for
// candidate IDs.
func (l *Lookup[K, E, V]) GetTx(tx Tx, values ...V) []K {
	if len(values) == 0 {
		return nil
	}
	return l.resolveTx(tx, values)
}

// Sorted is an ordered in-memory index on a field of type V extracted from
// entries of type E. V is constrained to cmp.Ordered so the storage can use
// the native `<` operator without a caller-supplied comparator. Sorted
// supports exact-match lookups via Filter (same semantics as Lookup) and
// ordered cursor-based pagination via Retrieve.OrderBy.
//
// Tx delta overlay is v1-scoped to equality Filter. Ordered cursor
// iteration via Retrieve.OrderBy / SortedQuery.walkOrder does NOT
// reflect uncommitted tx writes — it reads the committed sorted slice
// directly.
type Sorted[K IndexKey, E Entry[K], V cmp.Ordered] struct {
	name    string
	extract func(e *E) V

	mu      sync.RWMutex
	storage *sortedStorage[K, V]
	reverse map[K]V

	// deltaMu guards txDeltas, mirroring the Lookup pattern. Kept
	// independent from mu so cursor walks on the committed storage
	// aren't blocked by staging writers on disjoint transactions.
	deltaMu  sync.Mutex
	txDeltas map[*txState]*lookupDelta[K, V]
}

// NewSorted constructs a Sorted index over the provided extract function.
// V must satisfy cmp.Ordered (any built-in ordered primitive: signed and
// unsigned integers, floats, or strings).
func NewSorted[K IndexKey, E Entry[K], V cmp.Ordered](
	name string,
	extract func(e *E) V,
) *Sorted[K, E, V] {
	return &Sorted[K, E, V]{
		name:    name,
		extract: extract,
		storage: newSortedStorage[K, V](),
		reverse: make(map[K]V),
	}
}

// Name implements Index.
func (s *Sorted[K, E, V]) Name() string { return s.name }

func (s *Sorted[K, E, V]) populate() (error, func(E), func()) {
	s.mu.Lock()
	// Bulk-load: append every entry to the storage's tail without maintaining
	// the sort invariant per insert (the per-insert path is O(N) due to slice
	// shifting). Sort once at finish for an O(N log N) populate instead of
	// O(N²). The write lock is held across the whole phase, so concurrent
	// reads can never observe the partially sorted state.
	insert := func(entry E) {
		key := entry.GorpKey()
		value := s.extract(&entry)
		s.storage.entries = append(
			s.storage.entries,
			sortedEntry[K, V]{Value: value, Key: key},
		)
		s.reverse[key] = value
	}
	finish := func() {
		s.storage.sortBulk()
		s.mu.Unlock()
	}
	return nil, insert, finish
}

func (s *Sorted[K, E, V]) set(key K, entry E) {
	s.mu.Lock()
	defer s.mu.Unlock()
	newValue := s.extract(&entry)
	if oldValue, existed := s.reverse[key]; existed {
		if cmp.Compare(oldValue, newValue) == 0 {
			return
		}
		s.storage.remove(key, oldValue)
	}
	s.storage.put(key, newValue)
	s.reverse[key] = newValue
}

func (s *Sorted[K, E, V]) delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldValue, existed := s.reverse[key]
	if !existed {
		return
	}
	s.storage.remove(key, oldValue)
	delete(s.reverse, key)
}

// stageSet records a pending insert or update against the open
// transaction. See Lookup.stageSet for full semantics. Sorted reuses
// lookupDelta because only equality queries consult the delta in v1;
// ordered cursor iteration (Sorted.Ordered / walkOrder) remains
// committed-only.
func (s *Sorted[K, E, V]) stageSet(tx Tx, key K, entry E) {
	state := tx.txIdentity()
	if state == nil {
		return
	}
	value := s.extract(&entry)
	s.loadOrCreateDelta(state).stageSet(key, value)
}

// stageDelete records a pending deletion against the open transaction.
func (s *Sorted[K, E, V]) stageDelete(tx Tx, key K) {
	state := tx.txIdentity()
	if state == nil {
		return
	}
	s.loadOrCreateDelta(state).stageDelete(key)
}

func (s *Sorted[K, E, V]) loadOrCreateDelta(state *txState) *lookupDelta[K, V] {
	s.deltaMu.Lock()
	defer s.deltaMu.Unlock()
	if s.txDeltas == nil {
		s.txDeltas = make(map[*txState]*lookupDelta[K, V])
	}
	if d, ok := s.txDeltas[state]; ok {
		return d
	}
	d := newLookupDelta[K, V]()
	s.txDeltas[state] = d
	state.onCleanup(func() {
		s.deltaMu.Lock()
		delete(s.txDeltas, state)
		s.deltaMu.Unlock()
	})
	return d
}

// resolveTx computes the effective key set for an exact-match query
// under a given transaction. See Lookup.resolveTx for semantics. Only
// Sorted.Filter (exact-match) routes through this path; ordered cursor
// iteration via OrderBy does not consult the delta in v1.
func (s *Sorted[K, E, V]) resolveTx(tx Tx, values []V) []K {
	committed := s.Get(values...)
	state := tx.txIdentity()
	if state == nil {
		return committed
	}
	s.deltaMu.Lock()
	d, ok := s.txDeltas[state]
	s.deltaMu.Unlock()
	if !ok || d.isEmpty() {
		return committed
	}
	return d.merge(committed, values)
}

// Get returns the primary keys of entries whose indexed field matches any of
// the provided values.
func (s *Sorted[K, E, V]) Get(values ...V) []K {
	if len(values) == 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(values) == 1 {
		src := s.storage.get(values[0])
		out := make([]K, len(src))
		copy(out, src)
		return out
	}
	var out []K
	for _, v := range values {
		out = append(out, s.storage.get(v)...)
	}
	return out
}

// Filter returns an exact-match Filter[K, E] against the sorted index.
// Like Lookup.Filter, the returned filter carries a deferred resolver
// that merges committed index state with any per-tx delta at
// Retrieve.Exec time. Ordered cursor iteration (Sorted.Ordered / OrderBy)
// is not covered by the delta overlay in v1; only equality Filter.
func (s *Sorted[K, E, V]) Filter(values ...V) Filter[K, E] {
	captured := append([]V(nil), values...)
	return Filter[K, E]{
		resolve: func(_ context.Context, tx Tx) ([]K, func([]K) keyMembership[K], error) {
			return s.resolveTx(tx, captured), indexedKeyMembership[K], nil
		},
	}
}

// GetTx is the tx-aware counterpart to Get. See Lookup.GetTx for
// semantics.
func (s *Sorted[K, E, V]) GetTx(tx Tx, values ...V) []K {
	if len(values) == 0 {
		return nil
	}
	return s.resolveTx(tx, values)
}
