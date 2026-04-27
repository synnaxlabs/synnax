// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Filter is a composable query filter that evaluates entries. Eval is the
// semantic predicate that decides whether an entry matches; it receives both
// the decoded entry and the raw encoded bytes so filters constructed via
// MatchRaw can run their predicate at composition time inside Or/Not. Raw is
// an optional optimization hint that, when present, runs before decoding so
// that entries the predicate rejects are skipped without ever being decoded.
// Eval and Raw must agree: if Raw rejects, Eval would have rejected too.
type Filter[K Key, E Entry[K]] struct {
	// Eval evaluates an entry by its decoded value and/or its raw encoded
	// bytes. Filters constructed via Match ignore raw; filters constructed
	// via MatchRaw ignore the decoded value.
	Eval func(ctx Context, e *E, raw []byte) (bool, error)
	// Raw evaluates the raw encoded bytes before decoding as a pre-screen
	// optimization. Nil means no pre-screen; Eval is still authoritative.
	Raw func(data []byte) (bool, error)
}

// Match wraps a closure that operates on a decoded entry as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E, _ []byte) (bool, error) {
			return f(ctx, e)
		},
	}
}

// MatchRaw wraps a raw-byte predicate as a Filter. The predicate runs as a
// pre-decode pre-screen (skipping non-matching entries entirely) and again at
// the Eval stage so the filter composes correctly inside Or and Not.
func MatchRaw[K Key, E Entry[K]](f func(data []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{
		Raw: f,
		Eval: func(_ Context, _ *E, raw []byte) (bool, error) {
			return f(raw)
		},
	}
}

// And returns a filter that matches when ALL children match. Short-circuits on the
// first false for both Raw and Eval stages independently.
func And[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	f := Filter[K, E]{
		Eval: func(ctx Context, e *E, raw []byte) (bool, error) {
			for _, f := range filters {
				if f.Eval == nil {
					continue
				}
				ok, err := f.Eval(ctx, e, raw)
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
	return f
}

// Or returns a filter that matches when ANY child matches. Short-circuits on the
// first true. Raw pre-screening is not composed for Or because a branch without
// a Raw check may still match after decoding, so we must always decode.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E, raw []byte) (bool, error) {
			for _, f := range filters {
				if f.Eval == nil {
					continue
				}
				ok, err := f.Eval(ctx, e, raw)
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
}

// Not returns a filter that inverts the child. Raw pre-screening is not composed
// for Not because inverting a raw rejection requires decoding to check the Eval
// stage.
func Not[K Key, E Entry[K]](f Filter[K, E]) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E, raw []byte) (bool, error) {
			if f.Eval == nil {
				return false, nil
			}
			ok, err := f.Eval(ctx, e, raw)
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
		return Filter[K, E]{Eval: func(ctx Context, e *E, _ []byte) (bool, error) {
			return f(ctx, r, e)
		}}
	}
}

// AndBound returns a BoundFilter that matches when all provided filters
// match. Each child is bound to the same Retrieve before being composed via
// gorp.And, so the resulting filter inherits And's Eval/Raw composition
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
