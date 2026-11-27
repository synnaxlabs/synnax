// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

// Density returns the density of the given data type. If the data type has no known
// density, UnknownDensity is returned.
func (dt DataType) Density() Density {
	switch dt {
	case TimestampT, Float64T, Uint64T, Int64T:
		return Bit64
	case Float32T, Int32T, Uint32T:
		return Bit32
	case Int16T, Uint16T:
		return Bit16
	case Int8T, Uint8T, BooleanT:
		return Bit8
	case UUIDT:
		return Bit128
	case UnknownT, StringT, BytesT, JSONT:
		fallthrough
	default:
		return UnknownDensity
	}
}

// IsVariable returns true if the data type has a variable density.
func (dt DataType) IsVariable() bool {
	return dt == BytesT || dt == StringT || dt == JSONT
}

const (
	// UnknownT is an unknown data type.
	UnknownT DataType = ""
	// Uint8T is a data type for an 8-bit unsigned integer, i.e., a single byte.
	Uint8T DataType = DataType(DataTypeNameUint8)
	// Uint16T is a data type for a 16-bit unsigned integer.
	Uint16T DataType = DataType(DataTypeNameUint16)
	// Uint32T is a data type for a 32-bit unsigned integer.
	Uint32T DataType = DataType(DataTypeNameUint32)
	// Uint64T is a data type for a 64-bit unsigned integer.
	Uint64T DataType = DataType(DataTypeNameUint64)
	// Int8T is a data type for an 8-bit integer.
	Int8T DataType = DataType(DataTypeNameInt8)
	// Int16T is a data type for a 16-bit integer.
	Int16T DataType = DataType(DataTypeNameInt16)
	// Int32T is a data type for a 32-bit integer.
	Int32T DataType = DataType(DataTypeNameInt32)
	// Int64T is a data type for a 64-bit integer.
	Int64T DataType = DataType(DataTypeNameInt64)
	// Float32T is a data type for a 32-bit IEEE-754 floating point number.
	Float32T DataType = DataType(DataTypeNameFloat32)
	// Float64T is a data type for a 64-bit IEEE-754 floating point number.
	Float64T DataType = DataType(DataTypeNameFloat64)
	// TimestampT is a data type for a signed 64-bit nanosecond count since the Unix
	// epoch.
	TimestampT DataType = DataType(DataTypeNameTimestamp)
	// UUIDT is a data type for a 128-bit UUID value.
	UUIDT DataType = DataType(DataTypeNameUUID)
	// StringT is a variable density data type for a UTF-8 encoded string.
	StringT DataType = DataType(DataTypeNameString)
	// BytesT is a variable density data type for an arbitrary byte array.
	BytesT DataType = DataType(DataTypeNameBytes)
	// JSONT is a variable density data type for UTF-8 encoded JSON text.
	JSONT DataType = DataType(DataTypeNameJSON)
	// BooleanT is a data type for a boolean value. Samples are a single byte with
	// canonical values 0x00 (false) and 0x01 (true).
	BooleanT DataType = DataType(DataTypeNameBoolean)
)
