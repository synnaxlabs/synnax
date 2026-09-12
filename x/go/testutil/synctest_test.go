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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Bubble", func() {
	It("Should run the given function to completion", func() {
		ran := false
		Bubble(func() { ran = true })
		Expect(ran).To(BeTrue())
	})

	It("Should measure a wait exactly and take no real time", func() {
		start := time.Now()
		Bubble(func() {
			inner := time.Now()
			time.Sleep(time.Hour)
			Expect(time.Since(inner)).To(Equal(time.Hour))
		})
		Expect(time.Since(start)).To(BeNumerically("<", time.Second))
	})

	It("Should advance the clock only once every goroutine is blocked", func() {
		Bubble(func() {
			var (
				wg      sync.WaitGroup
				elapsed time.Duration
			)
			start := time.Now()
			wg.Go(func() {
				time.Sleep(30 * time.Minute)
				elapsed = time.Since(start)
			})
			time.Sleep(time.Hour)
			wg.Wait()
			Expect(elapsed).To(Equal(30 * time.Minute))
			Expect(time.Since(start)).To(Equal(time.Hour))
		})
	})
})
