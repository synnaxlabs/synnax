// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Filter is a composable query filter that evaluates entries. The struct carries
// closures for both raw-byte pre-screening and decoded-entry evaluation.
type Filter[K Key, E Entry[K]] struct {
	// Eval evaluates a decoded entry. Nil means no decoded-entry constraint.
	Eval func(ctx Context, e *E) (bool, error)
	// Raw evaluates the raw encoded bytes before decoding. Returning false skips
	// the entry without allocating a decoded value. Nil means no raw constraint.
	Raw func(data []byte) bool
}

// Match wraps a bare closure as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Eval: f}
}

// MatchRaw wraps a raw-byte predicate as a Filter. The predicate runs before
// decoding; returning false skips the entry entirely.
func MatchRaw[K Key, E Entry[K]](f func(data []byte) bool) Filter[K, E] {
	return Filter[K, E]{Raw: f}
}

// And returns a filter that matches when ALL children match. Short-circuits on the
// first false for both Raw and Eval stages independently.
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
	var raws []func([]byte) bool
	for _, child := range filters {
		if child.Raw != nil {
			raws = append(raws, child.Raw)
		}
	}
	if len(raws) > 0 {
		f.Raw = func(data []byte) bool {
			for _, r := range raws {
				if !r(data) {
					return false
				}
			}
			return true
		}
	}
	return f
}

// Or returns a filter that matches when ANY child matches. Short-circuits on the
// first true. Raw pre-screening is not composed for Or because a branch without
// a Raw check may still match after decoding, so we must always decode.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E) (bool, error) {
			for _, f := range filters {
				if f.Eval == nil {
					continue
				}
				ok, err := f.Eval(ctx, e)
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
		Eval: func(ctx Context, e *E) (bool, error) {
			if f.Eval == nil {
				return false, nil
			}
			ok, err := f.Eval(ctx, e)
			return !ok, err
		},
	}
}
