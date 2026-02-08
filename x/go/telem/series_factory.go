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
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/unsafe"
)

// NumericSample represents any numeric value that can be stored in a Series and have
// mathematical operations performed on it.
type NumericSample interface {
	uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 |
		float32 | float64 | TimeStamp
}

// FixedSample represents any numeric value that can be stored in a Series and has a
// fixed density.
type FixedSample interface{ NumericSample | uuid.UUID }

// VariableSample is a type that can be stored in a variable-density series.
type VariableSample interface{ []byte | string }

// Sample represents any value that can be stored in a non-JSON Series.
type Sample interface{ FixedSample | VariableSample }

// NewSeries creates a new Series from a slice of numeric values. It automatically
// determines the data type from the first element.
func NewSeries[T FixedSample](data []T) Series {
	return Series{DataType: InferDataType[T](), Data: MarshalSlice(data)}
}

// NewSeriesV is a variadic version of NewSeries that creates a new Series from
// individual numeric values.
func NewSeriesV[T FixedSample](data ...T) Series { return NewSeries(data) }

// MakeSeries allocates a new Series with the specified DataType and length. Note that
// this function allocates a length and not a capacity.
func MakeSeries(dt DataType, len int) Series {
	return Series{DataType: dt, Data: make([]byte, len*int(dt.Density()))}
}

// NewSeriesSecondsTSV creates a new Series containing TimeStamp values. All input
// timestamps are multiplied by SecondTS to convert them to the standard time unit used
// in the system.
func NewSeriesSecondsTSV(data ...TimeStamp) Series {
	for i := range data {
		data[i] *= SecondTS
	}
	return NewSeries(data)
}

// NewSeriesVariable creates a new Series from a slice of variable-density values,
// determining the data type from the type of the slice.
func NewSeriesVariable[T VariableSample](data []T) Series {
	return Series{DataType: InferDataType[T](), Data: MarshalVariable(data)}
}

// NewSeriesVariableV is a variadic version of NewSeriesVariable that creates a new
// Series from individual variable-density values.
func NewSeriesVariableV[T VariableSample](data ...T) Series {
	return NewSeriesVariable(data)
}

// NewSeriesJSON creates a new Series from a slice of JSON values.
func NewSeriesJSON[T any](data []T) (Series, error) {
	bytes, err := MarshalJSON(data)
	if err != nil {
		return Series{}, err
	}
	return Series{DataType: JSONT, Data: bytes}, nil
}

// NewSeriesJSONV constructs a new series from an arbitrary set of JSON values,
// marshaling each one in the process.
func NewSeriesJSONV[T any](data ...T) (Series, error) { return NewSeriesJSON(data) }

const newLine = '\n'

// MarshalJSON marshals a slice of JSON values into a byte slice, returning any errors
// encountered while marshalling.
func MarshalJSON[T any](data []T) ([]byte, error) {
	byteSlices := make([][]byte, len(data))
	var err error
	for i, v := range data {
		if byteSlices[i], err = json.Marshal(v); err != nil {
			return nil, err
		}
	}
	return MarshalVariable(byteSlices), nil
}

// MarshalVariable marshals a slice of variable-density values into a byte slice.
func MarshalVariable[T VariableSample](data []T) []byte {
	total := lo.SumBy(data, func(v T) int64 { return int64(len(v)) + 1 })
	b := make([]byte, total)
	offset := 0
	for _, d := range data {
		copy(b[offset:], d)
		b[offset+len(d)] = newLine
		offset += len(d) + 1
	}
	return b
}

// UnmarshalVariable unmarshals a byte slice into a slice of variable-density values.
func UnmarshalVariable[T VariableSample](b []byte) []T {
	var (
		offset int
		data   []T
	)
	for offset < len(b) {
		end := offset
		for b[end] != newLine {
			end++
		}
		data = append(data, T(b[offset:end]))
		offset = end + 1
	}
	return data
}

// UnmarshalJSON unmarshals a JSON-encoded byte slice into a slice of JSON values of the
// specified type T. It returns an error encountered during unmarshalling.
func UnmarshalJSON[T any](b []byte) ([]T, error) {
	byteSlices := UnmarshalVariable[[]byte](b)
	data := make([]T, len(byteSlices))
	for i, b := range byteSlices {
		if err := json.Unmarshal(b, &data[i]); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// MarshalSlice converts a slice of numeric values into a byte slice according to the
// specified DataType.
func MarshalSlice[T FixedSample](data []T) []byte {
	dt := InferDataType[T]()
	b := make([]byte, dt.Density().Size(int64(len(data))))
	typedData := unsafe.CastSlice[byte, T](b)
	copy(typedData, data)
	return b
}

// UnmarshalSlice converts a byte slice back into a slice of numeric values according to
// the specified type T.
func UnmarshalSlice[T FixedSample](b []byte) []T { return unsafe.CastSlice[byte, T](b) }

// UnmarshalSeries converts a Series' data back into a slice of the specified type T.
func UnmarshalSeries[T FixedSample](series Series) []T {
	return UnmarshalSlice[T](series.Data)
}

// ByteOrder is the standard order for encoding/decoding numeric values across the
// Synnax telemetry ecosystem.
var ByteOrder = binary.LittleEndian

// Arrange creates a new Series containing count values starting from start, with each
// subsequent value incremented by spacing. For example, Arrange(0, 5, 2) produces [0,
// 2, 4, 6, 8]. Panics if count is less than or equal to 0.
func Arrange[T NumericSample](start T, count int, spacing T) Series {
	data := make([]T, count)
	for i := range count {
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
		return NewSeriesVariableV(castToString(value))
	case JSONT:
		return castToJSON(value)
	case BytesT:
		return NewSeriesVariableV(castToBytes(value))
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
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8, TimeStamp:
		return fmt.Sprintf("%d", v)
	case float64, float32:
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func castToJSON(value any) Series {
	switch v := value.(type) {
	case string:
		return Series{DataType: JSONT, Data: MarshalVariable([]string{v})}
	case []byte:
		return Series{DataType: JSONT, Data: MarshalVariable([]string{string(v)})}
	default:
		return Series{DataType: JSONT, Data: lo.Must(MarshalJSON([]any{value}))}
	}
}

func castToBytes(value any) []byte {
	switch v := value.(type) {
	case []byte:
		return append(v, newLine)
	default:
		str := castToString(value)
		return append([]byte(str), newLine)
	}
}
