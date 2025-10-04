// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package value

import (
	"math"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/telem"
)

type Value struct {
	Type  ir.Type
	Value uint64
}

// PutUint64 stores a uint64 value
func (v Value) PutUint64(val uint64) Value {
	v.Value = val
	v.Type = ir.U64{}
	return v
}

// GetUint64 retrieves the stored value as uint64
func (v Value) GetUint64() uint64 {
	return v.Value
}

// PutUint32 stores a uint32 value
func (v Value) PutUint32(val uint32) Value {
	v.Value = uint64(val)
	v.Type = ir.U32{}
	return v
}

// GetUint32 retrieves the stored value as uint32
func (v Value) GetUint32() uint32 {
	return uint32(v.Value)
}

// PutUint16 stores a uint16 value
func (v Value) PutUint16(val uint16) Value {
	v.Value = uint64(val)
	v.Type = ir.U16{}
	return v
}

// GetUint16 retrieves the stored value as uint16
func (v Value) GetUint16() uint16 {
	return uint16(v.Value)
}

// PutUint8 stores a uint8 value
func (v Value) PutUint8(val uint8) Value {
	v.Value = uint64(val)
	v.Type = ir.U8{}
	return v
}

// GetUint8 retrieves the stored value as uint8
func (v Value) GetUint8() uint8 {
	return uint8(v.Value)
}

// PutInt64 stores an int64 value
func (v Value) PutInt64(val int64) Value {
	v.Value = uint64(val)
	v.Type = ir.I64{}
	return v
}

// GetInt64 retrieves the stored value as int64
func (v Value) GetInt64() int64 {
	return int64(v.Value)
}

// PutInt32 stores an int32 value
func (v Value) PutInt32(val int32) Value {
	v.Value = uint64(val)
	v.Type = ir.I32{}
	return v
}

// GetInt32 retrieves the stored value as int32
func (v Value) GetInt32() int32 {
	return int32(v.Value)
}

// PutInt16 stores an int16 value
func (v Value) PutInt16(val int16) Value {
	v.Value = uint64(val)
	v.Type = ir.I16{}
	return v
}

// GetInt16 retrieves the stored value as int16
func (v Value) GetInt16() int16 {
	return int16(v.Value)
}

// PutInt8 stores an int8 value
func (v Value) PutInt8(val int8) Value {
	v.Value = uint64(val)
	v.Type = ir.I8{}
	return v
}

// GetInt8 retrieves the stored value as int8
func (v Value) GetInt8() int8 {
	return int8(v.Value)
}

// PutFloat32 stores a float32 value
func (v Value) PutFloat32(val float32) Value {
	v.Value = uint64(math.Float32bits(val))
	v.Type = ir.F32{}
	return v
}

// GetFloat32 retrieves the stored value as float32
func (v Value) GetFloat32() float32 {
	return math.Float32frombits(uint32(v.Value))
}

// PutFloat64 stores a float64 value
func (v Value) PutFloat64(val float64) Value {
	v.Value = math.Float64bits(val)
	v.Type = ir.F64{}
	return v
}

// GetFloat64 retrieves the stored value as float64
func (v Value) GetFloat64() float64 {
	return math.Float64frombits(v.Value)
}

// Put stores an arbitrary value using type switching
func (v Value) Put(val any) Value {
	switch typed := val.(type) {
	case uint64:
		return v.PutUint64(typed)
	case uint32:
		return v.PutUint32(typed)
	case uint16:
		return v.PutUint16(typed)
	case uint8:
		return v.PutUint8(typed)
	case uint:
		return v.PutUint64(uint64(typed))
	case int64:
		return v.PutInt64(typed)
	case int32:
		return v.PutInt32(typed)
	case int16:
		return v.PutInt16(typed)
	case int8:
		return v.PutInt8(typed)
	case int:
		return v.PutInt64(int64(typed))
	case float64:
		return v.PutFloat64(typed)
	case float32:
		return v.PutFloat32(typed)
	case bool:
		if typed {
			return v.PutUint8(1)
		}
		return v.PutUint8(0)
	default:
		v.Value = 0
		return v
	}
}

