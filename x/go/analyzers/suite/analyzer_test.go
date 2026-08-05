// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package suite_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/x/analyzers/suite"
	"golang.org/x/tools/go/analysis/analysistest"
)

var _ = Describe("Analyzer", func() {
	It("Should accept a correctly laid out suite", func() {
		analysistest.Run(GinkgoT(), analysistest.TestData(), suite.Analyzer, "good")
	})

	It("Should flag bootstrap calls outside the suite file", func() {
		analysistest.Run(
			GinkgoT(), analysistest.TestData(), suite.Analyzer, "misplaced",
		)
	})

	It("Should flag spec containers in a package that never calls RunSpecs", func() {
		analysistest.Run(GinkgoT(), analysistest.TestData(), suite.Analyzer, "orphan")
	})

	It("Should flag a second RunSpecs call", func() {
		analysistest.Run(GinkgoT(), analysistest.TestData(), suite.Analyzer, "multi")
	})
})
