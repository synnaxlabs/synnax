// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Scope", func() {
	Describe("Root", func() {
		It("Should create a new root scope", func() {
			s := symbol.NewRoot(nil, nil)
			Expect(s.Children()).To(BeEmpty())
		})

		It("Should resolve through the given global resolver on a miss", func(bCtx SpecContext) {
			ch := symbol.Symbol{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8())}
			s := symbol.NewRoot(StaticResolver{ch}, nil)
			Expect(MustSucceed(s.Resolve(bCtx, "ch")).Name).To(Equal("ch"))
		})

		It("Should resolve lexically through a resolver set after construction", func(bCtx SpecContext) {
			enc := symbol.NewRoot(nil, nil)
			MustSucceed(enc.Add(bCtx, symbol.Symbol{
				Name: "shared", Kind: symbol.KindVariable, Type: types.I32(),
			}))
			fn := symbol.NewRoot(nil, nil)
			fn.SetLexicalResolver(enc)
			Expect(MustSucceed(fn.Resolve(bCtx, "shared")).Name).To(Equal("shared"))
			Expect(MustSucceed(fn.Search(bCtx, "shared"))).ToNot(BeEmpty())
		})
	})

	Describe("Add", func() {
		It("Should add a new variable scope", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			varScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(varScope.Name).To(Equal("x"))
			Expect(varScope.Type).To(Equal(types.I32()))
			Expect(varScope.Children()).To(BeEmpty())
			By("Using the root scope counter for it's ID")
			Expect(varScope.ID).To(Equal(0))
		})

		It("Should add a new function scope", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "my_func", Kind: symbol.KindFunction},
			))
			Expect(funcScope.Name).To(Equal("my_func"))
		})

		It("Should add a new func scope", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			stageScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "my_task", Kind: symbol.KindBlock},
			))
			Expect(stageScope.Name).To(Equal("my_task"))
		})

		DescribeTable("Should assign IDs to slot-allocating kinds",
			func(bCtx SpecContext, kind symbol.Kind) {
				rootScope := symbol.NewRoot(nil, nil)
				scope1 := MustSucceed(rootScope.Add(
					bCtx,
					symbol.Symbol{Name: "var1", Kind: kind, Type: types.I32()},
				))
				scope2 := MustSucceed(rootScope.Add(
					bCtx,
					symbol.Symbol{Name: "var2", Kind: kind, Type: types.I32()},
				))
				Expect(scope1.ID).To(Equal(0))
				Expect(scope2.ID).To(Equal(1))
			},
			Entry("Variable", symbol.KindVariable),
			Entry("StatefulVariable", symbol.KindStatefulVariable),
			Entry("Input", symbol.KindInput),
			Entry("Output", symbol.KindOutput),
			Entry("LoopVariable", symbol.KindLoopVariable),
		)

		It("Should give KindSequence its own ID space", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			seqScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "my_seq", Kind: symbol.KindSequence},
			))
			inner := MustSucceed(seqScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(inner.ID).To(Equal(0))
		})

		It("Should give KindFunction a Channels container", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "f", Kind: symbol.KindFunction},
			))
			Expect(funcScope.Channels.Read).ToNot(BeNil())
			Expect(funcScope.Channels.Write).ToNot(BeNil())
		})

		It("Should not give KindSequence a Channels container", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			seqScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "s", Kind: symbol.KindSequence},
			))
			Expect(seqScope.Channels.Read).To(BeNil())
		})

		It("Should correctly increment IDs for variables within function scopes", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "my_func", Kind: symbol.KindFunction},
			))
			firstVarScope := MustSucceed(funcScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(firstVarScope.ID).To(Equal(0))
			Expect(firstVarScope.Parent).ToNot(BeNil())
			Expect(firstVarScope.Parent).To(Equal(funcScope))
			secondVarScope := MustSucceed(funcScope.Add(
				bCtx,
				symbol.Symbol{Name: "y", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(secondVarScope.ID).To(Equal(1))
			Expect(secondVarScope.Parent).ToNot(BeNil())
			Expect(secondVarScope.Parent).To(Equal(funcScope))
		})

		It("Should not return error when adding duplicate symbol that shadows a global", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			scope1 := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(scope1).ToNot(BeNil())
			Expect(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I64()},
			)).ToNot(BeNil())
		})
		It("Should allow shadowing global symbols from resolver", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "x", Kind: symbol.KindInput, Type: types.F64()},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
		})
		It("Should resolve to local symbol when shadowing global", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "x", Kind: symbol.KindInput, Type: types.F64()},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			localScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			resolved := MustSucceed(rootScope.Resolve(bCtx, "x"))
			Expect(resolved).To(Equal(localScope))
			Expect(resolved.Kind).To(Equal(symbol.KindVariable))
			Expect(resolved.Type).To(Equal(types.I32()))
		})
		It("Should resolve to local symbol when shadowing global in nested scope", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "x", Kind: symbol.KindInput, Type: types.F64()},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			localScope := MustSucceed(funcScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			resolved := MustSucceed(funcScope.Resolve(bCtx, "x"))
			Expect(resolved).To(Equal(localScope))
			Expect(resolved.Type).To(Equal(types.I32()))
		})
		It("Should allow symbols with empty names", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			child := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "", Kind: symbol.KindBlock}))
			Expect(child.Name).To(Equal(""))
		})
	})

	Describe("GetChildByParserRule", func() {
		It("Should find child by parser rule", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			rule := antlr.NewBaseParserRuleContext(nil, 0)
			child := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32(), AST: rule},
			))
			found := MustSucceed(rootScope.GetChildByParserRule(rule))
			Expect(found).To(Equal(child))
		})

		It("Should return error when parser rule not found", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			scope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(scope).ToNot(BeNil())
			Expect(rootScope.GetChildByParserRule(antlr.NewBaseParserRuleContext(nil, 0))).Error().To(MatchError(ContainSubstring("could not find symbol matching parser rule")))
		})
	})

	Describe("FindChild", func() {
		It("Should find child by name", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			child := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			found := rootScope.FindChild("x")
			Expect(found).To(Equal(child))
		})

		It("Should return nil when name not found", func() {
			rootScope := symbol.NewRoot(nil, nil)
			found := rootScope.FindChild("nonexistent")
			Expect(found).To(BeNil())
		})
	})

	Describe("AddChild", func() {
		It("Should append the child and set its Parent", func() {
			parent := symbol.NewRoot(nil, nil)
			child := &symbol.Symbol{Name: "host_fn", Kind: symbol.KindFunction}
			parent.AddChild(child)
			Expect(child.Parent).To(Equal(parent))
			Expect(parent.FindChild("host_fn")).To(Equal(child))
		})

		It("Should return the receiver for chaining", func() {
			parent := symbol.NewRoot(nil, nil)
			child := &symbol.Symbol{Name: "host_fn", Kind: symbol.KindFunction}
			Expect(parent.AddChild(child)).To(Equal(parent))
		})

		It("Should append every child when called variadically", func() {
			parent := symbol.NewRoot(nil, nil)
			first := &symbol.Symbol{Name: "first", Kind: symbol.KindFunction}
			second := &symbol.Symbol{Name: "second", Kind: symbol.KindFunction}
			parent.AddChild(first, second)
			Expect(first.Parent).To(Equal(parent))
			Expect(second.Parent).To(Equal(parent))
			Expect(parent.FindChild("first")).To(Equal(first))
			Expect(parent.FindChild("second")).To(Equal(second))
		})

		It("Should append in insertion order", func() {
			parent := symbol.NewRoot(nil, nil)
			first := &symbol.Symbol{Name: "first", Kind: symbol.KindFunction}
			second := &symbol.Symbol{Name: "second", Kind: symbol.KindFunction}
			parent.AddChild(first)
			parent.AddChild(second)
			children := parent.Children()
			Expect(children[len(children)-2]).To(Equal(first))
			Expect(children[len(children)-1]).To(Equal(second))
		})

		It("Should not check for naming conflicts", func() {
			parent := symbol.NewRoot(nil, nil)
			parent.AddChild(&symbol.Symbol{Name: "dup", Kind: symbol.KindFunction})
			parent.AddChild(&symbol.Symbol{Name: "dup", Kind: symbol.KindFunction})
			conflicts := 0
			for _, c := range parent.Children() {
				if c.Name == "dup" {
					conflicts++
				}
			}
			Expect(conflicts).To(Equal(2))
		})

		It("Should not assign an ID", func() {
			parent := symbol.NewRoot(nil, nil)
			child := &symbol.Symbol{Name: "x", Kind: symbol.KindVariable}
			parent.AddChild(child)
			Expect(child.ID).To(Equal(0))
		})
	})

	Describe("Root", func() {
		It("Should return root scope from any depth", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			varScope := MustSucceed(funcScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(varScope.Root()).To(Equal(rootScope))
			Expect(funcScope.Root()).To(Equal(rootScope))
			Expect(rootScope.Root()).To(Equal(rootScope))
		})
	})

	Describe("Resolve", func() {
		It("Should resolve symbol in current scope", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			child := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			resolved := MustSucceed(rootScope.Resolve(bCtx, "x"))
			Expect(resolved).To(Equal(child))
		})
		It("Should resolve symbol from parent scope", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			global := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "global", Kind: symbol.KindVariable, Type: types.I32()}))
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			resolved := MustSucceed(funcScope.Resolve(bCtx, "global"))
			Expect(resolved).To(Equal(global))
		})
		It("Should resolve from global resolver", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "pi", Kind: symbol.KindInput, Type: types.F64()},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			resolved := MustSucceed(rootScope.Resolve(bCtx, "pi"))
			Expect(resolved.Name).To(Equal("pi"))
			Expect(resolved.Kind).To(Equal(symbol.KindInput))
		})
		It("Should prioritize local over parent scope", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			rootX := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			resolvedFromFunc := MustSucceed(funcScope.Resolve(bCtx, "x"))
			Expect(resolvedFromFunc).To(Equal(rootX))
		})
		It("Should return error for undefined symbol", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			Expect(rootScope.Resolve(bCtx, "undefined")).Error().To(
				MatchError(ContainSubstring("undefined symbol: undefined")),
			)
		})
		It("Should skip internal symbols from global resolver", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "host_fn", Kind: symbol.KindFunction, Type: types.F64(), Internal: true},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			Expect(rootScope.Resolve(bCtx, "host_fn")).Error().To(MatchError(ContainSubstring("undefined symbol: host_fn")))
		})
		It("Should resolve non-internal symbols from global resolver alongside internal ones", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "host_fn", Kind: symbol.KindFunction, Type: types.F64(), Internal: true},
				{Name: "user_fn", Kind: symbol.KindFunction, Type: types.F64()},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			resolved := MustSucceed(rootScope.Resolve(bCtx, "user_fn"))
			Expect(resolved.Name).To(Equal("user_fn"))
		})
		It("Should resolve internal symbols when IncludeInternal is passed", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "host_fn", Kind: symbol.KindFunction, Type: types.F64(), Internal: true},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			resolved := MustSucceed(rootScope.Resolve(bCtx, "host_fn", symbol.IncludeInternal))
			Expect(resolved.Name).To(Equal("host_fn"))
			Expect(resolved.Internal).To(BeTrue())
		})
		It("Should match a numeric name by ID rather than by Name", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			first := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "a", Kind: symbol.KindVariable, Type: types.I32()},
			))
			second := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "b", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(first.ID).To(Equal(0))
			Expect(second.ID).To(Equal(1))
			byID := MustSucceed(rootScope.Resolve(bCtx, "1"))
			Expect(byID).To(Equal(second))
		})
		It("Should report the origin scope on undefined-symbol error after a multi-level walk", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "f", Kind: symbol.KindFunction},
			))
			innerScope := MustSucceed(funcScope.Add(
				bCtx,
				symbol.Symbol{Name: "inner", Kind: symbol.KindBlock},
			))
			_, err := innerScope.Resolve(bCtx, "missing")
			var undefinedErr *symbol.UndefinedSymbolError
			Expect(errors.As(err, &undefinedErr)).To(BeTrue())
			Expect(undefinedErr.Name).To(Equal("missing"))
		})

		Describe("Module aliases", func() {
			// buildAmbientRoot constructs a root with a synthetic ambient
			// prelude that has a single "time" module containing "now".
			buildAmbientRoot := func(bCtx SpecContext) *symbol.Symbol {
				timeMod := &symbol.Symbol{Name: "time", Kind: symbol.KindModule}
				timeMod.AddChild(&symbol.Symbol{
					Name: "now", Kind: symbol.KindFunction, Type: types.F64(),
				})
				ambient := &symbol.Symbol{Kind: symbol.KindAmbient}
				ambient.AddChild(timeMod)
				root := symbol.NewRoot(nil, nil)
				ambient.AddChild(root)
				return root
			}

			// resolveQualified mirrors the caller-side pattern that AST walkers
			// use after Resolve was narrowed to single names: look up the head,
			// follow alias.Target if present, then look up the tail against
			// the resulting symbol's children.
			resolveQualified := func(
				bCtx SpecContext,
				scope *symbol.Symbol,
				head, tail string,
			) (*symbol.Symbol, error) {
				sym, err := scope.Resolve(bCtx, head)
				if err != nil {
					return nil, err
				}
				container := sym
				if container.Target != nil {
					container = container.Target
				}
				return container.Resolve(bCtx, tail)
			}

			It("Should fail head lookup when no alias exists for the module", func(bCtx SpecContext) {
				rootScope := buildAmbientRoot(bCtx)
				Expect(resolveQualified(bCtx, rootScope, "time", "now")).Error().
					To(MatchError(ContainSubstring("undefined symbol")))
			})

			It("Should resolve a qualified lookup through a same-name alias", func(bCtx SpecContext) {
				rootScope := buildAmbientRoot(bCtx)
				timeMod := rootScope.Parent.FindChild("time")
				MustSucceed(rootScope.Add(bCtx, symbol.Symbol{
					Name: "time", Kind: symbol.KindModuleAlias, Target: timeMod,
				}))
				resolved := MustSucceed(resolveQualified(bCtx, rootScope, "time", "now"))
				Expect(resolved.Name).To(Equal("now"))
			})

			It("Should resolve a qualified lookup through a renamed alias", func(bCtx SpecContext) {
				rootScope := buildAmbientRoot(bCtx)
				timeMod := rootScope.Parent.FindChild("time")
				MustSucceed(rootScope.Add(bCtx, symbol.Symbol{
					Name: "t", Kind: symbol.KindModuleAlias, Target: timeMod,
				}))
				resolved := MustSucceed(resolveQualified(bCtx, rootScope, "t", "now"))
				Expect(resolved.Name).To(Equal("now"))
			})

			It("Should mark the alias as used on head resolution", func(bCtx SpecContext) {
				rootScope := buildAmbientRoot(bCtx)
				timeMod := rootScope.Parent.FindChild("time")
				alias := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{
					Name: "t", Kind: symbol.KindModuleAlias, Target: timeMod,
				}))
				MustSucceed(rootScope.Resolve(bCtx, "t"))
				Expect(alias.Used).To(BeTrue())
			})

			It("Should leave the alias unused when usage tracking is suppressed", func(bCtx SpecContext) {
				rootScope := buildAmbientRoot(bCtx)
				timeMod := rootScope.Parent.FindChild("time")
				alias := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{
					Name: "t", Kind: symbol.KindModuleAlias, Target: timeMod,
				}))
				MustSucceed(rootScope.Resolve(bCtx, "t", symbol.WithoutUsageTracking))
				Expect(alias.Used).To(BeFalse())
			})

			It("Should not find members past the module seal", func(bCtx SpecContext) {
				rootScope := buildAmbientRoot(bCtx)
				timeMod := rootScope.Parent.FindChild("time")
				MustSucceed(rootScope.Add(bCtx, symbol.Symbol{
					Name: "time", Kind: symbol.KindModuleAlias, Target: timeMod,
				}))
				// A bare "outer" exists in the user root but `time.outer` must
				// not find it — module member lookup is sealed.
				MustSucceed(rootScope.Add(bCtx, symbol.Symbol{
					Name: "outer", Kind: symbol.KindFunction, Type: types.F64(),
				}))
				Expect(resolveQualified(bCtx, rootScope, "time", "outer")).Error().
					To(MatchError(ContainSubstring("undefined symbol")))
			})
		})
	})

	Describe("Search", func() {
		It("Should resolve symbols from children", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			fooScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(fooScope).ToNot(BeNil())
			foobarScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "foobar", Kind: symbol.KindVariable, Type: types.I64()}))
			Expect(foobarScope).ToNot(BeNil())
			barScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "bar", Kind: symbol.KindVariable, Type: types.F32()}))
			Expect(barScope).ToNot(BeNil())
			scopes := MustSucceed(rootScope.Search(bCtx, "foo"))
			Expect(scopes).To(HaveLen(2))
			names := []string{scopes[0].Name, scopes[1].Name}
			Expect(names).To(ContainElements("foo", "foobar"))
		})
		It("Should resolve symbols from global resolver", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "pi", Kind: symbol.KindInput, Type: types.F64()},
				{Name: "print", Kind: symbol.KindFunction},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			scopes := MustSucceed(rootScope.Search(bCtx, "p"))
			Expect(scopes).To(HaveLen(2))
			names := []string{scopes[0].Name, scopes[1].Name}
			Expect(names).To(ContainElements("pi", "print"))
		})
		It("Should resolve symbols from parent scope", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			globalScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "global", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(globalScope).ToNot(BeNil())
			globalTwoScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "globalTwo", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(globalTwoScope).ToNot(BeNil())
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			scopes := MustSucceed(funcScope.Search(bCtx, "global"))
			Expect(scopes).To(HaveLen(2))
			names := []string{scopes[0].Name, scopes[1].Name}
			Expect(names).To(ContainElements("global", "globalTwo"))
		})
		It("Should deduplicate symbols across all sources", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "x", Kind: symbol.KindInput, Type: types.F64()},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			rootX := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(rootX).ToNot(BeNil())
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			funcX := MustSucceed(funcScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I64()}))
			Expect(funcX).ToNot(BeNil())
			scopes := MustSucceed(funcScope.Search(bCtx, "x"))
			Expect(scopes).To(HaveLen(1))
			Expect(scopes[0].Type).To(Equal(types.I64()))
		})
		It("Should return empty slice for non-matching prefix", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
			scopes := MustSucceed(rootScope.Search(bCtx, "xyz"))
			Expect(scopes).To(BeEmpty())
		})
		It("Should return all symbols for empty prefix", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			fooScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(fooScope).ToNot(BeNil())
			barScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "bar", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(barScope).ToNot(BeNil())
			scopes := MustSucceed(rootScope.Search(bCtx, ""))
			Expect(scopes).To(HaveLen(2))
		})
		It("Should skip internal symbols from global resolver", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "element_add", Kind: symbol.KindFunction, Type: types.F64(), Internal: true},
				{Name: "len", Kind: symbol.KindFunction, Type: types.F64()},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			scopes := MustSucceed(rootScope.Search(bCtx, ""))
			Expect(scopes).To(HaveLen(1))
			Expect(scopes[0].Name).To(Equal("len"))
		})
		It("Should skip all internal symbols when searching by prefix", func(bCtx SpecContext) {
			globalResolver := StaticResolver{
				{Name: "element_add", Kind: symbol.KindFunction, Type: types.F64(), Internal: true},
				{Name: "element_sub", Kind: symbol.KindFunction, Type: types.F64(), Internal: true},
				{Name: "element_len", Kind: symbol.KindFunction, Type: types.F64()},
			}
			rootScope := symbol.NewRoot(globalResolver, nil)
			scopes := MustSucceed(rootScope.Search(bCtx, "element"))
			Expect(scopes).To(HaveLen(1))
			Expect(scopes[0].Name).To(Equal("element_len"))
		})
	})

	Describe("ClosestAncestorOfKind", func() {
		It("Should find closest ancestor of kind", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			blockScope := MustSucceed(funcScope.Add(bCtx, symbol.Symbol{Name: "block", Kind: symbol.KindBlock}))
			varScope := MustSucceed(blockScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			ancestor := MustSucceed(varScope.ClosestAncestorOfKind(symbol.KindFunction))
			Expect(ancestor).To(Equal(funcScope))
		})

		It("Should return self if matching kind", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			ancestor := MustSucceed(funcScope.ClosestAncestorOfKind(symbol.KindFunction))
			Expect(ancestor).To(Equal(funcScope))
		})

		It("Should return error when no ancestor found", func() {
			rootScope := symbol.NewRoot(nil, nil)
			Expect(rootScope.ClosestAncestorOfKind(symbol.KindChannel)).Error().To(
				MatchError(ContainSubstring("undefined symbol")),
			)
		})
	})

	Describe("String", func() {
		It("Should format scope as string", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "myFunc", Kind: symbol.KindFunction}))
			varScope := MustSucceed(funcScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(varScope).ToNot(BeNil())
			str := funcScope.String()
			Expect(str).To(ContainSubstring("name: myFunc"))
			Expect(str).To(ContainSubstring("kind: KindFunction"))
			Expect(str).To(ContainSubstring("name: x"))
			Expect(str).To(ContainSubstring("kind: KindVariable"))
			Expect(str).To(ContainSubstring("type: i32"))
		})
	})

	Describe("FilterChildrenByKind", func() {
		It("Should filter children by kind", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			var1 := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			Expect(funcScope).ToNot(BeNil())
			var2 := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "y", Kind: symbol.KindVariable, Type: types.I64()}))
			filtered := rootScope.FilterChildrenByKind(symbol.KindVariable)
			Expect(filtered).To(HaveLen(2))
			Expect(filtered).To(ContainElements(var1, var2))
		})
		It("Should return empty when no matches", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
			filtered := rootScope.FilterChildrenByKind(symbol.KindChannel)
			Expect(filtered).To(BeEmpty())
		})
	})

	Describe("AutoName", func() {
		It("Should generate name with prefix and incremented index", func() {
			rootScope := symbol.NewRoot(nil, nil)
			child1 := &symbol.Symbol{Parent: rootScope, Kind: symbol.KindBlock}
			child1.AutoName("stage_")
			Expect(child1.Name).To(Equal("stage_0"))
			child2 := &symbol.Symbol{Parent: rootScope, Kind: symbol.KindBlock}
			child2.AutoName("stage_")
			Expect(child2.Name).To(Equal("stage_1"))
		})
	})

	Describe("Channels", func() {
		Describe("NewChannels", func() {
			It("Should create empty Channels with initialized maps", func() {
				ch := types.NewChannels()
				Expect(ch.Read).ToNot(BeNil())
				Expect(ch.Write).ToNot(BeNil())
				Expect(ch.Read).To(BeEmpty())
				Expect(ch.Write).To(BeEmpty())
			})
		})
		Describe("Copy", func() {
			It("Should create deep copy of Channels", func() {
				ch := types.NewChannels()
				ch.Read[1] = "channel1"
				ch.Write[2] = "channel2"
				copied := ch.Copy()
				Expect(copied.Read).To(HaveLen(1))
				Expect(copied.Write).To(HaveLen(1))
				Expect(copied.Read[1]).To(Equal("channel1"))
				Expect(copied.Write[2]).To(Equal("channel2"))
				ch.Read[3] = "channel3"
				Expect(copied.Read).ToNot(HaveKey(uint32(3)))
			})
		})
		Describe("ResolveInputChannel", func() {
			It("Should fall back to Read when fnSym has no children (built-in functions)", func() {
				fnSym := &symbol.Symbol{Name: "on", Kind: symbol.KindFunction}
				nodeChannels := types.NewChannels()
				symbol.ResolveInputChannel(&nodeChannels, fnSym, "channel", 42, "my_sensor")
				Expect(nodeChannels.Read).To(HaveLen(1))
				Expect(nodeChannels.Read[42]).To(Equal("my_sensor"))
				Expect(nodeChannels.Write).To(BeEmpty())
			})

			It("Should replace internal Read ID with actual channel ID for user-defined functions", func(bCtx SpecContext) {
				fnSym := symbol.NewRoot(nil, nil)
				fnSym.Kind = symbol.KindFunction
				fnSym.Channels = types.NewChannels()
				inputParam := MustSucceed(fnSym.Add(bCtx, symbol.Symbol{
					Name: "channel",
					Kind: symbol.KindInput,
					Type: types.Chan(types.F64()),
				}))
				internalID := uint32(inputParam.ID)
				fnSym.Channels.Read[internalID] = "channel"
				nodeChannels := fnSym.Channels.Copy()
				Expect(nodeChannels.Read).To(HaveKey(internalID))
				symbol.ResolveInputChannel(&nodeChannels, fnSym, "channel", 42, "my_sensor")
				Expect(nodeChannels.Read).ToNot(HaveKey(internalID))
				Expect(nodeChannels.Read).To(HaveLen(1))
				Expect(nodeChannels.Read[42]).To(Equal("my_sensor"))
			})

			It("Should replace internal Write ID with actual channel ID for user-defined functions", func(bCtx SpecContext) {
				fnSym := symbol.NewRoot(nil, nil)
				fnSym.Kind = symbol.KindFunction
				fnSym.Channels = types.NewChannels()
				inputParam := MustSucceed(fnSym.Add(bCtx, symbol.Symbol{
					Name: "channel",
					Kind: symbol.KindInput,
					Type: types.Chan(types.F64()),
				}))
				internalID := uint32(inputParam.ID)
				fnSym.Channels.Write[internalID] = "channel"
				nodeChannels := fnSym.Channels.Copy()
				Expect(nodeChannels.Write).To(HaveKey(internalID))
				symbol.ResolveInputChannel(&nodeChannels, fnSym, "channel", 55, "output_channel")
				Expect(nodeChannels.Write).ToNot(HaveKey(internalID))
				Expect(nodeChannels.Write).To(HaveLen(1))
				Expect(nodeChannels.Write[55]).To(Equal("output_channel"))
				Expect(nodeChannels.Read).To(BeEmpty())
			})

			It("Should handle param that is both read and written", func(bCtx SpecContext) {
				fnSym := symbol.NewRoot(nil, nil)
				fnSym.Kind = symbol.KindFunction
				fnSym.Channels = types.NewChannels()
				inputParam := MustSucceed(fnSym.Add(bCtx, symbol.Symbol{
					Name: "channel",
					Kind: symbol.KindInput,
					Type: types.Chan(types.F64()),
				}))
				internalID := uint32(inputParam.ID)
				fnSym.Channels.Read[internalID] = "channel"
				fnSym.Channels.Write[internalID] = "channel"
				nodeChannels := fnSym.Channels.Copy()
				symbol.ResolveInputChannel(&nodeChannels, fnSym, "channel", 100, "bidirectional_channel")
				Expect(nodeChannels.Read).To(HaveLen(1))
				Expect(nodeChannels.Read[100]).To(Equal("bidirectional_channel"))
				Expect(nodeChannels.Write).To(HaveLen(1))
				Expect(nodeChannels.Write[100]).To(Equal("bidirectional_channel"))
			})

			It("Should use WriteChan access for built-in with WriteChan input param", func() {
				fnSym := &symbol.Symbol{
					Name: "set_authority",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{
						Inputs: types.Params{
							{Name: "channel", Type: types.WriteChan(types.U8())},
						},
					}),
				}
				nodeChannels := types.NewChannels()
				symbol.ResolveInputChannel(&nodeChannels, fnSym, "channel", 42, "valve")
				Expect(nodeChannels.Write).To(HaveLen(1))
				Expect(nodeChannels.Write[42]).To(Equal("valve"))
				Expect(nodeChannels.Read).To(BeEmpty())
			})

			It("Should use ReadChan access for built-in with ReadChan input param", func() {
				fnSym := &symbol.Symbol{
					Name: "on",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{
						Inputs: types.Params{
							{Name: "channel", Type: types.ReadChan(types.F64())},
						},
					}),
				}
				nodeChannels := types.NewChannels()
				symbol.ResolveInputChannel(&nodeChannels, fnSym, "channel", 10, "sensor")
				Expect(nodeChannels.Read).To(HaveLen(1))
				Expect(nodeChannels.Read[10]).To(Equal("sensor"))
				Expect(nodeChannels.Write).To(BeEmpty())
			})

			It("Should default to Read for built-in with plain Chan input param", func() {
				fnSym := &symbol.Symbol{
					Name: "custom",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{
						Inputs: types.Params{
							{Name: "channel", Type: types.Chan(types.F64())},
						},
					}),
				}
				nodeChannels := types.NewChannels()
				symbol.ResolveInputChannel(&nodeChannels, fnSym, "channel", 99, "ch")
				Expect(nodeChannels.Read).To(HaveLen(1))
				Expect(nodeChannels.Read[99]).To(Equal("ch"))
				Expect(nodeChannels.Write).To(BeEmpty())
			})
		})
	})
})

