// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package sample defines the value types a Series can hold and the byte-level encoding
// that casts those values to and from a Series data buffer. It knows nothing about
// Series or DataType, so the versioned telem types (which decode inside their methods)
// and the top-level telem constructors can both depend on it. Constraints match on the
// underlying kind (~int64, ~string, ...) so named telem types like TimeStamp satisfy
// them without this package importing them.
package sample

import (
	"encoding/binary"

	"github.com/google/uuid"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

// Numeric is any numeric value that can be stored in a Series and have mathematical
// operations performed on it.
type Numeric interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int8 | ~int16 | ~int32 | ~int64 |
		~float32 | ~float64
}

// Fixed is any value that can be stored in a Series and has a fixed density.
type Fixed interface{ Numeric | uuid.UUID }

// Variable is any value that can be stored in a variable-density Series.
type Variable interface{ ~[]byte | ~string }

// Sample is any value that can be stored in a non-JSON Series.
type Sample interface{ Fixed | Variable }

// ByteOrder is the standard order for encoding/decoding numeric values across the
// Synnax telemetry ecosystem.
var ByteOrder = binary.LittleEndian

// VariablePrefixSize is the number of bytes used for the uint32 LE length prefix in
// variable-density series encoding.
const VariablePrefixSize = 4

// MarshalFixed encodes a slice of fixed-width samples into their contiguous
// little-endian byte representation.
func MarshalFixed[T Fixed](data []T) []byte {
	src := xunsafe.CastSlice[T, byte](data)
	b := make([]byte, len(src))
	copy(b, src)
	return b
}

// UnmarshalFixed reinterprets a fixed-density buffer as a slice of T.
func UnmarshalFixed[T Fixed](b []byte) []T { return xunsafe.CastSlice[byte, T](b) }

// MarshalVariable encodes a slice of variable-length samples, prefixing each with its
// uint32 LE byte length.
func MarshalVariable[T Variable](data []T) []byte {
	total := 0
	for _, d := range data {
		total += VariablePrefixSize + len(d)
	}
	b := make([]byte, total)
	offset := 0
	for _, d := range data {
		ByteOrder.PutUint32(b[offset:], uint32(len(d)))
		offset += VariablePrefixSize
		copy(b[offset:], d)
		offset += len(d)
	}
	return b
}

// UnmarshalVariable decodes a length-prefixed variable-density buffer into a slice of
// T. Samples with a length prefix that overruns the buffer terminate decoding.
func UnmarshalVariable[T Variable](b []byte) []T {
	var data []T
	offset := 0
	for offset+VariablePrefixSize <= len(b) {
		length := int(ByteOrder.Uint32(b[offset:]))
		offset += VariablePrefixSize
		if offset+length > len(b) {
			break
		}
		data = append(data, T(b[offset:offset+length]))
		offset += length
	}
	return data
}
