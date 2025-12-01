// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary

import (
	"encoding/binary"
	"math"
	"unsafe"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/types"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

// EncodePrimitive encodes a primitive value to a byte slice using little-endian
// byte order for numeric types. This is a zero-allocation function for fixed-size
// numeric types.
func EncodePrimitive[K types.Primitive](value K) ([]byte, error) {
	return xunsafe.CastToBytes(value)
}

// DecodePrimitive decodes a byte slice into a primitive value using little-endian
// byte order for numeric types.
func DecodePrimitive[K types.Primitive](data []byte) (K, error) {
	var zero K
	switch any(zero).(type) {
	case int8:
		if len(data) < 1 {
			return zero, insufficientData(1, len(data))
		}
		return any(int8(data[0])).(K), nil
	case int16:
		if len(data) < 2 {
			return zero, insufficientData(2, len(data))
		}
		return any(int16(binary.LittleEndian.Uint16(data))).(K), nil
	case int32:
		if len(data) < 4 {
			return zero, insufficientData(4, len(data))
		}
		return any(int32(binary.LittleEndian.Uint32(data))).(K), nil
	case int64:
		if len(data) < 8 {
			return zero, insufficientData(8, len(data))
		}
		return any(int64(binary.LittleEndian.Uint64(data))).(K), nil
	case int:
		if len(data) < 8 {
			return zero, insufficientData(8, len(data))
		}
		return any(int(binary.LittleEndian.Uint64(data))).(K), nil
	case uint8:
		if len(data) < 1 {
			return zero, insufficientData(1, len(data))
		}
		return any(data[0]).(K), nil
	case uint16:
		if len(data) < 2 {
			return zero, insufficientData(2, len(data))
		}
		return any(binary.LittleEndian.Uint16(data)).(K), nil
	case uint32:
		if len(data) < 4 {
			return zero, insufficientData(4, len(data))
		}
		return any(binary.LittleEndian.Uint32(data)).(K), nil
	case uint64:
		if len(data) < 8 {
			return zero, insufficientData(8, len(data))
		}
		return any(binary.LittleEndian.Uint64(data)).(K), nil
	case uint:
		if len(data) < 8 {
			return zero, insufficientData(8, len(data))
		}
		return any(uint(binary.LittleEndian.Uint64(data))).(K), nil
	case float32:
		if len(data) < 4 {
			return zero, insufficientData(4, len(data))
		}
		return any(math.Float32frombits(binary.LittleEndian.Uint32(data))).(K), nil
	case float64:
		if len(data) < 8 {
			return zero, insufficientData(8, len(data))
		}
		return any(math.Float64frombits(binary.LittleEndian.Uint64(data))).(K), nil
	case string:
		return any(unsafe.String(unsafe.SliceData(data), len(data))).(K), nil
	case []byte:
		return any(data).(K), nil
	default:
		return zero, decodeUnknownPrimitiveType(zero)
	}
}

func encodeUnknownPrimitiveType[K types.Primitive](value K) error {
	return EncodeError.Wrapf("unknown primitive type: %T", value)
}

func decodeUnknownPrimitiveType[K types.Primitive](value K) error {
	return DecodeError.Wrapf("unknown primitive type: %T", value)
}

func insufficientData(expected, got int) error {
	return DecodeError.Wrapf("insufficient data: expected %d bytes, got %d", expected, got)
}
