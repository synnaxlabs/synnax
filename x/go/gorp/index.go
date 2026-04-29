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
	// populate returns an insert closure and a finish closure for the
	// populate phase. The caller must invoke insert once for every existing
	// entry in the table and then invoke finish exactly once. finish is
	// mandatory: implementations may hold a write lock across the populate
	// phase, and skipping finish leaks it.
	populate() (func(entry E), func(), error)
	// set records that the entry for key is now entry, replacing any prior
	// mapping for the same key in committed index state.
	set(key K, entry E)
	// delete removes any committed mapping for key.
	delete(key K)
	// stageSet records a pending insert or update of entry under key
	// against tx's per-tx delta. Committed index state is not modified
	// until tx commits. When tx has no per-tx identity (a DB used
	// directly), the mutation applies to committed state immediately.
	stageSet(tx Tx, key K, entry E)
	// stageDelete records a pending deletion of key against tx's per-tx
	// delta. Committed index state is not modified until tx commits.
	// When tx has no per-tx identity, the deletion applies to committed
	// state immediately.
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
	overlay deltaOverlay[K, V]
}

// NewLookup constructs a Lookup index with the given display name and extract
// function. The returned index is empty; register it on a Table through
// TableConfig.Indexes to populate it from the existing table contents and
// keep it in sync with future writes.
func NewLookup[K IndexKey, E Entry[K], V comparable](
	name string,
	extract func(e *E) V,
) *Lookup[K, E, V] {
	l := &Lookup[K, E, V]{
		name:    name,
		extract: extract,
		storage: newLookupStorage[K, V](),
		reverse: make(map[K]V),
	}
	l.overlay.flush = l.flushTx
	return l
}

// Name implements Index.
func (l *Lookup[K, E, V]) Name() string { return l.name }

//nolint:unused
func (l *Lookup[K, E, V]) populate() (func(E), func(), error) {
	l.mu.Lock()
	insert := func(entry E) {
		key := entry.GorpKey()
		value := l.extract(&entry)
		l.storage.put(key, value)
		l.reverse[key] = value
	}
	finish := func() { l.mu.Unlock() }
	return insert, finish, nil
}

//nolint:unused
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

//nolint:unused
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

//nolint:unused
func (l *Lookup[K, E, V]) stageSet(tx Tx, key K, entry E) {
	if tx.txIdentity() == nil {
		l.set(key, entry)
		return
	}
	l.overlay.stage(tx, key, l.extract(&entry))
}

//nolint:unused
func (l *Lookup[K, E, V]) stageDelete(tx Tx, key K) {
	if tx.txIdentity() == nil {
		l.delete(key)
		return
	}
	l.overlay.unstage(tx, key)
}

// flushTx promotes the staged tx delta into committed index state.
func (l *Lookup[K, E, V]) flushTx(d *delta[K, V]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, entry := range d.state {
		oldValue, existed := l.reverse[k]
		if entry.deleted {
			if existed {
				l.storage.remove(k, oldValue)
				delete(l.reverse, k)
			}
			continue
		}
		if existed {
			if oldValue == entry.value {
				continue
			}
			l.storage.remove(k, oldValue)
		}
		l.storage.put(k, entry.value)
		l.reverse[k] = entry.value
	}
}

func (l *Lookup[K, E, V]) resolveTx(tx Tx, values []V) []K {
	return l.overlay.resolve(tx, l.Get(values...), values)
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

// Filter returns a Filter[K, E] matching entries whose indexed field is
// any of values. The filter sees read-your-own-writes: an indexed
// Retrieve inside a write tx that created, updated, or deleted an
// entry observes those pending changes alongside committed index
// state. A Retrieve against a DB used directly returns committed
// state only.
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
// entries of type E. V is constrained to cmp.Ordered so the storage can
// compare values without a caller-supplied comparator. Sorted supports
// exact-match lookups via Filter (same semantics as Lookup) and ordered
// cursor-based pagination via Retrieve.OrderBy.
//
// Read-your-own-writes is v1-scoped to equality Filter. Ordered cursor
// iteration via Retrieve.OrderBy does NOT reflect uncommitted tx
// writes; an open write tx that staged inserts or deletes will not
// see those changes during ordered iteration.
type Sorted[K IndexKey, E Entry[K], V cmp.Ordered] struct {
	name    string
	extract func(e *E) V
	mu      sync.RWMutex
	storage *sortedStorage[K, V]
	reverse map[K]V
	overlay deltaOverlay[K, V]
}

// NewSorted constructs a Sorted index over the provided extract function.
// V must satisfy cmp.Ordered (any built-in ordered primitive: signed and
// unsigned integers, floats, or strings).
func NewSorted[K IndexKey, E Entry[K], V cmp.Ordered](
	name string,
	extract func(e *E) V,
) *Sorted[K, E, V] {
	s := &Sorted[K, E, V]{
		name:    name,
		extract: extract,
		storage: newSortedStorage[K, V](),
		reverse: make(map[K]V),
	}
	s.overlay.flush = s.flushTx
	return s
}

// Name implements Index.
func (s *Sorted[K, E, V]) Name() string { return s.name }

//nolint:unused
func (s *Sorted[K, E, V]) populate() (func(E), func(), error) {
	s.mu.Lock()
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
	return insert, finish, nil
}

//nolint:unused
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

//nolint:unused
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

//nolint:unused
func (s *Sorted[K, E, V]) stageSet(tx Tx, key K, entry E) {
	if tx.txIdentity() == nil {
		s.set(key, entry)
		return
	}
	s.overlay.stage(tx, key, s.extract(&entry))
}

//nolint:unused
func (s *Sorted[K, E, V]) stageDelete(tx Tx, key K) {
	if tx.txIdentity() == nil {
		s.delete(key)
		return
	}
	s.overlay.unstage(tx, key)
}

// flushTx promotes the staged tx delta into committed index state.
func (s *Sorted[K, E, V]) flushTx(d *delta[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, entry := range d.state {
		oldValue, existed := s.reverse[k]
		if entry.deleted {
			if existed {
				s.storage.remove(k, oldValue)
				delete(s.reverse, k)
			}
			continue
		}
		if existed {
			if cmp.Compare(oldValue, entry.value) == 0 {
				continue
			}
			s.storage.remove(k, oldValue)
		}
		s.storage.put(k, entry.value)
		s.reverse[k] = entry.value
	}
}

func (s *Sorted[K, E, V]) resolveTx(tx Tx, values []V) []K {
	return s.overlay.resolve(tx, s.Get(values...), values)
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

// Filter returns an exact-match Filter[K, E] matching entries whose
// indexed field is any of values. Read-your-own-writes semantics
// match Lookup.Filter. Ordered cursor iteration (Sorted.Ordered /
// OrderBy) is not covered; only equality Filter.
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
