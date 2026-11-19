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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/unsafe"
)

// Convert returns s with its samples converted to dt and every other field kept.
// Numbers and timestamps convert as a Go conversion does, a boolean is 1 or 0 as a
// number, and a number is true as a boolean unless it is 0. It returns s when dt is
// its data type, and an error when either type is not a number, a timestamp, or a
// boolean.
func Convert(s Series, dt DataType) (Series, error) {
	if s.DataType == dt {
		return s, nil
	}
	var (
		data []byte
		ok   bool
	)
	switch s.DataType {
	case Uint8T, BooleanT:
		data, ok = convertFrom[uint8](s.Data, dt)
	case Uint16T:
		data, ok = convertFrom[uint16](s.Data, dt)
	case Uint32T:
		data, ok = convertFrom[uint32](s.Data, dt)
	case Uint64T:
		data, ok = convertFrom[uint64](s.Data, dt)
	case Int8T:
		data, ok = convertFrom[int8](s.Data, dt)
	case Int16T:
		data, ok = convertFrom[int16](s.Data, dt)
	case Int32T:
		data, ok = convertFrom[int32](s.Data, dt)
	case Int64T, TimestampT:
		data, ok = convertFrom[int64](s.Data, dt)
	case Float32T:
		data, ok = convertFrom[float32](s.Data, dt)
	case Float64T:
		data, ok = convertFrom[float64](s.Data, dt)
	case UnknownT, UUIDT, StringT, BytesT, JSONT:
	}
	if !ok {
		return Series{}, errors.Newf("cannot convert a %s series to %s", s.DataType, dt)
	}
	s.DataType, s.Data = dt, data
	return s, nil
}

func convertFrom[S NumericSample](data []byte, dt DataType) ([]byte, bool) {
	src := unmarshalFixed[S](data)
	switch dt {
	case Uint8T:
		return convertTo[S, uint8](src), true
	case Uint16T:
		return convertTo[S, uint16](src), true
	case Uint32T:
		return convertTo[S, uint32](src), true
	case Uint64T:
		return convertTo[S, uint64](src), true
	case Int8T:
		return convertTo[S, int8](src), true
	case Int16T:
		return convertTo[S, int16](src), true
	case Int32T:
		return convertTo[S, int32](src), true
	case Int64T, TimestampT:
		return convertTo[S, int64](src), true
	case Float32T:
		return convertTo[S, float32](src), true
	case Float64T:
		return convertTo[S, float64](src), true
	case BooleanT:
		dst := make([]byte, len(src))
		for i, v := range src {
			if v != 0 {
				dst[i] = 1
			}
		}
		return dst, true
	case UnknownT, UUIDT, StringT, BytesT, JSONT:
	}
	return nil, false
}

func convertTo[S, T NumericSample](src []S) []byte {
	b := make([]byte, len(src)*int(InferDataType[T]().Density()))
	dst := unsafe.CastSlice[byte, T](b)
	for i, v := range src {
		dst[i] = T(v)
	}
	return b
}
