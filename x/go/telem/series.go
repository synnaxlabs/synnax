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
	"iter"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/bounds"
	xslices "github.com/synnaxlabs/x/slices"
	"github.com/synnaxlabs/x/stringer"
	"github.com/synnaxlabs/x/types"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

const newLineChar = '\n'

// Series is a strongly typed array of telemetry samples backed by an underlying binary
// buffer.
type Series struct {
	// TimeRange represents the time range occupied by the series' data.
	TimeRange TimeRange `json:"time_range" msgpack:"time_range"`
	// DataType is the data type of the series.
	DataType DataType `json:"data_type" msgpack:"data_type"`
	// Data is the underlying binary buffer.
	Data []byte `json:"data" msgpack:"data"`
	// Alignment defines the location of the series relative to other series in a
	// logical group. This is typically used for defining the position of the series
	// within a channel's data, but can be used for arbitrary purposes.
	Alignment Alignment `json:"alignment" msgpack:"alignment"`
	// cachedLength tracks the length of a series with a variable data type.
	cachedLength *int64
}

// Len returns the number of samples currently in the Series.
func (s Series) Len() int64 {
	if len(s.Data) == 0 {
		return 0
	}
	if s.DataType.IsVariable() {
		if s.cachedLength == nil {
			cl := int64(bytes.Count(s.Data, []byte{newLineChar}))
			s.cachedLength = &cl
		}
		return *s.cachedLength
	}
	return s.DataType.Density().SampleCount(s.Size())
}

// Size returns the number of bytes in the Series.
func (s Series) Size() Size { return Size(len(s.Data)) }

// Samples returns an iterator over the samples in the Series.
func (s Series) Samples() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		if s.DataType.IsVariable() {
			var (
				buf    []byte
				offset int
			)
			for i := range s.Data {
				if s.Data[i] == newLineChar {
					buf = s.Data[offset:i]
					offset = i + 1
					if !yield(buf) {
						return
					}
				}
			}
			return
		}
		den := int64(s.DataType.Density())
		for i := int64(0); i < s.Len(); i++ {
			b := s.Data[i*den : (i+1)*den]
			if !yield(b) {
				return
			}
		}
	}
}

// At returns the binary representation of the sample at the given index.
func (s Series) At(i int) []byte {
	i = xslices.ConvertNegativeIndex(i, int(s.Len()))
	if s.DataType.IsVariable() {
		var offset int
		for j := range s.Data {
			if s.Data[j] == newLineChar {
				if i == 0 {
					return s.Data[offset:j]
				}
				i--
				offset = j + 1
			}
		}
		panic(fmt.Sprintf(
			"index %v out of bounds for series with length %v",
			i,
			s.Len(),
		))
	}
	den := int(s.DataType.Density())
	return s.Data[i*den : (i+1)*den]
}

// Resize resizes the series to the specified number of samples. If the new length is
// smaller than the current length, the data is truncated. If the new length is larger,
// the data is extended with zero bytes. This function only supports fixed-density types
// and will panic if called on a variable-density series.
func (s *Series) Resize(length int64) {
	if length < 0 {
		panic("cannot resize series to negative length")
	}
	if s.DataType.IsVariable() {
		panic("cannot resize variable-density series")
	}
	var (
		density     = int(s.DataType.Density())
		targetSize  = int(length) * density
		currentSize = len(s.Data)
	)
	if targetSize == currentSize {
		return
	}
	if targetSize < currentSize {
		s.Data = s.Data[:targetSize]
	} else {
		s.Data = append(s.Data, make([]byte, targetSize-currentSize)...)
	}
}

// ValueAt returns the numeric value at the given index in the series. ValueAt supports
// negative indices, which will be wrapped around the end of the series. This function
// cannot be used for variable density series.
func ValueAt[T Sample](s Series, i int) T {
	i = xslices.ConvertNegativeIndex(i, int(s.Len()))
	data := xunsafe.CastSlice[byte, T](s.Data)
	return data[i]
}

// SetValueAt sets the value at the given index in the series. SetValueAt supports
// negative indices, which will be wrapped around the end of the series. This function
// cannot be used for variable density series.
func SetValueAt[T types.Numeric](s Series, i int, v T) {
	i = xslices.ConvertNegativeIndex(i, int(s.Len()))
	data := xunsafe.CastSlice[byte, T](s.Data)
	data[i] = v
}

// CopyValue copies the sample from src at the index srcIdx to the index srcIdx in src.
// dst and src must have the same DataType, and that DataType cannot be of variable
// density.
func CopyValue(dst, src Series, dstIdx, srcIdx int) {
	if dst.DataType != src.DataType || dst.DataType.IsVariable() || src.DataType.IsVariable() {
		panic("cannot copy values from non-variable series")
	}
	den := int(dst.DataType.Density())
	copy(dst.Data[dstIdx*den:(dstIdx+1)*den], src.Data[srcIdx*den:(srcIdx+1)*den])
}