var _ = Describe("Name conflicts", func() {
	// A conflict message names the kind of the symbol already holding the name,
	// exercising nounForKind for each kind.
	DescribeTable("Should name the colliding symbol's kind",
		func(bCtx SpecContext, kind symbol.Kind, noun string) {
			ast := MustSucceed(parser.ParseStatement("x := 1"))
			root := symbol.NewRoot(nil, nil)
			MustSucceed(root.Add(bCtx, symbol.Symbol{
				Name: "dup", Kind: kind, Type: types.I32(), AST: ast,
			}))
			_, err := root.Add(bCtx, symbol.Symbol{
				Name: "dup", Kind: symbol.KindVariable, Type: types.I32(), AST: ast,
			})
			Expect(err).To(MatchError(ContainSubstring("conflicts with existing " + noun)))
		},
		Entry("variable", symbol.KindVariable, "variable"),
		Entry("stateful variable", symbol.KindStatefulVariable, "variable"),
		Entry("channel", symbol.KindChannel, "channel"),
		Entry("function", symbol.KindFunction, "function"),
		Entry("input parameter", symbol.KindInput, "input parameter"),
		Entry("output parameter", symbol.KindOutput, "output parameter"),
		Entry("sequence", symbol.KindSequence, "sequence"),
		Entry("stage", symbol.KindStage, "stage"),
		Entry("constant", symbol.KindConstant, "constant"),
		Entry("module alias", symbol.KindModuleAlias, "import"),
		Entry("block falls back to symbol", symbol.KindBlock, "symbol"),
	)
})
