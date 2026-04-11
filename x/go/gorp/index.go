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

// Get returns the primary keys of entries whose indexed field matches any of
// the provided values.
func (l *Lookup[K, E, V]) Get(values ...V) []K {
	if len(values) == 0 {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	var out []K
	for _, v := range values {
		out = append(out, l.storage.get(v)...)
	}
	return out
}

// Filter returns a Filter[K, E] that rejects entries whose primary key is not
// in the candidate set computed from values.
func (l *Lookup[K, E, V]) Filter(values ...V) Filter[K, E] {
	return filterFromKeys[K, E](l.Get(values...))
}

// filterFromKeys builds a Filter whose Key stage accepts entries whose primary
// key is in the provided slice. An empty slice produces a filter that rejects
// everything (no matches possible).
func filterFromKeys[K IndexKey, E Entry[K]](keys []K) Filter[K, E] {
	if len(keys) == 0 {
		return Filter[K, E]{
			Key: func(_ K) (bool, error) { return false, nil },
		}
	}
	keySet := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		keySet[k] = struct{}{}
	}
	return Filter[K, E]{
		Key: func(k K) (bool, error) {
			_, ok := keySet[k]
			return ok, nil
		},
	}
}

// Sorted is an ordered in-memory index on a field of type V extracted from
// entries of type E. V is constrained to cmp.Ordered so the storage can use
// the native `<` operator without a caller-supplied comparator. Sorted
// supports exact-match lookups via Filter (same semantics as Lookup) and
// ordered cursor-based pagination via Retrieve.OrderBy.
type Sorted[K IndexKey, E Entry[K], V cmp.Ordered] struct {
	name    string
	extract func(e *E) V

	mu      sync.RWMutex
	storage *sortedStorage[K, V]
	reverse map[K]V
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
	insert := func(entry E) {
		key := entry.GorpKey()
		value := s.extract(&entry)
		s.storage.put(key, value)
		s.reverse[key] = value
	}
	finish := func() { s.mu.Unlock() }
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

// Get returns the primary keys of entries whose indexed field matches any of
// the provided values.
func (s *Sorted[K, E, V]) Get(values ...V) []K {
	if len(values) == 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []K
	for _, v := range values {
		out = append(out, s.storage.get(v)...)
	}
	return out
}

// Filter returns an exact-match Filter[K, E] against the sorted index.
func (s *Sorted[K, E, V]) Filter(values ...V) Filter[K, E] {
	return filterFromKeys[K, E](s.Get(values...))
}
