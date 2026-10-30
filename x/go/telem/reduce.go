// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"fmt"
	"math"
	"math/bits"
	"slices"
	"unsafe"

	"github.com/synnaxlabs/x/types"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

// PointsPerGroup returns the number of points a reduction emits for each group.
func (a Aggregation) PointsPerGroup() int64 {
	if a == AggregationMinMax {
		return 2
	}
	return 1
}

// maxGroupSize caps group sizes, because a group never spans an alignment domain.
const maxGroupSize = math.MaxUint32 - 1

// GroupSize returns the number of samples per group that brings count samples under
// limit points with the given aggregation. Each point stands for count / limit samples,
// rounded up. It returns 1, meaning no reduction, when limit is zero or count already
// fits under limit. The size caps at math.MaxUint32 - 1.
func GroupSize(count int64, limit uint32, agg Aggregation) uint32 {
	if limit == 0 || count <= int64(limit) {
		return 1
	}
	return agg.GroupSize(uint32(min((count-1)/int64(limit)+1, maxGroupSize)))
}

// GroupSize returns the group size of the aggregation for a read that decimation
// reduces with groups of the given size under the same point limit. Min/max groups are
// twice as large, because min/max keeps two points per group.
func (a Aggregation) GroupSize(decimated uint32) uint32 {
	if decimated <= 1 {
		return 1
	}
	return uint32(min(int64(decimated)*a.PointsPerGroup(), maxGroupSize))
}

// GroupStart returns the alignment of the first sample in the group that contains a.
// Groups of size groupSize are anchored at sample zero of each alignment domain, so a
// group never spans two domains.
func GroupStart(a Alignment, groupSize uint32) Alignment {
	return NewAlignment(a.DomainIndex(), a.SampleIndex()-a.SampleIndex()%groupSize)
}

// ReduceCut collapses the series into groups of groupSize samples and keeps the points
// that agg selects from each group. Group k of a domain holds the samples whose
// alignment sample index lies in [k*groupSize, (k+1)*groupSize), so every series on the
// same index reduces over the same groups. A group only partly inside the series
// reduces the samples that are present.
//
// The result's AlignmentMultiple is groupSize/2 for AggregationMinMax and groupSize
// otherwise, and its Alignment is the start of the first group. Each min/max group
// therefore occupies the alignments of its sample 0 and sample groupSize/2.
//
// ReduceCut also summarizes the first and last groups of s when s holds only part of
// them: head is nil when s starts at the start of a group, and tail is nil when s ends
// at the end of one. AbsorbNext and AbsorbPrev use the summaries to reduce a group cut
// between two series once.
//
// ReduceCut returns the series unchanged when groupSize is below 2 or when the data
// type is not numeric. It panics if the series is already reduced.
func (s Series) ReduceCut(
	agg Aggregation,
	groupSize uint32,
) (reduced Series, head, tail *Group) {
	if groupSize < 2 || s.Len() == 0 || !Reducible(s.DataType) {
		return s, nil, nil
	}
	if s.Multiple() > 1 {
		panic("telem: cannot reduce a series that is already reduced")
	}
	multiple := groupSize
	if agg == AggregationMinMax {
		multiple = groupSize / 2
	}
	data, head, tail := reducerFor(s.DataType).reduce(s, agg, groupSize)
	return Series{
		TimeRange:         s.TimeRange,
		DataType:          s.DataType,
		Data:              data,
		Alignment:         GroupStart(s.Alignment, groupSize),
		AlignmentMultiple: multiple,
	}, head, tail
}

// AbsorbNext reduces the group that s and next each hold part of once, in s. s and next
// are series reduced with ReduceCut in read order. tail summarizes the part s holds,
// and head the part next holds. It returns s with the points of the whole group, next
// without its points of the group, and the summary of the whole group. A float average
// can differ from a whole read by rounding. It panics unless tail and head summarize
// parts of the same group.
//
// The time ranges do not change, so the points of the group can lie past the end of
// s's range. When next becomes empty, the range of s extends to the end of next's.
func (s Series) AbsorbNext(next Series, tail, head Group) (Series, Series, Group) {
	merged := tail.merge(head)
	points := merged.data()
	s.Data = slices.Concat(s.Data[:len(s.Data)-len(points)], points)
	next.Data = next.Data[len(points):]
	next.Alignment = next.Alignment.AddSamples(merged.size)
	if next.Len() == 0 {
		s.TimeRange.End = next.TimeRange.End
	}
	return s, next, merged
}

