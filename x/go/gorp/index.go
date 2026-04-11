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

	"go.uber.org/zap"
)

// Index is a registered secondary index on a Table. Implementations are
// provided by gorp (Lookup, Sorted) and constructed via NewLookup / NewSorted.
// The interface methods are unexported so external code cannot substitute
// custom implementations; callers should use the provided generic types.
type Index[K Key, E Entry[K]] interface {
	// Name returns the human-readable name of the index, used in diagnostics.
	Name() string

	// populateBegin allocates backing storage and transitions the index into
	// the populated state. Returns an error if the index is already populated
	// (double registration).
	populateBegin() error
	// populateInsert inserts a decoded entry during initial population. The
	// entry must be the Table's entry type; each index casts internally.
	populateInsert(entry E)
	// populateFinish marks population as complete.
	populateFinish()

	// applySet is invoked by the Table observer when an entry is created or
	// updated. The index extracts the new indexed value from entry, removes
	// any stale mapping for key, and inserts the new one.
	applySet(key K, entry E)
	// applyDelete is invoked by the Table observer when an entry is deleted.
	// The index uses its reverse map to locate and remove the stale mapping.
	applyDelete(key K)
}

// lookupData holds the populated state of a Lookup index. A nil pointer means
// the index has not been registered on a Table yet.
type lookupData[K IndexKey, V comparable] struct {
	storage lookupStorage[K, V]
	reverse map[K]V
}

// Lookup is an in-memory exact-match index on a field of type V extracted
// from entries of type E. Construct with NewLookup and register on a Table
// via TableConfig.Indexes. Once registered, Get and Filter return matches
// via an O(1) backing lookup; before registration they DPanic and fall back
// to an extract-based scan, preserving correctness.
type Lookup[K IndexKey, E Entry[K], V comparable] struct {
	name    string
	extract func(e *E) V

	mu       sync.RWMutex
	data     *lookupData[K, V]
	warnOnce sync.Once
}

// NewLookup constructs a Lookup index with the given display name and extract
// function. The returned index is unpopulated; register it on a Table through
// TableConfig.Indexes to activate the fast path.
func NewLookup[K IndexKey, E Entry[K], V comparable](
	name string,
	extract func(e *E) V,
) *Lookup[K, E, V] {
	return &Lookup[K, E, V]{name: name, extract: extract}
}

// Name implements Index.
func (l *Lookup[K, E, V]) Name() string { return l.name }

func (l *Lookup[K, E, V]) populateBegin() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.data != nil {
		zap.S().DPanicf("gorp: index %q registered on more than one table", l.name)
		return nil
	}
	l.data = &lookupData[K, V]{
		storage: newLookupStorage[K, V](),
		reverse: make(map[K]V),
	}
	return nil
}

func (l *Lookup[K, E, V]) populateInsert(entry E) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.data == nil {
		return
	}
	key := entry.GorpKey()
	value := l.extract(&entry)
	l.data.storage.put(key, value)
	l.data.reverse[key] = value
}

func (l *Lookup[K, E, V]) populateFinish() {}

func (l *Lookup[K, E, V]) applySet(key K, entry E) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.data == nil {
		return
	}
	newValue := l.extract(&entry)
	if oldValue, existed := l.data.reverse[key]; existed {
		if oldValue == newValue {
			return
		}
		l.data.storage.remove(key, oldValue)
	}
	l.data.storage.put(key, newValue)
	l.data.reverse[key] = newValue
}

func (l *Lookup[K, E, V]) applyDelete(key K) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.data == nil {
		return
	}
	oldValue, existed := l.data.reverse[key]
	if !existed {
		return
	}
	l.data.storage.remove(key, oldValue)
	delete(l.data.reverse, key)
}

// Get returns the primary keys of entries whose indexed field matches any of
// the provided values. Returns nil if the index has not been populated; in
// that case the caller is expected to fall back to a scan.
func (l *Lookup[K, E, V]) Get(values ...V) []K {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.data == nil {
		l.warnOnce.Do(func() {
			zap.S().DPanicf(
				"gorp: Lookup index %q used before registration; falling back to scan",
				l.name,
			)
		})
		return nil
	}
	if len(values) == 0 {
		return nil
	}
	var out []K
	for _, v := range values {
		out = append(out, l.data.storage.get(v)...)
	}
	return out
}

// Filter returns a Filter[K, E] that rejects entries whose primary key is not
// in the candidate set computed from values. If the index is registered the
// candidate set is built via the O(1) forward map; otherwise the filter falls
// back to an extract-based Eval comparison and fires a DPanic once to surface
// the misconfiguration.
func (l *Lookup[K, E, V]) Filter(values ...V) Filter[K, E] {
	l.mu.RLock()
	registered := l.data != nil
	l.mu.RUnlock()
	if !registered {
		l.warnOnce.Do(func() {
			zap.S().DPanicf(
				"gorp: Lookup index %q used before registration; falling back to scan",
				l.name,
			)
		})
		return l.scanFilter(values)
	}
	keys := l.Get(values...)
	return filterFromKeys[K, E](keys)
}

