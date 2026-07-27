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

var _ = Describe("TimeStamp", func() {
	Describe("Now", func() {
		It("Should return the current time", func() {
			Expect(telem.Now().Time()).To(BeTemporally("~", time.Now(), time.Millisecond))
		})
	})

	Describe("NewTimeStamp", func() {
		It("Should construct a timestamp with nanosecond precision from a time.Time", func() {
			t := time.Now()
			ts := telem.NewTimeStamp(t)
			Expect(ts).To(Equal(telem.TimeStamp(t.UnixNano())))
			Expect(ts.Time()).To(BeTemporally("==", t))
		})
	})

	Describe("Since", func() {
		It("Should return a positive TimeSpan for a timestamp in the past", func() {
			past := telem.Now().Add(-telem.Second)
			elapsed := telem.Since(past)
			Expect(elapsed).To(BeNumerically(">=", telem.Second))
			Expect(elapsed).To(BeNumerically("<", 2*telem.Second))
		})

		It("Should return approximately zero for a timestamp equal to now", func() {
			now := telem.Now()
			elapsed := telem.Since(now)
			Expect(elapsed).To(BeNumerically(">=", 0))
			Expect(elapsed).To(BeNumerically("<", telem.Millisecond))
		})

		It("Should return a negative TimeSpan for a timestamp in the future", func() {
			future := telem.Now().Add(telem.Second)
			elapsed := telem.Since(future)
			Expect(elapsed).To(BeNumerically("<", 0))
			Expect(elapsed).To(BeNumerically(">", -2*telem.Second))
		})
	})
})
