// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package blank_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/analyzers/blank"
	"golang.org/x/tools/go/analysis/analysistest"
)

var _ = Describe("Analyzer", func() {
	It("Should detect and fix blank receivers", func() {
		testdata := analysistest.TestData()
		results := analysistest.RunWithSuggestedFixes(
			GinkgoT(), testdata, blank.Analyzer, "receivers",
		)
		count := 0
		for _, r := range results {
			count += len(r.Diagnostics)
		}
		Expect(count).To(Equal(3))
	})

	It("Should detect and fix all-blank parameter lists", func() {
		testdata := analysistest.TestData()
		results := analysistest.RunWithSuggestedFixes(
			GinkgoT(), testdata, blank.Analyzer, "params",
		)
		count := 0
		for _, r := range results {
			count += len(r.Diagnostics)
		}
		Expect(count).To(Equal(5))
	})

	It("Should not flag anonymous, named, or mixed declarations", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, blank.Analyzer, "clean",
		)
		for _, r := range results {
			Expect(r.Diagnostics).To(BeEmpty())
		}
	})
})
