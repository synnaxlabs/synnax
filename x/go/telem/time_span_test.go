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
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("TimeSpan", func() {
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
			Entry("zero", 0*telem.Nanosecond, "0s"),
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
			Expect(string(b)).To(Equal(`"1000000000"`))
		})
	})

	Describe("UnmarshalJSON", func() {
		It("Should unmarshal a time span from a number", func() {
			var ts telem.TimeSpan
			err := json.Unmarshal([]byte("1000000000"), &ts)
			Expect(err).To(BeNil())
			Expect(ts).To(Equal(telem.Second))
		})

		It("Should unmarshal a time span from a string", func() {
			var ts telem.TimeSpan
			err := json.Unmarshal([]byte("1000000000"), &ts)
			Expect(err).To(BeNil())
			Expect(ts).To(Equal(telem.Second))
		})
	})
})