func (v Value) Get() any {
	switch v.Type.(type) {
	case ir.U64:
		return v.GetUint64()
	case ir.U32:
		return v.GetUint32()
	case ir.U16:
		return v.GetUint16()
	case ir.U8:
		return v.GetUint8()
	case ir.I64:
		return v.GetInt64()
	case ir.I32:
		return v.GetInt32()
	case ir.I16:
		return v.GetInt16()
	case ir.I8:
		return v.GetInt8()
	case ir.F64:
		return v.GetFloat64()
	case ir.F32:
		return v.GetFloat32()
	default:
		// Default to uint64 for unknown types
		return v.GetUint64()
	}
}

func DataTypeToIRType(dt telem.DataType) ir.Type {
	switch dt {
	case telem.Uint64T:
		return ir.U64{}
	case telem.Uint32T:
		return ir.U32{}
	case telem.Uint16T:
		return ir.U16{}
	case telem.Uint8T:
		return ir.U8{}
	case telem.Int64T:
		return ir.I64{}
	case telem.Int32T:
		return ir.I32{}
	case telem.Int16T:
		return ir.I16{}
	case telem.Int8T:
		return ir.I8{}
	case telem.Float64T:
		return ir.F64{}
	case telem.Float32T:
		return ir.F32{}
	case telem.StringT:
		return ir.String{}
	case telem.TimeStampT:
		return ir.TimeStamp{}
	default:
		// For unknown types, return nil (untyped)
		// The actual type should be determined by context
		return nil
	}
}

func FromSeries(s telem.Series) []Value {
	length := int(s.Len())
	values := make([]Value, length)

	for i := 0; i < length; i++ {
		v := Value{Type: DataTypeToIRType(s.DataType)}

		switch s.DataType {
		case telem.Uint64T:
			v = v.PutUint64(telem.ValueAt[uint64](s, i))
		case telem.Uint32T:
			v = v.PutUint32(telem.ValueAt[uint32](s, i))
		case telem.Uint16T:
			v = v.PutUint16(telem.ValueAt[uint16](s, i))
		case telem.Uint8T:
			v = v.PutUint8(telem.ValueAt[uint8](s, i))
		case telem.Int64T:
			v = v.PutInt64(telem.ValueAt[int64](s, i))
		case telem.Int32T:
			v = v.PutInt32(telem.ValueAt[int32](s, i))
		case telem.Int16T:
			v = v.PutInt16(telem.ValueAt[int16](s, i))
		case telem.Int8T:
			v = v.PutInt8(telem.ValueAt[int8](s, i))
		case telem.Float64T:
			v = v.PutFloat64(telem.ValueAt[float64](s, i))
		case telem.Float32T:
			v = v.PutFloat32(telem.ValueAt[float32](s, i))
		}

		values[i] = v
	}

	return values
}

func ToSeries(values []Value, dt telem.DataType) telem.Series {
	if len(values) == 0 {
		return telem.Series{}
	}

	switch dt {
	case telem.Uint64T:
		data := make([]uint64, len(values))
		for i, v := range values {
			data[i] = v.toUint64()
		}
		return telem.NewSeries(data)
	case telem.Uint32T:
		data := make([]uint32, len(values))
		for i, v := range values {
			data[i] = uint32(v.toUint64())
		}
		return telem.NewSeries(data)
	case telem.Uint16T:
		data := make([]uint16, len(values))
		for i, v := range values {
			data[i] = uint16(v.toUint64())
		}
		return telem.NewSeries(data)
	case telem.Uint8T:
		data := make([]uint8, len(values))
		for i, v := range values {
			data[i] = uint8(v.toUint64())
		}
		return telem.NewSeries(data)
	case telem.Int64T:
		data := make([]int64, len(values))
		for i, v := range values {
			data[i] = v.toInt64()
		}
		return telem.NewSeries(data)
	case telem.Int32T:
		data := make([]int32, len(values))
		for i, v := range values {
			data[i] = v.toInt32()
		}
		return telem.NewSeries(data)
	case telem.Int16T:
		data := make([]int16, len(values))
		for i, v := range values {
			data[i] = v.toInt16()
		}
		return telem.NewSeries(data)
	case telem.Int8T:
		data := make([]int8, len(values))
		for i, v := range values {
			data[i] = v.toInt8()
		}
		return telem.NewSeries(data)
	case telem.Float64T:
		data := make([]float64, len(values))
		for i, v := range values {
			data[i] = v.toFloat64()
		}
		return telem.NewSeries(data)
	case telem.Float32T:
		data := make([]float32, len(values))
		for i, v := range values {
			data[i] = v.toFloat32()
		}
		return telem.NewSeries(data)
	default:
		return telem.Series{}
	}
}

