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

// Filter is a composable query filter applied to entries during a
// Retrieve. A filter can carry any subset of three independent
// constraints: a candidate primary-key set (keys), a raw-byte pre-screen
// that runs before decode (raw), and a decoded-entry predicate that
// runs after decode (eval). Compose with And, Or, and Not.
type Filter[K Key, E Entry[K]] struct {
	// eval evaluates a decoded entry against the raw key and value
	// bytes. Nil means no decoded-entry constraint.
	eval func(ctx Context, e *E, key, value []byte) (bool, error)
	// raw evaluates the encoded key and value before decoding. Returning
	// false skips the entry without allocating a decoded value. Nil
	// means no raw constraint.
	raw func(key, value []byte) (bool, error)
	// keys, if non-nil, is the candidate set of primary keys this filter
	// matches. A nil keys means the filter is unbounded.
	//
	// Composition: And intersects keys across children that have them;
	// Or unions across children only when every child has keys (a single
	// child without keys collapses Or back to unbounded); Not always
	// drops keys because inverting a key set requires the universe.
	keys []K
	// membership is a lazy O(1) mirror of keys. Nil when the filter has
	// no keys or was constructed without an IndexKey-constrained
	// builder; containsKey gates on it being non-nil.
	//
	// Lazy materialization avoids allocating an N-entry hashmap for a
	// keys slice that intersectKeys / unionKeys may end up walking
	// directly. For a 12500-key filter participating in a composition
	// where it is the walked side, the saving is ~150 KB per query.
	membership *lazyMembership[K]
	// resolve, if non-nil, computes keys and the membership build
	// function at Retrieve.Exec time. Index-backed constructors set it
	// to deliver read-your-own-writes against the open tx; And/Or
	// composition propagates resolvers when any child has one (Not
	// always drops resolve, since inverting a key set requires the
	// universe).
	//
	// A resolve return of (nil, nil, nil) means "no candidate keys" —
	// the execKeys path treats this as an empty result, NOT as
	// unbounded. An unbounded filter has no keys and no resolver.
	resolve resolveFilter[K]
}

// resolveFilter is the signature for a deferred Filter resolver. It
// returns the effective candidate keys for an indexed filter under the
// given transaction (merging committed index state with any per-tx
// delta) plus a build function for constructing an O(1) membership
// predicate over the returned keys.
type resolveFilter[K Key] func(
	ctx context.Context,
	tx Tx,
) (keys []K, build func([]K) keyMembership[K], err error)

// keyMembership is an O(1) membership predicate over a set of keys.
type keyMembership[K Key] interface {
	Contains(K) bool
}

// lazyMembership wraps a keys slice plus a deferred build function so
// the underlying keyMembership is materialized on first probe. Safe
// for concurrent use.
type lazyMembership[K Key] struct {
	once  sync.Once
	set   keyMembership[K]
	keys  []K
	build func([]K) keyMembership[K]
}

// contains reports whether k is in the underlying key set, materializing
// the membership predicate on first call.
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

// indexedKeyMembership builds an O(1) membership predicate over a set
// of comparable keys. Defined as a package-level generic (rather than
// a closure literal) so resolvers can return it without forcing a
// per-construction heap allocation.
func indexedKeyMembership[K IndexKey](keys []K) keyMembership[K] {
	return set.New(keys...)
}

// bytesIndexedKeyMembership builds an O(1) membership predicate over a
// set of []byte keys.
func bytesIndexedKeyMembership(keys [][]byte) keyMembership[[]byte] {
	m := make(bytesKeyMembership, len(keys))
	for _, b := range keys {
		m[string(b)] = struct{}{}
	}
	return m
}

// bytesKeyMembership is an O(1) membership predicate over a set of
// []byte keys, keyed internally by string conversion.
type bytesKeyMembership set.Set[string]

// Contains implements keyMembership[[]byte].
func (m bytesKeyMembership) Contains(k []byte) bool {
	_, ok := m[string(k)]
	return ok
}

// present reports whether the filter carries any active constraint. A
// zero-value Filter is treated as absent.
func (f Filter[K, E]) present() bool {
	return f.eval != nil ||
		f.raw != nil ||
		f.keys != nil ||
		f.membership != nil ||
		f.resolve != nil
}

// containsKey reports whether k is in the filter's keys set. O(1) when
// the filter carries a membership predicate; O(len(f.keys)) otherwise.
func (f Filter[K, E]) containsKey(k K) bool {
	if f.membership != nil {
		return f.membership.contains(k)
	}
	if f.keys == nil {
		return false
	}
	return linearContainsKey(f.keys, k)
}

// linearContainsKey reports whether k is in keys via O(n) scan.
// Handles []byte-shaped K via bytes.Equal since []byte is not ==
// comparable; other K kinds dispatch through interface equality.
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

// Match wraps a decoded-entry predicate as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{
		eval: func(ctx Context, e *E, _, _ []byte) (bool, error) {
			return f(ctx, e)
		},
	}
}

