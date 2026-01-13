// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package math provides extended math functionality.
package math

import "github.com/synnaxlabs/x/types"

const (
	// MaxUint20 is the maximum value of a 20-bit unsigned integer.
	MaxUint20 types.Uint20 = 2 ^ 20 - 1
	// MaxUint12 is the maximum value of a 12-bit unsigned integer.
	MaxUint12 types.Uint12 = 2 ^ 12 - 1
)

// IntPow efficiently returns the result of the operation x^n for two numbers. This
// implementation uses exponentiation by squaring. IntPow panics if x is zero and n is
// negative. See
// https://en.wikipedia.org/wiki/Exponentiation_by_squaring#With_constant_auxiliary_memory
func IntPow[T types.Numeric](x T, n int) T {
	if n < 0 {
		if x == 0 {
			panic("[math.IntPow]: Cannot raise zero to a negative power")
		}
		x = 1 / x
		n *= -1
	} else if n == 0 {
		return 1
	}
	y := T(1)
	for n > 1 {
		if n%2 == 1 {
			y *= x
			n--
		}
		x *= x
		n /= 2
	}
	return x * y
}
