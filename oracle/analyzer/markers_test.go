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
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Create and Output markers", func() {
	var loader *MockFileLoader
	BeforeEach(func() { loader = NewMockFileLoader() })

	It("Should accept @create on a type and @output on a field", func(ctx SpecContext) {
		source := `
			Thing struct {
				key    uuid @key
				name   string
				author uuid @output
				@create
			}
		`
		table, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
		Expect(diag.Ok()).To(BeTrue())

		typ := table.MustGet("x.Thing")
		Expect(typ.Domains).To(HaveKey("create"))

		form := typ.Form.(resolution.StructForm)
		author, ok := form.Field("author")
		Expect(ok).To(BeTrue())
		Expect(author.Domains).To(HaveKey("output"))

		name, ok := form.Field("name")
		Expect(ok).To(BeTrue())
		Expect(name.Domains).NotTo(HaveKey("output"))
	})
})
