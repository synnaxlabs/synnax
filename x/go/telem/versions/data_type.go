// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import v0 "github.com/synnaxlabs/x/telem/versions/v0"

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
