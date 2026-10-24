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
	"math"

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
			Entry(
				"a size past the largest uint32",
				int64(1)<<33,
				uint32(1),
				telem.AggregationDecimate,
				uint32(math.MaxUint32-1),
			),
			Entry(
				"a min/max size past the largest uint32",
				int64(1)<<40,
				uint32(2),
				telem.AggregationMinMax,
				uint32(math.MaxUint32-1),
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

		It("Should reduce a min/max group to NaN only when it starts with NaN", func() {
			s := telem.NewSeriesV(math.NaN(), 1, 2, 4, math.NaN(), 3)
			r := s.Reduce(telem.AggregationMinMax, 3).Unmarshal[float64]()
			Expect(r).To(HaveLen(4))
			Expect(math.IsNaN(r[0])).To(BeTrue())
			Expect(math.IsNaN(r[1])).To(BeTrue())
			Expect(r[2:]).To(Equal([]float64{4, 3}))
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

		DescribeTable(
			"Should average a group whose range overflows the type",
			func(s, expected telem.Series) {
				r := s.Reduce(telem.AggregationAverage, uint32(s.Len()))
				Expect(r.Data).To(Equal(expected.Data))
			},
			Entry(
				"float32",
				telem.NewSeriesV[float32](-math.MaxFloat32, math.MaxFloat32),
				telem.NewSeriesV[float32](0),
			),
			Entry(
				"float32 with a mean offset past the largest float32",
				telem.NewSeriesV[float32](
					-math.MaxFloat32,
					math.MaxFloat32,
					math.MaxFloat32,
				),
				telem.NewSeriesV[float32](math.MaxFloat32/3),
			),
			Entry(
				"float64",
				telem.NewSeriesV(-math.MaxFloat64, math.MaxFloat64),
				telem.NewSeriesV[float64](0),
			),
			Entry(
				"int8",
				telem.NewSeriesV[int8](-100, 100),
				telem.NewSeriesV[int8](0),
			),
			Entry(
				"int8 with a mean offset past the largest int8",
				telem.NewSeriesV[int8](-128, 127, 127),
				telem.NewSeriesV[int8](42),
			),
			Entry(
				"int64",
				telem.NewSeriesV[int64](math.MinInt64, math.MaxInt64),
				telem.NewSeriesV[int64](-1),
			),
		)

		It("Should scale the mean of a partial group to its count", func() {
			r := telem.NewSeriesV(0.1, 0.2, 2.3).Reduce(telem.AggregationAverage, 4)
			Expect(r.Unmarshal[float64]()).To(Equal([]float64{0.8666666666666665}))
		})

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

	Describe("Group", func() {
		// at returns s with its first sample at sample index at.
		at := func(s telem.Series, at uint32) telem.Series {
			s.Alignment = telem.NewAlignment(0, at)
			return s
		}

		DescribeTable("Should merge the parts of a cut group into the whole group",
			func(agg telem.Aggregation, parts ...telem.Series) {
				whole := telem.NewMultiSeries(parts)
				size := uint32(whole.Len())
				expected := telem.Series{
					DataType: whole.DataType(),
					Data:     whole.Data(),
				}.Reduce(agg, size)
				g := parts[0].Summarize(agg, size)
				for _, p := range parts[1:] {
					g = g.Merge(p.Summarize(agg, size))
				}
				Expect(g.Data()).To(Equal(expected.Data))
				Expect(g.Start()).To(Equal(telem.NewAlignment(0, 0)))
			},
			Entry(
				"min_max with the max before the min",
				telem.AggregationMinMax,
				telem.NewSeriesV[float64](2, 9),
				at(telem.NewSeriesV[float64](4, -3), 2),
			),
			Entry(
				"min_max with the min in both parts",
				telem.AggregationMinMax,
				telem.NewSeriesV[int32](1, 5),
				at(telem.NewSeriesV[int32](3, 1), 2),
			),
			Entry(
				"min_max with NaN first in a later part",
				telem.AggregationMinMax,
				telem.NewSeriesV[float32](3, 1),
				at(telem.NewSeriesV(float32(math.NaN()), 7), 2),
			),
			Entry(
				"min_max with NaN first in the group",
				telem.AggregationMinMax,
				telem.NewSeriesV(math.NaN(), 1),
				at(telem.NewSeriesV[float64](5), 2),
			),
			Entry(
				"average of three parts",
				telem.AggregationAverage,
				telem.NewSeriesV[float64](1, 2),
				at(telem.NewSeriesV[float64](3), 2),
				at(telem.NewSeriesV[float64](6, 8), 3),
			),
			Entry(
				"average of int64 extremes",
				telem.AggregationAverage,
				telem.NewSeriesV[int64](math.MinInt64, math.MaxInt64),
				at(telem.NewSeriesV[int64](math.MinInt64, math.MinInt64), 2),
			),
			Entry(
				"average of uint64 extremes",
				telem.AggregationAverage,
				telem.NewSeriesV[uint64](math.MaxUint64, math.MaxUint64),
				at(telem.NewSeriesV[uint64](math.MaxUint64, 0), 2),
			),
			Entry(
				"average of uint8 extremes",
				telem.AggregationAverage,
				telem.NewSeriesV[uint8](255, 255),
				at(telem.NewSeriesV[uint8](255, 254), 2),
			),
			Entry(
				"decimate",
				telem.AggregationDecimate,
				telem.NewSeriesV[int16](4, 2),
				at(telem.NewSeriesV[int16](1), 2),
			),
		)

		It("Should panic when summarizing the samples of two groups", func() {
			Expect(func() {
				telem.NewSeriesV[float64](1, 2, 3).Summarize(telem.AggregationMinMax, 2)
			}).To(PanicWith("telem: can only summarize the samples of one group"))
		})

		It("Should panic when merging the summaries of two groups", func() {
			a := telem.NewSeriesV[float64](1, 2).Summarize(telem.AggregationMinMax, 2)
			b := at(telem.NewSeriesV[float64](3, 4), 2).
				Summarize(telem.AggregationMinMax, 2)
			Expect(func() { a.Merge(b) }).To(
				PanicWith("telem: cannot merge the summaries of two different groups"),
			)
		})
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
})
