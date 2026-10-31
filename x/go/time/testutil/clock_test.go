// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

var _ = Describe("NewClock", func() {
	It("Should start at the given time", func() {
		start := time.Unix(1_700_000_000, 0)
		clock := timetestutil.NewClock(start)
		Expect(clock.Now()).To(Equal(start))
		clock.Advance(time.Minute)
		Expect(clock.Now()).To(Equal(start.Add(time.Minute)))
	})
})

var _ = Describe("Clock", func() {
	It("Should start at the zero time", func() {
		Expect((&timetestutil.Clock{}).Now()).To(Equal(time.Time{}))
	})
	It("Should advance by the supplied duration", func() {
		f := &timetestutil.Clock{}
		f.Advance(5 * time.Second)
		f.Advance(2 * time.Second)
		Expect(f.Now()).To(Equal(time.Time{}.Add(7 * time.Second)))
	})

	Describe("Pending", func() {
		It("Should count only timers that have not fired", func() {
			clock := &timetestutil.Clock{}
			Expect(clock.Pending()).To(Equal(0))
			clock.RunAt(clock.Now().Add(10*time.Millisecond), func() {})
			clock.RunAt(clock.Now().Add(30*time.Millisecond), func() {})
			Expect(clock.Pending()).To(Equal(2))
			clock.Advance(10 * time.Millisecond)
			Expect(clock.Pending()).To(Equal(1))
			clock.Advance(20 * time.Millisecond)
			Expect(clock.Pending()).To(Equal(0))
		})
		It("Should not count a deadline that already passed", func() {
			clock := &timetestutil.Clock{}
			clock.RunAt(clock.Now().Add(-time.Second), func() {})
			Expect(clock.Pending()).To(Equal(0))
		})
		It("Should stop counting a timer that was stopped", func() {
			clock := &timetestutil.Clock{}
			t := clock.RunAt(clock.Now().Add(time.Second), func() {})
			Expect(clock.Pending()).To(Equal(1))
			Expect(t.Stop()).To(BeTrue())
			clock.Advance(time.Second)
			Expect(clock.Pending()).To(Equal(0))
		})
	})

	Describe("RunAt", func() {
		It("Should call scheduled functions when Advance crosses the deadline",
			func() {
				f := &timetestutil.Clock{}
				done := make(chan struct{})
				f.RunAt(time.Time{}.Add(time.Second), func() { close(done) })
				go f.Advance(2 * time.Second)
				Eventually(done).Should(BeClosed())
			},
		)
		It("Should call scheduled functions whose deadline has already passed",
			func() {
				f := &timetestutil.Clock{}
				done := make(chan struct{})
				f.Advance(2 * time.Second)
				f.RunAt(time.Time{}.Add(time.Second), func() { close(done) })
				Eventually(done).Should(BeClosed())
			},
		)
		It("Should not call functions whose deadline has not been crossed", func() {
			f := &timetestutil.Clock{}
			var called atomic.Int32
			t := f.RunAt(time.Time{}.Add(time.Second), func() {
				called.Add(1)
			})
			DeferCleanup(func() { t.Stop() })
			f.Advance(500 * time.Millisecond)
			Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(0)))
		})
		It("Should call previously-scheduled functions on a later Advance", func() {
			f := &timetestutil.Clock{}
			done := make(chan struct{})
			f.RunAt(time.Time{}.Add(time.Second), func() { close(done) })
			f.Advance(500 * time.Millisecond)
			go f.Advance(700 * time.Millisecond)
			Eventually(done).Should(BeClosed())
		})
		It("Should let Stop cancel a pending timer", func() {
			f := &timetestutil.Clock{}
			var called atomic.Int32
			t := f.RunAt(time.Time{}.Add(time.Second), func() {
				called.Add(1)
			})
			Expect(t.Stop()).To(BeTrue())
			go f.Advance(2 * time.Second)
			Consistently(called.Load, time.Millisecond*20).Should(Equal(int32(0)))
		})
		It("Should call only functions whose deadline has been crossed", func() {
			f := &timetestutil.Clock{}
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
			f := &timetestutil.Clock{}
			fired := make(chan struct{})
			t := f.RunAt(time.Time{}.Add(time.Second), func() {
				close(fired)
			})
			go f.Advance(2 * time.Second)
			Eventually(fired).Should(BeClosed())
			Expect(t.Stop()).To(BeFalse())
		})
		It("Should return false from Stop after a prior Stop", func() {
			f := &timetestutil.Clock{}
			t := f.RunAt(time.Time{}.Add(time.Second), func() {})
			Expect(t.Stop()).To(BeTrue())
			Expect(t.Stop()).To(BeFalse())
		})
	})
})
