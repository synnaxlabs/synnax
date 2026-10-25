// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import "cmp"

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
	walkOrder(limit int) []K
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
	after *V
}

// Ordered constructs a SortedQuery for an ordered walk of the index in the
// given direction. The returned handle is passed to Retrieve.OrderBy.
func (s *SortedIndex[K, E, V]) Ordered(dir Direction) SortedQuery[K, E, V] {
	return SortedQuery[K, E, V]{sorted: s, dir: dir}
}

// After sets the resume cursor for cursor-based pagination. The walk skips
// every entry whose value is less than or equal to cursor (Asc) or greater
// than or equal to cursor (Desc), so the next page begins strictly past the
// previous page's last visited value. Calling After replaces any prior
// cursor on this query.
func (q SortedQuery[K, E, V]) After(cursor V) SortedQuery[K, E, V] {
	q.after = &cursor
	return q
}

// walkOrder returns up to limit keys in the configured direction,
// starting after the resume cursor if one was set. A limit of 0 means
// unbounded.
//
// walkOrder does not see uncommitted tx writes. Entries staged via a
// write tx (stageSet / stageDelete) are invisible to ordered iteration;
// only SortedIndex.Filter merges the per-tx delta. A Where filter combined
// with OrderBy is applied as a post-filter against the committed
// entries, so the same caveat extends to the combined query. A failed
// populate leaves the index empty for the walk.
//
//nolint:unused
func (q SortedQuery[K, E, V]) walkOrder(limit int) []K {
	return q.sorted.store.Walk(q.dir == DirectionDesc, q.after, limit)
}
