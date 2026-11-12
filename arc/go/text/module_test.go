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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
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
				Name: "outf",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   103,
			},
			{
				Name: "out8",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   102,
			},
			{
				Name: "outs",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.String()),
				ID:   103,
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

		It(
			"Should read a user module constant and a math constant the same way",
			func(ctx SpecContext) {
				sources["m"] = `export limit := 5`
				i, diag := analyzeIR(
					ctx,
					"import (m math)\n\nm.limit -> out\nmath.inf -> outf",
				)
				Expect(diag.Ok()).To(BeTrue(), diag.String())
				var constants []string
				for _, n := range i.Nodes {
					if n.Type == "constant" {
						constants = append(constants, n.Key)
					}
				}
				Expect(constants).To(HaveLen(2))
			},
		)

		It("Should hide a member that is not exported", func(ctx SpecContext) {
			sources["m"] = `func f() i64 { return 1 }`
			diag := analyze(ctx, "import m\n\ntrig -> m.f() -> out")
			Expect(diag.String()).To(
				ContainSubstring(`'f' is not exported by module "m"`),
			)
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
			sources["m"] = "import time\n\nexport func f() i64 ns { return time.now() }"
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
			Expect(diag.String()).To(ContainSubstring(
				`module "a" does not compile: 1:7 module "d" does not compile: ` +
					`1:29 undefined symbol: nope`,
			))
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
			Expect(diag.String()).To(ContainSubstring(
				`module "m" does not compile: 1:29 undefined symbol: nope`,
			))
		})

		It("Should report a module that does not parse", func(ctx SpecContext) {
			sources["m"] = `export func f( {`
			diag := analyze(ctx, "import m\n\ntrig -> m.f() -> out")
			Expect(diag.String()).To(ContainSubstring(`module "m" does not compile`))
		})

		It(
			"Should report a module that does not compile once, at the import",
			func(ctx SpecContext) {
				sources["m"] = "export c := 1\n\nexport abc := \"a\"1."
				diag := analyze(ctx, "import m\n\n"+
					"f\"{m.c}\" -> outs\n\nm.abc -> outs\n\ntrig -> m.f() -> out")
				Expect(*diag).To(HaveLen(1), diag.String())
				Expect((*diag)[0].Message).To(Equal(
					`module "m" does not compile: 3:19 mismatched input '<EOF>' ` +
						`expecting {'->', '=>'}`,
				))
				Expect((*diag)[0].Range.Start.Line).To(BeZero())
			},
		)

		It(
			"Should report an unknown module once, at the import",
			func(ctx SpecContext) {
				diag := analyze(ctx, "import nope\n\ntrig -> nope.f() -> out")
				Expect(*diag).To(HaveLen(1), diag.String())
				Expect((*diag)[0].Message).To(Equal(`unknown module "nope"`))
			},
		)

		Describe("Imported sequences", func() {
			BeforeEach(func() {
				sources["m"] = `export sequence s { stage a { trig -> out8 } }`
			})

			It("Should bind a top-level activation to the module scope", func(
				ctx SpecContext,
			) {
				i, diag := analyzeIR(ctx, "import m\n\ntrig > 0 => m.s")
				Expect(diag.Ok()).To(BeTrue(), diag.String())
				Expect(findTopLevelScope(i, "m.s").Activations).To(HaveLen(1))
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

			It("Should key an activation by the module path, not the alias", func(
				ctx SpecContext,
			) {
				i, diag := analyzeIR(ctx, `
import m as mm

sequence main {
    stage b { trig > 0 => mm.s }
}
`)
				Expect(diag.Ok()).To(BeTrue(), diag.String())
				main := findTopLevelScope(i, "main")
				Expect(main.Transitions[0].Target.Variant).To(
					Equal(ir.ScopeTarget{Key: "m.s"}),
				)
			})

			It(
				"Should not activate a local scope of the same name",
				func(ctx SpecContext) {
					i, diag := analyzeIR(ctx, `
import m

sequence s { stage a { trig -> out8 } }

sequence main {
    stage b { trig > 0 => m.s }
}
`)
					Expect(diag.Ok()).To(BeTrue(), diag.String())
					Expect(findTopLevelScope(i, "s").Activations).To(BeEmpty())
					main := findTopLevelScope(i, "main")
					Expect(main.Transitions[0].Target.Variant).To(
						Equal(ir.ScopeTarget{Key: "m.s"}),
					)
				},
			)
		})

		Describe("Search", func() {
			It(
				"Should return the paths that start with the term",
				func(ctx SpecContext) {
					loader := text.NewLoader(text.LoaderConfig{
						Paths: func(context.Context) ([]string, error) {
							return []string{"lib.m", "lib.sub.n", "other"}, nil
						},
					})
					modules := MustSucceed(loader.Search(ctx, "lib."))
					Expect(modules).To(HaveLen(2))
					Expect(modules[0].Name).To(Equal("lib.m"))
					Expect(modules[0].Kind).To(Equal(symbol.KindModule))
					Expect(modules[1].Name).To(Equal("lib.sub.n"))
				},
			)

			It("Should return nothing without paths", func(ctx SpecContext) {
				loader := text.NewLoader(text.LoaderConfig{})
				Expect(loader.Search(ctx, "m")).To(BeEmpty())
			})

			It("Should return the error of paths", func(ctx SpecContext) {
				loader := text.NewLoader(text.LoaderConfig{
					Paths: func(context.Context) ([]string, error) {
						return nil, errors.New("cat")
					},
				})
				Expect(loader.Search(ctx, "m")).Error().To(MatchError("cat"))
			})
		})

		It("Should report an unknown module", func(ctx SpecContext) {
			diag := analyze(ctx, "import nope\n\ntrig -> nope.f() -> out")
			Expect(diag.String()).To(ContainSubstring(`unknown module "nope"`))
		})
	})

	Describe("A module variable of another unit", func() {
		analyzeStage := func(ctx context.Context, statement string) diagnostics.Diagnostic {
			GinkgoHelper()
			sources["m"] = `export t := 120s`
			diag := analyze(ctx, "import (m time)\n\nsequence main {\nd := 1min\n"+
				"twice := m.t * 2\nstage s1 {\n"+statement+"\n}\n}")
			Expect(*diag).To(HaveLen(1), diag.String())
			return (*diag)[0]
		}

		DescribeTable("Should warn when a write of a module variable truncates",
			func(ctx SpecContext, statement string) {
				d := analyzeStage(ctx, statement+"\ntrig -> time.wait(1s) -> out8")
				Expect(d.Code).To(Equal(codes.UnitTruncation.Code))
			},
			Entry("the variable", `d = m.t`),
			Entry("an assigned calculation", `d = m.t * 2`),
			Entry("a calculation in a flow", `m.t * 2 -> d`),
			Entry("a variable that the module variable sets", `d = twice * 2`),
		)

		It("Should accept a calculation that goes to an input", func(ctx SpecContext) {
			sources["m"] = `export t := 120s`
			diag := analyze(
				ctx,
				"import (m time)\n\ntrig -> time.wait(m.t * 2) -> out8",
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(*diag).To(BeEmpty())
		})
	})

	Describe("Loaded paths", func() {
		It("Should list every module loaded, at any depth, by path", func(
			ctx SpecContext,
		) {
			sources["d"] = `export func h() i64 { return 1 }`
			sources["a"] = "import d\n\nexport func f() i64 { return d.h() }"
			root := newRoot()
			_, diag := text.Analyze(
				ctx,
				parse("import a\n\ntrig -> a.f() -> out"),
				root,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(text.LoadedPaths(root)).To(Equal([]string{"a", "d"}))
		})

		It("Should leave out a module that does not compile", func(ctx SpecContext) {
			sources["a"] = `export func f() i64 { return nope }`
			root := newRoot()
			_, diag := text.Analyze(
				ctx,
				parse("import a\n\ntrig -> a.f() -> out"),
				root,
			)
			Expect(diag.Ok()).To(BeFalse())
			Expect(text.LoadedPaths(root)).To(BeEmpty())
		})

		It("Should list nothing for a root with no module resolver", func() {
			Expect(text.LoadedPaths(symbol.NewRoot(chans, nil))).To(BeNil())
		})
	})

	Describe("Keys", func() {
		// A loaded module lowers with its import path on every key it owns.
		analyzeAt := func(ctx context.Context, path, source string) ir.IR {
			GinkgoHelper()
			root := newRoot()
			root.Name, root.UserDefined = path, true
			i, diag := text.AnalyzeModule(ctx, parse(source), root)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			return i
		}

		It("Should key a function by the module path", func(ctx SpecContext) {
			i := analyzeAt(ctx, "lib.m", `export func f() i64 { return 1 }`)
			Expect(i.Functions).To(HaveLen(1))
			Expect(i.Functions[0].Key).To(Equal("lib.m.f"))
			Expect(i.Functions[0].Symbol).To(Equal(i.Symbols.FindChild("f")))
		})

		It("Should key a call node's type by the module path", func(ctx SpecContext) {
			i := analyzeAt(ctx, "lib.m", `
func f() i64 { return 1 }
export sequence s { stage a { trig -> f() -> out } }
`)
			var nodeTypes []string
			for _, n := range i.Nodes {
				nodeTypes = append(nodeTypes, n.Type)
			}
			Expect(nodeTypes).To(ContainElement("lib.m.f"))
		})

		It("Should prefix every node key with the module path", func(ctx SpecContext) {
			i := analyzeAt(ctx, "lib.m", `
export sequence s { stage a { trig -> out8 } }
export stage t { trig -> out8 }
`)
			Expect(i.Nodes).ToNot(BeEmpty())
			for _, n := range i.Nodes {
				Expect(n.Key).To(HavePrefix("lib.m."), n.Key)
			}
		})

		It("Should key top-level scopes by the module path", func(ctx SpecContext) {
			i := analyzeAt(ctx, "lib.m", `
export sequence s { stage a { trig -> out8 } }
export stage t { trig -> out8 }
`)
			Expect(findTopLevelScope(i, "lib.m.s").Steps[0].Key()).To(Equal("a"))
			Expect(findTopLevelScope(i, "lib.m.t").Key).To(Equal("lib.m.t"))
		})

		It(
			"Should keep a sibling activation on the module path",
			func(ctx SpecContext) {
				i := analyzeAt(ctx, "lib.m", `
export sequence s { stage a { trig > 0 => t } }
sequence t { stage b { trig -> out8 } }
`)
				s := findTopLevelScope(i, "lib.m.s")
				Expect(s.Transitions[0].Target.Variant).To(
					Equal(ir.ScopeTarget{Key: "lib.m.t"}),
				)
			},
		)
	})

	Describe("Link", func() {
		functionKeys := func(i ir.IR) []string {
			keys := make([]string, 0, len(i.Functions))
			for _, f := range i.Functions {
				keys = append(keys, f.Key)
			}
			return keys
		}
		nodeTypes := func(i ir.IR) []string {
			nodeTypes := make([]string, 0, len(i.Nodes))
			for _, n := range i.Nodes {
				nodeTypes = append(nodeTypes, n.Type)
			}
			return nodeTypes
		}

		It("Should link an exported function behind a call node", func(
			ctx SpecContext,
		) {
			sources["m"] = `export func f() i64 { return 1 }`
			i, diag := analyzeIR(ctx, "import m\n\ntrig -> m.f() -> out")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(functionKeys(i)).To(ConsistOf("m.f"))
			Expect(nodeTypes(i)).To(ContainElement("m.f"))
			f, found := i.Functions.Find("m.f")
			Expect(found).To(BeTrue())
			Expect(f.Symbol.Root()).ToNot(Equal(i.Symbols))
		})

		It("Should link a function called from a body", func(ctx SpecContext) {
			sources["m"] = `export func f() i64 { return 1 }`
			i, diag := analyzeIR(
				ctx,
				"import m\n\nfunc g() i64 { return m.f() }\n\ntrig -> g() -> out",
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(functionKeys(i)).To(ConsistOf("g", "m.f"))
		})

		It("Should link each module once across importers", func(ctx SpecContext) {
			sources["d"] = `export func h() i64 { return 1 }`
			sources["a"] = "import d\n\nexport func f() i64 { return d.h() }"
			sources["b"] = "import d\n\nexport func g() i64 { return d.h() }"
			i, diag := analyzeIR(
				ctx,
				"import (a b)\n\ntrig -> a.f() -> out\ntrig -> b.g() -> out",
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(functionKeys(i)).To(ConsistOf("a.f", "b.g", "d.h"))
		})

		It("Should link two modules of one name under aliases", func(ctx SpecContext) {
			sources["a.m"] = `export func f() i64 { return 1 }`
			sources["b.m"] = `export func f() i64 { return 2 }`
			i, diag := analyzeIR(
				ctx,
				"import (a.m b.m as bm)\n\ntrig -> m.f() -> out\ntrig -> bm.f() -> out",
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(functionKeys(i)).To(ConsistOf("a.m.f", "b.m.f"))
			Expect(nodeTypes(i)).To(ContainElements("a.m.f", "b.m.f"))
		})

		It("Should link an exported sequence and its nodes", func(ctx SpecContext) {
			sources["m"] = `export sequence s { stage a { trig -> out8 } }`
			i, diag := analyzeIR(ctx, "import m\n\ntrig > 0 => m.s")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			s := findTopLevelScope(i, "m.s")
			Expect(s.Steps[0].Key()).To(Equal("a"))
			Expect(s.Activations).To(HaveLen(1))
			var keys []string
			for _, n := range i.Nodes {
				keys = append(keys, n.Key)
			}
			Expect(keys).To(ContainElement(HavePrefix("m.")))
		})

		It("Should give linked members a root stratum of their own", func(
			ctx SpecContext,
		) {
			sources["m"] = `
export func f() i64 { return 1 }
export sequence s { stage a { trig -> out8 } }
`
			i, diag := analyzeIR(ctx, "import m\n\nfunc g() i64 { return m.f() }")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(findTopLevelScope(i, "m.s").Key).To(Equal("m.s"))
		})

		It("Should link an exported stage and its nodes", func(ctx SpecContext) {
			sources["m"] = `export stage t { trig -> out8 }`
			i, diag := analyzeIR(ctx, "import m\n\ntrig > 0 => m.t")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			t := findTopLevelScope(i, "m.t")
			Expect(t.Activations).To(HaveLen(1))
			var keys []string
			for _, stratum := range t.Strata {
				for _, member := range stratum {
					keys = append(keys, *member.NodeKey)
				}
			}
			Expect(keys).To(ConsistOf("m.on_trig_0", "m.write_out8_0"))
		})

		It("Should keep a nested jump to a module scope", func(ctx SpecContext) {
			sources["m"] = `export sequence s { stage a { trig -> out8 } }`
			i, diag := analyzeIR(ctx, `
import m

sequence main {
    stage b { trig > 0 => m.s }
}
`)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(findTopLevelScope(i, "main").Transitions[0].Target.Variant).To(
				Equal(ir.ScopeTarget{Key: "m.s"}),
			)
			Expect(findTopLevelScope(i, "m.s").Key).To(Equal("m.s"))
		})

		It("Should keep an importer's inline activation beside a module's", func(
			ctx SpecContext,
		) {
			sources["m"] = `
export sequence s { stage a { trig -> sequence { stage b { trig -> out8 } } } }
`
			i, diag := analyzeIR(ctx, `
import m

trig -> sequence { stage c { trig -> out8 } }
trig > 0 => m.s
`)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			var inline []ir.Scope
			var walk func(s ir.Scope)
			walk = func(s ir.Scope) {
				for _, stratum := range s.Strata {
					for _, m := range stratum {
						if m.Scope != nil {
							walk(*m.Scope)
						}
					}
				}
				for _, m := range s.Steps {
					if m.Scope != nil {
						walk(*m.Scope)
					}
				}
				if strings.HasPrefix(s.Key, ir.InlinePrefix) {
					inline = append(inline, s)
				}
			}
			walk(i.Root)
			Expect(inline).To(HaveLen(2))
			for _, s := range inline {
				Expect(s.Activations).To(HaveLen(1), s.Key)
			}
			Expect(inline[0].Activations[0].On).ToNot(
				Equal(inline[1].Activations[0].On),
			)
		})

		It("Should read an exported channel alias as its channel", func(
			ctx SpecContext,
		) {
			sources["m"] = `export b := trig`
			i, diag := analyzeIR(ctx, "import m\n\nm.b -> out8")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			Expect(i.Nodes).To(HaveLen(2))
			Expect(i.Nodes[0].Type).To(Equal("on"))
			Expect(i.Nodes[0].Inputs[0].Value).To(Equal(uint32(100)))
		})

		It("Should read an exported expression variable from its node", func(
			ctx SpecContext,
		) {
			sources["m"] = `export c := trig + 1`
			i, diag := analyzeIR(ctx, "import m\n\nm.c -> out8")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			var sources []string
			for _, e := range i.Edges {
				if e.Target.Node == "write_out8_0" {
					sources = append(sources, e.Source.Node)
				}
			}
			Expect(sources).To(ConsistOf(HavePrefix("m.var_c")))
		})

		It("Should reject a write to an imported variable", func(ctx SpecContext) {
			sources["m"] = "export b := trig\nexport k := 1"
			diag := analyze(ctx, "import m\n\ntrig -> m.b")
			Expect(diag.String()).To(ContainSubstring(
				"cannot write to 'm.b': it belongs to an imported module",
			))
			diag = analyze(ctx, "import m\n\ntrig -> m.k")
			Expect(diag.String()).To(ContainSubstring(
				"cannot reassign top-level variable 'm.k'",
			))
		})

		It("Should let a module write its own sequence variable", func(
			ctx SpecContext,
		) {
			sources["m"] = `
export sequence s {
	total u8 $= 0
	stage a { trig -> total }
}`
			diag := analyze(ctx, "import m\n\ntrig > 0 => m.s")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})

		It("Should read a dependency's expression variable from a module", func(
			ctx SpecContext,
		) {
			sources["b"] = `export c := trig + 1`
			sources["a"] = "import b\n\nexport stage s { b.c -> out8 }"
			i, diag := analyzeIR(ctx, "import a\n\ntrig > 0 => a.s")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			var sources []string
			for _, e := range i.Edges {
				if e.Target.Node == "a.write_out8_0" {
					sources = append(sources, e.Source.Node)
				}
			}
			Expect(sources).To(ConsistOf(HavePrefix("b.var_c")))
			Expect(i.Nodes).To(HaveLen(len(set.New(lo.Map(i.Nodes,
				func(n ir.Node, _ int) string { return n.Key })...))))
		})

		It("Should link a module's reassigned variable once", func(ctx SpecContext) {
			sources["m"] = `
export sequence s {
	k := trig + 1
	stage a { k = trig + 2 }
	stage b { k -> out8 }
}`
			i, diag := analyzeIR(ctx, "import m\n\ntrig > 0 => m.s")
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			keys := functionKeys(i)
			Expect(keys).To(HaveLen(len(set.New(keys...))))
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
