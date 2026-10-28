// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errmatch_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/x/analyzers/errmatch"
	"golang.org/x/tools/go/analysis/analysistest"
)

var _ = Describe("Analyzer", func() {
	DescribeTable("Should report each assertion that does not identify the error",
		func(pkg string) {
			analysistest.Run(
				GinkgoT(),
				analysistest.TestData(),
				errmatch.Analyzer,
				pkg,
			)
		},
		Entry("matched, successful, and non-error assertions", "good"),
		Entry("HaveOccurred, Not(Succeed), and Not(BeNil) on errors", "bare"),
	)
})
