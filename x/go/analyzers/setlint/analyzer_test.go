// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package setlint_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/analyzers/setlint"
	"golang.org/x/tools/go/analysis/analysistest"
)

var _ = Describe("Analyzer", func() {
	It("Should detect map[T]struct{} patterns", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, setlint.Analyzer, "mapstruct",
		)
		Expect(results).ToNot(BeEmpty())
		count := 0
		for _, r := range results {
			count += len(r.Diagnostics)
		}
		Expect(count).To(Equal(7))
	})

	It("Should detect map[T]bool patterns", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, setlint.Analyzer, "mapbool",
		)
		Expect(results).ToNot(BeEmpty())
		var count int
		for _, r := range results {
			count += len(r.Diagnostics)
		}
		Expect(count).To(Equal(7))
	})

	It("Should not flag clean maps with non-set value types", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, setlint.Analyzer, "clean",
		)
		for _, r := range results {
			Expect(r.Diagnostics).To(BeEmpty())
		}
	})
})
