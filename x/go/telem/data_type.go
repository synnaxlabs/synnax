// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "github.com/google/uuid"

// DataType is a string that represents a data type.
type DataType string

// Density returns the density of the given data type. If the data type has no known
// density, UnknownDensity is returned.
func (dt DataType) Density() Density {
	switch dt {
	case TimeStampT, Float64T, Uint64T, Int64T:
		return Bit64
	case Float32T, Int32T, Uint32T:
		return Bit32
	case Int16T, Uint16T:
		return Bit16
	case Int8T, Uint8T:
		return Bit8
	case UUIDT:
		return Bit128
	default:
		return UnknownDensity
	}
}

// IsVariable returns true if the data type has a variable density.
func (dt DataType) IsVariable() bool {
	return dt == BytesT || dt == StringT || dt == JSONT
}

// InferDataType infers the data type of the given Sample.
func InferDataType[T Sample2]() DataType {
	var t T
	switch any(t).(type) {
	case uint8:
		return Uint8T
	case uint16:
		return Uint16T
	case uint32:
		return Uint32T
	case uint64:
		return Uint64T
	case int8:
		return Int8T
	case int16:
		return Int16T
	case int32:
		return Int32T
	case int64:
		return Int64T
	case float32:
		return Float32T
	case float64:
		return Float64T
	case TimeStamp:
		return TimeStampT
	case uuid.UUID:
		return UUIDT
	case string:
		return StringT
	case []byte:
		return BytesT
	default:
		// degenerate case, should never hit this path.
		return UnknownT
	}
}

const (
	// UnknownT is an unknown data type.
	UnknownT DataType = ""
	// Uint8T is a data type for an 8-bit unsigned integer, i.e., a single byte.
	Uint8T DataType = "uint8"
	// Uint16T is a data type for a 16-bit unsigned integer.
	Uint16T DataType = "uint16"
	// Uint32T is a data type for a 32-bit unsigned integer.
	Uint32T DataType = "uint32"
	// Uint64T is a data type for a 64-bit unsigned integer.
	Uint64T DataType = "uint64"
	// Int8T is a data type for an 8-bit integer.
	Int8T DataType = "int8"
	// Int16T is a data type for a 16-bit integer.
	Int16T DataType = "int16"
	// Int32T is a data type for a 32-bit integer.
	Int32T DataType = "int32"
	// Int64T is a data type for a 64-bit integer.
	Int64T DataType = "int64"
	// Float32T is a data type for a 32-bit IEEE-754 floating point number.
	Float32T DataType = "float32"
	// Float64T is a data type for a 64-bit IEEE-754 floating point number.
	Float64T DataType = "float64"
	// TimeStampT is a data type for a signed 64-bit nanosecond count since the Unix
	// epoch.
	TimeStampT DataType = "timestamp"
	// UUIDT is a data type for a 128-bit UUID value.
	UUIDT DataType = "uuid"
	// StringT is a variable density data type for a UTF-8 encoded string.
	StringT DataType = "string"
	// BytesT is a variable density data type for an arbitrary byte array.
	BytesT DataType = "bytes"
	// JSONT is a variable density data type for UTF-8 encoded JSON text.
	JSONT DataType = "json"
)
