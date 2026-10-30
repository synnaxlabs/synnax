// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"cmp"
	"context"
	"slices"
	"sort"
	"sync"

	"github.com/synnaxlabs/x/set"
)

type entry[K comparable, V cmp.Ordered] struct {
	key   K
	value V
}

// Sorted is an ordered index from values of type V to the primary keys of type K that
// hold them. It answers exact-match lookups like Lookup and also walks its keys in
// value order. Sorted is safe for concurrent use.
type Sorted[S Scope, K comparable, V cmp.Ordered] struct {
	// mu guards entries, reverse, and population.err. Populate holds it for writing
	// until the load finishes.
	mu         sync.RWMutex
	population population
	// entries holds the committed mappings in ascending value order. An insert costs a
	// binary search and a slice shift.
	entries []entry[K, V]
	reverse map[K]V
	overlay overlay[S, *delta[K, V]]
	// published holds the deltas of committing scopes. Every read sees the entries
	// they add.
	published set.Set[*delta[K, V]]
}

// NewSorted returns an empty Sorted index that waits for its bulk load.
func NewSorted[S Scope, K comparable, V cmp.Ordered]() *Sorted[S, K, V] {
	s := &Sorted[S, K, V]{
		population: newPopulation(),
		reverse:    make(map[K]V),
		published:  set.New[*delta[K, V]](),
	}
	s.overlay.create = newDelta[K, V]
	s.overlay.publish, s.overlay.flush, s.overlay.discard = s.publish, s.flush, s.discard
	return s
}

// Populate starts the bulk load and locks the index until it finishes. insert adds a
// committed mapping. finish ends the load and must be called exactly once; a non-nil
// err makes the index invalid, and every later read returns err.
func (s *Sorted[S, K, V]) Populate() (insert func(K, V), finish func(err error)) {
	s.mu.Lock()
	insert = func(key K, value V) {
		s.entries = append(s.entries, entry[K, V]{key: key, value: value})
		s.reverse[key] = value
	}
	return insert, func(err error) {
		if err == nil {
			slices.SortFunc(s.entries, func(a, b entry[K, V]) int {
				return cmp.Compare(a.value, b.value)
			})
		}
		s.population.finish(err)
		s.mu.Unlock()
	}
}

// Wait blocks until the bulk load finishes or ctx is done. It returns the load's error
// or ctx.Err().
func (s *Sorted[S, K, V]) Wait(ctx context.Context) error {
	return s.population.wait(ctx)
}

// Set maps key to value in committed state, replacing any prior value of key.
func (s *Sorted[S, K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.put(key, value)
}

// Delete removes key from committed state.
func (s *Sorted[S, K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remove(key)
}

// Stage maps key to value in the changes of scope. The change applies to committed
// state when scope commits.
func (s *Sorted[S, K, V]) Stage(scope S, key K, value V) {
	if isNone(scope) {
		s.Set(key, value)
		return
	}
	s.overlay.load(scope).set(key, value)
}

// StageDelete removes key in the changes of scope. The removal applies to committed
// state when scope commits.
func (s *Sorted[S, K, V]) StageDelete(scope S, key K) {
	if isNone(scope) {
		s.Delete(key)
		return
	}
	s.overlay.load(scope).delete(key)
}

// Get returns the keys mapped to any of values, with the changes of scope applied. It
// blocks until the bulk load finishes. The caller owns the returned slice, which is
// never nil.
func (s *Sorted[S, K, V]) Get(scope S, values ...V) ([]K, error) {
	if len(values) == 0 {
		return []K{}, nil
	}
	<-s.population.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.population.err; err != nil {
		return nil, err
	}
	var committed []K
	for _, v := range values {
		for i, hi := s.lowerBound(v), s.upperBound(v); i < hi; i++ {
			committed = append(committed, s.entries[i].key)
		}
	}
	committed = withPublished(committed, s.published, values)
	return mergeValues(&s.overlay, scope, committed, values), nil
}

// Walk returns up to limit committed keys in ascending value order, or in descending
// order when desc is true. A non-nil after starts the walk strictly past *after. A
// limit of 0 means no limit. Walk does not see staged changes, blocks until the bulk
// load finishes, and returns nil if the load failed.
func (s *Sorted[S, K, V]) Walk(desc bool, after *V, limit int) []K {
	if limit < 0 {
		return nil
	}
	<-s.population.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.population.err != nil || len(s.entries) == 0 {
		return nil
	}
	keys := make([]K, 0, limit)
	full := func() bool { return limit > 0 && len(keys) >= limit }
	if desc {
		start := len(s.entries) - 1
		if after != nil {
			start = s.lowerBound(*after) - 1
		}
		for i := start; i >= 0 && !full(); i-- {
			keys = append(keys, s.entries[i].key)
		}
		return keys
	}
	var start int
	if after != nil {
		start = s.upperBound(*after)
	}
	for i := start; i < len(s.entries) && !full(); i++ {
		keys = append(keys, s.entries[i].key)
	}
	return keys
}

// lowerBound returns the first index whose value is at least value.
func (s *Sorted[S, K, V]) lowerBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].value >= value
	})
}

// upperBound returns the first index whose value is above value.
func (s *Sorted[S, K, V]) upperBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].value > value
	})
}

func (s *Sorted[S, K, V]) put(key K, value V) {
	if prev, ok := s.reverse[key]; ok {
		if cmp.Compare(prev, value) == 0 {
			return
		}
		s.removeEntry(key, prev)
	}
	s.entries = slices.Insert(
		s.entries,
		s.upperBound(value),
		entry[K, V]{key: key, value: value},
	)
	s.reverse[key] = value
}

func (s *Sorted[S, K, V]) remove(key K) {
	prev, ok := s.reverse[key]
	if !ok {
		return
	}
	s.removeEntry(key, prev)
	delete(s.reverse, key)
}

func (s *Sorted[S, K, V]) removeEntry(key K, value V) {
	for i, hi := s.lowerBound(value), s.upperBound(value); i < hi; i++ {
		if s.entries[i].key == key {
			s.entries = slices.Delete(s.entries, i, i+1)
			return
		}
	}
}

func (s *Sorted[S, K, V]) publish(d *delta[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.published.Add(d)
}

func (s *Sorted[S, K, V]) discard(d *delta[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.published.Remove(d)
}

func (s *Sorted[S, K, V]) flush(d *delta[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.published.Remove(d)
	for k, st := range d.state {
		if st.deleted {
			s.remove(k)
			continue
		}
		s.put(k, st.value)
	}
}
