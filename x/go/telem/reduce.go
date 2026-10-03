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
	"slices"

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

// GroupSize returns the number of samples per group that brings count samples under
// limit points with the given aggregation. It returns 1, meaning no reduction, when
// limit is zero or count already fits under limit. Min/max group sizes are always even.
func GroupSize(count int64, limit uint32, agg Aggregation) uint32 {
	if limit == 0 || count <= int64(limit) {
		return 1
	}
	groups := max(int64(limit)/agg.PointsPerGroup(), 1)
	size := (count + groups - 1) / groups
	if agg == AggregationMinMax {
		size += size % 2
	}
	return uint32(size)
}

// GroupStart returns the alignment of the first sample in the group that contains a.
// Groups of size groupSize are anchored at sample zero of each alignment domain, so a
// group never spans two domains.
func GroupStart(a Alignment, groupSize uint32) Alignment {
	return NewAlignment(a.DomainIndex(), a.SampleIndex()-a.SampleIndex()%groupSize)
}

// Reduce collapses the series into groups of groupSize samples and keeps the points
// that agg selects from each group. Group k of a domain holds the samples whose
// alignment sample index lies in [k*groupSize, (k+1)*groupSize), so every series on
// the same index reduces over the same groups. A group only partly inside the series
// reduces the samples that are present.
//
// The result's AlignmentMultiple is groupSize/2 for AggregationMinMax and groupSize
// otherwise, and its Alignment is the start of the first group. Each min/max group
// therefore occupies the alignments of its sample 0 and sample groupSize/2.
//
// Reduce returns the series unchanged when groupSize is below 2 or when the data type
// is not numeric. It panics if the series is already reduced.
func (s Series) Reduce(agg Aggregation, groupSize uint32) Series {
	if groupSize < 2 || s.Len() == 0 {
		return s
	}
	if s.Multiple() > 1 {
		panic("telem: cannot reduce a series that is already reduced")
	}
	var data []byte
	switch s.DataType {
	case Float64T:
		data = reduceData(s, agg, groupSize, floatOffset[float64])
	case Float32T:
		data = reduceData(s, agg, groupSize, floatOffset[float32])
	case Int64T, TimestampT:
		data = reduceData(s, agg, groupSize, intOffset[int64])
	case Int32T:
		data = reduceData(s, agg, groupSize, intOffset[int32])
	case Int16T:
		data = reduceData(s, agg, groupSize, intOffset[int16])
	case Int8T:
		data = reduceData(s, agg, groupSize, intOffset[int8])
	case Uint64T:
		data = reduceData(s, agg, groupSize, intOffset[uint64])
	case Uint32T:
		data = reduceData(s, agg, groupSize, intOffset[uint32])
	case Uint16T:
		data = reduceData(s, agg, groupSize, intOffset[uint16])
	case Uint8T, BooleanT:
		data = reduceData(s, agg, groupSize, intOffset[uint8])
	default:
		return s
	}
	multiple := groupSize
	if agg == AggregationMinMax {
		multiple = groupSize / 2
	}
	return Series{
		TimeRange:         s.TimeRange,
		DataType:          s.DataType,
		Data:              data,
		Alignment:         GroupStart(s.Alignment, groupSize),
		AlignmentMultiple: multiple,
	}
}

// Reduce joins each run of alignment-contiguous series and reduces it as one series, so
// that a group split across two series yields one set of points. See Series.Reduce.
func (m MultiSeries) Reduce(agg Aggregation, groupSize uint32) MultiSeries {
	if groupSize < 2 || !Reducible(m.DataType()) {
		return m
	}
	out := MultiSeries{Series: make([]Series, 0, len(m.Series))}
	for start := 0; start < len(m.Series); {
		end := start + 1
		for end < len(m.Series) &&
			m.Series[end-1].Multiple() == m.Series[end].Multiple() &&
			m.Series[end-1].AlignmentBounds().Upper == m.Series[end].Alignment {
			end++
		}
		run := m.Series[start]
		if end-start > 1 {
			joined := MultiSeries{Series: m.Series[start:end]}
			run.Data = joined.Data()
			run.TimeRange = joined.TimeRange()
		}
		out.Series = append(out.Series, run.Reduce(agg, groupSize))
		start = end
	}
	return out
}

// intOffset returns v-lo for v >= lo. The uint64 subtraction is exact and cannot
// overflow the narrower type.
func intOffset[T types.Integer](v, lo T) float64 {
	return float64(uint64(v) - uint64(lo))
}

func floatOffset[T types.Floating](v, lo T) float64 { return float64(v - lo) }

func reduceData[T types.SizedNumeric](
	s Series,
	agg Aggregation,
	groupSize uint32,
	offset func(v, lo T) float64,
) []byte {
	var (
		in    = xunsafe.CastSlice[byte, T](s.Data)
		size  = int(groupSize)
		first = int(s.Alignment.SampleIndex() % groupSize)
		n     = (first + len(in) + size - 1) / size
		out   = make([]T, 0, n*int(agg.PointsPerGroup()))
	)
	// start is the index in the series where the current group begins. Only the first
	// group can start before the series.
	for start := -first; start < len(in); start += size {
		group := in[max(start, 0):min(start+size, len(in))]
		switch agg {
		case AggregationMinMax:
			minI, maxI := 0, 0
			for i, v := range group {
				if v < group[minI] {
					minI = i
				}
				if v > group[maxI] {
					maxI = i
				}
			}
			out = append(out, group[min(minI, maxI)], group[max(minI, maxI)])
		case AggregationAverage:
			// Summing offsets from the minimum keeps 64-bit values, such as timestamps,
			// exact where their float64 conversion would round.
			lo := slices.Min(group)
			var sum float64
			for _, v := range group {
				sum += offset(v, lo)
			}
			out = append(out, lo+T(sum/float64(len(group))))
		case AggregationDecimate:
			out = append(out, group[0])
		default:
			panic(fmt.Sprintf("telem: unknown aggregation %q", agg))
		}
	}
	return xunsafe.CastSlice[T, byte](out)
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
