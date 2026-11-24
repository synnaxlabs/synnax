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
	stdctx "context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/testutil"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

type failingModules struct{}

func (failingModules) Resolve(_ stdctx.Context, path string) (*symbol.Symbol, error) {
	return nil, errors.Newf("%q is an automation and cannot be imported", path)
}

func (failingModules) Search(stdctx.Context, string) ([]*symbol.Symbol, error) {
	return nil, nil
}

var _ = Describe("Import Pass", func() {
	// A fresh "time" module is allocated in BeforeEach so each test's
	// AddChild doesn't rewrite a shared module's Parent.
	var root *symbol.Symbol
	BeforeEach(func() {
		dynChannels := StaticResolver{
			{
				Name: "ch",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   10,
			},
		}
		timeModule := &symbol.Symbol{Name: "time", Kind: symbol.KindModule}
		timeModule.AddChild(&symbol.Symbol{
			Name: "now",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: "result", Type: types.I64()}},
			}),
		})
		root = NewRoot(dynChannels)
		root.Parent.AddChild(timeModule)
	})

	Describe("collectImports", func() {
		It(
			"Should install a KindModuleAlias child on the root scope",
			func(bCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`import time`))
				ctx := context.NewRoot(bCtx, prog, root)
				analyzer.AnalyzeProgram(ctx)
				alias := ctx.Scope.FindChild("time")
				Expect(alias).ToNot(BeNil())
				Expect(alias.Kind).To(Equal(symbol.KindModuleAlias))
				Expect(alias.Target).ToNot(BeNil())
				Expect(alias.Target.Name).To(Equal("time"))
			},
		)

		It(
			"Should resolve a module through the root's module resolver",
			func(bCtx SpecContext) {
				userModule := symbol.Symbol{Name: "lib.m", Kind: symbol.KindModule}
				root := symbol.NewRoot(
					nil,
					nil,
					symbol.WithModules(StaticResolver{userModule}),
				)
				prog := MustSucceed(parser.Parse(`import lib.m`))
				ctx := context.NewRoot(bCtx, prog, root)
				analyzer.AnalyzeProgram(ctx)
				alias := ctx.Scope.FindChild("m")
				Expect(alias).ToNot(BeNil())
				Expect(alias.Target.Name).To(Equal("lib.m"))
			},
		)

		It("Should diagnose a duplicate import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import ( time time )`))
			ctx := context.NewRoot(bCtx, prog, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.String()).To(ContainSubstring("duplicate import"))
		})

		DescribeTable(
			"Should give a fix that removes a duplicate import",
			func(bCtx SpecContext, source, fixed string) {
				ctx := context.NewRoot(bCtx, MustSucceed(parser.Parse(source)), root)
				analyzer.AnalyzeProgram(ctx)
				var fixes []diagnostics.Fix
				for _, d := range *ctx.Diagnostics {
					if d.Code == codes.DuplicateImport.Code {
						fixes = append(fixes, d.Fixes...)
					}
				}
				Expect(fixes).To(HaveLen(1))
				Expect(fixes[0].Title).To(Equal("Remove duplicate import"))
				Expect(testutil.ApplyFix(source, fixes[0])).To(Equal(fixed))
			},
			Entry(
				"in its own statement",
				"import time\nimport time\n\nfunc f() {}\n",
				"import time\n\nfunc f() {}\n",
			),
			Entry(
				"in a block",
				"import (\n    time\n    time\n)\n",
				"import (\n    time\n)\n",
			),
			Entry(
				"with the same alias",
				"import time as t\nimport time as t\n",
				"import time as t\n",
			),
		)

		It(
			"Should give no fix when a different module has the alias",
			func(bCtx SpecContext) {
				root.Parent.AddChild(&symbol.Symbol{
					Name: "math",
					Kind: symbol.KindModule,
				})
				prog := MustSucceed(parser.Parse("import time\nimport math as time\n"))
				ctx := context.NewRoot(bCtx, prog, root)
				analyzer.AnalyzeProgram(ctx)
				Expect(*ctx.Diagnostics).To(ContainElement(SatisfyAll(
					HaveField("Code", codes.DuplicateImport.Code),
					HaveField("Fixes", BeEmpty()),
				)))
			},
		)

		It("Should diagnose an unknown module", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import banana`))
			ctx := context.NewRoot(bCtx, prog, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(
				ctx.Diagnostics.String(),
			).To(ContainSubstring(`unknown module "banana"`))
		})

		It(
			"Should give a code to each error of the module resolver",
			func(bCtx SpecContext) {
				root := symbol.NewRoot(nil, nil, symbol.WithModules(failingModules{}))
				prog := MustSucceed(parser.Parse(`import fill`))
				ctx := context.NewRoot(bCtx, prog, root)
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(
					`[E4000]: import failed: "fill" is an automation and cannot be imported`,
				))
			},
		)

		It(
			"Should not double-report an unknown module as unused",
			func(bCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`import banana`))
				ctx := context.NewRoot(bCtx, prog, root)
				analyzer.AnalyzeProgram(ctx)
				Expect(
					ctx.Diagnostics.String(),
				).ToNot(ContainSubstring(`unused import: "banana"`))
			},
		)

		It("Should accept an import with no entries", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`import ()`))
			ctx := context.NewRoot(bCtx, prog, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It(
			"Should leave Children empty when there are no imports or declarations",
			func(bCtx SpecContext) {
				prog := MustSucceed(parser.Parse(`func f() {}`))
				ctx := context.NewRoot(bCtx, prog, root)
				analyzer.AnalyzeProgram(ctx)
				Expect(
					ctx.Scope.FilterChildrenByKind(symbol.KindModuleAlias),
				).To(BeEmpty())
			},
		)
	})

	Describe("reportUnusedImports", func() {
		It("Should flag an unused import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				import time
				func f() {}
			`))
			ctx := context.NewRoot(bCtx, prog, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(
				ctx.Diagnostics.String(),
			).To(ContainSubstring(`unused import: "time"`))
		})

		It("Should not flag a used import", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				import time
				func f() i64 { return time.now() }
			`))
			ctx := context.NewRoot(bCtx, prog, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.String()).ToNot(ContainSubstring("is unused"))
		})
	})
})
