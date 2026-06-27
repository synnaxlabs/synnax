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

// ShouldNotLeakGoroutines snapshots the currently running goroutines and registers a
// Ginkgo DeferCleanup that asserts no new goroutines remain when the enclosing node
// finishes.
//
// Place this inside a lifecycle hook (BeforeSuite, BeforeAll, BeforeEach,
// JustBeforeEach). Do not call it from inside an It block unless you specifically want
// a mid-test baseline.
//
// For the common case of "check every spec in this Describe", prefer
// ShouldNotLeakGoroutinesPerSpec.
func ShouldNotLeakGoroutines() {
	snapshot := gleak.Goroutines()
	ginkgo.DeferCleanup(func() { assertNoLeakedGoroutines(snapshot) })
}

// ShouldNotLeakGoroutinesPerSpec wires per-spec leak checking into the current
// container. Every spec inside it gets a fresh snapshot taken just before the spec body
// runs and a cleanup assertion that fires after every AfterEach and DeferCleanup the
// user registered.
//
// Two hooks split the work to satisfy contradictory ordering constraints:
//
// 1. The DeferCleanup that runs the assertion is registered from a BeforeEach (early in
// spec setup) so that Ginkgo's LIFO cleanup chain runs every user-registered
// DeferCleanup BEFORE the leak check fires.
//
// 2. The snapshot itself is captured in a JustBeforeEach (the last hook before the spec
// body) so that goroutines spawned by BeforeEach AND BeforeAll setup are part of the
// steady-state baseline.
//
// The snapshot is stashed in a holder allocated fresh for every spec, so the assertion
// DeferCleanup closes over a stable per-spec value even though snapshot capture happens
// later in the lifecycle.
//
// Returns a bool so it can be invoked at file scope for suite-wide coverage:
//
//	var _ = ShouldNotLeakGoroutinesPerSpec()
func ShouldNotLeakGoroutinesPerSpec() bool {
	var current *snapshotHolder
	ginkgo.BeforeEach(func() {
		h := &snapshotHolder{}
		current = h
		ginkgo.DeferCleanup(func() {
			// If a BeforeEach panicked or skipped before JustBeforeEach captured the
			// snapshot, h.snapshot is nil. Running the assertion in that state would
			// treat every currently running goroutine as a leak, stacking a noisy
			// second failure on top of the real one. Bail out instead.
			if h.snapshot == nil {
				return
			}
			assertNoLeakedGoroutines(h.snapshot)
		})
	})
	return ginkgo.JustBeforeEach(func() { current.snapshot = gleak.Goroutines() })
}

type snapshotHolder struct{ snapshot []gleak.Goroutine }

// assertNoLeakedGoroutines runs the leak assertion synchronously against the supplied
// baseline, ignoring goroutines that are known to be unstoppable process-globals.
func assertNoLeakedGoroutines(snapshot []gleak.Goroutine) {
	args := make([]any, 0, 2)
	args = append(args, snapshot)
	// fasthttp lazily starts a single process-global goroutine (guarded by
	// serverDateOnce) the first time any server writes a response; it refreshes the
	// cached HTTP Date header every second and loops forever with no way to stop it.
	// Upstream closed the request to add a shutdown mechanism as not-planned:
	// https://github.com/valyala/fasthttp/issues/2257
	args = append(
		args,
		gleak.IgnoringCreator("github.com/valyala/fasthttp.updateServerDate"),
	)
	// Every fasthttp server starts a worker-pool janitor that sleeps for
	// MaxIdleWorkerDuration (10s) between cleanup passes. On server shutdown it is
	// signaled to stop, but the signal is only observed after the in-progress sleep
	// returns, so it can outlive its server by up to 10s. That is far longer than the
	// leak assertion's polling window, so ignore it; a genuinely leaked server is still
	// caught by its other (acceptor/connection) goroutines.
	args = append(
		args,
		gleak.IgnoringCreator("github.com/valyala/fasthttp.(*workerPool).Start"),
	)
	// gofiber's monitor middleware (mounted on /metrics in debug mode) starts a single
	// process-global goroutine, guarded by a package-level sync.Once, that refreshes
	// cached process/OS statistics every cfg.Refresh and loops forever. It has no
	// shutdown mechanism, so it outlives the server that mounted it; ignore it the same
	// way as the fasthttp process-globals above.
	args = append(
		args,
		gleak.IgnoringCreator("github.com/gofiber/contrib/v3/monitor.New.func1"),
	)
	assertion := gomega.Eventually(gleak.Goroutines)
	assertion.ShouldNot(gleak.HaveLeaked(args...))
}
