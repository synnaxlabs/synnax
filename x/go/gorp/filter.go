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
	"context"
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
	// no keys or was constructed without a membership builder;
	// containsKey gates on it being non-nil.
	//
	// Lazy materialization avoids allocating an N-entry hashmap for a
	// keys slice that intersectKeys / unionKeys may end up walking
	// directly. For a 12500-key filter participating in a composition
	// where it is the walked side, the saving is ~150 KB per query.
	membership *lazyMembership[K]
	// resolve, if non-nil, computes the effective candidate keys (and
	// optionally an updated eval closure) at Retrieve.Exec time.
	// Index-backed constructors set it to deliver read-your-own-writes
	// against the open tx; And / Or composition propagates resolvers
	// when any child has one. Not, even though it always produces an
	// unbounded result, still carries a resolver when its child is
	// resolver-backed so the per-call eval can close over the
	// materialized child without sharing mutable state across Execs.
	//
	// A resolved value with nil keys means "no candidate keys" — the
	// execKeys path treats this as an empty result, NOT as unbounded.
	// An unbounded filter has no keys and no resolver.
	resolve resolveFilter[K, E]
}

// evalFunc is the signature of Filter.eval. Kept as a named alias so
// resolved (which threads an optional eval through the resolve
// protocol) can reference it without restating the parameter list.
type evalFunc[K Key, E Entry[K]] func(ctx Context, e *E, key, value []byte) (bool, error)

// resolved is the payload returned by a Filter.resolve call. keys and
// build follow the same semantics they did when resolve returned them
// directly. eval, when non-nil, replaces the construction-time
// Filter.eval for the duration of this Exec. The eval field lets
// composers like Or and Not close over per-call materialized state
// rather than mutating shared captured state.
type resolved[K Key, E Entry[K]] struct {
	keys  []K
	build func([]K) keyMembership[K]
	eval  evalFunc[K, E]
}

// resolveFilter is the signature for a deferred Filter resolver. It
// produces the per-Exec resolution payload by reading committed index
// state and merging any per-tx delta staged against tx.
type resolveFilter[K Key, E Entry[K]] func(
	ctx context.Context,
	tx Tx,
) (resolved[K, E], error)

// keyMembership is an O(1) membership predicate over a set of keys.
type keyMembership[K Key] interface {
	Contains(K) bool
}

