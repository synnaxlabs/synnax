package symbol_test

import (
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Scope", func() {
	Describe("CreateRoot", func() {
		It("Should create a new root scope", func() {
			s := symbol.CreateRoot(nil)
			Expect(s.GlobalResolver).To(BeNil())
			Expect(s.Children).To(BeEmpty())
			Expect(s.Counter).ToNot(BeNil())
			Expect(*s.Counter).To(Equal(0))
		})

		It("Should create a new root scope with a global resolver", func() {
			s := symbol.CreateRoot(symbol.MapResolver{})
			Expect(s.GlobalResolver).ToNot(BeNil())
		})
	})

	Describe("Add", func() {
		It("Should add a new variable scope", func() {
			rootScope := symbol.CreateRoot(nil)
			varScope := MustSucceed(rootScope.Add(
				"x",
				symbol.KindVariable,
				types.I32{},
				nil,
			))
			Expect(varScope.Name).To(Equal("x"))
			Expect(varScope.Type).To(Equal(types.I32{}))
			By("Not creating a counter for variables")
			Expect(varScope.Counter).To(BeNil())
			Expect(varScope.GlobalResolver).To(BeNil())
			Expect(varScope.Children).To(BeEmpty())
			By("Using the root scope counter for it's ID")
			Expect(varScope.ID).To(Equal(0))
		})

		It("Should add a new function scope", func() {
			rootScope := symbol.CreateRoot(nil)
			funcScope := MustSucceed(rootScope.Add(
				"my_func",
				symbol.KindFunction,
				types.Function{},
				nil,
			))
			Expect(funcScope.Name).To(Equal("my_func"))
			Expect(funcScope.Type).To(Equal(types.Function{}))
			By("Creating a counter for functions")
			Expect(funcScope.Counter).ToNot(BeNil())
			Expect(*funcScope.Counter).To(Equal(0))
		})

		It("Should add a new task scope", func() {
			rootScope := symbol.CreateRoot(nil)
			taskScope := MustSucceed(rootScope.Add(
				"my_task",
				symbol.KindTask,
				types.Function{},
				nil,
			))
			Expect(taskScope.Name).To(Equal("my_task"))
			By("Creating a counter for tasks")
			Expect(taskScope.Counter).ToNot(BeNil())
			Expect(*taskScope.Counter).To(Equal(0))
		})

		DescribeTable("Should assign IDs to variable kinds",
			func(kind symbol.Kind) {
				rootScope := symbol.CreateRoot(nil)
				scope1 := MustSucceed(rootScope.Add("var1", kind, types.I32{}, nil))
				scope2 := MustSucceed(rootScope.Add("var2", kind, types.I32{}, nil))
				Expect(scope1.ID).To(Equal(0))
				Expect(scope2.ID).To(Equal(1))
			},
			Entry("Variable", symbol.KindVariable),
			Entry("StatefulVariable", symbol.KindStatefulVariable),
			Entry("Param", symbol.KindParam),
		)

		It("Should correctly increment IDs for variables within function scopes", func() {
			rootScope := symbol.CreateRoot(nil)
			funcScope := MustSucceed(rootScope.Add(
				"my_func",
				symbol.KindFunction,
				types.Function{},
				nil,
			))
			firstVarScope := MustSucceed(funcScope.Add(
				"x",
				symbol.KindVariable,
				types.I32{},
				nil,
			))
			Expect(firstVarScope.ID).To(Equal(0))
			Expect(firstVarScope.Counter).To(BeNil())
			Expect(firstVarScope.Parent).ToNot(BeNil())
			Expect(firstVarScope.Parent).To(Equal(funcScope))
			secondVarScope := MustSucceed(funcScope.Add(
				"y",
				symbol.KindVariable,
				types.I32{},
				nil,
			))
			Expect(secondVarScope.ID).To(Equal(1))
			Expect(secondVarScope.Counter).To(BeNil())
			Expect(secondVarScope.Parent).ToNot(BeNil())
			Expect(secondVarScope.Parent).To(Equal(funcScope))
		})

		It("Should return error when adding duplicate symbol", func() {
			rootScope := symbol.CreateRoot(nil)
			MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			_, err := rootScope.Add("x", symbol.KindVariable, types.I64{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("conflicts with existing symbol"))
		})
	})

	Describe("GetChildByParserRule", func() {
		It("Should find child by parser rule", func() {
			rootScope := symbol.CreateRoot(nil)
			rule := antlr.NewBaseParserRuleContext(nil, 0)
			child := MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, rule))
			found := MustSucceed(rootScope.GetChildByParserRule(rule))
			Expect(found).To(Equal(child))
		})

		It("Should return error when parser rule not found", func() {
			rootScope := symbol.CreateRoot(nil)
			MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			_, err := rootScope.GetChildByParserRule(antlr.NewBaseParserRuleContext(nil, 0))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not find symbol matching parser rule"))
		})
	})

	Describe("FindChildByName", func() {
		It("Should find child by name", func() {
			rootScope := symbol.CreateRoot(nil)
			child := MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			found := rootScope.FindChildByName("x")
			Expect(found).To(Equal(child))
		})

		It("Should return nil when name not found", func() {
			rootScope := symbol.CreateRoot(nil)
			found := rootScope.FindChildByName("nonexistent")
			Expect(found).To(BeNil())
		})
	})

	Describe("FindChild", func() {
		It("Should find child matching predicate", func() {
			rootScope := symbol.CreateRoot(nil)
			MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			child := MustSucceed(rootScope.Add("y", symbol.KindParam, types.I64{}, nil))
			found := rootScope.FindChild(func(s *symbol.Scope) bool {
				return s.Kind == symbol.KindParam
			})
			Expect(found).To(Equal(child))
		})

		It("Should return nil when no match", func() {
			rootScope := symbol.CreateRoot(nil)
			MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			found := rootScope.FindChild(func(s *symbol.Scope) bool {
				return s.Kind == symbol.KindFunction
			})
			Expect(found).To(BeNil())
		})
	})

	Describe("FilterChildren", func() {
		It("Should filter children by predicate", func() {
			rootScope := symbol.CreateRoot(nil)
			var1 := MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			MustSucceed(rootScope.Add("f", symbol.KindFunction, types.Function{}, nil))
			var2 := MustSucceed(rootScope.Add("y", symbol.KindVariable, types.I64{}, nil))
			filtered := rootScope.FilterChildren(func(s *symbol.Scope) bool {
				return s.Kind == symbol.KindVariable
			})
			Expect(filtered).To(HaveLen(2))
			Expect(filtered).To(ContainElements(var1, var2))
		})

		It("Should return empty when no matches", func() {
			rootScope := symbol.CreateRoot(nil)
			MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			filtered := rootScope.FilterChildren(func(s *symbol.Scope) bool {
				return s.Kind == symbol.KindTask
			})
			Expect(filtered).To(BeEmpty())
		})
	})

	Describe("Root", func() {
		It("Should return root scope from any depth", func() {
			rootScope := symbol.CreateRoot(nil)
			funcScope := MustSucceed(rootScope.Add("f", symbol.KindFunction, types.Function{}, nil))
			varScope := MustSucceed(funcScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			Expect(varScope.Root()).To(Equal(rootScope))
			Expect(funcScope.Root()).To(Equal(rootScope))
			Expect(rootScope.Root()).To(Equal(rootScope))
		})
	})

	Describe("Resolve", func() {
		It("Should resolve symbol in current scope", func() {
			rootScope := symbol.CreateRoot(nil)
			child := MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			resolved := MustSucceed(rootScope.Resolve("x"))
			Expect(resolved).To(Equal(child))
		})

		It("Should resolve symbol from parent scope", func() {
			rootScope := symbol.CreateRoot(nil)
			global := MustSucceed(rootScope.Add("global", symbol.KindVariable, types.I32{}, nil))
			funcScope := MustSucceed(rootScope.Add("f", symbol.KindFunction, types.Function{}, nil))
			resolved := MustSucceed(funcScope.Resolve("global"))
			Expect(resolved).To(Equal(global))
		})

		It("Should resolve from global resolver", func() {
			globalResolver := symbol.MapResolver{
				"pi": symbol.Symbol{Name: "pi", Kind: symbol.KindConfigParam, Type: types.F64{}},
			}
			rootScope := symbol.CreateRoot(globalResolver)
			resolved := MustSucceed(rootScope.Resolve("pi"))
			Expect(resolved.Name).To(Equal("pi"))
			Expect(resolved.Kind).To(Equal(symbol.KindConfigParam))
		})

		It("Should prioritize local over parent scope", func() {
			rootScope := symbol.CreateRoot(nil)
			rootX := MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			funcScope := MustSucceed(rootScope.Add("f", symbol.KindFunction, types.Function{}, nil))
			resolvedFromFunc := MustSucceed(funcScope.Resolve("x"))
			Expect(resolvedFromFunc).To(Equal(rootX))
		})

		It("Should return error for undefined symbol", func() {
			rootScope := symbol.CreateRoot(nil)
			_, err := rootScope.Resolve("undefined")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("undefined symbol: undefined"))
		})
	})

	Describe("ClosestAncestorOfKind", func() {
		It("Should find closest ancestor of kind", func() {
			rootScope := symbol.CreateRoot(nil)
			funcScope := MustSucceed(rootScope.Add("f", symbol.KindFunction, types.Function{}, nil))
			blockScope := MustSucceed(funcScope.Add("block", symbol.KindBlock, nil, nil))
			varScope := MustSucceed(blockScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			ancestor := MustSucceed(varScope.ClosestAncestorOfKind(symbol.KindFunction))
			Expect(ancestor).To(Equal(funcScope))
		})

		It("Should return self if matching kind", func() {
			rootScope := symbol.CreateRoot(nil)
			funcScope := MustSucceed(rootScope.Add("f", symbol.KindFunction, types.Function{}, nil))
			ancestor := MustSucceed(funcScope.ClosestAncestorOfKind(symbol.KindFunction))
			Expect(ancestor).To(Equal(funcScope))
		})

		It("Should return error when no ancestor found", func() {
			rootScope := symbol.CreateRoot(nil)
			_, err := rootScope.ClosestAncestorOfKind(symbol.KindTask)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("undefined symbol"))
		})
	})

	Describe("FirstChildOfKind", func() {
		It("Should find first child of kind", func() {
			rootScope := symbol.CreateRoot(nil)
			MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			funcChild := MustSucceed(rootScope.Add("f", symbol.KindFunction, types.Function{}, nil))
			MustSucceed(rootScope.Add("g", symbol.KindFunction, types.Function{}, nil))
			first := MustSucceed(rootScope.FirstChildOfKind(symbol.KindFunction))
			Expect(first).To(Equal(funcChild))
		})

		It("Should return error when no child of kind", func() {
			rootScope := symbol.CreateRoot(nil)
			MustSucceed(rootScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			_, err := rootScope.FirstChildOfKind(symbol.KindTask)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("undefined symbol"))
		})
	})

	Describe("String", func() {
		It("Should format scope as string", func() {
			rootScope := symbol.CreateRoot(nil)
			funcScope := MustSucceed(rootScope.Add("myFunc", symbol.KindFunction, types.Function{}, nil))
			MustSucceed(funcScope.Add("x", symbol.KindVariable, types.I32{}, nil))
			str := funcScope.String()
			Expect(str).To(ContainSubstring("name: myFunc"))
			Expect(str).To(ContainSubstring("kind: KindFunction"))
			Expect(str).To(ContainSubstring("name: x"))
			Expect(str).To(ContainSubstring("kind: KindVariable"))
			Expect(str).To(ContainSubstring("type: i32"))
		})
	})
})
