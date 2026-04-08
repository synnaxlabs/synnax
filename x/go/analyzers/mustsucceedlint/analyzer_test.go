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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/analyzers/mustsucceedlint"
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
		results := analysistest.Run(
			GinkgoT(), testdata, mustsucceedlint.Analyzer, "example",
		)
		Expect(results).ToNot(BeEmpty())
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

	It("Should only emit a single import edit across multiple diagnostics in the same file", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, mustsucceedlint.Analyzer, "example",
		)
		Expect(results).ToNot(BeEmpty())
		importEditCount := 0
		for _, r := range results {
			for _, d := range r.Diagnostics {
				for _, fix := range d.SuggestedFixes {
					for _, edit := range fix.TextEdits {
						if strings.Contains(string(edit.NewText), "testutil") {
							importEditCount++
						}
					}
				}
			}
		}
		Expect(importEditCount).To(Equal(1))
	})

	It("Should emit the import as its own separate diagnostic, not bundled into a code fix", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, mustsucceedlint.Analyzer, "example",
		)
		Expect(results).ToNot(BeEmpty())
		// The import must be its own diagnostic so that golangci-lint can apply
		// it independently. If it's bundled as a second TextEdit inside a code
		// replacement diagnostic, golangci-lint silently drops it.
		foundImportDiagnostic := false
		for _, r := range results {
			for _, d := range r.Diagnostics {
				if strings.Contains(d.Message, "import") {
					Expect(d.SuggestedFixes).To(HaveLen(1))
					Expect(d.SuggestedFixes[0].TextEdits).To(HaveLen(1))
					Expect(string(d.SuggestedFixes[0].TextEdits[0].NewText)).To(
						ContainSubstring("testutil"),
					)
					foundImportDiagnostic = true
				}
			}
		}
		Expect(foundImportDiagnostic).To(BeTrue())
		// Also verify no code replacement diagnostic has a testutil import
		// edit bundled into it.
		for _, r := range results {
			for _, d := range r.Diagnostics {
				if !strings.Contains(d.Message, "import") {
					for _, fix := range d.SuggestedFixes {
						for _, edit := range fix.TextEdits {
							Expect(string(edit.NewText)).ToNot(
								ContainSubstring("testutil"),
							)
						}
					}
				}
			}
		}
	})

	It("Should remove LHS and assignment when all LHS vars are blank", func() {
		testdata := analysistest.TestData()
		results := analysistest.Run(
			GinkgoT(), testdata, mustsucceedlint.Analyzer, "example",
		)
		Expect(results).ToNot(BeEmpty())
		foundBlankFix := false
		for _, r := range results {
			for _, d := range r.Diagnostics {
				for _, fix := range d.SuggestedFixes {
					for _, edit := range fix.TextEdits {
						text := string(edit.NewText)
						if text == "MustSucceed(returnsValErr())" ||
							text == "MustSucceed2(returnsTwoValErr())" {
							foundBlankFix = true
						}
					}
				}
			}
		}
		Expect(foundBlankFix).To(BeTrue())
	})
})
