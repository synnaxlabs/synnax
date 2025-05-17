// Copyright 2025 Synnax Labs, Inc.
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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Matchers", func() {
	Describe("MatchSeries", func() {
		It("Should return true if two series match", func() {
			matched := MustSucceed(telem.MatchSeries(telem.NewSeriesV[uint8](1, 2, 3)).Match(telem.NewSeriesV[uint8](1, 2, 3)))
			Expect(matched).To(BeTrue())
		})

		DescribeTable("Series that do not match", func(expected telem.Series, actual telem.Series, message string) {
			matcher := telem.MatchSeries(expected)
			matched := MustSucceed(matcher.Match(actual))
			Expect(matched).To(BeFalse())
			diff := matcher.FailureMessage(actual)
			Expect(diff).To(Equal(message))
		},
			Entry(
				"Mismatched Data",
				telem.NewSeriesV[uint8](1, 2, 3),
				telem.NewSeriesV[uint8](1, 2, 4),
				`Series did not match:
Data:
	Expected: [1 2 3]
	Actual: [1 2 4]`,
			),
			Entry("Mismatched Data Types",
				telem.NewSeriesV[uint8](1, 2, 3),
				telem.NewSeriesV[uint64](1, 2, 4),
				`Series did not match:
DataType:
	Expected: uint8
	Actual: uint64`,
			),
			Entry("Mismatched Alignments",
				telem.Series{
					DataType:  telem.Float64T,
					Data:      telem.MarshalSlice[float64]([]float64{1, 2, 3}),
					Alignment: telem.NewAlignment(1, 2),
				},
				telem.Series{
					DataType:  telem.Float64T,
					Data:      telem.MarshalSlice[float64]([]float64{1, 2, 3}),
					Alignment: telem.NewAlignment(1, 3),
				},
				`Series did not match:
Alignment:
	Expected: 1-2
	Actual: 1-3`,
			),
			Entry("Mismatched Time Ranges",
				telem.Series{
					DataType:  telem.Float64T,
					Data:      telem.MarshalSlice[float64]([]float64{1, 2, 3}),
					Alignment: telem.NewAlignment(1, 2),
					TimeRange: telem.NewSecondsRange(1, 2),
				},
				telem.Series{
					DataType:  telem.Float64T,
					Data:      telem.MarshalSlice[float64]([]float64{1, 2, 3}),
					Alignment: telem.NewAlignment(1, 2),
					TimeRange: telem.NewSecondsRange(1, 3),
				},
				`Series did not match:
TimeRange:
	Expected: 1970-01-01T00:00:01Z - :02 (1s)
	Actual: 1970-01-01T00:00:01Z - :03 (2s)`,
			),
		)
	})

	Describe("MatchSeriesData", func() {
		It("Should only match against the series data", func() {
			s1 := telem.NewSecondsTSV(1, 2, 3)
			s2 := telem.NewSecondsTSV(1, 2, 3)
			s1.Alignment = 55
			s2.Alignment = 56
			Expect(s1).To(telem.MatchSeriesData(s2))
		})

		It("Should return false when the series data does not match", func() {
			s1 := telem.NewSecondsTSV(1, 2, 3)
			s2 := telem.NewSecondsTSV(1, 2, 4)
			matcher := telem.MatchSeriesData(s1)
			matched := MustSucceed(matcher.Match(s2))
			Expect(matched).To(BeFalse())
			diff := matcher.FailureMessage(s2)
			Expect(diff).To(ContainSubstring("Data:"))
		})

		It("Should return false when the data types do not match", func() {
			s1 := telem.NewSecondsTSV(1, 2, 3)
			s2 := telem.NewSeriesV[uint8](1, 2, 4)
			matcher := telem.MatchSeriesData(s1)
			matched := MustSucceed(matcher.Match(s2))
			Expect(matched).To(BeFalse())
			diff := matcher.FailureMessage(s2)
			Expect(diff).To(ContainSubstring("DataType:"))
			Expect(diff).ToNot(ContainSubstring("Data:"))
		})
	})

	Describe("MatchFrame", func() {
		It("Should return true if two frames match", func() {
			f1 := telem.UnaryFrame[int64](1, telem.NewSeriesV[int64](1, 2, 3))
			f2 := telem.UnaryFrame[int64](1, telem.NewSeriesV[int64](1, 2, 3))
			matched := MustSucceed(telem.MatchFrame(f1).Match(f2))
			Expect(matched).To(BeTrue())
		})

		It("Should return true for two empty frames", func() {
			f1 := telem.Frame[int64]{}
			f2 := telem.Frame[int64]{}
			matched := MustSucceed(telem.MatchFrame(f1).Match(f2))
			Expect(matched).To(BeTrue())
		})

		It("Should return false if frame counts do not match", func() {
			f1 := telem.UnaryFrame[int64](1, telem.NewSeriesV[int64](1, 2, 3))
			f2 := telem.MultiFrame(
				[]int64{1, 2},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
				},
			)
			matcher := telem.MatchFrame(f1)
			matched := MustSucceed(matcher.Match(f2))
			Expect(matched).To(BeFalse())
			Expect(matcher.FailureMessage(f2)).To(ContainSubstring("Frames have different counts"))
		})

		It("Should return false if series data for a key does not match", func() {
			f1 := telem.UnaryFrame[int64](1, telem.NewSeriesV[int64](1, 2, 3))
			f2 := telem.UnaryFrame[int64](1, telem.NewSeriesV[int64](1, 2, 4))
			matcher := telem.MatchFrame(f1)
			matched := MustSucceed(matcher.Match(f2))
			Expect(matched).To(BeFalse())
			Expect(matcher.FailureMessage(f2)).To(ContainSubstring("Data:"))
		})

		It("Should return false if keys do not match", func() {
			f1 := telem.UnaryFrame[int64](1, telem.NewSeriesV[int64](1, 2, 3))
			f2 := telem.UnaryFrame[int64](2, telem.NewSeriesV[int64](1, 2, 3))
			matcher := telem.MatchFrame(f1)
			matched := MustSucceed(matcher.Match(f2))
			Expect(matched).To(BeFalse())
			// The exact failure message will depend on the implementation
		})

		It("Should provide a negated failure message", func() {
			f1 := telem.UnaryFrame[int64](1, telem.NewSeriesV[int64](1, 2, 3))
			f2 := telem.UnaryFrame[int64](1, telem.NewSeriesV[int64](1, 2, 3))
			matcher := telem.MatchFrame(f1)
			msg := matcher.NegatedFailureMessage(f2)
			Expect(msg).To(ContainSubstring("Frame"))
		})
	})
})
