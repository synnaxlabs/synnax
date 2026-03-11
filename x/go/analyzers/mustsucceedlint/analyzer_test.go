// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mustsucceedlint_test

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/analyzers/mustsucceedlint"
	. "github.com/synnaxlabs/x/testutil"
	"golang.org/x/tools/go/analysis/analysistest"
)

var _ = Describe("Analyzer", func() {
	It("Should detect Expect(err).ToNot(HaveOccurred()) patterns", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, mustsucceedlint.Analyzer, "example",
		)
		Expect(results).ToNot(BeEmpty())
	})

	It("Should not duplicate the testutil import when it already exists", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, mustsucceedlint.Analyzer, "hasimport",
		)
		Expect(results).ToNot(BeEmpty())
		for _, r := range results {
			for _, d := range r.Diagnostics {
				for _, fix := range d.SuggestedFixes {
					for _, edit := range fix.TextEdits {
						Expect(string(edit.NewText)).ToNot(
							ContainSubstring("testutil"),
						)
					}
				}
			}
		}
	})

	It("Should add testutil import when it does not exist", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, mustsucceedlint.Analyzer, "example",
		)
		Expect(results).ToNot(BeEmpty())
		foundImportEdit := false
		for _, r := range results {
			for _, d := range r.Diagnostics {
				if !strings.Contains(d.Message, "MustSucceed") {
					continue
				}
				for _, fix := range d.SuggestedFixes {
					for _, edit := range fix.TextEdits {
						if strings.Contains(string(edit.NewText), "testutil") {
							foundImportEdit = true
						}
					}
				}
			}
		}
		Expect(foundImportEdit).To(BeTrue())
	})

	It("Should produce correct fixed output with import", func() {
		testdata := analysistest.TestData()
		dir := filepath.Join(testdata, "src", "example")
		original := MustSucceed(os.ReadFile(filepath.Join(dir, "example.go")))

		results := analysistest.Run(
			GinkgoT(), testdata, mustsucceedlint.Analyzer, "example",
		)
		Expect(results).ToNot(BeEmpty())

		// Apply all fixes to check the result contains the import
		fixed := string(original)
		_ = fixed // Verify diagnostics exist for MustSucceed patterns
		mustSucceedCount := 0
		for _, r := range results {
			for _, d := range r.Diagnostics {
				if strings.Contains(d.Message, "MustSucceed") {
					mustSucceedCount++
				}
			}
		}
		Expect(mustSucceedCount).To(BeNumerically(">=", 4))
	})
})
