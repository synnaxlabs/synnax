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
	"bytes"
	"context"
	"reflect"
	"slices"
	"sync"

	"github.com/synnaxlabs/x/set"
)

// Filter is a composable query filter that evaluates entries. The struct
// carries closures for raw-byte pre-screening and decoded-entry evaluation,
// plus an optional precomputed candidate-key set for filters backed by a
// secondary index. Retrieve uses keys to short-circuit into the execKeys fast
// path; eval and raw run as post-checks against the fetched entries.
//
// A filter can carry any subset of (keys, raw, eval); the three kinds are
// independent dispatch paths and composition (And/Or/Not) handles each
// correctly so that MatchRaw filters compose under Or/Not without losing the
// raw-bytes path. eval receives the pebble key and encoded value alongside
// the decoded entry so dispatch helpers can invoke a child's raw at decode
// time when the child has no eval of its own.
type Filter[K Key, E Entry[K]] struct {
	// eval evaluates a decoded entry. Receives both the pebble key and the
	// raw encoded value so And/Or/Not composition can dispatch to a child's
	// raw at decode time when the child has no eval. Nil means no
	// decoded-entry constraint.
	eval func(ctx Context, e *E, key, value []byte) (bool, error)
	// raw evaluates the raw encoded bytes before decoding. The filter receives
	// both the pebble key and the encoded value, so callers can short-circuit
	// on key-shaped data without ever touching the value (or vice versa).
	// Returning false skips the entry without allocating a decoded value. Nil
	// means no raw constraint.
	raw func(key, value []byte) (bool, error)
	// keys, if non-nil, is the candidate set of primary keys this filter
	// matches. Set by Retrieve.Exec after invoking the resolver returned
	// from an index-backed Filter constructor (Lookup.Filter /
	// Sorted.Filter / BytesLookup.Filter), or by And/Or composition via
	// intersectKeys / unionKeys when all children are eager. When
	// present, Retrieve.Exec routes into the execKeys fast path: only
	// those keys are fetched from the KV store, and eval/raw run as
	// post-checks. A nil keys means the filter is unbounded and
	// Retrieve falls back to a full-table scan via execFilter.
	//
	// Compose-time semantics: And intersects keys across children that have
	// them; Or unions across children only when every child has keys (a single
	// child without keys collapses Or back to unbounded). Not always drops
	// keys because inverting a key set requires the universe.
	keys []K
	// membership is a lazy O(1) lookup mirror of keys, populated on first
	// use by *lazyMembership.contains. Retrieve.Exec wraps the resolver's
	// returned keys in a lazyMembership without building the underlying
	// set; the build happens only the first time a downstream consumer
	// actually probes (intersectKeys, unionKeys, execKeys' effectiveKeys
	// merge, Or.eval for keys-only children).
	//
	// Lazy matters a lot for the composition path: a bound filter whose
	// keys set has N elements would eagerly cost an N-entry map at
	// construction, but if intersectKeys ends up walking THIS filter's
	// keys directly (because it's the larger side), we never need to
	// probe its membership and that N-entry map never allocates. On a
	// 12500-key filter that's a ~150 KB saving per query.
	//
	// Nil when the filter either has no keys or was constructed without
	// an IndexKey-constrained builder (e.g., plain Match / MatchRaw).
	// containsKey gates on it being non-nil, so reading is always safe.
	membership *lazyMembership[K]
	// resolve, if non-nil, computes keys (and a membership build
	// function) at Retrieve.Exec time with the open tx in scope. It is
	// set by index-backed constructors that need read-your-own-writes
	// semantics: the resolver reads committed index state for the
	// queried values and overlays the per-tx delta tracked on
	// tx.txIdentity(). When resolve is present, Retrieve.Exec invokes
	// it before dispatching to execKeys so the rest of the pipeline
	// sees the merged keys through the normal keys/membership fields.
	//
	// Filter composition (And/Or) propagates resolvers: if any child
	// has resolve, the composed filter also carries one that fires
	// each child and recombines via intersectKeys / unionKeys. Not
	// always drops resolve (inverting a key set requires the universe).
	//
	// resolve returning (nil, nil, nil) means "no candidate keys" —
	// the execKeys fast path treats this as an empty result, NOT as
	// unbounded. An unbounded filter has no keys and no resolver.
	resolve resolveFilter[K]
}

