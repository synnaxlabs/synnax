// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "github.com/synnaxlabs/x/types"

type Series struct {
	// TimeRange represents the time range occupied by the series' data.
	TimeRange TimeRange `json:"time_range" msgpack:"time_range"`
	// DataType is the data type of the series.
	DataType DataType `json:"data_type" msgpack:"data_type"`
	// Data is the underlying binary buffer.
	Data      []byte    `json:"data" msgpack:"data"`
	Alignment Alignment `json:"alignment" msgpack:"alignment"`
}

// Len returns the number of samples currently in the Series.
func (s Series) Len() int64 { return s.DataType.Density().SampleCount(s.Size()) }

// Size returns the number of bytes in the Series.
func (s Series) Size() Size { return Size(len(s.Data)) }

func ValueAt[T types.Numeric](a Series, i int64) T {
	b := a.Data[i*int64(a.DataType.Density()) : (i+1)*int64(a.DataType.Density())]
	return UnmarshalF[T](a.DataType)(b)
}
