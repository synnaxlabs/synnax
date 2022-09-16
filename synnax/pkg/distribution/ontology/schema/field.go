package schema

import "github.com/google/uuid"

type FieldType uint8

func (f FieldType) AssertValue(v interface{}) bool {
	switch f {
	case String:
		return assertValueType[string](v)
	case Int:
		return assertValueType[int](v)
	case Int8:
		return assertValueType[int8](v)
	case Int16:
		return assertValueType[float64](v)
	case Int32:
		return assertValueType[int32](v)
	case Int64:
		return assertValueType[int64](v)
	case Uint8:
		return assertValueType[uint8](v)
	case Uint16:
		return assertValueType[uint16](v)
	case Uint32:
		return assertValueType[uint32](v)
	case Uint64:
		return assertValueType[uint64](v)
	case Float32:
		return assertValueType[float32](v)
	case Float64:
		return assertValueType[float64](v)
	case Bool:
		return assertValueType[bool](v)
	case UUID:
		return assertValueType[uuid.UUID](v)
	default:
		panic("[FieldType]")
	}
}

const (
	String FieldType = iota
	Int
	Int8
	Int16
	Int32
	Int64
	Uint8
	Uint16
	Uint32
	Uint64
	Float32
	Float64
	Bool
	UUID
)

type Value interface {
	string |
		int |
		int8 |
		int16 |
		int32 |
		int64 |
		uint8 |
		uint16 |
		uint32 |
		uint64 |
		float32 |
		float64 |
		bool |
		uuid.UUID
}

func assertValueType[V Value](v interface{}) bool { _, ok := v.(V); return ok }

type Field struct {
	Type FieldType
}