// AlignmentBounds returns the alignment bounds of the series. The lower bound is the
// alignment of the first sample, and the upper bound is the alignment of the last
// sample + 1. The lower bound is inclusive, while the upper bound is exclusive.
func (s Series) AlignmentBounds() AlignmentBounds {
	return AlignmentBounds{
		Lower: s.Alignment,
		Upper: NewAlignment(
			s.Alignment.DomainIndex(),
			s.Alignment.SampleIndex()+uint32(s.Len()),
		),
	}
}

// String implements the fmt.Stringer interface.
func (s Series) String() string {
	var b strings.Builder
	_, _ = fmt.Fprintf(
		&b,
		"Series{Alignment: %v, TimeRange: %v, DataType: %v, Len: %d, Size: %d bytes, Contents: ",
		s.Alignment.String(),
		s.TimeRange.String(),
		s.DataType,
		s.Len(),
		s.Size(),
	)
	b.WriteString(s.DataString())
	b.WriteString("}")
	return b.String()
}

// Downsample returns a copy of the Series with the data down sampled by the given
// factor, i.e., 1 out of every factor samples is kept.
func (s Series) Downsample(factor int) Series {
	if factor <= 1 || len(s.Data) == 0 {
		return s
	}
	var oData []byte
	if s.DataType.IsVariable() {
		iLines := bytes.Split(s.Data, []byte{newLineChar})
		oLines := make([][]byte, 0, len(iLines)/factor+1)
		for i := 0; i < len(iLines); i += factor {
			if i < len(iLines) {
				oLines = append(oLines, iLines[i])
			}
		}
		oData = bytes.Join(oLines, []byte{newLineChar})
	} else {
		seriesLength := len(s.Data) / factor
		oData = make([]byte, 0, seriesLength)
		for i := int64(0); i < s.Len(); i += int64(factor) {
			start := i * int64(s.DataType.Density())
			end := start + int64(s.DataType.Density())
			oData = append(oData, s.Data[start:end]...)
		}
	}
	return Series{
		TimeRange: s.TimeRange,
		DataType:  s.DataType,
		Data:      oData,
		Alignment: s.Alignment,
	}
}

const maxDisplayValues = 12

func truncateAndFormatSlice[T any](slice []T) string {
	return stringer.TruncateAndFormatSlice(slice, maxDisplayValues)
}

// DeepCopy creates a deep copy of the series, including all of its data.
func (s Series) DeepCopy() Series {
	return Series{
		TimeRange: s.TimeRange,
		Alignment: s.Alignment,
		DataType:  s.DataType,
		Data:      slices.Clone(s.Data),
	}
}

// DataString returns a string representation of the data in a series.
func (s Series) DataString() string {
	if s.Len() == 0 {
		return "[]"
	}
	if s.DataType.IsVariable() {
		return truncateAndFormatSlice(UnmarshalStrings(s.Data))
	}
	switch s.DataType {
	case Float64T:
		return truncateAndFormatSlice(UnmarshalSeries[float64](s))
	case Float32T:
		return truncateAndFormatSlice(UnmarshalSeries[float32](s))
	case Int64T:
		return truncateAndFormatSlice(UnmarshalSeries[int64](s))
	case Int32T:
		return truncateAndFormatSlice(UnmarshalSeries[int32](s))
	case Int16T:
		return truncateAndFormatSlice(UnmarshalSeries[int16](s))
	case Int8T:
		return truncateAndFormatSlice(UnmarshalSeries[int8](s))
	case Uint64T:
		return truncateAndFormatSlice(UnmarshalSeries[uint64](s))
	case Uint32T:
		return truncateAndFormatSlice(UnmarshalSeries[uint32](s))
	case Uint16T:
		return truncateAndFormatSlice(UnmarshalSeries[uint16](s))
	case Uint8T:
		return truncateAndFormatSlice(UnmarshalSeries[uint8](s))
	case TimeStampT:
		first, last := xslices.Truncate(UnmarshalSeries[TimeStamp](s), maxDisplayValues)
		firstDeltas := make([]string, len(first)-1)
		for i := 1; i < len(first); i++ {
			firstDeltas[i-1] = "+" + TimeSpan(first[i]-first[0]).String()
		}
		firstDeltaStr := strings.Trim(fmt.Sprintf("%v", firstDeltas), "[]")
		if len(last) == 0 {
			return fmt.Sprintf("[%s %v]", first[0], firstDeltaStr)
		}
		lastDeltas := make([]string, len(last))
		for i := range last {
			lastDeltas[i] = "+" + TimeSpan(last[i]-first[0]).String()
		}
		lastDeltaStr := strings.Trim(fmt.Sprintf("%v", lastDeltas), "[]")
		return fmt.Sprintf("[%s %v ... %v]", first[0], firstDeltaStr, lastDeltaStr)
	default:
		return fmt.Sprintf("%v", s.Data)
	}
}

// AlignmentBounds is a set of lower and upper bounds for the alignment of a
// multi-sample data structure (such as a Series or MultiSeries). The lower bound
// represents the alignment of the first sample, while the upper bound represents the
// alignment of the last sample + 1. The lower bound is inclusive, while the upper bound
// is exclusive.
type AlignmentBounds = bounds.Bounds[Alignment]

