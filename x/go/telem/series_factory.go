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

	"github.com/samber/lo"
	xbinary "github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/unsafe"
)

// Sample represents any numeric value that can be stored in a Series.
// It must satisfy the Sample interface.
type Sample = types.SizedNumeric

// NewSeries creates a new Series from a slice of numeric values. It automatically
// determines the data type from the first element.
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
func MakeSeries(dt DataType, len int) Series {
	return Series{DataType: dt, Data: make([]byte, len*int(dt.Density()))}
}

// NewSeriesSecondsTSV creates a new Series containing TimeStamp values. All input timestamps
// are multiplied by SecondTS to convert them to the standard time unit used in the
// system.
func NewSeriesSecondsTSV(data ...TimeStamp) Series {
	for i := range data {
		data[i] *= SecondTS
	}
	return Series{DataType: TimeStampT, Data: MarshalSlice(data)}
}

// NewSeriesStrings creates a new Series from a slice of strings. The strings are stored with
// newline characters as delimiters.
func NewSeriesStrings(data []string) Series {
	return Series{DataType: StringT, Data: MarshalStrings(data, StringT)}
}

// NewSeriesStringsV is a variadic version of NewSeriesStrings that creates a new Series from
// individual string values.
func NewSeriesStringsV(data ...string) Series { return NewSeriesStrings(data) }

// NewSeriesStaticJSONV constructs a new series from an arbitrary set of JSON values,
// marshaling each one in the process.
func NewSeriesStaticJSONV[T any](data ...T) (series Series) {
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
func UnmarshalStrings(b []byte) []string {
	var (
		offset = 0
		data   []string
	)
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
	dt := InferDataType[T]()
	b := make([]byte, dt.Density().Size(int64(len(data))))
	typedData := unsafe.CastSlice[byte, T](b)
	copy(typedData, data)
	return b
}

// UnmarshalSlice converts a byte slice back into a slice of numeric values according to
// the specified DataType.
func UnmarshalSlice[T Sample](b []byte, dt DataType) []T {
	return unsafe.CastSlice[byte, T](b)
}

// UnmarshalSeries converts a Series' data back into a slice of the original type.
func UnmarshalSeries[T Sample](series Series) []T {
	return unsafe.CastSlice[byte, T](series.Data)
}

// ByteOrder is the standard order for encoding/decoding numeric values across
// the Synnax telemetry ecosystem.
var ByteOrder = binary.LittleEndian

// Arrange creates a new Series containing count values starting from start, with each
// subsequent value incremented by spacing. For example, Arrange(0, 5, 2) produces [0,
// 2, 4, 6, 8]. Panics if count is less than or equal to 0.
func Arrange[T Sample](start T, count int, spacing T) Series {
	data := make([]T, count)
	for i := 0; i < count; i++ {
		data[i] = start + T(i)*spacing
	}
	return NewSeries(data)
}

// NewSeriesFromAny creates a single-value Series from a value of type any, casting it
// to the specified DataType. This function preserves numeric precision by avoiding
// unnecessary intermediate conversions. Supports numeric types, strings, TimeStamp,
// JSON, and bytes. Panics if the value cannot be converted to the target DataType.
func NewSeriesFromAny(value any, dt DataType) Series {
	switch dt {
	case Int64T:
		return NewSeriesV(castToInt64(value))
	case Int32T:
		return NewSeriesV(castToInt32(value))
	case Int16T:
		return NewSeriesV(castToInt16(value))
	case Int8T:
		return NewSeriesV(castToInt8(value))
	case Uint64T:
		return NewSeriesV(castToUint64(value))
	case Uint32T:
		return NewSeriesV(castToUint32(value))
	case Uint16T:
		return NewSeriesV(castToUint16(value))
	case Uint8T:
		return NewSeriesV(castToUint8(value))
	case Float64T:
		return NewSeriesV(castToFloat64(value))
	case Float32T:
		return NewSeriesV(castToFloat32(value))
	case TimeStampT:
		return NewSeriesV(castToTimeStamp(value))
	case StringT:
		return NewSeriesStringsV(castToString(value))
	case JSONT:
		return castToJSON(value)
	case BytesT:
		return castToBytes(value)
	default:
		panic(fmt.Sprintf("unsupported data type %s", dt))
	}
}

func castToInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint:
		return int64(v)
	case uint64:
		return int64(v)
	case uint32:
		return int64(v)
	case uint16:
		return int64(v)
	case uint8:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case TimeStamp:
		return int64(v)
	case string:
		panic("cannot cast string to int64")
	default:
		panic(fmt.Sprintf("cannot cast %T to int64", value))
	}
}

func castToInt32(value any) int32 { return int32(castToInt64(value)) }

func castToInt16(value any) int16 { return int16(castToInt64(value)) }

func castToInt8(value any) int8 { return int8(castToInt64(value)) }

func castToUint64(value any) uint64 {
	switch v := value.(type) {
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	case int32:
		return uint64(v)
	case int16:
		return uint64(v)
	case int8:
		return uint64(v)
	case uint:
		return uint64(v)
	case uint64:
		return v
	case uint32:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint8:
		return uint64(v)
	case float64:
		return uint64(v)
	case float32:
		return uint64(v)
	case TimeStamp:
		return uint64(v)
	case string:
		panic("cannot cast string to uint64")
	default:
		panic(fmt.Sprintf("cannot cast %T to uint64", value))
	}
}

func castToUint32(value any) uint32 { return uint32(castToUint64(value)) }

func castToUint16(value any) uint16 { return uint16(castToUint64(value)) }

func castToUint8(value any) uint8 { return uint8(castToUint64(value)) }

func castToFloat64(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	case float64:
		return v
	case float32:
		return float64(v)
	case TimeStamp:
		return float64(v)
	case string:
		panic("cannot cast string to float64")
	default:
		panic(fmt.Sprintf("cannot cast %T to float64", value))
	}
}

func castToFloat32(value any) float32 { return float32(castToFloat64(value)) }

func castToTimeStamp(value any) TimeStamp {
	switch v := value.(type) {
	case TimeStamp:
		return v
	case string:
		panic("cannot cast string to TimeStamp")
	default:
		return TimeStamp(castToInt64(value))
	}
}

func castToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int16:
		return fmt.Sprintf("%d", v)
	case int8:
		return fmt.Sprintf("%d", v)
	case uint:
		return fmt.Sprintf("%d", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	case uint32:
		return fmt.Sprintf("%d", v)
	case uint16:
		return fmt.Sprintf("%d", v)
	case uint8:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case float32:
		return fmt.Sprintf("%g", v)
	case TimeStamp:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func castToJSON(value any) Series {
	switch v := value.(type) {
	case string:
		return Series{DataType: JSONT, Data: MarshalStrings([]string{v}, JSONT)}
	case []byte:
		return Series{DataType: JSONT, Data: MarshalStrings([]string{string(v)}, JSONT)}
	default:
		jsonStr := xbinary.MustEncodeJSONToString(value)
		return Series{DataType: JSONT, Data: MarshalStrings([]string{jsonStr}, JSONT)}
	}
}

func castToBytes(value any) Series {
	switch v := value.(type) {
	case []byte:
		return Series{DataType: BytesT, Data: append(v, newLine)}
	case string:
		return Series{DataType: BytesT, Data: append([]byte(v), newLine)}
	default:
		str := castToString(value)
		return Series{DataType: BytesT, Data: append([]byte(str), newLine)}
	}
}
