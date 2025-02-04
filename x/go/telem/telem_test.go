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
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	"time"
)

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

	})

	Describe("TimeRange", func() {

		Describe("Stringer", func() {
			It("Should format a time range properly", func() {
				ts1 := 2*telem.DayTS + 20*telem.MinuteTS + 283*telem.MillisecondTS + 900*telem.MicrosecondTS
				ts2 := 4*telem.DayTS + 20*telem.MinuteTS + 283*telem.MillisecondTS + 900*telem.MicrosecondTS
				Expect(fmt.Sprintf("%v", ts1.Range(ts2))).To(Equal("1970-01-03T00:20:00.283Z - 1970-01-05T00:20:00.283Z"))
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
					Entry("max", telem.TimeSpanMax, "max time span"),
					Entry("combine", 2*telem.Day+80*telem.Minute+1*telem.Millisecond+500*telem.Microsecond+5*telem.Nanosecond, "2d 1h 20m 1ms 500µs 5ns"),
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
		})

		Describe("size", func() {
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
	})
	Describe("Marshal", func() {
		Specify("marshal int", func() {
			og := []int16{1, 2, 3, 4}
			marshalled := telem.MarshalSlice(og, telem.Int16T)
			unmarshalled := telem.UnmarshalSlice[int16](marshalled, telem.Int16T)
			Expect(og).To(Equal(unmarshalled))
		})
	})
})
