// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"math"
	"strconv"
)

// AppendFloatText appends the shortest text that parses back to v at bitSize (32 or
// 64) and returns the extended buffer. The text is in exponent notation below 1e-6 or
// from 1e21, with an unpadded exponent (1e-7, 1e+21). Infinities are Infinity and
// -Infinity.
func AppendFloatText(dst []byte, v float64, bitSize int) []byte {
	if math.IsInf(v, 1) {
		return append(dst, "Infinity"...)
	}
	if math.IsInf(v, -1) {
		return append(dst, "-Infinity"...)
	}
	if a := math.Abs(v); a != 0 && (a < 1e-6 || a >= 1e21) {
		dst = strconv.AppendFloat(dst, v, 'e', -1, bitSize)
		// strconv pads a one-digit exponent with a zero.
		if n := len(dst); dst[n-2] == '0' && (dst[n-3] == '+' || dst[n-3] == '-') {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
		return dst
	}
	return strconv.AppendFloat(dst, v, 'f', -1, bitSize)
}
