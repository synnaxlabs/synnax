// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Filter is a composable query filter that evaluates entries. The struct carries an
// Eval closure for in-process evaluation. The struct (rather than a bare function type)
// allows future optimization capabilities (raw-byte evaluation, index hints) to be
// added without changing the API.
type Filter[K Key, E Entry[K]] struct {
	Eval func(ctx Context, e *E) (bool, error)
}

// Match wraps a bare closure as a Filter.
func Match[K Key, E Entry[K]](f func(ctx Context, e *E) (bool, error)) Filter[K, E] {
	return Filter[K, E]{Eval: f}
}

// And returns a filter that matches when ALL children match. Short-circuits on the
// first false.
func And[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E) (bool, error) {
			for _, f := range filters {
				ok, err := f.Eval(ctx, e)
				if err != nil || !ok {
					return false, err
				}
			}
			return true, nil
		},
	}
}

// Or returns a filter that matches when ANY child matches. Short-circuits on the
// first true.
func Or[K Key, E Entry[K]](filters ...Filter[K, E]) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E) (bool, error) {
			for _, f := range filters {
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

// Not returns a filter that inverts the child.
func Not[K Key, E Entry[K]](f Filter[K, E]) Filter[K, E] {
	return Filter[K, E]{
		Eval: func(ctx Context, e *E) (bool, error) {
			ok, err := f.Eval(ctx, e)
			return !ok, err
		},
	}
}
