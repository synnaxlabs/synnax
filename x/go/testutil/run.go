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
	"slices"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak/goroutine"
	"github.com/onsi/gomega/types"
)

// GoRun calls run in a goroutine with a child of ctx. It returns once that goroutine
// and
// every goroutine started after it are blocked or gone and no new one appears between
// two polls, or after blockTimeout. The goroutines that run starts, nested ones
// included, are then in the baseline of a later leak check. The returned stop ends
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
	waitUntilSettled(<-id)
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
		gomega.Eventually(errs).WithTimeout(stopTimeout).Should(gomega.Receive(&err))
		gomega.Expect(err).To(matcher)
	})
	ginkgo.DeferCleanup(stop)
	return stop
}

const (
	stopTimeout       = 10 * time.Second
	blockTimeout      = 10 * time.Second
	blockPollInterval = time.Millisecond
)

// waitUntilSettled waits until no goroutine with an ID of at least first runs and the
// set of those goroutines is the same in two polls in a row. Goroutine IDs only grow,
// so the set holds every goroutine that the goroutine with ID first started.
func waitUntilSettled(first uint64) {
	deadline := time.Now().Add(blockTimeout)
	var prev []uint64
	for time.Now().Before(deadline) {
		ids, busy := since(first)
		if !busy && slices.Equal(ids, prev) {
			return
		}
		prev = ids
		time.Sleep(blockPollInterval)
	}
}

// since returns the sorted IDs of the goroutines with an ID of at least first, and
// whether one of them is running or runnable.
func since(first uint64) (ids []uint64, busy bool) {
	for _, g := range goroutine.Goroutines() {
		if g.ID < first {
			continue
		}
		ids = append(ids, g.ID)
		busy = busy || g.State == "running" || g.State == "runnable"
	}
	slices.Sort(ids)
	return ids, busy
}
