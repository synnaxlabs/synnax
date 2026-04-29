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

// Direction is the iteration direction used by a Sorted index when it is
// consumed via Retrieve.OrderBy.
type Direction uint8

const (
	// Asc walks the sorted index from smallest to largest value.
	Asc Direction = iota
	// Desc walks the sorted index from largest to smallest value.
	Desc
)

// OrderQuery is an opaque ordering handle produced by Sorted.Ordered and
// consumed by Retrieve.OrderBy.
type OrderQuery[K Key, E Entry[K]] interface {
	walkOrder(limit int) []K
}

// SortedQuery is the handle returned by Sorted.Ordered. Pass it to
// Retrieve.OrderBy to drive an ordered walk; chain After to set a
// resume cursor for pagination.
type SortedQuery[K IndexKey, E Entry[K], V cmp.Ordered] struct {
	sorted    *Sorted[K, E, V]
	dir       Direction
	cursor    V
	hasCursor bool
}

// Ordered constructs a SortedQuery for an ordered walk of the index in the
// given direction. The returned handle is passed to Retrieve.OrderBy.
func (s *Sorted[K, E, V]) Ordered(dir Direction) SortedQuery[K, E, V] {
	return SortedQuery[K, E, V]{sorted: s, dir: dir}
}

// After sets the resume cursor for cursor-based pagination. The walk skips
// every entry whose value is less than or equal to cursor (Asc) or greater
// than or equal to cursor (Desc), so the next page begins strictly past the
// previous page's last visited value. Calling After replaces any prior
// cursor on this query.
func (q SortedQuery[K, E, V]) After(cursor V) SortedQuery[K, E, V] {
	q.cursor = cursor
	q.hasCursor = true
	return q
}

// walkOrder returns up to limit keys in the configured direction,
// starting after the resume cursor if one was set. A limit of 0 means
// unbounded.
//
// walkOrder does not see uncommitted tx writes. Entries staged via a
// write tx (stageSet / stageDelete) are invisible to ordered iteration;
// only Sorted.Filter merges the per-tx delta. A Where filter combined
// with OrderBy is applied as a post-filter against the committed
// entries, so the same caveat extends to the combined query.
//
//nolint:unused
func (q SortedQuery[K, E, V]) walkOrder(limit int) []K {
	q.sorted.mu.RLock()
	defer q.sorted.mu.RUnlock()
	entries := q.sorted.storage.entries
	if len(entries) == 0 {
		return nil
	}
	var start int
	switch q.dir {
	case Asc:
		if !q.hasCursor {
			start = 0
		} else {
			start = q.sorted.storage.upperBound(q.cursor)
		}
	case Desc:
		if !q.hasCursor {
			start = len(entries) - 1
		} else {
			start = q.sorted.storage.lowerBound(q.cursor) - 1
		}
	}
	return walkSorted(entries, start, q.dir, limit)
}

// walkSorted walks the sorted entry slice from start in the given direction,
// emitting up to limit keys. A limit of 0 means unbounded.
//
//nolint:unused
func walkSorted[K IndexKey, V cmp.Ordered](
	entries []sortedEntry[K, V],
	start int,
	dir Direction,
	limit int,
) []K {
	if limit < 0 {
		return nil
	}
	keys := make([]K, 0, limit)
	emitted := 0
	switch dir {
	case Asc:
		for i := start; i < len(entries); i++ {
			if limit > 0 && emitted >= limit {
				break
			}
			keys = append(keys, entries[i].Key)
			emitted++
		}
	case Desc:
		for i := start; i >= 0; i-- {
			if limit > 0 && emitted >= limit {
				break
			}
			keys = append(keys, entries[i].Key)
			emitted++
		}
	}
	return keys
}