// scanFilter produces an Eval-based Filter that compares each decoded entry's
// extracted value against the candidate set. Used as the fallback when the
// index has not been populated.
func (l *Lookup[K, E, V]) scanFilter(values []V) Filter[K, E] {
	if len(values) == 0 {
		return Filter[K, E]{
			Eval: func(_ Context, _ *E) (bool, error) { return false, nil },
		}
	}
	valueSet := make(map[V]struct{}, len(values))
	for _, v := range values {
		valueSet[v] = struct{}{}
	}
	return Filter[K, E]{
		Eval: func(_ Context, e *E) (bool, error) {
			_, ok := valueSet[l.extract(e)]
			return ok, nil
		},
	}
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

// sortedData holds the populated state of a Sorted index.
type sortedData[K IndexKey, V comparable] struct {
	storage *sortedStorage[K, V]
	reverse map[K]V
}

// Sorted is an ordered in-memory index on a field of type V extracted from
// entries of type E. It supports exact-match lookups via Filter (same
// semantics as Lookup) and ordered cursor-based pagination via Retrieve.OrderBy.
type Sorted[K IndexKey, E Entry[K], V comparable] struct {
	name    string
	extract func(e *E) V
	less    func(a, b V) bool

	mu       sync.RWMutex
	data     *sortedData[K, V]
	warnOnce sync.Once
}

// NewSorted constructs a Sorted index. If less is nil, a native comparison is
// selected based on the kind of V (supports integers, floats, and strings).
// Callers must pass a non-nil less function for other ordered types.
func NewSorted[K IndexKey, E Entry[K], V comparable](
	name string,
	extract func(e *E) V,
	less func(a, b V) bool,
) *Sorted[K, E, V] {
	if less == nil {
		less = defaultLess[V]()
	}
	if less == nil {
		zap.S().DPanicf(
			"gorp: Sorted index %q constructed with nil less and no default ordering for value type",
			name,
		)
	}
	return &Sorted[K, E, V]{name: name, extract: extract, less: less}
}

// Name implements Index.
func (s *Sorted[K, E, V]) Name() string { return s.name }

func (s *Sorted[K, E, V]) populateBegin() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data != nil {
		zap.S().DPanicf("gorp: index %q registered on more than one table", s.name)
		return nil
	}
	s.data = &sortedData[K, V]{
		storage: newSortedStorage[K, V](s.less),
		reverse: make(map[K]V),
	}
	return nil
}

func (s *Sorted[K, E, V]) populateInsert(entry E) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		return
	}
	key := entry.GorpKey()
	value := s.extract(&entry)
	s.data.storage.put(key, value)
	s.data.reverse[key] = value
}

func (s *Sorted[K, E, V]) populateFinish() {}

func (s *Sorted[K, E, V]) applySet(key K, entry E) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		return
	}
	newValue := s.extract(&entry)
	if oldValue, existed := s.data.reverse[key]; existed {
		if oldValue == newValue {
			return
		}
		s.data.storage.remove(key, oldValue)
	}
	s.data.storage.put(key, newValue)
	s.data.reverse[key] = newValue
}

func (s *Sorted[K, E, V]) applyDelete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		return
	}
	oldValue, existed := s.data.reverse[key]
	if !existed {
		return
	}
	s.data.storage.remove(key, oldValue)
	delete(s.data.reverse, key)
}

// Get returns the primary keys of entries whose indexed field matches any of
// the provided values.
func (s *Sorted[K, E, V]) Get(values ...V) []K {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data == nil {
		s.warnOnce.Do(func() {
			zap.S().DPanicf(
				"gorp: Sorted index %q used before registration; falling back to scan",
				s.name,
			)
		})
		return nil
	}
	if len(values) == 0 {
		return nil
	}
	var out []K
	for _, v := range values {
		out = append(out, s.data.storage.get(v)...)
	}
	return out
}

// Filter returns an exact-match Filter[K, E] against the sorted index. Uses
// the same fast-path semantics as Lookup.Filter.
func (s *Sorted[K, E, V]) Filter(values ...V) Filter[K, E] {
	s.mu.RLock()
	registered := s.data != nil
	s.mu.RUnlock()
	if !registered {
		s.warnOnce.Do(func() {
			zap.S().DPanicf(
				"gorp: Sorted index %q used before registration; falling back to scan",
				s.name,
			)
		})
		return s.scanFilter(values)
	}
	return filterFromKeys[K, E](s.Get(values...))
}

func (s *Sorted[K, E, V]) scanFilter(values []V) Filter[K, E] {
	if len(values) == 0 {
		return Filter[K, E]{
			Eval: func(_ Context, _ *E) (bool, error) { return false, nil },
		}
	}
	valueSet := make(map[V]struct{}, len(values))
	for _, v := range values {
		valueSet[v] = struct{}{}
	}
	return Filter[K, E]{
		Eval: func(_ Context, e *E) (bool, error) {
			_, ok := valueSet[s.extract(e)]
			return ok, nil
		},
	}
}
