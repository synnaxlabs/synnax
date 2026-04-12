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
	"slices"

	"github.com/synnaxlabs/x/set"
)

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
	// Sorted.Filter) via newIndexedFilter, and by And/Or composition via
	// intersectKeys / unionKeys. When present, Retrieve.Exec converts the
	// query into the execKeys fast path: only those keys are fetched from the
	// KV store, and Eval/Raw run as post-checks. A nil Keys means the filter
	// is unbounded and Retrieve falls back to a full-table scan via
	// execFilter.
	//
	// Compose-time semantics: And intersects Keys across children that have
	// them; Or unions across children only when every child has Keys (a single
	// child without Keys collapses Or back to unbounded). Not always drops
	// Keys because inverting a key set requires the universe.
	Keys []K
	// membership is a private O(1) lookup mirror of Keys, populated by
	// newIndexedFilter at construction time. Because newIndexedFilter is
	// constrained to K : IndexKey (i.e. strictly comparable), the underlying
	// set.Set uses == directly without any-boxing or panicking on ~[]byte.
	// Composition (And/Or) and execKeys both probe through this predicate
	// instead of doing a linear scan with any() equality.
	//
	// Filters constructed without Keys leave membership nil; containsKey
	// gates on it being non-nil, so reading is always safe.
	membership keyMembership[K]
	// rebuildMembership reconstructs a fresh membership predicate over a
	// caller-provided slice of K. Captured at index-filter construction time
	// (when K is provably IndexKey) and propagated through And/Or composition
	// so the resulting Filter can rebuild membership over its intersected /
	// unioned Keys without needing the IndexKey constraint at the call site.
	// Nil for filters that never carried a Keys set.
	rebuildMembership func([]K) keyMembership[K]
}

// keyMembership is the typed O(1) membership predicate carried alongside
// Filter.Keys. The concrete implementation is set.Set[K], constructed only
// inside newIndexedFilter where K is provably comparable.
type keyMembership[K Key] interface {
	Contains(K) bool
}

// newIndexedFilter constructs a Filter whose Keys is a precomputed candidate
// set produced by an in-memory secondary index. Both the slice and a paired
// O(1) membership predicate are stored on the Filter so downstream consumers
// (intersectKeys, unionKeys, Retrieve.execKeys) can probe without any-boxing
// or linear scans.
//
// The K constraint is IndexKey (not Key) precisely so set.Set[K] is
// type-safe: index-backed filters are the only path that ever populates
// Filter.Keys, and they always know K is strictly comparable. The returned
// Filter is type-erased back to Filter[K, E] for K : Key, but the captured
// rebuildMembership closure preserves the comparable view so And/Or
// composition can rebuild a typed membership over its result keys.
func newIndexedFilter[K IndexKey, E Entry[K]](keys []K) Filter[K, E] {
	rebuild := func(k []K) keyMembership[K] { return set.FromSlice(k) }
	return Filter[K, E]{
		Keys:              keys,
		membership:        rebuild(keys),
		rebuildMembership: rebuild,
	}
}

// containsKey reports whether k is present in the filter's precomputed Keys
// set. Returns false when membership is nil, which happens when the filter
// either has no Keys set at all or carries an Eval-only constraint. Routes
// through the typed predicate built at construction, so the comparison uses
// == directly and never allocates.
func (f Filter[K, E]) containsKey(k K) bool {
	if f.membership == nil {
		return false
	}
	return f.membership.Contains(k)
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
	f.Keys, f.rebuildMembership = intersectKeys[K, E](filters)
	if f.rebuildMembership != nil && f.Keys != nil {
		f.membership = f.rebuildMembership(f.Keys)
	}
	return f
}

// Or returns a filter that matches when ANY child matches. Short-circuits on
// the first true. Raw pre-screening is not composed for Or because a branch
// without a Raw check may still match after decoding, so we must always
// decode. Keys composes only when EVERY child has Keys (otherwise an
// unbounded child means the result is unbounded), and equals the union of
// all children's Keys.
//
// Eval handles index-backed children (Eval == nil, Keys != nil) by probing
// the child's typed membership against the entry's key. This is required
// because execKeys fetches the union of all children's Keys then runs each
// fetched entry through Or.Eval; without the membership probe, no Keys-only
// child would ever return true and the post-filter would drop every result.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	f := Filter[K, E]{
		Eval: func(ctx Context, e *E) (bool, error) {
			key := (*e).GorpKey()
			for _, f := range filters {
				if f.Eval != nil {
					ok, err := f.Eval(ctx, e)
					if err != nil {
						return false, err
					}
					if ok {
						return true, nil
					}
					continue
				}
				if f.Keys != nil && f.containsKey(key) {
					return true, nil
				}
			}
			return false, nil
		},
	}
	f.Keys, f.rebuildMembership = unionKeys[K, E](filters)
	if f.rebuildMembership != nil && f.Keys != nil {
		f.membership = f.rebuildMembership(f.Keys)
	}
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

