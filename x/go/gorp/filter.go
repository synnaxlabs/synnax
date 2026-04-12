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
	"sync"

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
	// Raw evaluates the raw encoded bytes before decoding. The filter receives
	// both the pebble key and the encoded value, so callers can short-circuit
	// on key-shaped data without ever touching the value (or vice versa).
	// Returning false skips the entry without allocating a decoded value. Nil
	// means no raw constraint.
	Raw func(key, value []byte) (bool, error)
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
	// membership is a lazy O(1) lookup mirror of Keys, populated on first
	// use by *lazyMembership.contains. Index-backed filter constructors
	// (newIndexedFilter, newBytesIndexedFilter) wrap Keys in a lazyMembership
	// without building the underlying set; the build happens only the first
	// time a downstream consumer actually probes (intersectKeys, unionKeys,
	// execKeys' effectiveKeys merge, Or.Eval for Keys-only children).
	//
	// Lazy matters a lot for the composition path: a bound filter whose
	// Keys set has N elements would eagerly cost an N-entry map at
	// construction, but if intersectKeys ends up walking THIS filter's
	// Keys directly (because it's the larger side), we never need to
	// probe its membership and that N-entry map never allocates. On a
	// 12500-key filter that's a ~150 KB saving per query.
	//
	// Nil when the filter either has no Keys or was constructed without
	// an IndexKey-constrained builder (e.g., plain Match / MatchRaw).
	// containsKey gates on it being non-nil, so reading is always safe.
	membership *lazyMembership[K]
}

// keyMembership is the typed O(1) membership predicate carried alongside
// Filter.Keys. Implementations are constructed at IndexKey-constrained call
// sites and type-erased back through keyMembership[K] for K : Key.
type keyMembership[K Key] interface {
	Contains(K) bool
}

// lazyMembership wraps a Keys slice plus a deferred build function so the
// underlying keyMembership is only materialized on first probe. Consumers
// call contains() which does a sync.Once-guarded build + lookup.
//
// The build function is captured at the IndexKey-constrained construction
// site (newIndexedFilter or newBytesIndexedFilter), which is where K is
// provably comparable. Propagation through And/Or carries the same build
// function to the composed Filter, so any Filter derived from an indexed
// source — directly or via composition — can rebuild membership on demand
// without re-establishing the IndexKey constraint at the consumer.
type lazyMembership[K Key] struct {
	once  sync.Once
	set   keyMembership[K]
	keys  []K
	build func([]K) keyMembership[K]
}

// contains reports whether k is in the underlying key set. On first call,
// builds the keyMembership via the captured build function; subsequent
// calls hit the cached result. sync.Once makes this safe under concurrent
// access even though Filter values are usually goroutine-local, and the
// cost of the Once guard after first call is a single atomic load.
func (l *lazyMembership[K]) contains(k K) bool {
	l.once.Do(func() { l.set = l.build(l.keys) })
	return l.set.Contains(k)
}

// newLazyMembership wraps keys in a lazy membership that materializes via
// build on first probe. Does not allocate the underlying set.
func newLazyMembership[K Key](
	keys []K,
	build func([]K) keyMembership[K],
) *lazyMembership[K] {
	return &lazyMembership[K]{keys: keys, build: build}
}

// indexedKeyMembership is the package-level build function used by
// newIndexedFilter. Extracted so the closure captured by every indexed
// filter is a reference to a single static generic function rather than
// an escaped closure, avoiding a per-construction heap allocation that a
// closure literal would cause.
func indexedKeyMembership[K IndexKey](keys []K) keyMembership[K] {
	return set.FromSlice(keys)
}

// newIndexedFilter constructs a Filter whose Keys is a precomputed candidate
// set produced by an in-memory secondary index. Only the slice is stored
// eagerly; the membership predicate is wrapped in a *lazyMembership and
// built on first probe (see lazyMembership.contains).
//
// The K constraint is IndexKey (not Key) precisely so set.Set[K] is
// type-safe: index-backed filters are the only path that ever populates
// Filter.Keys, and they always know K is strictly comparable. The returned
// Filter is type-erased back to Filter[K, E] for K : Key, but the captured
// build function preserves the comparable view so And/Or composition can
// rebuild a typed membership over its result keys when needed.
func newIndexedFilter[K IndexKey, E Entry[K]](keys []K) Filter[K, E] {
	return Filter[K, E]{
		Keys:       keys,
		membership: newLazyMembership(keys, indexedKeyMembership[K]),
	}
}

// bytesIndexedKeyMembership is the build function counterpart to
// indexedKeyMembership for []byte-keyed tables.
func bytesIndexedKeyMembership(keys [][]byte) keyMembership[[]byte] {
	m := make(bytesKeyMembership, len(keys))
	for _, b := range keys {
		m[string(b)] = struct{}{}
	}
	return m
}

