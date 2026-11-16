// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package context_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	analyzerContext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/testutil"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Context", func() {
	Describe("Root", func() {
		It("Should initialize all fields correctly", func(specCtx SpecContext) {
			ast := testutil.NewMockAST(1)
			ctx := analyzerContext.NewRoot(specCtx, ast, nil)
			Expect(ctx.Context).To(Equal(specCtx))
			Expect(ctx.Scope).ToNot(BeNil())
			Expect(ctx.Diagnostics).To(HaveValue(BeEmpty()))
			Expect(*ctx.Diagnostics).To(BeEmpty())
			Expect(ctx.Constraints).ToNot(BeNil())
			Expect(ctx.TypeMap).ToNot(BeNil())
			Expect(ctx.TypeMap).To(BeEmpty())
			Expect(ctx.FmtStrings).ToNot(BeNil())
			Expect(ctx.FmtStrings).To(BeEmpty())
			Expect(ctx.CallEdges).ToNot(BeNil())
			Expect(*ctx.CallEdges).To(BeEmpty())
			Expect(ctx.AST).To(Equal(ast))
			Expect(ctx.TypeHint).To(Equal(types.Type{}))
			Expect(ctx.InTypeInferenceMode).To(BeFalse())
		})
	})

	Describe("RejectTopLevel", func() {
		It(
			"Should reject a variable declared at the top level",
			func(specCtx SpecContext) {
				ctx := analyzerContext.NewRoot(specCtx, testutil.NewMockAST(1), nil)
				x := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
					Name: "x", Kind: symbol.KindVariable, Type: types.I64(),
				}))
				Expect(ctx.RejectTopLevel(x, "x")).To(BeTrue())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).
					To(Equal("cannot reassign top-level variable 'x'"))
			},
		)

		It("Should accept a variable declared in a scope", func(specCtx SpecContext) {
			ctx := analyzerContext.NewRoot(specCtx, testutil.NewMockAST(1), nil)
			seq := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "s", Kind: symbol.KindSequence,
			}))
			y := MustSucceed(seq.Add(ctx, symbol.Symbol{
				Name: "y", Kind: symbol.KindVariable, Type: types.I64(),
			}))
			Expect(ctx.RejectTopLevel(y, "y")).To(BeFalse())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should accept a top-level symbol that is not a variable", func(
			specCtx SpecContext,
		) {
			ctx := analyzerContext.NewRoot(specCtx, testutil.NewMockAST(1), nil)
			f := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "f", Kind: symbol.KindFunction,
			}))
			Expect(ctx.RejectTopLevel(f, "f")).To(BeFalse())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})
	})

	Describe("FloatConstant", func() {
		newContext := func(
			specCtx SpecContext,
			code string,
		) analyzerContext.Context[parser.IExpressionContext] {
			GinkgoHelper()
			custom := &symbol.Symbol{Name: "custom", Kind: symbol.KindModule}
			custom.AddChild(&symbol.Symbol{
				Name:         "text",
				Kind:         symbol.KindConstant,
				DefaultValue: "text",
			})
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), custom))
			symbol.AutoImportModules(root)
			return analyzerContext.NewRoot(
				specCtx,
				MustSucceed(parser.ParseExpression(code)),
				root,
			)
		}

		DescribeTable("Should return the value",
			func(specCtx SpecContext, code string, expected float64) {
				ctx := newContext(specCtx, code)
				Expect(MustBeOk(ctx.FloatConstant(ctx.AST))).To(Equal(expected))
			},
			Entry("a constant", "math.inf", math.Inf(1)),
			Entry("a negated constant", "-math.inf", math.Inf(-1)),
			Entry("a double negation", "--math.inf", math.Inf(1)),
			Entry("parentheses", "((math.inf))", math.Inf(1)),
			Entry("an f32 cast", "f32(-math.inf)", math.Inf(-1)),
			Entry("an f64 cast with a unit", "f64 psi(math.inf)", math.Inf(1)),
		)

		It("Should return NaN for math.nan", func(specCtx SpecContext) {
			ctx := newContext(specCtx, "-(math.nan)")
			Expect(math.IsNaN(MustBeOk(ctx.FloatConstant(ctx.AST)))).To(BeTrue())
		})

		DescribeTable("Should return false",
			func(specCtx SpecContext, code string) {
				ctx := newContext(specCtx, code)
				_, ok := ctx.FloatConstant(ctx.AST)
				Expect(ok).To(BeFalse())
			},
			Entry("a binary expression", "math.inf + 1.0"),
			Entry("an integer cast", "i64(math.inf)"),
			Entry("a bare name", "x"),
			Entry("a literal", "1.5"),
			Entry("an undefined head", "nope.inf"),
			Entry("an undefined member", "math.nope"),
			Entry("a member that is not a constant", "math.avg"),
			Entry("a constant without a float value", "custom.text"),
		)
	})

	Describe("ScopedKey", func() {
		It("Should name the enclosing function", func(specCtx SpecContext) {
			ast := MustSucceed(parser.Parse("func f() {\n    x := 1\n}"))
			ctx := analyzerContext.NewRoot(specCtx, ast, nil)
			fn := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "f",
				Kind: symbol.KindFunction,
			}))
			block := MustSucceed(fn.Add(ctx, symbol.Symbol{Kind: symbol.KindBlock}))
			Expect(ctx.WithScope(block).ScopedKey("call", ast)).To(Equal("call_f_1_0"))
		})

		It("Should leave the function empty outside one", func(specCtx SpecContext) {
			ast := MustSucceed(parser.Parse("x -> y"))
			ctx := analyzerContext.NewRoot(specCtx, ast, nil)
			Expect(ctx.ScopedKey("call", ast)).To(Equal("call__1_0"))
		})
	})

	Describe("Child", func() {
		It("Should share all pointers except AST", func(specCtx SpecContext) {
			var (
				parentAST = testutil.NewMockAST(1)
				childAST  = testutil.NewMockAST(2)
				parent    = analyzerContext.NewRoot(specCtx, parentAST, nil)
				child     = parent.Child(childAST)
			)
			Expect(child.AST).To(Equal(childAST))
			Expect(child.AST).ToNot(Equal(parent.AST))
			Expect(child.Context).To(Equal(parent.Context))
			Expect(child.Scope).To(BeIdenticalTo(parent.Scope))
			Expect(child.Diagnostics).To(BeIdenticalTo(parent.Diagnostics))
			Expect(child.Constraints).To(BeIdenticalTo(parent.Constraints))
			Expect(child.CallEdges).To(BeIdenticalTo(parent.CallEdges))
			Expect(child.TypeHint).To(Equal(parent.TypeHint))
			Expect(child.InTypeInferenceMode).To(Equal(parent.InTypeInferenceMode))
		})

		It("Should share state mutations", func(ctx SpecContext) {
			var (
				parentAST = testutil.NewMockAST(1)
				childAST  = testutil.NewMockAST(2)
				parent    = analyzerContext.NewRoot(ctx, parentAST, nil)
				child     = parent.Child(childAST)
			)
			child.Diagnostics.Add(diagnostics.Warningf(childAST, "test diagnostic"))
			Expect(*parent.Diagnostics).To(HaveLen(1))
			testAST := testutil.NewMockAST(3)
			child.TypeMap[testAST] = types.I32()
			Expect(parent.TypeMap[testAST]).To(Equal(types.I32()))
			child.FmtStrings[nil] = []parser.FmtSegment{{Text: "a"}}
			Expect(parent.FmtStrings).To(HaveLen(1))
		})

		It(
			"Should preserve parent's TypeHint and InTypeInferenceMode",
			func(ctx SpecContext) {
				parentAST := testutil.NewMockAST(1)
				childAST := testutil.NewMockAST(2)
				parent := analyzerContext.NewRoot(ctx, parentAST, nil)
				parent.TypeHint = types.F64()
				parent.InTypeInferenceMode = true
				child := parent.Child(childAST)
				Expect(child.TypeHint).To(Equal(types.F64()))
				Expect(child.InTypeInferenceMode).To(BeTrue())
			},
		)
	})

	Describe("WithScope", func() {
		It("Should return new context with updated scope", func(specCtx SpecContext) {
			var (
				ast           = testutil.NewMockAST(1)
				ctx           = analyzerContext.NewRoot(specCtx, ast, nil)
				originalScope = ctx.Scope
				newScope      = MustSucceed(ctx.Scope.Add(specCtx, symbol.Symbol{
					Name: "test",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{}),
				}))
				newCtx = ctx.WithScope(newScope)
			)
			Expect(newCtx.Scope).To(Equal(newScope))
			Expect(newCtx.Scope).ToNot(Equal(originalScope))
			Expect(ctx.Scope).To(Equal(originalScope))
			Expect(newCtx.Context).To(Equal(ctx.Context))
			Expect(newCtx.Diagnostics).To(BeIdenticalTo(ctx.Diagnostics))
			Expect(newCtx.Constraints).To(BeIdenticalTo(ctx.Constraints))
			Expect(newCtx.AST).To(Equal(ctx.AST))
			Expect(newCtx.TypeHint).To(Equal(ctx.TypeHint))
			Expect(newCtx.InTypeInferenceMode).To(Equal(ctx.InTypeInferenceMode))
		})
	})

	Describe("WithModule", func() {
		It(
			"Should return new context with the module flag set",
			func(specCtx SpecContext) {
				var (
					ast    = testutil.NewMockAST(1)
					ctx    = analyzerContext.NewRoot(specCtx, ast, nil)
					newCtx = ctx.WithModule(true)
				)
				Expect(newCtx.Module).To(BeTrue())
				Expect(ctx.Module).To(BeFalse())
				Expect(newCtx.Scope).To(Equal(ctx.Scope))
				Expect(newCtx.Diagnostics).To(BeIdenticalTo(ctx.Diagnostics))
			},
		)

		It("Should carry the module flag into a child", func(specCtx SpecContext) {
			var (
				parent = analyzerContext.NewRoot(specCtx, testutil.NewMockAST(1), nil)
				child  = parent.WithModule(true).Child(testutil.NewMockAST(2))
			)
			Expect(child.Module).To(BeTrue())
		})
	})

	Describe("WithTypeHint", func() {
		It(
			"Should return new context with updated type hint",
			func(specCtx SpecContext) {
				var (
					ast    = testutil.NewMockAST(1)
					ctx    = analyzerContext.NewRoot(specCtx, ast, nil)
					newCtx = ctx.WithTypeHint(types.F64())
				)
				Expect(newCtx.TypeHint).To(Equal(types.F64()))
				Expect(ctx.TypeHint).To(Equal(types.Type{}))
				Expect(newCtx.Context).To(Equal(ctx.Context))
				Expect(newCtx.Scope).To(Equal(ctx.Scope))
				Expect(newCtx.Diagnostics).To(BeIdenticalTo(ctx.Diagnostics))
				Expect(newCtx.Constraints).To(BeIdenticalTo(ctx.Constraints))
				Expect(newCtx.AST).To(Equal(ctx.AST))
				Expect(newCtx.InTypeInferenceMode).To(Equal(ctx.InTypeInferenceMode))
			},
		)

		It("Should allow chaining with WithScope", func(specCtx SpecContext) {
			var (
				ast      = testutil.NewMockAST(1)
				ctx      = analyzerContext.NewRoot(specCtx, ast, nil)
				newScope = MustSucceed(ctx.Scope.Add(specCtx, symbol.Symbol{
					Name: "test",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{}),
				}))
				newCtx = ctx.WithTypeHint(types.I32()).WithScope(newScope)
			)
			Expect(newCtx.TypeHint).To(Equal(types.I32()))
			Expect(newCtx.Scope).To(Equal(newScope))
		})
	})

	Describe("Integration", func() {
		It(
			"Should support realistic workflow with one parsed AST",
			func(specCtx SpecContext) {
				var (
					prog     = MustSucceed(parser.Parse(`func test() {}`))
					rootCtx  = analyzerContext.NewRoot(specCtx, prog, nil)
					newScope = MustSucceed(rootCtx.Scope.Add(specCtx, symbol.Symbol{
						Name: "x",
						Kind: symbol.KindVariable,
						Type: types.I32(),
					}))
					mockChild = testutil.NewMockAST(99)
					finalCtx  = rootCtx.Child(mockChild).
							WithScope(newScope).
							WithTypeHint(types.String())
				)
				Expect(finalCtx.AST).To(Equal(mockChild))
				Expect(finalCtx.TypeHint).To(Equal(types.String()))
				Expect(finalCtx.Scope).To(Equal(newScope))
				finalCtx.Diagnostics.Add(diagnostics.Errorf(finalCtx.AST, "test"))
				Expect(*rootCtx.Diagnostics).To(HaveLen(1))
			},
		)
	})

	Describe("ResolveOwnScope", func() {
		It(
			"Should resolve a named declaration by its identifier",
			func(specCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`sequence main {}`))
				ctx := analyzerContext.NewRoot(specCtx, ast, nil)
				analyzer.AnalyzeProgram(ctx)
				seqDecl := ast.AllTopLevelItem()[0].SequenceDeclaration()
				scope := MustSucceed(analyzerContext.ResolveOwnScope(
					ctx.Child(seqDecl),
				))
				Expect(scope.Name).To(Equal("main"))
				Expect(scope.Kind).To(Equal(symbol.KindSequence))
			},
		)

		It(
			"Should resolve an anonymous declaration by its parser rule",
			func(specCtx SpecContext) {
				ast := MustSucceed(parser.Parse(`
				sequence main {
					stage {}
				}
			`))
				ctx := analyzerContext.NewRoot(specCtx, ast, nil)
				analyzer.AnalyzeProgram(ctx)
				seqDecl := ast.AllTopLevelItem()[0].SequenceDeclaration()
				seqScope := MustSucceed(ctx.Scope.Resolve(ctx, "main"))
				stageDecl := seqDecl.AllSequenceItem()[0].StageDeclaration()
				scope := MustSucceed(analyzerContext.ResolveOwnScope(
					ctx.Child(stageDecl).WithScope(seqScope),
				))
				Expect(scope.Kind).To(Equal(symbol.KindStage))
			},
		)
	})
})
