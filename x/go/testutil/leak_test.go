// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
)

// holdUntilSignal is the named function we leak in test cases so it can be
// matched by IgnoringTopFunction. Defined at package scope to give it a stable
// fully qualified name.
func holdUntilSignal(done <-chan struct{}) { <-done }

var _ = Describe("Leak", func() {
	Describe("buildLeakConfig", func() {
		It("returns a zero config when no options are supplied", func() {
			cfg := buildLeakConfig(nil)
			Expect(cfg.timeout).To(BeZero())
			Expect(cfg.polling).To(BeZero())
			Expect(cfg.filters).To(BeEmpty())
		})

		It("applies LeakWithin to the timeout field", func() {
			cfg := buildLeakConfig([]LeakOption{LeakWithin(2 * time.Second)})
			Expect(cfg.timeout).To(Equal(2 * time.Second))
		})

		It("applies LeakPolling to the polling field", func() {
			cfg := buildLeakConfig([]LeakOption{LeakPolling(50 * time.Millisecond)})
			Expect(cfg.polling).To(Equal(50 * time.Millisecond))
		})

		It("accumulates filters from multiple LeakIgnoring calls", func() {
			cfg := buildLeakConfig([]LeakOption{
				LeakIgnoring(gleak.IgnoringTopFunction("foo.bar")),
				LeakIgnoring(gleak.IgnoringTopFunction("baz.qux")),
			})
			Expect(cfg.filters).To(HaveLen(2))
		})

		It("composes options in order", func() {
			cfg := buildLeakConfig([]LeakOption{
				LeakWithin(1 * time.Second),
				LeakPolling(10 * time.Millisecond),
				LeakIgnoring(gleak.IgnoringTopFunction("foo")),
			})
			Expect(cfg.timeout).To(Equal(1 * time.Second))
			Expect(cfg.polling).To(Equal(10 * time.Millisecond))
			Expect(cfg.filters).To(HaveLen(1))
		})
	})

	Describe("assertNoLeakedGoroutines", func() {
		It("passes when no goroutines have leaked", func() {
			snapshot := gleak.Goroutines()
			assertNoLeakedGoroutines(snapshot, leakConfig{})
		})

		It("passes when a forked goroutine has joined before the check runs", func() {
			snapshot := gleak.Goroutines()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() { defer wg.Done() }()
			wg.Wait()
			assertNoLeakedGoroutines(snapshot, leakConfig{})
		})

		It("fails when a forked goroutine has not exited", func() {
			snapshot := gleak.Goroutines()
			done := make(chan struct{})
			defer close(done)
			go holdUntilSignal(done)
			Eventually(gleak.Goroutines).Should(HaveLen(len(snapshot) + 1))

			failures := InterceptGomegaFailures(func() {
				assertNoLeakedGoroutines(snapshot, leakConfig{
					timeout: 100 * time.Millisecond,
					polling: 10 * time.Millisecond,
				})
			})
			Expect(failures).ToNot(BeEmpty())
		})

		It("ignores leaked goroutines that match a LeakIgnoring filter", func() {
			snapshot := gleak.Goroutines()
			done := make(chan struct{})
			defer close(done)
			go holdUntilSignal(done)
			Eventually(gleak.Goroutines).Should(HaveLen(len(snapshot) + 1))

			cfg := buildLeakConfig([]LeakOption{
				LeakWithin(100 * time.Millisecond),
				LeakIgnoring(gleak.IgnoringTopFunction(
					"github.com/synnaxlabs/x/testutil.holdUntilSignal",
				)),
			})
			assertNoLeakedGoroutines(snapshot, cfg)
		})

		It("respects LeakWithin when polling for drain", func() {
			snapshot := gleak.Goroutines()
			done := make(chan struct{})
			go func() {
				time.Sleep(50 * time.Millisecond)
				close(done)
			}()
			go func() { <-done }()
			Eventually(gleak.Goroutines).Should(HaveLen(len(snapshot) + 2))

			assertNoLeakedGoroutines(snapshot, leakConfig{
				timeout: 1 * time.Second,
				polling: 10 * time.Millisecond,
			})
		})
	})

	Describe("ShouldNotLeakGoroutines", func() {
		It("does not fail a clean spec", func() {
			ShouldNotLeakGoroutines()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() { defer wg.Done() }()
			wg.Wait()
		})

		It("does not fail when LeakIgnoring suppresses an intentional leak", func() {
			done := make(chan struct{})
			DeferCleanup(func() { close(done) })
			ShouldNotLeakGoroutines(LeakIgnoring(gleak.IgnoringTopFunction(
				"github.com/synnaxlabs/x/testutil.holdUntilSignal",
			)))
			go holdUntilSignal(done)
		})
	})

	Describe("ShouldNotLeakGoroutinesPerSpec", func() {
		Context("with no leaks", func() {
			ShouldNotLeakGoroutinesPerSpec()

			It("passes for a spec that does no work", func() {})

			It("passes for a spec that forks and joins a goroutine", func() {
				var wg sync.WaitGroup
				wg.Add(1)
				go func() { defer wg.Done() }()
				wg.Wait()
			})
		})
	})
})
