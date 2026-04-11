// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Filter is a composable query filter that evaluates entries. The struct
// carries closures for raw-byte pre-screening and decoded-entry evaluation,
// plus an optional precomputed candidate-key set for filters backed by a
// secondary index. Retrieve uses Keys to short-circuit into the execKeys fast
// path; Eval and Raw run as post-checks against the fetched entries.
type Filter[K Key, E Entry[K]] struct {
	// Eval evaluates a decoded entry. Nil means no decoded-entry constraint.
	Eval func(ctx Context, e *E) (bool, error)
	// Raw evaluates the raw encoded bytes before decoding. Returning false skips
	// the entry without allocating a decoded value. Nil means no raw constraint.
	Raw func(data []byte) (bool, error)
	// Keys, if non-nil, is the precomputed set of candidate primary keys this
	// filter matches. Set by index-backed filter constructors (Lookup.Filter /
	// Sorted.Filter). When present, Retrieve.Exec converts the query into the
	// execKeys fast path: only those keys are fetched from the KV store, and
	// Eval/Raw run as post-checks. A nil Keys means the filter is unbounded
	// and Retrieve falls back to a full-table scan via execFilter.
	//
	// Compose-time semantics: And intersects Keys across children that have
	// them; Or unions across children only when every child has Keys (a single
	// child without Keys collapses Or back to unbounded). Not always drops
	// Keys because inverting a key set requires the universe.
	Keys []K
}

// Match wraps a bare closure as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Eval: f}
}

// MatchRaw wraps a raw-byte predicate as a Filter. The predicate runs before
// decoding; returning false skips the entry entirely.
func MatchRaw[K Key, E Entry[K]](f func(data []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Raw: f}
}

// And returns a filter that matches when ALL children match. Eval and Raw
// stages compose by short-circuiting on the first false. Keys composes by
// intersecting child key sets: a result Keys is set whenever at least one
// child has Keys, and equals the intersection of all children's Keys
// (children without Keys are unbounded and don't restrict the intersection).
func And[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	f := Filter[K, E]{
		Eval: func(ctx Context, e *E) (bool, error) {
			for _, f := range filters {
				if f.Eval == nil {
					continue
				}
				ok, err := f.Eval(ctx, e)
				if err != nil || !ok {
					return false, err
				}
			}
			return true, nil
		},
	}
	var raws []func([]byte) (bool, error)
	for _, child := range filters {
		if child.Raw != nil {
			raws = append(raws, child.Raw)
		}
	}
	if len(raws) > 0 {
		f.Raw = func(data []byte) (bool, error) {
			for _, r := range raws {
				ok, err := r(data)
				if err != nil || !ok {
					return false, err
				}
			}
			return true, nil
		}
	}
	f.Keys = intersectKeys[K, E](filters)
	return f
}

// Or returns a filter that matches when ANY child matches. Short-circuits on
// the first true. Raw pre-screening is not composed for Or because a branch
// without a Raw check may still match after decoding, so we must always
// decode. Keys composes only when EVERY child has Keys (otherwise an
// unbounded child means the result is unbounded), and equals the union of
// all children's Keys.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	f := Filter[K, E]{
		Eval: func(ctx Context, e *E) (bool, error) {
			for _, f := range filters {
				if f.Eval == nil {
					continue
				}
				ok, err := f.Eval(ctx, e)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
			return false, nil
		},
	}
	f.Keys = unionKeys[K, E](filters)
	return f
}

// Not returns a filter that inverts the child. Raw pre-screening is not
// composed for Not because inverting a raw rejection requires decoding to
// check the Eval stage. Keys are dropped for the same reason: inverting a
// key set requires the universe of all keys, which the filter does not have.
func Not[K Key, E Entry[K]](f Filter[K, E]) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E) (bool, error) {
			if f.Eval == nil {
				return false, nil
			}
			ok, err := f.Eval(ctx, e)
			return !ok, err
		},
	}
}

// intersectKeys returns the intersection of every child filter's Keys.
// Children with nil Keys are treated as unbounded and do not restrict the
// result. Returns nil if no child has Keys (the And filter is unbounded).
// Returns an empty non-nil slice if children have non-overlapping key sets.
//
// When exactly one child has Keys the input slice is returned directly
// without copying. Filter.Keys is treated as immutable by every consumer
// (Retrieve.execKeys passes it to GetMany which doesn't mutate), so sharing
// is safe and avoids a per-Where allocation on the common single-indexed-
// filter path.
func intersectKeys[K Key, E Entry[K]](filters []Filter[K, E]) []K {
	var bounded [][]K
	for _, f := range filters {
		if f.Keys != nil {
			bounded = append(bounded, f.Keys)
		}
	}
	if len(bounded) == 0 {
		return nil
	}
	if len(bounded) == 1 {
		return bounded[0]
	}
	// Start from the smallest list to minimize the membership-test count.
	smallest := 0
	for i := 1; i < len(bounded); i++ {
		if len(bounded[i]) < len(bounded[smallest]) {
			smallest = i
		}
	}
	candidates := bounded[smallest]
	out := make([]K, 0, len(candidates))
candidateLoop:
	for _, c := range candidates {
		for i, list := range bounded {
			if i == smallest {
				continue
			}
			if !containsKey(list, c) {
				continue candidateLoop
			}
		}
		out = append(out, c)
	}
	return out
}

// unionKeys returns the union of every child filter's Keys, or nil if any
// child has nil Keys (the Or filter is unbounded). Duplicates are removed by
// linear scan; the typical Keys list is small (<10 entries).
func unionKeys[K Key, E Entry[K]](filters []Filter[K, E]) []K {
	if len(filters) == 0 {
		return nil
	}
	for _, f := range filters {
		if f.Keys == nil {
			return nil
		}
	}
	var out []K
	for _, f := range filters {
		for _, k := range f.Keys {
			if !containsKey(out, k) {
				out = append(out, k)
			}
		}
	}
	return out
}

// containsKey reports whether keys contains target via linear scan. Uses a
// generic equality helper because K may not be comparable in the type-set
// sense (gorp.Key admits ~[]byte), so map[K] is unavailable here.
func containsKey[K Key](keys []K, target K) bool {
	for _, k := range keys {
		if keysEqual(k, target) {
			return true
		}
	}
	return false
}

// keysEqual reports whether two keys are equal. Uses interface equality on
// the boxed values, which works for every Key type that satisfies "strictly
// comparable". For ~[]byte keys (which are not strictly comparable) this
// will panic at runtime; in practice indexed filters require IndexKey, so
// the only path that builds non-nil Filter.Keys produces strictly comparable
// keys.
func keysEqual[K Key](a, b K) bool {
	return any(a) == any(b)
}
