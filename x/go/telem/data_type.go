// Copyright 2025 Synnax Labs, Inc.
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
	"reflect"
	"strings"
)

// DataType is a string that represents a data type.
type DataType string

// Density returns the density of the given data type. If the data type has no known
// density, DensityUnknown is returned.
func (d DataType) Density() Density { return dataTypeDensities[d] }

func (d DataType) IsVariable() bool {
	return d == BytesT || d == StringT || d == JSONT
}

func NewDataType[T any](v T) DataType {
	t := reflect.TypeOf(v)
	dt, ok := dataTypes[strings.ToLower(t.Name())]
	if !ok {
		panic(fmt.Sprintf("unknown data type %s", t.Name()))
	}
	return dt
}

var (
	UnknownT   DataType = ""
	TimeStampT          = DataType("timestamp")
	UUIDT               = DataType("uuid")
	Float64T   DataType = "float64"
	Float32T   DataType = "float32"
	Int64T     DataType = "int64"
	Int32T     DataType = "int32"
	Int16T     DataType = "int16"
	Int8T      DataType = "int8"
	Uint64T    DataType = "uint64"
	Uint32T    DataType = "uint32"
	Uint16T    DataType = "uint16"
	Uint8T     DataType = "uint8"
	BytesT     DataType = "bytes"
	StringT    DataType = "string"
	JSONT      DataType = "json"
)

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
	"json":      JSONT,
}

var dataTypeDensities = map[DataType]Density{
	TimeStampT: Bit64,
	UUIDT:      Bit128,
	Float64T:   Bit64,
	Float32T:   Bit32,
	Int32T:     Bit32,
	Int16T:     Bit16,
	Int8T:      Bit8,
	Uint64T:    Bit64,
	Uint32T:    Bit32,
	Uint16T:    Bit16,
	Uint8T:     Bit8,
	Int64T:     Bit64,
	BytesT:     DensityUnknown,
	StringT:    DensityUnknown,
	JSONT:      DensityUnknown,
	UnknownT:   DensityUnknown,
}
