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

// Entry is a position in a Sorted index: a value and the key that holds it.
type Entry[K comparable, V cmp.Ordered] struct {
	Value V
	Key   K
}

// Bound is one end of a range of values. A nil *Bound leaves its end of the range open.
type Bound[V cmp.Ordered] struct {
	Value V
	// Exclusive leaves Value out of the range.
	Exclusive bool
}

// InRange reports whether value lies between lower and upper.
func InRange[V cmp.Ordered](value V, lower, upper *Bound[V]) bool {
	if lower != nil {
		if c := cmp.Compare(value, lower.Value); c < 0 || (c == 0 && lower.Exclusive) {
			return false
		}
	}
	if upper != nil {
		if c := cmp.Compare(value, upper.Value); c > 0 || (c == 0 && upper.Exclusive) {
			return false
		}
	}
	return true
}

// Sorted is an ordered index from values of type V to the primary keys of type K that
// hold them. It answers exact-match lookups like Lookup and also walks its entries in
// value order, then key order. Sorted is safe for concurrent use.
type Sorted[S Scope, K comparable, V cmp.Ordered] struct {
	// mu guards entries, reverse, and population.err. Populate holds it for writing
	// until the load finishes.
	mu         sync.RWMutex
	population population
	// entries holds the committed mappings in ascending order. An insert costs a binary
	// search and a slice shift.
	entries     []Entry[K, V]
	reverse     map[K]V
	compareKeys func(a, b K) int
	overlay     overlay[S, *delta[K, V]]
}

// NewSorted returns an empty Sorted index that waits for its bulk load. compareKeys
// orders the keys of entries with equal values.
func NewSorted[S Scope, K comparable, V cmp.Ordered](
	compareKeys func(a, b K) int,
) *Sorted[S, K, V] {
	s := &Sorted[S, K, V]{
		population:  newPopulation(),
		reverse:     make(map[K]V),
		compareKeys: compareKeys,
	}
	s.overlay.create, s.overlay.flush = newDelta[K, V], s.flush
	return s
}

func (s *Sorted[S, K, V]) compare(a, b Entry[K, V]) int {
	if c := cmp.Compare(a.Value, b.Value); c != 0 {
		return c
	}
	return s.compareKeys(a.Key, b.Key)
}