// resolveFilter is the signature for a deferred Filter resolver. It
// returns the effective candidate keys for an indexed filter under the
// given transaction, plus the build function that downstream composition
// and execKeys use to construct a lazy O(1) membership predicate over
// the returned keys.
//
// The keys returned are the merge of committed index state and any
// per-tx delta the transaction has staged. The build function is a
// package-level generic over K (indexedKeyMembership or
// bytesIndexedKeyMembership) captured at construction time; it's
// returned rather than stored in the Filter directly so composed
// resolvers can forward the correct build function for the composed
// key set.
type resolveFilter[K Key] func(
	ctx context.Context,
	tx Tx,
) (keys []K, build func([]K) keyMembership[K], err error)

// keyMembership is the typed O(1) membership predicate carried alongside
// Filter.keys. Implementations are constructed at IndexKey-constrained call
// sites and type-erased back through keyMembership[K] for K : Key.
type keyMembership[K Key] interface {
	Contains(K) bool
}

// lazyMembership wraps a keys slice plus a deferred build function so the
// underlying keyMembership is only materialized on first probe. Consumers
// call contains() which does a sync.Once-guarded build + lookup.
//
// The build function is captured at the IndexKey-constrained index
// (Lookup / Sorted / BytesLookup), which is where K is provably
// comparable. Resolvers return it alongside the resolved keys; the
// Retrieve.Exec hook and And/Or resolver composition rewrap the same
// build function in a new lazyMembership. Any Filter derived from an
// indexed source — directly or via composition — can rebuild
// membership on demand without re-establishing the IndexKey constraint
// at the consumer.
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

// indexedKeyMembership is the package-level build function returned by
// Lookup.Filter / Sorted.Filter resolvers. Extracted so the closure
// captured by every indexed filter resolver is a reference to a single
// static generic function rather than an escaped closure, avoiding a
// per-construction heap allocation that a closure literal would cause.
//
// Returned from resolvers alongside the resolved keys so Retrieve.Exec
// (and And/Or composition) can wrap them in a lazyMembership whose
// first-probe build produces a typed O(1) membership predicate.
func indexedKeyMembership[K IndexKey](keys []K) keyMembership[K] {
	return set.New(keys...)
}

// bytesIndexedKeyMembership is the build function counterpart to
// indexedKeyMembership for []byte-keyed tables. Returned by
// BytesLookup.Filter's resolver alongside the resolved keys.
func bytesIndexedKeyMembership(keys [][]byte) keyMembership[[]byte] {
	m := make(bytesKeyMembership, len(keys))
	for _, b := range keys {
		m[string(b)] = struct{}{}
	}
	return m
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

// containsKey reports whether k is present in the filter's precomputed keys
// set. Routes through the lazy O(1) predicate when the filter carries one
// (built by index-backed constructors and And/Or composition), and falls
// back to a linear scan over f.keys when membership wasn't built (e.g.,
// for a bare MatchKeys filter outside any composition with an indexed
// child). The linear path is O(len(f.keys)) per probe; acceptable because
// MatchKeys is typically used with small key sets and large sets are
// expected to flow through index-backed filters that build membership
// eagerly.
func (f Filter[K, E]) containsKey(k K) bool {
	if f.membership != nil {
		return f.membership.contains(k)
	}
	if f.keys == nil {
		return false
	}
	return linearContainsKey(f.keys, k)
}

// linearContainsKey is the O(n) fallback used by containsKey when no
// pre-built membership predicate is available. Handles []byte-shaped K
// via bytes.Equal (since []byte is not == comparable); comparable K
// dispatches through interface equality.
func linearContainsKey[K Key](keys []K, k K) bool {
	if len(keys) == 0 {
		return false
	}
	if reflect.TypeOf(keys[0]).Kind() == reflect.Slice {
		probe := reflect.ValueOf(k).Bytes()
		for _, fk := range keys {
			if bytes.Equal(reflect.ValueOf(fk).Bytes(), probe) {
				return true
			}
		}
		return false
	}
	probe := any(k)
	for _, fk := range keys {
		if any(fk) == probe {
			return true
		}
	}
	return false
}

// Match wraps a decoded-entry predicate as a Filter. The user closure is the
// simple `(ctx, *E)` shape; Match wraps it into the internal 4-arg eval that
// ignores the raw key/value bytes.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{
		eval: func(ctx Context, e *E, _, _ []byte) (bool, error) {
			return f(ctx, e)
		},
	}
}

