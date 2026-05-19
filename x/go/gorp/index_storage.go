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
	"slices"
	"sort"
)

// sortedEntry is a single (value, key) pair inside a SortedIndex index slice.
type sortedEntry[K Key, V cmp.Ordered] struct {
	// key is the entry's primary key.
	key K
	// value is the indexed field value that orders this entry.
	value V
}

// sortedStorage backs a SortedIndex index. It keeps entries in ascending order
// of V using the native `<` operator (V is constrained to cmp.Ordered).
// Insertion is O(log n) binary search plus O(n) slice shift. At the
// target scale (<100k entries) this is acceptable; if profiling shows
// the shift cost matters, swap the backing for a B-tree without changing
// the outer API. Within equal values, entries are kept in insertion
// order; removal scans that sub-range for an exact key match.
type sortedStorage[K Key, V cmp.Ordered] struct {
	// entries is the index contents in ascending V order. Within equal
	// values, entries are kept in insertion order.
	entries []sortedEntry[K, V]
}

func newSortedStorage[K Key, V cmp.Ordered]() *sortedStorage[K, V] {
	return &sortedStorage[K, V]{}
}

// lowerBound returns the first index i such that entries[i].Value >= value.
func (s *sortedStorage[K, V]) lowerBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].value >= value
	})
}

// upperBound returns the first index i such that entries[i].Value > value.
func (s *sortedStorage[K, V]) upperBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].value > value
	})
}

func (s *sortedStorage[K, V]) put(key K, value V) {
	i := s.upperBound(value)
	s.entries = slices.Insert(s.entries, i, sortedEntry[K, V]{value: value, key: key})
}

func (s *sortedStorage[K, V]) remove(key K, value V) {
	lo := s.lowerBound(value)
	hi := s.upperBound(value)
	for i := lo; i < hi; i++ {
		if s.entries[i].key == key {
			s.entries = slices.Delete(s.entries, i, i+1)
			return
		}
	}
}

func (s *sortedStorage[K, V]) get(value V) []K {
	lo := s.lowerBound(value)
	hi := s.upperBound(value)
	if lo == hi {
		return nil
	}
	out := make([]K, hi-lo)
	for i := lo; i < hi; i++ {
		out[i-lo] = s.entries[i].key
	}
	return out
}

// sortBulk sorts the entries slice by Value. Used by SortedIndex.populate to
// finalize a bulk-loaded index in O(N log N) instead of inserting one
// entry at a time at O(N²).
//
//nolint:unused
func (s *sortedStorage[K, V]) sortBulk() {
	slices.SortFunc(s.entries, func(a, b sortedEntry[K, V]) int {
		return cmp.Compare(a.value, b.value)
	})
}
