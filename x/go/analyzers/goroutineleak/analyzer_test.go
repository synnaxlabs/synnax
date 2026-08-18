// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package goroutineleak_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/x/analyzers/goroutineleak"
	"golang.org/x/tools/go/analysis/analysistest"
)

var _ = Describe("Analyzer", func() {
	It("Should accept a well-formed suite", func() {
		analysistest.Run(
			GinkgoT(),
			analysistest.TestData(),
			goroutineleak.Analyzer,
			"good",
		)
	})

	It("Should flag a suite that does not register the per-spec leak check", func() {
		analysistest.Run(
			GinkgoT(),
			analysistest.TestData(),
			goroutineleak.Analyzer,
			"missingperspec",
		)
	})

	It("Should flag BeforeSuite/BeforeAll nodes missing the leak check", func() {
		analysistest.Run(
			GinkgoT(),
			analysistest.TestData(),
			goroutineleak.Analyzer,
			"badsetup",
		)
	})
})
