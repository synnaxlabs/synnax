// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package time_test

import (
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xtime "github.com/synnaxlabs/x/time"
)

var _ = Describe("Clock", func() {
	Describe("Real", func() {
		It("Should return a value bracketed by time.Now", func() {
			before := time.Now()
			got := xtime.Real.Now()
			after := time.Now()
			Expect(got).To(BeTemporally(">=", before))
			Expect(got).To(BeTemporally("<=", after))
		})
		It("Should call RunAt functions once the deadline passes", func() {
			done := make(chan struct{})
			xtime.Real.RunAt(time.Now().Add(5*time.Millisecond), func() {
				close(done)
			})
			Eventually(done, time.Millisecond*100).Should(BeClosed())
		})
		It("Should call RunAt functions with a past deadline", func() {
			done := make(chan struct{})
			xtime.Real.RunAt(time.Now().Add(-time.Hour), func() { close(done) })
			Eventually(done, time.Millisecond*100).Should(BeClosed())
		})
		It("Should let Stop cancel a pending RunAt timer", func() {
			var called atomic.Bool
			t := xtime.Real.RunAt(time.Now().Add(time.Hour), func() {
				called.Store(true)
			})
			Expect(t.Stop()).To(BeTrue())
			Consistently(called.Load, time.Millisecond*20).Should(BeFalse())
		})
	})

	Describe("Fake", func() {
		It("Should start at the zero time", func() {
			Expect((&xtime.Fake{}).Now()).To(Equal(time.Time{}))
		})
		It("Should advance by the supplied duration", func() {
			f := &xtime.Fake{}
			f.Advance(5 * time.Second)
			f.Advance(2 * time.Second)
			Expect(f.Now()).To(Equal(time.Time{}.Add(7 * time.Second)))
		})

		Describe("RunAt", func() {
			It("Should call scheduled functions when Advance crosses the deadline",
				func() {
					f := &xtime.Fake{}
					done := make(chan struct{})
					f.RunAt(time.Time{}.Add(time.Second), func() { close(done) })
					go f.Advance(2 * time.Second)
					Eventually(done).Should(BeClosed())
				},
			)
			It("Should call scheduled functions whose deadline has already passed",
				func() {
					f := &xtime.Fake{}
					done := make(chan struct{})
					f.Advance(2 * time.Second)
					f.RunAt(time.Time{}.Add(time.Second), func() { close(done) })
					Eventually(done).Should(BeClosed())
				},
			)
			It("Should not call functions whose deadline has not been crossed", func() {
				f := &xtime.Fake{}
				var called atomic.Int32
				t := f.RunAt(time.Time{}.Add(time.Second), func() {
					called.Add(1)
				})
				DeferCleanup(func() { t.Stop() })
				f.Advance(500 * time.Millisecond)
				Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(0)))
			})
			It("Should call previously-scheduled functions on a later Advance", func() {
				f := &xtime.Fake{}
				done := make(chan struct{})
				f.RunAt(time.Time{}.Add(time.Second), func() { close(done) })
				f.Advance(500 * time.Millisecond)
				go f.Advance(700 * time.Millisecond)
				Eventually(done).Should(BeClosed())
			})
			It("Should let Stop cancel a pending timer", func() {
				f := &xtime.Fake{}
				var called atomic.Int32
				t := f.RunAt(time.Time{}.Add(time.Second), func() {
					called.Add(1)
				})
				Expect(t.Stop()).To(BeTrue())
				go f.Advance(2 * time.Second)
				Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(0)))
			})
			It("Should call only functions whose deadline has been crossed", func() {
				f := &xtime.Fake{}
				earlyFired := make(chan struct{})
				var late atomic.Int32
				early := f.RunAt(time.Time{}.Add(time.Second), func() {
					close(earlyFired)
				})
				lateT := f.RunAt(time.Time{}.Add(10*time.Second), func() {
					late.Add(1)
				})
				DeferCleanup(func() { early.Stop(); lateT.Stop() })
				go f.Advance(2 * time.Second)
				Eventually(earlyFired).Should(BeClosed())
				Consistently(late.Load, time.Millisecond*20).Should(Equal(int32(0)))
			})
			It("Should return false from Stop once the timer has fired", func() {
				f := &xtime.Fake{}
				fired := make(chan struct{})
				t := f.RunAt(time.Time{}.Add(time.Second), func() {
					close(fired)
				})
				go f.Advance(2 * time.Second)
				Eventually(fired).Should(BeClosed())
				Expect(t.Stop()).To(BeFalse())
			})
			It("Should return false from Stop after a prior Stop", func() {
				f := &xtime.Fake{}
				t := f.RunAt(time.Time{}.Add(time.Second), func() {})
				Expect(t.Stop()).To(BeTrue())
				Expect(t.Stop()).To(BeFalse())
			})
		})
	})
})
