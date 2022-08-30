package telem

import (
	"bytes"
	"encoding/binary"
	binaryx "github.com/arya-analytics/x/binary"
)

// DataType is a string that represents a data type.
type DataType string

func (d DataType) Density() Density { return dataTypeDensities[d] }

var (
	DataTypeUnknown DataType = ""
	Float64         DataType = "float64"
	Float32         DataType = "float32"
	Int64           DataType = "int64"
	Int32           DataType = "int32"
	Int16           DataType = "int16"
	Int8            DataType = "int8"
	Uint64          DataType = "uint64"
	Uint32          DataType = "uint32"
	Uint16          DataType = "uint16"
	Byte            DataType = "byte"
	Bytes           DataType = "bytes"
)

var dataTypeDensities = map[DataType]Density{
	DataTypeUnknown: DensityUnknown,
	Int32:           Bit32,
	Int16:           Bit16,
	Int8:            Bit8,
	Uint64:          Bit64,
	Uint32:          Bit32,
	Uint16:          Bit16,
	Byte:            Bit8,
	Int64:           Bit64,
	Bytes:           DensityUnknown,
}

type DataTypeConstraint interface {
	float64 |
		float32 |
		int64 |
		int32 |
		int64 |
		int32 |
		int16 |
		int8 |
		uint64 |
		uint32 |
		uint16 |
		byte |
		[]byte
}

func ParseBuffer[D DataTypeConstraint](r bytes.Buffer, dt DataType) ([]D, error) {
	parser := binaryx.NewBufferParser(binary.LittleEndian)
	switch dt {
	case Float64:
		d, err := parser.Float64(r)
		return any(d).([]D), err
	case Float32:
		d, err := parser.Float32(r)
		return any(d).([]D), err
	case Int64:
		d, err := parser.Int64(r)
		return any(d).([]D), err
	case Int32:
		d, err := parser.Int32(r)
		return any(d).([]D), err
	case Int16:
		d, err := parser.Int16(r)
		return any(d).([]D), err
	case Int8:
		d, err := parser.Int8(r)
		return any(d).([]D), err
	case Uint64:
		d, err := parser.Uint64(r)
		return any(d).([]D), err
	case Uint32:
		d, err := parser.Uint32(r)
		return any(d).([]D), err
	case Uint16:
		d, err := parser.Uint16(r)
		return any(d).([]D), err
	case Byte:
		d, err := parser.Byte(r)
		return any(d).([]D), err
	case Bytes:
		return any(r.Bytes()).([]D), nil
	default:
		return nil, InvalidDataType
	}
}
