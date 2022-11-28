package telem

import (
	"encoding/binary"
	"github.com/synnaxlabs/x/types"
)

func NewArray[T types.Numeric](data []T) (arr Array) {
	if len(data) == 0 {
		panic("cannot infer data type from empty array")
	}
	arr.DataType = NewDataType[T](data[0])
	arr.Data = MarshalSlice(data, arr.DataType)
	return arr
}

func NewArrayV[T types.Numeric](data ...T) (arr Array) {
	return NewArray[T](data)
}

func NewSecondsTSV(data ...TimeStamp) (arr Array) {
	for i := range data {
		data[i] *= SecondTS
	}
	arr.DataType = TimeStampT
	arr.Data = MarshalSlice(data, arr.DataType)
	return arr
}

func MarshalSlice[T types.Numeric](data []T, dt DataType) []byte {
	b := make([]byte, dt.Density().Size(int64(len(data))))
	m := MarshalF[T](dt)
	for i, v := range data {
		base := i * int(dt.Density())
		m(b[base:base+int(dt.Density())], v)
	}
	return b
}

func UnmarshalSlice[T types.Numeric](b []byte, dt DataType) (data []T) {
	data = make([]T, len(b)/int(dt.Density()))
	um := UnmarshalF[T](dt)
	for i := range data {
		base := i * int(dt.Density())
		data[i] = um(b[base : base+int(dt.Density())])
	}
	return data
}

func Unmarshal[T types.Numeric](arr Array) []T {
	return UnmarshalSlice[T](arr.Data, arr.DataType)
}

func MarshalF[T types.Numeric](dt DataType) func(b []byte, v T) {
	switch dt {
	case Float64T:
		return func(b []byte, v T) { ByteOrder.PutUint64(b, uint64(v)) }
	case Float32T:
		return func(b []byte, v T) { ByteOrder.PutUint32(b, uint32(v)) }
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
	case ByteT:
		return func(b []byte, v T) { b[0] = byte(v) }
	case TimeStampT:
		return func(b []byte, v T) { ByteOrder.PutUint64(b, uint64(v)) }
	}
	panic("unsupported data type")
}

func UnmarshalF[T types.Numeric](dt DataType) func(b []byte) T {
	switch dt {
	case Float64T:
		return func(b []byte) T { return T(ByteOrder.Uint64(b)) }
	case Float32T:
		return func(b []byte) T { return T(ByteOrder.Uint32(b)) }
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
	case ByteT:
		return func(b []byte) T { return T(b[0]) }
	case TimeStampT:
		return func(b []byte) T { return T(ByteOrder.Uint64(b)) }
	}
	panic("unsupported data type")
}

var ByteOrder = binary.LittleEndian
