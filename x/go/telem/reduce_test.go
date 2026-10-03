// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Reduce", func() {
	Describe("PointsPerGroup", func() {
		DescribeTable("returns the points each group emits",
			func(agg telem.Aggregation, expected int64) {
				Expect(agg.PointsPerGroup()).To(Equal(expected))
			},
			Entry("min/max", telem.AggregationMinMax, int64(2)),
			Entry("average", telem.AggregationAverage, int64(1)),
			Entry("decimate", telem.AggregationDecimate, int64(1)),
		)
	})

	Describe("GroupSize", func() {
		DescribeTable(
			"computes the samples per group",
			func(count int64, limit uint32, agg telem.Aggregation, expected uint32) {
				Expect(telem.GroupSize(count, limit, agg)).To(Equal(expected))
			},
			Entry(
				"no limit",
				int64(1e6),
				uint32(0),
				telem.AggregationMinMax,
				uint32(1),
			),
			Entry(
				"count under the limit",
				int64(100),
				uint32(100),
				telem.AggregationMinMax,
				uint32(1),
			),
			Entry(
				"min/max",
				int64(1000),
				uint32(100),
				telem.AggregationMinMax,
				uint32(20),
			),
			Entry(
				"min/max rounds odd sizes up",
				int64(1010),
				uint32(100),
				telem.AggregationMinMax,
				uint32(22),
			),
			Entry(
				"min/max with a limit of one",
				int64(10),
				uint32(1),
				telem.AggregationMinMax,
				uint32(10),
			),
			Entry(
				"average",
				int64(1000),
				uint32(100),
				telem.AggregationAverage,
				uint32(10),
			),
			Entry(
				"decimate rounds up",
				int64(1001),
				uint32(100),
				telem.AggregationDecimate,
				uint32(11),
			),
		)
	})

	Describe("GroupStart", func() {
		DescribeTable(
			"returns the first alignment of the containing group",
			func(a telem.Alignment, size uint32, expected telem.Alignment) {
				Expect(telem.GroupStart(a, size)).To(Equal(expected))
			},
			Entry(
				"group start",
				telem.NewAlignment(3, 8),
				uint32(4),
				telem.NewAlignment(3, 8),
			),
			Entry(
				"inside a group",
				telem.NewAlignment(3, 10),
				uint32(4),
				telem.NewAlignment(3, 8),
			),
			Entry(
				"size one",
				telem.NewAlignment(3, 10),
				uint32(1),
				telem.NewAlignment(3, 10),
			),
		)
	})

	Describe("Series.Reduce", func() {
		It(
			"Should keep the min and max of each group in the order they occurred",
			func() {
				s := telem.NewSeriesV[float64](3, 1, 4, 1, 5, 9, 2, 6)
				s.Alignment = telem.NewAlignment(2, 0)
				s.TimeRange = telem.TimeRange{Start: 10, End: 20}
				r := s.Reduce(telem.AggregationMinMax, 4)
				Expect(r.Unmarshal[float64]()).To(Equal([]float64{1, 4, 9, 2}))
				Expect(r.Alignment).To(Equal(telem.NewAlignment(2, 0)))
				Expect(r.AlignmentMultiple).To(Equal(uint32(2)))
				Expect(r.TimeRange).To(Equal(s.TimeRange))
				Expect(r.DataType).To(Equal(telem.Float64T))
			},
		)

		It("Should reduce a group that starts before the series", func() {
			s := telem.NewSeriesV[int32](10, 20, 30, 40, 50)
			s.Alignment = telem.NewAlignment(0, 2)
			r := s.Reduce(telem.AggregationMinMax, 4)
			Expect(r.Unmarshal[int32]()).To(Equal([]int32{10, 20, 30, 50}))
			Expect(r.Alignment).To(Equal(telem.NewAlignment(0, 0)))
		})

		It("Should repeat the sample of a group with one sample present", func() {
			s := telem.NewSeriesV[int16](7, 8, 9)
			s.Alignment = telem.NewAlignment(0, 3)
			r := s.Reduce(telem.AggregationMinMax, 2)
			Expect(r.Unmarshal[int16]()).To(Equal([]int16{7, 7, 8, 9}))
			Expect(r.Alignment).To(Equal(telem.NewAlignment(0, 2)))
			Expect(r.AlignmentMultiple).To(Equal(uint32(1)))
		})

		It("Should keep the mean of each group", func() {
			s := telem.NewSeriesV[float32](1, 2, 3, 4, 5, 6)
			r := s.Reduce(telem.AggregationAverage, 3)
			Expect(r.Unmarshal[float32]()).To(Equal([]float32{2, 5}))
			Expect(r.AlignmentMultiple).To(Equal(uint32(3)))
		})

		It("Should round the mean of an integer group down", func() {
			s := telem.NewSeriesV[int64](1, 2, -1, -2)
			r := s.Reduce(telem.AggregationAverage, 2)
			Expect(r.Unmarshal[int64]()).To(Equal([]int64{1, -2}))
		})

		It("Should keep the mean of timestamps exact", func() {
			base := telem.TimeStamp(1_790_000_000_000_000_001)
			s := telem.NewSeriesV(base, base+2, base+4)
			r := s.Reduce(telem.AggregationAverage, 3)
			Expect(
				r.Unmarshal[telem.TimeStamp](),
			).To(Equal([]telem.TimeStamp{base + 2}))
		})

		It(
			"Should average a narrow signed group whose range overflows the type",
			func() {
				s := telem.NewSeriesV[int8](-100, 100)
				r := s.Reduce(telem.AggregationAverage, 2)
				Expect(r.Unmarshal[int8]()).To(Equal([]int8{0}))
			},
		)

		It("Should panic on an unknown aggregation", func() {
			s := telem.NewSeriesV[float32](1, 2, 3, 4)
			Expect(func() { s.Reduce(telem.Aggregation("median"), 2) }).To(PanicWith(
				`telem: unknown aggregation "median"`,
			))
		})

		It("Should panic when the series is already reduced", func() {
			s := telem.NewSeriesV[float32](1, 2, 3, 4)
			s.AlignmentMultiple = 2
			Expect(func() { s.Reduce(telem.AggregationAverage, 2) }).To(PanicWith(
				"telem: cannot reduce a series that is already reduced",
			))
		})

		It("Should keep the first sample of each group when decimating", func() {
			s := telem.NewSeriesV[uint8](1, 2, 3, 4, 5, 6)
			s.Alignment = telem.NewAlignment(0, 1)
			r := s.Reduce(telem.AggregationDecimate, 2)
			Expect(r.Unmarshal[uint8]()).To(Equal([]uint8{1, 2, 4, 6}))
			Expect(r.Alignment).To(Equal(telem.NewAlignment(0, 0)))
			Expect(r.AlignmentMultiple).To(Equal(uint32(2)))
		})

		It("Should keep the first and last timestamp of each min/max group", func() {
			s := telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6)
			r := s.Reduce(telem.AggregationMinMax, 4)
			Expect(r.Unmarshal[telem.TimeStamp]()).To(Equal([]telem.TimeStamp{
				telem.SecondTS * 1,
				telem.SecondTS * 4,
				telem.SecondTS * 5,
				telem.SecondTS * 6,
			}))
			Expect(r.DataType).To(Equal(telem.TimestampT))
		})

		It(
			"Should reduce an index and its data channel to matching alignments",
			func() {
				alignment := telem.NewAlignment(4, 6)
				idx := telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8, 9)
				idx.Alignment = alignment
				data := telem.NewSeriesV[float64](9, 8, 7, 6, 5, 4, 3, 2, 1)
				data.Alignment = alignment
				rIdx := idx.Reduce(telem.AggregationMinMax, 4)
				rData := data.Reduce(telem.AggregationMinMax, 4)
				Expect(rIdx.Len()).To(Equal(rData.Len()))
				Expect(rIdx.AlignmentBounds()).To(Equal(rData.AlignmentBounds()))
			},
		)

		It("Should treat booleans as bytes", func() {
			s := telem.NewSeriesV(false, true, false, false)
			r := s.Reduce(telem.AggregationMinMax, 2)
			Expect(r.Unmarshal[bool]()).To(Equal([]bool{false, true, false, false}))
			Expect(r.DataType).To(Equal(telem.BooleanT))
		})

		DescribeTable(
			"returns the series unchanged",
			func(s telem.Series, agg telem.Aggregation, size uint32) {
				Expect(s.Reduce(agg, size)).To(Equal(s))
			},
			Entry(
				"group size one",
				telem.NewSeriesV[float64](1, 2, 3),
				telem.AggregationMinMax,
				uint32(1),
			),
			Entry(
				"empty series",
				telem.NewSeriesV[float64](),
				telem.AggregationMinMax,
				uint32(2),
			),
			Entry(
				"strings",
				telem.NewSeriesV("a", "b", "c"),
				telem.AggregationMinMax,
				uint32(2),
			),
		)
	})

	Describe("MultiSeries.Reduce", func() {
		It("Should reduce a group split across contiguous series once", func() {
			a := telem.NewSeriesV[int64](1, 2, 3)
			a.Alignment = telem.NewAlignment(0, 0)
			a.TimeRange = telem.TimeRange{Start: 1, End: 4}
			b := telem.NewSeriesV[int64](4, 5, 6)
			b.Alignment = telem.NewAlignment(0, 3)
			b.TimeRange = telem.TimeRange{Start: 4, End: 7}
			r := telem.NewMultiSeriesV(a, b).Reduce(telem.AggregationMinMax, 4)
			Expect(r.Series).To(HaveLen(1))
			Expect(r.Series[0].Unmarshal[int64]()).To(Equal([]int64{1, 4, 5, 6}))
			Expect(r.Series[0].TimeRange).To(Equal(telem.TimeRange{Start: 1, End: 7}))
			Expect(r.Series[0].Alignment).To(Equal(telem.NewAlignment(0, 0)))
		})

		It("Should reduce series with a gap between them separately", func() {
			a := telem.NewSeriesV[int64](1, 2)
			a.Alignment = telem.NewAlignment(0, 0)
			b := telem.NewSeriesV[int64](3, 4)
			b.Alignment = telem.NewAlignment(1, 0)
			r := telem.NewMultiSeriesV(a, b).Reduce(telem.AggregationAverage, 2)
			Expect(r.Series).To(HaveLen(2))
			Expect(r.Series[0].Unmarshal[int64]()).To(Equal([]int64{1}))
			Expect(r.Series[1].Unmarshal[int64]()).To(Equal([]int64{3}))
			Expect(r.Series[1].Alignment).To(Equal(telem.NewAlignment(1, 0)))
		})

		DescribeTable("returns the multi-series unchanged",
			func(m telem.MultiSeries, agg telem.Aggregation, size uint32) {
				Expect(m.Reduce(agg, size)).To(Equal(m))
			},
			Entry(
				"group size one",
				telem.NewMultiSeriesV(telem.NewSeriesV[int64](1, 2)),
				telem.AggregationMinMax,
				uint32(1),
			),
			Entry(
				"strings",
				telem.NewMultiSeriesV(telem.NewSeriesV("a", "b")),
				telem.AggregationMinMax,
				uint32(2),
			),
		)
	})

	Describe("Reducible", func() {
		DescribeTable("reports whether Series.Reduce aggregates the data type",
			func(dt telem.DataType, expected bool) {
				Expect(telem.Reducible(dt)).To(Equal(expected))
			},
			Entry("float64", telem.Float64T, true),
			Entry("float32", telem.Float32T, true),
			Entry("int64", telem.Int64T, true),
			Entry("timestamp", telem.TimestampT, true),
			Entry("int32", telem.Int32T, true),
			Entry("int16", telem.Int16T, true),
			Entry("int8", telem.Int8T, true),
			Entry("uint64", telem.Uint64T, true),
			Entry("uint32", telem.Uint32T, true),
			Entry("uint16", telem.Uint16T, true),
			Entry("uint8", telem.Uint8T, true),
			Entry("boolean", telem.BooleanT, true),
			Entry("string", telem.StringT, false),
			Entry("json", telem.JSONT, false),
			Entry("uuid", telem.UUIDT, false),
		)
	})

	Describe("Series.AlignmentBounds", func() {
		It("Should span every alignment step of a reduced series", func() {
			s := telem.NewSeriesV[float64](1, 2, 3)
			s.Alignment = telem.NewAlignment(1, 10)
			s.AlignmentMultiple = 5
			Expect(s.AlignmentBounds()).To(Equal(telem.AlignmentBounds{
				Lower: telem.NewAlignment(1, 10),
				Upper: telem.NewAlignment(1, 25),
			}))
		})
	})

	Describe("Series.Multiple", func() {
		DescribeTable("returns the alignment steps per sample",
			func(multiple, expected uint32) {
				s := telem.Series{AlignmentMultiple: multiple}
				Expect(s.Multiple()).To(Equal(expected))
			},
			Entry("zero means one", uint32(0), uint32(1)),
			Entry("set", uint32(4), uint32(4)),
		)
	})

	Describe("Series.DeepCopy", func() {
		It("Should copy the alignment multiple", func() {
			s := telem.NewSeriesV[float64](1, 2)
			s.AlignmentMultiple = 3
			Expect(s.DeepCopy().AlignmentMultiple).To(Equal(uint32(3)))
		})
	})
})
