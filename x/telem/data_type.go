// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"github.com/synnaxlabs/x/types"
	"reflect"
)

// DataType is a string that represents a data type.
type DataType string

func (d DataType) Density() Density { return dataTypeDensities[d] }

func NewDataType[T types.Numeric](v T) DataType {
	t := reflect.TypeOf(v)
	switch t.Name() {
	case "int8":
		return Int8T
	case "int16":
		return Int16T
	case "int32":
		return Int32T
	case "int64":
		return Int64T
	case "uint8":
		return ByteT
	case "uint16":
		return Uint16T
	case "uint32":
		return Uint32T
	case "uint64":
		return Uint64T
	case "float32":
		return Float32T
	case "float64":
		return Float64T
	case "TimeStamp":
		return TimeStampT
	default:
		panic("unsupported data type")
	}
}

var (
	TimeStampT          = DataType("timestamp")
	UnknownT   DataType = ""
	Float64T   DataType = "float64"
	Float32T   DataType = "float32"
	Int64T     DataType = "int64"
	Int32T     DataType = "int32"
	Int16T     DataType = "int16"
	Int8T      DataType = "int8"
	Uint64T    DataType = "uint64"
	Uint32T    DataType = "uint32"
	Uint16T    DataType = "uint16"
	ByteT      DataType = "byte"
	BytesT     DataType = "bytes"
)

var dataTypeDensities = map[DataType]Density{
	TimeStampT: Bit64,
	UnknownT:   DensityUnknown,
	Float64T:   Bit64,
	Float32T:   Bit32,
	Int32T:     Bit32,
	Int16T:     Bit16,
	Int8T:      Bit8,
	Uint64T:    Bit64,
	Uint32T:    Bit32,
	Uint16T:    Bit16,
	ByteT:      Bit8,
	Int64T:     Bit64,
	BytesT:     DensityUnknown,
}
