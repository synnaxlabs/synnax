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
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
	xslices "github.com/synnaxlabs/x/slices"
	latest "github.com/synnaxlabs/x/telem/types/v0"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

// NewSeries creates a new Series from a slice of sample values. It automatically
// determines the data type from the type parameter.
func NewSeries[T Sample](data []T) Series {
	var t T
	switch any(t).(type) {
	case uint8:
		return newFixedSeries(any(data).([]uint8))
	case uint16:
		return newFixedSeries(any(data).([]uint16))
	case uint32:
		return newFixedSeries(any(data).([]uint32))
	case uint64:
		return newFixedSeries(any(data).([]uint64))
	case int8:
		return newFixedSeries(any(data).([]int8))
	case int16:
		return newFixedSeries(any(data).([]int16))
	case int32:
		return newFixedSeries(any(data).([]int32))
	case int64:
		return newFixedSeries(any(data).([]int64))
	case float32:
		return newFixedSeries(any(data).([]float32))
	case float64:
		return newFixedSeries(any(data).([]float64))
	case TimeStamp:
		return newFixedSeries(any(data).([]TimeStamp))
	case uuid.UUID:
		return newFixedSeries(any(data).([]uuid.UUID))
	case string:
		return newVariableSeries(any(data).([]string))
	case []byte:
		return newVariableSeries(any(data).([][]byte))
	}
	// degenerate case, should never hit this path.
	panic(fmt.Sprintf("unsupported sample type %T", t))
}

// NewSeriesV is a variadic version of NewSeries.
func NewSeriesV[T Sample](data ...T) Series { return NewSeries(data) }

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

func newFixedSeries[T FixedSample](data []T) Series {
	return Series{DataType: InferDataType[T](), Data: latest.MarshalFixed(data)}
}

func newVariableSeries[T VariableSample](data []T) Series {
	return Series{DataType: InferDataType[T](), Data: latest.MarshalVariable(data)}
}

// NewJSONSeries creates a new JSON Series from a slice of JSON values. It returns an
// error if the data cannot be marshalled into JSON.
func NewJSONSeries[T any](data []T) (Series, error) {
	byteSlices := make([][]byte, len(data))
	var err error
	for i, v := range data {
		if byteSlices[i], err = json.Marshal(v); err != nil {
			return Series{}, err
		}
	}
	return Series{DataType: JSONT, Data: latest.MarshalVariable(byteSlices)}, nil
}

// NewJSONSeriesV constructs a new JSON Series from an arbitrary set of JSON values,
// marshaling each one in the process. It returns an error if the data cannot be
// marshalled into JSON.
func NewJSONSeriesV[T any](data ...T) (Series, error) { return NewJSONSeries(data) }

// MarshalVariableSample wraps a single variable-length sample with a uint32 LE length
// prefix. This is useful for code that accumulates samples into a Series.Data buffer
// incrementally rather than using NewSeriesV.
func MarshalVariableSample(raw []byte) []byte {
	return latest.MarshalVariable([][]byte{raw})
}

// UnmarshalSeries converts a Series back into a slice of the specified data type. Note
// that this function does NOT check the Series' DataType, it simply unmarshals the data
// according to type T.
func UnmarshalSeries[T Sample](series Series) []T {
	var t T
	switch any(t).(type) {
	case uint8:
		return any(latest.UnmarshalFixed[uint8](series.Data)).([]T)
	case uint16:
		return any(latest.UnmarshalFixed[uint16](series.Data)).([]T)
	case uint32:
		return any(latest.UnmarshalFixed[uint32](series.Data)).([]T)
	case uint64:
		return any(latest.UnmarshalFixed[uint64](series.Data)).([]T)
	case int8:
		return any(latest.UnmarshalFixed[int8](series.Data)).([]T)
	case int16:
		return any(latest.UnmarshalFixed[int16](series.Data)).([]T)
	case int32:
		return any(latest.UnmarshalFixed[int32](series.Data)).([]T)
	case int64:
		return any(latest.UnmarshalFixed[int64](series.Data)).([]T)
	case float32:
		return any(latest.UnmarshalFixed[float32](series.Data)).([]T)
	case float64:
		return any(latest.UnmarshalFixed[float64](series.Data)).([]T)
	case TimeStamp:
		return any(latest.UnmarshalFixed[TimeStamp](series.Data)).([]T)
	case uuid.UUID:
		return any(latest.UnmarshalFixed[uuid.UUID](series.Data)).([]T)
	case string:
		return any(latest.UnmarshalVariable[string](series.Data)).([]T)
	case []byte:
		return any(latest.UnmarshalVariable[[]byte](series.Data)).([]T)
	}
	// degenerate case, should never hit this path.
	panic(fmt.Sprintf("unsupported sample type %T", t))
}

