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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak/goroutine"
	"github.com/onsi/gomega/types"
)

// GoRun calls run in a goroutine with a child of ctx. It returns once that goroutine
// and the goroutines it starts, directly or through others, are blocked or gone and
// none appears between two polls, or after blockTimeout. The goroutines of run are then
// in the baseline of a later leak check. The returned stop ends the child, waits for
// run to return, and returns its error. Later calls to stop return the same error.
// GoRun works outside Ginkgo, such as in benchmarks.
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

// waitUntilSettled waits until the goroutine with ID root and every goroutine it
// started, directly or through others, are blocked or gone, and that set of goroutines
// is the same in two polls in a row.
func waitUntilSettled(root uint64) {
	deadline := time.Now().Add(blockTimeout)
	var prev []uint64
	for time.Now().Before(deadline) {
		ids, busy := descendants(root)
		if !busy && slices.Equal(ids, prev) {
			return
		}
		prev = ids
		time.Sleep(blockPollInterval)
	}
}

// descendants returns the sorted IDs of root and of the goroutines it started,
// directly or through others, and whether one of them is running or runnable.
// Goroutine IDs do not follow creation order across processors, so it follows the
// creator of each goroutine instead.
func descendants(root uint64) (ids []uint64, busy bool) {
	all := goroutine.Goroutines()
	children := make(map[uint64][]goroutine.Goroutine, len(all))
	queue := make([]goroutine.Goroutine, 0, 1)
	for _, g := range all {
		if g.ID == root {
			queue = append(queue, g)
		}
		if creator, ok := creatorID(g.Backtrace); ok {
			children[creator] = append(children[creator], g)
		}
	}
	for len(queue) > 0 {
		g := queue[0]
		queue = queue[1:]
		ids = append(ids, g.ID)
		busy = busy || g.State == "running" || g.State == "runnable"
		queue = append(queue, children[g.ID]...)
	}
	slices.Sort(ids)
	return ids, busy
}

// creatorID returns the ID of the goroutine that started the goroutine with the given
// backtrace, from its "created by ... in goroutine N" line.
func creatorID(backtrace string) (uint64, bool) {
	const createdBy, in = "created by ", " in goroutine "
	start := strings.LastIndex(backtrace, createdBy)
	if start < 0 {
		return 0, false
	}
	line, _, _ := strings.Cut(backtrace[start:], "\n")
	_, id, found := strings.Cut(line, in)
	if !found {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	return n, err == nil
}
