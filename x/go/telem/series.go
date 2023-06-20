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
	TimeRange TimeRange `json:"time_range" msgpack:"time_range"`
	DataType  DataType  `json:"data_type" msgpack:"data_type"`
	Data      []byte    `json:"data" msgpack:"data"`
}

func (s Series) Len() int64 { return s.DataType.Density().SampleCount(s.Size()) }

func (s Series) Size() Size { return Size(len(s.Data)) }

func ValueAt[T types.Numeric](a Series, i int64) T {
	b := a.Data[i*int64(a.DataType.Density()) : (i+1)*int64(a.DataType.Density())]
	return UnmarshalF[T](a.DataType)(b)
}
