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
		It("Should deliver a value on After after the duration elapses", func() {
			Eventually(xtime.Real.After(5*time.Millisecond), time.Millisecond*100).
				Should(Receive())
		})
		It("Should call AfterFunc functions after the duration elapses", func() {
			done := make(chan struct{})
			xtime.Real.AfterFunc(5*time.Millisecond, func() { close(done) })
			Eventually(done, time.Millisecond*100).Should(BeClosed())
		})
		It("Should let Stop cancel a pending AfterFunc timer", func() {
			var called atomic.Bool
			t := xtime.Real.AfterFunc(time.Hour, func() { called.Store(true) })
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

		Describe("After", func() {
			It("Should fire timers whose deadline has been crossed", func() {
				f := &xtime.Fake{}
				ch := f.After(time.Second)
				go f.Advance(2 * time.Second)
				var got time.Time
				Eventually(ch).Should(Receive(&got))
				Expect(got).To(Equal(time.Time{}.Add(2 * time.Second)))
			})
			It("Should close timer channels after firing", func() {
				f := &xtime.Fake{}
				ch := f.After(time.Second)
				go f.Advance(2 * time.Second)
				Eventually(ch).Should(Receive())
				Eventually(ch).Should(BeClosed())
			})
			It("Should not fire timers whose deadline has not been crossed", func() {
				f := &xtime.Fake{}
				ch := f.After(time.Second)
				f.Advance(500 * time.Millisecond)
				Consistently(ch, time.Millisecond*50).ShouldNot(Receive())
			})
			It("Should fire only timers whose deadline has been crossed", func() {
				f := &xtime.Fake{}
				early := f.After(time.Second)
				late := f.After(10 * time.Second)
				go f.Advance(2 * time.Second)
				Eventually(early).Should(Receive())
				Consistently(late, time.Millisecond*50).ShouldNot(Receive())
			})
			It("Should fire previously-registered timers on a later Advance", func() {
				f := &xtime.Fake{}
				ch := f.After(time.Second)
				f.Advance(500 * time.Millisecond)
				go f.Advance(700 * time.Millisecond)
				Eventually(ch).Should(Receive())
			})
		})

		Describe("AfterFunc", func() {
			It(
				"Should call scheduled functions when Advance crosses the deadline",
				func() {
					f := &xtime.Fake{}
					done := make(chan struct{})
					f.AfterFunc(time.Second, func() { close(done) })
					go f.Advance(2 * time.Second)
					Eventually(done).Should(BeClosed())
				},
			)
			It("Should not call functions whose deadline has not been crossed", func() {
				f := &xtime.Fake{}
				var called atomic.Int32
				t := f.AfterFunc(time.Second, func() { called.Add(1) })
				DeferCleanup(func() { t.Stop() })
				f.Advance(500 * time.Millisecond)
				Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(0)))
			})
			It("Should call only functions whose deadline has been crossed", func() {
				f := &xtime.Fake{}
				earlyFired := make(chan struct{})
				var late atomic.Int32
				early := f.AfterFunc(time.Second, func() { close(earlyFired) })
				lateT := f.AfterFunc(10*time.Second, func() { late.Add(1) })
				DeferCleanup(func() { early.Stop(); lateT.Stop() })
				go f.Advance(2 * time.Second)
				Eventually(earlyFired).Should(BeClosed())
				Consistently(late.Load, time.Millisecond*20).Should(Equal(int32(0)))
			})
			It("Should let Stop cancel a pending timer", func() {
				f := &xtime.Fake{}
				var called atomic.Int32
				t := f.AfterFunc(time.Second, func() { called.Add(1) })
				Expect(t.Stop()).To(BeTrue())
				go f.Advance(2 * time.Second)
				Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(0)))
			})
			It("Should return false from Stop once the timer has fired", func() {
				f := &xtime.Fake{}
				fired := make(chan struct{})
				t := f.AfterFunc(time.Second, func() { close(fired) })
				go f.Advance(2 * time.Second)
				Eventually(fired).Should(BeClosed())
				Expect(t.Stop()).To(BeFalse())
			})
			It("Should return false from Stop after a prior Stop", func() {
				f := &xtime.Fake{}
				t := f.AfterFunc(time.Second, func() {})
				Expect(t.Stop()).To(BeTrue())
				Expect(t.Stop()).To(BeFalse())
			})
		})
	})
})
