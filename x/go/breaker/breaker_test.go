// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package breaker_test

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/breaker"
	. "github.com/synnaxlabs/x/testutil"
	xtime "github.com/synnaxlabs/x/time"
)

var _ = Describe("Config", func() {
	Describe("Validate", func() {
		valid := breaker.Config{
			BaseInterval: time.Second,
			Scale:        1,
			MaxRetries:   3,
			Clock:        xtime.Real,
		}

		It("Should accept a fully specified config", func() {
			Expect(valid.Validate()).To(Succeed())
		})

		DescribeTable("Should reject a field outside its bounds",
			func(mutate func(breaker.Config) breaker.Config, msg string) {
				Expect(mutate(valid).Validate()).To(MatchError(ContainSubstring(msg)))
			},
			Entry("negative base interval", func(c breaker.Config) breaker.Config {
				c.BaseInterval = -time.Second
				return c
			}, "base_interval: must be greater than or equal to"),
			Entry("max retries below infinite", func(c breaker.Config) breaker.Config {
				c.MaxRetries = breaker.InfiniteRetries - 1
				return c
			}, "max_retries: must be greater than or equal to"),
			Entry("scale below one", func(c breaker.Config) breaker.Config {
				c.Scale = 0.5
				return c
			}, "scale: must be greater than or equal to"),
			Entry("nil clock", func(c breaker.Config) breaker.Config {
				c.Clock = nil
				return c
			}, "clock: must be non-nil"),
		)
	})

	Describe("Override", func() {
		It("Should take a field from the override when the base is zero", func() {
			clock := &xtime.Fake{}
			merged := breaker.Config{}.Override(breaker.Config{
				BaseInterval: time.Minute,
				Scale:        3,
				MaxRetries:   7,
				Clock:        clock,
			})
			Expect(merged.BaseInterval).To(Equal(time.Minute))
			Expect(merged.Scale).To(Equal(float64(3)))
			Expect(merged.MaxRetries).To(Equal(7))
			Expect(merged.Clock).To(Equal(xtime.Clock(clock)))
		})

		It("Should keep the base field when the override is zero", func() {
			base := breaker.Config{
				BaseInterval: time.Minute,
				Scale:        3,
				MaxRetries:   7,
				Clock:        xtime.Real,
			}
			Expect(base.Override(breaker.Config{})).To(Equal(base))
		})
	})
})

var _ = Describe("Breaker", func() {
	// advanceTo releases a single pending Wait by crossing interval. Stopping one tick
	// short first pins the deadline to interval rather than to any earlier one.
	advanceTo := func(clock *xtime.Fake, waited chan bool, interval time.Duration) {
		GinkgoHelper()
		Eventually(clock.Pending).Should(Equal(1))
		clock.Advance(interval - time.Millisecond)
		Expect(clock.Pending()).To(Equal(1))
		clock.Advance(time.Millisecond)
		Eventually(waited).Should(Receive(BeTrue()))
	}

	It("By default should not allow retry", func(specCtx SpecContext) {
		ctx, cancel := context.WithCancel(specCtx)
		DeferCleanup(cancel)
		b := MustSucceed(breaker.NewBreaker(ctx))
		Expect(b.Wait()).To(BeFalse())
	})

	It("Should stop waiting when the context is canceled", func(specCtx SpecContext) {
		ctx, cancel := context.WithCancel(specCtx)
		DeferCleanup(cancel)
		clock := &xtime.Fake{}
		b := MustSucceed(breaker.NewBreaker(ctx, breaker.Config{
			BaseInterval: time.Hour,
			MaxRetries:   breaker.InfiniteRetries,
			Clock:        clock,
		}))
		waited := make(chan bool, 1)
		var wg sync.WaitGroup
		wg.Go(func() { waited <- b.Wait() })
		Eventually(clock.Pending).Should(Equal(1))
		cancel()
		Eventually(waited).Should(Receive(BeFalse()))
		wg.Wait()
	})

	It("Should scale the interval every time it waits", func(ctx SpecContext) {
		clock := &xtime.Fake{}
		b := MustSucceed(breaker.NewBreaker(ctx, breaker.Config{
			BaseInterval: 10 * time.Millisecond,
			Scale:        2,
			MaxRetries:   10,
			Clock:        clock,
		}))
		start := clock.Now()
		waited := make(chan bool, 5)
		var wg sync.WaitGroup
		wg.Go(func() {
			for range 5 {
				waited <- b.Wait()
			}
		})
		for _, interval := range []time.Duration{10, 20, 40, 80, 160} {
			advanceTo(clock, waited, interval*time.Millisecond)
		}
		wg.Wait()
		Expect(clock.Now().Sub(start)).To(Equal(310 * time.Millisecond))
	})

	It("Should stop allowing retries at the maximum", func(ctx SpecContext) {
		clock := &xtime.Fake{}
		b := MustSucceed(breaker.NewBreaker(ctx, breaker.Config{
			BaseInterval: 10 * time.Millisecond,
			Scale:        1,
			MaxRetries:   2,
			Clock:        clock,
		}))
		waited := make(chan bool, 1)
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() { waited <- b.Wait() })
			advanceTo(clock, waited, 10*time.Millisecond)
			wg.Wait()
		}
		Expect(b.Wait()).To(BeFalse())
		Expect(clock.Pending()).To(Equal(0))
	})

	Describe("Reset", func() {
		It("Should return the breaker to the base interval", func(ctx SpecContext) {
			clock := &xtime.Fake{}
			b := MustSucceed(breaker.NewBreaker(ctx, breaker.Config{
				BaseInterval: 10 * time.Millisecond,
				Scale:        2,
				MaxRetries:   10,
				Clock:        clock,
			}))
			waited := make(chan bool, 1)
			wait := func(interval time.Duration) {
				GinkgoHelper()
				var wg sync.WaitGroup
				wg.Go(func() { waited <- b.Wait() })
				advanceTo(clock, waited, interval)
				wg.Wait()
			}
			wait(10 * time.Millisecond)
			wait(20 * time.Millisecond)
			b.Reset()
			wait(10 * time.Millisecond)
		})
	})
})
