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
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func DataTypeInferTest[T any](expected telem.DataType) func() {
	return func() {
		dt := telem.InferDataType[T]()
		ExpectWithOffset(1, dt).To(Equal(expected))
	}
}

var _ = Describe("Telem", func() {

	Describe("TimeStamp", func() {

		Describe("Now", func() {
			It("Should return the current time", func() {
				Expect(telem.Now().Time()).To(BeTemporally("~", time.Now(), time.Millisecond))
			})
		})

		Describe("Stringer", func() {
			It("Should format a time properly", func() {
				ts := 90*telem.DayTS + 20*telem.MinuteTS + 283*telem.MillisecondTS + 900*telem.MicrosecondTS
				Expect(fmt.Sprintf("%v", ts)).To(Equal("1970-04-01T00:20:00.283Z"))
			})
			It("Should do EoT", func() {
				Expect(fmt.Sprintf("%v", telem.TimeStampMax)).To(Equal("end of time"))
			})
		})

		Describe("Name", func() {
			It("Should initialize a new timestamp based on the provided time", func() {
				t := time.Now()
				t0 := telem.NewTimeStamp(t)
				Expect(t0.Time()).To(BeTemporally("~", t, time.Millisecond))
			})
		})

		Describe("IsZero", func() {
			It("Should return true if the timestamp is zero", func() {
				Expect(telem.TimeStampMin.IsZero()).To(BeTrue())
				Expect(telem.TimeStampMax.IsZero()).To(BeFalse())
			})
		})

		Describe("Before", func() {
			It("Should return true if the timestamp is after the provided one", func() {
				Expect(telem.TimeStampMin.After(telem.TimeStampMax)).To(BeFalse())
				Expect(telem.TimeStampMax.After(telem.TimeStampMin)).To(BeTrue())
			})
			It("Should return false if the timestamp is equal to the provided one", func() {
				Expect(telem.TimeStampMin.After(telem.TimeStampMin)).To(BeFalse())
				Expect(telem.TimeStampMax.After(telem.TimeStampMax)).To(BeFalse())
			})
		})

		Describe("After", func() {
			It("Should return true if the timestamp is before the provided one", func() {
				Expect(telem.TimeStampMin.Before(telem.TimeStampMax)).To(BeTrue())
				Expect(telem.TimeStampMax.Before(telem.TimeStampMin)).To(BeFalse())
			})
			It("Should return false if the timestamp is equal to the provided one", func() {
				Expect(telem.TimeStampMin.Before(telem.TimeStampMin)).To(BeFalse())
				Expect(telem.TimeStampMax.Before(telem.TimeStampMax)).To(BeFalse())
			})
		})

		Describe("add", func() {
			It("Should return a new timestamp with the provided timespan added to it", func() {
				t0 := telem.TimeStamp(0)
				t1 := t0.Add(telem.Second)
				Expect(t1).To(Equal(telem.TimeStamp(1 * telem.Second)))
			})
		})

		Describe("sub", func() {
			It("Should return a new timestamp with the provided timespan subtracted from it", func() {
				t0 := telem.TimeStamp(0)
				t1 := t0.Sub(telem.Second)
				Expect(t1).To(Equal(telem.TimeStamp(-1 * telem.Second)))
			})
		})

		Describe("SpanRange", func() {
			It("Should return the correct time range", func() {
				t0 := telem.TimeStamp(0)
				r := t0.SpanRange(telem.Second)
				Expect(r.Start).To(Equal(t0))
				Expect(r.End).To(Equal(t0.Add(telem.Second)))
			})
			It("Should swap the start and end if the start is after the end", func() {
				t0 := telem.TimeStamp(0)
				r := telem.TimeStamp(telem.Second).SpanRange(-1 * telem.Second)
				Expect(r.Start).To(Equal(t0))
				Expect(r.End).To(Equal(t0.Add(telem.Second)))
			})
		})

		Describe("Bounds", func() {
			It("Should return the correct time range", func() {
				t0 := telem.TimeStamp(0)
				t1 := t0.Add(telem.Second)
				r := t0.Range(t1)
				Expect(r.Start).To(Equal(t0))
				Expect(r.End).To(Equal(t1))
			})
		})

		Describe("MarshalJSON", func() {
			It("Should marshal the time span into a string", func() {
				b := MustSucceed(json.Marshal(telem.TimeStamp(telem.Second)))
				Expect(string(b)).To(Equal("\"1000000000\""))
			})
		})

		Describe("Marshal + Unmarshal JSON", func() {
			It("Should unmarshal a time span from a number", func() {
				var ts telem.TimeStamp
				err := json.Unmarshal([]byte(`1000000000`), &ts)
				Expect(err).To(BeNil())
				Expect(ts).To(Equal(telem.TimeStamp(telem.Second)))
			})

			It("Should unmarshal a time span from a string", func() {
				var ts telem.TimeStamp
				err := json.Unmarshal([]byte(`"1000000000"`), &ts)
				Expect(err).To(BeNil())
				Expect(ts).To(Equal(telem.TimeStamp(telem.Second)))
			})
		})

	})

	Describe("TimeRange", func() {

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

		Describe("NewSecondsRange", func() {
			It("Should instantiate a time range between a particular starting number of seconds and ending number of seconds", func() {
				tr := telem.NewSecondsRange(1, 5)
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
				union := tr.Intersection(tr2)
				Expect(union.Start).To(Equal(3 * telem.SecondTS))
				Expect(union.End).To(Equal(5 * telem.SecondTS))
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
				union := tr.Intersection(tr2)
				Expect(union.Start).To(Equal(3 * telem.SecondTS))
				Expect(union.End).To(Equal(8 * telem.SecondTS))
			})
			Specify("2 Fully contain 1", func() {
				tr := (2 * telem.SecondTS).Range(5 * telem.SecondTS)
				tr2 := (1 * telem.SecondTS).Range(8 * telem.SecondTS)
				union := tr.Intersection(tr2)
				Expect(union.Start).To(Equal(2 * telem.SecondTS))
				Expect(union.End).To(Equal(5 * telem.SecondTS))
			})

		})

		DescribeTable("Point Intersection", func(
			tr telem.TimeRange,
			ts telem.TimeStamp,
			expectedBefore telem.TimeSpan,
			expectedAfter telem.TimeSpan,
		) {
			before, after := tr.PointIntersection(ts)
			Expect(before).To(Equal(expectedBefore))
			Expect(after).To(Equal(expectedAfter))
		},
			Entry(
				"Completely within",
				telem.NewSecondsRange(1, 5),
				telem.TimeStamp(telem.Second*3),
				telem.Second*2,
				telem.Second*2,
			),
			Entry("At Start",
				telem.NewSecondsRange(1, 5),
				telem.TimeStamp(telem.Second*1),
				telem.Second*0,
				telem.Second*4,
			),
			Entry("At End",
				telem.NewSecondsRange(1, 5),
				telem.TimeStamp(telem.Second*5),
				telem.Second*4,
				telem.Second*0,
			),
			Entry("Before Start",
				telem.NewSecondsRange(1, 5),
				telem.TimeStamp(0),
				telem.Second*-1,
				telem.Second*5,
			),
		)

		Describe("MakeValid", func() {
			It("Should swap the start and end timestamps if they are out of order", func() {
				tr := telem.NewSecondsRange(5, 1)
				Expect(tr.MakeValid()).To(Equal(telem.NewSecondsRange(1, 5)))
			})

			It("Should not swap the start and end timestamps if they are in order", func() {
				tr := telem.TimeRange{Start: 0, End: 1}
				Expect(tr.MakeValid()).To(Equal(telem.TimeRange{Start: 0, End: 1}))
			})
		})

		Describe("Midpoint", func() {
			It("Should return the midpoint of the time range", func() {
				tr := telem.NewSecondsRange(4, 12)
				Expect(tr.Midpoint()).To(Equal(telem.TimeStamp(8 * telem.Second)))
			})
		})

		Describe("MaxUnion", func() {
			Specify("Overlap, first before second", func() {
				tr := (0 * telem.SecondTS).Range(5 * telem.SecondTS)
				tr2 := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
				union := tr.MaxUnion(tr2)
				Expect(union.Start).To(Equal(telem.TimeStamp(0)))
				Expect(union.End).To(Equal(8 * telem.SecondTS))
			})
			Specify("Overlap, second before first", func() {
				tr2 := (0 * telem.SecondTS).Range(5 * telem.SecondTS)
				tr := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
				union := tr.MaxUnion(tr2)
				Expect(union.Start).To(Equal(telem.TimeStamp(0)))
				Expect(union.End).To(Equal(8 * telem.SecondTS))
			})
			Specify("1 Fully contain 2", func() {
				tr := (0 * telem.SecondTS).Range(10 * telem.SecondTS)
				tr2 := (3 * telem.SecondTS).Range(8 * telem.SecondTS)
				union := tr.MaxUnion(tr2)
				Expect(union.Start).To(Equal(0 * telem.SecondTS))
				Expect(union.End).To(Equal(10 * telem.SecondTS))
			})
			Specify("2 Fully contain 1", func() {
				tr := (2 * telem.SecondTS).Range(5 * telem.SecondTS)
				tr2 := (1 * telem.SecondTS).Range(8 * telem.SecondTS)
				union := tr.MaxUnion(tr2)
				Expect(union.Start).To(Equal(1 * telem.SecondTS))
				Expect(union.End).To(Equal(8 * telem.SecondTS))
			})
		})
	})

	Describe("TimeSpan", func() {

		Describe("Duration", func() {
			It("Should return the correct time span", func() {
				ts := telem.Second
				Expect(ts.Duration()).To(Equal(time.Second))
			})
		})

		Describe("Stringer", func() {
			DescribeTable("Should format a timespan properly", func(span telem.TimeSpan, expected string) {
				Expect(fmt.Sprintf("%v", span)).To(Equal(expected))
			},
				Entry("nano", 1*telem.Nanosecond, "1ns"),
				Entry("micro", 1*telem.Microsecond, "1µs"),
				Entry("milli", 1*telem.Millisecond, "1ms"),
				Entry("second", 1*telem.Second, "1s"),
				Entry("minute", 1*telem.Minute, "1m"),
				Entry("hour", 1*telem.Hour, "1h"),
				Entry("combine", 2*telem.Day+80*telem.Minute+1*telem.Millisecond+500*telem.Microsecond+5*telem.Nanosecond, "2d 1h 20m 1ms 500µs 5ns"),
				Entry("gap between unit levels", 2*telem.Hour+2*telem.Second, "2h 2s"),
			)
		})

		Describe("Seconds", func() {
			It("Should return the correct number of seconds in the span", func() {
				ts := telem.Millisecond
				Expect(ts.Seconds()).To(Equal(0.001))
			})
		})

		Describe("IsZero", func() {
			It("Should return true if the time span is zero", func() {
				Expect(telem.TimeSpanMax.IsZero()).To(BeFalse())
				Expect(telem.TimeSpanZero.IsZero()).To(BeTrue())
			})
		})

		Describe("IsMax", func() {
			It("Should return true if the time span is the maximum", func() {
				Expect(telem.TimeSpanMax.IsMax()).To(BeTrue())
				Expect(telem.TimeSpanZero.IsMax()).To(BeFalse())
			})
		})

		Describe("ByteSize", func() {
			It("Should return the correct byte size", func() {
				Expect(telem.Second.ByteSize(1, 8)).To(Equal(telem.Size(8)))
			})
		})

		Describe("Truncate", func() {
			It("Should Truncate to the nearest second", func() {
				ts := 1*telem.Second + 500*telem.Millisecond
				truncated := ts.Truncate(telem.Second)
				Expect(truncated).To(Equal(1 * telem.Second))
			})

			It("Should Truncate to the nearest minute", func() {
				ts := 1*telem.Minute + 30*telem.Second
				truncated := ts.Truncate(telem.Minute)
				Expect(truncated).To(Equal(1 * telem.Minute))
			})

			It("Should Truncate to the nearest hour", func() {
				ts := 1*telem.Hour + 30*telem.Minute
				truncated := ts.Truncate(telem.Hour)
				Expect(truncated).To(Equal(1 * telem.Hour))
			})

			It("Should Truncate to the nearest day", func() {
				ts := 1*telem.Day + 12*telem.Hour
				truncated := ts.Truncate(telem.Day)
				Expect(truncated).To(Equal(1 * telem.Day))
			})

			It("Should Truncate to the nearest millisecond", func() {
				ts := 1*telem.Millisecond + 500*telem.Microsecond
				truncated := ts.Truncate(telem.Millisecond)
				Expect(truncated).To(Equal(1 * telem.Millisecond))
			})

			It("Should Truncate to the nearest microsecond", func() {
				ts := 1*telem.Microsecond + 500*telem.Nanosecond
				truncated := ts.Truncate(telem.Microsecond)
				Expect(truncated).To(Equal(1 * telem.Microsecond))
			})

			It("Should handle zero values", func() {
				ts := telem.TimeSpanZero
				truncated := ts.Truncate(telem.Second)
				Expect(truncated).To(Equal(telem.TimeSpanZero))
			})

			It("Should handle negative values", func() {
				ts := -1*telem.Second - 500*telem.Millisecond
				truncated := ts.Truncate(telem.Second)
				Expect(truncated).To(Equal(-1 * telem.Second))
			})

			It("Should handle arbitrary units", func() {
				ts := 1234 * telem.Nanosecond
				truncated := ts.Truncate(100 * telem.Nanosecond)
				Expect(truncated).To(Equal(1200 * telem.Nanosecond))
			})

			It("Should truncate a compound set of units", func() {
				ts := 1*telem.Hour + telem.Second*30 + telem.Millisecond*500
				truncated := ts.Truncate(telem.Second)
				Expect(truncated).To(Equal(1*telem.Hour + telem.Second*30))
			})

			It("Should truncate microseconds", func() {
				ts := 1*telem.Second + 10*telem.Microsecond
				truncated := ts.Truncate(telem.Microsecond)
				Expect(truncated).To(Equal(1*telem.Second + 10*telem.Microsecond))
			})

			It("Should truncate a 0 time span", func() {
				ts := 0 * telem.Second
				truncated := ts.Truncate(telem.Second)
				Expect(truncated).To(Equal(0 * telem.Second))
			})

			It("Should handle a 0 truncation target", func() {
				ts := 1 * telem.Second
				truncated := ts.Truncate(0)
				Expect(truncated).To(Equal(1 * telem.Second))
			})
		})

		Describe("MarshalJSON", func() {
			It("Should marshal the time span into a string", func() {
				b := MustSucceed(json.Marshal(telem.Second))
				Expect(string(b)).To(Equal("\"1000000000\""))
			})
		})

		Describe("UnmarshalJSON", func() {
			It("Should unmarshal a time span from a number", func() {
				var ts telem.TimeSpan
				err := json.Unmarshal([]byte(`1000000000`), &ts)
				Expect(err).To(BeNil())
				Expect(ts).To(Equal(telem.Second))
			})

			It("Should unmarshal a time span from a string", func() {
				var ts telem.TimeSpan
				err := json.Unmarshal([]byte(`"1000000000"`), &ts)
				Expect(err).To(BeNil())
				Expect(ts).To(Equal(telem.Second))
			})
		})

	})

	Describe("Size", func() {
		Describe("Report", func() {
			It("Should return the correct string", func() {
				s := telem.Size(0)
				Expect(s.String()).To(Equal("0B"))
			})
		})
	})

	Describe("Rate", func() {
		Describe("Period", func() {
			It("Should return the correct period for the data rate", func() {
				Expect(telem.Rate(1).Period()).To(Equal(telem.Second))
			})
		})
		Describe("Distance", func() {
			It("Should return the number of samples that fit in the span", func() {
				Expect(telem.Rate(10).SampleCount(telem.Second)).To(Equal(10))
			})
		})
		Describe("SpanTo", func() {
			It("Should return the span of the provided samples", func() {
				Expect(telem.Rate(10).Span(10)).To(Equal(telem.Second))
			})
		})
		Describe("SizeSpan", func() {
			It("Should return the span of the provided number of bytes", func() {
				Expect(telem.Rate(10).SizeSpan(16, telem.Bit64)).To(Equal(200 * telem.Millisecond))
			})
		})
	})

	Describe("Marshal", func() {
		Specify("marshal int", func() {
			og := []int16{1, 2, 3, 4}
			marshalled := telem.MarshalSlice(og)
			unmarshalled := telem.UnmarshalSlice[int16](marshalled, telem.Int16T)
			Expect(og).To(Equal(unmarshalled))
		})
	})

	Describe("DataType", func() {
		Describe("Infer", func() {
			Specify("float64", DataTypeInferTest[float64](telem.Float64T))
			Specify("float32", DataTypeInferTest[float32](telem.Float32T))
			Specify("int64", DataTypeInferTest[int64](telem.Int64T))
			Specify("int32", DataTypeInferTest[int32](telem.Int32T))
			Specify("int16", DataTypeInferTest[int16](telem.Int16T))
			Specify("int8", DataTypeInferTest[int8](telem.Int8T))
			Specify("uint64", DataTypeInferTest[uint64](telem.Uint64T))
			Specify("uint32", DataTypeInferTest[uint32](telem.Uint32T))
			Specify("uint16", DataTypeInferTest[uint16](telem.Uint16T))
			Specify("uint8", DataTypeInferTest[uint8](telem.Uint8T))
			Specify("string", DataTypeInferTest[string](telem.StringT))

			It("Should panic if a a struct if provided", func() {
				Expect(func() {
					telem.InferDataType[struct{}]()
				}).To(Panic())
			})
		})

		DescribeTable("Density", func(dataType telem.DataType, expected telem.Density) {
			Expect(dataType.Density()).To(Equal(expected))
		},
			Entry("float64", telem.Float64T, telem.Bit64),
			Entry("float32", telem.Float32T, telem.Bit32),
			Entry("int64", telem.Int64T, telem.Bit64),
			Entry("int32", telem.Int32T, telem.Bit32),
			Entry("int16", telem.Int16T, telem.Bit16),
			Entry("int8", telem.Int8T, telem.Bit8),
			Entry("uint64", telem.Uint64T, telem.Bit64),
			Entry("uint32", telem.Uint32T, telem.Bit32),
			Entry("uint16", telem.Uint16T, telem.Bit16),
			Entry("uint8", telem.Uint8T, telem.Bit8),
			Entry("string", telem.StringT, telem.DensityUnknown),
			Entry("timestamp", telem.TimeStampT, telem.Bit64),
			Entry("uuid", telem.UUIDT, telem.Bit128),
		)
	})

	Describe("Alignment", func() {
		Describe("NewAlignment", func() {
			It("Should construct the alignment from the given domain and sample indexes", func() {
				align := telem.NewAlignment(2, 1)
				Expect(align.SampleIndex()).To(Equal(uint32(1)))
				Expect(align.DomainIndex()).To(Equal(uint32(2)))
			})
			It("Should construct a zero alignment", func() {
				Expect(uint64(telem.NewAlignment(0, 0))).To(Equal(uint64(0)))
			})
		})

		Describe("MarshalJSON", func() {
			It("Should marshal the alignment as a JSON string", func() {
				align := telem.NewAlignment(2, 1)
				marshalled := MustSucceed(align.MarshalJSON())
				Expect(string(marshalled)).To(Equal(fmt.Sprintf("\"%v\"", uint64(align))))
			})
		})

		Describe("UnmarshalJSON", func() {
			It("Should unmarshal the alignment from a JSON string", func() {
				align := telem.NewAlignment(2, 1)
				marshalled := MustSucceed(align.MarshalJSON())
				var unmarshalled telem.Alignment
				Expect(unmarshalled.UnmarshalJSON(marshalled)).To(Succeed())
				Expect(unmarshalled).To(Equal(align))
			})
		})

		Describe("AddSamples", func() {
			It("Should add to the alignment sample index", func() {
				align := telem.NewAlignment(2, 1)
				align = align.AddSamples(3)
				Expect(align.SampleIndex()).To(Equal(uint32(4)))
			})
		})

		Describe("LeadingAlignment", func() {
			It("Should return the global leading alignment standard", func() {
				align := telem.LeadingAlignment(2, 1)
				Expect(align.DomainIndex()).To(Equal(telem.ZeroLeadingAlignment + 2))
			})
		})
	})
})
