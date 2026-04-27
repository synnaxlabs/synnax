// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Filter is a composable query filter. Eval matches against the decoded entry;
// Raw, when set, pre-screens entries by raw bytes before decoding.
type Filter[K Key, E Entry[K]] struct {
	// Eval matches against the decoded entry; raw bytes are passed too for
	// filters that need them. Nil falls back to Raw in combinators.
	Eval func(ctx Context, e *E, raw []byte) (bool, error)
	// Raw pre-screens by raw bytes before decoding. Optional.
	Raw func(data []byte) (bool, error)
}

// Match wraps a decoded-entry predicate as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E, _ []byte) (bool, error) {
			return f(ctx, e)
		},
	}
}

// MatchRaw wraps a raw-byte predicate as a Filter. Under Where or And it
// pre-screens before decoding; under Or or Not it is called at Eval time so
// composition is correct, at the cost of decoding every entry.
func MatchRaw[K Key, E Entry[K]](f func(data []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Raw: f}
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

// Or returns a filter that matches when ANY child matches. Short-circuits on
// the first true. Raw pre-screening is composed only when every child is
// raw-only (Raw set, Eval nil); a single non-raw child forces a full decode
// of every entry, with raw-only children evaluated at decode time.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	f := Filter[K, E]{
		Eval: func(ctx Context, e *E, raw []byte) (bool, error) {
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
	f.Raw = func(data []byte) (bool, error) {
		for _, child := range filters {
			ok, err := child.Raw(data)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	f.Eval = nil
	return f
}

// Not returns a filter that inverts the child. When the child is raw-only,
// Not composes an inverted Raw so the result still pre-screens before
// decoding; otherwise Not falls back to evaluating the child at decode time.
func Not[K Key, E Entry[K]](f Filter[K, E]) Filter[K, E] {
	out := Filter[K, E]{
		Eval: func(ctx Context, e *E, raw []byte) (bool, error) {
			ok, err := evalChild(ctx, f, e, raw)
			return !ok, err
		},
	}
	if f.Raw != nil && f.Eval == nil {
		raw := f.Raw
		out.Raw = func(data []byte) (bool, error) {
			ok, err := raw(data)
			return !ok, err
		}
		out.Eval = nil
	}
	return out
}

// allRawOnly reports whether every filter has Raw set and Eval nil.
func allRawOnly[K Key, E Entry[K]](filters []Filter[K, E]) bool {
	for _, f := range filters {
		if f.Raw == nil || f.Eval != nil {
			return false
		}
	}
	return true
}

// evalChild runs a child's Eval if set, else its Raw against raw bytes. A
// child with neither matches vacuously.
func evalChild[K Key, E Entry[K]](
	ctx Context,
	f Filter[K, E],
	e *E,
	raw []byte,
) (bool, error) {
	if f.Eval != nil {
		return f.Eval(ctx, e, raw)
	}
	if f.Raw != nil {
		return f.Raw(raw)
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
