// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Filter is a composable query filter. eval matches against the decoded entry;
// raw, when set, pre-screens entries by raw bytes before decoding. keys, when
// set, restricts the filter to a bounded set of primary keys and lets
// Retrieve.Exec dispatch to the multi-get fast path instead of a full scan.
type Filter[K Key, E Entry[K]] struct {
	// eval matches against the decoded entry; raw bytes are passed too for
	// filters that need them. Nil falls back to raw in combinators.
	eval func(ctx Context, e *E, raw []byte) (bool, error)
	// raw pre-screens by raw bytes before decoding. Optional.
	raw func(data []byte) (bool, error)
	// keys, when non-nil, bounds the filter to the given set of primary keys.
	// nil means unbounded (no key restriction). When the resolved query
	// filter has keys != nil, Retrieve.Exec uses a multi-get fast path
	// instead of iterating the full table.
	keys []K
}

// Match wraps a decoded-entry predicate as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{
		eval: func(ctx Context, e *E, _ []byte) (bool, error) {
			return f(ctx, e)
		},
	}
}

// MatchRaw wraps a raw-byte predicate as a Filter. The result has Raw set and
// Eval nil so it pre-screens before decoding. Composition rules:
//
//   - Under And, Raw is preserved alongside any sibling's Raw, so pre-screening
//     still applies (sibling Evals run after decode as usual).
//   - Under Or, Raw is preserved only when every sibling is also raw-only; if
//     any sibling has Eval, the whole Or has to decode every entry to evaluate
//     that sibling, and the MatchRaw branch is evaluated at decode time using
//     the raw bytes carried alongside the decoded entry.
//   - Under Not, Raw is inverted and pre-screening still applies.
func MatchRaw[K Key, E Entry[K]](f func(data []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{raw: f}
}

// MatchKeys returns a filter that restricts results to the given primary keys.
// The returned filter's Keys slice owns its own storage, so callers may mutate
// the input without affecting the filter. An empty (non-nil) slice produces a
// bounded filter that matches no entries; nil keys would be unbounded, so the
// constructor always allocates at least an empty slice.
func MatchKeys[K Key, E Entry[K]](keys ...K) Filter[K, E] {
	return Filter[K, E]{keys: keys}
}

// And returns a filter that matches when ALL children match. Short-circuits on the
// first false for both Raw and Eval stages independently. Keys propagate when
// exactly one child is bounded: the result inherits that child's Keys and the
// other children's Eval / Raw become post-filters on the multi-get fast path.
// When two or more children are bounded, the result drops Keys and falls back
// to a full scan with the AND'd Eval evaluated per row — composing two bounded
// children at the same level is rare in practice (callers concatenate keys at
// the call site instead), so the optimization isn't worth the cost of
// constraining K to comparable here.
func And[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	if len(filters) == 1 {
		return filters[0]
	}
	f := Filter[K, E]{
		eval: func(ctx Context, e *E, raw []byte) (bool, error) {
			for _, f := range filters {
				if f.eval == nil {
					continue
				}
				ok, err := f.eval(ctx, e, raw)
				if err != nil || !ok {
					return false, err
				}
			}
			return true, nil
		},
	}
	var raws []func([]byte) (bool, error)
	for _, child := range filters {
		if child.raw != nil {
			raws = append(raws, child.raw)
		}
	}
	if len(raws) > 0 {
		f.raw = func(data []byte) (bool, error) {
			for _, r := range raws {
				ok, err := r(data)
				if err != nil || !ok {
					return false, err
				}
			}
			return true, nil
		}
	}
	f.keys = singleBoundedKeys(filters)
	return f
}

// Or returns a filter that matches when ANY child matches. Short-circuits on
// the first true. Raw pre-screening is composed only when every child is
// raw-only (Raw set, Eval nil); a single non-raw child forces a full decode
// of every entry, with raw-only children evaluated at decode time. Or always
// drops Keys: unioning bounded children would require equality on K (i.e.
// constraining K to comparable), and the case where every child is bounded
// is rare enough that the simpler full-scan fallback is preferable. When
// index-backed filters land and resolve to bounded Keys at query time, the
// resolver itself can union with comparable K it owns.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	if len(filters) == 1 {
		return filters[0]
	}
	f := Filter[K, E]{
		eval: func(ctx Context, e *E, raw []byte) (bool, error) {
			for _, f := range filters {
				ok, err := evalChild(ctx, f, e, raw)
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
	if len(filters) == 0 || !allRawOnly(filters) {
		return f
	}
	f.raw = func(data []byte) (bool, error) {
		for _, child := range filters {
			ok, err := child.raw(data)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	f.eval = nil
	return f
}

// Not returns a filter that inverts the child. When the child is raw-only,
// Not composes an inverted Raw so the result still pre-screens before
// decoding; otherwise Not falls back to evaluating the child at decode time.
// Not always drops Keys: inverting a key set requires the universe of all
// keys, which the filter doesn't have, so the inverted filter falls through
// to a full scan with the negated Eval evaluated per row. A child with only
// Keys (no Eval / Raw) gets a synthesized Eval that checks key membership
// before negation, so Not(MatchKeys(...)) still produces correct results.
func Not[K Key, E Entry[K]](f Filter[K, E]) Filter[K, E] {
	out := Filter[K, E]{
		eval: func(ctx Context, e *E, raw []byte) (bool, error) {
			ok, err := evalChild(ctx, f, e, raw)
			return !ok, err
		},
	}
	if f.raw != nil && f.eval == nil {
		raw := f.raw
		out.raw = func(data []byte) (bool, error) {
			ok, err := raw(data)
			return !ok, err
		}
		out.eval = nil
	}
	return out
}

// allRawOnly reports whether every filter has Raw set and Eval nil.
func allRawOnly[K Key, E Entry[K]](filters []Filter[K, E]) bool {
	for _, f := range filters {
		if f.raw == nil || f.eval != nil {
			return false
		}
	}
	return true
}

// singleBoundedKeys returns the Keys of the unique bounded child if exactly
// one child has Keys != nil; returns nil otherwise. The result is the
// caller's storage — callers must not mutate it.
func singleBoundedKeys[K Key, E Entry[K]](filters []Filter[K, E]) []K {
	var (
		out   []K
		count = 0
	)
	for _, f := range filters {
		if f.keys == nil {
			continue
		}
		count++
		if count > 1 {
			return nil
		}
		out = f.keys
	}
	return out
}

// evalChild runs a child's Eval if set, else its Raw against raw bytes. A
// child with neither matches vacuously.
func evalChild[K Key, E Entry[K]](
	ctx Context,
	f Filter[K, E],
	e *E,
	raw []byte,
) (bool, error) {
	if f.eval != nil {
		return f.eval(ctx, e, raw)
	}
	if f.raw != nil {
		return f.raw(raw)
	}
	return true, nil
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
		return Filter[K, E]{eval: func(ctx Context, e *E, _ []byte) (bool, error) {
			return f(ctx, r, e)
		}}
	}
}

// AndBound returns a BoundFilter that matches when all provided filters
// match. Each child is bound to the same Retrieve before being composed via
// gorp.And, so the resulting filter inherits And's Eval/Raw composition semantics.
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
