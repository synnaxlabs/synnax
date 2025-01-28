// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"bytes"
	"go.uber.org/zap"

	"github.com/synnaxlabs/x/types"
)

type Series struct {
	// TimeRange represents the time range occupied by the series' data.
	TimeRange TimeRange `json:"time_range" msgpack:"time_range"`
	// DataType is the data type of the series.
	DataType DataType `json:"data_type" msgpack:"data_type"`
	// Data is the underlying binary buffer.
	Data []byte `json:"data" msgpack:"data"`
	// Alignment can be used to define the alignment of the series relative to other
	// series in a logical group. This is typically used for defining the position of
	// the series within a channel's data, but can be used for arbitrary purposes.
	Alignment AlignmentPair `json:"alignment" msgpack:"alignment"`
	// cachedLength tracks the length of a series with a variable data type.
	cachedLength *int64
}

// Len returns the number of samples currently in the Series.
func (s Series) Len() int64 {
	if s.DataType.IsVariable() {
		if s.cachedLength == nil {
			cl := int64(bytes.Count(s.Data, []byte("\n")))
			s.cachedLength = &cl
		}
		return *s.cachedLength
	}
	return s.DataType.Density().SampleCount(s.Size())
}

// Size returns the number of bytes in the Series.
func (s Series) Size() Size { return Size(len(s.Data)) }

// Split separates the series into individual samples, where each byte slice is the
// encoded value of a sample. Warning: this can add a lot of heap pressure if the
// series is large.
func (s Series) Split() [][]byte {
	if s.DataType.IsVariable() {
		split := bytes.Split(s.Data, []byte("\n"))
		if len(split) == 0 {
			return nil
		}
		if len(split[len(split)-1]) == 0 {
			split = split[:len(split)-1]
		}
		return split
	}
	o := make([][]byte, s.Len())
	for i := int64(0); i < s.Len(); i++ {
		o[i] = s.Data[i*int64(s.DataType.Density()) : (i+1)*int64(s.DataType.Density())]
	}
	return o
}

// ValueAt returns the numeric value at the given index in the series. ValueAt supports
// negative indices, which will be wrapped around the end of the series. This function
// cannot be used for variable density series.
func ValueAt[T types.Numeric](s Series, i int64) (o T) {
	if s.DataType.IsVariable() {
		zap.S().DPanic("ValueAt cannot be used on variable density series")
		return
	}
	if i < 0 {
		i += s.Len()
	}
	b := s.Data[i*int64(s.DataType.Density()) : (i+1)*int64(s.DataType.Density())]
	return UnmarshalF[T](s.DataType)(b)
}

// SetValueAt sets the value at the given index in the series. SetValueAt supports
// negative indices, which will be wrapped around the end of the series. This function
// cannot be used for variable density series.
func SetValueAt[T types.Numeric](s Series, i int64, v T) {
	if s.DataType.IsVariable() {
		zap.S().DPanic("ValueAt cannot be used on variable density series")
		return
	}
	if i < 0 {
		i += s.Len()
	}
	f := MarshalF[T](s.DataType)
	f(s.Data[i*int64(s.DataType.Density()):], v)
}
