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

	"go.uber.org/zap"
)

// Direction is the iteration direction used by a Sorted index when it is
// consumed via Retrieve.OrderBy.
type Direction uint8

const (
	// Asc walks the sorted index from smallest to largest value.
	Asc Direction = iota
	// Desc walks the sorted index from largest to smallest value.
	Desc
)

// OrderBy is an opaque handle returned by Sorted.Ordered and consumed by
// Retrieve.OrderBy. It carries direction and a walk closure that, given an
// optional cursor and page limit, returns the next page of primary keys
// along with a cursor usable to resume the walk. The value type V of the
// underlying Sorted index is erased so that Retrieve does not need to know it.
//
// The Key constraint is Key (not IndexKey) so that Retrieve, which does not
// require IndexKey, can hold an OrderBy[K, E]. The only valid source of an
// OrderBy is Sorted.Ordered, which does require IndexKey, so the walk
// closures are always constructed over comparable keys in practice.
type OrderBy[K Key, E Entry[K]] struct {
	name string
	walk func(after any, limit int) (keys []K, nextCursor any)
}

// Name returns the display name of the underlying sorted index.
func (o OrderBy[K, E]) Name() string { return o.name }

// Ordered returns an OrderBy handle that walks the Sorted index in the given
// direction. The handle is passed to Retrieve.OrderBy.
func (s *Sorted[K, E, V]) Ordered(dir Direction) OrderBy[K, E] {
	name := s.name
	return OrderBy[K, E]{
		name: name,
		walk: func(after any, limit int) ([]K, any) {
			s.mu.RLock()
			defer s.mu.RUnlock()
			entries := s.storage.entries
			if len(entries) == 0 {
				return nil, nil
			}
			var start int
			switch dir {
			case Asc:
				if after == nil {
					start = 0
				} else {
					typed, ok := after.(V)
					if !ok {
						zap.S().DPanicf(
							"gorp: OrderBy cursor for index %q has wrong type", name,
						)
						return nil, nil
					}
					start = s.storage.upperBound(typed)
				}
			case Desc:
				if after == nil {
					start = len(entries) - 1
				} else {
					typed, ok := after.(V)
					if !ok {
						zap.S().DPanicf(
							"gorp: OrderBy cursor for index %q has wrong type", name,
						)
						return nil, nil
					}
					start = s.storage.lowerBound(typed) - 1
				}
			}
			return walkSorted(entries, start, dir, limit)
		},
	}
}

// walkSorted walks the sorted entry slice from start in the given direction,
// emitting up to limit keys. A limit of 0 means unbounded. Returns the keys
// and the value of the last visited entry as the next cursor; the caller
// passes that cursor back via Retrieve.After to continue pagination.
func walkSorted[K IndexKey, V cmp.Ordered](
	entries []sortedEntry[K, V],
	start int,
	dir Direction,
	limit int,
) ([]K, any) {
	if limit < 0 {
		return nil, nil
	}
	keys := make([]K, 0, limit)
	var lastValue V
	var emitted int
	switch dir {
	case Asc:
		for i := start; i < len(entries); i++ {
			if limit > 0 && emitted >= limit {
				break
			}
			keys = append(keys, entries[i].Key)
			lastValue = entries[i].Value
			emitted++
		}
	case Desc:
		for i := start; i >= 0; i-- {
			if limit > 0 && emitted >= limit {
				break
			}
			keys = append(keys, entries[i].Key)
			lastValue = entries[i].Value
			emitted++
		}
	}
	if emitted == 0 {
		return keys, nil
	}
	return keys, lastValue
}
