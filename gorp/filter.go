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

	"github.com/synnaxlabs/gorp/internal/keyset"
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
	// keys is the candidate set of primary keys this filter matches. The zero value
	// is unbounded.
	//
	// Composition: And intersects keys across children that have them; Or unions
	// across children only when every child has keys (a single child without keys
	// collapses Or back to unbounded); Not always drops keys because inverting a key
	// set requires the universe.
	keys keyset.Set[K]
	// resolve, if non-nil, computes the effective candidate keys (and
	// optionally an updated eval closure) at Retrieve.Exec time.
	// Index-backed constructors set it to deliver read-your-own-writes
	// against the open tx; And / Or composition propagates resolvers
	// when any child has one. Not, even though it always produces an
	// unbounded result, still carries a resolver when its child is
	// resolver-backed so the per-call eval can close over the
	// materialized child without sharing mutable state across Execs.
	resolve resolveFilter[K, E]
}

// evalFunc is the signature of Filter.eval. Kept as a named alias so
// resolved (which threads an optional eval through the resolve
// protocol) can reference it without restating the parameter list.
type evalFunc[K Key, E Entry[K]] func(ctx Context, e *E, key, value []byte) (bool, error)

// resolved is the payload returned by a Filter.resolve call. keys replaces the
// filter's keys for the duration of this Exec. eval, when non-nil, replaces the
// construction-time Filter.eval for the duration of this Exec. The eval field lets
// composers like Or and Not close over per-call materialized state rather than
// mutating shared captured state.
type resolved[K Key, E Entry[K]] struct {
	keys keyset.Set[K]
	eval evalFunc[K, E]
}

// resolveFilter is the signature for a deferred Filter resolver. It
// produces the per-Exec resolution payload by reading committed index
// state and merging any per-tx delta staged against tx.
type resolveFilter[K Key, E Entry[K]] func(
	ctx context.Context,
	tx Tx,
) (resolved[K, E], error)

// present reports whether the filter carries any active constraint. A
// zero-value Filter is treated as absent.
func (f Filter[K, E]) present() bool {
	return f.eval != nil || f.raw != nil || f.keys.Bounded() || f.resolve != nil
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
// An empty keys argument produces a filter bounded by zero keys, which matches
// nothing.
func MatchKeys[K Key, E Entry[K]](keys ...K) Filter[K, E] {
	return Filter[K, E]{keys: keyset.Of(keys)}
}

// MatchKeysFunc returns a Filter that restricts results to the keys that f returns.
// Exec, Count, and Exists each call f with the open transaction, so f can read state
// the transaction staged. A key with no entry is a non-match, while a bare MatchKeys
// fails Exec with query.ErrNotFound.
func MatchKeysFunc[K Key, E Entry[K]](f func(ctx Context) ([]K, error)) Filter[K, E] {
	return Filter[K, E]{
		resolve: func(ctx context.Context, tx Tx) (resolved[K, E], error) {
			keys, err := f(Context{Context: ctx, Tx: tx})
			if err != nil {
				return resolved[K, E]{}, err
			}
			return resolved[K, E]{keys: keyset.Hashed(keys)}, nil
		},
	}
}

// And returns a filter that matches when ALL children match. Each
// child's keys, raw, and eval constraints are composed independently:
// keys are intersected, raw runs as a short-circuit AND pre-screen,
// and eval runs as an AND-composed post-decode predicate. The raw
// pre-screen survives composition, so MatchRaw children retain their
// no-decode skip.
func And[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	var f Filter[K, E]
	f.eval = andEval(filters)

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
			materialized, err := materializeFilters(ctx, tx, filters)
			if err != nil {
				return resolved[K, E]{}, err
			}
			return resolved[K, E]{
				keys: keyset.Intersect(filterKeys(materialized)...),
				eval: andEval(materialized),
			}, nil
		}
		return f
	}
	f.keys = keyset.Intersect(filterKeys(filters)...)
	return f
}

// andEval returns the eval closure used by And composition, or nil when no child has
// an eval. It does not check the keys of each child, because the keys of the And are
// their intersection.
func andEval[K Key, E Entry[K]](filters []Filter[K, E]) evalFunc[K, E] {
	if !slices.ContainsFunc(
		filters,
		func(f Filter[K, E]) bool { return f.eval != nil },
	) {
		return nil
	}
	return func(ctx Context, e *E, key, value []byte) (bool, error) {
		for _, child := range filters {
			if child.eval == nil {
				continue
			}
			if ok, err := child.eval(ctx, e, key, value); err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	}
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
		// Materialize the children fresh per Exec so concurrent Execs never share
		// mutable child state.
		f.resolve = func(ctx context.Context, tx Tx) (resolved[K, E], error) {
			materialized, err := materializeFilters(ctx, tx, filters)
			if err != nil {
				return resolved[K, E]{}, err
			}
			return resolved[K, E]{
				keys: keyset.Union(filterKeys(materialized)...),
				eval: orEval(materialized),
			}, nil
		}
		return f
	}

	f.keys = keyset.Union(filterKeys(filters)...)

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
	allKeysOnly := f.keys.Bounded()
	if allKeysOnly {
		for _, child := range filters {
			if child.eval != nil || child.raw != nil {
				allKeysOnly = false
				break
			}
		}
	}
	if !allKeysOnly {
		f.eval = orEval(filters)
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
	if f.raw != nil && f.eval == nil && !f.keys.Bounded() && f.resolve == nil {
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
	// concurrent eval reads.
	if f.resolve != nil {
		return Filter[K, E]{
			resolve: func(ctx context.Context, tx Tx) (resolved[K, E], error) {
				res, err := f.resolve(ctx, tx)
				if err != nil {
					return resolved[K, E]{}, err
				}
				m := f
				m.keys = res.keys
				if res.eval != nil {
					m.eval = res.eval
				}
				return resolved[K, E]{eval: notEval(m)}, nil
			},
		}
	}
	// Eager child (no resolver). Capturing f directly is safe — there
	// is no resolve writing to it.
	return Filter[K, E]{eval: notEval(f)}
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
	if !f.keys.Contains(entryKey) {
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
		if f.raw == nil || f.eval != nil || f.keys.Bounded() || f.resolve != nil {
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
		if res.eval != nil {
			out[i].eval = res.eval
		}
	}
	return out, nil
}

// filterKeys returns the candidate key set of each filter.
func filterKeys[K Key, E Entry[K]](filters []Filter[K, E]) []keyset.Set[K] {
	sets := make([]keyset.Set[K], len(filters))
	for i, f := range filters {
		sets[i] = f.keys
	}
	return sets
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
