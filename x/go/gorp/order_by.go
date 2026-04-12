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

// OrderQuery is the V-erased handle stored on Retrieve. Implementations are
// produced by Sorted.Ordered (returning a SortedQuery[K, E, V]) and consumed
// by Retrieve.OrderBy. The interface erases V at the boundary so Retrieve
// (which is parameterized only on K and E) can store it without needing to
// know the value type. Implementations close over the typed cursor and
// direction inside their receiver, so the per-entry hot path is fully typed
// even though the interface signature is not.
type OrderQuery[K Key, E Entry[K]] interface {
	walkOrder(limit int) []K
}

// SortedQuery is the typed handle returned by Sorted.Ordered. It carries the
// direction, an optional resume cursor, and a back-reference to the index it
// was constructed from. SortedQuery satisfies OrderQuery[K, E] via the
// walkOrder method, which dispatches the per-page walk against the typed V
// captured in the receiver — no `any` boxing on either the cursor input or
// the per-entry comparison.
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

// walkOrder implements OrderQuery[K, E]. It takes the configured direction
// and cursor (both typed via the receiver's V), seeks to the resume point in
// the underlying sorted slice via binary search, and walks up to limit
// entries from there.
//
// IMPORTANT: walkOrder reads the committed sorted slice directly and does
// NOT consult the per-tx delta overlay. Entries staged via a write tx
// (stageSet / stageDelete) are invisible to ordered cursor iteration.
// Only the equality Filter path (Sorted.Filter) merges the tx delta.
// This is a known v1 limitation; if your use case requires ordered
// iteration that sees uncommitted writes, the delta merge must be
// extended to produce a sorted view — tracked as v2 follow-up work.
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