// lazyMembership wraps a keys slice plus a deferred build function so
// the underlying keyMembership is materialized on first probe. Safe
// for concurrent use.
type lazyMembership[K Key] struct {
	// once guards a single build call across concurrent probes.
	once sync.Once
	// set is the materialized membership predicate; nil until once fires.
	set keyMembership[K]
	// keys is the candidate set passed to build.
	keys []K
	// build constructs the keyMembership from keys on first probe.
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
func indexedKeyMembership[K Key](keys []K) keyMembership[K] {
	return set.New(keys...)
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
	return slices.Contains(f.keys, k)
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
			entryKey := (*e).GorpKey()
			for _, child := range filters {
				if child.keys != nil && !child.containsKey(entryKey) {
					return false, nil
				}
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
		f.resolve = func(ctx context.Context, tx Tx) (resolved[K, E], error) {
			materialized, err := materializeFilters[K, E](ctx, tx, filters)
			if err != nil {
				return resolved[K, E]{}, err
			}
			keys, build := intersectKeys[K, E](materialized)
			return resolved[K, E]{keys: keys, build: build}, nil
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
//     against the entry and ORs the results.
//
// When every child is keys-only with a complete keys set, both f.eval
// and f.raw are left nil and matching reduces to membership in the
// union.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	var f Filter[K, E]

	if anyHasResolver(filters) {
		// Resolver path: materialize the children fresh per Exec so
		// concurrent Execs never share mutable child state. The
		// resolver also produces the per-Exec eval closure, which
		// overrides this construction-time fallback on the happy path.
		// The fallback survives when resolveFilter swallows
		// ErrIndexInvalid: keys / membership get cleared but the eval
		// stays installed so the sequential-scan path still applies
		// each child's predicate. Closing over the unmaterialized
		// children is safe here because evalChild skips the keys
		// check when keys is nil and dispatches to each child's
		// scan-fallback eval (set by LookupIndex.Filter /
		// SortedIndex.Filter).
		f.eval = orEval[K, E](filters)
		f.resolve = func(ctx context.Context, tx Tx) (resolved[K, E], error) {
			materialized, err := materializeFilters[K, E](ctx, tx, filters)
			if err != nil {
				return resolved[K, E]{}, err
			}
			keys, build := unionKeys[K, E](materialized)
			return resolved[K, E]{
				keys:  keys,
				build: build,
				eval:  orEval[K, E](materialized),
			}, nil
		}
		return f
	}

	var build func([]K) keyMembership[K]
	f.keys, build = unionKeys[K, E](filters)
	if build != nil && f.keys != nil {
		f.membership = newLazyMembership(f.keys, build)
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
		f.eval = orEval[K, E](filters)
	}
	return f
}

// orEval returns the per-call eval closure used by Or composition. It
// runs each child's full predicate and ORs the results, short-circuiting
// on the first match. Defined as a package-level generic so resolver-
// path Or compositions can close over freshly materialized children
// without smuggling them through Filter struct fields.
func orEval[K Key, E Entry[K]](filters []Filter[K, E]) evalFunc[K, E] {
	return func(ctx Context, e *E, key, value []byte) (bool, error) {
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

// Not returns a filter that inverts the child. The result is always
// unbounded (its keys field is nil): inverting a candidate set
// requires the universe of all keys. When the child is raw-only, Not
// composes an inverted raw so the pre-decode skip survives.
func Not[K Key, E Entry[K]](f Filter[K, E]) Filter[K, E] {
	// Raw-only child: inverting the raw predicate preserves the
	// pre-decode skip. No eval, no resolve needed.
	if f.raw != nil && f.eval == nil && f.keys == nil && f.resolve == nil {
		raw := f.raw
		return Filter[K, E]{
			raw: func(key, value []byte) (bool, error) {
				ok, err := raw(key, value)
				return !ok, err
			},
		}
	}
	// Resolver-backed child: defer eval construction until Exec so it
	// can close over a fresh materialized copy of f. Sharing the
	// captured f across Execs would race resolve's writes against
	// concurrent eval reads. The construction-time eval is the
	// sequential-scan fallback used when resolveFilter swallows
	// ErrIndexInvalid: f.keys / f.membership remain nil, so evalChild
	// skips the keys check and dispatches to f's scan-fallback eval.
	if f.resolve != nil {
		return Filter[K, E]{
			eval: notEval[K, E](f),
			resolve: func(ctx context.Context, tx Tx) (resolved[K, E], error) {
				res, err := f.resolve(ctx, tx)
				if err != nil {
					return resolved[K, E]{}, err
				}
				m := f
				m.keys = res.keys
				if res.build != nil && res.keys != nil {
					m.membership = newLazyMembership(res.keys, res.build)
				} else {
					m.membership = nil
				}
				if res.eval != nil {
					m.eval = res.eval
				}
				return resolved[K, E]{eval: notEval[K, E](m)}, nil
			},
		}
	}
	// Eager child (no resolver). Capturing f directly is safe — there
	// is no resolve writing to it.
	return Filter[K, E]{eval: notEval[K, E](f)}
}

// notEval returns the per-call eval closure used by Not composition.
// Defined as a package-level generic so Not's resolver path can close
// over a freshly materialized child without smuggling it through
// Filter struct fields.
func notEval[K Key, E Entry[K]](f Filter[K, E]) evalFunc[K, E] {
	return func(ctx Context, e *E, key, value []byte) (bool, error) {
		entryKey := (*e).GorpKey()
		ok, err := evalChild(ctx, f, e, entryKey, key, value)
		return !ok, err
	}
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
// mutated. Resolver-returned eval overrides are installed on the copy
// so Or / Not's per-call eval closures see the right predicate for
// each child.
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
		res, err := child.resolve(ctx, tx)
		if err != nil {
			return nil, err
		}
		out[i] = child
		out[i].keys = res.keys
		if res.build != nil && res.keys != nil {
			out[i].membership = newLazyMembership(res.keys, res.build)
		} else {
			out[i].membership = nil
		}
		if res.eval != nil {
			out[i].eval = res.eval
		}
	}
	return out, nil
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
func AndBound[R any, K Key, E Entry[K]](
	fs ...BoundFilter[R, K, E],
) BoundFilter[R, K, E] {
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
func OrBound[R any, K Key, E Entry[K]](
	fs ...BoundFilter[R, K, E],
) BoundFilter[R, K, E] {
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
