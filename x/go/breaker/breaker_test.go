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

var _ = Describe("Breaker", func() {
	It("By default should not allow retry", func(specCtx SpecContext) {
		ctx, cancel := context.WithCancel(specCtx)
		b := MustSucceed(breaker.NewBreaker(ctx))
		Expect(b.Wait()).To(BeFalse())
		cancel()
	})

	It("Should stop waiting when the context is canceled", func(specCtx SpecContext) {
		ctx, cancel := context.WithCancel(specCtx)
		defer cancel()
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
			interval *= time.Millisecond
			Eventually(clock.Pending).Should(Equal(1))
			// One tick short of the interval leaves the timer pending, which is what
			// pins the deadline to the scaled interval rather than any earlier one.
			clock.Advance(interval - time.Millisecond)
			Expect(clock.Pending()).To(Equal(1))
			clock.Advance(time.Millisecond)
			Eventually(waited).Should(Receive(BeTrue()))
		}
		wg.Wait()
		Expect(clock.Now().Sub(start)).To(Equal(310 * time.Millisecond))
	})

	It("Should reset back to the base interval", func(ctx SpecContext) {
		clock := &xtime.Fake{}
		b := MustSucceed(breaker.NewBreaker(ctx, breaker.Config{
			BaseInterval: 10 * time.Millisecond,
			Scale:        2,
			MaxRetries:   10,
			Clock:        clock,
		}))
		waited := make(chan bool, 1)
		advanceOnce := func(interval time.Duration) {
			GinkgoHelper()
			var wg sync.WaitGroup
			wg.Go(func() { waited <- b.Wait() })
			Eventually(clock.Pending).Should(Equal(1))
			clock.Advance(interval - time.Millisecond)
			Expect(clock.Pending()).To(Equal(1))
			clock.Advance(time.Millisecond)
			Eventually(waited).Should(Receive(BeTrue()))
			wg.Wait()
		}
		advanceOnce(10 * time.Millisecond)
		advanceOnce(20 * time.Millisecond)
		b.Reset()
		advanceOnce(10 * time.Millisecond)
	})
})
