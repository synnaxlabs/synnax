// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package strconv extends the standard library's strconv with the float text form every
// Synnax runtime writes.
package strconv

import (
	"bytes"
	"math"
	"strconv"
)

// AppendFloat appends the shortest text that parses back to v at bitSize (32 or 64) and
// returns the extended buffer. The text is in exponent notation when the decimal
// exponent of its shortest digits is below -6 or from 21, with an unpadded exponent
// (1e-7, 1e+21). NaN is NaN, and infinities are Infinity and -Infinity.
func AppendFloat(dst []byte, v float64, bitSize int) []byte {
	if math.IsNaN(v) {
		return append(dst, "NaN"...)
	}
	if math.IsInf(v, 1) {
		return append(dst, "Infinity"...)
	}
	if math.IsInf(v, -1) {
		return append(dst, "-Infinity"...)
	}
	start := len(dst)
	dst = strconv.AppendFloat(dst, v, 'e', -1, bitSize)
	e := start + bytes.LastIndexByte(dst[start:], 'e')
	exp, err := strconv.Atoi(string(dst[e+1:]))
	if err != nil {
		panic(err)
	}
	if exp >= -6 && exp < 21 {
		return strconv.AppendFloat(dst[:start], v, 'f', -1, bitSize)
	}
	// strconv pads a one-digit exponent with a zero.
	if n := len(dst); dst[n-2] == '0' && n-e == 4 {
		dst[n-2] = dst[n-1]
		dst = dst[:n-1]
	}
	return dst
}

// FormatFloat returns the text that AppendFloat appends for v.
func FormatFloat(
	v float64,
	bitSize int,
) string {
	return string(AppendFloat(nil, v, bitSize))
}