// MatchRaw wraps a raw-byte predicate as a Filter. The predicate runs before
// decoding and receives both the pebble key and the encoded value; returning
// false skips the entry entirely. Under And the raw pre-screen composes by
// AND-short-circuit, under Or it survives only when every child is raw-only,
// and under Not it inverts in place when the child is raw-only — preserving
// the no-decode pre-screen path through composition wherever semantically
// sound.
func MatchRaw[K Key, E Entry[K]](f func(key, value []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{raw: f}
}

// MatchKeys returns a Filter that restricts results to entries whose primary
// key is in the given set. This is the unified replacement for the
// previous Retrieve.WhereKeys method: pass MatchKeys to Where to take the
// execKeys fast path, and compose with And/Or to intersect with other
// filters (including index-backed ones — a Where(idx.Filter(...)) ANDed with
// Where(MatchKeys(...)) intersects the candidate sets via intersectKeys
// without allocating a new closure).
//
// An empty (nil) keys argument produces a non-nil empty keys slice so the
// filter is treated as "bounded by zero keys" (matches nothing) rather than
// unbounded.
func MatchKeys[K Key, E Entry[K]](keys ...K) Filter[K, E] {
	if keys == nil {
		keys = []K{}
	}
	return Filter[K, E]{keys: keys}
}

// And returns a filter that matches when ALL children match. Composes each
// child's three dispatch paths independently:
//
//   - keys: intersected across children that have keys; children without
//     keys are unbounded and don't restrict the intersection. See
//     intersectKeys for the lazy-membership cost model.
//   - raw: AND-short-circuited across every child that has raw, so the
//     pre-decode pre-screen survives composition.
//   - eval: AND-composed at decode time, dispatching per child kind:
//     eval-set children run their eval, raw-only children invoke raw
//     against the raw bytes, keys-only children are skipped (the keys
//     intersection above already guarantees the entry is in their set
//     when execKeys fetched it).
//
// The eval and raw closures are skipped entirely when no child has them.
// For an "all children are index-backed keys-only filters" composition this
// means And allocates nothing but the intersection slice and a lazy
// membership wrapper — no closure capture, no heap dispatch.
func And[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	var f Filter[K, E]

	// eval composition: AND of every eval-set child at decode time. raw
	// children are handled by the f.raw pre-screen below (running them again
	// in eval would double-evaluate every entry); keys-only children are
	// handled by the keys intersection (entries reach decode only when in
	// every child's keys).
	hasAnyEval := false
	for _, child := range filters {
		if child.eval != nil {
			hasAnyEval = true
			break
		}
	}
	if hasAnyEval {
		f.eval = func(ctx Context, e *E, key, value []byte) (bool, error) {
			for _, child := range filters {
				if child.eval == nil {
					continue
				}
				ok, err := child.eval(ctx, e, key, value)
				if err != nil || !ok {
					return false, err
				}
			}
			return true, nil
		}
	}

	// raw composition: pre-screen short-circuits AND across every child
	// that has raw. A single raw child becomes f.raw directly (no wrapper
	// closure); 2+ get the iterating wrapper.
	var firstRaw func(key, value []byte) (bool, error)
	rawCount := 0
	for _, child := range filters {
		if child.raw != nil {
			if rawCount == 0 {
				firstRaw = child.raw
			}
			rawCount++
		}
	}
	switch rawCount {
	case 0:
		// No raw at any child: leave f.raw nil, no closure.
	case 1:
		f.raw = firstRaw
	default:
		raws := make([]func([]byte, []byte) (bool, error), 0, rawCount)
		for _, child := range filters {
			if child.raw != nil {
				raws = append(raws, child.raw)
			}
		}
		f.raw = func(key, value []byte) (bool, error) {
			for _, r := range raws {
				ok, err := r(key, value)
				if err != nil || !ok {
					return false, err
				}
			}
			return true, nil
		}
	}

	// keys composition (resolver-aware). If any child carries a deferred
	// resolver, defer the entire intersection: compose a resolver that
	// fires each resolver-child against the open tx, materializes the
	// resulting filter list, and intersects.
	if anyHasResolver(filters) {
		f.resolve = func(ctx context.Context, tx Tx) ([]K, func([]K) keyMembership[K], error) {
			materialized, err := materializeFilters[K, E](ctx, tx, filters)
			if err != nil {
				return nil, nil, err
			}
			keys, build := intersectKeys[K, E](materialized)
			return keys, build, nil
		}
		return f
	}
	var build func([]K) keyMembership[K]
	f.keys, build = intersectKeys[K, E](filters)
	if build != nil && f.keys != nil {
		f.membership = newLazyMembership(f.keys, build)
	}
	return f
}

// Or returns a filter that matches when ANY child matches. Composes the
// three dispatch paths so MatchRaw filters survive composition under Or:
//
//   - keys: union ONLY when every child has keys. A single non-keys child
//     collapses the union to unbounded; the result has nil keys and the
//     fallback eval composition runs over a full scan.
//   - raw: pre-screen survives ONLY when every child is raw-only (no eval,
//     no keys). A single non-raw child forces decode of every entry; the
//     raw children dispatch against raw bytes inside eval at decode time.
//   - eval: OR-composed at decode time, dispatching per child kind via
//     evalChild — checking keys membership for keys-set children, calling
//     raw for raw-only children, and eval for eval-set children. Each
//     child's full predicate (keys ∧ eval/raw) is evaluated and OR'd.
//
// If every child is keys-only and every child has a complete keys set, both
// f.eval and f.raw are left nil: execKeys fetches exactly the union and
// match() short-circuits to true.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	var f Filter[K, E]

	// keys composition (resolver-aware).
	if anyHasResolver(filters) {
		// Any resolver-child means the union cannot be computed at
		// compose time. Defer everything. Copy the filters slice
		// first: the Or resolver mutates resolver-carrying children
		// in place so the eval closure (which captures the same
		// slice) sees materialized keys/membership when it probes
		// them at match time.
		filters = append([]Filter[K, E](nil), filters...)
		f.resolve = func(ctx context.Context, tx Tx) ([]K, func([]K) keyMembership[K], error) {
			if err := materializeFiltersMut[K, E](ctx, tx, filters); err != nil {
				return nil, nil, err
			}
			keys, build := unionKeys[K, E](filters)
			return keys, build, nil
		}
	} else {
		var build func([]K) keyMembership[K]
		f.keys, build = unionKeys[K, E](filters)
		if build != nil && f.keys != nil {
			f.membership = newLazyMembership(f.keys, build)
		}
	}

	// raw composition: pre-screen survives only when every child is
	// raw-only. In that case the raw OR is the full predicate and we
	// don't need an eval closure.
	if len(filters) > 0 && allRawOnly(filters) {
		f.raw = func(key, value []byte) (bool, error) {
			for _, child := range filters {
				ok, err := child.raw(key, value)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
			return false, nil
		}
		return f
	}

	// If the union composed successfully AND every child was keys-only,
	// we can skip the eval closure: execKeys will fetch exactly the
	// union and match() can short-circuit to true.
	allKeysOnly := f.keys != nil
	if allKeysOnly {
		for _, child := range filters {
			if child.eval != nil || child.raw != nil {
				allKeysOnly = false
				break
			}
		}
	}
	if !allKeysOnly {
		f.eval = func(ctx Context, e *E, key, value []byte) (bool, error) {
			entryKey := (*e).GorpKey()
			for _, child := range filters {
				ok, err := evalChild(ctx, child, e, entryKey, key, value)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
			return false, nil
		}
	}
	return f
}

// Not returns a filter that inverts the child. Drops the child's keys from
// the result (inverting a key set requires the universe of all keys, which
// the filter does not have) but forwards the child's resolver so deferred
// keys are still materialized into the captured child at exec time, where
// the inverted dispatched predicate (via evalChild) reads them. When the
// child is raw-only (raw set, eval nil, keys nil, resolve nil), Not composes
// an inverted raw so the pre-decode pre-screen survives Not as well.
func Not[K Key, E Entry[K]](f Filter[K, E]) Filter[K, E] {
	out := Filter[K, E]{
		eval: func(ctx Context, e *E, key, value []byte) (bool, error) {
			entryKey := (*e).GorpKey()
			ok, err := evalChild(ctx, f, e, entryKey, key, value)
			return !ok, err
		},
	}
	if f.raw != nil && f.eval == nil && f.keys == nil && f.resolve == nil {
		// Pure raw child: the inverted raw is the full predicate, no eval
		// closure needed. Preserves the no-decode pre-screen path through
		// Not.
		raw := f.raw
		out.raw = func(key, value []byte) (bool, error) {
			ok, err := raw(key, value)
			return !ok, err
		}
		out.eval = nil
	}
	if f.resolve != nil {
		// Forward the child's resolver so that deferred keys get
		// materialized into the captured `f` value (which out.eval reads via
		// evalChild). out.resolve itself returns nil keys: inverting a
		// candidate set requires the universe, so Not is always unbounded.
		out.resolve = func(ctx context.Context, tx Tx) ([]K, func([]K) keyMembership[K], error) {
			keys, build, err := f.resolve(ctx, tx)
			if err != nil {
				return nil, nil, err
			}
			f.keys = keys
			if build != nil && keys != nil {
				f.membership = newLazyMembership(keys, build)
			} else {
				f.membership = nil
			}
			return nil, nil, nil
		}
	}
	return out
}

// evalChild evaluates a child's full predicate at decode time, dispatching
// across the (keys, raw, eval) field combination the child carries. Used by
// Or and Not where each child's full pass condition must be checked
// independently against the decoded entry. Returns true if the entry passes
// every constraint the child has set.
//
// Dispatch:
//   - If keys is set, the entry must be in keys (containsKey probe).
//   - If eval is set, run eval as the post-decode predicate.
//   - Else if raw is set, run raw against the raw bytes.
//   - Else (keys-only) the keys check above is the full predicate.
//   - Else (vacuous match) return true.
func evalChild[K Key, E Entry[K]](
	ctx Context,
	f Filter[K, E],
	e *E,
	entryKey K,
	key, value []byte,
) (bool, error) {
	if f.keys != nil && !f.containsKey(entryKey) {
		return false, nil
	}
	if f.eval != nil {
		return f.eval(ctx, e, key, value)
	}
	if f.raw != nil {
		return f.raw(key, value)
	}
	return true, nil
}

// allRawOnly reports whether every filter has raw set and no eval, keys, or
// resolve. Used by Or to decide whether the raw-pre-screen path can survive
// composition (it can only when every child is purely a raw filter).
func allRawOnly[K Key, E Entry[K]](filters []Filter[K, E]) bool {
	for _, f := range filters {
		if f.raw == nil || f.eval != nil || f.keys != nil || f.resolve != nil {
			return false
		}
	}
	return true
}

// anyHasResolver reports whether any child filter carries a deferred
// resolver. Used by And/Or to decide between the eager compose-time
// path and the deferred resolve-time path.
func anyHasResolver[K Key, E Entry[K]](filters []Filter[K, E]) bool {
	for _, child := range filters {
		if child.resolve != nil {
			return true
		}
	}
	return false
}

// materializeFilters returns a slice of filters with every resolver-child
// materialized against the open tx. Eager children are copied through
// unchanged. The returned slice is a fresh copy; the input is not
// mutated. Used by And's deferred resolver, which does not need
// in-place updates because And.eval never probes children's memberships
// directly.
func materializeFilters[K Key, E Entry[K]](
	ctx context.Context,
	tx Tx,
	filters []Filter[K, E],
) ([]Filter[K, E], error) {
	out := make([]Filter[K, E], len(filters))
	for i, child := range filters {
		if child.resolve == nil {
			out[i] = child
			continue
		}
		keys, build, err := child.resolve(ctx, tx)
		if err != nil {
			return nil, err
		}
		out[i] = child
		out[i].keys = keys
		if build != nil && keys != nil {
			out[i].membership = newLazyMembership(keys, build)
		} else {
			out[i].membership = nil
		}
	}
	return out, nil
}

// materializeFiltersMut is the in-place counterpart to materializeFilters. It
// overwrites resolver-carrying children with their materialized form so
// downstream closures that captured the same slice observe the
// post-resolution state. Used by Or whose eval closure may probe child
// membership at match time when the union degrades to an unbounded
// filter. Callers must ensure the slice is owned by the composed
// filter (copy before calling) so the mutation isn't observed by the
// original caller.
func materializeFiltersMut[K Key, E Entry[K]](
	ctx context.Context,
	tx Tx,
	filters []Filter[K, E],
) error {
	for i := range filters {
		if filters[i].resolve == nil {
			continue
		}
		keys, build, err := filters[i].resolve(ctx, tx)
		if err != nil {
			return err
		}
		filters[i].keys = keys
		if build != nil && keys != nil {
			filters[i].membership = newLazyMembership(keys, build)
		} else {
			filters[i].membership = nil
		}
	}
	return nil
}

// intersectKeys returns the intersection of every child filter's keys plus
// the build function needed to construct a lazy membership over the result.
// Children with nil keys are treated as unbounded and do not restrict the
// result. The returned slice is nil when no child has keys (the And filter
// is unbounded), and an empty non-nil slice when children have
// non-overlapping key sets.
//
// When exactly one child has keys, that child's slice is returned directly
// without copying — Filter.keys is treated as immutable by every consumer
// (Retrieve.execKeys passes it to GetMany unchanged), so sharing is safe
// and avoids a per-Where allocation on the common single-indexed-filter
// path.
//
// The multi-child path sorts a working copy of the bounded children by keys
// length and walks the LARGEST one, probing the memberships of the smaller
// ones for each candidate. This looks counterintuitive — walking the
// larger side does more comparison work — but it's the memory-optimal
// choice: walking a filter means touching its keys slice directly without
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
		if f.keys != nil {
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
		return bounded[0].keys, build
	}
	slices.SortFunc(bounded, func(a, b Filter[K, E]) int {
		return len(a.keys) - len(b.keys)
	})
	candidates := bounded[len(bounded)-1].keys
	rest := bounded[:len(bounded)-1]
	out := make([]K, 0, len(bounded[0].keys))
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

// unionKeys returns the union of every child filter's keys plus the build
// function needed to construct a lazy membership over the result. The
// returned slice is nil when any child has nil keys (the Or filter is
// unbounded). Duplicates across children are detected by probing the
// lazily-materialized membership predicates of earlier children, which is
// O(1) per probe after the one-time build.
//
// Filters are processed in ascending-keys-length order, which puts the
// largest child last. Under lazy membership, walking child i triggers
// membership builds for filters[:i] (the smaller ones) but never for
// filters[i+1:]. Sorting ascending means the largest filter's membership
// is never built — only probed against by later iterations, which don't
// happen because the largest is the last walk step. Symmetric to the
// intersectKeys flip: the largest contributes its keys directly, no map.
func unionKeys[K Key, E Entry[K]](
	filters []Filter[K, E],
) (keys []K, build func([]K) keyMembership[K]) {
	if len(filters) == 0 {
		return nil, nil
	}
	bounded := make([]Filter[K, E], 0, len(filters))
	var total int
	for _, f := range filters {
		if f.keys == nil {
			return nil, nil
		}
		bounded = append(bounded, f)
		total += len(f.keys)
		if build == nil && f.membership != nil {
			build = f.membership.build
		}
	}
	slices.SortFunc(bounded, func(a, b Filter[K, E]) int {
		return len(a.keys) - len(b.keys)
	})
	out := make([]K, 0, total)
	for i, f := range bounded {
		for _, k := range f.keys {
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
		return Filter[K, E]{eval: func(ctx Context, e *E, _, _ []byte) (bool, error) {
			return f(ctx, r, e)
		}}
	}
}

// AndBound returns a BoundFilter that matches when all provided filters
// match. Each child is bound to the same Retrieve before being composed via
// gorp.And, so the resulting filter inherits And's eval/raw/keys composition
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
