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
	"fmt"
	"strings"

	"github.com/synnaxlabs/x/types"
)

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

// IsVariable returns true if the data type has a variable density i.e. is a string,
// JSON, or bytes.
func (dt DataType) IsVariable() bool {
	return dt == BytesT || dt == StringT || dt == JSONT
}

var dataTypes = map[string]DataType{
	"timestamp": TimeStampT,
	"uuid":      UUIDT,
	"float64":   Float64T,
	"float32":   Float32T,
	"int64":     Int64T,
	"int32":     Int32T,
	"int16":     Int16T,
	"int8":      Int8T,
	"uint8":     Uint8T,
	"uint64":    Uint64T,
	"uint32":    Uint32T,
	"uint16":    Uint16T,
	"bytes":     BytesT,
	"string":    StringT,
	"[]uint8":   BytesT,
}

// InferDataType infers the data type of the given generic type argument. Panics
// if the data type cannot be inferred.
func InferDataType[T any]() DataType {
	name := strings.ToLower(types.Name[T]())
	if dt, ok := dataTypes[name]; ok {
		return dt
	}
	panic(fmt.Sprintf("unknown data type %s", name))
}

const (
	// UnknownT is an unknown data type.
	UnknownT DataType = ""
	// TimeStampT is a data type for a 64-bit nanosecond integer since the
	// epoch in UTC.
	TimeStampT = DataType("timestamp")
	// UUIDT is a data type for a UUID V4.
	UUIDT = DataType("uuid")
	// Float64T is a data type for a 64-bit IEE754 floating point number.
	Float64T DataType = "float64"
	// Float32T is a data type for a 32-bit IEE754 floating point number.
	Float32T DataType = "float32"
	// Int64T is a data type for a 64-bit integer.
	Int64T DataType = "int64"
	// Int32T is a data type for a 32-bit integer.
	Int32T DataType = "int32"
	// Int16T is a data type for a 16-bit integer.
	Int16T DataType = "int16"
	// Int8T is a data type for an 8-bit integer.
	Int8T DataType = "int8"
	// Uint64T is a data type for a 64-bit unsigned integer.
	Uint64T DataType = "uint64"
	// Uint32T is a data type for a 32-bit unsigned integer.
	Uint32T DataType = "uint32"
	// Uint16T is a data type for a 16-bit unsigned integer.
	Uint16T DataType = "uint16"
	// Uint8T is a data type for an 8-bit unsigned integer, i.e., a single byte.
	Uint8T DataType = "uint8"
	// BytesT is a variable density data type for an arbitrary byte array
	BytesT DataType = "bytes"
	// StringT is a variable density data type for a UTF-8 encoded string.
	StringT DataType = "string"
	// JSONT is a variable density data type for a JSON structure.
	JSONT DataType = "json"
)