// MatchRaw wraps a raw-byte predicate as a Filter. The predicate runs
// before decoding and receives the pebble key and encoded value;
// returning false skips the entry without allocating a decoded value.
func MatchRaw[K Key, E Entry[K]](f func(key, value []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{raw: f}
}

// MatchKeys returns a Filter that restricts results to entries whose
// primary key is in the given set. Compose with And or Or to intersect
// or union with other filters (including index-backed ones).
//
// An empty (nil) keys argument produces a non-nil empty keys slice so
// the filter is treated as "bounded by zero keys" (matches nothing)
// rather than unbounded.
func MatchKeys[K Key, E Entry[K]](keys ...K) Filter[K, E] {
	if keys == nil {
		keys = []K{}
	}
	return Filter[K, E]{keys: keys}
}

// And returns a filter that matches when ALL children match. Each
// child's keys, raw, and eval constraints are composed independently:
// keys are intersected, raw runs as a short-circuit AND pre-screen,
// and eval runs as an AND-composed post-decode predicate. The raw
// pre-screen survives composition, so MatchRaw children retain their
// no-decode skip.
func And[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	var f Filter[K, E]

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

// Or returns a filter that matches when ANY child matches.
//
// Composition rules:
//   - keys are unioned only when every child has keys; a single
//     unbounded child collapses the union to unbounded.
//   - raw survives as a pre-screen only when every child is raw-only
//     (no eval, no keys); otherwise the raw paths dispatch inside
//     eval at decode time.
//   - eval evaluates each child's full predicate (keys ∧ eval/raw)
//     against the entry and OR's the results.
//
// When every child is keys-only with a complete keys set, both f.eval
// and f.raw are left nil and matching reduces to membership in the
// union.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	var f Filter[K, E]

	if anyHasResolver(filters) {
		// Or's eval closure may probe child membership at match
		// time after the resolver runs, so the resolver mutates
		// each child in place. Copy the slice so the original
		// caller is not affected.
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

// Not returns a filter that inverts the child. The result is always
// unbounded (its keys field is nil): inverting a candidate set
// requires the universe of all keys. When the child is raw-only, Not
// composes an inverted raw so the pre-decode skip survives.
func Not[K Key, E Entry[K]](f Filter[K, E]) Filter[K, E] {
	out := Filter[K, E]{
		eval: func(ctx Context, e *E, key, value []byte) (bool, error) {
			entryKey := (*e).GorpKey()
			ok, err := evalChild(ctx, f, e, entryKey, key, value)
			return !ok, err
		},
	}
	if f.raw != nil && f.eval == nil && f.keys == nil && f.resolve == nil {
		raw := f.raw
		out.raw = func(key, value []byte) (bool, error) {
			ok, err := raw(key, value)
			return !ok, err
		}
		out.eval = nil
	}
	if f.resolve != nil {
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

// evalChild evaluates a child's full predicate at decode time. Returns
// true if the entry passes every constraint the child carries. A child
// with no constraints is a vacuous match.
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

// allRawOnly reports whether every filter has raw set and no eval,
// keys, or resolve.
func allRawOnly[K Key, E Entry[K]](filters []Filter[K, E]) bool {
	for _, f := range filters {
		if f.raw == nil || f.eval != nil || f.keys != nil || f.resolve != nil {
			return false
		}
	}
	return true
}

// anyHasResolver reports whether any child filter carries a deferred
// resolver.
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
// mutated.
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

// materializeFiltersMut materializes every resolver-carrying child in
// place so closures that captured the same slice observe the
// post-resolution state. Callers must own the slice (copy before
// calling) so the mutation isn't observed by the original caller.
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

// intersectKeys returns the intersection of every child filter's keys
// plus the build function for a lazy membership over the result.
// Children with nil keys are treated as unbounded and do not restrict
// the intersection. Returns nil when no child has keys (unbounded);
// returns an empty non-nil slice when bounded children do not overlap.
//
// The multi-child path walks the LARGEST child's keys directly,
// probing the smaller children's lazy memberships for each candidate.
// This trades CPU for memory: it never materializes the largest
// child's membership map, which on a 12500-key filter is a ~150 KB
// per-query saving — at the price of doing the membership probes
// against the smaller side instead of the larger.
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

// unionKeys returns the union of every child filter's keys plus the
// build function for a lazy membership over the result. Returns nil
// when any child has nil keys (unbounded).
//
// Filters are processed in ascending keys-length order so the largest
// child contributes its keys directly without its membership map ever
// being built — symmetric to intersectKeys' memory-optimal walk.
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

// BoundFilter is a Filter that requires a service-defined Retrieve type
// R to produce its constraints. Use BoundFilter when the filter needs
// to read from R (e.g. a service's indexes or providers); pure
// constructors that don't need R can ignore the parameter.
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

// AndBound returns a BoundFilter that matches when all provided
// filters match. Each child is bound to the same Retrieve and the
// results are composed via And.
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
