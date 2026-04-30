// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Filter is a composable query filter built up via Match, MatchRaw, MatchKeys,
// And, Or, and Not.
type Filter[K Key, E Entry[K]] struct {
	eval func(ctx Context, e *E, raw []byte) (bool, error)
	raw  func(data []byte) (bool, error)
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

// MatchRaw wraps a raw-byte predicate as a Filter. The predicate runs on the
// raw encoded bytes of an entry, before decoding.
func MatchRaw[K Key, E Entry[K]](f func(data []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{raw: f}
}

// MatchKeys returns a filter that restricts results to the given primary keys.
// An empty key set produces a filter that matches no entries.
func MatchKeys[K Key, E Entry[K]](keys ...K) Filter[K, E] {
	if keys == nil {
		keys = []K{}
	}
	return Filter[K, E]{keys: keys}
}

// And returns a filter that matches when ALL children match. Short-circuits on
// the first false.
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
// the first true.
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

// Not returns a filter that inverts the child.
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

func allRawOnly[K Key, E Entry[K]](filters []Filter[K, E]) bool {
	for _, f := range filters {
		if f.raw == nil || f.eval != nil {
			return false
		}
	}
	return true
}

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

// BoundFilter is a Filter that needs access to a caller-supplied value R when
// it is built. R is typically a service-specific Retrieve type that exposes
// indexes or other state the filter depends on.
type BoundFilter[R any, K Key, E Entry[K]] func(r R) Filter[K, E]

// MatchBound wraps a predicate that needs access to R as a BoundFilter.
func MatchBound[R any, K Key, E Entry[K]](
	f func(ctx Context, r R, e *E) (bool, error),
) BoundFilter[R, K, E] {
	return func(r R) Filter[K, E] {
		return Filter[K, E]{eval: func(ctx Context, e *E, _ []byte) (bool, error) {
			return f(ctx, r, e)
		}}
	}
}

// AndBound returns a BoundFilter that matches when ALL children match.
func AndBound[R any, K Key, E Entry[K]](fs ...BoundFilter[R, K, E]) BoundFilter[R, K, E] {
	return func(r R) Filter[K, E] {
		inner := make([]Filter[K, E], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return And(inner...)
	}
}

// OrBound returns a BoundFilter that matches when ANY child matches.
func OrBound[R any, K Key, E Entry[K]](fs ...BoundFilter[R, K, E]) BoundFilter[R, K, E] {
	return func(r R) Filter[K, E] {
		inner := make([]Filter[K, E], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return Or(inner...)
	}
}

// NotBound returns a BoundFilter that inverts the provided filter.
func NotBound[R any, K Key, E Entry[K]](f BoundFilter[R, K, E]) BoundFilter[R, K, E] {
	return func(r R) Filter[K, E] {
		return Not(f(r))
	}
}
