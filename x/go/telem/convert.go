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
	"math"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/unsafe"
)

// ErrOutOfRange is the cause of each OutOfRangeError.
var ErrOutOfRange = errors.New("sample out of range")

// OutOfRangeError is returned by Convert and ValueAs for a sample that a data type
// cannot hold. It wraps ErrOutOfRange.
type OutOfRangeError struct {
	// Value is the sample.
	Value any
	// From is the data type of the series that holds the sample.
	From DataType
	// To is the data type that cannot hold the sample.
	To DataType
}

// Error implements error.
func (e OutOfRangeError) Error() string {
	return fmt.Sprintf(
		"value %v of %s series is out of range of %s",
		e.Value,
		withArticle(e.From),
		e.To,
	)
}

// Unwrap returns ErrOutOfRange.
func (e OutOfRangeError) Unwrap() error { return ErrOutOfRange }

// ConversionError is returned by Convert and ValueAs when the samples of one data type
// do not convert to another.
type ConversionError struct {
	// From is the data type of the series.
	From DataType
	// To is the data type that the series does not convert to.
	To DataType
}

// Error implements error.
func (e ConversionError) Error() string {
	return fmt.Sprintf("cannot convert %s series to %s", withArticle(e.From), e.To)
}

// withArticle returns dt after the indefinite article of its name.
func withArticle(dt DataType) string {
	if strings.HasPrefix(string(dt), "int") {
		return "an " + string(dt)
	}
	return "a " + string(dt)
}

// Convert returns s with its samples converted to dt and every other field kept.
// Numbers and timestamps convert as a Go conversion does, with a float truncated toward
// zero for an integer type. A boolean is 1 or 0 as a number, and a number is true as a
// boolean unless it is 0. It returns s when dt is its data type. It returns an
// OutOfRangeError when a sample is out of the range of dt or is a NaN for an integer
// type, and a ConversionError when either type is not a number, a timestamp, or a
// boolean.
func Convert(s Series, dt DataType) (Series, error) {
	if s.DataType == dt {
		return s, nil
	}
	t, ok := targetOf(dt)
	if _, from := targetOf(s.DataType); !ok || !from {
		return Series{}, ConversionError{From: s.DataType, To: dt}
	}
	var (
		data []byte
		err  error
	)
	switch s.DataType {
	case Uint8T, BooleanT:
		data, err = convertFrom[uint8](s.Data, s.DataType, t)
	case Uint16T:
		data, err = convertFrom[uint16](s.Data, s.DataType, t)
	case Uint32T:
		data, err = convertFrom[uint32](s.Data, s.DataType, t)
	case Uint64T:
		data, err = convertFrom[uint64](s.Data, s.DataType, t)
	case Int8T:
		data, err = convertFrom[int8](s.Data, s.DataType, t)
	case Int16T:
		data, err = convertFrom[int16](s.Data, s.DataType, t)
	case Int32T:
		data, err = convertFrom[int32](s.Data, s.DataType, t)
	case Int64T, TimestampT:
		data, err = convertFrom[int64](s.Data, s.DataType, t)
	case Float32T:
		data, err = convertFrom[float32](s.Data, s.DataType, t)
	case Float64T:
		data, err = convertFrom[float64](s.Data, s.DataType, t)
	case UnknownT, UUIDT, StringT, BytesT, JSONT:
	}
	if err != nil {
		return Series{}, err
	}
	s.DataType, s.Data = dt, data
	return s, nil
}

// ValueAs returns sample i of s as T, converted as Convert converts it, with no
// allocation. A negative i counts from the end. It returns the errors that Convert
// returns.
func ValueAs[T NumericSample](s Series, i int) (T, error) {
	dt := InferDataType[T]()
	if s.DataType == dt {
		return s.ValueAt[T](i), nil
	}
	t, ok := targetOf(dt)
	if !ok {
		return 0, ConversionError{From: s.DataType, To: dt}
	}
	switch s.DataType {
	case Uint8T, BooleanT:
		return valueAs[uint8, T](s.ValueAt[uint8](i), s.DataType, t)
	case Uint16T:
		return valueAs[uint16, T](s.ValueAt[uint16](i), s.DataType, t)
	case Uint32T:
		return valueAs[uint32, T](s.ValueAt[uint32](i), s.DataType, t)
	case Uint64T:
		return valueAs[uint64, T](s.ValueAt[uint64](i), s.DataType, t)
	case Int8T:
		return valueAs[int8, T](s.ValueAt[int8](i), s.DataType, t)
	case Int16T:
		return valueAs[int16, T](s.ValueAt[int16](i), s.DataType, t)
	case Int32T:
		return valueAs[int32, T](s.ValueAt[int32](i), s.DataType, t)
	case Int64T, TimestampT:
		return valueAs[int64, T](s.ValueAt[int64](i), s.DataType, t)
	case Float32T:
		return valueAs[float32, T](s.ValueAt[float32](i), s.DataType, t)
	case Float64T:
		return valueAs[float64, T](s.ValueAt[float64](i), s.DataType, t)
	case UnknownT, UUIDT, StringT, BytesT, JSONT:
	}
	return 0, ConversionError{From: s.DataType, To: dt}
}

