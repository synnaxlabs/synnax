// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// This package only supports little-endian architectures. The unsafe byte-level
// operations assume little-endian memory layout.
//go:build 386 || amd64 || arm || arm64 || loong64 || mips64le || mipsle || ppc64le || riscv64 || wasm

package unsafe

import (
	"bytes"
	"fmt"
	"unsafe"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
)

// ReinterpretSlice re-interprets a slice of one type as a slice of another type that
// has the same size. A and B must have the same size (i.e. float32 -> uint32,
// int64 -> uint64) in order to work. Panics if sizes don't match.
//
// IF YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func ReinterpretSlice[A, B types.Sized](in []A) []B {
	var a A
	var b B
	if unsafe.Sizeof(a) != unsafe.Sizeof(b) {
		panic(fmt.Sprintf(
			"unsafe.ReinterpretSlice: mismatched sizes %d (%s) and %d (%s)",
			unsafe.Sizeof(a),
			types.Name[A](),
			unsafe.Sizeof(b),
			types.Name[B](),
		))
	}
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
func CastSlice[A, B types.Sized](in []A) []B {
	if len(in) == 0 {
		return nil
	}
	var (
		a     A
		b     B
		sizeA = unsafe.Sizeof(a)
		sizeB = unsafe.Sizeof(b)
	)
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

// CastBytes reinterprets a byte slice as a single value of type T.
// Returns an error if the byte slice is shorter than sizeof(T).
//
// IMPORTANT: This function operates at the byte level and does not perform type conversion.
// IF YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func CastBytes[T types.Sized](bytes []byte) (T, error) {
	var t T
	if uintptr(len(bytes)) < unsafe.Sizeof(t) {
		return t, errors.Newf(
			"unsafe.CastBytes: byte slice too short (%d bytes) for %s (%d bytes)",
			len(bytes),
			types.Name[T](),
			unsafe.Sizeof(t),
		)
	}
	return *(*T)(unsafe.Pointer(&bytes[0])), nil
}

// CastToBytes reinterprets a single value of type T as a byte slice. This is the
// inverse of CastBytes. The returned slice will have length sizeof(T).
//
// IMPORTANT: This function operates at the byte level and does not perform type conversion.
// The byte order is platform-dependent (little-endian on x86/ARM64).
// IF YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func CastToBytes[T types.Sized](in T) []byte {
	size := int(unsafe.Sizeof(in))
	out := make([]byte, size)
	*(*T)(unsafe.Pointer(&out[0])) = in
	return out
}

// ReinterpretMap re-interprets a map of one type as a map of another type. Note that the input
// and output maps must have compatible memory layouts in order for this to work. IF
// YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func ReinterpretMap[A, B, C, D types.Sized](in map[A]B) map[C]D {
	if len(in) == 0 {
		return nil
	}
	return *(*map[C]D)(unsafe.Pointer(&in))
}

// ReinterpretMapKeys re-interprets a map's keys from one type to another while preserving
// values of any type. A and B must have the same memory layout (e.g., uint32 -> type alias).
//
// IF YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func ReinterpretMapKeys[A, B types.Sized, V any](in map[A]V) map[B]V {
	if len(in) == 0 {
		return nil
	}
	return *(*map[B]V)(unsafe.Pointer(&in))
}

// ReinterpretMapValues re-interprets a map's values from one type to another while preserving
// keys of any comparable type. A and B must have the same memory layout (e.g., uint64 -> type alias).
//
// IF YOU DON'T KNOW WHAT YOU'RE DOING, DON'T USE THIS.
func ReinterpretMapValues[K comparable, A, B types.Sized](in map[K]A) map[K]B {
	if len(in) == 0 {
		return nil
	}
	return *(*map[K]B)(unsafe.Pointer(&in))
}

// EncodePrimitive encodes a primitive value to a byte slice using little-endian
// byte order for numeric types.
func EncodePrimitive[K types.Primitive](value K) []byte {
	switch v := any(value).(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	default:
		// All remaining Primitive types are fixed-size numerics
		size := int(unsafe.Sizeof(value))
		out := make([]byte, size)
		copy(out, unsafe.Slice((*byte)(unsafe.Pointer(&value)), size))
		return out
	}
}

// DecodePrimitive decodes a byte slice into a primitive value using little-endian
// byte order for numeric types. For []byte types, the data is cloned to prevent
// issues with buffer reuse.
func DecodePrimitive[K types.Primitive](data []byte) K {
	var zero K
	switch any(zero).(type) {
	case string:
		return any(string(data)).(K)
	case []byte:
		return any(bytes.Clone(data)).(K)
	default:
		return *(*K)(unsafe.Pointer(&data[0]))
	}
}
