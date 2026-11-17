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
	"context"
	"sync"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak/goroutine"
	"github.com/onsi/gomega/types"
)

// StartRun calls run in a goroutine and returns once that goroutine first blocks or
// returns, so the goroutines that run starts before it blocks are in the baseline of a
// later leak check. The returned stop ends the context of run, waits for run to return,
// and asserts that its error satisfies every matcher, or is nil when there are no
// matchers. A DeferCleanup calls stop, and stop does nothing after its first call. The
// context derives from GinkgoT().Context(), so it outlives the setup node that calls
// StartRun.
func StartRun(
	run func(ctx context.Context) error,
	matchers ...types.GomegaMatcher,
) (stop func()) {
	ginkgo.GinkgoHelper()
	ctx, cancel := context.WithCancel(ginkgo.GinkgoT().Context())
	errs := make(chan error, 1)
	id := make(chan uint64, 1)
	go func() {
		id <- goroutine.Current().ID
		errs <- run(ctx)
	}()
	matcher := gomega.Succeed()
	if len(matchers) > 0 {
		matcher = gomega.SatisfyAll(matchers...)
	}
	stop = sync.OnceFunc(func() {
		cancel()
		var err error
		gomega.Eventually(errs).Should(gomega.Receive(&err))
		gomega.Expect(err).To(matcher)
	})
	ginkgo.DeferCleanup(stop)
	gomega.Eventually(running).WithArguments(<-id).Should(gomega.BeFalse())
	return stop
}

// running reports whether the goroutine with the given ID exists and is not blocked.
func running(id uint64) bool {
	for _, g := range goroutine.Goroutines() {
		if g.ID == id {
			return g.State == "running" || g.State == "runnable"
		}
	}
	return false
}
