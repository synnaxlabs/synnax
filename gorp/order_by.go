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
	"iter"
	"math/bits"
	"slices"

	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/gorp/internal/keyset"
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
	// walk yields the keys of the walk in order, each with a check that reports whether
	// an entry read at the key still holds the value the walk found it at. It takes
	// batch keys from the index at a time, or every key at once when batch is 0. When
	// keys is bounded, the walk yields only its keys.
	walk(keys keyset.Set[K], batch int) iter.Seq2[K, func(e *E) bool]
	// sort orders entries by the values they hold now, in the walk direction, and drops
	// each entry that is not strictly past the walk's cursor.
	sort(entries []E) []E
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

// walk does not see uncommitted tx writes. Entries staged via a write tx (stageSet /
// stageDelete) are invisible to ordered iteration; only SortedIndex.Filter and
// FilterRange merge the per-tx delta. A failed populate leaves the index empty for the
// walk.
//
//nolint:unused
func (q SortedQuery[K, E, V]) walk(
	keys keyset.Set[K],
	batch int,
) iter.Seq2[K, func(e *E) bool] {
	desc := q.dir == DirectionDesc
	return func(yield func(K, func(e *E) bool) bool) {
		emit := func(entries []index.Entry[K, V]) bool {
			for _, entry := range entries {
				holds := func(e *E) bool {
					return cmp.Compare(q.sorted.extract(e), entry.Value) == 0
				}
				if !yield(entry.Key, holds) {
					return false
				}
			}
			return true
		}
		var in func(K) bool
		if keys.Bounded() {
			// Sorting few keys costs less than walking past the entries they skip.
			if n := len(keys.Keys()); n*bits.Len(uint(n)) <= q.sorted.store.Len() {
				emit(q.sorted.store.Order(keys.Keys(), desc, q.after))
				return
			}
			in = keys.Probe()
		}
		for after := q.after; ; {
			next := q.sorted.store.Walk(desc, after, batch, in)
			if len(next) == 0 || !emit(next) || batch == 0 {
				return
			}
			after = &next[len(next)-1]
		}
	}
}

//nolint:unused
func (q SortedQuery[K, E, V]) sort(entries []E) []E {
	compareKeys := key.Order[K]()
	compare := func(a, b index.Entry[K, V]) int {
		c := cmp.Compare(a.Value, b.Value)
		if c == 0 {
			c = compareKeys(a.Key, b.Key)
		}
		if q.dir == DirectionDesc {
			return -c
		}
		return c
	}
	at := func(e *E) index.Entry[K, V] {
		return index.Entry[K, V]{Key: (*e).GorpKey(), Value: q.sorted.extract(e)}
	}
	slices.SortFunc(entries, func(a, b E) int { return compare(at(&a), at(&b)) })
	if q.after == nil {
		return entries
	}
	past := slices.IndexFunc(
		entries,
		func(e E) bool { return compare(at(&e), *q.after) > 0 },
	)
	if past == -1 {
		return entries[:0]
	}
	return entries[past:]
}
