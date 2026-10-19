// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package set

import "slices"

// linearMax is the most values Contains scans linearly. Measured crossover against a
// map lookup is ~5 values for strings and ~20 for uint32.
const linearMax = 8

// Contains returns a membership check for vals. It scans vals linearly when there are
// at most a few of them and builds a Set otherwise. vals must not be modified while
// the check is in use.
func Contains[T comparable](vals ...T) func(T) bool {
	if len(vals) <= linearMax {
		return func(v T) bool { return slices.Contains(vals, v) }
	}
	return New(vals...).Contains
}