// newBytesIndexedFilter is the []byte-keyed analogue of newIndexedFilter.
// It exists because []byte is not strictly comparable and so cannot satisfy
// IndexKey, but tables whose primary key is []byte (relationships, composite
// encoded keys) still need indexed filters that route through execKeys. The
// membership predicate is a string-keyed map populated from the byte
// slices, which preserves O(1) probes at the cost of a no-alloc string
// conversion per Contains call.
func newBytesIndexedFilter[E Entry[[]byte]](keys [][]byte) Filter[[]byte, E] {
	return Filter[[]byte, E]{
		Keys:       keys,
		membership: newLazyMembership(keys, bytesIndexedKeyMembership),
	}
}

// bytesKeyMembership is a string-keyed O(1) membership predicate over a set
// of []byte keys. It is the byte-table counterpart to set.Set[K] used by
// strictly-comparable IndexKey tables. Contains converts the probe to a
// string at the call site; on modern Go this is a no-alloc conversion.
type bytesKeyMembership map[string]struct{}

// Contains implements keyMembership[[]byte].
func (m bytesKeyMembership) Contains(k []byte) bool {
	_, ok := m[string(k)]
	return ok
}

// containsKey reports whether k is present in the filter's precomputed Keys
// set. Returns false when membership is nil, which happens when the filter
// either has no Keys set at all or carries an Eval-only constraint. Routes
// through the lazy predicate, which triggers a one-time build on first call.
func (f Filter[K, E]) containsKey(k K) bool {
	if f.membership == nil {
		return false
	}
	return f.membership.contains(k)
}

// Match wraps a bare closure as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Eval: f}
}

// MatchRaw wraps a raw-byte predicate as a Filter. The predicate runs before
// decoding and receives both the pebble key and the encoded value; returning
// false skips the entry entirely.
func MatchRaw[K Key, E Entry[K]](f func(key, value []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Raw: f}
}

