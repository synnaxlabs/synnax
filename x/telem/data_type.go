package telem

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
	Float64:         Bit64,
	Float32:         Bit32,
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
