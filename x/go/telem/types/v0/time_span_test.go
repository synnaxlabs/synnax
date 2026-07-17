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
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v0 "github.com/synnaxlabs/x/telem/types/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("TimeSpan", func() {
	Describe("Duration", func() {
		It("Should return the correct time span", func() {
			ts := v0.Second
			Expect(ts.Duration()).To(Equal(time.Second))
		})
	})

	Describe("Stringer", func() {
		DescribeTable("Should format a timespan properly", func(span v0.TimeSpan, expected string) {
			Expect(fmt.Sprintf("%v", span)).To(Equal(expected))
		},
			Entry("zero", 0*v0.Nanosecond, "0s"),
			Entry("nano", 1*v0.Nanosecond, "1ns"),
			Entry("micro", 1*v0.Microsecond, "1µs"),
			Entry("milli", 1*v0.Millisecond, "1ms"),
			Entry("second", 1*v0.Second, "1s"),
			Entry("minute", 1*v0.Minute, "1m"),
			Entry("hour", 1*v0.Hour, "1h"),
			Entry("combine", 2*v0.Day+80*v0.Minute+1*v0.Millisecond+500*v0.Microsecond+5*v0.Nanosecond, "2d 1h 20m 1ms 500µs 5ns"),
			Entry("gap between unit levels", 2*v0.Hour+2*v0.Second, "2h 2s"),
		)
	})

	Describe("Seconds", func() {
		It("Should return the correct number of seconds in the span", func() {
			ts := v0.Millisecond
			Expect(ts.Seconds()).To(Equal(0.001))
		})
	})

	Describe("IsZero", func() {
		It("Should return true if the time span is zero", func() {
			Expect(v0.TimeSpanMax.IsZero()).To(BeFalse())
			Expect(v0.TimeSpanZero.IsZero()).To(BeTrue())
		})
	})

	Describe("IsMax", func() {
		It("Should return true if the time span is the maximum", func() {
			Expect(v0.TimeSpanMax.IsMax()).To(BeTrue())
			Expect(v0.TimeSpanZero.IsMax()).To(BeFalse())
		})
	})

	Describe("ByteSize", func() {
		It("Should return the correct byte size", func() {
			Expect(v0.Second.ByteSize(1, 8)).To(Equal(v0.Size(8)))
		})
	})

	Describe("Truncate", func() {
		It("Should Truncate to the nearest second", func() {
			ts := 1*v0.Second + 500*v0.Millisecond
			truncated := ts.Truncate(v0.Second)
			Expect(truncated).To(Equal(1 * v0.Second))
		})

		It("Should Truncate to the nearest minute", func() {
			ts := 1*v0.Minute + 30*v0.Second
			truncated := ts.Truncate(v0.Minute)
			Expect(truncated).To(Equal(1 * v0.Minute))
		})

		It("Should Truncate to the nearest hour", func() {
			ts := 1*v0.Hour + 30*v0.Minute
			truncated := ts.Truncate(v0.Hour)
			Expect(truncated).To(Equal(1 * v0.Hour))
		})

		It("Should Truncate to the nearest day", func() {
			ts := 1*v0.Day + 12*v0.Hour
			truncated := ts.Truncate(v0.Day)
			Expect(truncated).To(Equal(1 * v0.Day))
		})

		It("Should Truncate to the nearest millisecond", func() {
			ts := 1*v0.Millisecond + 500*v0.Microsecond
			truncated := ts.Truncate(v0.Millisecond)
			Expect(truncated).To(Equal(1 * v0.Millisecond))
		})

		It("Should Truncate to the nearest microsecond", func() {
			ts := 1*v0.Microsecond + 500*v0.Nanosecond
			truncated := ts.Truncate(v0.Microsecond)
			Expect(truncated).To(Equal(1 * v0.Microsecond))
		})

		It("Should handle zero values", func() {
			ts := v0.TimeSpanZero
			truncated := ts.Truncate(v0.Second)
			Expect(truncated).To(Equal(v0.TimeSpanZero))
		})

		It("Should handle negative values", func() {
			ts := -1*v0.Second - 500*v0.Millisecond
			truncated := ts.Truncate(v0.Second)
			Expect(truncated).To(Equal(-1 * v0.Second))
		})

		It("Should handle arbitrary units", func() {
			ts := 1234 * v0.Nanosecond
			truncated := ts.Truncate(100 * v0.Nanosecond)
			Expect(truncated).To(Equal(1200 * v0.Nanosecond))
		})

		It("Should truncate a compound set of units", func() {
			ts := 1*v0.Hour + v0.Second*30 + v0.Millisecond*500
			truncated := ts.Truncate(v0.Second)
			Expect(truncated).To(Equal(1*v0.Hour + v0.Second*30))
		})

		It("Should truncate microseconds", func() {
			ts := 1*v0.Second + 10*v0.Microsecond
			truncated := ts.Truncate(v0.Microsecond)
			Expect(truncated).To(Equal(1*v0.Second + 10*v0.Microsecond))
		})

		It("Should truncate a 0 time span", func() {
			ts := 0 * v0.Second
			truncated := ts.Truncate(v0.Second)
			Expect(truncated).To(Equal(0 * v0.Second))
		})

		It("Should handle a 0 truncation target", func() {
			ts := 1 * v0.Second
			truncated := ts.Truncate(0)
			Expect(truncated).To(Equal(1 * v0.Second))
		})
	})

	Describe("MarshalJSON", func() {
		It("Should marshal the time span into a string", func() {
			b := MustSucceed(json.Marshal(v0.Second))
			Expect(string(b)).To(Equal(`"1000000000"`))
		})
	})

	Describe("UnmarshalJSON", func() {
		It("Should unmarshal a time span from a number", func() {
			var ts v0.TimeSpan
			Expect(json.Unmarshal([]byte("1000000000"), &ts)).To(Succeed())
			Expect(ts).To(Equal(v0.Second))
		})

		It("Should unmarshal a time span from a string", func() {
			var ts v0.TimeSpan
			Expect(json.Unmarshal([]byte(`"1000000000"`), &ts)).To(Succeed())
			Expect(ts).To(Equal(v0.Second))
		})

		It("Should return an error and leave the time span untouched on invalid input", func() {
			ts := v0.Second
			Expect(json.Unmarshal([]byte(`"not-a-number"`), &ts)).To(
				MatchError(ContainSubstring("invalid syntax")),
			)
			Expect(ts).To(Equal(v0.Second))
		})
	})
})
