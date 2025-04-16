// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math

import "github.com/synnaxlabs/x/types"

const MaxUint20 types.Uint20 = 2<<19 - 1
const MaxUint12 types.Uint12 = 2<<11 - 1

// IntPow returns x^n for two integers x and n. IntPow requires n to be a
// non-negative integer.
func IntPow(x, n int) int {
	return expBySquaring(1, x, n)
}

// expBySquaring returns y*x^n for integers y, x, and n. This requires n to be
// a non-negative integer.
func expBySquaring(y, x, n int) int {
	if n == 0 {
		return y
	}
	if n%2 == 0 {
		return expBySquaring(y, x*x, n/2)
	}
	return expBySquaring(x*y, x*x, (n-1)/2)
}
