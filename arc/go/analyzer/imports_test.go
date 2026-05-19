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
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Import Pass", func() {
	resolver := symbol.CompoundResolver{
		&symbol.ModuleResolver{
			Name: "time",
			Members: symbol.MapResolver{
				"now": {
					Name: "time.now",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{
						Outputs: types.Params{{Name: "result", Type: types.I64()}},
					}),
				},
			},
		},
		symbol.MapResolver{
			"ch": {Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10},
		},
	}

	Describe("collectImports", func() {
		It("Should populate the root scope's ImportSet", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import time`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Scope.Imports).ToNot(BeNil())
			_, ok := ctx.Scope.Imports.Lookup("time")
			Expect(ok).To(BeTrue())
		})

		It("Should diagnose a duplicate import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import ( time time )`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("duplicate import"))
		})

		It("Should diagnose an unknown module", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import banana`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).To(ContainSubstring(`unknown module "banana"`))
		})

		It("Should not double-report an unknown module as unused", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import banana`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).ToNot(ContainSubstring(`imported module "banana" is unused`))
		})

		It("Should accept an import with no entries", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import ()`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should leave Imports nil when no source-level imports exist and no statements are present", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`func f() {}`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Scope.Imports).ToNot(BeNil())
			Expect(ctx.Scope.Imports.All()).To(BeEmpty())
		})
	})

	Describe("reportUnusedImports", func() {
		It("Should flag an unused import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				import time
				func f() {}
			`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).To(ContainSubstring(`imported module "time" is unused`))
		})

		It("Should not flag a used import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				import time
				func f() i64 { return time.now() }
			`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).ToNot(ContainSubstring("is unused"))
		})

		It("Should be a no-op when the import set is nil", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`func f() {}`))
			ctx := context.CreateRoot(bCtx, prog, resolver)
			ctx.Scope.Imports = nil
			Expect(func() { analyzer.AnalyzeProgram(ctx) }).ToNot(Panic())
		})
	})
})
