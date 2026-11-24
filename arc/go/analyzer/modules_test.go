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
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	xlsp "github.com/synnaxlabs/x/lsp"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("Modules", func() {
	channels := StaticResolver{
		{Name: "in", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10},
		{Name: "out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11},
	}
	analyze := func(
		bCtx SpecContext,
		source string,
		module bool,
	) context.Context[parser.IProgramContext] {
		GinkgoHelper()
		prog := MustSucceed(parser.Parse(source))
		ctx := context.NewRoot(bCtx, prog, NewRoot(channels)).WithModule(module)
		analyzer.AnalyzeProgram(ctx)
		return ctx
	}

	Describe("Module rules", func() {
		It("Should reject an authority block in a module", func(bCtx SpecContext) {
			ctx := analyze(bCtx, "authority 255", true)
			Expect(ctx.Diagnostics.String()).To(
				ContainSubstring("[E4006]: a module cannot have an authority block"),
			)
		})

		It("Should reject a top-level flow in a module", func(bCtx SpecContext) {
			ctx := analyze(bCtx, "in -> out", true)
			Expect(ctx.Diagnostics.String()).To(
				ContainSubstring("[E4007]: a module cannot have a top-level flow"),
			)
		})

		It(
			"Should allow an authority block and a top-level flow outside a module",
			func(bCtx SpecContext) {
				ctx := analyze(bCtx, "authority 255\nin -> out", false)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			},
		)

		It("Should reject export outside a module", func(bCtx SpecContext) {
			ctx := analyze(bCtx, "export func f() {}", false)
			Expect(ctx.Diagnostics.String()).To(
				ContainSubstring("export is only valid in a module"),
			)
		})

		DescribeTable(
			"Should give a fix that removes export outside a module",
			func(bCtx SpecContext, source, fixed string) {
				ctx := analyze(bCtx, source, false)
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				fixes := (*ctx.Diagnostics)[0].Fixes
				Expect(fixes).To(HaveLen(1))
				Expect(fixes[0].Title).To(Equal("Remove export"))
				Expect(fixes[0].Edits).To(HaveLen(1))
				Expect(xlsp.ApplyIncrementalChange(
					source,
					&protocol.TextDocumentContentChangePartial{
						Range: fixes[0].Edits[0].Range,
						Text:  fixes[0].Edits[0].NewText,
					},
				)).To(Equal(fixed))
			},
			Entry("function", "export func f() {}", "func f() {}"),
			Entry(
				"sequence",
				"export sequence s { stage a {} }",
				"sequence s { stage a {} }",
			),
			Entry("stage", "export stage s {}", "stage s {}"),
			Entry("variable", "export x := 1", "x := 1"),
			Entry(
				"declaration after another",
				"func g() {}\nexport func f() {}",
				"func g() {}\nfunc f() {}",
			),
			Entry("declaration on the next line", "export\nfunc f() {}", "func f() {}"),
			Entry(
				"comment after export",
				"export // why\nfunc f() {}",
				"// why\nfunc f() {}",
			),
		)

		It("Should give no fix for an export in a module", func(bCtx SpecContext) {
			ctx := analyze(bCtx, "export sequence { stage s {} }", true)
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Fixes).To(BeEmpty())
		})

		It("Should reject an unnamed exported declaration", func(bCtx SpecContext) {
			ctx := analyze(bCtx, "export sequence { stage s {} }", true)
			Expect(ctx.Diagnostics.String()).To(
				ContainSubstring("an exported declaration needs a name"),
			)
		})
	})

	Describe("Export marking", func() {
		DescribeTable(
			"Should mark an exported declaration",
			func(bCtx SpecContext, source, name string) {
				ctx := analyze(bCtx, source, true)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(ctx.Scope.FindChild(name).Exported).To(BeTrue())
			},
			Entry("function", "export func f() {}", "f"),
			Entry("sequence", "export sequence s { stage a {} }", "s"),
			Entry("stage", "export stage s {}", "s"),
			Entry("variable", "export x := 1", "x"),
		)

		It("Should leave an unexported declaration unmarked", func(bCtx SpecContext) {
			ctx := analyze(bCtx, "export func f() {}\nfunc g() {}", true)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(ctx.Scope.FindChild("g").Exported).To(BeFalse())
		})

		It("Should reject an exported stateful variable", func(bCtx SpecContext) {
			ctx := analyze(bCtx, "export s $= 1", true)
			Expect(ctx.Diagnostics.String()).To(ContainSubstring(
				"stateful variables cannot be declared at the top level",
			))
		})
	})
})
