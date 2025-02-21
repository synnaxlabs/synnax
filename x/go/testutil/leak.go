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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
)

func ShouldNotLeakGoroutines() {
	grs := gleak.Goroutines()
	DeferCleanup(func() {
		Expect(gleak.Goroutines()).ShouldNot(gleak.HaveLeaked(grs))
	})
}

func ShouldNotLeakGoroutinesDuringEach() {
	BeforeEach(func() { ShouldNotLeakGoroutines() })
}

func ShouldNotLeakRoutinesJustBeforeEach() {
	JustBeforeEach(func() { ShouldNotLeakGoroutines() })
}
