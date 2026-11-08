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

	"github.com/synnaxlabs/gorp/internal/index"
)

// Direction is the iteration direction used by a SortedIndex index when it is
// consumed via Retrieve.OrderBy.
type Direction uint8

const (
	// DirectionAsc walks the sorted index from smallest to largest value.
	DirectionAsc Direction = iota
	// DirectionDesc walks the sorted index from largest to smallest value.
	DirectionDesc
)

// OrderQuery is an opaque ordering handle produced by SortedIndex.Ordered and
// consumed by Retrieve.OrderBy.
type OrderQuery[K Key, E Entry[K]] interface {
	// walker returns a function that yields the next n keys of the walk on each call,
	// and no keys once the walk ends. An n of 0 yields every remaining key. When keys
	// is non-nil, the walk visits only those keys.
	walker(keys []K) func(n int) []K
}

// SortedQuery is the handle returned by SortedIndex.Ordered. Pass it to
// Retrieve.OrderBy to drive an ordered walk; chain After to set a
// resume cursor for pagination.
type SortedQuery[K Key, E Entry[K], V cmp.Ordered] struct {
	// sorted is the index being walked.
	sorted *SortedIndex[K, E, V]
	// dir is the walk direction.
	dir Direction
	// after is the resume cursor for paginated walks. Nil starts the walk at the first
	// entry.
	after *index.Entry[K, V]
}

// Ordered constructs a SortedQuery for an ordered walk of the index in the
// given direction. The returned handle is passed to Retrieve.OrderBy.
func (s *SortedIndex[K, E, V]) Ordered(dir Direction) SortedQuery[K, E, V] {
	return SortedQuery[K, E, V]{sorted: s, dir: dir}
}

// After sets the resume cursor for cursor-based pagination: the indexed value and the
// key of the last entry of the previous page. The walk starts strictly past the cursor
// in the walk direction, so no entry that shares the value is lost. The cursor need not
// name an entry that still exists. Calling After replaces any prior cursor.
func (q SortedQuery[K, E, V]) After(value V, key K) SortedQuery[K, E, V] {
	q.after = &index.Entry[K, V]{Value: value, Key: key}
	return q
}

// walker does not see uncommitted tx writes. Entries staged via a write tx (stageSet /
// stageDelete) are invisible to ordered iteration; only SortedIndex.Filter merges the
// per-tx delta. A failed populate leaves the index empty for the walk.
//
//nolint:unused
func (q SortedQuery[K, E, V]) walker(keys []K) func(n int) []K {
	desc := q.dir == DirectionDesc
	if keys != nil {
		ordered := q.sorted.store.Order(keys, desc, q.after)
		return func(n int) []K {
			if n == 0 || n > len(ordered) {
				n = len(ordered)
			}
			next := ordered[:n]
			ordered = ordered[n:]
			return entryKeys(next)
		}
	}
	after := q.after
	return func(n int) []K {
		next := q.sorted.store.Walk(desc, after, n)
		if len(next) > 0 {
			after = &next[len(next)-1]
		}
		return entryKeys(next)
	}
}

func entryKeys[K Key, V cmp.Ordered](entries []index.Entry[K, V]) []K {
	keys := make([]K, len(entries))
	for i, e := range entries {
		keys[i] = e.Key
	}
	return keys
}
