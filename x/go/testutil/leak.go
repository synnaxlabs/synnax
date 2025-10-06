// Copyright 2025 Synnax Labs, Inc.
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

// ShouldNotLeakGoroutines should be placed in a BeforeSuite, BeforeAll, BeforeEach,
// or JustBeforeEach block of a Ginkgo based test suite. It asserts that all goroutines
// forked from the point the block was just called to the point of block cleanup are
// correctly shut down and not leaked.
func ShouldNotLeakGoroutines() {
	grs := gleak.Goroutines()
	ginkgo.DeferCleanup(func() {
		gomega.Eventually(gleak.Goroutines).ShouldNot(gleak.HaveLeaked(grs))
	})
}

// ShouldNotLeakGoroutinesBeforeEach asserts that no goroutines are leaked during
// each spec contained within the same block that this function is called
// inside. This function differs from ShouldNotLeakRoutinesJustBeforeEach
// in that it takes into account goroutines forked in BeforeEach blocks contained
// within the same container spec.
func ShouldNotLeakGoroutinesBeforeEach() {
	ginkgo.BeforeEach(func() { ShouldNotLeakGoroutines() })
}

// ShouldNotLeakRoutinesJustBeforeEach asserts that no goroutines are leaked
// during each spec contained within the same block that this function
// is called in. It differs from ShouldNotLeakGoroutinesBeforeEach in that it runs
// after all BeforeEach blocks have run, meaning that goroutines forked in
// BeforeEach blocks are ignored.
func ShouldNotLeakRoutinesJustBeforeEach() {
	ginkgo.JustBeforeEach(func() { ShouldNotLeakGoroutines() })
}
