// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package timer_test

import (
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/timer"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Now", func() {
	It("Should advance at the rate of the system clock", func() {
		start, wall := timer.Now(), time.Now()
		time.Sleep(50 * time.Millisecond)
		Expect(timer.Now() - start).
			To(BeNumerically("~", time.Since(wall), time.Millisecond))
	})
})

var _ = Describe("Wall", func() {
	It("Should read the wall clock", func() {
		Expect(timer.Wall()).To(BeTemporally("~", time.Now(), 20*time.Millisecond))
	})

	It("Should advance in steps finer than 0.1 ms", func() {
		start := timer.Wall()
		var step time.Duration
		for range 10_000_000 {
			if step = timer.Wall().Sub(start); step > 0 {
				break
			}
		}
		Expect(step).To(BeNumerically(">", 0))
		Expect(step).To(BeNumerically("<", 100*time.Microsecond))
	})
})

// Serial, because a spec that spins holds a CPU core, and other specs running at the
// same time make the lateness that these specs measure.
var _ = Describe("Timer", Serial, func() {
	var t *timer.Timer
	BeforeEach(func() { t = MustOpen(timer.New(timer.DefaultSpin)) })

	It("Should not fire before Reset", func() {
		Consistently(t.C).WithTimeout(20 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should fire on its deadline and not before", func() {
		Expect(t.Reset(50 * time.Millisecond)).To(Succeed())
		Consistently(t.C).
			WithTimeout(45 * time.Millisecond).
			WithPolling(time.Millisecond).
			ShouldNot(Receive())
		Eventually(t.C).WithTimeout(time.Second).Should(Receive())
	})

	DescribeTable("Should fire right away for a span of zero or less",
		func(d time.Duration) {
			Expect(t.Reset(d)).To(Succeed())
			Eventually(t.C).WithTimeout(50 * time.Millisecond).Should(Receive())
		},
		Entry("zero", time.Duration(0)),
		Entry("negative", -time.Second),
	)

	It("Should fire once per Reset", func() {
		Expect(t.Reset(time.Millisecond)).To(Succeed())
		Eventually(t.C).WithTimeout(time.Second).Should(Receive())
		Consistently(t.C).WithTimeout(20 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should fire at the span of the latest Reset", func() {
		Expect(t.Reset(time.Hour)).To(Succeed())
		Expect(t.Reset(5 * time.Millisecond)).To(Succeed())
		Eventually(t.C).WithTimeout(time.Second).Should(Receive())
		Expect(t.Reset(5 * time.Millisecond)).To(Succeed())
		Expect(t.Reset(time.Hour)).To(Succeed())
		Consistently(t.C).WithTimeout(30 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should not fire after Stop", func() {
		Expect(t.Reset(10 * time.Millisecond)).To(Succeed())
		Expect(t.Stop()).To(Succeed())
		Consistently(t.C).WithTimeout(40 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should drop a pending fire on Stop", func() {
		Expect(t.Reset(0)).To(Succeed())
		Eventually(func() int { return len(t.C) }).Should(Equal(1))
		Expect(t.Stop()).To(Succeed())
		Expect(t.C).ToNot(Receive())
	})

	It("Should not fire for a deadline that a Reset replaced", func() {
		const span = 20 * time.Microsecond
		for i := range 2000 {
			Expect(t.Reset(span)).To(Succeed())
			// The wait sweeps the wake of the first deadline, so its fire races the
			// second Reset.
			wait := span + time.Duration(i%100)*time.Microsecond
			for start := timer.Now(); timer.Now()-start < wait; {
			}
			Expect(t.Reset(time.Hour)).To(Succeed())
			time.Sleep(100 * time.Microsecond)
			Expect(t.C).ToNot(Receive())
		}
	})

	It("Should fire within half a millisecond of its deadline", func() {
		Expect(medianLateness(t, 10*time.Millisecond, 50)).
			To(BeNumerically("<", 500*time.Microsecond))
	})

	It("Should wake more often than once per millisecond", func() {
		const (
			period = 500 * time.Microsecond
			fires  = 400
		)
		start := time.Now()
		for range fires {
			Expect(t.Reset(period)).To(Succeed())
			Eventually(t.C).
				WithTimeout(time.Second).
				WithPolling(10 * time.Microsecond).
				Should(Receive())
		}
		Expect(time.Since(start)).To(BeNumerically("<", fires*875*time.Microsecond))
	})

	DescribeTable("Should never fire early, whatever its spin",
		func(spin time.Duration) {
			s := MustOpen(timer.New(spin))
			const span = 5 * time.Millisecond
			for range 20 {
				start := timer.Now()
				Expect(s.Reset(span)).To(Succeed())
				Eventually(s.C).WithTimeout(time.Second).Should(Receive())
				Expect(timer.Now() - start).To(BeNumerically(">=", span))
			}
		},
		Entry("none", time.Duration(0)),
		Entry("part of the wait", 2*time.Millisecond),
		Entry("the whole wait", timer.SpinAll),
	)

	It("Should fire within 50 µs of its deadline when it spins the whole wait", func() {
		s := MustOpen(timer.New(timer.SpinAll))
		Expect(medianLateness(s, 2*time.Millisecond, 50)).
			To(BeNumerically("<", 50*time.Microsecond))
	})

	It("Should hold a wait of DefaultSpin as a whole-wait spin does", func() {
		if timer.DefaultSpin == 0 {
			Skip("DefaultSpin does not spin")
		}
		s := MustOpen(timer.New(timer.SpinAll))
		whole := medianLateness(s, timer.DefaultSpin, 50)
		Expect(medianLateness(t, timer.DefaultSpin, 50)).
			To(BeNumerically("~", whole, 10*time.Microsecond))
	})

	It("Should not fire after Stop during a spin", func() {
		s := MustOpen(timer.New(timer.SpinAll))
		Expect(s.Reset(20 * time.Millisecond)).To(Succeed())
		time.Sleep(5 * time.Millisecond)
		Expect(s.Stop()).To(Succeed())
		Consistently(s.C).WithTimeout(40 * time.Millisecond).ShouldNot(Receive())
	})

	It("Should not hold up Reset while it spins", func() {
		s := MustOpen(timer.New(timer.SpinAll))
		const resets = 50
		took := make([]time.Duration, resets)
		for i := range took {
			Expect(s.Reset(3 * time.Millisecond)).To(Succeed())
			for start := timer.Now(); timer.Now()-start < 2*time.Millisecond; {
			}
			start := timer.Now()
			Expect(s.Reset(time.Millisecond)).To(Succeed())
			took[i] = timer.Now() - start
			Eventually(s.C).WithTimeout(time.Second).Should(Receive())
		}
		slices.Sort(took)
		// A Reset that waits on the spin takes the 1 ms left of it.
		Expect(took[resets/2]).To(BeNumerically("<", 100*time.Microsecond))
	})
})

// medianLateness returns the median of how late t fires over fires waits of span. It
// reads C directly: Eventually polls, so the wait between polls would count as
// lateness, and its polling keeps the Go runtime awake, which hides the lateness of a
// time.Timer.
func medianLateness(t *timer.Timer, span time.Duration, fires int) time.Duration {
	GinkgoHelper()
	late := make([]time.Duration, fires)
	for i := range late {
		start := timer.Now()
		Expect(t.Reset(span)).To(Succeed())
		select {
		case <-t.C:
		case <-time.After(time.Second):
			Fail("timer did not fire")
		}
		late[i] = timer.Now() - start - span
	}
	slices.Sort(late)
	return late[fires/2]
}
