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
	"fmt"
	"math"

	"github.com/samber/lo"
	xbinary "github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/types"
)

// Sample represents any numeric value that can be stored in a Series.
// It must satisfy the types.Numeric interface.
type Sample interface{ types.Numeric }

// NewSeries creates a new Series from a slice of numeric values. It automatically
// determines the data type from the first element. Panics if the input slice is empty.
func NewSeries[T Sample](data []T) Series {
	return Series{
		DataType: InferDataType[T](),
		Data:     MarshalSlice(data),
	}
}

// NewSeriesV is a variadic version of NewSeries that creates a new Series from
// individual numeric values.
func NewSeriesV[T Sample](data ...T) Series { return NewSeries(data) }

// MakeSeries allocates a new Series with the specified DataType and length. Note that
// this function allocates a length and not a capacity.
func MakeSeries(dt DataType, len int64) Series {
	return Series{DataType: dt, Data: make([]byte, len*int64(dt.Density()))}
}

// NewSecondsTSV creates a new Series containing TimeStamp values. All input timestamps
// are multiplied by SecondTS to convert them to the standard time unit used in the
// system.
func NewSecondsTSV(data ...TimeStamp) Series {
	for i := range data {
		data[i] *= SecondTS
	}
	return Series{DataType: TimeStampT, Data: MarshalSlice(data)}
}

// NewStrings creates a new Series from a slice of strings. The strings are stored with
// newline characters as delimiters.
func NewStrings(data []string) Series {
	return Series{DataType: StringT, Data: MarshalStrings(data, StringT)}
}

// NewStringsV is a variadic version of NewStrings that creates a new Series from
// individual string values.
func NewStringsV(data ...string) Series { return NewStrings(data) }

func NewStaticJSONV[T any](data ...T) (series Series) {
	series.DataType = JSONT
	strings := make([]string, len(data))
	for i, v := range data {
		strings[i] = xbinary.MustEncodeJSONToString(v)
	}
	series.Data = MarshalStrings(strings, series.DataType)
	return series
}

const newLine = '\n'

// MarshalStrings converts a slice of strings into a byte slice. Each string is
// terminated with a newline character. Panics if the DataType is not variable.
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

// UnmarshalStrings converts a byte slice back into a slice of strings. It assumes
// strings are separated by newline characters.
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

// MarshalSlice converts a slice of numeric values into a byte slice according to the
// specified DataType.
func MarshalSlice[T Sample](data []T) []byte {
	var (
		dt = InferDataType[T]()
		b  = make([]byte, dt.Density().Size(int64(len(data))))
		m  = MarshalF[T](dt)
	)
	for i, v := range data {
		base := i * int(dt.Density())
		m(b[base:base+int(dt.Density())], v)
	}
	return b
}

// UnmarshalSlice converts a byte slice back into a slice of numeric values according to
// the specified DataType.
func UnmarshalSlice[T Sample](b []byte, dt DataType) []T {
	data := make([]T, len(b)/int(dt.Density()))
	um := UnmarshalF[T](dt)
	for i := range data {
		base := i * int(dt.Density())
		data[i] = um(b[base : base+int(dt.Density())])
	}
	return data
}

// Unmarshal converts a Series' data back into a slice of the original type.
func Unmarshal[T Sample](series Series) []T {
	return UnmarshalSlice[T](series.Data, series.DataType)
}

// ByteOrder specifies the byte order used for encoding numeric values. The package uses
// little-endian byte order by default.
var ByteOrder = binary.LittleEndian

