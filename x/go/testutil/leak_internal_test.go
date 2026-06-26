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
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
)

// These specs drive assertNoLeakedGoroutines directly and capture its failure with
// InterceptGomegaFailures so that a deliberate leak verifies the check without failing
// the enclosing suite.
var _ = ginkgo.Describe("Leak (failure paths)", func() {
	ginkgo.Describe("assertNoLeakedGoroutines", func() {
		ginkgo.It("fails when a goroutine leaks past the snapshot", func() {
			snapshot := gleak.Goroutines()
			block := make(chan struct{})
			done := make(chan struct{})
			go func() {
				defer close(done)
				<-block
			}()
			ginkgo.DeferCleanup(func() {
				close(block)
				gomega.Eventually(done).Should(gomega.BeClosed())
			})

			failures := gomega.InterceptGomegaFailures(func() {
				assertNoLeakedGoroutines(snapshot)
			})
			gomega.Expect(failures).ToNot(gomega.BeEmpty())
		})

		ginkgo.It("does not fail when goroutines return to the baseline", func() {
			snapshot := gleak.Goroutines()
			released := make(chan struct{})
			done := make(chan struct{})
			go func() {
				defer close(done)
				<-released
			}()
			close(released)
			gomega.Eventually(done).Should(gomega.BeClosed())

			failures := gomega.InterceptGomegaFailures(func() {
				assertNoLeakedGoroutines(snapshot)
			})
			gomega.Expect(failures).To(gomega.BeEmpty())
		})
	})
})
