// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unsafe

import "unsafe"

// ReinterpretSlice re-interprets a slice of one type as a slice of another type. Note that
// A and B must have compatible memory layouts in order for this to work. IF YOU DON'T
// KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func ReinterpretSlice[A, B any](in []A) []B {
	if len(in) == 0 {
		return nil
	}
	return unsafe.Slice((*B)(unsafe.Pointer(&in[0])), len(in))
}

// CastSlice reinterprets a slice of one type as a slice of another type with different
// element sizes. The output slice length is calculated based on the byte sizes of the
// types: len(output) = len(input) * sizeof(A) / sizeof(B). This is useful for reinterpreting
// data between types of different densities (e.g., []float64 to []uint8).
//
// IMPORTANT: This function operates at the byte level and does not perform type conversion.
// The total byte count must be evenly divisible by the target type size. IF YOU DON'T
// KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func CastSlice[A, B any](in []A) []B {
	if len(in) == 0 {
		return nil
	}
	var b B
	newLen := (len(in) * int(unsafe.Sizeof(in[0]))) / int(unsafe.Sizeof(b))
	return unsafe.Slice((*B)(unsafe.Pointer(&in[0])), newLen)
}

// ReinterpretMap re-interprets a map of one type as a map of another type. Note that the input
// and output maps must have compatible memory layouts in order for this to work. IF
// YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func ReinterpretMap[A comparable, B any, C comparable, D any](in map[A]B) map[C]D {
	if len(in) == 0 {
		return nil
	}
	return *(*map[C]D)(unsafe.Pointer(&in))
}
