// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op

import "math/bits"

const signBit = 1 << 63

// Mean is the running state of an Avg reduction. The zero value holds no samples. Avg
// clears it and restarts the count when prevCount is zero or the output is empty.
type Mean struct {
	// sum holds the samples of floats.
	sum float64
	// hi and lo hold the 128-bit sum of integer samples, XORed with a bias.
	hi, lo uint64
}

type signed interface {
	~int8 | ~int16 | ~int32 | ~int64
}

type unsigned interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

func add128[T signed | unsigned](m *Mean, samples []T, bias uint64) {
	var carry uint64
	for _, s := range samples {
		m.lo, carry = bits.Add64(m.lo, uint64(s)^bias, 0)
		m.hi += carry
	}
}

func avgUnsigned[T unsigned](m *Mean, samples []T, count int64) T {
	add128(m, samples, 0)
	quo, _ := bits.Div64(m.hi, m.lo, uint64(count))
	return T(quo)
}

// avgSigned truncates toward zero. The sign bias maps int64 onto uint64 in order, so
// the quotient is the floor of the biased mean.
func avgSigned[T signed](m *Mean, samples []T, count int64) T {
	add128(m, samples, signBit)
	quo, rem := bits.Div64(m.hi, m.lo, uint64(count))
	avg := int64(quo ^ signBit)
	if avg < 0 && rem != 0 {
		avg++
	}
	return T(avg)
}
