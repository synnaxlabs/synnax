// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package nolint_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/x/analyzers/nolint"
	"golang.org/x/tools/go/analysis/analysistest"
)

var _ = Describe("Analyzer", func() {
	DescribeTable("Should report each //nolint directive",
		func(pkg string) {
			analysistest.Run(GinkgoT(), analysistest.TestData(), nolint.Analyzer, pkg)
		},
		Entry("comments that are not directives", "clean"),
		Entry("bare, specific, and spaced directives", "directives"),
	)
})
