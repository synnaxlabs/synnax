// Copyright 2025 Synnax Labs, Inc.
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
	"fmt"
	"strings"

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

const maxDisplayValues = 12
const endDisplayCount = 5

// truncateSlice returns a string representation of a slice, showing only the first and last few elements
// if the slice is longer than maxDisplayValues
func truncateSlice[T any](slice []T) string {
	if len(slice) <= maxDisplayValues {
		return fmt.Sprintf("%v", slice)
	}

	// Create string representations of first and last elements
	first := slice[:5]
	last := slice[len(slice)-endDisplayCount:]

	// Convert to string and trim the brackets
	firstStr := strings.Trim(fmt.Sprintf("%v", first), "[]")
	lastStr := strings.Trim(fmt.Sprintf("%v", last), "[]")

	return fmt.Sprintf("[%s ... %s]", firstStr, lastStr)
}

// String implements the fmt.Stringer interface.
func (s Series) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Series{TimeRange: %v, DataType: %v, Len: %d, Size: %d bytes, Contents: ",
		s.TimeRange.RawString(),
		s.DataType,
		s.Len(),
		s.Size(),
	)

	if s.Len() == 0 {
		b.WriteString("[]}")
		return b.String()
	}

	var contents string
	if s.DataType.IsVariable() {
		contents = truncateSlice(UnmarshalStrings(s.Data))
	} else {
		switch s.DataType {
		case Float64T:
			contents = truncateSlice(Unmarshal[float64](s))
		case Float32T:
			contents = truncateSlice(Unmarshal[float32](s))
		case Int64T:
			contents = truncateSlice(Unmarshal[int64](s))
		case Int32T:
			contents = truncateSlice(Unmarshal[int32](s))
		case Int16T:
			contents = truncateSlice(Unmarshal[int16](s))
		case Int8T:
			contents = truncateSlice(Unmarshal[int8](s))
		case Uint64T:
			contents = truncateSlice(Unmarshal[uint64](s))
		case Uint32T:
			contents = truncateSlice(Unmarshal[uint32](s))
		case Uint16T:
			contents = truncateSlice(Unmarshal[uint16](s))
		case Uint8T:
			contents = truncateSlice(Unmarshal[uint8](s))
		case TimeStampT:
			contents = truncateSlice(Unmarshal[TimeStamp](s))
		case StringT:
			contents = truncateSlice(UnmarshalStrings(s.Data))
		case JSONT:
			contents = truncateSlice(UnmarshalStrings(s.Data))
		default:
			contents = fmt.Sprintf("%v", s.Data)
		}
	}

	b.WriteString(contents)
	b.WriteString("}")
	return b.String()
}
