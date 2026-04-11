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
// closures for key-based pre-decode rejection, raw-byte pre-screening, and
// decoded-entry evaluation. Retrieve applies the stages in order: Key → Raw → Eval.
// Any nil stage is skipped.
type Filter[K Key, E Entry[K]] struct {
	// Eval evaluates a decoded entry. Nil means no decoded-entry constraint.
	Eval func(ctx Context, e *E) (bool, error)
	// Raw evaluates the raw encoded bytes before decoding. Returning false skips
	// the entry without allocating a decoded value. Nil means no raw constraint.
	Raw func(data []byte) (bool, error)
	// Key evaluates the primary key before decoding. Used by indexed filters that
	// can reject entries based on a precomputed candidate set. Returning false
	// skips the entry without reading its value bytes. Nil means no key constraint.
	Key func(k K) (bool, error)
}

// Match wraps a bare closure as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Eval: f}
}

// MatchRaw wraps a raw-byte predicate as a Filter. The predicate runs before
// decoding; returning false skips the entry entirely.
func MatchRaw[K Key, E Entry[K]](f func(data []byte) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Raw: f}
}

// MatchKey wraps a primary-key predicate as a Filter. The predicate runs before
// the value is read; returning false skips the entry without decoding.
func MatchKey[K Key, E Entry[K]](f func(k K) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Key: f}
}

// And returns a filter that matches when ALL children match. Short-circuits on the
// first false for each stage independently.
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
	var keys []func(K) (bool, error)
	for _, child := range filters {
		if child.Key != nil {
			keys = append(keys, child.Key)
		}
	}
	if len(keys) > 0 {
		f.Key = func(k K) (bool, error) {
			for _, kf := range keys {
				ok, err := kf(k)
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
// a Raw check may still match after decoding, so we must always decode. Key
// pre-checks follow the same rule: Or only composes a key stage when every branch
// provides one, so a branch without a key constraint can still admit the entry.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	f := Filter[K, E]{
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
	allHaveKey := len(filters) > 0
	for _, child := range filters {
		if child.Key == nil {
			allHaveKey = false
			break
		}
	}
	if allHaveKey {
		keyFns := make([]func(K) (bool, error), len(filters))
		for i, child := range filters {
			keyFns[i] = child.Key
		}
		f.Key = func(k K) (bool, error) {
			for _, kf := range keyFns {
				ok, err := kf(k)
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

// Not returns a filter that inverts the child. Raw pre-screening is not composed
// for Not because inverting a raw rejection requires decoding to check the Eval
// stage. Key pre-checks follow the same rule: inverting a key rejection still
// requires evaluating the rest of the filter, so the key stage is dropped.
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