func (v Value) Eq(other Value) bool {
	return v.compare(other) == 0
}

func (v Value) Gt(other Value) bool {
	return v.compare(other) > 0
}

func (v Value) Ge(other Value) bool {
	return v.compare(other) >= 0
}

func (v Value) Lt(other Value) bool {
	return v.compare(other) < 0
}

func (v Value) Le(other Value) bool {
	return v.compare(other) <= 0
}

func (v Value) compare(other Value) int {
	switch v.Type.(type) {
	case ir.F64:
		left := v.GetFloat64()
		right := other.toFloat64()
		if left < right {
			return -1
		} else if left > right {
			return 1
		}
		return 0
	case ir.F32:
		left := v.GetFloat32()
		right := other.toFloat32()
		if left < right {
			return -1
		} else if left > right {
			return 1
		}
		return 0
	case ir.I64:
		left := v.GetInt64()
		right := other.toInt64()
		if left < right {
			return -1
		} else if left > right {
			return 1
		}
		return 0
	case ir.I32:
		left := v.GetInt32()
		right := other.toInt32()
		if left < right {
			return -1
		} else if left > right {
			return 1
		}
		return 0
	case ir.I16:
		left := v.GetInt16()
		right := other.toInt16()
		if left < right {
			return -1
		} else if left > right {
			return 1
		}
		return 0
	case ir.I8:
		left := v.GetInt8()
		right := other.toInt8()
		if left < right {
			return -1
		} else if left > right {
			return 1
		}
		return 0
	default: // U64, U32, U16, U8
		left := v.GetUint64()
		right := other.toUint64()
		if left < right {
			return -1
		} else if left > right {
			return 1
		}
		return 0
	}
}

// Type coercion helpers

func (v Value) toFloat64() float64 {
	switch v.Type.(type) {
	case ir.F64:
		return v.GetFloat64()
	case ir.F32:
		return float64(v.GetFloat32())
	case ir.I64:
		return float64(v.GetInt64())
	case ir.I32:
		return float64(v.GetInt32())
	case ir.I16:
		return float64(v.GetInt16())
	case ir.I8:
		return float64(v.GetInt8())
	default:
		return float64(v.GetUint64())
	}
}

func (v Value) toFloat32() float32 {
	switch v.Type.(type) {
	case ir.F64:
		return float32(v.GetFloat64())
	case ir.F32:
		return v.GetFloat32()
	case ir.I64:
		return float32(v.GetInt64())
	case ir.I32:
		return float32(v.GetInt32())
	case ir.I16:
		return float32(v.GetInt16())
	case ir.I8:
		return float32(v.GetInt8())
	default:
		return float32(v.GetUint64())
	}
}

func (v Value) toInt64() int64 {
	switch v.Type.(type) {
	case ir.F64:
		return int64(v.GetFloat64())
	case ir.F32:
		return int64(v.GetFloat32())
	case ir.I64:
		return v.GetInt64()
	case ir.I32:
		return int64(v.GetInt32())
	case ir.I16:
		return int64(v.GetInt16())
	case ir.I8:
		return int64(v.GetInt8())
	default:
		return int64(v.GetUint64())
	}
}

func (v Value) toInt32() int32 {
	switch v.Type.(type) {
	case ir.F64:
		return int32(v.GetFloat64())
	case ir.F32:
		return int32(v.GetFloat32())
	case ir.I64:
		return int32(v.GetInt64())
	case ir.I32:
		return v.GetInt32()
	case ir.I16:
		return int32(v.GetInt16())
	case ir.I8:
		return int32(v.GetInt8())
	default:
		return int32(v.GetUint64())
	}
}

func (v Value) toInt16() int16 {
	switch v.Type.(type) {
	case ir.F64:
		return int16(v.GetFloat64())
	case ir.F32:
		return int16(v.GetFloat32())
	case ir.I64:
		return int16(v.GetInt64())
	case ir.I32:
		return int16(v.GetInt32())
	case ir.I16:
		return v.GetInt16()
	case ir.I8:
		return int16(v.GetInt8())
	default:
		return int16(v.GetUint64())
	}
}

