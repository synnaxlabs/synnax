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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/telem/types/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("TimeStamp", func() {
	Describe("Stringer", func() {
		It("Should format a time properly", func() {
			ts := 90*v0.DayTS + 20*v0.MinuteTS + 283*v0.MillisecondTS + 900*v0.MicrosecondTS
			Expect(fmt.Sprintf("%v", ts)).To(Equal("1970-04-01T00:20:00.283Z"))
		})
		It("Should do EoT", func() {
			Expect(fmt.Sprintf("%v", v0.TimeStampMax)).To(Equal("end of time"))
		})
	})

	Describe("IsZero", func() {
		It("Should return true if the timestamp is zero", func() {
			Expect(v0.TimeStampMin.IsZero()).To(BeTrue())
			Expect(v0.TimeStampMax.IsZero()).To(BeFalse())
		})
	})

	Describe("Before", func() {
		It("Should return true if the timestamp is after the provided one", func() {
			Expect(v0.TimeStampMin.After(v0.TimeStampMax)).To(BeFalse())
			Expect(v0.TimeStampMax.After(v0.TimeStampMin)).To(BeTrue())
		})
		It("Should return false if the timestamp is equal to the provided one", func() {
			Expect(v0.TimeStampMin.After(v0.TimeStampMin)).To(BeFalse())
			Expect(v0.TimeStampMax.After(v0.TimeStampMax)).To(BeFalse())
		})
	})

	Describe("After", func() {
		It("Should return true if the timestamp is before the provided one", func() {
			Expect(v0.TimeStampMin.Before(v0.TimeStampMax)).To(BeTrue())
			Expect(v0.TimeStampMax.Before(v0.TimeStampMin)).To(BeFalse())
		})
		It("Should return false if the timestamp is equal to the provided one", func() {
			Expect(v0.TimeStampMin.Before(v0.TimeStampMin)).To(BeFalse())
			Expect(v0.TimeStampMax.Before(v0.TimeStampMax)).To(BeFalse())
		})
	})

	Describe("add", func() {
		It("Should return a new timestamp with the provided timespan added to it", func() {
			t0 := v0.TimeStamp(0)
			t1 := t0.Add(v0.Second)
			Expect(t1).To(Equal(v0.TimeStamp(1 * v0.Second)))
		})
	})

	Describe("sub", func() {
		It("Should return a new timestamp with the provided timespan subtracted from it", func() {
			t0 := v0.TimeStamp(0)
			t1 := t0.Sub(v0.Second)
			Expect(t1).To(Equal(v0.TimeStamp(-1 * v0.Second)))
		})
	})

	Describe("SpanRange", func() {
		It("Should return the correct time range", func() {
			t0 := v0.TimeStamp(0)
			r := t0.SpanRange(v0.Second)
			Expect(r.Start).To(Equal(t0))
			Expect(r.End).To(Equal(t0.Add(v0.Second)))
		})
		It("Should swap the start and end if the start is after the end", func() {
			t0 := v0.TimeStamp(0)
			r := v0.TimeStamp(v0.Second).SpanRange(-1 * v0.Second)
			Expect(r.Start).To(Equal(t0))
			Expect(r.End).To(Equal(t0.Add(v0.Second)))
		})
	})

	Describe("Bounds", func() {
		It("Should return the correct time range", func() {
			t0 := v0.TimeStamp(0)
			t1 := t0.Add(v0.Second)
			r := t0.Range(t1)
			Expect(r.Start).To(Equal(t0))
			Expect(r.End).To(Equal(t1))
		})
	})

	Describe("Span", func() {
		It("Should return the time span between two timestamps", func() {
			span := (v0.SecondTS * 5).Span(v0.SecondTS * 20)
			Expect(span).To(Equal(v0.Second * 15))
		})

		It("Should work correctly when the arg timestamp is before the original timestamp", func() {
			span := (v0.SecondTS * 20).Span(v0.SecondTS * 5)
			Expect(span).To(Equal(-v0.Second * 15))
		})
	})

	Describe("MarshalJSON", func() {
		It("Should marshal the time stamp into a string", func() {
			b := MustSucceed(json.Marshal(v0.TimeStamp(v0.Second)))
			Expect(string(b)).To(Equal(`"1000000000"`))
		})
	})

	Describe("UnmarshalJSON", func() {
		It("Should unmarshal a time stamp from a number", func() {
			var ts v0.TimeStamp
			Expect(json.Unmarshal([]byte("1000000000"), &ts)).To(Succeed())
			Expect(ts).To(Equal(v0.TimeStamp(v0.Second)))
		})

		It("Should unmarshal a time stamp from a string", func() {
			var ts v0.TimeStamp
			Expect(json.Unmarshal([]byte(`"1000000000"`), &ts)).To(Succeed())
			Expect(ts).To(Equal(v0.TimeStamp(v0.Second)))
		})

		It("Should return an error and leave the time stamp untouched on invalid input", func() {
			ts := v0.TimeStamp(v0.Second)
			Expect(json.Unmarshal([]byte(`"not-a-number"`), &ts)).To(
				MatchError(ContainSubstring("invalid syntax")),
			)
			Expect(ts).To(Equal(v0.TimeStamp(v0.Second)))
		})
	})
})