func MarshalInt8[T types.Numeric](b []byte, v T)   { b[0] = byte(v) }
func MarshalInt16[T types.Numeric](b []byte, v T)  { ByteOrder.PutUint16(b, uint16(v)) }
func MarshalInt32[T types.Numeric](b []byte, v T)  { ByteOrder.PutUint32(b, uint32(v)) }
func MarshalInt64[T types.Numeric](b []byte, v T)  { ByteOrder.PutUint64(b, uint64(v)) }
func MarshalUint8[T types.Numeric](b []byte, v T)  { b[0] = byte(v) }
func MarshalUint16[T types.Numeric](b []byte, v T) { ByteOrder.PutUint16(b, uint16(v)) }
func MarshalUint32[T types.Numeric](b []byte, v T) { ByteOrder.PutUint32(b, uint32(v)) }
func MarshalUint64[T types.Numeric](b []byte, v T) { ByteOrder.PutUint64(b, uint64(v)) }
func MarshalFloat32[T types.Numeric](b []byte, v T) {
	ByteOrder.PutUint32(b, math.Float32bits(float32(v)))
}
func MarshalFloat64[T types.Numeric](b []byte, v T) {
	ByteOrder.PutUint64(b, math.Float64bits(float64(v)))
}
func MarshalTimeStamp[T types.Numeric](b []byte, v T) { ByteOrder.PutUint64(b, uint64(v)) }

// MarshalF returns a function that can marshal a single value of type K into a byte
// slice according to the specified DataType. Panics if the DataType is not supported.
func MarshalF[T types.Numeric](dt DataType) func(b []byte, v T) {
	switch dt {
	case Float64T:
		return MarshalFloat64[T]
	case Float32T:
		return MarshalFloat32[T]
	case Int64T:
		return MarshalInt64[T]
	case Int32T:
		return MarshalInt32[T]
	case Int16T:
		return MarshalInt16[T]
	case Int8T:
		return MarshalInt8[T]
	case Uint64T:
		return MarshalUint64[T]
	case Uint32T:
		return MarshalUint32[T]
	case Uint16T:
		return MarshalUint16[T]
	case Uint8T:
		return MarshalUint8[T]
	case TimeStampT:
		return MarshalTimeStamp[T]
	}
	panic(fmt.Sprintf("unsupported data type %s", dt))
}

func UnmarshalInt8[T types.Numeric](b []byte) T   { return T(b[0]) }
func UnmarshalInt16[T types.Numeric](b []byte) T  { return T(ByteOrder.Uint16(b)) }
func UnmarshalInt32[T types.Numeric](b []byte) T  { return T(ByteOrder.Uint32(b)) }
func UnmarshalInt64[T types.Numeric](b []byte) T  { return T(ByteOrder.Uint64(b)) }
func UnmarshalUint8[T types.Numeric](b []byte) T  { return T(b[0]) }
func UnmarshalUint16[T types.Numeric](b []byte) T { return T(ByteOrder.Uint16(b)) }
func UnmarshalUint32[T types.Numeric](b []byte) T { return T(ByteOrder.Uint32(b)) }
func UnmarshalUint64[T types.Numeric](b []byte) T { return T(ByteOrder.Uint64(b)) }
func UnmarshalFloat32[T types.Numeric](b []byte) T {
	return T(math.Float32frombits(ByteOrder.Uint32(b)))
}
func UnmarshalFloat64[T types.Numeric](b []byte) T {
	return T(math.Float64frombits(ByteOrder.Uint64(b)))
}
func UnmarshalTimeStamp[T types.Numeric](b []byte) T { return T(TimeStamp(ByteOrder.Uint64(b))) }

// UnmarshalF returns a function that can unmarshal a byte slice into a single value of
// type K according to the specified DataType. Panics if the DataType is not supported.
func UnmarshalF[T types.Numeric](dt DataType) func(b []byte) T {
	switch dt {
	case Float64T:
		return UnmarshalFloat64[T]
	case Float32T:
		return UnmarshalFloat32[T]
	case Int64T:
		return UnmarshalInt64[T]
	case Int32T:
		return UnmarshalInt32[T]
	case Int16T:
		return UnmarshalInt16[T]
	case Int8T:
		return UnmarshalInt8[T]
	case Uint64T:
		return UnmarshalUint64[T]
	case Uint32T:
		return UnmarshalUint32[T]
	case Uint16T:
		return UnmarshalUint16[T]
	case Uint8T:
		return UnmarshalUint8[T]
	case TimeStampT:
		return UnmarshalTimeStamp[T]
	}
	panic(fmt.Sprintf("unsupported data type %s", dt))
}