// UnmarshalJSONSeries unmarshals a JSON-encoded series into a slice of JSON values
// of the specified type T. This function does NOT check the Series' DataType, it simply
// unmarshals the data according to type T.
func UnmarshalJSONSeries[T any](s Series) ([]T, error) {
	byteSlices := UnmarshalSeries[[]byte](s)
	data := make([]T, len(byteSlices))
	for i, b := range byteSlices {
		if err := json.Unmarshal(b, &data[i]); err != nil {
			return nil, err
		}
	}
	return data, nil
}

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
// to the specified DataType. Supports numeric types, strings, TimeStamp, JSON, and
// bytes. Panics if the value cannot be converted to the target DataType.
func NewSeriesFromAny(value any, dt DataType) Series {
	switch dt {
	case Uint8T:
		return NewSeriesV(castNumeric[uint8](value))
	case Uint16T:
		return NewSeriesV(castNumeric[uint16](value))
	case Uint32T:
		return NewSeriesV(castNumeric[uint32](value))
	case Uint64T:
		return NewSeriesV(castNumeric[uint64](value))
	case Int8T:
		return NewSeriesV(castNumeric[int8](value))
	case Int16T:
		return NewSeriesV(castNumeric[int16](value))
	case Int32T:
		return NewSeriesV(castNumeric[int32](value))
	case Int64T:
		return NewSeriesV(castNumeric[int64](value))
	case Float32T:
		return NewSeriesV(castNumeric[float32](value))
	case Float64T:
		return NewSeriesV(castNumeric[float64](value))
	case TimeStampT:
		return NewSeriesV(castNumeric[TimeStamp](value))
	case UUIDT:
		return NewSeriesV(castToUUID(value))
	case StringT:
		return NewSeriesV(castToString(value))
	case BytesT:
		return NewSeriesV(castToBytes(value))
	case JSONT:
		return lo.Must(NewJSONSeriesV(value))
	default:
		panic(fmt.Sprintf("unsupported data type %s", dt))
	}
}

func castNumeric[T NumericSample](value any) T {
	switch v := value.(type) {
	case uint:
		return T(v)
	case uint8:
		return T(v)
	case uint16:
		return T(v)
	case uint32:
		return T(v)
	case uint64:
		return T(v)
	case int:
		return T(v)
	case int8:
		return T(v)
	case int16:
		return T(v)
	case int32:
		return T(v)
	case int64:
		return T(v)
	case float32:
		return T(v)
	case float64:
		return T(v)
	case TimeStamp:
		return T(v)
	case TimeSpan:
		return T(v)
	default:
		var t T
		panic(fmt.Sprintf("cannot cast %T to %T", value, t))
	}
}

func castToString(value any) string {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, TimeStamp, TimeSpan:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", value)
	}
}

