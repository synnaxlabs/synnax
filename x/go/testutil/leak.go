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
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
)

// LeakOption tunes a ShouldNotLeakGoroutines call.
type LeakOption func(*leakConfig)

type leakConfig struct {
	timeout time.Duration
	polling time.Duration
	filters []any
}

// LeakWithin sets the maximum time to wait for goroutines to drain before
// failing the leak assertion. The default matches Gomega's Eventually default
// (1 second). Bump this for suites with intentionally slow shutdown paths.
func LeakWithin(d time.Duration) LeakOption {
	return func(c *leakConfig) { c.timeout = d }
}

// LeakPolling sets the interval at which the leak assertion re-checks for
// stragglers. The default matches Gomega's Eventually default (10ms).
func LeakPolling(d time.Duration) LeakOption {
	return func(c *leakConfig) { c.polling = d }
}

// LeakIgnoring adds gleak filter matchers (e.g. gleak.IgnoringTopFunction,
// gleak.IgnoringCreator, gleak.IgnoringInBacktrace) for goroutines that are
// expected to outlive the spec. Use sparingly. A leak that needs filtering
// is almost always a bug in the production code or test cleanup, not a
// reason to suppress the check.
func LeakIgnoring(matchers ...any) LeakOption {
	return func(c *leakConfig) { c.filters = append(c.filters, matchers...) }
}

// ShouldNotLeakGoroutines snapshots the currently running goroutines and
// registers a Ginkgo DeferCleanup that asserts no new goroutines remain
// when the enclosing node finishes.
//
// Place this inside a lifecycle hook (BeforeSuite, BeforeAll, BeforeEach,
// JustBeforeEach). Do not call it from inside an It block unless you
// specifically want a mid-test baseline.
//
// For the common case of "check every spec in this Describe", prefer
// ShouldNotLeakGoroutinesPerSpec.
func ShouldNotLeakGoroutines(opts ...LeakOption) {
	cfg := buildLeakConfig(opts)
	snapshot := gleak.Goroutines()
	ginkgo.DeferCleanup(func() { assertNoLeakedGoroutines(snapshot, cfg) })
}

func buildLeakConfig(opts []LeakOption) leakConfig {
	cfg := leakConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// assertNoLeakedGoroutines runs the leak assertion synchronously against the
// supplied baseline. Exposed at package scope so tests can drive the check
// without going through Ginkgo's DeferCleanup machinery.
func assertNoLeakedGoroutines(snapshot []gleak.Goroutine, cfg leakConfig) {
	args := make([]any, 0, len(cfg.filters)+1)
	args = append(args, snapshot)
	args = append(args, cfg.filters...)
	assertion := gomega.Eventually(gleak.Goroutines)
	if cfg.timeout > 0 {
		assertion = assertion.WithTimeout(cfg.timeout)
	}
	if cfg.polling > 0 {
		assertion = assertion.WithPolling(cfg.polling)
	}
	assertion.ShouldNot(gleak.HaveLeaked(args...))
}

// ShouldNotLeakGoroutinesPerSpec wires per-spec leak checking into the
// current container. Every spec inside it gets a fresh snapshot taken just
// before the spec body runs and a cleanup assertion that fires after every
// AfterEach and DeferCleanup the user registered.
//
// Two hooks split the work to satisfy contradictory ordering constraints:
//
//   - The DeferCleanup that runs the assertion is registered from a
//     BeforeEach (early in spec setup) so that Ginkgo's LIFO cleanup chain
//     runs every user-registered DeferCleanup BEFORE the leak check fires.
//   - The snapshot itself is captured in a JustBeforeEach (the last hook
//     before the spec body) so that goroutines spawned by BeforeEach AND
//     BeforeAll setup are part of the steady-state baseline.
//
// The snapshot is stashed in a holder allocated fresh for every spec, so the
// assertion DeferCleanup closes over a stable per-spec value even though
// snapshot capture happens later in the lifecycle.
//
// Returns a bool so it can be invoked at file scope for suite-wide coverage:
//
//	var _ = ShouldNotLeakGoroutinesPerSpec()
func ShouldNotLeakGoroutinesPerSpec(opts ...LeakOption) bool {
	type snapshotHolder struct{ snapshot []gleak.Goroutine }
	var current *snapshotHolder
	cfg := buildLeakConfig(opts)
	ginkgo.BeforeEach(func() {
		h := &snapshotHolder{}
		current = h
		ginkgo.DeferCleanup(func() { assertNoLeakedGoroutines(h.snapshot, cfg) })
	})
	return ginkgo.JustBeforeEach(func() {
		current.snapshot = gleak.Goroutines()
	})
}
