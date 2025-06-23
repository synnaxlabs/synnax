// Copyright 2025 Synnax Labs, Inc.
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
	MaxUint20 types.Uint20 = 2<<19 - 1
	// MaxUint12 is the maximum value of a 12-bit unsigned integer.
	MaxUint12 types.Uint12 = 2<<11 - 1
)

// IntPow efficiently returns the result of the operation base^exponent for two
// numbers. IntPow panics if exponent is negative. This implementation uses
// exponentiation by squaring. See
// https://en.wikipedia.org/wiki/Exponentiation_by_squaring
func IntPow(base, exponent int) int {
	if exponent < 0 {
		panic("[math] IntPow: negative exponent")
	}
	y := 1
	for exponent > 0 {
		if exponent%2 == 1 {
			y *= base
		}
		base *= base
		exponent /= 2
	}
	return y
}