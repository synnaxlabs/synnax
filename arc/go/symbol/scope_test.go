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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Scope", func() {
	Describe("CreateRootScope", func() {
		It("Should create a new root scope", func() {
			s := symbol.CreateRootScope(nil)
			Expect(s.GlobalResolver).To(BeNil())
			Expect(s.Children).To(BeEmpty())
			Expect(s.Counter).ToNot(BeNil())
			Expect(*s.Counter).To(Equal(0))
		})

		It("Should create a new root scope with a global resolver", func() {
			s := symbol.CreateRootScope(symbol.MapResolver{})
			Expect(s.GlobalResolver).ToNot(BeNil())
		})
	})

	Describe("Add", func() {
		It("Should add a new variable scope", func() {
			rootScope := symbol.CreateRootScope(nil)
			varScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(varScope.Name).To(Equal("x"))
			Expect(varScope.Type).To(Equal(types.I32()))
			By("Not creating a counter for variables")
			Expect(varScope.Counter).To(BeNil())
			Expect(varScope.GlobalResolver).To(BeNil())
			Expect(varScope.Children).To(BeEmpty())
			By("Using the root scope counter for it's ID")
			Expect(varScope.ID).To(Equal(0))
		})

		It("Should add a new function scope", func() {
			rootScope := symbol.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "my_func", Kind: symbol.KindFunction},
			))
			Expect(funcScope.Name).To(Equal("my_func"))
			By("Creating a counter for functions")
			Expect(funcScope.Counter).ToNot(BeNil())
			Expect(*funcScope.Counter).To(Equal(0))
		})

		It("Should add a new func scope", func() {
			rootScope := symbol.CreateRootScope(nil)
			stageScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "my_task", Kind: symbol.KindBlock},
			))
			Expect(stageScope.Name).To(Equal("my_task"))
		})

		DescribeTable("Should assign IDs to variable kinds",
			func(kind symbol.Kind) {
				rootScope := symbol.CreateRootScope(nil)
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
			Entry("KindStatefulVariable", symbol.KindStatefulVariable),
			Entry("Input", symbol.KindInput),
		)

		It("Should correctly increment IDs for variables within function scopes", func() {
			rootScope := symbol.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "my_func", Kind: symbol.KindFunction},
			))
			firstVarScope := MustSucceed(funcScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(firstVarScope.ID).To(Equal(0))
			Expect(firstVarScope.Counter).To(BeNil())
			Expect(firstVarScope.Parent).ToNot(BeNil())
			Expect(firstVarScope.Parent).To(Equal(funcScope))
			secondVarScope := MustSucceed(funcScope.Add(
				bCtx,
				symbol.Symbol{Name: "y", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(secondVarScope.ID).To(Equal(1))
			Expect(secondVarScope.Counter).To(BeNil())
			Expect(secondVarScope.Parent).ToNot(BeNil())
			Expect(secondVarScope.Parent).To(Equal(funcScope))
		})

		It("Should not return error when adding duplicate symbol that shadows a global", func() {
			rootScope := symbol.CreateRootScope(nil)
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
		It("Should allow shadowing global symbols from resolver", func() {
			globalResolver := symbol.MapResolver{
				"x": symbol.Symbol{Name: "x", Kind: symbol.KindConfig, Type: types.F64()},
			}
			rootScope := symbol.CreateRootScope(globalResolver)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
		})
		It("Should allow symbols with empty names", func() {
			rootScope := symbol.CreateRootScope(nil)
			child := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "", Kind: symbol.KindBlock}))
			Expect(child.Name).To(Equal(""))
		})
	})

	Describe("GetChildByParserRule", func() {
		It("Should find child by parser rule", func() {
			rootScope := symbol.CreateRootScope(nil)
			rule := antlr.NewBaseParserRuleContext(nil, 0)
			child := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32(), AST: rule},
			))
			found := MustSucceed(rootScope.GetChildByParserRule(rule))
			Expect(found).To(Equal(child))
		})

		It("Should return error when parser rule not found", func() {
			rootScope := symbol.CreateRootScope(nil)
			scope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(scope).ToNot(BeNil())
			Expect(rootScope.GetChildByParserRule(antlr.NewBaseParserRuleContext(nil, 0))).Error().To(MatchError(ContainSubstring("could not find symbol matching parser rule")))
		})
	})

	Describe("FindChildByName", func() {
		It("Should find child by name", func() {
			rootScope := symbol.CreateRootScope(nil)
			child := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			found := rootScope.FindChildByName("x")
			Expect(found).To(Equal(child))
		})

		It("Should return nil when name not found", func() {
			rootScope := symbol.CreateRootScope(nil)
			found := rootScope.FindChildByName("nonexistent")
			Expect(found).To(BeNil())
		})
	})

	Describe("FindChild", func() {
		It("Should find child matching predicate", func() {
			rootScope := symbol.CreateRootScope(nil)
			scope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(scope).ToNot(BeNil())
			child := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "y", Kind: symbol.KindInput, Type: types.I64()},
			))
			found := rootScope.FindChild(func(s *symbol.Scope) bool {
				return s.Kind == symbol.KindInput
			})
			Expect(found).To(Equal(child))
		})

		It("Should return nil when no match", func() {
			rootScope := symbol.CreateRootScope(nil)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
			found := rootScope.FindChild(func(s *symbol.Scope) bool {
				return s.Kind == symbol.KindFunction
			})
			Expect(found).To(BeNil())
		})
	})

	Describe("FilterChildren", func() {
		It("Should filter children by predicate", func() {
			rootScope := symbol.CreateRootScope(nil)
			var1 := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			Expect(funcScope).ToNot(BeNil())
			var2 := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "y", Kind: symbol.KindVariable, Type: types.I64()}))
			filtered := rootScope.FilterChildren(func(s *symbol.Scope) bool {
				return s.Kind == symbol.KindVariable
			})
			Expect(filtered).To(HaveLen(2))
			Expect(filtered).To(ContainElements(var1, var2))
		})

		It("Should return empty when no matches", func() {
			rootScope := symbol.CreateRootScope(nil)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
			filtered := rootScope.FilterChildren(func(s *symbol.Scope) bool {
				return s.Kind == symbol.KindChannel
			})
			Expect(filtered).To(BeEmpty())
		})
	})

	Describe("Root", func() {
		It("Should return root scope from any depth", func() {
			rootScope := symbol.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			varScope := MustSucceed(funcScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(varScope.Root()).To(Equal(rootScope))
			Expect(funcScope.Root()).To(Equal(rootScope))
			Expect(rootScope.Root()).To(Equal(rootScope))
		})
	})

	Describe("Resolve", func() {
		It("Should resolve symbol in current scope", func() {
			rootScope := symbol.CreateRootScope(nil)
			child := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			resolved := MustSucceed(rootScope.Resolve(bCtx, "x"))
			Expect(resolved).To(Equal(child))
		})
		It("Should resolve symbol from parent scope", func() {
			rootScope := symbol.CreateRootScope(nil)
			global := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "global", Kind: symbol.KindVariable, Type: types.I32()}))
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			resolved := MustSucceed(funcScope.Resolve(bCtx, "global"))
			Expect(resolved).To(Equal(global))
		})
		It("Should resolve from global resolver", func() {
			globalResolver := symbol.MapResolver{
				"pi": symbol.Symbol{Name: "pi", Kind: symbol.KindConfig, Type: types.F64()},
			}
			rootScope := symbol.CreateRootScope(globalResolver)
			resolved := MustSucceed(rootScope.Resolve(bCtx, "pi"))
			Expect(resolved.Name).To(Equal("pi"))
			Expect(resolved.Kind).To(Equal(symbol.KindConfig))
		})
		It("Should prioritize local over parent scope", func() {
			rootScope := symbol.CreateRootScope(nil)
			rootX := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			resolvedFromFunc := MustSucceed(funcScope.Resolve(bCtx, "x"))
			Expect(resolvedFromFunc).To(Equal(rootX))
		})
		It("Should return error for undefined symbol", func() {
			rootScope := symbol.CreateRootScope(nil)
			_, err := rootScope.Resolve(bCtx, "undefined")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("undefined symbol: undefined")))
		})
	})

	Describe("Search", func() {
		It("Should resolve symbols from children", func() {
			rootScope := symbol.CreateRootScope(nil)
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
		It("Should resolve symbols from global resolver", func() {
			globalResolver := symbol.MapResolver{
				"pi":    symbol.Symbol{Name: "pi", Kind: symbol.KindConfig, Type: types.F64()},
				"print": symbol.Symbol{Name: "print", Kind: symbol.KindFunction},
			}
			rootScope := symbol.CreateRootScope(globalResolver)
			scopes := MustSucceed(rootScope.Search(bCtx, "p"))
			Expect(scopes).To(HaveLen(2))
			names := []string{scopes[0].Name, scopes[1].Name}
			Expect(names).To(ContainElements("pi", "print"))
		})
		It("Should resolve symbols from parent scope", func() {
			rootScope := symbol.CreateRootScope(nil)
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
		It("Should deduplicate symbols across all sources", func() {
			globalResolver := symbol.MapResolver{
				"x": symbol.Symbol{Name: "x", Kind: symbol.KindConfig, Type: types.F64()},
			}
			rootScope := symbol.CreateRootScope(globalResolver)
			rootX := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(rootX).ToNot(BeNil())
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			funcX := MustSucceed(funcScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I64()}))
			Expect(funcX).ToNot(BeNil())
			scopes := MustSucceed(funcScope.Search(bCtx, "x"))
			Expect(scopes).To(HaveLen(1))
			Expect(scopes[0].Type).To(Equal(types.I64()))
		})
		It("Should return empty slice for non-matching prefix", func() {
			rootScope := symbol.CreateRootScope(nil)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
			scopes := MustSucceed(rootScope.Search(bCtx, "xyz"))
			Expect(scopes).To(BeEmpty())
		})
		It("Should return all symbols for empty prefix", func() {
			rootScope := symbol.CreateRootScope(nil)
			fooScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "foo", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(fooScope).ToNot(BeNil())
			barScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "bar", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(barScope).ToNot(BeNil())
			scopes := MustSucceed(rootScope.Search(bCtx, ""))
			Expect(scopes).To(HaveLen(2))
		})
	})

	Describe("ClosestAncestorOfKind", func() {
		It("Should find closest ancestor of kind", func() {
			rootScope := symbol.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			blockScope := MustSucceed(funcScope.Add(bCtx, symbol.Symbol{Name: "block", Kind: symbol.KindBlock}))
			varScope := MustSucceed(blockScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			ancestor := MustSucceed(varScope.ClosestAncestorOfKind(symbol.KindFunction))
			Expect(ancestor).To(Equal(funcScope))
		})

		It("Should return self if matching kind", func() {
			rootScope := symbol.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			ancestor := MustSucceed(funcScope.ClosestAncestorOfKind(symbol.KindFunction))
			Expect(ancestor).To(Equal(funcScope))
		})

		It("Should return error when no ancestor found", func() {
			rootScope := symbol.CreateRootScope(nil)
			_, err := rootScope.ClosestAncestorOfKind(symbol.KindChannel)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("undefined symbol")))
		})
	})

	Describe("FirstChildOfKind", func() {
		It("Should find first child of kind", func() {
			rootScope := symbol.CreateRootScope(nil)
			varScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(varScope).ToNot(BeNil())
			funcChild := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			funcG := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "g", Kind: symbol.KindFunction}))
			Expect(funcG).ToNot(BeNil())
			first := MustSucceed(rootScope.FirstChildOfKind(symbol.KindFunction))
			Expect(first).To(Equal(funcChild))
		})

		It("Should return error when no child of kind", func() {
			rootScope := symbol.CreateRootScope(nil)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
			_, err := rootScope.FirstChildOfKind(symbol.KindChannel)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("undefined symbol")))
		})
	})

	Describe("String", func() {
		It("Should format scope as string", func() {
			rootScope := symbol.CreateRootScope(nil)
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
		It("Should filter children by kind", func() {
			rootScope := symbol.CreateRootScope(nil)
			var1 := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			funcScope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "f", Kind: symbol.KindFunction}))
			Expect(funcScope).ToNot(BeNil())
			var2 := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "y", Kind: symbol.KindVariable, Type: types.I64()}))
			filtered := rootScope.FilterChildrenByKind(symbol.KindVariable)
			Expect(filtered).To(HaveLen(2))
			Expect(filtered).To(ContainElements(var1, var2))
		})
		It("Should return empty when no matches", func() {
			rootScope := symbol.CreateRootScope(nil)
			scope := MustSucceed(rootScope.Add(bCtx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
			filtered := rootScope.FilterChildrenByKind(symbol.KindChannel)
			Expect(filtered).To(BeEmpty())
		})
	})

	Describe("AutoName", func() {
		It("Should generate name with prefix and incremented index", func() {
			rootScope := symbol.CreateRootScope(nil)
			child1 := &symbol.Scope{Parent: rootScope, Symbol: symbol.Symbol{Kind: symbol.KindBlock}}
			child1.AutoName("stage_")
			Expect(child1.Name).To(Equal("stage_0"))
			child2 := &symbol.Scope{Parent: rootScope, Symbol: symbol.Symbol{Kind: symbol.KindBlock}}
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
				Expect(ch.Read).To(HaveLen(0))
				Expect(ch.Write).To(HaveLen(0))
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
	})
})
