// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/types"
	"math"
)

type Value interface {
	types.Numeric
}

func NewSeries[T Value](data []T) (series Series) {
	if len(data) == 0 {
		panic("cannot infer data type from empty array")
	}
	series.DataType = NewDataType[T](data[0])
	series.Data = MarshalSlice(data, series.DataType)
	return series
}

func NewSeriesV[T Value](data ...T) (series Series) {
	return NewSeries[T](data)
}

func AllocSeries(dt DataType, size int64) (series Series) {
	series.DataType = dt
	series.Data = make([]byte, size*int64(dt.Density()))
	return series
}

func NewSecondsTSV(data ...TimeStamp) (series Series) {
	for i := range data {
		data[i] *= SecondTS
	}
	series.DataType = TimeStampT
	series.Data = MarshalSlice(data, series.DataType)
	return series
}

func NewStrings(data []string) (series Series) {
	series.DataType = StringT
	series.Data = MarshalStrings(data, series.DataType)
	return series
}

func NewStringsV(data ...string) (series Series) {
	return NewStrings(data)
}

const newLine = '\n'

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

func MarshalSlice[T Value](data []T, dt DataType) []byte {
	b := make([]byte, dt.Density().Size(int64(len(data))))
	m := MarshalF[T](dt)
	for i, v := range data {
		base := i * int(dt.Density())
		m(b[base:base+int(dt.Density())], v)
	}
	return b
}

func UnmarshalSlice[T Value](b []byte, dt DataType) (data []T) {
	data = make([]T, len(b)/int(dt.Density()))
	um := UnmarshalF[T](dt)
	for i := range data {
		base := i * int(dt.Density())
		data[i] = um(b[base : base+int(dt.Density())])
	}
	return data
}

func Unmarshal[T Value](series Series) []T {
	return UnmarshalSlice[T](series.Data, series.DataType)
}

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

var ByteOrder = binary.LittleEndian
