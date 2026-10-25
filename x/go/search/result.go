// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search

import "slices"

// Result is the outcome of a search.
type Result[K comparable] struct {
	// Keys holds the key of each matching entry, closest match first.
	Keys []K
	// ranks holds the rank of each key. A lower rank is a closer match.
	ranks []uint32
	// complete is true when every key matches every word of the term.
	complete bool
}

// newRank packs a match score and a token count into a rank that sorts higher scores
// first, then fewer tokens.
func newRank(score int32, tokens int) uint32 {
	const maxScore = 1<<24 - 1
	return uint32(maxScore-min(score, maxScore))<<8 | uint32(min(tokens, 255))
}

// Merge combines the results of one term searched in several indexes, closest match
// first. Results whose entries match every word of the term replace the results that
// match only some words, so Merge returns what one index holding every entry would.
// Equal matches keep the order of results. Merge returns at most limit keys, or every
// key when limit is 0. Searching each index with the same limit does not change the
// result.
func Merge[K comparable](limit int, results ...Result[K]) Result[K] {
	complete := slices.ContainsFunc(results, func(r Result[K]) bool {
		return r.complete && len(r.Keys) > 0
	})
	total := 0
	for _, r := range results {
		if r.complete == complete {
			total += len(r.Keys)
		}
	}
	packed := make([]uint64, 0, total)
	for _, r := range results {
		if r.complete != complete {
			continue
		}
		for _, rank := range r.ranks {
			packed = append(packed, uint64(rank)<<32|uint64(len(packed)))
		}
	}
	slices.Sort(packed)
	packed = packed[:limited(len(packed), limit)]
	keys := make([]K, 0, total)
	for _, r := range results {
		if r.complete == complete {
			keys = append(keys, r.Keys...)
		}
	}
	merged := Result[K]{
		Keys:     make([]K, len(packed)),
		ranks:    make([]uint32, len(packed)),
		complete: complete,
	}
	for i, p := range packed {
		merged.Keys[i] = keys[uint32(p)]
		merged.ranks[i] = uint32(p >> 32)
	}
	return merged
}

// limited returns n cut to limit, or n when limit is 0.
func limited(n, limit int) int {
	if limit > 0 {
		return min(n, limit)
	}
	return n
}

// MapKeys returns r with each key replaced by f of the key.
func MapKeys[K, L comparable](r Result[K], f func(K) L) Result[L] {
	keys := make([]L, len(r.Keys))
	for i, k := range r.Keys {
		keys[i] = f(k)
	}
	return Result[L]{Keys: keys, ranks: r.ranks, complete: r.complete}
}
