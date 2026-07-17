// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v0 "github.com/synnaxlabs/x/telem/types/v0"
)

var _ = Describe("TimeRange", func() {
	Describe("Stringer", func() {
		DescribeTable("Should format time ranges with appropriate precision",
			func(start, end time.Time, expected string) {
				tr := v0.TimeRange{
					Start: v0.NewTimeStamp(start),
					End:   v0.NewTimeStamp(end),
				}
				Expect(tr.String()).To(Equal(expected))
			},
			Entry("nanoseconds differ",
				time.Date(2024, 3, 15, 10, 30, 45, 100, time.UTC),
				time.Date(2024, 3, 15, 10, 30, 45, 500, time.UTC),
				"2024-03-15T10:30:45.000000100Z - .000000500 (400ns)"),
			Entry("microseconds differ",
				time.Date(2024, 3, 15, 10, 30, 45, 100000, time.UTC),
				time.Date(2024, 3, 15, 10, 30, 45, 500000, time.UTC),
				"2024-03-15T10:30:45.000100Z - .000500 (400µs)"),
			Entry("milliseconds differ",
				time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC),
				time.Date(2024, 3, 15, 10, 30, 45, 500e6, time.UTC),
				"2024-03-15T10:30:45Z - .500 (500ms)"),
			Entry("seconds differ",
				time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC),
				time.Date(2024, 3, 15, 10, 30, 55, 0, time.UTC),
				"2024-03-15T10:30:45Z - :55 (10s)"),
			Entry("minutes differ",
				time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC),
				time.Date(2024, 3, 15, 10, 45, 0, 0, time.UTC),
				"2024-03-15T10:30:00Z - 45:00 (15m)"),
			Entry("hours differ",
				time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC),
				time.Date(2024, 3, 15, 11, 45, 0, 0, time.UTC),
				"2024-03-15T10:30:00Z - 11:45:00 (1h 15m)"),
			Entry("days differ",
				time.Date(2024, 3, 15, 23, 45, 0, 0, time.UTC),
				time.Date(2024, 3, 16, 0, 15, 0, 0, time.UTC),
				"2024-03-15T23:45:00Z - 03-16T00:15:00 (30m)"),
			Entry("months differ",
				time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				time.Date(2024, 2, 15, 10, 30, 0, 0, time.UTC),
				"2024-01-15T10:30:00Z - 02-15T10:30:00 (31d)"),
			Entry("years differ",
				time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				"2024-01-15T10:30:00Z - 2025-01-15T10:30:00 (366d)"),
			Entry("identical timestamps",
				time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				"2024-01-15T10:30:00Z - 10:30:00 (0s)"),
		)
	})

	Describe("NewRangeSeconds", func() {
		It("Should instantiate a time range between a particular starting number of seconds and ending number of seconds", func() {
			tr := v0.NewRangeSeconds(1, 5)
			Expect(tr.Start).To(Equal(v0.SecondTS * 1))
			Expect(tr.End).To(Equal(v0.SecondTS * 5))
		})
	})

	Describe("SpanTo", func() {
		It("Should return the correct time span", func() {
			tr := v0.TimeRange{
				Start: v0.TimeStamp(0),
				End:   v0.TimeStamp(v0.Second),
			}
			Expect(tr.Span()).To(Equal(v0.Second))
		})
	})

	Describe("IsZero", func() {
		It("Should return true if the time range is zero", func() {
			Expect(v0.TimeRangeMin.IsZero()).To(BeFalse())
			Expect(v0.TimeRangeMax.IsZero()).To(BeFalse())
			Expect(v0.TimeRangeZero.IsZero()).To(BeTrue())
		})
	})

	Describe("BoundBy", func() {

		It("Should bound the time range to the provided constraints", func() {
			tr := v0.TimeRange{
				Start: v0.TimeStamp(v0.Second),
				End:   v0.TimeStamp(v0.Second * 4),
			}
			bound := v0.TimeRange{
				Start: v0.TimeStamp(2 * v0.Second),
				End:   v0.TimeStamp(v0.Second * 3),
			}
			bounded := tr.BoundBy(bound)
			Expect(bounded.Start).To(Equal(bound.Start))
			Expect(bounded.End).To(Equal(bound.End))

		})

		It("Should bound the time range even if the start is after the end", func() {

			tr := v0.TimeRange{
				Start: v0.TimeStamp(v0.Second * 4),
				End:   v0.TimeStamp(v0.Second),
			}

			bound := v0.TimeRange{
				Start: v0.TimeStamp(2 * v0.Second),
				End:   v0.TimeStamp(v0.Second * 3),
			}

			bounded := tr.BoundBy(bound)
			Expect(bounded.Start).To(Equal(bound.End))
			Expect(bounded.End).To(Equal(bound.Start))
		})

		It("Should bound the time range even if there is zero overlap", func() {
			tr := v0.TimeRange{
				Start: v0.TimeStamp(v0.Second * 10),
				End:   v0.TimeStamp(v0.Second * 14),
			}

			bound := v0.TimeRange{
				Start: v0.TimeStamp(2 * v0.Second),
				End:   v0.TimeStamp(v0.Second * 3),
			}

			bounded := tr.BoundBy(bound)
			Expect(bounded.Start).To(Equal(bound.End))
			Expect(bounded.End).To(Equal(bound.End))
		})
	})

	Describe("ContainsStamp", func() {
		It("Should return true when the range contains the timestamp", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			Expect(tr.ContainsStamp(v0.TimeStamp(4 * v0.Second))).To(BeTrue())
			By("Being inclusive at the lower bound")
			Expect(tr.ContainsStamp(v0.TimeStamp(0 * v0.Second))).To(BeTrue())
			By("Being exclusive at the upper bound")
			Expect(tr.ContainsStamp(v0.TimeStamp(5 * v0.Second))).To(BeFalse())
		})
	})

	Describe("ContainsRange", func() {
		It("Should return true when the ranges overlap with one another", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			Expect(tr.ContainsRange(v0.TimeStamp(1).SpanRange(2 * v0.Second))).To(BeTrue())
		})
		It("Should return false when the start of one range is the end of another", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			tr2 := v0.TimeStamp(5 * v0.Second).SpanRange(5 * v0.Second)
			Expect(tr.ContainsRange(tr2)).To(BeFalse())
			Expect(tr2.ContainsRange(tr)).To(BeFalse())
		})
		It("Should return true if checked against itself", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			Expect(tr.ContainsRange(tr)).To(BeTrue())
		})
	})

	Describe("WhereOverlapsWith", func() {
		It("Should return true when the ranges overlap with one another", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			Expect(tr.OverlapsWith(v0.TimeStamp(1).SpanRange(2 * v0.Second))).To(BeTrue())
		})

		It("Should return false when the start of one range is the end of another", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			tr2 := v0.TimeStamp(5 * v0.Second).SpanRange(5 * v0.Second)
			Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			Expect(tr2.OverlapsWith(tr)).To(BeFalse())
		})

		It("Should return true if the start timestamps of the two ranges are equal", func() {
			tr1 := v0.TimeStamp(5 * v0.Second).SpanRange(5 * v0.Second)
			tr2 := v0.TimeStamp(5 * v0.Second).SpanRange(10 * v0.Second)
			Expect(tr1.OverlapsWith(tr2)).To(BeTrue())
		})

		It("Should return true if checked against itself", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			Expect(tr.OverlapsWith(tr)).To(BeTrue())
		})

		It("Should return false if the ranges do not overlap", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			tr2 := v0.TimeStamp(5 * v0.Second).SpanRange(5 * v0.Second)
			Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			Expect(tr2.OverlapsWith(tr)).To(BeFalse())
		})

		It("Should return true if one range is contained within the other", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			tr2 := v0.TimeStamp(1).SpanRange(2 * v0.Second)
			Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			Expect(tr2.OverlapsWith(tr)).To(BeTrue())
		})

		Context("Main TimeRange is not point, argument is point", func() {
			It("Should return true if the point is contained", func() {
				tr := v0.NewRangeSeconds(1, 3)
				tr2 := v0.NewRangeSeconds(2, 2)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})

			It("Should return false if the point is NOT contained", func() {
				tr := v0.NewRangeSeconds(1, 3)
				tr2 := v0.NewRangeSeconds(4, 4)
				Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			})

			It("Should return false if the point is at the end", func() {
				tr := v0.NewRangeSeconds(1, 3)
				tr2 := v0.NewRangeSeconds(3, 3)
				Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			})

			It("Should return true if the point is at the start", func() {
				tr := v0.NewRangeSeconds(1, 3)
				tr2 := v0.NewRangeSeconds(1, 1)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})
		})

		Context("Main TimeRange is point, argument is point", func() {
			It("Should return true if the points are identical", func() {
				tr := v0.NewRangeSeconds(1, 1)
				tr2 := v0.NewRangeSeconds(1, 1)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})

			It("Should return false if the points are NOT identical", func() {
				tr := v0.NewRangeSeconds(1, 1)
				tr2 := v0.NewRangeSeconds(2, 2)
				Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			})
		})

		Context("Main TimeRange is point, argument is not point", func() {
			It("Should return true if the argument contains the point", func() {
				tr := v0.NewRangeSeconds(2, 2)
				tr2 := v0.NewRangeSeconds(1, 3)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})

			It("Should return false if point is at the end of the argument", func() {
				tr := v0.NewRangeSeconds(3, 3)
				tr2 := v0.NewRangeSeconds(1, 3)
				Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			})

			It("Should return true if point is at the start", func() {
				tr := v0.NewRangeSeconds(3, 3)
				tr2 := v0.NewRangeSeconds(3, 5)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})
		})

	})

	Describe("Swap", func() {
		It("Should swap the start and end times", func() {
			tr := v0.TimeStamp(0).SpanRange(5 * v0.Second)
			Expect(tr.Start).To(Equal(v0.TimeStamp(0)))
			Expect(tr.End).To(Equal(v0.TimeStamp(5 * v0.Second)))
			tr = tr.Swap()
			Expect(tr.Start).To(Equal(v0.TimeStamp(5 * v0.Second)))
			Expect(tr.End).To(Equal(v0.TimeStamp(0)))
		})
	})

	Describe("Intersection", func() {
		Specify("Overlap, first before second", func() {
			tr := (0 * v0.SecondTS).Range(5 * v0.SecondTS)
			tr2 := (3 * v0.SecondTS).Range(8 * v0.SecondTS)
			intersection := tr.Intersection(tr2)
			Expect(intersection.Start).To(Equal(3 * v0.SecondTS))
			Expect(intersection.End).To(Equal(5 * v0.SecondTS))
		})

		Specify("Overlap, second before first", func() {
			tr2 := (0 * v0.SecondTS).Range(5 * v0.SecondTS)
			tr := (3 * v0.SecondTS).Range(8 * v0.SecondTS)
			union := tr.Intersection(tr2)
			Expect(union.Start).To(Equal(3 * v0.SecondTS))
			Expect(union.End).To(Equal(5 * v0.SecondTS))
		})

		Specify("1 Fully contain 2", func() {
			tr := (0 * v0.SecondTS).Range(10 * v0.SecondTS)
			tr2 := (3 * v0.SecondTS).Range(8 * v0.SecondTS)
			intersection := tr.Intersection(tr2)
			Expect(intersection.Start).To(Equal(3 * v0.SecondTS))
			Expect(intersection.End).To(Equal(8 * v0.SecondTS))
		})

		Specify("2 Fully contain 1", func() {
			tr := (2 * v0.SecondTS).Range(5 * v0.SecondTS)
			tr2 := (1 * v0.SecondTS).Range(8 * v0.SecondTS)
			intersection := tr.Intersection(tr2)
			Expect(intersection.Start).To(Equal(2 * v0.SecondTS))
			Expect(intersection.End).To(Equal(5 * v0.SecondTS))
		})

		Specify("Completely separate", func() {
			tr := (0 * v0.SecondTS).Range(10 * v0.SecondTS)
			tr2 := (100 * v0.SecondTS).Range(108 * v0.SecondTS)
			intersection := tr.Intersection(tr2)
			Expect(intersection).To(Equal(v0.TimeRangeZero))
		})

	})

	DescribeTable("Split", func(
		tr v0.TimeRange,
		ts v0.TimeStamp,
		expectedBefore v0.TimeSpan,
		expectedAfter v0.TimeSpan,
	) {
		before, after := tr.Split(ts)
		Expect(before.Span()).To(Equal(expectedBefore))
		Expect(after.Span()).To(Equal(expectedAfter))
	},
		Entry(
			"Completely within",
			v0.NewRangeSeconds(1, 5),
			v0.TimeStamp(v0.Second*3),
			v0.Second*2,
			v0.Second*2,
		),
		Entry("At Start",
			v0.NewRangeSeconds(1, 5),
			v0.TimeStamp(v0.Second*1),
			v0.Second*0,
			v0.Second*4,
		),
		Entry("At End",
			v0.NewRangeSeconds(1, 5),
			v0.TimeStamp(v0.Second*5),
			v0.Second*4,
			v0.Second*0,
		),
		Entry("Before Start",
			v0.NewRangeSeconds(1, 5),
			v0.TimeStamp(0),
			v0.Second*-1,
			v0.Second*5,
		),
		Entry("After End",
			v0.NewRangeSeconds(1, 5),
			v0.TimeStamp(v0.Second*20),
			v0.Second*19,
			-v0.Second*15,
		),
	)

	Describe("MakeValid", func() {
		It("Should swap the start and end timestamps if they are out of order", func() {
			tr := v0.NewRangeSeconds(5, 1)
			Expect(tr.MakeValid()).To(Equal(v0.NewRangeSeconds(1, 5)))
		})

		It("Should not swap the start and end timestamps if they are in order", func() {
			tr := v0.TimeRange{Start: 0, End: 1}
			Expect(tr.MakeValid()).To(Equal(v0.TimeRange{Start: 0, End: 1}))
		})
	})

	Describe("Midpoint", func() {
		It("Should return the midpoint of the time range", func() {
			tr := v0.NewRangeSeconds(4, 12)
			Expect(tr.Midpoint()).To(Equal(v0.TimeStamp(8 * v0.Second)))
		})
	})

	Describe("Union", func() {

		Specify("Overlap, first before second", func() {
			tr := (0 * v0.SecondTS).Range(5 * v0.SecondTS)
			tr2 := (3 * v0.SecondTS).Range(8 * v0.SecondTS)
			union := tr.Union(tr2)
			Expect(union.Start).To(Equal(v0.TimeStamp(0)))
			Expect(union.End).To(Equal(8 * v0.SecondTS))
		})

		Specify("Overlap, second before first", func() {
			tr2 := (0 * v0.SecondTS).Range(5 * v0.SecondTS)
			tr := (3 * v0.SecondTS).Range(8 * v0.SecondTS)
			union := tr.Union(tr2)
			Expect(union.Start).To(Equal(v0.TimeStamp(0)))
			Expect(union.End).To(Equal(8 * v0.SecondTS))
		})

		Specify("1 Fully contain 2", func() {
			tr := (0 * v0.SecondTS).Range(10 * v0.SecondTS)
			tr2 := (3 * v0.SecondTS).Range(8 * v0.SecondTS)
			union := tr.Union(tr2)
			Expect(union.Start).To(Equal(0 * v0.SecondTS))
			Expect(union.End).To(Equal(10 * v0.SecondTS))
		})

		Specify("2 Fully contain 1", func() {
			tr := (2 * v0.SecondTS).Range(5 * v0.SecondTS)
			tr2 := (1 * v0.SecondTS).Range(8 * v0.SecondTS)
			union := tr.Union(tr2)
			Expect(union.Start).To(Equal(1 * v0.SecondTS))
			Expect(union.End).To(Equal(8 * v0.SecondTS))
		})

	})
})
