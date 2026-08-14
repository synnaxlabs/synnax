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
	. "github.com/synnaxlabs/x/testutil"
)

// interceptCleanupFailures swaps Gomega's fail handler for one that records messages
// instead of aborting, so that a failure raised by a leak check's DeferCleanup
// assertion can be observed rather than failing the spec. It returns a pointer to the
// recorded messages and registers a DeferCleanup that restores ginkgo.Fail.
//
// Call it before the leak check under test registers its assertion: the restore
// DeferCleanup registered here must run AFTER that assertion (Ginkgo's LIFO cleanup
// order gives us this), so the assertion records into failures while the recording
// handler is still installed.
func interceptCleanupFailures() *[]string {
	failures := new([]string)
	DeferCleanup(func() { RegisterFailHandler(Fail) })
	RegisterFailHandler(func(message string, _ ...int) {
		*failures = append(*failures, message)
	})
	return failures
}

var _ = Describe("Leak", func() {
	Describe("ShouldNotLeakGoroutines", func() {
		It("passes for a spec that forks and joins a goroutine", func() {
			ShouldNotLeakGoroutines()
			var wg sync.WaitGroup
			wg.Go(func() {})
			wg.Wait()
		})

		It("catches a goroutine left running by the spec", func() {
			block := make(chan struct{})
			done := make(chan struct{})
			var failures *[]string

			DeferCleanup(func() {
				close(block)
				Eventually(done).Should(BeClosed())
				Expect(*failures).ToNot(BeEmpty())
			})

			failures = interceptCleanupFailures()
			ShouldNotLeakGoroutines()

			go func() { defer close(done); <-block }()
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

		Context("with a leak", func() {
			var (
				failures *[]string
				block    chan struct{}
				done     chan struct{}
			)

			BeforeEach(func() {
				block = make(chan struct{})
				done = make(chan struct{})
				DeferCleanup(func() {
					close(block)
					Eventually(done).Should(BeClosed())
					Expect(*failures).ToNot(BeEmpty())
				})
			})
			BeforeEach(func() { failures = interceptCleanupFailures() })
			ShouldNotLeakGoroutinesPerSpec()

			It("catches a goroutine left running by the spec", func() {
				go func() { defer close(done); <-block }()
			})
		})
	})
})