// Populate starts the bulk load and locks the index until it finishes. insert adds a
// committed mapping. finish ends the load and must be called exactly once; a non-nil
// err makes the index invalid, and every later read returns err.
func (s *Sorted[S, K, V]) Populate() (insert func(K, V), finish func(err error)) {
	s.mu.Lock()
	insert = func(key K, value V) {
		s.entries = append(s.entries, Entry[K, V]{Key: key, Value: value})
		s.reverse[key] = value
	}
	return insert, func(err error) {
		if err == nil {
			slices.SortFunc(s.entries, s.compare)
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
	s.overlay.load(scope).set(key, []V{value})
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
	committed := set.New[K]()
	for _, v := range values {
		for i, hi := s.lowerBound(v), s.upperBound(v); i < hi; i++ {
			committed.Add(s.entries[i].Key)
		}
	}
	return mergeValues(&s.overlay, scope, committed, values), nil
}

// Committing returns the keys that a transaction in its commit maps to any of values.
// See Lookup.Committing for its contract.
func (s *Sorted[S, K, V]) Committing(values ...V) []K {
	return committingKeys(&s.overlay, values)
}

// Range returns the keys whose values lie between lower and upper, with the changes of
// scope applied. It blocks until the bulk load finishes. The caller owns the returned
// slice, which is never nil.
func (s *Sorted[S, K, V]) Range(scope S, lower, upper *Bound[V]) ([]K, error) {
	<-s.population.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.population.err; err != nil {
		return nil, err
	}
	lo, hi := s.span(lower, upper)
	keys := make([]K, hi-lo)
	for i, e := range s.entries[lo:hi] {
		keys[i] = e.Key
	}
	d, ok := s.overlay.get(scope)
	if !ok {
		return keys, nil
	}
	keys = slices.DeleteFunc(keys, func(k K) bool {
		_, staged := d.state[k]
		return staged
	})
	return appendInRange(keys, d, lower, upper), nil
}

// CommittingRange returns the keys that a transaction in its commit maps to a value
// between lower and upper. See Lookup.Committing for its contract.
func (s *Sorted[S, K, V]) CommittingRange(lower, upper *Bound[V]) []K {
	s.overlay.mu.RLock()
	defer s.overlay.mu.RUnlock()
	var keys []K
	for scope, d := range s.overlay.deltas {
		if scope.Committing() {
			keys = appendInRange(keys, d, lower, upper)
		}
	}
	return keys
}

// Count returns the number of committed entries whose values lie between lower and
// upper, in O(log n). It blocks until the bulk load finishes.
func (s *Sorted[S, K, V]) Count(lower, upper *Bound[V]) int {
	<-s.population.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	lo, hi := s.span(lower, upper)
	return hi - lo
}

// Keep returns each key of keys that Range or CommittingRange would return, in
// O(len(keys)). It blocks until the bulk load finishes. The caller owns the returned
// slice.
func (s *Sorted[S, K, V]) Keep(scope S, keys []K, lower, upper *Bound[V]) []K {
	committing := set.New(s.CommittingRange(lower, upper)...)
	<-s.population.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	var state map[K]staged[V]
	if d, ok := s.overlay.get(scope); ok {
		state = d.state
	}
	kept := make([]K, 0, len(keys))
	for _, k := range keys {
		v, ok := s.reverse[k]
		if st, in := state[k]; in {
			ok = !st.deleted
			if ok {
				v = st.values[0]
			}
		}
		if (ok && InRange(v, lower, upper)) || committing.Contains(k) {
			kept = append(kept, k)
		}
	}
	return kept
}

// Len returns the number of committed entries. It blocks until the bulk load finishes.
func (s *Sorted[S, K, V]) Len() int {
	<-s.population.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// appendInRange appends to keys each key that d maps to a value between lower and
// upper.
func appendInRange[K comparable, V cmp.Ordered](
	keys []K,
	d *delta[K, V],
	lower, upper *Bound[V],
) []K {
	for k, st := range d.state {
		if !st.deleted && InRange(st.values[0], lower, upper) {
			keys = append(keys, k)
		}
	}
	return keys
}

// Walk returns up to limit committed entries in ascending order, or in descending order
// when desc is true. A non-nil after starts the walk strictly past *after, which need
// not be an entry of the index. A non-nil in skips each entry whose key it rejects, and
// the skipped entries do not count toward limit. A limit of 0 means no limit. Walk does
// not see staged changes, blocks until the bulk load finishes, and returns nil if the
// load failed.
func (s *Sorted[S, K, V]) Walk(
	desc bool,
	after *Entry[K, V],
	limit int,
	in func(K) bool,
) []Entry[K, V] {
	if limit < 0 {
		return nil
	}
	<-s.population.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.population.err != nil || len(s.entries) == 0 {
		return nil
	}
	n := len(s.entries)
	if limit > 0 {
		n = min(n, limit)
	}
	walked := make([]Entry[K, V], 0, n)
	full := func() bool { return limit > 0 && len(walked) >= limit }
	visit := func(e Entry[K, V]) {
		if in == nil || in(e.Key) {
			walked = append(walked, e)
		}
	}
	if desc {
		start := len(s.entries) - 1
		if after != nil {
			start = s.position(*after) - 1
		}
		for i := start; i >= 0 && !full(); i-- {
			visit(s.entries[i])
		}
		return walked
	}
	var start int
	if after != nil {
		start = s.position(*after)
		if start < len(s.entries) && s.compare(s.entries[start], *after) == 0 {
			start++
		}
	}
	for i := start; i < len(s.entries) && !full(); i++ {
		visit(s.entries[i])
	}
	return walked
}

// Order returns the committed entries of keys in the order Walk visits them, strictly
// past after when after is non-nil. It drops the keys the index does not hold. Order
// does not see staged changes, blocks until the bulk load finishes, and returns nil if
// the load failed.
func (s *Sorted[S, K, V]) Order(keys []K, desc bool, after *Entry[K, V]) []Entry[K, V] {
	<-s.population.done
	s.mu.RLock()
	if s.population.err != nil {
		s.mu.RUnlock()
		return nil
	}
	ordered := make([]Entry[K, V], 0, len(keys))
	for _, k := range keys {
		if v, ok := s.reverse[k]; ok {
			ordered = append(ordered, Entry[K, V]{Key: k, Value: v})
		}
	}
	s.mu.RUnlock()
	compare := s.compare
	if desc {
		compare = func(a, b Entry[K, V]) int { return s.compare(b, a) }
	}
	slices.SortFunc(ordered, compare)
	if after == nil {
		return ordered
	}
	start, found := slices.BinarySearchFunc(ordered, *after, compare)
	if found {
		start++
	}
	return ordered[start:]
}

// lowerBound returns the first index whose value is at least value.
func (s *Sorted[S, K, V]) lowerBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].Value >= value
	})
}

// upperBound returns the first index whose value is above value.
func (s *Sorted[S, K, V]) upperBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].Value > value
	})
}

// span returns the positions [lo, hi) of the committed entries whose values lie between
// lower and upper.
func (s *Sorted[S, K, V]) span(lower, upper *Bound[V]) (lo, hi int) {
	lo, hi = 0, len(s.entries)
	if lower != nil {
		if lower.Exclusive {
			lo = s.upperBound(lower.Value)
		} else {
			lo = s.lowerBound(lower.Value)
		}
	}
	if upper != nil {
		if upper.Exclusive {
			hi = s.lowerBound(upper.Value)
		} else {
			hi = s.upperBound(upper.Value)
		}
	}
	return lo, max(lo, hi)
}

// position returns the first index whose entry orders at or after e.
func (s *Sorted[S, K, V]) position(e Entry[K, V]) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.compare(s.entries[i], e) >= 0
	})
}

func (s *Sorted[S, K, V]) put(key K, value V) {
	if prev, ok := s.reverse[key]; ok {
		if cmp.Compare(prev, value) == 0 {
			return
		}
		s.removeEntry(key, prev)
	}
	e := Entry[K, V]{Key: key, Value: value}
	s.entries = slices.Insert(s.entries, s.position(e), e)
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
	e := Entry[K, V]{Key: key, Value: value}
	if i := s.position(e); i < len(s.entries) && s.entries[i].Key == key {
		s.entries = slices.Delete(s.entries, i, i+1)
	}
}

func (s *Sorted[S, K, V]) flush(d *delta[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, st := range d.state {
		if st.deleted {
			s.remove(k)
			continue
		}
		s.put(k, st.values[0])
	}
}
