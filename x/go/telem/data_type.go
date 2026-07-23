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
	"github.com/google/uuid"
	v0 "github.com/synnaxlabs/x/telem/versions/v0"
)

const (
	// UnknownT is an unknown data type.
	UnknownT DataType = v0.UnknownT
	// Uint8T is a data type for an 8-bit unsigned integer, i.e., a single byte.
	Uint8T DataType = v0.Uint8T
	// Uint16T is a data type for a 16-bit unsigned integer.
	Uint16T DataType = v0.Uint16T
	// Uint32T is a data type for a 32-bit unsigned integer.
	Uint32T DataType = v0.Uint32T
	// Uint64T is a data type for a 64-bit unsigned integer.
	Uint64T DataType = v0.Uint64T
	// Int8T is a data type for an 8-bit integer.
	Int8T DataType = v0.Int8T
	// Int16T is a data type for a 16-bit integer.
	Int16T DataType = v0.Int16T
	// Int32T is a data type for a 32-bit integer.
	Int32T DataType = v0.Int32T
	// Int64T is a data type for a 64-bit integer.
	Int64T DataType = v0.Int64T
	// Float32T is a data type for a 32-bit IEEE-754 floating point number.
	Float32T DataType = v0.Float32T
	// Float64T is a data type for a 64-bit IEEE-754 floating point number.
	Float64T DataType = v0.Float64T
	// TimeStampT is a data type for a signed 64-bit nanosecond count since the Unix
	// epoch.
	TimeStampT DataType = v0.TimeStampT
	// UUIDT is a data type for a 128-bit UUID value.
	UUIDT DataType = v0.UUIDT
	// StringT is a variable density data type for a UTF-8 encoded string.
	StringT DataType = v0.StringT
	// BytesT is a variable density data type for an arbitrary byte array.
	BytesT DataType = v0.BytesT
	// JSONT is a variable density data type for UTF-8 encoded JSON text.
	JSONT DataType = v0.JSONT
)

// InferDataType infers the data type of the given Sample.
func InferDataType[T Sample]() DataType {
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
