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

var _ = Describe("TimeStamp", func() {
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

	Describe("Span", func() {
		It("Should return the time span between two timestamps", func() {
			span := (telem.SecondTS * 5).Span(telem.SecondTS * 20)
			Expect(span).To(Equal(telem.Second * 15))
		})

		It("Should work correctly when the arg timestamp is before the original timestamp", func() {
			span := (telem.SecondTS * 20).Span(telem.SecondTS * 5)
			Expect(span).To(Equal(-telem.Second * 15))
		})
	})

	Describe("MarshalJSON", func() {
		It("Should marshal the time stamp into a string", func() {
			b := MustSucceed(json.Marshal(telem.TimeStamp(telem.Second)))
			Expect(string(b)).To(Equal(`"1000000000"`))
		})
	})

	Describe("UnmarshalJSON", func() {
		It("Should unmarshal a time stamp from a number", func() {
			var ts telem.TimeStamp
			err := json.Unmarshal([]byte("1000000000"), &ts)
			Expect(err).To(BeNil())
			Expect(ts).To(Equal(telem.TimeStamp(telem.Second)))
		})

		It("Should unmarshal a time stamp from a string", func() {
			var ts telem.TimeStamp
			err := json.Unmarshal([]byte("1000000000"), &ts)
			Expect(err).To(BeNil())
			Expect(ts).To(Equal(telem.TimeStamp(telem.Second)))
		})
	})
})
