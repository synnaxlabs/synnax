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
	"encoding/binary"
	"math"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/types"
)

// Value represents any numeric value that can be stored in a Series.
// It must satisfy the types.Numeric interface.
type Value interface{ types.Numeric }

// NewSeries creates a new Series from a slice of numeric values.
// It automatically determines the data type from the first element.
// Panics if the input slice is empty.
func NewSeries[T Value](data []T) (series Series) {
	if len(data) == 0 {
		panic("cannot infer data type from empty array")
	}
	series.DataType = NewDataType[T](data[0])
	series.Data = MarshalSlice(data, series.DataType)
	return series
}

// NewSeriesV is a variadic version of NewSeries that creates a new Series
// from individual numeric values.
func NewSeriesV[T Value](data ...T) (series Series) {
	return NewSeries[T](data)
}

// AllocSeries allocates a new Series with the specified DataType and length. Note that
// this function allocates a length and not a capacity.
func AllocSeries(dt DataType, size int64) (series Series) {
	series.DataType = dt
	series.Data = make([]byte, size*int64(dt.Density()))
	return series
}

// NewSecondsTSV creates a new Series containing TimeStamp values.
// All input timestamps are multiplied by SecondTS to convert them to the standard
// time unit used in the system.
func NewSecondsTSV(data ...TimeStamp) (series Series) {
	for i := range data {
		data[i] *= SecondTS
	}
	series.DataType = TimeStampT
	series.Data = MarshalSlice(data, series.DataType)
	return series
}

// NewStrings creates a new Series from a slice of strings.
// The strings are stored with newline characters as delimiters.
func NewStrings(data []string) (series Series) {
	series.DataType = StringT
	series.Data = MarshalStrings(data, series.DataType)
	return series
}

// NewStringsV is a variadic version of NewStrings that creates a new Series
// from individual string values.
func NewStringsV(data ...string) (series Series) { return NewStrings(data) }

const newLine = '\n'

// MarshalStrings converts a slice of strings into a byte slice.
// Each string is terminated with a newline character.
// Panics if the DataType is not variable length.
func MarshalStrings(data []string, dt DataType) []byte {
	if !dt.IsVariable() {
		panic("data type must be variable length")
	}
	total := lo.SumBy(data, func(s string) int64 { return int64(len(s)) + 1 })
	b := make([]byte, total)
	offset := 0
	for _, s := range data {
		copy(b[offset:], s)
		b[offset+len(s)] = newLine
		offset += len(s) + 1
	}
	return b
}

// UnmarshalStrings converts a byte slice back into a slice of strings.
// It assumes strings are separated by newline characters.
func UnmarshalStrings(b []byte) (data []string) {
	offset := 0
	for offset < len(b) {
		end := offset
		for b[end] != newLine {
			end++
		}
		data = append(data, string(b[offset:end]))
		offset = end + 1
	}
	return data
}

// MarshalSlice converts a slice of numeric values into a byte slice according
// to the specified DataType.
func MarshalSlice[T Value](data []T, dt DataType) []byte {
	b := make([]byte, dt.Density().Size(int64(len(data))))
	m := MarshalF[T](dt)
	for i, v := range data {
		base := i * int(dt.Density())
		m(b[base:base+int(dt.Density())], v)
	}
	return b
}

// UnmarshalSlice converts a byte slice back into a slice of numeric values
// according to the specified DataType.
func UnmarshalSlice[T Value](b []byte, dt DataType) (data []T) {
	data = make([]T, len(b)/int(dt.Density()))
	um := UnmarshalF[T](dt)
	for i := range data {
		base := i * int(dt.Density())
		data[i] = um(b[base : base+int(dt.Density())])
	}
	return data
}

// Unmarshal converts a Series' data back into a slice of the original type.
func Unmarshal[T Value](series Series) []T {
	return UnmarshalSlice[T](series.Data, series.DataType)
}

// ByteOrder specifies the byte order used for encoding numeric values.
// The package uses little-endian byte order by default.
var ByteOrder = binary.LittleEndian

// MarshalF returns a function that can marshal a single value of type T
// into a byte slice according to the specified DataType.
// Panics if the DataType is not supported.
func MarshalF[T types.Numeric](dt DataType) func(b []byte, v T) {
	switch dt {
	case Float64T:
		return func(b []byte, v T) {
			bits := math.Float64bits(float64(v))
			ByteOrder.PutUint64(b, bits)
		}
	case Float32T:
		return func(b []byte, v T) {
			bits := math.Float32bits(float32(v))
			ByteOrder.PutUint32(b, bits)
		}
	case Int64T:
		return func(b []byte, v T) { ByteOrder.PutUint64(b, uint64(v)) }
	case Int32T:
		return func(b []byte, v T) { ByteOrder.PutUint32(b, uint32(v)) }
	case Int16T:
		return func(b []byte, v T) { ByteOrder.PutUint16(b, uint16(v)) }
	case Int8T:
		return func(b []byte, v T) { b[0] = byte(v) }
	case Uint64T:
		return func(b []byte, v T) { ByteOrder.PutUint64(b, uint64(v)) }
	case Uint32T:
		return func(b []byte, v T) { ByteOrder.PutUint32(b, uint32(v)) }
	case Uint16T:
		return func(b []byte, v T) { ByteOrder.PutUint16(b, uint16(v)) }
	case Uint8T:
		return func(b []byte, v T) { b[0] = byte(v) }
	case TimeStampT:
		return func(b []byte, v T) { ByteOrder.PutUint64(b, uint64(v)) }
	}
	panic("unsupported data type")
}

// UnmarshalF returns a function that can unmarshal a byte slice into
// a single value of type T according to the specified DataType.
// Panics if the DataType is not supported.
func UnmarshalF[T types.Numeric](dt DataType) func(b []byte) (res T) {
	switch dt {
	case Float64T:
		return func(b []byte) T {
			bits := ByteOrder.Uint64(b)
			return T(math.Float64frombits(bits))
		}
	case Float32T:
		return func(b []byte) T {
			bits := ByteOrder.Uint32(b)
			return T(math.Float32frombits(bits))
		}
	case Int64T:
		return func(b []byte) T { return T(ByteOrder.Uint64(b)) }
	case Int32T:
		return func(b []byte) T { return T(ByteOrder.Uint32(b)) }
	case Int16T:
		return func(b []byte) T { return T(ByteOrder.Uint16(b)) }
	case Int8T:
		return func(b []byte) T { return T(b[0]) }
	case Uint64T:
		return func(b []byte) T { return T(ByteOrder.Uint64(b)) }
	case Uint32T:
		return func(b []byte) T { return T(ByteOrder.Uint32(b)) }
	case Uint16T:
		return func(b []byte) T { return T(ByteOrder.Uint16(b)) }
	case Uint8T:
		return func(b []byte) T { return T(b[0]) }
	case TimeStampT:
		return func(b []byte) T { return T(ByteOrder.Uint64(b)) }
	}
	panic("unsupported data type")
}
