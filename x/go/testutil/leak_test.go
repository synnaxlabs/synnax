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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Leak", func() {
	Describe("ShouldNotLeakGoroutines", func() {
		It("does not fail a spec that forks and joins a goroutine", func() {
			ShouldNotLeakGoroutines()
			var wg sync.WaitGroup
			wg.Go(func() {})
			wg.Wait()
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
	})
})