// target is a type that numbers convert to: its kind ('u', 'i', 'f', or 'b' for a
// boolean), its bit width, and its data type.
type target struct {
	kind     byte
	bits     int
	dataType DataType
}

func targetOf(dt DataType) (target, bool) {
	switch dt {
	case Uint8T:
		return target{'u', 8, dt}, true
	case Uint16T:
		return target{'u', 16, dt}, true
	case Uint32T:
		return target{'u', 32, dt}, true
	case Uint64T:
		return target{'u', 64, dt}, true
	case Int8T:
		return target{'i', 8, dt}, true
	case Int16T:
		return target{'i', 16, dt}, true
	case Int32T:
		return target{'i', 32, dt}, true
	case Int64T, TimestampT:
		return target{'i', 64, dt}, true
	case Float32T:
		return target{'f', 32, dt}, true
	case Float64T:
		return target{'f', 64, dt}, true
	case BooleanT:
		return target{'b', 8, dt}, true
	case UnknownT, UUIDT, StringT, BytesT, JSONT:
	}
	return target{}, false
}

// kindOf returns the kind of the samples of a number type: 'u', 'i', or 'f'.
func kindOf(dt DataType) byte {
	switch dt {
	case Int8T, Int16T, Int32T, Int64T, TimestampT:
		return 'i'
	case Float32T, Float64T:
		return 'f'
	case Uint8T, Uint16T, Uint32T, Uint64T, BooleanT, UnknownT, UUIDT, StringT, BytesT,
		JSONT:
	}
	return 'u'
}

func (t target) holdsUint(u uint64) bool {
	switch t.kind {
	case 'u':
		return t.bits == 64 || u < 1<<t.bits
	case 'i':
		return u < 1<<(t.bits-1)
	default:
		return true
	}
}

func (t target) holdsInt(i int64) bool {
	switch t.kind {
	case 'u':
		return i >= 0 && t.holdsUint(uint64(i))
	case 'i':
		return t.bits == 64 || (i >= -(1<<(t.bits-1)) && i < 1<<(t.bits-1))
	default:
		return true
	}
}

func (t target) holdsFloat(f float64) bool {
	trunc := math.Trunc(f)
	switch t.kind {
	case 'u':
		return !math.IsNaN(f) && trunc >= 0 && trunc < math.Ldexp(1, t.bits)
	case 'i':
		limit := math.Ldexp(1, t.bits-1)
		return !math.IsNaN(f) && trunc >= -limit && trunc < limit
	case 'f':
		return t.bits == 64 || math.IsNaN(f) || math.IsInf(f, 0) ||
			math.Abs(f) <= math.MaxFloat32
	default:
		return true
	}
}

// holds reports whether t holds v, a sample of a series of the data type from.
func holds[S NumericSample](t target, v S, from DataType) bool {
	switch kindOf(from) {
	case 'i':
		return t.holdsInt(int64(v))
	case 'f':
		return t.holdsFloat(float64(v))
	default:
		return t.holdsUint(uint64(v))
	}
}

func outOfRange[S NumericSample](v S, from DataType, t target) error {
	return OutOfRangeError{Value: v, From: from, To: t.dataType}
}

func valueAs[S, T NumericSample](v S, from DataType, t target) (T, error) {
	if !holds(t, v, from) {
		return 0, outOfRange(v, from, t)
	}
	return T(v), nil
}

func convertFrom[S NumericSample](
	data []byte,
	from DataType,
	t target,
) ([]byte, error) {
	src := unmarshalFixed[S](data)
	for _, v := range src {
		if !holds(t, v, from) {
			return nil, outOfRange(v, from, t)
		}
	}
	switch t.dataType {
	case Uint8T:
		return convertTo[S, uint8](src), nil
	case Uint16T:
		return convertTo[S, uint16](src), nil
	case Uint32T:
		return convertTo[S, uint32](src), nil
	case Uint64T:
		return convertTo[S, uint64](src), nil
	case Int8T:
		return convertTo[S, int8](src), nil
	case Int16T:
		return convertTo[S, int16](src), nil
	case Int32T:
		return convertTo[S, int32](src), nil
	case Int64T, TimestampT:
		return convertTo[S, int64](src), nil
	case Float32T:
		return convertTo[S, float32](src), nil
	case Float64T:
		return convertTo[S, float64](src), nil
	case BooleanT:
		dst := make([]byte, len(src))
		for i, v := range src {
			if v != 0 {
				dst[i] = 1
			}
		}
		return dst, nil
	case UnknownT, UUIDT, StringT, BytesT, JSONT:
	}
	return nil, ConversionError{From: from, To: t.dataType}
}

func convertTo[S, T NumericSample](src []S) []byte {
	b := make([]byte, len(src)*int(InferDataType[T]().Density()))
	dst := unsafe.CastSlice[byte, T](b)
	for i, v := range src {
		dst[i] = T(v)
	}
	return b
}