func (v Value) toInt8() int8 {
	switch v.Type.(type) {
	case ir.F64:
		return int8(v.GetFloat64())
	case ir.F32:
		return int8(v.GetFloat32())
	case ir.I64:
		return int8(v.GetInt64())
	case ir.I32:
		return int8(v.GetInt32())
	case ir.I16:
		return int8(v.GetInt16())
	case ir.I8:
		return v.GetInt8()
	default:
		return int8(v.GetUint64())
	}
}

func (v Value) toUint64() uint64 {
	switch v.Type.(type) {
	case ir.F64:
		return uint64(v.GetFloat64())
	case ir.F32:
		return uint64(v.GetFloat32())
	case ir.I64:
		return uint64(v.GetInt64())
	case ir.I32:
		return uint64(v.GetInt32())
	case ir.I16:
		return uint64(v.GetInt16())
	case ir.I8:
		return uint64(v.GetInt8())
	default:
		return v.GetUint64()
	}
}

func (v Value) Add(other Value) Value {
	result := Value{Type: v.Type}

	switch v.Type.(type) {
	case ir.F64:
		return result.PutFloat64(v.GetFloat64() + other.toFloat64())
	case ir.F32:
		return result.PutFloat32(v.GetFloat32() + other.toFloat32())
	case ir.I64:
		return result.PutInt64(v.GetInt64() + other.toInt64())
	case ir.I32:
		return result.PutInt32(v.GetInt32() + other.toInt32())
	case ir.I16:
		return result.PutInt16(v.GetInt16() + other.toInt16())
	case ir.I8:
		return result.PutInt8(v.GetInt8() + other.toInt8())
	case ir.U64:
		return result.PutUint64(v.GetUint64() + other.toUint64())
	case ir.U32:
		return result.PutUint32(v.GetUint32() + uint32(other.toUint64()))
	case ir.U16:
		return result.PutUint16(v.GetUint16() + uint16(other.toUint64()))
	case ir.U8:
		return result.PutUint8(v.GetUint8() + uint8(other.toUint64()))
	default:
		return result.PutUint64(v.GetUint64() + other.toUint64())
	}
}

func (v Value) Sub(other Value) Value {
	result := Value{Type: v.Type}

	switch v.Type.(type) {
	case ir.F64:
		return result.PutFloat64(v.GetFloat64() - other.toFloat64())
	case ir.F32:
		return result.PutFloat32(v.GetFloat32() - other.toFloat32())
	case ir.I64:
		return result.PutInt64(v.GetInt64() - other.toInt64())
	case ir.I32:
		return result.PutInt32(v.GetInt32() - other.toInt32())
	case ir.I16:
		return result.PutInt16(v.GetInt16() - other.toInt16())
	case ir.I8:
		return result.PutInt8(v.GetInt8() - other.toInt8())
	case ir.U64:
		return result.PutUint64(v.GetUint64() - other.toUint64())
	case ir.U32:
		return result.PutUint32(v.GetUint32() - uint32(other.toUint64()))
	case ir.U16:
		return result.PutUint16(v.GetUint16() - uint16(other.toUint64()))
	case ir.U8:
		return result.PutUint8(v.GetUint8() - uint8(other.toUint64()))
	default:
		return result.PutUint64(v.GetUint64() - other.toUint64())
	}
}

func (v Value) Mul(other Value) Value {
	result := Value{Type: v.Type}
	switch v.Type.(type) {
	case ir.F64:
		return result.PutFloat64(v.GetFloat64() * other.toFloat64())
	case ir.F32:
		return result.PutFloat32(v.GetFloat32() * other.toFloat32())
	case ir.I64:
		return result.PutInt64(v.GetInt64() * other.toInt64())
	case ir.I32:
		return result.PutInt32(v.GetInt32() * other.toInt32())
	case ir.I16:
		return result.PutInt16(v.GetInt16() * other.toInt16())
	case ir.I8:
		return result.PutInt8(v.GetInt8() * other.toInt8())
	case ir.U64:
		return result.PutUint64(v.GetUint64() * other.toUint64())
	case ir.U32:
		return result.PutUint32(v.GetUint32() * uint32(other.toUint64()))
	case ir.U16:
		return result.PutUint16(v.GetUint16() * uint16(other.toUint64()))
	case ir.U8:
		return result.PutUint8(v.GetUint8() * uint8(other.toUint64()))
	default:
		return result.PutUint64(v.GetUint64() * other.toUint64())
	}
}