// AbsorbPrev mirrors AbsorbNext for prev, the series read before s: s takes the points
// of the group they each hold part of, and prev drops its own. tail summarizes the part
// of the group that prev holds, and head the part that s holds.
func (s Series) AbsorbPrev(prev Series, tail, head Group) (Series, Series, Group) {
	merged := tail.merge(head)
	points := merged.data()
	s.Data = slices.Concat(points, s.Data[len(points):])
	prev.Data = prev.Data[:len(prev.Data)-len(points)]
	if prev.Len() == 0 {
		s.TimeRange.Start = prev.TimeRange.Start
	}
	return s, prev, merged
}

// Group summarizes the part of one reduction group that a reduced series holds. See
// Series.ReduceCut.
type Group struct {
	tally
	start       Alignment
	aggregation Aggregation
	dataType    DataType
	size        uint32
	// samples holds the first sample and the smallest and largest samples that are
	// not NaN.
	samples []byte
}

// Start returns the alignment of the first sample position of the group.
func (g Group) Start() Alignment { return g.start }

// merge returns the summary of the samples of g followed by those of next.
func (g Group) merge(next Group) Group {
	if g.start != next.start ||
		g.aggregation != next.aggregation ||
		g.dataType != next.dataType ||
		g.size != next.size {
		panic("telem: cannot merge the summaries of two different groups")
	}
	return reducerFor(g.dataType).merge(g, next)
}

// data returns the points the group reduces to, encoded as its data type.
func (g Group) data() []byte { return reducerFor(g.dataType).data(g) }

// tally holds the parts of a group summary that do not depend on the data type. lowAt
// and highAt are the sample indices of the smallest and largest samples, which are
// only set when numeric is true. sum is the sum of each sample divided by the group
// size for floats. sumHi and sumLo are the 128-bit sum of each sample's distance from
// the type's minimum for integers, which keeps the mean exact.
type tally struct {
	count         uint64
	lowAt, highAt uint32
	numeric       bool
	sum           float64
	sumHi, sumLo  uint64
}

// summary is the summary of the samples of one group of type T. See Group.
type summary[T types.SizedNumeric] struct {
	tally
	first, low, high T
}

// summarize returns the summary of in, the samples of one group, the first of which
// has sample index at. float is true when T is a float type.
func summarize[T types.SizedNumeric](
	in []T,
	at uint32,
	agg Aggregation,
	size uint32,
	float bool,
) summary[T] {
	s := summary[T]{count: uint64(len(in)), first: in[0]}
	switch agg {
	case AggregationMinMax:
		s.low, s.high, s.lowAt, s.highAt, s.numeric = extremes(in, at)
	case AggregationAverage:
		if float {
			s.sum = scaledSum(in, float64(size))
		} else {
			s.sumHi, s.sumLo = offsetSum(in)
		}
	case AggregationDecimate:
	default:
		panic(fmt.Sprintf("telem: unknown aggregation %q", agg))
	}
	return s
}

// extremes returns the smallest and largest samples of in that are not NaN, with their
// sample indices, given that in[0] has sample index at. numeric is false when every
// sample is NaN.
func extremes[T types.SizedNumeric](
	in []T,
	at uint32,
) (low, high T, lowAt, highAt uint32, numeric bool) {
	for i, v := range in {
		pos := at + uint32(i)
		switch {
		case math.IsNaN(float64(v)):
		case !numeric:
			low, high, lowAt, highAt, numeric = v, v, pos, pos, true
		case v < low:
			low, lowAt = v, pos
		case v > high:
			high, highAt = v, pos
		}
	}
	return low, high, lowAt, highAt, numeric
}