// AlignmentBoundsZero is a set of alignment bounds whose lower and upper bound are both
// zero.
var AlignmentBoundsZero = AlignmentBounds{}

// MultiSeries is a collection of ordered Series that share the same data type.
type MultiSeries struct{ Series []Series }

func sortSeriesByAlignment(s1, s2 Series) int {
	return int(s1.Alignment - s2.Alignment)
}

// NewMultiSeries constructs a new MultiSeries from the given set of Series. The series
// are sorted by their alignment, and the data type of the series must be the same. If
// the data types are different, a panic will occur.
func NewMultiSeries(series []Series) MultiSeries {
	if len(series) == 0 {
		return MultiSeries{}
	}
	dt := series[0].DataType
	for _, s := range series {
		if s.DataType != dt {
			panic(fmt.Sprintf(
				"cannot create MultiSeries with different data types: %v != %v",
				dt,
				s.DataType,
			))
		}
	}
	slices.SortFunc(series, sortSeriesByAlignment)
	return MultiSeries{Series: series}
}

// MultiSeriesAtAlignment returns the value at the given alignment in the MultiSeries.
func MultiSeriesAtAlignment[T types.FixedNumeric](
	ms MultiSeries,
	alignment Alignment,
) T {
	for _, s := range ms.Series {
		if s.AlignmentBounds().Contains(alignment) {
			return ValueAt[T](s, int(alignment-s.Alignment))
		}
	}
	panic(fmt.Sprintf(
		"alignment %v out of bounds for multi series with alignment bounds %v",
		alignment,
		ms.AlignmentBounds(),
	))
}

// NewMultiSeriesV constructs a new MultiSeries from the given set of variadic Series.
// The series are sorted by their alignment, and the data type of the series must be the
// same. If the data types are different, a panic will occur.
func NewMultiSeriesV(series ...Series) MultiSeries { return NewMultiSeries(series) }

// AlignmentBounds returns the alignment bounds of the MultiSeries. The lower bound is
// the alignment of the first sample in the series, and the upper bound is the alignment
// of the last sample in the series + 1, i.e., the lower value is inclusive, and the
// upper value is exclusive.
func (m MultiSeries) AlignmentBounds() AlignmentBounds {
	if len(m.Series) == 0 {
		return AlignmentBoundsZero
	}
	return AlignmentBounds{
		Lower: m.Series[0].AlignmentBounds().Lower,
		Upper: m.Series[len(m.Series)-1].AlignmentBounds().Upper,
	}
}

// TimeRange returns the time range of the MultiSeries, where the start time is the
// start time of the first series, and the end time is the end time of the last series.
// The start time is inclusive and the end time is exclusive.
func (m MultiSeries) TimeRange() TimeRange {
	if len(m.Series) != 0 {
		return TimeRange{
			Start: m.Series[0].TimeRange.Start,
			End:   m.Series[len(m.Series)-1].TimeRange.End,
		}
	}
	return TimeRangeZero
}

// Append appends a series to the MultiSeries. The series must have the same data type
// as the MultiSeries. If the data types are different, a panic will occur.
func (m MultiSeries) Append(series Series) MultiSeries {
	if series.DataType != m.DataType() && len(m.Series) > 0 {
		panic(fmt.Sprintf(
			"cannot append series with different data types: %v != %v",
			m.DataType(),
			series.DataType,
		))
	}
	m.Series = append(m.Series, series)
	return m
}

// FilterGreaterThanOrEqualTo returns a new MultiSeries with all series that have an
// upper alignment bound greater than the given alignment. This is useful for filtering
// out series that are not relevant to the given alignment.
func (m MultiSeries) FilterGreaterThanOrEqualTo(a Alignment) MultiSeries {
	if len(m.Series) == 0 {
		return m
	}
	// Hot path optimization that does a quick check that the alignment of all series is
	// above the filter threshold, so we don't need to re-allocate a new slice.
	if m.Series[0].AlignmentBounds().Upper > a {
		return m
	}
	if m.Series[len(m.Series)-1].AlignmentBounds().Upper < a {
		return MultiSeries{}
	}
	return MultiSeries{
		Series: lo.Filter(m.Series, func(s Series, _ int) bool {
			return s.AlignmentBounds().Upper > a
		}),
	}
}

// Len returns the aggregate length of all series in the MultiSeries.
func (m MultiSeries) Len() int64 {
	return lo.SumBy(m.Series, func(s Series) int64 { return s.Len() })
}

// DataType returns the data type of the multi series. If the multi series is empty, the
// data type is UnknownT.
func (m MultiSeries) DataType() DataType {
	if len(m.Series) != 0 {
		return m.Series[0].DataType
	}
	return UnknownT
}

// Data returns a byte slice containing the aggregated data of all series in the
// MultiSeries. Note that this function allocates an entirely new byte slice, and is
// computationally expensive.
func (m MultiSeries) Data() []byte {
	if len(m.Series) == 0 {
		return nil
	}
	data := make([]byte, 0, m.Len())
	for _, s := range m.Series {
		data = append(data, s.Data...)
	}
	return data
}
