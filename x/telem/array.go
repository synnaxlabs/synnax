package telem

import "github.com/synnaxlabs/x/types"

type Array struct {
	TimeRange TimeRange `json:"time_range" msgpack:"time_range"`
	DataType  DataType  `json:"data_type" msgpack:"data_type"`
	Data      []byte    `json:"data" msgpack:"data"`
}

func (a Array) Len() int64 { return a.DataType.Density().SampleCount(a.Size()) }

func (a Array) Size() Size { return Size(len(a.Data)) }

func ValueAt[T types.Numeric](a Array, i int64) T {
	b := a.Data[i*int64(a.DataType.Density()) : (i+1)*int64(a.DataType.Density())]
	return UnmarshalF[T](a.DataType)(b)
}