// scaledSum returns the sum of each sample of in divided by size. Dividing before
// summing keeps the sum finite for any finite samples.
func scaledSum[T types.SizedNumeric](in []T, size float64) float64 {
	var sum float64
	for _, v := range in {
		sum += float64(v) / size
	}
	return sum
}

// offsetSum returns the 128-bit sum of each sample's distance from the smallest value
// of the integer type T.
func offsetSum[T types.SizedNumeric](in []T) (hi, lo uint64) {
	lowest := uint64(typeMin[T]())
	for _, v := range in {
		var carry uint64
		lo, carry = bits.Add64(lo, uint64(v)-lowest, 0)
		hi += carry
	}
	return hi, lo
}

// merge returns the summary of the samples of s followed by those of next.
func (s summary[T]) merge(next summary[T]) summary[T] {
	s.count += next.count
	if next.numeric && (!s.numeric || next.low < s.low) {
		s.low, s.lowAt = next.low, next.lowAt
	}
	if next.numeric && (!s.numeric || next.high > s.high) {
		s.high, s.highAt = next.high, next.highAt
	}
	s.numeric = s.numeric || next.numeric
	s.sum += next.sum
	var carry uint64
	s.sumLo, carry = bits.Add64(s.sumLo, next.sumLo, 0)
	s.sumHi += next.sumHi + carry
	return s
}

// appendPoints appends the points the group reduces to under agg. float is true when T
// is a float type.
func (s summary[T]) appendPoints(
	out []T,
	agg Aggregation,
	size uint32,
	float bool,
) []T {
	switch agg {
	case AggregationMinMax:
		return appendExtremes(out, s.first, s.low, s.high, s.lowAt, s.highAt, s.numeric)
	case AggregationAverage:
		if float {
			return append(out, T(floatMean(s.sum, s.count, size)))
		}
		return append(out, intMean[T](s.sumHi, s.sumLo, s.count))
	case AggregationDecimate:
		return append(out, s.first)
	default:
		panic(fmt.Sprintf("telem: unknown aggregation %q", agg))
	}
}

// appendExtremes appends the min/max points of a group whose first sample is first, in
// the order the samples occur.
func appendExtremes[T types.SizedNumeric](
	out []T,
	first, low, high T,
	lowAt, highAt uint32,
	numeric bool,
) []T {
	// A scan that starts at NaN never moves, so the group reduces to NaN.
	if !numeric || math.IsNaN(float64(first)) {
		return append(out, first, first)
	}
	if highAt < lowAt {
		return append(out, high, low)
	}
	return append(out, low, high)
}

// floatMean returns the mean of count samples whose sum, each divided by size, is sum.
func floatMean(sum float64, count uint64, size uint32) float64 {
	if count == uint64(size) {
		return sum
	}
	return sum * (float64(size) / float64(count))
}

// intMean returns the mean, rounded down, of count samples of the integer type T whose
// offset sum is hi and lo. See offsetSum.
func intMean[T types.SizedNumeric](hi, lo, count uint64) T {
	offset, _ := bits.Div64(hi, lo, count)
	return T(uint64(typeMin[T]()) + offset)
}

func (s summary[T]) export(
	start Alignment,
	dt DataType,
	agg Aggregation,
	size uint32,
) Group {
	return Group{
		tally:       s.tally,
		start:       start,
		aggregation: agg,
		dataType:    dt,
		size:        size,
		samples:     xunsafe.CastSlice[T, byte]([]T{s.first, s.low, s.high}),
	}
}

func importGroup[T types.SizedNumeric](g Group) summary[T] {
	samples := xunsafe.CastSlice[byte, T](g.samples)
	return summary[T]{
		tally: g.tally,
		first: samples[0],
		low:   samples[1],
		high:  samples[2],
	}
}

// numericReducer reduces the samples of one data type.
type numericReducer interface {
	// reduce returns the reduced data of s and the summaries of the groups s holds only
	// part of. See Series.ReduceCut.
	reduce(s Series, agg Aggregation, groupSize uint32) ([]byte, *Group, *Group)
	merge(a, b Group) Group
	data(g Group) []byte
}