// And returns a filter that matches when ALL children match. Eval and Raw
// stages compose by short-circuiting on the first false. Keys composes by
// intersecting child key sets: a result Keys is set whenever at least one
// child has Keys, and equals the intersection of all children's Keys
// (children without Keys are unbounded and don't restrict the intersection).
//
// The Eval and Raw closure allocations are skipped entirely when no child
// has them. For a common "all children are index-backed Keys-only filters"
// composition this means And allocates nothing but the intersection slice
// and a lazy membership wrapper — no closure capture, no heap dispatch.
func And[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	var f Filter[K, E]
	hasAnyEval := false
	for _, child := range filters {
		if child.Eval != nil {
			hasAnyEval = true
			break
		}
	}
	if hasAnyEval {
		// Only build the iterating closure when at least one child has
		// an Eval predicate. If every child is Keys-only, execKeys
		// fetches exactly the intersection and match() is trivially
		// true — so leaving f.Eval == nil is semantically correct and
		// saves the closure + captured-filters allocation.
		f.Eval = func(ctx Context, e *E) (bool, error) {
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
		}
	}
	var firstRaw func(key, value []byte) (bool, error)
	rawCount := 0
	for _, child := range filters {
		if child.Raw != nil {
			if rawCount == 0 {
				firstRaw = child.Raw
			}
			rawCount++
		}
	}
	switch rawCount {
	case 0:
		// No Raw at any child: leave f.Raw nil, no closure.
	case 1:
		// Exactly one Raw: assign directly, no wrapper closure.
		f.Raw = firstRaw
	default:
		// 2+ Raw children: need the iterating wrapper. Rebuild the list
		// here to avoid leaking every child (including Raw-less ones)
		// into the captured environment.
		raws := make([]func([]byte, []byte) (bool, error), 0, rawCount)
		for _, child := range filters {
			if child.Raw != nil {
				raws = append(raws, child.Raw)
			}
		}
		f.Raw = func(key, value []byte) (bool, error) {
			for _, r := range raws {
				ok, err := r(key, value)
				if err != nil || !ok {
					return false, err
				}
			}
			return true, nil
		}
	}
	var build func([]K) keyMembership[K]
	f.Keys, build = intersectKeys[K, E](filters)
	if build != nil && f.Keys != nil {
		f.membership = newLazyMembership(f.Keys, build)
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
//
// If every child is Keys-only and every child has a full Keys set (so Or
// could compose a complete union), f.Eval is left nil: execKeys fetches
// exactly the union and match() is trivially true, so the closure is dead
// weight. Mixed compositions still need the closure.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	var f Filter[K, E]
	var build func([]K) keyMembership[K]
	f.Keys, build = unionKeys[K, E](filters)
	if build != nil && f.Keys != nil {
		f.membership = newLazyMembership(f.Keys, build)
	}
	// If the union composed successfully AND every child was Keys-only,
	// we can skip the Eval closure: execKeys will fetch exactly the
	// union and match() can short-circuit to true.
	allKeysOnly := f.Keys != nil
	if allKeysOnly {
		for _, child := range filters {
			if child.Eval != nil {
				allKeysOnly = false
				break
			}
		}
	}
	if !allKeysOnly {
		f.Eval = func(ctx Context, e *E) (bool, error) {
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
		}
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
// the build function needed to construct a lazy membership over the result.
// Children with nil Keys are treated as unbounded and do not restrict the
// result. The returned slice is nil when no child has Keys (the And filter
// is unbounded), and an empty non-nil slice when children have
// non-overlapping key sets.
//
// When exactly one child has Keys, that child's slice is returned directly
// without copying — Filter.Keys is treated as immutable by every consumer
// (Retrieve.execKeys passes it to GetMany unchanged), so sharing is safe
// and avoids a per-Where allocation on the common single-indexed-filter
// path.
//
// The multi-child path sorts a working copy of the bounded children by Keys
// length and walks the LARGEST one, probing the memberships of the smaller
// ones for each candidate. This looks counterintuitive — walking the
// larger side does more comparison work — but it's the memory-optimal
// choice: walking a filter means touching its Keys slice directly without
// ever calling containsKey on it, so we never materialize its membership
// map. With lazy membership, the walked side's hmap is never allocated.
//
// Concretely, for a common composition like Where(And(narrow, wide)) where
// narrow has 1 key and wide has 12500 keys, walking smaller + probing
// wider would build a 12500-entry hmap just to serve one probe. Walking
// wider + probing narrow's lazily-built 1-entry hmap costs ~12500 extra
// map lookups but saves the 12500-entry allocation — a ~150 KB per-op
// reduction, which is the dominant cost in the composition benchmark.
// For control-system workloads where GC pauses are the bottleneck, this
// trade (up to ~2x CPU for proportional memory reduction) is a clear win.
//
// Membership probing always goes through each filter's lazy predicate,
// which builds on first call and caches via sync.Once. Subsequent probes
// on the same filter hit the cached map at no extra cost.
func intersectKeys[K Key, E Entry[K]](
	filters []Filter[K, E],
) (keys []K, build func([]K) keyMembership[K]) {
	bounded := make([]Filter[K, E], 0, len(filters))
	for _, f := range filters {
		if f.Keys != nil {
			bounded = append(bounded, f)
			if build == nil && f.membership != nil {
				build = f.membership.build
			}
		}
	}
	if len(bounded) == 0 {
		return nil, nil
	}
	if len(bounded) == 1 {
		return bounded[0].Keys, build
	}
	slices.SortFunc(bounded, func(a, b Filter[K, E]) int {
		return len(a.Keys) - len(b.Keys)
	})
	// Walk the LARGEST (last after ascending sort), probe the smaller
	// siblings' lazily-built memberships. See the method comment for
	// why this is the memory-optimal choice.
	candidates := bounded[len(bounded)-1].Keys
	rest := bounded[:len(bounded)-1]
	// Pre-size the result to the SMALLEST side's length, not the walker's:
	// the intersection can't be larger than any single child's key set, and
	// bounded[0] is the smallest after the ascending sort. Sizing to the
	// walker's length (which could be orders of magnitude larger) would
	// allocate a result buffer wildly out of proportion to the actual
	// intersection size — measured at ~50 KB per op on the composition
	// benchmark before this fix.
	out := make([]K, 0, len(bounded[0].Keys))
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
	return out, build
}

// unionKeys returns the union of every child filter's Keys plus the build
// function needed to construct a lazy membership over the result. The
// returned slice is nil when any child has nil Keys (the Or filter is
// unbounded). Duplicates across children are detected by probing the
// lazily-materialized membership predicates of earlier children, which is
// O(1) per probe after the one-time build.
//
// Filters are processed in ascending-Keys-length order, which puts the
// largest child last. Under lazy membership, walking child i triggers
// membership builds for filters[:i] (the smaller ones) but never for
// filters[i+1:]. Sorting ascending means the largest filter's membership
// is never built — only probed against by later iterations, which don't
// happen because the largest is the last walk step. Symmetric to the
// intersectKeys flip: the largest contributes its Keys directly, no map.
func unionKeys[K Key, E Entry[K]](
	filters []Filter[K, E],
) (keys []K, build func([]K) keyMembership[K]) {
	if len(filters) == 0 {
		return nil, nil
	}
	bounded := make([]Filter[K, E], 0, len(filters))
	var total int
	for _, f := range filters {
		if f.Keys == nil {
			return nil, nil
		}
		bounded = append(bounded, f)
		total += len(f.Keys)
		if build == nil && f.membership != nil {
			build = f.membership.build
		}
	}
	slices.SortFunc(bounded, func(a, b Filter[K, E]) int {
		return len(a.Keys) - len(b.Keys)
	})
	out := make([]K, 0, total)
	for i, f := range bounded {
		for _, k := range f.Keys {
			seen := false
			for _, prior := range bounded[:i] {
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
	return out, build
}
