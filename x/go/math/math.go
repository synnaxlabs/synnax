// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// math is a package that extends math functionality.

package math

// IntPow returns x^n for two integers x and n. IntPow requires n to be a
// nonnegative integer.
func IntPow(x, n int) int {
	return exp_by_squaring(1, x, n)
}

// exp_by_squaring returns y*x^n for integers y, x, and n. This requires n to be
// a nonnegative integer.
func exp_by_squaring(y, x, n int) int {
	if n == 0 {
		return y
	}
	if n%2 == 0 {
		return exp_by_squaring(y, x*x, n/2)
	}
	return exp_by_squaring(x*y, x*x, (n-1)/2)
}
