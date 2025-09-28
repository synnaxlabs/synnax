// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Scope", func() {
	Describe("CreateRootScope", func() {
		It("Should create a new root scope", func() {
			s := ir.CreateRootScope(nil)
			Expect(s.GlobalResolver).To(BeNil())
			Expect(s.Children).To(BeEmpty())
			Expect(s.Counter).ToNot(BeNil())
			Expect(*s.Counter).To(Equal(0))
		})

		It("Should create a new root scope with a global resolver", func() {
			s := ir.CreateRootScope(ir.MapResolver{})
			Expect(s.GlobalResolver).ToNot(BeNil())
		})
	})

	Describe("Add", func() {
		It("Should add a new variable scope", func() {
			rootScope := ir.CreateRootScope(nil)
			varScope := MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}},
			))
			Expect(varScope.Name).To(Equal("x"))
			Expect(varScope.Type).To(Equal(ir.I32{}))
			By("Not creating a counter for variables")
			Expect(varScope.Counter).To(BeNil())
			Expect(varScope.GlobalResolver).To(BeNil())
			Expect(varScope.Children).To(BeEmpty())
			By("Using the root scope counter for it's ID")
			Expect(varScope.ID).To(Equal(0))
		})

		It("Should add a new function scope", func() {
			rootScope := ir.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "my_func", Kind: ir.KindFunction, Type: ir.Function{}},
			))
			Expect(funcScope.Name).To(Equal("my_func"))
			Expect(funcScope.Type).To(Equal(ir.Function{}))
			By("Creating a counter for functions")
			Expect(funcScope.Counter).ToNot(BeNil())
			Expect(*funcScope.Counter).To(Equal(0))
		})

		It("Should add a new stage scope", func() {
			rootScope := ir.CreateRootScope(nil)
			stageScope := MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "my_task", Kind: ir.KindStage, Type: ir.Function{}},
			))
			Expect(stageScope.Name).To(Equal("my_task"))
			By("Creating a counter for tasks")
			Expect(stageScope.Counter).ToNot(BeNil())
			Expect(*stageScope.Counter).To(Equal(0))
		})

		DescribeTable("Should assign IDs to variable kinds",
			func(kind ir.Kind) {
				rootScope := ir.CreateRootScope(nil)
				scope1 := MustSucceed(rootScope.Add(
					ctx,
					ir.Symbol{Name: "var1", Kind: kind, Type: ir.I32{}},
				))
				scope2 := MustSucceed(rootScope.Add(
					ctx,
					ir.Symbol{Name: "var2", Kind: kind, Type: ir.I32{}},
				))
				Expect(scope1.ID).To(Equal(0))
				Expect(scope2.ID).To(Equal(1))
			},
			Entry("Variable", ir.KindVariable),
			Entry("StatefulVariable", ir.KindStatefulVariable),
			Entry("Param", ir.KindParam),
		)

		It("Should correctly increment IDs for variables within function scopes", func() {
			rootScope := ir.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "my_func", Kind: ir.KindFunction, Type: ir.Function{}},
			))
			firstVarScope := MustSucceed(funcScope.Add(
				ctx,
				ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}},
			))
			Expect(firstVarScope.ID).To(Equal(0))
			Expect(firstVarScope.Counter).To(BeNil())
			Expect(firstVarScope.Parent).ToNot(BeNil())
			Expect(firstVarScope.Parent).To(Equal(funcScope))
			secondVarScope := MustSucceed(funcScope.Add(
				ctx,
				ir.Symbol{Name: "y", Kind: ir.KindVariable, Type: ir.I32{}},
			))
			Expect(secondVarScope.ID).To(Equal(1))
			Expect(secondVarScope.Counter).To(BeNil())
			Expect(secondVarScope.Parent).ToNot(BeNil())
			Expect(secondVarScope.Parent).To(Equal(funcScope))
		})

		It("Should return error when adding duplicate symbol", func() {
			rootScope := ir.CreateRootScope(nil)
			MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}},
			))
			_, err := rootScope.Add(
				ctx,
				ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I64{}},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("conflicts with existing symbol"))
		})
	})

	Describe("GetChildByParserRule", func() {
		It("Should find child by parser rule", func() {
			rootScope := ir.CreateRootScope(nil)
			rule := antlr.NewBaseParserRuleContext(nil, 0)
			child := MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}, ParserRule: rule},
			))
			found := MustSucceed(rootScope.GetChildByParserRule(rule))
			Expect(found).To(Equal(child))
		})

		It("Should return error when parser rule not found", func() {
			rootScope := ir.CreateRootScope(nil)
			MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}},
			))
			_, err := rootScope.GetChildByParserRule(antlr.NewBaseParserRuleContext(nil, 0))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not find symbol matching parser rule"))
		})
	})

	Describe("FindChildByName", func() {
		It("Should find child by name", func() {
			rootScope := ir.CreateRootScope(nil)
			child := MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}},
			))
			found := rootScope.FindChildByName("x")
			Expect(found).To(Equal(child))
		})

		It("Should return nil when name not found", func() {
			rootScope := ir.CreateRootScope(nil)
			found := rootScope.FindChildByName("nonexistent")
			Expect(found).To(BeNil())
		})
	})

	Describe("FindChild", func() {
		It("Should find child matching predicate", func() {
			rootScope := ir.CreateRootScope(nil)
			MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}},
			))
			child := MustSucceed(rootScope.Add(
				ctx,
				ir.Symbol{Name: "y", Kind: ir.KindParam, Type: ir.I64{}},
			))
			found := rootScope.FindChild(func(s *ir.Scope) bool {
				return s.Kind == ir.KindParam
			})
			Expect(found).To(Equal(child))
		})

		It("Should return nil when no match", func() {
			rootScope := ir.CreateRootScope(nil)
			MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			found := rootScope.FindChild(func(s *ir.Scope) bool {
				return s.Kind == ir.KindFunction
			})
			Expect(found).To(BeNil())
		})
	})

	Describe("FilterChildren", func() {
		It("Should filter children by predicate", func() {
			rootScope := ir.CreateRootScope(nil)
			var1 := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "f", Kind: ir.KindFunction, Type: ir.Function{}}))
			var2 := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "y", Kind: ir.KindVariable, Type: ir.I64{}}))
			filtered := rootScope.FilterChildren(func(s *ir.Scope) bool {
				return s.Kind == ir.KindVariable
			})
			Expect(filtered).To(HaveLen(2))
			Expect(filtered).To(ContainElements(var1, var2))
		})

		It("Should return empty when no matches", func() {
			rootScope := ir.CreateRootScope(nil)
			MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			filtered := rootScope.FilterChildren(func(s *ir.Scope) bool {
				return s.Kind == ir.KindStage
			})
			Expect(filtered).To(BeEmpty())
		})
	})

	Describe("Root", func() {
		It("Should return root scope from any depth", func() {
			rootScope := ir.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "f", Kind: ir.KindFunction, Type: ir.Function{}}))
			varScope := MustSucceed(funcScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			Expect(varScope.Root()).To(Equal(rootScope))
			Expect(funcScope.Root()).To(Equal(rootScope))
			Expect(rootScope.Root()).To(Equal(rootScope))
		})
	})

	Describe("Resolve", func() {
		It("Should resolve symbol in current scope", func() {
			rootScope := ir.CreateRootScope(nil)
			child := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			resolved := MustSucceed(rootScope.Resolve(ctx, "x"))
			Expect(resolved).To(Equal(child))
		})

		It("Should resolve symbol from parent scope", func() {
			rootScope := ir.CreateRootScope(nil)
			global := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "global", Kind: ir.KindVariable, Type: ir.I32{}}))
			funcScope := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "f", Kind: ir.KindFunction, Type: ir.Function{}}))
			resolved := MustSucceed(funcScope.Resolve(ctx, "global"))
			Expect(resolved).To(Equal(global))
		})

		It("Should resolve from global resolver", func() {
			globalResolver := ir.MapResolver{
				"pi": ir.Symbol{Name: "pi", Kind: ir.KindConfigParam, Type: ir.F64{}},
			}
			rootScope := ir.CreateRootScope(globalResolver)
			resolved := MustSucceed(rootScope.Resolve(ctx, "pi"))
			Expect(resolved.Name).To(Equal("pi"))
			Expect(resolved.Kind).To(Equal(ir.KindConfigParam))
		})

		It("Should prioritize local over parent scope", func() {
			rootScope := ir.CreateRootScope(nil)
			rootX := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			funcScope := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "f", Kind: ir.KindFunction, Type: ir.Function{}}))
			resolvedFromFunc := MustSucceed(funcScope.Resolve(ctx, "x"))
			Expect(resolvedFromFunc).To(Equal(rootX))
		})

		It("Should return error for undefined symbol", func() {
			rootScope := ir.CreateRootScope(nil)
			_, err := rootScope.Resolve(ctx, "undefined")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("undefined symbol: undefined"))
		})
	})

	Describe("ClosestAncestorOfKind", func() {
		It("Should find closest ancestor of kind", func() {
			rootScope := ir.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "f", Kind: ir.KindFunction, Type: ir.Function{}}))
			blockScope := MustSucceed(funcScope.Add(ctx, ir.Symbol{Name: "block", Kind: ir.KindBlock}))
			varScope := MustSucceed(blockScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			ancestor := MustSucceed(varScope.ClosestAncestorOfKind(ir.KindFunction))
			Expect(ancestor).To(Equal(funcScope))
		})

		It("Should return self if matching kind", func() {
			rootScope := ir.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "f", Kind: ir.KindFunction, Type: ir.Function{}}))
			ancestor := MustSucceed(funcScope.ClosestAncestorOfKind(ir.KindFunction))
			Expect(ancestor).To(Equal(funcScope))
		})

		It("Should return error when no ancestor found", func() {
			rootScope := ir.CreateRootScope(nil)
			_, err := rootScope.ClosestAncestorOfKind(ir.KindStage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("undefined symbol"))
		})
	})

	Describe("FirstChildOfKind", func() {
		It("Should find first child of kind", func() {
			rootScope := ir.CreateRootScope(nil)
			MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			funcChild := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "f", Kind: ir.KindFunction, Type: ir.Function{}}))
			MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "g", Kind: ir.KindFunction, Type: ir.Function{}}))
			first := MustSucceed(rootScope.FirstChildOfKind(ir.KindFunction))
			Expect(first).To(Equal(funcChild))
		})

		It("Should return error when no child of kind", func() {
			rootScope := ir.CreateRootScope(nil)
			MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			_, err := rootScope.FirstChildOfKind(ir.KindStage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("undefined symbol"))
		})
	})

	Describe("String", func() {
		It("Should format scope as string", func() {
			rootScope := ir.CreateRootScope(nil)
			funcScope := MustSucceed(rootScope.Add(ctx, ir.Symbol{Name: "myFunc", Kind: ir.KindFunction, Type: ir.Function{}}))
			MustSucceed(funcScope.Add(ctx, ir.Symbol{Name: "x", Kind: ir.KindVariable, Type: ir.I32{}}))
			str := funcScope.String()
			Expect(str).To(ContainSubstring("name: myFunc"))
			Expect(str).To(ContainSubstring("kind: KindFunction"))
			Expect(str).To(ContainSubstring("name: x"))
			Expect(str).To(ContainSubstring("kind: KindVariable"))
			Expect(str).To(ContainSubstring("type: i32"))
		})
	})
})
