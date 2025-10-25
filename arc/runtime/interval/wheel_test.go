// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package interval_test

import (
	"context"
	"sync"
	gotime "time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/runtime/interval"
)

var _ = Describe("TimeWheel", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		wheel  *interval.Wheel
		fired  map[string]int
		mu     sync.Mutex
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		fired = make(map[string]int)

		onTick := func(key string) {
			mu.Lock()
			fired[key]++
			mu.Unlock()
		}

		wheel = interval.NewWheel(10*gotime.Millisecond, onTick)
	})

	AfterEach(func() {
		cancel()
		wheel.Stop()
	})

	Describe("Single Interval", func() {
		It("Should fire at regular intervals", func() {
			wheel.Register("interval_1", 50*gotime.Millisecond, 0)
			wheel.Start(ctx)

			// Wait for ~150ms (should fire 3 times)
			gotime.Sleep(155 * gotime.Millisecond)

			mu.Lock()
			count := fired["interval_1"]
			mu.Unlock()

			Expect(count).To(BeNumerically(">=", 2))
			Expect(count).To(BeNumerically("<=", 4))
		})

		It("Should respect initial delay", func() {
			wheel.Register("interval_1", 50*gotime.Millisecond, 100*gotime.Millisecond)
			wheel.Start(ctx)

			// After 75ms, should not have fired yet (initial delay is 100ms)
			gotime.Sleep(75 * gotime.Millisecond)
			mu.Lock()
			count1 := fired["interval_1"]
			mu.Unlock()
			Expect(count1).To(Equal(0))

			// After another 100ms (175ms total), should have fired once
			gotime.Sleep(100 * gotime.Millisecond)
			mu.Lock()
			count2 := fired["interval_1"]
			mu.Unlock()
			Expect(count2).To(BeNumerically(">=", 1))
		})
	})

	Describe("Multiple Intervals", func() {
		It("Should handle different periods", func() {
			wheel.Register("interval_1", 50*gotime.Millisecond, 0)
			wheel.Register("interval_2", 100*gotime.Millisecond, 0)
			wheel.Start(ctx)

			gotime.Sleep(210 * gotime.Millisecond)

			mu.Lock()
			count1 := fired["interval_1"]
			count2 := fired["interval_2"]
			mu.Unlock()

			// Interval 1 (50ms) should fire ~4 times
			Expect(count1).To(BeNumerically(">=", 3))
			Expect(count1).To(BeNumerically("<=", 5))

			// Interval 2 (100ms) should fire ~2 times
			Expect(count2).To(BeNumerically(">=", 1))
			Expect(count2).To(BeNumerically("<=", 3))
		})
	})

	Describe("State Management", func() {
		It("Should track tick count", func() {
			wheel.Register("interval_1", 50*gotime.Millisecond, 0)
			wheel.Start(ctx)

			gotime.Sleep(155 * gotime.Millisecond)

			tick, _, _, ok := wheel.GetState("interval_1")
			Expect(ok).To(BeTrue())
			Expect(tick).To(BeNumerically(">=", 2))
		})

		It("Should return elapsed time", func() {
			wheel.Start(ctx)
			wheel.Register("interval_1", 50*gotime.Millisecond, 0)

			gotime.Sleep(100 * gotime.Millisecond)

			_, _, elapsed, ok := wheel.GetState("interval_1")
			Expect(ok).To(BeTrue())
			Expect(int64(elapsed)).To(BeNumerically(">=", 100*1e6)) // nanoseconds
		})
	})

	Describe("Enable/Disable", func() {
		It("Should stop firing when disabled", func() {
			wheel.Register("interval_1", 50*gotime.Millisecond, 0)
			wheel.Start(ctx)

			gotime.Sleep(100 * gotime.Millisecond)

			mu.Lock()
			countBefore := fired["interval_1"]
			mu.Unlock()

			wheel.Disable("interval_1")
			gotime.Sleep(100 * gotime.Millisecond)

			mu.Lock()
			countAfter := fired["interval_1"]
			mu.Unlock()

			Expect(countAfter).To(Equal(countBefore))
		})

		It("Should resume firing when re-enabled", func() {
			wheel.Register("interval_1", 50*gotime.Millisecond, 0)
			wheel.Start(ctx)
			wheel.Disable("interval_1")

			gotime.Sleep(100 * gotime.Millisecond)

			wheel.Enable("interval_1")
			gotime.Sleep(100 * gotime.Millisecond)

			mu.Lock()
			count := fired["interval_1"]
			mu.Unlock()

			Expect(count).To(BeNumerically(">", 0))
		})
	})

	Describe("Unregister", func() {
		It("Should stop firing when unregistered", func() {
			wheel.Register("interval_1", 50*gotime.Millisecond, 0)
			wheel.Start(ctx)

			gotime.Sleep(100 * gotime.Millisecond)

			wheel.Unregister("interval_1")
			mu.Lock()
			countBefore := fired["interval_1"]
			mu.Unlock()

			gotime.Sleep(100 * gotime.Millisecond)

			mu.Lock()
			countAfter := fired["interval_1"]
			mu.Unlock()

			Expect(countAfter).To(Equal(countBefore))

			_, _, _, ok := wheel.GetState("interval_1")
			Expect(ok).To(BeFalse())
		})
	})
})
