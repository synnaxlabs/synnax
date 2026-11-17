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
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak/goroutine"
	"github.com/onsi/gomega/types"
)

// GoRun calls run in a goroutine with a child of ctx. It returns once that goroutine
// first blocks or returns, or after blockTimeout, so the goroutines that run starts
// before it blocks are in the baseline of a later leak check. The returned stop ends
// the child, waits for run to return, and returns its error. Later calls to stop
// return the same error. GoRun works outside Ginkgo, such as in benchmarks.
func GoRun(
	ctx context.Context,
	run func(ctx context.Context) error,
) (stop func() error) {
	ctx, cancel := context.WithCancel(ctx)
	errs := make(chan error, 1)
	id := make(chan uint64, 1)
	go func() {
		id <- goroutine.Current().ID
		errs <- run(ctx)
	}()
	waitUntilBlocked(<-id)
	return sync.OnceValue(func() error {
		cancel()
		return <-errs
	})
}

// StartRun calls run with GoRun under a child of GinkgoT().Context(), so the run
// outlives the setup node that calls StartRun. The returned stop ends the run, waits
// for it to return, and asserts that its error satisfies every matcher, or is nil when
// there are no matchers. A DeferCleanup calls stop, and stop does nothing after its
// first call.
func StartRun(
	run func(ctx context.Context) error,
	matchers ...types.GomegaMatcher,
) (stop func()) {
	ginkgo.GinkgoHelper()
	stopRun := GoRun(ginkgo.GinkgoT().Context(), run)
	matcher := gomega.Succeed()
	if len(matchers) > 0 {
		matcher = gomega.SatisfyAll(matchers...)
	}
	stop = sync.OnceFunc(func() {
		errs := make(chan error, 1)
		go func() { errs <- stopRun() }()
		var err error
		gomega.Eventually(errs).Should(gomega.Receive(&err))
		gomega.Expect(err).To(matcher)
	})
	ginkgo.DeferCleanup(stop)
	return stop
}

const (
	blockTimeout      = 10 * time.Second
	blockPollInterval = time.Millisecond
)

func waitUntilBlocked(id uint64) {
	deadline := time.Now().Add(blockTimeout)
	for running(id) && time.Now().Before(deadline) {
		time.Sleep(blockPollInterval)
	}
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