func castToBytes(value any) []byte {
	switch v := value.(type) {
	case uint8:
		return []byte{v}
	case uint16:
		return latest.ByteOrder.AppendUint16(nil, v)
	case uint32:
		return latest.ByteOrder.AppendUint32(nil, v)
	case uint64:
		return latest.ByteOrder.AppendUint64(nil, v)
	case int8:
		return []byte{byte(v)}
	case int16:
		return latest.ByteOrder.AppendUint16(nil, uint16(v))
	case int32:
		return latest.ByteOrder.AppendUint32(nil, uint32(v))
	case int64:
		return latest.ByteOrder.AppendUint64(nil, uint64(v))
	case float32:
		return latest.ByteOrder.AppendUint32(nil, math.Float32bits(v))
	case float64:
		return latest.ByteOrder.AppendUint64(nil, math.Float64bits(v))
	case TimeStamp:
		return latest.ByteOrder.AppendUint64(nil, uint64(v))
	case TimeSpan:
		return latest.ByteOrder.AppendUint64(nil, uint64(v))
	case uuid.UUID:
		return v[:]
	case string:
		return []byte(v)
	case []byte:
		return v
	default:
		panic(fmt.Sprintf("cannot cast %T to []byte", value))
	}
}

func castToUUID(value any) uuid.UUID {
	switch v := value.(type) {
	case uuid.UUID:
		return v
	case string:
		return uuid.MustParse(v)
	case []byte:
		return lo.Must(uuid.FromBytes(v))
	default:
		panic(fmt.Sprintf("cannot cast %T to uuid.UUID", value))
	}
}

// ValueAt returns the numeric value at the given index in the series. ValueAt supports
// negative indices, which will be wrapped around the end of the series. This function
// cannot be used for variable density series.
func ValueAt[T FixedSample](s Series, i int) T {
	i = xslices.ConvertNegativeIndex(i, int(s.Len()))
	data := xunsafe.CastSlice[byte, T](s.Data)
	return data[i]
}

// SetValueAt sets the value at the given index in the series. SetValueAt supports
// negative indices, which will be wrapped around the end of the series. This function
// cannot be used for variable density series.
func SetValueAt[T FixedSample](s Series, i int, v T) {
	i = xslices.ConvertNegativeIndex(i, int(s.Len()))
	data := xunsafe.CastSlice[byte, T](s.Data)
	data[i] = v
}

// CopyValue copies the sample from src at the index srcIdx to the index srcIdx in src.
// dst and src must have the same DataType, and that DataType cannot be of variable
// density.
func CopyValue(dst, src Series, dstIdx, srcIdx int) {
	if dst.DataType != src.DataType || dst.DataType.IsVariable() || src.DataType.IsVariable() {
		panic("cannot copy values from non-variable series")
	}
	den := int(dst.DataType.Density())
	copy(dst.Data[dstIdx*den:(dstIdx+1)*den], src.Data[srcIdx*den:(srcIdx+1)*den])
}

// NewMultiSeries constructs a new MultiSeries from the given set of Series. The series
// are sorted by their alignment, and the data type of the series must be the same. If
// the data types are different, a panic will occur.
func NewMultiSeries(series []Series) MultiSeries {
	if len(series) == 0 {
		return MultiSeries{}
	}
	dt := series[0].DataType
	for _, s := range series {
		if s.DataType != dt {
			panic(fmt.Sprintf(
				"cannot create MultiSeries with different data types: %v != %v",
				dt,
				s.DataType,
			))
		}
	}
	slices.SortFunc(series, sortSeriesByAlignment)
	return MultiSeries{Series: series}
}

func sortSeriesByAlignment(s1, s2 Series) int { return int(s1.Alignment - s2.Alignment) }

// NewMultiSeriesV constructs a new MultiSeries from the given set of variadic Series.
// The series are sorted by their alignment, and the data type of the series must be the
// same. If the data types are different, a panic will occur.
func NewMultiSeriesV(series ...Series) MultiSeries { return NewMultiSeries(series) }

// MultiSeriesAtAlignment returns the value at the given alignment in the MultiSeries.
func MultiSeriesAtAlignment[T FixedSample](ms MultiSeries, alignment Alignment) T {
	for _, s := range ms.Series {
		if s.AlignmentBounds().Contains(alignment) {
			return ValueAt[T](s, int(alignment-s.Alignment))
		}
	}
	panic(fmt.Sprintf(
		"alignment %v out of bounds for multi series with alignment bounds %v",
		alignment,
		ms.AlignmentBounds(),
	))
}
