// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unsafe

import (
	"fmt"
	"unsafe"

	"github.com/synnaxlabs/x/types"
)

// ReinterpretSlice re-interprets a slice of one type as a slice of another type that
// has the same density. A and B must have the same density (i.e. float32 -> uint32,
// int64 -> uint64) in order for to work.
//
// IF YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
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
	var (
		b     B
		sizeA = unsafe.Sizeof(in[0])
		sizeB = unsafe.Sizeof(b)
	)
	if sizeB == 0 {
		panic("unsafe.CastSlice: zero-sized target type")
	}
	totalBytes := uintptr(len(in)) * sizeA
	if totalBytes%sizeB != 0 {
		panic(fmt.Sprintf(
			"unsafe.CastSlice: incompatible element size %v (%s) with total byte length %v and element with size %v (%s)",
			sizeA,
			types.Name[A](),
			totalBytes,
			sizeB,
			types.Name[B](),
		))
	}
	newLen := int(totalBytes / sizeB)
	ptr := unsafe.Pointer(&in[0])
	// Hot path: types have perfect memory alignment, so we can just do a cast of the
	// underlying slice instead of a deep copy.
	if uintptr(ptr)%unsafe.Alignof(b) == 0 {
		return unsafe.Slice((*B)(ptr), newLen)
	}
	out := make([]B, newLen)
	srcBytes := unsafe.Slice((*byte)(ptr), totalBytes)
	dstBytes := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), totalBytes)
	copy(dstBytes, srcBytes)
	return out
}

// CastBytes reinterprets a byte slice as a single value of type T. This is equivalent
// to CastSlice[byte, T](bytes)[0] but more convenient for single-value conversions.
// The byte slice must contain at least sizeof(T) bytes.
//
// IMPORTANT: This function operates at the byte level and does not perform type conversion.
// IF YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func CastBytes[T any](bytes []byte) T {
	return CastSlice[byte, T](bytes)[0]
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
