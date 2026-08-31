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
	"testing"
	"testing/synctest"

	"github.com/onsi/ginkgo/v2"
)

// suiteT holds the *testing.T that RunSpecs was given. synctest.Test takes a
// *testing.T and Ginkgo hands spec bodies no equivalent, so a suite that runs
// bubbles has to record it. RegisterSuiteT is the only writer and runs once per
// suite, before any spec.
var suiteT *testing.T

// RegisterSuiteT records t so Bubble can start a bubble from inside a spec. Call it
// from the suite's Test function alongside RegisterFailHandler.
func RegisterSuiteT(t *testing.T) { suiteT = t }

// Bubble runs f in a synctest bubble, where the time package reads a fake clock that
// jumps forward only when every goroutine in the bubble is durably blocked. A spec
// that would otherwise wait out real timers finishes instantly and measures exact
// durations instead of approximate ones.
//
// Build any context f needs from context.Background, never from the spec's: a
// goroutine blocked on a context from outside the bubble is never durably blocked, so
// the clock would stall. Deriving one from the bubble's own testing.T.Context is worse
// still, because context.WithCancel starts a propagation goroutine that outlives f and
// deadlocks the bubble.
//
// Gomega's Eventually and Consistently do not work inside a bubble: they leave a
// poller blocked when f returns. A bubble is deterministic, so assert directly.
//
// Every goroutine f starts must exit, and every resource holding a background
// goroutine must close, before f returns; Bubble blocks until the bubble drains. The
// suite must call RegisterSuiteT.
func Bubble(f func()) {
	ginkgo.GinkgoHelper()
	synctest.Test(suiteT, func(*testing.T) {
		defer ginkgo.GinkgoRecover()
		f()
	})
}
