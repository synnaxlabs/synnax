// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Loader", func() {
	var (
		sources map[string]string
		loads   map[string]int
		chans   = StaticResolver{
			{
				Name: "trig",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   100,
			},
			{
				Name: "out",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I64()),
				ID:   101,
			},
			{
				Name: "out8",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   102,
			},
		}
	)
	BeforeEach(func() {
		sources = map[string]string{}
		loads = map[string]int{}
	})
	newRoot := func() *symbol.Symbol {
		loader := text.NewLoader(text.LoaderConfig{
			Source: func(_ context.Context, path string) (text.Text, error) {
				loads[path]++
				src, ok := sources[path]
				if !ok {
					return text.Text{}, errors.Wrapf(
						symbol.ErrModuleNotFound,
						"%q",
						path,
					)
				}
				return text.Text{Raw: src}, nil
			},
			Channels: chans,
			Ambient:  stl.NewSymbols,
		})
		return symbol.NewRoot(chans, stl.NewSymbols(), symbol.WithModules(loader))
	}
	parse := func(source string) text.Text {
		GinkgoHelper()
		t, diag := text.Parse(text.Text{Raw: source})
		if diag != nil {
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		}
		return t
	}
	analyzeIR := func(
		ctx context.Context,
		source string,
	) (ir.IR, *diagnostics.Diagnostics) {
		GinkgoHelper()
		return text.Analyze(ctx, parse(source), newRoot())
	}
	analyze := func(ctx context.Context, source string) *diagnostics.Diagnostics {
		GinkgoHelper()
		_, diag := analyzeIR(ctx, source)
		return diag
	}
	analyzeModule := func(
		ctx context.Context,
		source string,
	) (ir.IR, *diagnostics.Diagnostics) {
		GinkgoHelper()
		return text.AnalyzeModule(ctx, parse(source), newRoot())
	}

	Describe("Resolution", func() {
		It(
			"Should resolve an exported function of a user module",
			func(ctx SpecContext) {
				sources["m"] = `export func f() i64 { return 1 }`
				diag := analyze(ctx, "import m\n\ntrig -> m.f() -> out")
				Expect(diag.Ok()).To(BeTrue(), diag.String())
			},
		)

		It("Should hide a member that is not exported", func(ctx SpecContext) {
			sources["m"] = `func f() i64 { return 1 }`
			diag := analyze(ctx, "import m\n\ntrig -> m.f() -> out")
			Expect(diag.String()).To(ContainSubstring("undefined symbol: f"))
		})

		It("Should bind an alias", func(ctx SpecContext) {
			sources["m"] = `export func f() i64 { return 1 }`
			diag := analyze(ctx, "import m as mm\n\ntrig -> mm.f() -> out")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})

		It("Should resolve a module by its full path", func(ctx SpecContext) {
			sources["lib.sub.m"] = `export func f() i64 { return 1 }`
			diag := analyze(ctx, "import lib.sub.m\n\ntrig -> m.f() -> out")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})

		It("Should need an alias for two modules with one name", func(ctx SpecContext) {
			sources["a.m"] = `export func f() i64 { return 1 }`
			sources["b.m"] = `export func g() i64 { return 2 }`
			diag := analyze(ctx, "import (a.m b.m)\n\ntrig -> m.f() -> out")
			Expect(diag.String()).To(ContainSubstring("duplicate import"))
			diag = analyze(
				ctx,
				"import (a.m b.m as bm)\n\ntrig -> m.f() -> out\ntrig -> bm.g() -> out",
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})

		It("Should resolve a module's own imports", func(ctx SpecContext) {
			sources["m"] = "import time\n\nexport func f() i64 { return time.now() }"
			diag := analyze(ctx, "import m\n\ntrig -> m.f() -> out")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})

		It("Should analyze a module imported twice once", func(ctx SpecContext) {
			sources["d"] = `export func h() i64 { return 1 }`
			sources["a"] = "import d\n\nexport func f() i64 { return d.h() }"
			sources["b"] = "import d\n\nexport func g() i64 { return d.h() }"
			diag := analyze(
				ctx,
				"import (a b)\n\ntrig -> a.f() -> out\ntrig -> b.g() -> out",
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(loads["d"]).To(Equal(1))
		})

		It("Should load a module that fails once", func(ctx SpecContext) {
			sources["d"] = `export func h() i64 { return nope }`
			sources["a"] = "import d\n\nexport func f() i64 { return d.h() }"
			sources["b"] = "import d\n\nexport func g() i64 { return d.h() }"
			diag := analyze(
				ctx,
				"import (a b)\n\ntrig -> a.f() -> out\ntrig -> b.g() -> out",
			)
			Expect(diag.String()).To(ContainSubstring(`module "a" does not compile`))
			Expect(diag.String()).To(ContainSubstring(`module "b" does not compile`))
			Expect(loads["d"]).To(Equal(1))
		})

		It("Should report an import cycle with its chain", func(ctx SpecContext) {
			sources["a"] = "import b\n\nexport func f() i64 { return b.g() }"
			sources["b"] = "import a\n\nexport func g() i64 { return a.f() }"
			diag := analyze(ctx, "import a\n\ntrig -> a.f() -> out")
			Expect(diag.String()).To(ContainSubstring("import cycle: a -> b -> a"))
		})

		It(
			"Should report a failure beside a cycle as its own",
			func(ctx SpecContext) {
				sources["a"] = "import c\n\nexport func f() i64 { return c.h() }"
				sources["c"] = "import a\n\nexport func h() i64 { return a.f() }"
				sources["b"] = `export func g() i64 { return nope }`
				diag := analyze(
					ctx,
					"import (a b)\n\ntrig -> a.f() -> out\ntrig -> b.g() -> out",
				)
				Expect(diag.String()).To(ContainSubstring("import cycle: a -> c -> a"))
				Expect(
					diag.String(),
				).To(ContainSubstring(`module "b" does not compile`))
			},
		)

		It("Should report a module that does not compile", func(ctx SpecContext) {
			sources["m"] = `export func f() i64 { return nope }`
			diag := analyze(ctx, "import m\n\ntrig -> m.f() -> out")
			Expect(diag.String()).To(ContainSubstring(`module "m" does not compile`))
		})

		It("Should report a module that does not parse", func(ctx SpecContext) {
			sources["m"] = `export func f( {`
			diag := analyze(ctx, "import m\n\ntrig -> m.f() -> out")
			Expect(diag.String()).To(ContainSubstring(`module "m" does not compile`))
		})

		Describe("Imported sequences", func() {
			BeforeEach(func() {
				sources["m"] = `export sequence s { stage a { trig -> out8 } }`
			})

			It("Should activate by the qualified name", func(ctx SpecContext) {
				diag := analyze(ctx, "import m\n\ntrig > 0 => m.s")
				Expect(diag.Ok()).To(BeTrue(), diag.String())
			})

			It(
				"Should not bind to a sibling stage of the same name",
				func(ctx SpecContext) {
					i, diag := analyzeIR(ctx, `
import m

sequence main {
    stage s { trig -> out8 }
    stage b { trig > 0 => m.s }
}
`)
					Expect(diag.Ok()).To(BeTrue(), diag.String())
					main := findTopLevelScope(i, "main")
					Expect(main.Transitions).To(HaveLen(1))
					Expect(main.Transitions[0].Target.Variant).To(
						Equal(ir.ScopeTarget{Key: "m.s"}),
					)
				},
			)

			It(
				"Should not activate a local scope of the same name",
				func(ctx SpecContext) {
					i, diag := analyzeIR(ctx, `
import m

sequence s { stage a { trig -> out8 } }

trig > 0 => m.s
`)
					Expect(diag.Ok()).To(BeTrue(), diag.String())
					Expect(findTopLevelScope(i, "s").Activations).To(BeEmpty())
				},
			)
		})

		It("Should not search import paths", func(ctx SpecContext) {
			loader := text.NewLoader(text.LoaderConfig{})
			Expect(loader.Search(ctx, "m")).To(BeEmpty())
		})

		It("Should report an unknown module", func(ctx SpecContext) {
			diag := analyze(ctx, "import nope\n\ntrig -> nope.f() -> out")
			Expect(diag.String()).To(ContainSubstring(`unknown module "nope"`))
		})
	})

	Describe("Module rules", func() {
		It("Should reject export outside a module", func(ctx SpecContext) {
			diag := analyze(ctx, `export func f() i64 { return 1 }`)
			Expect(
				diag.String(),
			).To(ContainSubstring("export is only valid in a module"))
		})

		It("Should reject a top-level flow in a module", func(ctx SpecContext) {
			_, diag := analyzeModule(ctx, "trig -> out")
			Expect(
				diag.String(),
			).To(ContainSubstring("a module cannot have a top-level flow"))
		})

		It("Should reject an authority block in a module", func(ctx SpecContext) {
			_, diag := analyzeModule(ctx, "authority 255")
			Expect(diag.String()).To(
				ContainSubstring("a module cannot have an authority block"),
			)
		})

		It("Should reject an anonymous exported sequence", func(ctx SpecContext) {
			_, diag := analyzeModule(ctx, "export sequence { stage s {} }")
			Expect(diag.String()).To(
				ContainSubstring("an exported declaration needs a name"),
			)
		})

		It("Should mark exported declarations", func(ctx SpecContext) {
			i, diag := analyzeModule(ctx, `
export func f() i64 { return 1 }
func g() i64 { return 2 }
export sequence s { stage a { trig -> out8 } }
export stage t { trig -> out8 }
`)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(i.Symbols.FindChild("f").Exported).To(BeTrue())
			Expect(i.Symbols.FindChild("g").Exported).To(BeFalse())
			Expect(i.Symbols.FindChild("s").Exported).To(BeTrue())
			Expect(i.Symbols.FindChild("t").Exported).To(BeTrue())
		})

		It("Should reject an exported stateful variable", func(ctx SpecContext) {
			_, diag := analyzeModule(ctx, "export s $= 1")
			Expect(diag.String()).To(
				ContainSubstring(
					"stateful variables cannot be declared at the top level",
				),
			)
		})

		It("Should export a variable of any kind", func(ctx SpecContext) {
			i, diag := analyzeModule(ctx, `
export a := 1
export b := trig
export c := 1 + 2
`)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			for _, name := range []string{"a", "b", "c"} {
				Expect(i.Symbols.FindChild(name).Exported).To(BeTrue(), name)
			}
		})
	})
})