func (v Value) Div(other Value) Value {
	result := Value{Type: v.Type}
	switch v.Type.(type) {
	case ir.F64:
		return result.PutFloat64(v.GetFloat64() / other.toFloat64())
	case ir.F32:
		return result.PutFloat32(v.GetFloat32() / other.toFloat32())
	case ir.I64:
		divisor := other.toInt64()
		if divisor == 0 {
			return result.PutInt64(0)
		}
		return result.PutInt64(v.GetInt64() / divisor)
	case ir.I32:
		divisor := other.toInt32()
		if divisor == 0 {
			return result.PutInt32(0)
		}
		return result.PutInt32(v.GetInt32() / divisor)
	case ir.I16:
		divisor := other.toInt16()
		if divisor == 0 {
			return result.PutInt16(0)
		}
		return result.PutInt16(v.GetInt16() / divisor)
	case ir.I8:
		divisor := other.toInt8()
		if divisor == 0 {
			return result.PutInt8(0)
		}
		return result.PutInt8(v.GetInt8() / divisor)
	case ir.U64:
		divisor := other.toUint64()
		if divisor == 0 {
			return result.PutUint64(0)
		}
		return result.PutUint64(v.GetUint64() / divisor)
	case ir.U32:
		divisor := uint32(other.toUint64())
		if divisor == 0 {
			return result.PutUint32(0)
		}
		return result.PutUint32(v.GetUint32() / divisor)
	case ir.U16:
		divisor := uint16(other.toUint64())
		if divisor == 0 {
			return result.PutUint16(0)
		}
		return result.PutUint16(v.GetUint16() / divisor)
	case ir.U8:
		divisor := uint8(other.toUint64())
		if divisor == 0 {
			return result.PutUint8(0)
		}
		return result.PutUint8(v.GetUint8() / divisor)
	default:
		divisor := other.toUint64()
		if divisor == 0 {
			return result.PutUint64(0)
		}
		return result.PutUint64(v.GetUint64() / divisor)
	}
}

func (v Value) Mod(other Value) Value {
	result := Value{Type: v.Type}
	switch v.Type.(type) {
	case ir.F64:
		return result.PutFloat64(math.Mod(v.GetFloat64(), other.toFloat64()))
	case ir.F32:
		return result.PutFloat32(float32(math.Mod(float64(v.GetFloat32()), float64(other.toFloat32()))))
	case ir.I64:
		divisor := other.toInt64()
		if divisor == 0 {
			return result.PutInt64(0)
		}
		return result.PutInt64(v.GetInt64() % divisor)
	case ir.I32:
		divisor := other.toInt32()
		if divisor == 0 {
			return result.PutInt32(0)
		}
		return result.PutInt32(v.GetInt32() % divisor)
	case ir.I16:
		divisor := other.toInt16()
		if divisor == 0 {
			return result.PutInt16(0)
		}
		return result.PutInt16(v.GetInt16() % divisor)
	case ir.I8:
		divisor := other.toInt8()
		if divisor == 0 {
			return result.PutInt8(0)
		}
		return result.PutInt8(v.GetInt8() % divisor)
	case ir.U64:
		divisor := other.toUint64()
		if divisor == 0 {
			return result.PutUint64(0)
		}
		return result.PutUint64(v.GetUint64() % divisor)
	case ir.U32:
		divisor := uint32(other.toUint64())
		if divisor == 0 {
			return result.PutUint32(0)
		}
		return result.PutUint32(v.GetUint32() % divisor)
	case ir.U16:
		divisor := uint16(other.toUint64())
		if divisor == 0 {
			return result.PutUint16(0)
		}
		return result.PutUint16(v.GetUint16() % divisor)
	case ir.U8:
		divisor := uint8(other.toUint64())
		if divisor == 0 {
			return result.PutUint8(0)
		}
		return result.PutUint8(v.GetUint8() % divisor)
	default:
		divisor := other.toUint64()
		if divisor == 0 {
			return result.PutUint64(0)
		}
		return result.PutUint64(v.GetUint64() % divisor)
	}
}
