// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Default Invariant", func() {
	var loader *MockFileLoader
	BeforeEach(func() { loader = NewMockFileLoader() })

	It("Should reject a required bool field that defaults to true", func(ctx SpecContext) {
		source := `
			Cfg struct {
				visible bool = true
			}
		`
		_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
		Expect(diag.Ok()).To(BeFalse())
		Expect(diag.Error()).To(ContainSubstring("default invariant"))
	})

	It("Should accept a required bool field that defaults to false", func(ctx SpecContext) {
		source := `
			Cfg struct {
				visible bool = false
			}
		`
		_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
		Expect(diag.Ok()).To(BeTrue())
	})

	It("Should exempt a nullable field from the invariant", func(ctx SpecContext) {
		source := `
			Cfg struct {
				visible bool? = true
			}
		`
		_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
		Expect(diag.Ok()).To(BeTrue())
	})
})
