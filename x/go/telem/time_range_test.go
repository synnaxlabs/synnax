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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("TimeRange", func() {
	Describe("Stringer", func() {
		DescribeTable("Should format time ranges with appropriate precision",
			func(start, end time.Time, expected string) {
				tr := telem.TimeRange{
					Start: telem.NewTimeStamp(start),
					End:   telem.NewTimeStamp(end),
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
			tr := telem.NewRangeSeconds(1, 5)
			Expect(tr.Start).To(Equal(telem.SecondTS * 1))
			Expect(tr.End).To(Equal(telem.SecondTS * 5))
		})
	})

	Describe("SpanTo", func() {
		It("Should return the correct time span", func() {
			tr := telem.TimeRange{
				Start: telem.TimeStamp(0),
				End:   telem.TimeStamp(telem.Second),
			}
			Expect(tr.Span()).To(Equal(telem.Second))
		})
	})

	Describe("IsZero", func() {
		It("Should return true if the time range is zero", func() {
			Expect(telem.TimeRangeMin.IsZero()).To(BeFalse())
			Expect(telem.TimeRangeMax.IsZero()).To(BeFalse())
			Expect(telem.TimeRangeZero.IsZero()).To(BeTrue())
		})
	})

	Describe("BoundBy", func() {

		It("Should bound the time range to the provided constraints", func() {
			tr := telem.TimeRange{
				Start: telem.TimeStamp(telem.Second),
				End:   telem.TimeStamp(telem.Second * 4),
			}
			bound := telem.TimeRange{
				Start: telem.TimeStamp(2 * telem.Second),
				End:   telem.TimeStamp(telem.Second * 3),
			}
			bounded := tr.BoundBy(bound)
			Expect(bounded.Start).To(Equal(bound.Start))
			Expect(bounded.End).To(Equal(bound.End))

		})

		It("Should bound the time range even if the start is after the end", func() {

			tr := telem.TimeRange{
				Start: telem.TimeStamp(telem.Second * 4),
				End:   telem.TimeStamp(telem.Second),
			}

			bound := telem.TimeRange{
				Start: telem.TimeStamp(2 * telem.Second),
				End:   telem.TimeStamp(telem.Second * 3),
			}

			bounded := tr.BoundBy(bound)
			Expect(bounded.Start).To(Equal(bound.End))
			Expect(bounded.End).To(Equal(bound.Start))
		})

		It("Should bound the time range even if there is zero overlap", func() {
			tr := telem.TimeRange{
				Start: telem.TimeStamp(telem.Second * 10),
				End:   telem.TimeStamp(telem.Second * 14),
			}

			bound := telem.TimeRange{
				Start: telem.TimeStamp(2 * telem.Second),
				End:   telem.TimeStamp(telem.Second * 3),
			}

			bounded := tr.BoundBy(bound)
			Expect(bounded.Start).To(Equal(bound.End))
			Expect(bounded.End).To(Equal(bound.End))
		})
	})

	Describe("ContainsStamp", func() {
		It("Should return true when the range contains the timestamp", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			Expect(tr.ContainsStamp(telem.TimeStamp(4 * telem.Second))).To(BeTrue())
			By("Being inclusive at the lower bound")
			Expect(tr.ContainsStamp(telem.TimeStamp(0 * telem.Second))).To(BeTrue())
			By("Being exclusive at the upper bound")
			Expect(tr.ContainsStamp(telem.TimeStamp(5 * telem.Second))).To(BeFalse())
		})
	})

	Describe("ContainsRange", func() {
		It("Should return true when the ranges overlap with one another", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			Expect(tr.ContainsRange(telem.TimeStamp(1).SpanRange(2 * telem.Second))).To(BeTrue())
		})
		It("Should return false when the start of one range is the end of another", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			tr2 := telem.TimeStamp(5 * telem.Second).SpanRange(5 * telem.Second)
			Expect(tr.ContainsRange(tr2)).To(BeFalse())
			Expect(tr2.ContainsRange(tr)).To(BeFalse())
		})
		It("Should return true if checked against itself", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			Expect(tr.ContainsRange(tr))
		})
	})

	Describe("WhereOverlapsWith", func() {
		It("Should return true when the ranges overlap with one another", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			Expect(tr.OverlapsWith(telem.TimeStamp(1).SpanRange(2 * telem.Second))).To(BeTrue())
		})

		It("Should return false when the start of one range is the end of another", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			tr2 := telem.TimeStamp(5 * telem.Second).SpanRange(5 * telem.Second)
			Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			Expect(tr2.OverlapsWith(tr)).To(BeFalse())
		})

		It("Should return true if the start timestamps of the two ranges are equal", func() {
			tr1 := telem.TimeStamp(5 * telem.Second).SpanRange(5 * telem.Second)
			tr2 := telem.TimeStamp(5 * telem.Second).SpanRange(10 * telem.Second)
			Expect(tr1.OverlapsWith(tr2)).To(BeTrue())
		})

		It("Should return true if checked against itself", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			Expect(tr.OverlapsWith(tr))
		})

		It("Should return false if the ranges do not overlap", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			tr2 := telem.TimeStamp(5 * telem.Second).SpanRange(5 * telem.Second)
			Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			Expect(tr2.OverlapsWith(tr)).To(BeFalse())
		})

		It("Should return true if one range is contained within the other", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			tr2 := telem.TimeStamp(1).SpanRange(2 * telem.Second)
			Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			Expect(tr2.OverlapsWith(tr)).To(BeTrue())
		})

		Context("Main TimeRange is not point, argument is point", func() {
			It("Should return true if the point is contained", func() {
				tr := telem.NewRangeSeconds(1, 3)
				tr2 := telem.NewRangeSeconds(2, 2)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})

			It("Should return false if the point is NOT contained", func() {
				tr := telem.NewRangeSeconds(1, 3)
				tr2 := telem.NewRangeSeconds(4, 4)
				Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			})

			It("Should return false if the point is at the end", func() {
				tr := telem.NewRangeSeconds(1, 3)
				tr2 := telem.NewRangeSeconds(3, 3)
				Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			})

			It("Should return true if the point is at the start", func() {
				tr := telem.NewRangeSeconds(1, 3)
				tr2 := telem.NewRangeSeconds(1, 1)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})
		})

		Context("Main TimeRange is point, argument is point", func() {
			It("Should return true if the points are identical", func() {
				tr := telem.NewRangeSeconds(1, 1)
				tr2 := telem.NewRangeSeconds(1, 1)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})

			It("Should return false if the points are NOT identical", func() {
				tr := telem.NewRangeSeconds(1, 1)
				tr2 := telem.NewRangeSeconds(2, 2)
				Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			})
		})

		Context("Main TimeRange is point, argument is not point", func() {
			It("Should return true if the argument contains the point", func() {
				tr := telem.NewRangeSeconds(2, 2)
				tr2 := telem.NewRangeSeconds(1, 3)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})

			It("Should return false if point is at the end of the argument", func() {
				tr := telem.NewRangeSeconds(3, 3)
				tr2 := telem.NewRangeSeconds(1, 3)
				Expect(tr.OverlapsWith(tr2)).To(BeFalse())
			})

			It("Should return true if point is at the start", func() {
				tr := telem.NewRangeSeconds(3, 3)
				tr2 := telem.NewRangeSeconds(3, 5)
				Expect(tr.OverlapsWith(tr2)).To(BeTrue())
			})
		})

	})

	Describe("Swap", func() {
		It("Should swap the start and end times", func() {
			tr := telem.TimeStamp(0).SpanRange(5 * telem.Second)
			Expect(tr.Start).To(Equal(telem.TimeStamp(0)))
			Expect(tr.End).To(Equal(telem.TimeStamp(5 * telem.Second)))
			tr = tr.Swap()
			Expect(tr.Start).To(Equal(telem.TimeStamp(5 * telem.Second)))
			Expect(tr.End).To(Equal(telem.TimeStamp(0)))
		})
	})

	Describe("Intersection", func() {
		Specify("Overlap, first before second", func() {
			tr := (0 * telem.SecondTS).Range(5 * telem.SecondTS)
			tr2 := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
			intersection := tr.Intersection(tr2)
			Expect(intersection.Start).To(Equal(3 * telem.SecondTS))
			Expect(intersection.End).To(Equal(5 * telem.SecondTS))
		})

		Specify("Overlap, second before first", func() {
			tr2 := (0 * telem.SecondTS).Range(5 * telem.SecondTS)
			tr := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
			union := tr.Intersection(tr2)
			Expect(union.Start).To(Equal(3 * telem.SecondTS))
			Expect(union.End).To(Equal(5 * telem.SecondTS))
		})

		Specify("1 Fully contain 2", func() {
			tr := (0 * telem.SecondTS).Range(10 * telem.SecondTS)
			tr2 := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
			intersection := tr.Intersection(tr2)
			Expect(intersection.Start).To(Equal(3 * telem.SecondTS))
			Expect(intersection.End).To(Equal(8 * telem.SecondTS))
		})

		Specify("2 Fully contain 1", func() {
			tr := (2 * telem.SecondTS).Range(5 * telem.SecondTS)
			tr2 := (1 * telem.SecondTS).Range(8 * telem.SecondTS)
			intersection := tr.Intersection(tr2)
			Expect(intersection.Start).To(Equal(2 * telem.SecondTS))
			Expect(intersection.End).To(Equal(5 * telem.SecondTS))
		})

		Specify("Completely separate", func() {
			tr := (0 * telem.SecondTS).Range(10 * telem.SecondTS)
			tr2 := (100 * telem.SecondTS).Range(108 * telem.SecondTS)
			intersection := tr.Intersection(tr2)
			Expect(intersection).To(Equal(telem.TimeRangeZero))
		})

	})

	DescribeTable("Split", func(
		tr telem.TimeRange,
		ts telem.TimeStamp,
		expectedBefore telem.TimeSpan,
		expectedAfter telem.TimeSpan,
	) {
		before, after := tr.Split(ts)
		Expect(before.Span()).To(Equal(expectedBefore))
		Expect(after.Span()).To(Equal(expectedAfter))
	},
		Entry(
			"Completely within",
			telem.NewRangeSeconds(1, 5),
			telem.TimeStamp(telem.Second*3),
			telem.Second*2,
			telem.Second*2,
		),
		Entry("At Start",
			telem.NewRangeSeconds(1, 5),
			telem.TimeStamp(telem.Second*1),
			telem.Second*0,
			telem.Second*4,
		),
		Entry("At End",
			telem.NewRangeSeconds(1, 5),
			telem.TimeStamp(telem.Second*5),
			telem.Second*4,
			telem.Second*0,
		),
		Entry("Before Start",
			telem.NewRangeSeconds(1, 5),
			telem.TimeStamp(0),
			telem.Second*-1,
			telem.Second*5,
		),
		Entry("After End",
			telem.NewRangeSeconds(1, 5),
			telem.TimeStamp(telem.Second*20),
			telem.Second*19,
			-telem.Second*15,
		),
	)

	Describe("MakeValid", func() {
		It("Should swap the start and end timestamps if they are out of order", func() {
			tr := telem.NewRangeSeconds(5, 1)
			Expect(tr.MakeValid()).To(Equal(telem.NewRangeSeconds(1, 5)))
		})

		It("Should not swap the start and end timestamps if they are in order", func() {
			tr := telem.TimeRange{Start: 0, End: 1}
			Expect(tr.MakeValid()).To(Equal(telem.TimeRange{Start: 0, End: 1}))
		})
	})

	Describe("Midpoint", func() {
		It("Should return the midpoint of the time range", func() {
			tr := telem.NewRangeSeconds(4, 12)
			Expect(tr.Midpoint()).To(Equal(telem.TimeStamp(8 * telem.Second)))
		})
	})

	Describe("Union", func() {

		Specify("Overlap, first before second", func() {
			tr := (0 * telem.SecondTS).Range(5 * telem.SecondTS)
			tr2 := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
			union := tr.Union(tr2)
			Expect(union.Start).To(Equal(telem.TimeStamp(0)))
			Expect(union.End).To(Equal(8 * telem.SecondTS))
		})

		Specify("Overlap, second before first", func() {
			tr2 := (0 * telem.SecondTS).Range(5 * telem.SecondTS)
			tr := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
			union := tr.Union(tr2)
			Expect(union.Start).To(Equal(telem.TimeStamp(0)))
			Expect(union.End).To(Equal(8 * telem.SecondTS))
		})

		Specify("1 Fully contain 2", func() {
			tr := (0 * telem.SecondTS).Range(10 * telem.SecondTS)
			tr2 := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
			union := tr.Union(tr2)
			Expect(union.Start).To(Equal(0 * telem.SecondTS))
			Expect(union.End).To(Equal(10 * telem.SecondTS))
		})

		Specify("2 Fully contain 1", func() {
			tr := (2 * telem.SecondTS).Range(5 * telem.SecondTS)
			tr2 := (1 * telem.SecondTS).Range(8 * telem.SecondTS)
			union := tr.Union(tr2)
			Expect(union.Start).To(Equal(1 * telem.SecondTS))
			Expect(union.End).To(Equal(8 * telem.SecondTS))
		})

	})
})
