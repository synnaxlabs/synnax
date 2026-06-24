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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	. "github.com/synnaxlabs/x/testutil"
)

// holdUntilSignal is the named function we leak in test cases so it can be matched by
// IgnoringTopFunction. Defined at package scope to give it a stable fully qualified
// name.
func holdUntilSignal(done <-chan struct{}) { <-done }

const holdUntilSignalName = "github.com/synnaxlabs/x/testutil_test.holdUntilSignal"

var _ = Describe("Leak", func() {
	Describe("ShouldNotLeakGoroutines", func() {
		It("does not fail a spec that forks and joins a goroutine", func() {
			ShouldNotLeakGoroutines()
			var wg sync.WaitGroup
			wg.Go(func() {})
			wg.Wait()
		})

		// The leak check is registered after the close(done) cleanup, so it runs first
		// (LIFO) while holdUntilSignal is still parked. The spec only passes if the
		// check sees the live goroutine and the LeakIgnoring filter suppresses it,
		// exercising detection and filtering end to end.
		It("ignores a goroutine still running when the check fires", func() {
			done := make(chan struct{})
			DeferCleanup(func() { close(done) })
			ShouldNotLeakGoroutines(LeakIgnoring(
				gleak.IgnoringTopFunction(holdUntilSignalName),
			))
			go holdUntilSignal(done)
			Eventually(gleak.Goroutines).Should(ContainElement(
				HaveField("TopFunction", holdUntilSignalName),
			))
		})
	})

	Describe("ShouldNotLeakGoroutinesPerSpec", func() {
		Context("with no leaks", func() {
			ShouldNotLeakGoroutinesPerSpec()

			It("passes for a spec that does no work", func() {})

			It("passes for a spec that forks and joins a goroutine", func() {
				var wg sync.WaitGroup
				wg.Go(func() {})
				wg.Wait()
			})
		})

		// The goroutine is started in the spec and torn down in AfterAll (not a
		// per-spec cleanup), so it is still running when the per-spec check fires and
		// is only tolerated because it matches the LeakIgnoring filter.
		Context("with an ignored leak", Ordered, func() {
			ShouldNotLeakGoroutinesPerSpec(LeakIgnoring(
				gleak.IgnoringTopFunction(holdUntilSignalName),
			))
			done := make(chan struct{})
			AfterAll(func() { close(done) })

			It("does not flag a goroutine matched by the ignore filter", func() {
				go holdUntilSignal(done)
				Eventually(gleak.Goroutines).Should(ContainElement(
					HaveField("TopFunction", holdUntilSignalName),
				))
			})
		})
	})
})