type typedReducer[T types.SizedNumeric] struct {
	// float is true when T is a float type.
	float bool
}

func (r typedReducer[T]) reduce(
	s Series,
	agg Aggregation,
	groupSize uint32,
) (data []byte, head, tail *Group) {
	var (
		in    = xunsafe.CastSlice[byte, T](s.Data)
		at    = s.Alignment.SampleIndex()
		size  = int(groupSize)
		first = int(at % groupSize)
		n     = (first + len(in) + size - 1) / size
		out   = make([]T, 0, n*int(agg.PointsPerGroup()))
		scale = float64(groupSize)
	)
	// start is the index in the series where the current group begins. Only the first
	// group can start before the series.
	for start := -first; start < len(in); start += size {
		lo, hi := max(start, 0), min(start+size, len(in))
		group := in[lo:hi]
		if hi-lo < size {
			sum := summarize(group, at+uint32(lo), agg, groupSize, r.float)
			out = sum.appendPoints(out, agg, groupSize, r.float)
			g := sum.export(
				GroupStart(s.Alignment.AddSamples(uint32(lo)), groupSize),
				s.DataType,
				agg,
				groupSize,
			)
			if start < 0 {
				head = &g
			}
			if hi < start+size {
				tail = &g
			}
			continue
		}
		switch agg {
		case AggregationMinMax:
			low, high, lowAt, highAt, numeric := extremes(group, at+uint32(lo))
			out = appendExtremes(out, group[0], low, high, lowAt, highAt, numeric)
		case AggregationAverage:
			count := uint64(len(group))
			if r.float {
				mean := floatMean(scaledSum(group, scale), count, groupSize)
				out = append(out, T(mean))
				break
			}
			sumHi, sumLo := offsetSum(group)
			out = append(out, intMean[T](sumHi, sumLo, count))
		case AggregationDecimate:
			out = append(out, group[0])
		default:
			panic(fmt.Sprintf("telem: unknown aggregation %q", agg))
		}
	}
	return xunsafe.CastSlice[T, byte](out), head, tail
}

func (typedReducer[T]) merge(a, b Group) Group {
	return importGroup[T](a).
		merge(importGroup[T](b)).
		export(a.start, a.dataType, a.aggregation, a.size)
}

func (r typedReducer[T]) data(g Group) []byte {
	return xunsafe.CastSlice[T, byte](
		importGroup[T](g).appendPoints(nil, g.aggregation, g.size, r.float),
	)
}

func reducerFor(dt DataType) numericReducer {
	switch dt {
	case Float64T:
		return typedReducer[float64]{float: true}
	case Float32T:
		return typedReducer[float32]{float: true}
	case Int64T, TimestampT:
		return typedReducer[int64]{}
	case Int32T:
		return typedReducer[int32]{}
	case Int16T:
		return typedReducer[int16]{}
	case Int8T:
		return typedReducer[int8]{}
	case Uint64T:
		return typedReducer[uint64]{}
	case Uint32T:
		return typedReducer[uint32]{}
	case Uint16T:
		return typedReducer[uint16]{}
	case Uint8T, BooleanT:
		return typedReducer[uint8]{}
	default:
		panic(fmt.Sprintf("telem: no reduction for data type %s", dt))
	}
}

// typeMin returns the smallest value of the integer type T.
func typeMin[T types.SizedNumeric]() T {
	var zero T
	if zero-1 > zero {
		return zero
	}
	return T(int64(-1) << (8*unsafe.Sizeof(zero) - 1))
}

// Reducible returns true if Series.Reduce can aggregate series of the data type.
func Reducible(dt DataType) bool {
	switch dt {
	case Float64T, Float32T, Int64T, TimestampT, Int32T, Int16T, Int8T,
		Uint64T, Uint32T, Uint16T, Uint8T, BooleanT:
		return true
	default:
		return false
	}
}