// BoundFilter is a Filter parameterized over a service-defined Retrieve type
// R. It's the underlying type behind the per-service Filter alias emitted by
// the oracle query plugin: a closure that takes the caller's Retrieve and
// produces a gorp.Filter[K, E] bound to it. Service code uses BoundFilter so
// filter constructors can read from r.indexes / r.label / r.hostProvider when
// evaluated by Retrieve.Where, while pure constructors can ignore r entirely.
//
// The MatchBound / AndBound / OrBound / NotBound helpers below let generated
// code stay one-liner thin instead of re-emitting the same closure plumbing
// per service.
type BoundFilter[R any, K Key, E Entry[K]] func(r R) Filter[K, E]

// MatchBound wraps a closure that needs the Retrieve R into a BoundFilter.
// The Retrieve value is supplied by the per-service Where method when the
// query is evaluated.
func MatchBound[R any, K Key, E Entry[K]](
	f func(ctx Context, r R, e *E) (bool, error),
) BoundFilter[R, K, E] {
	return func(r R) Filter[K, E] {
		return Filter[K, E]{Eval: func(ctx Context, e *E) (bool, error) {
			return f(ctx, r, e)
		}}
	}
}

// AndBound returns a BoundFilter that matches when all provided filters
// match. Each child is bound to the same Retrieve before being composed via
// gorp.And, so the resulting filter inherits And's Eval/Raw/Keys composition
// semantics.
func AndBound[R any, K Key, E Entry[K]](fs ...BoundFilter[R, K, E]) BoundFilter[R, K, E] {
	return func(r R) Filter[K, E] {
		inner := make([]Filter[K, E], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return And(inner...)
	}
}

// OrBound returns a BoundFilter that matches when any provided filter
// matches. Bound children are composed via gorp.Or.
func OrBound[R any, K Key, E Entry[K]](fs ...BoundFilter[R, K, E]) BoundFilter[R, K, E] {
	return func(r R) Filter[K, E] {
		inner := make([]Filter[K, E], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return Or(inner...)
	}
}

// NotBound returns a BoundFilter that inverts the provided filter via
// gorp.Not after binding it to the Retrieve.
func NotBound[R any, K Key, E Entry[K]](f BoundFilter[R, K, E]) BoundFilter[R, K, E] {
	return func(r R) Filter[K, E] {
		return Not(f(r))
	}
}

// intersectKeys returns the intersection of every child filter's Keys plus
// a closure capable of rebuilding a typed membership predicate over the
// result. Children with nil Keys are treated as unbounded and do not
// restrict the result. The returned slice is nil when no child has Keys
// (the And filter is unbounded), and an empty non-nil slice when children
// have non-overlapping key sets.
//
// When exactly one child has Keys, that child's slice is returned directly
// without copying — Filter.Keys is treated as immutable by every consumer
// (Retrieve.execKeys passes it to GetMany unchanged), so sharing is safe
// and avoids a per-Where allocation on the common single-indexed-filter
// path.
//
// The multi-child path sorts a working copy of the bounded children by Keys
// length so the smallest set is the candidate driver. Membership probing on
// the larger sets uses each filter's O(1) typed predicate — no any-boxing,
// no panic on ~[]byte tables (which can't reach this path because indexed
// filters require IndexKey).
func intersectKeys[K Key, E Entry[K]](
	filters []Filter[K, E],
) (keys []K, rebuild func([]K) keyMembership[K]) {
	var bounded []Filter[K, E]
	for _, f := range filters {
		if f.Keys != nil {
			bounded = append(bounded, f)
			if rebuild == nil {
				rebuild = f.rebuildMembership
			}
		}
	}
	if len(bounded) == 0 {
		return nil, nil
	}
	if len(bounded) == 1 {
		return bounded[0].Keys, rebuild
	}
	slices.SortFunc(bounded, func(a, b Filter[K, E]) int {
		return len(a.Keys) - len(b.Keys)
	})
	candidates, rest := bounded[0].Keys, bounded[1:]
	out := make([]K, 0, len(candidates))
	for _, c := range candidates {
		inAll := true
		for _, f := range rest {
			if !f.containsKey(c) {
				inAll = false
				break
			}
		}
		if inAll {
			out = append(out, c)
		}
	}
	return out, rebuild
}

// unionKeys returns the union of every child filter's Keys plus a closure
// capable of rebuilding a typed membership predicate over the result. The
// returned slice is nil when any child has nil Keys (the Or filter is
// unbounded). Duplicates across children are detected by probing the
// membership predicates of earlier children, which is O(1) per probe and
// never any-boxes.
func unionKeys[K Key, E Entry[K]](
	filters []Filter[K, E],
) (keys []K, rebuild func([]K) keyMembership[K]) {
	if len(filters) == 0 {
		return nil, nil
	}
	var total int
	for _, f := range filters {
		if f.Keys == nil {
			return nil, nil
		}
		total += len(f.Keys)
		if rebuild == nil {
			rebuild = f.rebuildMembership
		}
	}
	out := make([]K, 0, total)
	for i, f := range filters {
		for _, k := range f.Keys {
			seen := false
			for _, prior := range filters[:i] {
				if prior.containsKey(k) {
					seen = true
					break
				}
			}
			if !seen {
				out = append(out, k)
			}
		}
	}
	return out, rebuild
}
