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
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Import Pass", func() {
	// timeModule is a sealed "time" module attached to each test's ambient
	// prelude so `import time` resolves. dynChannels is the per-test
	// dynamic resolver, providing the bare channel symbol "ch".
	timeModule := symbol.NewModule("time", symbol.Symbol{
		Name: "now",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: "result", Type: types.I64()}},
		}),
	})
	dynChannels := StaticResolver{
		"ch": {Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10},
	}
	newRoot := func() *symbol.Symbol {
		root := func() *symbol.Symbol {
			root := symbol.CreateRoot(nil)
			root.AttachToAmbient(stl.Symbols...)
			for i := range dynChannels {
				s := dynChannels[i]
				root.Parent.AddChild(&s)
			}
			return root
		}()
		root.Parent.AddChild(timeModule)
		return root
	}

	Describe("collectImports", func() {
		It("Should install a KindModuleAlias child on the root scope", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import time`))
			ctx := context.CreateRoot(bCtx, prog, newRoot())
			analyzer.AnalyzeProgram(ctx)
			alias := ctx.Scope.FindChildByName("time")
			Expect(alias).ToNot(BeNil())
			Expect(alias.Kind).To(Equal(symbol.KindModuleAlias))
			Expect(alias.Target).ToNot(BeNil())
			Expect(alias.Target.Name).To(Equal("time"))
		})

		It("Should diagnose a duplicate import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import ( time time )`))
			ctx := context.CreateRoot(bCtx, prog, newRoot())
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("duplicate import"))
		})

		It("Should diagnose an unknown module", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import banana`))
			ctx := context.CreateRoot(bCtx, prog, newRoot())
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).To(ContainSubstring(`unknown module "banana"`))
		})

		It("Should not double-report an unknown module as unused", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import banana`))
			ctx := context.CreateRoot(bCtx, prog, newRoot())
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).ToNot(ContainSubstring(`imported module "banana" is unused`))
		})

		It("Should accept an import with no entries", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import ()`))
			ctx := context.CreateRoot(bCtx, prog, newRoot())
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should leave Children empty when there are no imports or declarations", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`func f() {}`))
			ctx := context.CreateRoot(bCtx, prog, newRoot())
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Scope.FilterChildrenByKind(symbol.KindModuleAlias)).To(BeEmpty())
		})
	})

	Describe("reportUnusedImports", func() {
		It("Should flag an unused import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				import time
				func f() {}
			`))
			ctx := context.CreateRoot(bCtx, prog, newRoot())
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).To(ContainSubstring(`imported module "time" is unused`))
		})

		It("Should not flag a used import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				import time
				func f() i64 { return time.now() }
			`))
			ctx := context.CreateRoot(bCtx, prog, newRoot())
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).ToNot(ContainSubstring("is unused"))
		})
	})
})
