// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package iter provides utilities for working with Go iterators.
package iter

import "iter"

// Map returns an iterator that applies f to each element of the input iterator.
func Map[T any, R any](iter iter.Seq[T], f func(T) R) iter.Seq[R] {
	return func(yield func(R) bool) {
		for v := range iter {
			if !yield(f(v)) {
				return
			}
		}
	}
}
