// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"encoding/binary"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/unsafe"
)

// NumericSample represents any numeric value that can be stored in a Series and have
// mathematical operations performed on it.
type NumericSample interface {
	uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 |
		float32 | float64 | TimeStamp
}

// FixedSample represents any value that can be stored in a Series and has a fixed
// density.
type FixedSample interface{ NumericSample | uuid.UUID }

// VariableSample is a type that can be stored in a variable-density series.
type VariableSample interface{ []byte | string }

// Sample represents any value that can be stored in a non-JSON Series.
type Sample interface{ FixedSample | VariableSample }

// ByteOrder is the standard order for encoding/decoding numeric values across the
// Synnax telemetry ecosystem.
var ByteOrder = binary.LittleEndian

// variableLengthPrefixSize is the number of bytes used for the uint32 LE length prefix
// in variable-density series encoding.
const variableLengthPrefixSize = 4

// MarshalFixed encodes fixed-width samples into their contiguous little-endian byte
// representation.
func MarshalFixed[T FixedSample](data ...T) []byte {
	src := unsafe.CastSlice[T, byte](data)
	b := make([]byte, len(src))
	copy(b, src)
	return b
}

// UnmarshalFixed reinterprets a fixed-density buffer as a slice of T.
func UnmarshalFixed[T FixedSample](b []byte) []T { return unsafe.CastSlice[byte, T](b) }

// MarshalVariable encodes variable-length samples, prefixing each with its uint32 LE
// byte length.
func MarshalVariable[T VariableSample](data ...T) []byte {
	var total int
	for _, d := range data {
		total += variableLengthPrefixSize + len(d)
	}
	b := make([]byte, total)
	var offset int
	for _, d := range data {
		ByteOrder.PutUint32(b[offset:], uint32(len(d)))
		offset += variableLengthPrefixSize
		copy(b[offset:], d)
		offset += len(d)
	}
	return b
}

// UnmarshalVariable decodes a length-prefixed variable-density buffer into a slice of
// T. Samples with a length prefix that overruns the buffer terminate decoding.
func UnmarshalVariable[T VariableSample](b []byte) []T {
	var data []T
	var offset int
	for offset+variableLengthPrefixSize <= len(b) {
		length := int(ByteOrder.Uint32(b[offset:]))
		offset += variableLengthPrefixSize
		if offset+length > len(b) {
			break
		}
		data = append(data, T(b[offset:offset+length]))
		offset += length
	}
	return data
}
