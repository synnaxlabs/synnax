// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement_test

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/analyzer/testutil"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

// readWriteChan builds the read+write channel type a bare channel alias carries.
func readWriteChan(elem types.Type) types.Type {
	t := types.Chan(elem)
	t.ChanDirection = types.ChanDirectionRead | types.ChanDirectionWrite
	return t
}

var _ = Describe("Statement", func() {
	// Helper to set up function context for tests that need it
	setupFunctionContext := func(ctx context.Context[parser.IBlockContext]) {
		GinkgoHelper()
		scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
			Name: "testFunc",
			Kind: symbol.KindFunction,
		}))
		Expect(scope).ToNot(BeNil())
		fn := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
		ctx.Scope = fn
	}

	Describe("Variable Declaration", func() {
		Context("local variables", func() {
			DescribeTable(
				"type inference and validation",
				func(bCtx SpecContext, code string, expectOk bool, assertion func(context.Context[parser.IStatementContext])) {
					stmt := MustSucceed(parser.ParseStatement(code))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(Equal(expectOk))
					if assertion != nil {
						assertion(ctx)
					}
				},
				Entry(
					"explicit type with initializer",
					`x i32 := 42`,
					true,
					func(ctx context.Context[parser.IStatementContext]) {
						sym := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
						Expect(sym.Type).To(Equal(types.I32()))
					},
				),
				Entry(
					"inferred type from float literal",
					`x := 3.14`,
					true,
					func(ctx context.Context[parser.IStatementContext]) {
						Expect(*ctx.Diagnostics).To(BeEmpty())
						sym := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
						Expect(sym.Type.Kind).To(Equal(types.KindVariable))
						Expect(sym.Type.Constraint).ToNot(BeNil())
						Expect(
							sym.Type.Constraint.Kind,
						).To(Equal(types.KindFloatConstant))
					},
				),
			)

			DescribeTable("folds a default value from a literal initializer",
				func(bCtx SpecContext, code string, expected any) {
					stmt := MustSucceed(parser.ParseStatement(code))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue())
					sym := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
					Expect(sym.DefaultValue).To(Equal(expected))
				},
				Entry("i8 positive", `x i8 := 7`, int8(7)),
				Entry("i8 negative", `x i8 := -5`, int8(-5)),
				Entry("i16 negative", `x i16 := -5`, int16(-5)),
				Entry("i32 negative", `x i32 := -5`, int32(-5)),
				Entry("i64 positive", `x i64 := 5`, int64(5)),
				Entry("i64 negative", `x i64 := -5`, int64(-5)),
				Entry("f32 negative", `x f32 := -2.5`, float32(-2.5)),
				Entry("f64 positive", `x f64 := 2.5`, float64(2.5)),
				Entry("f64 negative", `x f64 := -2.5`, float64(-2.5)),
				Entry("i8 type minimum", `x i8 := -128`, int8(-128)),
				Entry("i16 type minimum", `x i16 := -32768`, int16(-32768)),
				Entry("i32 type minimum", `x i32 := -2147483648`, int32(-2147483648)),
			)

			It(
				"should detect type mismatch between declaration and initializer",
				func(bCtx SpecContext) {
					stmt := MustSucceed(parser.ParseStatement(`x i32 := "hello"`))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("type mismatch: cannot assign str to 'x' (type i32)"))
				},
			)

			It(
				"should reject a negative literal assigned to a string variable",
				func(bCtx SpecContext) {
					stmt := MustSucceed(parser.ParseStatement(`x str := -5`))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("integer is not compatible with str"))
				},
			)

			It("should detect duplicate variable declaration", func(bCtx SpecContext) {
				stmt := MustSucceed(parser.ParseBlock(`{
					x := 1
					x := 1
				}`))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("name x conflicts with existing variable"))
			})

			It(
				"should detect a duplicate declaration whose type does not infer",
				func(bCtx SpecContext) {
					stmt := MustSucceed(parser.ParseBlock(`{
					x := 1
					x := undefined_name
				}`))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(*ctx.Diagnostics).To(HaveLen(2))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(Equal("undefined symbol: undefined_name"))
					Expect(
						(*ctx.Diagnostics)[1].Message,
					).To(ContainSubstring("name x conflicts with existing variable"))
				},
			)

			It(
				"should detect undefined variable in initializer",
				func(bCtx SpecContext) {
					stmt := MustSucceed(parser.ParseStatement(`x := y + 1`))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("undefined symbol: y"))
				},
			)
		})

		Context("stateful variables", func() {
			analyzeInStage := func(
				bCtx SpecContext, code string,
			) context.Context[parser.IStatementContext] {
				GinkgoHelper()
				stmt := MustSucceed(parser.ParseStatement(code))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				ctx.Scope = MustSucceed(ctx.Scope.Add(bCtx, symbol.Symbol{
					Name: "s1", Kind: symbol.KindStage,
				}))
				statement.Analyze(ctx)
				return ctx
			}

			It(
				"should analyze a stateful variable with inferred type",
				func(bCtx SpecContext) {
					ctx := analyzeInStage(bCtx, `counter $= 0`)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue())
					Expect(*ctx.Diagnostics).To(BeEmpty())
					sym := MustSucceed(ctx.Scope.Resolve(ctx, "counter"))
					Expect(sym.Kind).To(Equal(symbol.KindStatefulVariable))
					Expect(sym.Type.Kind).To(Equal(types.KindVariable))
					Expect(sym.Type.Constraint).ToNot(BeNil())
					Expect(
						sym.Type.Constraint.Kind,
					).To(Equal(types.KindIntegerConstant))
				},
			)

			It(
				"should analyze stateful variable with explicit type",
				func(bCtx SpecContext) {
					ctx := analyzeInStage(bCtx, `total f32 $= 0.0`)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue())
					Expect(*ctx.Diagnostics).To(BeEmpty())
					sym := MustSucceed(ctx.Scope.Resolve(ctx, "total"))
					Expect(sym.Type).To(Equal(types.F32()))
				},
			)

			It(
				"should detect a duplicate stateful variable whose type does not infer",
				func(bCtx SpecContext) {
					block := MustSucceed(parser.ParseBlock(`{
					x $= 1
					x $= undefined_name
				}`))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					ctx.Scope = MustSucceed(ctx.Scope.Add(bCtx, symbol.Symbol{
						Name: "s1", Kind: symbol.KindStage,
					}))
					statement.AnalyzeBlock(ctx)
					Expect(*ctx.Diagnostics).To(HaveLen(2))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(Equal("undefined symbol: undefined_name"))
					Expect(
						(*ctx.Diagnostics)[1].Message,
					).To(ContainSubstring("name x conflicts with existing variable"))
				},
			)

			It(
				"should detect an undefined symbol in a stateful initializer",
				func(bCtx SpecContext) {
					ctx := analyzeInStage(bCtx, `y $= undefined_name`)
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(Equal("undefined symbol: undefined_name"))
				},
			)

			It(
				"rejects a stateful variable declared at the top level",
				func(bCtx SpecContext) {
					stmt := MustSucceed(parser.ParseStatement(`counter $= 0`))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(
						"stateful variables cannot be declared at the top level",
					))
				},
			)
		})

		Context("scoped declarations", func() {
			// sensorChan lets reactive and alias declarations read a channel.
			sensorChan := []symbol.Symbol{
				{
					Kind: symbol.KindChannel,
					Name: "sensor",
					Type: types.Chan(types.F64()),
				},
			}
			// Declarations go into a stage child; top-level statefuls are illegal.
			declareIn := func(
				bCtx SpecContext, root *symbol.Symbol, code string,
			) context.Context[parser.IVariableDeclarationContext] {
				GinkgoHelper()
				stage := root.FindChild("body")
				if stage == nil {
					stage = MustSucceed(root.Add(bCtx, symbol.Symbol{
						Name: "body", Kind: symbol.KindStage,
					}))
				}
				stmt := MustSucceed(parser.ParseStatement(code))
				ctx := context.NewRoot(bCtx, stmt.VariableDeclaration(), root)
				ctx.Scope = stage
				statement.AnalyzeVariableDeclaration(ctx)
				return ctx
			}

			DescribeTable(
				"classify the declared variable",
				func(bCtx SpecContext, code, name string, classify func(*symbol.Symbol)) {
					ctx := declareIn(bCtx, NewRoot(nil, sensorChan...), code)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					classify(MustSucceed(ctx.Scope.Resolve(ctx, name)))
				},
				Entry("folded literal", "gain := 42.0", "gain", func(s *symbol.Symbol) {
					Expect(s.IsLiteral()).To(BeTrue())
					Expect(s.DefaultValue).ToNot(BeNil())
				}),
				Entry(
					"computed literal",
					"scaled := 2 * 3",
					"scaled",
					func(s *symbol.Symbol) {
						Expect(s.IsLiteral()).To(BeTrue())
					},
				),
				Entry(
					"bare-channel alias",
					"alias := sensor",
					"alias",
					func(s *symbol.Symbol) {
						Expect(s.IsChannelReadWrite()).To(BeTrue())
						Expect(s.SourceID).ToNot(BeNil())
					},
				),
				Entry(
					"reactive expression",
					"derived := sensor + 1",
					"derived",
					func(s *symbol.Symbol) {
						Expect(s.IsReactive()).To(BeTrue())
					},
				),
				Entry(
					"stateful literal",
					"counter $= 0",
					"counter",
					func(s *symbol.Symbol) {
						Expect(s.Kind).To(Equal(symbol.KindStatefulVariable))
						Expect(s.IsLiteral()).To(BeTrue())
					},
				),
			)

			DescribeTable("reject an invalid stateful initializer",
				func(bCtx SpecContext, code string) {
					ctx := declareIn(bCtx, NewRoot(nil, sensorChan...), code)
					statement.CheckStatefulInitializers(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(
						ContainSubstring("cannot be assigned to stateful"),
					)
				},
				Entry("bare channel", "bad $= sensor"),
				Entry("channel-read expression", "bad $= sensor + 1"),
			)

			It("keeps a bare alias to a literal a literal", func(bCtx SpecContext) {
				root := NewRoot(nil, sensorChan...)
				declareIn(bCtx, root, "gain := 42.0")
				ctx := declareIn(bCtx, root, "mirror := gain")
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(
					MustSucceed(ctx.Scope.Resolve(ctx, "mirror")).IsLiteral(),
				).To(BeTrue())
			})

			It("promotes a local that reads a reactive local", func(bCtx SpecContext) {
				root := NewRoot(nil, sensorChan...)
				declareIn(bCtx, root, "s := sensor * 2.0")
				ctx := declareIn(bCtx, root, "z := s + 1.0")
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				z := MustSucceed(ctx.Scope.Resolve(ctx, "z"))
				Expect(z.IsComputed()).To(BeTrue())
				statement.PromoteChannelReadLocals(ctx)
				Expect(z.IsReactive()).To(BeTrue())
			})

			It("promotes a copy of a promoted local", func(bCtx SpecContext) {
				root := NewRoot(nil, sensorChan...)
				declareIn(bCtx, root, "s := sensor * 2.0")
				declareIn(bCtx, root, "z := s + 1.0")
				ctx := declareIn(bCtx, root, "w := z")
				statement.PromoteChannelReadLocals(ctx)
				Expect(
					MustSucceed(ctx.Scope.Resolve(ctx, "w")).IsReactive(),
				).To(BeTrue())
			})

			It(
				"rejects a stateful that reads a reactive local",
				func(bCtx SpecContext) {
					root := NewRoot(nil, sensorChan...)
					declareIn(bCtx, root, "s := sensor * 2.0")
					ctx := declareIn(bCtx, root, "bad $= s + 1.0")
					statement.CheckStatefulInitializers(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(
						ContainSubstring("cannot be assigned to stateful"),
					)
				},
			)

			It("classifies a bare call as computed", func(bCtx SpecContext) {
				root := NewRoot(nil, symbol.Symbol{
					Name: "one",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F64()},
						},
					}),
				})
				ctx := declareIn(bCtx, root, "x := one()")
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(
					MustSucceed(ctx.Scope.Resolve(ctx, "x")).IsComputed(),
				).To(BeTrue())
				expr := ctx.AST.LocalVariable().Expression()
				Expect(MustSucceed(root.GetChildByParserRule(expr)).Kind).To(
					Equal(symbol.KindFunction),
				)
			})

			It("rejects a stateful bound to a ':=' variable", func(bCtx SpecContext) {
				root := NewRoot(nil)
				declareIn(bCtx, root, "s $= 0")
				ctx := declareIn(bCtx, root, "x := s")
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(
					ContainSubstring("stateful variables cannot be assigned"),
				)
			})

			It(
				"accepts a stateful initialized from a variable",
				func(bCtx SpecContext) {
					root := NewRoot(nil)
					declareIn(bCtx, root, "base i64 := 5")
					ctx := declareIn(bCtx, root, "total i64 $= base")
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
			)

			It(
				"accepts a stateful initialized from a computed expression",
				func(bCtx SpecContext) {
					root := NewRoot(nil)
					ctx := declareIn(bCtx, root, "total i64 $= 2 + 3")
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
			)

			It(
				"accepts a negated-literal stateful initializer",
				func(bCtx SpecContext) {
					root := NewRoot(nil)
					ctx := declareIn(bCtx, root, "total i64 $= -5")
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
			)

			It(
				"makes an explicitly-typed alias to a reactive variable reactive",
				func(bCtx SpecContext) {
					root := NewRoot(nil, sensorChan...)
					declareIn(bCtx, root, "derived := sensor + 1")
					ctx := declareIn(bCtx, root, "mirror f64 := derived")
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					Expect(
						MustSucceed(ctx.Scope.Resolve(ctx, "mirror")).IsReactive(),
					).To(BeTrue())
				},
			)
		})
	})

	Describe("Typed series literals", func() {
		analyze := func(
			specCtx SpecContext,
			code string,
		) context.Context[parser.IProgramContext] {
			GinkgoHelper()
			ast := MustSucceed(parser.Parse("func f() {v f32} {\n" + code + "\n}"))
			ctx := context.NewRoot(specCtx, ast, NewRoot(nil))
			analyzer.AnalyzeProgram(ctx)
			return ctx
		}

		inIndex := func(node antlr.Tree) bool {
			for ; node != nil; node = node.GetParent() {
				if _, ok := node.(parser.IIndexOrSliceContext); ok {
					return true
				}
			}
			return false
		}

		DescribeTable(
			"Should give every numeric literal the element type",
			func(specCtx SpecContext, code string, expected types.Type, count int) {
				ctx := analyze(specCtx, code)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				literals := 0
				for node, t := range ctx.TypeMap {
					lit, ok := node.(parser.ILiteralContext)
					if !ok || lit.NumericLiteral() == nil || inIndex(lit) {
						continue
					}
					literals++
					Expect(t).To(Equal(expected), node.GetText())
				}
				Expect(literals).To(Equal(count))
			},
			Entry("f32", "y series f32 := [1.5]", types.F32(), 1),
			Entry("an integer into an empty series", "y := []\ny += 1", types.F64(), 1),
			Entry(
				"an integer element into an empty series",
				"y := []\ny[0] = 1",
				types.F64(),
				1,
			),
			Entry(
				"an integer series into an empty series",
				"y := []\ny = [1]",
				types.F64(),
				1,
			),
			Entry(
				"an integer then a float into an empty series",
				"y := []\ny += 1\ny += 1.5",
				types.F64(),
				2,
			),
			Entry(
				"a float then an integer into an empty series",
				"y := []\ny += 1.5\ny += 1",
				types.F64(),
				2,
			),
			Entry("f64", "y series f64 := [1.5]", types.F64(), 1),
			Entry(
				"every f32 element",
				"y series f32 := [1.5, 2.5, 3.5]",
				types.F32(),
				3,
			),
			Entry("mixed literals", "y series f32 := [1, 2.5]", types.F32(), 2),
			Entry("integer in f32", "y series f32 := [3]", types.F32(), 1),
			Entry("i32", "y series i32 := [1, 2]", types.I32(), 2),
			Entry("i8", "y series i8 := [1]", types.I8(), 1),
			Entry("u8", "y series u8 := [200]", types.U8(), 1),
			Entry("u64", "y series u64 := [1]", types.U64(), 1),
			Entry("arithmetic", "y series f32 := [1.5 + 2]", types.F32(), 2),
			Entry("integer arithmetic", "y series i32 := [1 + 2 * 3]", types.I32(), 3),
			Entry("negated", "y series f32 := [-1.5]", types.F32(), 1),
			Entry("parentheses", "y series f32 := [(1.5 + 2)]", types.F32(), 2),
			Entry("with a variable", "y series f32 := [1.5 + v]", types.F32(), 1),
			Entry("after a variable", "y series f32 := [v, 1.5]", types.F32(), 1),
			Entry("reassigned", "y series f32 := [1.5]\ny = [2.5]", types.F32(), 2),
			Entry("untyped float first", "y := [1.5, 2.0, 3]", types.F64(), 3),
			Entry("untyped integer first", "y := [1, 2.0, 2.5]", types.F64(), 3),
		)

		It(
			"Should reject a float element in an integer series",
			func(specCtx SpecContext) {
				ctx := analyze(specCtx, "y series i32 := [1.5]")
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(
					"cannot assign series float to 'y' (type series i32)",
				))
			},
		)

		DescribeTable("Should default an untyped empty series to f64",
			func(specCtx SpecContext, code string) {
				ctx := analyze(specCtx, code)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				s := MustSucceed(ctx.Scope.Resolve(ctx, "f"))
				block := s.FilterChildrenByKind(symbol.KindBlock)[0]
				y := MustSucceed(block.Resolve(ctx, "y"))
				Expect(ctx.Constraints.ApplySubstitutions(y.Type)).
					To(Equal(types.Series(types.F64())))
			},
			Entry("declared", "y := []"),
			Entry("reassigned", "y := []\ny = []"),
		)

		DescribeTable("Should bind an untyped empty series on reassignment",
			func(specCtx SpecContext, code string) {
				ctx := analyze(specCtx, code)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				s := MustSucceed(ctx.Scope.Resolve(ctx, "f"))
				block := s.FilterChildrenByKind(symbol.KindBlock)[0]
				y := MustSucceed(block.Resolve(ctx, "y"))
				Expect(ctx.Constraints.ApplySubstitutions(y.Type)).
					To(Equal(types.Series(types.F32())))
			},
			Entry("a literal", "y := []\ny = [v]"),
			Entry("a series", "s series f32 := [1.5]\ny := []\ny = s"),
		)

		DescribeTable("Should accept an empty series of any element type",
			func(specCtx SpecContext, element string) {
				ctx := analyze(specCtx, "y series "+element+" := []\ny = []")
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			},
			Entry("bool", "bool"),
			Entry("str", "str"),
			Entry("f32", "f32"),
			Entry("i8", "i8"),
			Entry("u64", "u64"),
			Entry("timestamp", "i64 ns"),
		)

		DescribeTable("Should use an untyped empty series as numeric",
			func(specCtx SpecContext, code string) {
				ctx := analyze(specCtx, code)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			},
			Entry("compound assignment", "y := []\ny += v"),
			Entry("indexed compound assignment", "y := []\ny[0] += v"),
			Entry("negation", "y := []\nz := -y"),
		)

		DescribeTable("Should bind untyped empty series to what an operator needs",
			func(specCtx SpecContext, code, message string) {
				ctx := analyze(specCtx, code)
				if message == "" {
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					return
				}
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(message))
			},
			Entry("arithmetic", "a := []\nb := []\nz := a * b\na = [true]", "bool"),
			Entry("subtraction", "a := []\nb := []\nz := a - b\nb = [true]", "bool"),
			Entry("addition", "a := []\nb := []\nz := a + b\na = [\"x\"]", "str"),
			Entry("a comparison", "a := []\nb := []\nz := a < b\na = [true]", "bool"),
			Entry("a logical and", "a := []\nz := a and a\na = [1.5]", "bool"),
			Entry("a logical or", "a := []\nz := a or a\na = [1.5]", "bool"),
			Entry("not", "a := []\nz := not a\na = [1.5]", "bool"),
			Entry("numeric use", "a := []\nb := []\nz := a * b\na = [v]", ""),
			Entry("bool use", "a := []\nz := not a\na = [true]", ""),
		)

		DescribeTable(
			"Should give an untyped variable one type",
			func(specCtx SpecContext, code, message string) {
				ctx := analyze(specCtx, code)
				if message == "" {
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					return
				}
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(message))
			},
			Entry(
				"a scalar read after an assignment",
				"x := 1.5\nx = v\nw f32 := x",
				"",
			),
			Entry(
				"a scalar read before an assignment",
				"x := 1.5\nw := x\nx = v",
				"cannot assign",
			),
			Entry(
				"a series read after an assignment",
				"y := [1.5]\ny = [v]\nw series f32 := y",
				"",
			),
			Entry(
				"a series read before an assignment",
				"y := [1.5]\nw := y\ny = [v]",
				"cannot assign",
			),
			Entry(
				"an element read before an assignment",
				"y := [1.5]\nw := y[0]\ny = [v]",
				"cannot assign",
			),
			Entry(
				"an empty series read before a use",
				"y := []\nn := len(y)\ny = [v]",
				"",
			),
			Entry("an integer into a float", "x := 0.0\nx = 1\nw f64 := x", ""),
			Entry("an integer added to a float", "x := 0.0\nx += 1\nw f64 := x", ""),
			Entry("a float after a typed write", "x := 0\nx = v\nx = 1.5", ""),
			Entry("a typed write after a float", "x := 0.0\nx = 1.0\nx = v", ""),
			Entry(
				"an integer write after a float added to an empty series",
				"n i32 := 1\ny := []\ny += 1.5\ny[0] = n",
				"i32",
			),
			Entry(
				"an integer write after a float written to an empty series element",
				"n i32 := 1\ny := []\nw := len(y)\ny[0] += 1.5\ny[0] = n",
				"i32",
			),
			Entry(
				"a typed element after a float bound an open element",
				"n i32 := 1\ne := []\ny := [e[0], 1.5, n]",
				"float is not compatible with i32",
			),
			Entry(
				"a series of literals added to a series",
				"y := [1.5, 2.5]\ny += [1, 2]\nw series f64 := y",
				"",
			),
			Entry("a typed write after an integer", "x := 0\nx = 1\nx = v", ""),
			Entry("a float added after a typed write", "x := 0\nx = v\nx += 1.5", ""),
			Entry(
				"an integer into a float element",
				"y := [1.5]\ny[0] = 1\nw series f64 := y",
				"",
			),
		)

		DescribeTable("Should store a typed series element at its kind",
			func(specCtx SpecContext, code, message string) {
				ctx := analyze(specCtx, code)
				if message == "" {
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					return
				}
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(message))
			},
			Entry("a unit on the element", "p f32 psi := v\ny series f32 := [p]", ""),
			Entry("a unit on the series", "y series f32 psi := [v]", ""),
			Entry("another kind", "p f64 := 1.5\ny series f32 := [p]", "cannot assign"),
			Entry(
				"a float after an exact float into an integer",
				"n i32 := 1\ny := [2.0, 1.5]\ny[0] = n",
				"i32 is not compatible with float",
			),
		)

		DescribeTable("Should check a returned series literal",
			func(specCtx SpecContext, returnType, message string) {
				ast := MustSucceed(parser.Parse(
					"func g() series " + returnType + " {\nreturn [1.5]\n}",
				))
				ctx := context.NewRoot(specCtx, ast, NewRoot(nil))
				analyzer.AnalyzeProgram(ctx)
				if message == "" {
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					return
				}
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(message))
			},
			Entry("a float series", "f32", ""),
			Entry("an integer series", "i32", "cannot return"),
		)

		DescribeTable("Should reject a series returned from a scalar function",
			func(specCtx SpecContext, code string) {
				ast := MustSucceed(parser.Parse(
					"func g() f64 {\ndata := [1.0, 2.0, 3.0]\n" + code + "\n}",
				))
				ctx := context.NewRoot(specCtx, ast, NewRoot(nil))
				analyzer.AnalyzeProgram(ctx)
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(
					"cannot return series f64 from 'g': expected f64",
				))
			},
			Entry("an untyped slice variable", "sub := data[1:3]\nreturn sub"),
			Entry("a series variable", "return data"),
			Entry("a slice", "return data[1:3]"),
		)

		DescribeTable("Should reject a series and a scalar mixed in a declaration",
			func(specCtx SpecContext, code, message string) {
				ctx := analyze(specCtx, "data := [1.0, 2.0, 3.0]\n"+code)
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(message))
			},
			Entry(
				"a slice into a scalar",
				"x f64 := data[1:3]",
				"cannot assign series f64 to 'x' (type f64)",
			),
			Entry(
				"a scalar into a series",
				"y f64 := 2.0\nz series f64 := y",
				"cannot assign f64 to 'z' (type series f64)",
			),
			Entry(
				"a slice into a stateful scalar",
				"x f64 $= data[1:3]",
				"cannot assign series f64 to 'x' (type f64)",
			),
		)

		DescribeTable(
			"Should keep an untyped integer variable an integer",
			func(specCtx SpecContext, code, message string) {
				ctx := analyze(specCtx, code)
				if message == "" {
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					return
				}
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(message))
			},
			Entry("reassignment", "x := 0\nx = 1.5", "cannot assign float to 'x'"),
			Entry("an integer-valued float write", "x := 0\nx = 2.0\nw i64 := x", ""),
			Entry(
				"a float after an integer-valued float write",
				"x := 0\nx = 2.0\nx = 1.5",
				"cannot assign float to 'x'",
			),
			Entry(
				"compound assignment",
				"x := 0\nx += 1.5",
				"expected integer, got float",
			),
			Entry(
				"indexed assignment",
				"y := [1, 3]\ny[0] = 1.5",
				"expected integer, got float",
			),
			Entry(
				"indexed compound assignment",
				"y := [1, 3]\ny[0] += 1.5",
				"expected integer, got float",
			),
			Entry(
				"series compound assignment",
				"y := [1, 3]\ny += 1.5",
				"expected integer, got float",
			),
			Entry(
				"series compound assignment of a series",
				"y := [1, 3]\ny += [1.5]",
				"expected integer, got float",
			),
			Entry(
				"a series element",
				"y := [1, 3]\nz := [y[0], 1.5]",
				"incompatible type float",
			),
			Entry(
				"arithmetic",
				"y := [1, 3]\nw f32 := y[0] + v",
				"cannot use i64 and f32",
			),
			Entry(
				"a series of another type",
				"y := [1, 3]\nw series f32 := [y[0]]",
				"cannot assign series i64 to 'w'",
			),
			Entry("an integer", "x := 0\nx = 2", ""),
			Entry("an integer into a float", "x := 0.0\nx = 1", ""),
			Entry("a float into an empty series", "y := []\ny += 1.5", ""),
			Entry(
				"an integer then a float into an empty series",
				"y := []\ny += 1\ny += 1.5",
				"",
			),
			Entry(
				"a float then an integer into an empty series",
				"y := []\ny += 1.5\ny += 1",
				"",
			),
			Entry(
				"a float element after an integer element into an empty series",
				"y := []\ny[0] = 1\ny[0] = 1.5",
				"",
			),
			Entry(
				"a float series after an integer series into an empty series",
				"y := []\ny = [1]\ny = [1.5]",
				"",
			),
			Entry(
				"a bool after an integer into an empty series",
				"y := []\ny[0] = 1\ny[0] = true",
				"bool",
			),
			Entry(
				"an integer into an empty series of a declared type",
				"y := []\ny[0] = 1\nw i32 := y[0]",
				"",
			),
			Entry(
				"a float series after an integer into an empty series",
				"y := []\ny += 1\ny = [1.5]",
				"",
			),
		)

		It(
			"Should keep a series read in a literal at its own type",
			func(specCtx SpecContext) {
				ctx := analyze(specCtx, "y := [1, 3]\nw := [y[0]]")
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				s := MustSucceed(ctx.Scope.Resolve(ctx, "f"))
				block := s.FilterChildrenByKind(symbol.KindBlock)[0]
				y := MustSucceed(block.Resolve(ctx, "y"))
				Expect(
					ctx.Constraints.ApplySubstitutions(y.Type),
				).To(Equal(types.Series(types.I64())))
			},
		)

		DescribeTable("Should reject an empty series in non-numeric arithmetic",
			func(specCtx SpecContext, code, message string) {
				ctx := analyze(specCtx, code)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(message))
			},
			Entry("bool on the right",
				"flags series bool := [true]\nz := [] * flags", "cannot use bool"),
			Entry("str on the right",
				"names series str := [\"a\"]\nz := [] - names", "cannot use str"),
			Entry("bool on the left",
				"flags series bool := [true]\nz := flags * []", "cannot use"),
		)

		It(
			"Should reject a typed integer after a float literal",
			func(specCtx SpecContext) {
				ctx := analyze(specCtx, "z i32 := 1\nx := [1, 1.5, z]")
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(
					"series element 3 has incompatible type i32, expected float",
				))
			},
		)

		It(
			"Should report an incompatible element once",
			func(specCtx SpecContext) {
				ctx := analyze(specCtx, "a i32 := 1\nx := [a, 12.5]")
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(
					"series element 2 has incompatible type",
				))
			},
		)
	})

	Describe("Assignment", func() {
		DescribeTable("assignment validation",
			func(bCtx SpecContext, code string, expectOk bool, errorSubstring string) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				setupFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(Equal(expectOk))
				if !expectOk && errorSubstring != "" {
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring(errorSubstring))
				}
			},
			Entry("valid assignment to existing variable", `{
				x := 42
				x = 99
			}`, true, ""),
			Entry("assignment to undefined variable", `{
				x = 42
			}`, false, "undefined"),
			Entry("type mismatch in assignment", `{
				x i32 := 10
				x = "hello"
			}`, false, "type mismatch"),
			Entry("scalar to series type mismatch", `{
				x := 12
				x = [1, 2]
			}`, false, "type mismatch"),
		)

		It(
			"validates an assignment through the exported entry point",
			func(bCtx SpecContext) {
				root := NewRoot(nil)
				stage := MustSucceed(root.Add(bCtx, symbol.Symbol{
					Name: "s1", Kind: symbol.KindStage,
				}))
				decl := MustSucceed(parser.ParseStatement("x := 42"))
				declCtx := context.NewRoot(bCtx, decl.VariableDeclaration(), root)
				declCtx.Scope = stage
				statement.AnalyzeVariableDeclaration(declCtx)
				assign := MustSucceed(parser.ParseStatement("x = 99"))
				ctx := context.NewRoot(bCtx, assign.Assignment(), root)
				ctx.Scope = stage
				statement.AnalyzeAssignment(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			},
		)
	})

	Describe("Reassignment and Rebind", func() {
		channels := []symbol.Symbol{
			{Kind: symbol.KindChannel, Name: "sensor", Type: types.Chan(types.F64())},
			{Kind: symbol.KindChannel, Name: "backup", Type: types.Chan(types.F64())},
			{
				Kind: symbol.KindChannel,
				Name: "wave",
				Type: types.Chan(types.Series(types.F64())),
			},
		}
		declareIn := func(bCtx SpecContext, root, scope *symbol.Symbol, code string) {
			GinkgoHelper()
			stmt := MustSucceed(parser.ParseStatement(code))
			ctx := context.NewRoot(bCtx, stmt.VariableDeclaration(), root)
			ctx.Scope = scope
			statement.AnalyzeVariableDeclaration(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		}
		assignIn := func(
			bCtx SpecContext, root, scope *symbol.Symbol, code string,
		) context.Context[parser.IAssignmentContext] {
			GinkgoHelper()
			stmt := MustSucceed(parser.ParseStatement(code))
			ctx := context.NewRoot(bCtx, stmt.Assignment(), root)
			ctx.Scope = scope
			statement.AnalyzeAssignment(ctx)
			return ctx
		}
		newStage := func(bCtx SpecContext, root *symbol.Symbol) *symbol.Symbol {
			GinkgoHelper()
			return MustSucceed(root.Add(bCtx, symbol.Symbol{
				Name: "s1", Kind: symbol.KindStage,
			}))
		}

		It(
			"Should re-point a channel-read variable at a new derivation",
			func(bCtx SpecContext) {
				root := NewRoot(nil, channels...)
				stage := newStage(bCtx, root)
				declareIn(bCtx, root, stage, "x := sensor + 1")
				ctx := assignIn(bCtx, root, stage, "x = sensor * 2")
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(MustSucceed(ctx.Scope.Resolve(ctx, "x")).Reassigned).To(BeTrue())
			},
		)

		It(
			"Should reject re-pointing a channel-read variable at another structure",
			func(bCtx SpecContext) {
				root := NewRoot(nil, channels...)
				stage := newStage(bCtx, root)
				declareIn(bCtx, root, stage, "x := sensor + 1")
				ctx := assignIn(bCtx, root, stage, "x = [1.0, 2.0]")
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(
					ContainSubstring("type mismatch: cannot reassign"),
				)
				Expect(
					MustSucceed(ctx.Scope.Resolve(ctx, "x")).Reassigned,
				).To(BeFalse())
			},
		)

		It(
			"Should reject re-pointing a channel-read variable at a constant",
			func(bCtx SpecContext) {
				root := NewRoot(nil, channels...)
				stage := newStage(bCtx, root)
				declareIn(bCtx, root, stage, "x := sensor + 1")
				ctx := assignIn(bCtx, root, stage, "x = 5")
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(
					ContainSubstring("from a constant value"),
				)
				Expect(
					MustSucceed(ctx.Scope.Resolve(ctx, "x")).Reassigned,
				).To(BeFalse())
			},
		)

		It(
			"Should reject re-pointing a channel-read variable at a cast constant",
			func(bCtx SpecContext) {
				root := NewRoot(nil, channels...)
				stage := newStage(bCtx, root)
				declareIn(bCtx, root, stage, "x := sensor + 1")
				ctx := assignIn(bCtx, root, stage, "x = f32(5)")
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(
					ContainSubstring("from a constant value"),
				)
				Expect(
					MustSucceed(ctx.Scope.Resolve(ctx, "x")).Reassigned,
				).To(BeFalse())
			},
		)

		It(
			"Should accept re-pointing a channel-read variable at a variable expression",
			func(bCtx SpecContext) {
				root := NewRoot(nil, channels...)
				stage := newStage(bCtx, root)
				declareIn(bCtx, root, stage, "base := sensor + 1")
				declareIn(bCtx, root, stage, "x := sensor + 2")
				ctx := assignIn(bCtx, root, stage, "x = base + 1")
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(MustSucceed(ctx.Scope.Resolve(ctx, "x")).Reassigned).To(BeTrue())
			},
		)

		It("Should reject reassigning a top-level variable", func(bCtx SpecContext) {
			root := NewRoot(nil, channels...)
			declareIn(bCtx, root, root, "gain := 2")
			stage := newStage(bCtx, root)
			ctx := assignIn(bCtx, root, stage, "gain = 3")
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(
				ContainSubstring("cannot reassign top-level variable 'gain'"),
			)
			ctx = assignIn(bCtx, root, stage, "gain += 1")
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(
				ContainSubstring("cannot reassign top-level variable 'gain'"),
			)
		})

		It(
			"Should reject re-pointing a top-level channel-read variable",
			func(bCtx SpecContext) {
				root := NewRoot(nil, channels...)
				declareIn(bCtx, root, root, "x := sensor + 1")
				stage := newStage(bCtx, root)
				ctx := assignIn(bCtx, root, stage, "x = sensor * 2")
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(
					ContainSubstring("cannot reassign top-level variable 'x'"),
				)
				Expect(
					MustSucceed(ctx.Scope.Resolve(ctx, "x")).Reassigned,
				).To(BeFalse())
			},
		)
	})

	Describe("If Statement", func() {
		DescribeTable(
			"valid if statements",
			func(bCtx SpecContext, code string) {
				stmt := MustSucceed(parser.ParseStatement(code))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			},
			Entry("simple if", `if true { x := 42 }`),
			Entry(
				"if-else chain",
				`if false { x := 1 } else if true { y := 2 } else { z := 3 }`,
			),
			Entry("nested blocks", `if true {
				x := 42
				if true { y := x + 1 }
			}`),
		)

		It("should detect undefined variable in condition", func(bCtx SpecContext) {
			stmt := MustSucceed(parser.ParseStatement(`if x > 10 { y := 1 }`))
			ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
			statement.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect(
				(*ctx.Diagnostics)[0].Message,
			).To(ContainSubstring("undefined symbol: x"))
		})

		DescribeTable(
			"numeric condition deprecation warnings",
			func(bCtx SpecContext, code string, warningCount int) {
				stmt := MustSucceed(parser.ParseStatement(code))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				warnings := testutil.Warnings(ctx.Diagnostics)
				Expect(warnings).To(HaveLen(warningCount))
				for _, w := range warnings {
					Expect(w.Message).To(Equal(
						"numeric conditions are deprecated; use an explicit " +
							"comparison like x != 0",
					))
				}
			},
			Entry("integer literal condition", `if 1 { x := 42 }`, 1),
			Entry("float literal condition", `if 1.5 { x := 42 }`, 1),
			Entry(
				"numeric else-if condition",
				`if false { x := 1 } else if 2 { y := 2 }`,
				1,
			),
			Entry(
				"numeric if and else-if conditions",
				`if 1 { x := 1 } else if 2 { y := 2 }`,
				2,
			),
			Entry("bool literal condition", `if true { x := 42 }`, 0),
			Entry("comparison condition", `if 1 > 0 { x := 42 }`, 0),
		)

		It("should warn on a numeric variable condition", func(bCtx SpecContext) {
			block := MustSucceed(parser.ParseBlock(`{
				x := 1
				if x { y := 2 }
			}`))
			ctx := context.NewRoot(bCtx, block, NewRoot(nil))
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(testutil.Warnings(ctx.Diagnostics)).To(HaveLen(1))
		})

		It("should not warn on a bool variable condition", func(bCtx SpecContext) {
			block := MustSucceed(parser.ParseBlock(`{
				x := true
				if x { y := 2 }
			}`))
			ctx := context.NewRoot(bCtx, block, NewRoot(nil))
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})
	})

	Describe("Block", func() {
		DescribeTable("block analysis",
			func(bCtx SpecContext, code string, expectOk bool, errorSubstring string) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(Equal(expectOk))
				if !expectOk && errorSubstring != "" {
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring(errorSubstring))
				} else if expectOk {
					Expect(*ctx.Diagnostics).To(BeEmpty())
				}
			},
			Entry("multiple statements", `{
				x := 1
				y := 2
				z := x + y
			}`, true, ""),
			Entry("variable visibility within block", `{
				x := 1
				y := x + 2
				z := x + y
			}`, true, ""),
			Entry("error with undefined symbol", `{
				x := 1
				y := undefined
			}`, false, "undefined symbol: undefined"),
		)
	})

	Describe("Expression Statement", func() {
		It(
			"should analyze standalone expression with existing variable",
			func(bCtx SpecContext) {
				stmt := MustSucceed(parser.ParseStatement(`x + 1`))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
					Name: "x",
					Kind: symbol.KindVariable,
					Type: types.I64(),
				}))
				Expect(scope).ToNot(BeNil())
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			},
		)

		It("should detect errors in standalone expression", func(bCtx SpecContext) {
			stmt := MustSucceed(parser.ParseStatement(`undefined_var + 1`))
			ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
			statement.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect(
				(*ctx.Diagnostics)[0].Message,
			).To(ContainSubstring("undefined symbol"))
		})
	})

	Describe("Channel Operations", func() {
		var channels []symbol.Symbol

		BeforeEach(func() {
			channels = []symbol.Symbol{
				{
					Kind: symbol.KindChannel,
					Name: "sensor",
					Type: types.Chan(types.F64()),
				},
				{
					Kind: symbol.KindChannel,
					Name: ir.DefaultOutputParam,
					Type: types.Chan(types.F64()),
				},
				{
					Kind: symbol.KindChannel,
					Name: "int_chan",
					Type: types.Chan(types.I32()),
				},
				{
					Kind: symbol.KindChannel,
					Name: "string_chan",
					Type: types.Chan(types.String()),
				},
			}
		})

		setupChannelFunctionContext := func(ctx context.Context[parser.IBlockContext]) {
			GinkgoHelper()
			scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "testFunc",
				Kind: symbol.KindFunction,
			}))
			Expect(scope).ToNot(BeNil())
			fn := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
			ctx.Scope = fn
		}

		Context("channel assignment in imperative context", func() {
			DescribeTable(
				"channel write validation",
				func(bCtx SpecContext, code string, expectOk bool, errorSubstring string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot[parser.IBlockContext](
						bCtx,
						block,
						NewRoot(nil, channels...),
					)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(
						ctx.Diagnostics.Ok(),
					).To(Equal(expectOk), ctx.Diagnostics.String())
					if !expectOk && errorSubstring != "" {
						Expect(
							(*ctx.Diagnostics)[0].Message,
						).To(ContainSubstring(errorSubstring))
					}
				},
				Entry("literal value", `{ output = 42.0 }`, true, ""),
				Entry("variable value", `{
					value := 42.0
					output = value
				}`, true, ""),
				Entry("type mismatch", `{ output = "hello" }`, false, "type mismatch"),
			)

			DescribeTable("should reject a series written to a scalar channel",
				func(bCtx SpecContext, value string) {
					block := MustSucceed(parser.ParseBlock(
						"{\ndata series f64 := [1.0, 2.0, 3.0]\noutput = " + value + "\n}",
					))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, channels...))
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.String()).To(ContainSubstring(
						"cannot write series f64 to channel 'output' (type f64)",
					))
				},
				Entry("a slice", "data[1:3]"),
				Entry("an open slice", "data[1:]"),
				Entry("a slice of a slice", "data[0:3][1:2]"),
				Entry("the series", "data"),
			)

			DescribeTable("should accept a scalar written to a scalar channel",
				func(bCtx SpecContext, value string) {
					block := MustSucceed(parser.ParseBlock(
						"{\ndata series f64 := [1.0, 2.0, 3.0]\noutput = " + value + "\n}",
					))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, channels...))
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("an index", "data[1]"),
				Entry("an index of a slice", "data[1:3][0]"),
				Entry("a channel read", "sensor"),
			)

			It("should detect undefined channel in assignment", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{ undefined_channel = 42.0 }`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				setupChannelFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined"))
			})
		})

		Context("channel reads in imperative context", func() {
			DescribeTable("channel read type inference",
				func(bCtx SpecContext, chanName string, expectedType types.Type) {
					code := "current := " + chanName
					stmt := MustSucceed(parser.ParseStatement(code))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil, channels...))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					Expect(*ctx.Diagnostics).To(BeEmpty())
					varScope := MustSucceed(ctx.Scope.Resolve(ctx, "current"))
					Expect(varScope.Type).To(Equal(expectedType))
				},
				Entry("f64 channel", "sensor", readWriteChan(types.F64())),
				Entry("i32 channel", "int_chan", readWriteChan(types.I32())),
				Entry("i32 channel addition", "int_chan + 1", types.I32()),
			)

			It(
				"should give a channel alias a source, not an internal channel",
				func(bCtx SpecContext) {
					stmt := MustSucceed(parser.ParseStatement("current := sensor"))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil, channels...))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					varScope := MustSucceed(ctx.Scope.Resolve(ctx, "current"))
					Expect(varScope.IsChannelReadWrite()).To(BeTrue())
					Expect(varScope.SourceID).ToNot(BeNil())
				},
			)
		})

		Context("channel read/write assignment to scalar variables", func() {
			DescribeTable(
				"should accept valid channel read/write to scalar assignments",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot[parser.IBlockContext](
						bCtx,
						block,
						NewRoot(nil, channels...),
					)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("chan f64 channel read/write assigned to f64 scalar", `{
					local_ref := sensor
					value f64 := 0.0
					value = local_ref
				}`),
				Entry("chan f64 channel read/write assigned to stateful f64 scalar", `{
					local_ref := sensor
					value f64 $= 0.0
					value = local_ref
				}`),
				Entry(
					"chan f64 channel read/write assigned to inferred-type variable",
					`{
					local_ref := sensor
					value := 0.0
					value = local_ref
				}`,
				),
				Entry("chan i32 channel read/write assigned to i32 scalar", `{
					local_ref := int_chan
					value i32 := 0
					value = local_ref
				}`),
				Entry("read/write-to-read/write preserves chan type", `{
					local_ref := sensor
					other_ref := local_ref
				}`),
				Entry("chan channel read/write in comparison", `{
					local_ref := sensor
					x f64 := 0.0
					if local_ref > 100.0 { x = 1.0 }
				}`),
				Entry("chan channel read/write in arithmetic", `{
					local_ref := sensor
					result f64 := local_ref + 1.0
				}`),
				Entry("chan channel read/write written to channel target", `{
					sensor_ref := sensor
					output = sensor_ref
				}`),
				Entry("arithmetic expr of channel read/write assigned to scalar", `{
					local_ref := sensor
					value f64 := 0.0
					value = local_ref * 2.0
				}`),
			)

			It(
				"should reject type mismatch after unwrapping channel read/write",
				func(bCtx SpecContext) {
					block := MustSucceed(parser.ParseBlock(`{
					local_ref := int_chan
					value f64 := 0.0
					value = local_ref
				}`))
					ctx := context.NewRoot[parser.IBlockContext](
						bCtx,
						block,
						NewRoot(nil, channels...),
					)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("type mismatch"))
				},
			)
		})

		Context("series literals with channel read/write elements", func() {
			DescribeTable(
				"should accept valid series literals containing channel read/write variables",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot[parser.IBlockContext](
						bCtx,
						block,
						NewRoot(nil, channels...),
					)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("channel read/write then exact-integer float literal", `{
					ref := sensor
					x := [ref, 1.0]
				}`),
				Entry("exact-integer float literal then channel read/write", `{
					ref := sensor
					x := [1.0, ref]
				}`),
				Entry("two channel read/write variables of same type", `{
					ref1 := sensor
					ref2 := sensor
					x := [ref1, ref2]
				}`),
				Entry("channel read/write then int literal", `{
					ref := sensor
					x := [ref, 1]
				}`),
				Entry("int literal then channel read/write", `{
					ref := sensor
					x := [1, ref]
				}`),
				Entry("channel read/write with arithmetic", `{
					ref := sensor
					x := [ref + 1.0, 2.0]
				}`),
				Entry("i32 channel read/write then int literal", `{
					ref := int_chan
					x := [ref, 42]
				}`),
				Entry("int literal then i32 channel read/write", `{
					ref := int_chan
					x := [42, ref]
				}`),
				Entry("channel read/write then non-exact float literal", `{
					ref := sensor
					x := [ref, 1.5]
				}`),
				Entry("non-exact float literal then channel read/write", `{
					ref := sensor
					x := [1.5, ref]
				}`),
				Entry("channel read/write then pi-like float literal", `{
					ref := sensor
					x := [ref, 3.14]
				}`),
			)

			DescribeTable(
				"should reject invalid series literals containing channel read/write variables",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot[parser.IBlockContext](
						bCtx,
						block,
						NewRoot(nil, channels...),
					)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("f64 channel read/write then string literal", `{
					ref := sensor
					x := [ref, "hello"]
				}`),
				Entry("string literal then f64 channel read/write", `{
					ref := sensor
					x := ["hello", ref]
				}`),
				Entry("f64 channel read/write and i32 channel read/write", `{
					f_ref := sensor
					i_ref := int_chan
					x := [f_ref, i_ref]
				}`),
				Entry("i32 channel read/write and f64 channel read/write", `{
					i_ref := int_chan
					f_ref := sensor
					x := [i_ref, f_ref]
				}`),
				Entry("non-exact float literal then i32 channel read/write", `{
					ref := int_chan
					x := [1.5, ref]
				}`),
			)
		})
	})

	Describe("Compound Assignment", func() {
		Context("valid compound assignments", func() {
			DescribeTable("numeric types",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					Expect(*ctx.Diagnostics).To(BeEmpty())
				},
				Entry("i64 plus equals", `{
					x i64 := 10
					x += 5
				}`),
				Entry("i32 minus equals", `{
					x i32 := 10
					x -= 5
				}`),
				Entry("f64 multiply equals", `{
					x f64 := 10.0
					x *= 2.5
				}`),
				Entry("f32 divide equals", `{
					x f32 := 10.0
					x /= 2.0
				}`),
				Entry("i64 modulo equals", `{
					x i64 := 17
					x %= 5
				}`),
				Entry("inferred type plus equals", `{
					x := 10
					x += 5
				}`),
				Entry("compound with expression", `{
					x i64 := 10
					y i64 := 5
					x += y * 2
				}`),
			)

			It("should allow string concatenation with +=", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
					s str := "hello"
					s += " world"
				}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			})
		})

		Context("invalid compound assignments", func() {
			var channels []symbol.Symbol

			BeforeEach(func() {
				channels = []symbol.Symbol{
					{
						Kind: symbol.KindChannel,
						Name: "sensor",
						Type: types.Chan(types.F64()),
					},
				}
			})

			It("should reject compound assignment on channels", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{ sensor += 1.0 }`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil, channels...))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("compound assignment not supported on channels"))
			})

			DescribeTable("strings only support +=",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("strings only support += operator"))
				},
				Entry("minus equals on string", `{
					s str := "hello"
					s -= "h"
				}`),
				Entry("multiply equals on string", `{
					s str := "hello"
					s *= "x"
				}`),
				Entry("divide equals on string", `{
					s str := "hello"
					s /= "x"
				}`),
				Entry("modulo equals on string", `{
					s str := "hello"
					s %= "x"
				}`),
			)

			It(
				"should reject compound assignment with type mismatch",
				func(bCtx SpecContext) {
					block := MustSucceed(parser.ParseBlock(`{
					x i32 := 10
					x += "hello"
				}`))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("type mismatch"))
				},
			)

			It(
				"Should reject compound assignments of inferred literal float vs. integer",
				func(bCtx SpecContext) {
					block := MustSucceed(parser.ParseBlock(`{
					x := 10
					y := x + 3.2
				}`))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("type mismatch"))
				},
			)

			It(
				"should reject compound assignment on undefined variable",
				func(bCtx SpecContext) {
					block := MustSucceed(parser.ParseBlock(`{ undefined_var += 5 }`))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("undefined symbol: undefined_var"))
				},
			)

			DescribeTable("should reject a series in a scalar compound assignment",
				func(bCtx SpecContext, statements string) {
					block := MustSucceed(parser.ParseBlock(
						"{\ndata series f64 := [1.0, 2.0, 3.0]\nx f64 := 1.0\n" +
							statements + "\n}",
					))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.String()).To(ContainSubstring(
						"cannot use series f64 in compound assignment to f64",
					))
				},
				Entry("a slice added", "x += data[1:3]"),
				Entry("a slice subtracted", "x -= data[1:3]"),
				Entry("an open slice", "x *= data[:2]"),
				Entry("the series", "x += data"),
			)

			DescribeTable("should reject a series in an indexed compound assignment",
				func(bCtx SpecContext, statements string) {
					block := MustSucceed(parser.ParseBlock(
						"{\ndata series f64 := [1.0, 2.0, 3.0]\n" + statements + "\n}",
					))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.String()).To(ContainSubstring(
						"cannot use series f64 in compound assignment to series " +
							"element of type f64",
					))
				},
				Entry("a slice added", "data[0] += data[1:3]"),
				Entry("a slice multiplied", "data[0] *= data[1:3]"),
				Entry("the series", "data[0] += data"),
			)

			DescribeTable("should accept an element in a compound assignment",
				func(bCtx SpecContext, statements string) {
					block := MustSucceed(parser.ParseBlock(
						"{\ndata series f64 := [1.0, 2.0, 3.0]\nx f64 := 1.0\n" +
							statements + "\n}",
					))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("an index into a scalar", "x += data[1]"),
				Entry("an index of a slice into a scalar", "x += data[1:3][0]"),
				Entry("an index of a slice into an element", "data[0] += data[1:3][0]"),
			)

			DescribeTable("valid indexed compound assignments",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					Expect(*ctx.Diagnostics).To(BeEmpty())
				},
				Entry("i32 plus equals", `{
					arr series i32 := [1, 2, 3]
					arr[0] += 5
				}`),
				Entry("i64 minus equals", `{
					arr series i64 := [10, 20, 30]
					arr[1] -= 5
				}`),
				Entry("f64 multiply equals", `{
					arr series f64 := [1.0, 2.0, 3.0]
					arr[2] *= 2.5
				}`),
				Entry("f32 divide equals", `{
					arr series f32 := [10.0, 20.0]
					arr[0] /= 2.0
				}`),
				Entry("i64 modulo equals", `{
					arr series i64 := [17, 23]
					arr[0] %= 5
				}`),
				Entry("with variable index", `{
					arr series i32 := [1, 2, 3]
					i i32 := 1
					arr[i] += 10
				}`),
				Entry("with expression index", `{
					arr series i32 := [1, 2, 3]
					arr[1 + 1] += 100
				}`),
				Entry("with variable value", `{
					arr series i32 := [1, 2, 3]
					delta i32 := 42
					arr[0] += delta
				}`),
			)

			It("should reject slice compound assignment", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
					arr series i32 := [1, 2, 3]
					arr[0:2] += 5
				}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("slice compound assignment not supported"))
			})

			It(
				"should reject indexed compound assignment on non-series type",
				func(bCtx SpecContext) {
					block := MustSucceed(parser.ParseBlock(`{
					x i32 := 5
					x[0] += 1
				}`))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("indexed"))
				},
			)

			// Whole-series compound assignment tests
			DescribeTable("valid whole-series compound assignments",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					Expect(*ctx.Diagnostics).To(BeEmpty())
				},
				Entry("series += scalar f64", `{
					s series f64 := [1.0, 2.0, 3.0]
					s += 5.0
				}`),
				Entry("series -= scalar i32", `{
					s series i32 := [10, 20, 30]
					s -= 5
				}`),
				Entry("series *= scalar", `{
					s series f64 := [1.0, 2.0]
					s *= 2.0
				}`),
				Entry("series /= scalar", `{
					s series f64 := [10.0, 20.0]
					s /= 2.0
				}`),
				Entry("series %= scalar", `{
					s series i64 := [17, 23]
					s %= 5
				}`),
				Entry("series += series", `{
					a series f64 := [1.0, 2.0]
					b series f64 := [10.0, 20.0]
					a += b
				}`),
				Entry("series -= series", `{
					a series i32 := [100, 200]
					b series i32 := [10, 20]
					a -= b
				}`),
				Entry("series *= series", `{
					a series f64 := [2.0, 3.0]
					b series f64 := [4.0, 5.0]
					a *= b
				}`),
			)

			It(
				"should reject series compound assignment with incompatible scalar type",
				func(bCtx SpecContext) {
					block := MustSucceed(parser.ParseBlock(`{
					s series i32 := [1, 2, 3]
					s += "hello"
				}`))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("type mismatch"))
				},
			)

			It(
				"should reject series compound assignment with mismatched series element types",
				func(bCtx SpecContext) {
					block := MustSucceed(parser.ParseBlock(`{
					a series i32 := [1, 2, 3]
					b series f64 := [1.0, 2.0, 3.0]
					a += b
				}`))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("type mismatch"))
				},
			)
		})
	})

	Describe("Mixed Type Scenarios", func() {
		It("should handle complex nested structures", func(bCtx SpecContext) {
			block := MustSucceed(parser.ParseBlock(`{
				x := 10
				y $= 20
				if x < y {
					z := x + y
					z = z * 2
				}
			}`))
			ctx := context.NewRoot(bCtx, block, NewRoot(nil))
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("should properly track types through assignments", func(bCtx SpecContext) {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 10
				y := x
				z := y + 5
			}`))
			ctx := context.NewRoot(bCtx, block, NewRoot(nil))
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It(
			"should return an error when assigning incompatible types",
			func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
				x i32 := 10
				y f32 := x
			}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("type mismatch: cannot assign i32 to 'y' (type f32)"))
			},
		)
	})

	Describe("Series Literals", func() {
		Context("valid series literals with literals only", func() {
			DescribeTable("should accept",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("integer literals", `{ x := [1, 2, 3] }`),
				Entry("float literals", `{ x := [1.0, 2.0, 3.0] }`),
				Entry("mixed int and float literals", `{ x := [1, 2.0, 3] }`),
				Entry("int then float literal", `{ x := [1, 2.0] }`),
				Entry("float then int literal", `{ x := [1.0, 2] }`),
				Entry("empty series", `{ x := [] }`),
				Entry("single element", `{ x := [42] }`),
				Entry("unary negation", `{ x := [-1, -2, -3] }`),
			)
		})

		Context("valid series literals with expressions", func() {
			DescribeTable("should accept",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("additive expressions", `{ x := [1 + 2, 3 + 4] }`),
				Entry("multiplicative expressions", `{ x := [2 * 3, 4 * 5] }`),
				Entry("mixed operators", `{ x := [1 + 2, 3 * 4, 10 - 5] }`),
				Entry("nested parentheses", `{ x := [(1 + 2) * 3, 4 ^ 2] }`),
				Entry("division", `{ x := [10 / 2, 20 / 4] }`),
				Entry("modulo", `{ x := [10 % 3, 20 % 7] }`),
			)
		})

		Context("valid series literals with variables", func() {
			DescribeTable("should accept",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("inferred type variables", `{
					a := 1
					b := 2
					x := [a, b]
				}`),
				Entry("typed i32 variables", `{
					a i32 := 1
					b i32 := 2
					x := [a, b]
				}`),
				Entry("typed i64 variables", `{
					a i64 := 1
					b i64 := 2
					x := [a, b]
				}`),
				Entry("typed f32 variables", `{
					a f32 := 1.0
					b f32 := 2.0
					x := [a, b]
				}`),
				Entry("typed f64 variables", `{
					a f64 := 1.0
					b f64 := 2.0
					x := [a, b]
				}`),
				Entry("three variables", `{
					a := 1
					b := 2
					c := 3
					x := [a, b, c]
				}`),
				Entry("variable expressions", `{
					a := 1
					b := 2
					x := [a + b, a - b, a * b]
				}`),
			)
		})

		Context("valid series literals mixing variables and literals", func() {
			DescribeTable("should accept",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("literal first then variable", `{
					a := 1
					x := [1, a]
				}`),
				Entry("variable first then literal", `{
					a := 1
					x := [a, 1]
				}`),
				Entry("typed variable then literal", `{
					a i32 := 1
					x := [a, 2]
				}`),
				Entry("literal then typed variable", `{
					a i32 := 1
					x := [2, a]
				}`),
				Entry("variable sandwiched by literals", `{
					a := 10
					x := [1, a, 100]
				}`),
				Entry("literal sandwiched by variables", `{
					a := 1
					b := 3
					x := [a, 2, b]
				}`),
				Entry("variable expression with literal", `{
					a := 10
					x := [a + 5, 20]
				}`),
				Entry("literal with variable expression", `{
					a := 10
					x := [5, a * 2]
				}`),
				Entry("inferred float variable and int literal", `{
					a := 12.0
					x := [a, 5]
				}`),
			)
		})

		Context("series literals with function calls", func() {
			fn := func(name string, t types.Type) symbol.Symbol {
				return symbol.Symbol{
					Name: name,
					Kind: symbol.KindFunction,
					Exec: symbol.ExecBoth,
					Type: types.Function(types.FunctionProperties{
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: t}},
					}),
				}
			}
			var funcs []symbol.Symbol

			BeforeEach(func() {
				funcs = []symbol.Symbol{
					fn("getI32", types.I32()),
					fn("anotherI32", types.I32()),
					fn("getI64", types.I64()),
					fn("getF32", types.F32()),
					fn("getF64", types.F64()),
					fn("getStr", types.String()),
				}
			})

			DescribeTable(
				"valid function call combinations",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, funcs...))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("two same-type function calls", `{ x := [getI32(), getI32()] }`),
				Entry(
					"two different functions same return type",
					`{ x := [getI32(), anotherI32()] }`,
				),
				Entry("function call then literal", `{ x := [getI32(), 42] }`),
				Entry("literal then function call", `{ x := [42, getI32()] }`),
				Entry("function call then expression", `{ x := [getI32(), 1 + 2] }`),
				Entry("expression then function call", `{ x := [1 + 2, getI32()] }`),
				Entry(
					"function call in expression then literal",
					`{ x := [getI32() + 1, 42] }`,
				),
				Entry(
					"literal then function call in expression",
					`{ x := [42, getI32() + 1] }`,
				),
				Entry(
					"function call plus literal in expression",
					`{ x := [1 + getI32(), 42] }`,
				),
				Entry(
					"three function calls",
					`{ x := [getI32(), getI32(), getI32()] }`,
				),
				Entry(
					"function call sandwiched by literals",
					`{ x := [1, getI32(), 2] }`,
				),
				Entry(
					"literal sandwiched by function calls",
					`{ x := [getI32(), 1, getI32()] }`,
				),
			)

			DescribeTable("valid function call with variable combinations",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, funcs...))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("function call then compatible variable", `{
					v i32 := 10
					x := [getI32(), v]
				}`),
				Entry("compatible variable then function call", `{
					v i32 := 10
					x := [v, getI32()]
				}`),
				Entry("function call then variable expression", `{
					v i32 := 10
					x := [getI32(), v + 1]
				}`),
				Entry("variable expression then function call", `{
					v i32 := 10
					x := [v + 1, getI32()]
				}`),
				Entry("function call in expression with variable", `{
					v i32 := 10
					x := [getI32() + v, 42]
				}`),
				Entry("variable in expression with function call", `{
					v i32 := 10
					x := [v + getI32(), 42]
				}`),
				Entry("three elements func var literal", `{
					v i32 := 10
					x := [getI32(), v, 42]
				}`),
				Entry("three elements var func literal", `{
					v i32 := 10
					x := [v, getI32(), 42]
				}`),
				Entry("three elements literal func var", `{
					v i32 := 10
					x := [42, getI32(), v]
				}`),
			)

			DescribeTable(
				"invalid function call combinations",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, funcs...))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 then f64 function calls", `{ x := [getI32(), getF64()] }`),
				Entry("f64 then i32 function calls", `{ x := [getF64(), getI32()] }`),
				Entry("i32 then i64 function calls", `{ x := [getI32(), getI64()] }`),
				Entry(
					"i32 then string function calls",
					`{ x := [getI32(), getStr()] }`,
				),
				Entry(
					"string then i32 function calls",
					`{ x := [getStr(), getI32()] }`,
				),
				Entry(
					"f32 then string function calls",
					`{ x := [getF32(), getStr()] }`,
				),
				Entry(
					"f64 then string function calls",
					`{ x := [getF64(), getStr()] }`,
				),
				Entry(
					"i32 function then string literal",
					`{ x := [getI32(), "hello"] }`,
				),
				Entry(
					"string literal then i32 function",
					`{ x := ["hello", getI32()] }`,
				),
				Entry("string function then int literal", `{ x := [getStr(), 42] }`),
				Entry("int literal then string function", `{ x := [42, getStr()] }`),
				Entry(
					"f64 function then string literal",
					`{ x := [getF64(), "hello"] }`,
				),
				Entry(
					"three functions last mismatched",
					`{ x := [getI32(), getI32(), getStr()] }`,
				),
				Entry(
					"three functions first mismatched",
					`{ x := [getStr(), getI32(), getI32()] }`,
				),
				Entry(
					"three functions middle mismatched",
					`{ x := [getI32(), getStr(), getI32()] }`,
				),
			)

			DescribeTable("invalid function call with variable combinations",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, funcs...))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 function then string variable", `{
					s str := "hello"
					x := [getI32(), s]
				}`),
				Entry("string variable then i32 function", `{
					s str := "hello"
					x := [s, getI32()]
				}`),
				Entry("f64 function then string variable", `{
					s str := "hello"
					x := [getF64(), s]
				}`),
				Entry("string function then i32 variable", `{
					v i32 := 10
					x := [getStr(), v]
				}`),
				Entry("i32 variable then string function", `{
					v i32 := 10
					x := [v, getStr()]
				}`),
				Entry("i32 function then f64 variable", `{
					v f64 := 10.0
					x := [getI32(), v]
				}`),
				Entry("f64 variable then i32 function", `{
					v f64 := 10.0
					x := [v, getI32()]
				}`),
				Entry("three elements func var mismatch", `{
					s str := "hello"
					x := [getI32(), 42, s]
				}`),
				Entry("three elements var func mismatch", `{
					s str := "hello"
					x := [s, getI32(), 42]
				}`),
				Entry("function expression then string", `{
					x := [getI32() + 1, "hello"]
				}`),
				Entry("string then function expression", `{
					x := ["hello", getI32() + 1]
				}`),
			)
		})

		Context("invalid series literals - type mismatches with literals", func() {
			DescribeTable("should reject",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("int then string", `{ x := [1, "hello"] }`),
				Entry("string then int", `{ x := ["hello", 1] }`),
				Entry("float then string", `{ x := [1.0, "hello"] }`),
				Entry("string then float", `{ x := ["hello", 1.0] }`),
				Entry("int int string", `{ x := [1, 2, "hello"] }`),
				Entry("string int int", `{ x := ["hello", 1, 2] }`),
				Entry("int string int", `{ x := [1, "hello", 2] }`),
			)
		})

		Context("invalid series literals - type mismatches with variables", func() {
			DescribeTable("should reject two mismatched variables",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 and str variables", `{
					a i32 := 1
					b str := "hello"
					x := [a, b]
				}`),
				Entry("str and i32 variables", `{
					a str := "hello"
					b i32 := 1
					x := [a, b]
				}`),
				Entry("f64 and str variables", `{
					a f64 := 1.0
					b str := "hello"
					x := [a, b]
				}`),
				Entry("i32 and f64 variables", `{
					a i32 := 1
					b f64 := 2.0
					x := [a, b]
				}`),
				Entry("f64 and i32 variables", `{
					a f64 := 1.0
					b i32 := 2
					x := [a, b]
				}`),
				Entry("i32 and f32 variables", `{
					a i32 := 1
					b f32 := 2.0
					x := [a, b]
				}`),
				Entry("i64 and f64 variables", `{
					a i64 := 1
					b f64 := 2.0
					x := [a, b]
				}`),
				// Inferred type variables - these should also be rejected for
				// consistency with explicit type annotations above
				Entry("inferred int and inferred float variables", `{
					a := 5
					b := 12.0
					x := [a, b]
				}`),
				Entry("inferred float and inferred int variables", `{
					a := 12.0
					b := 5
					x := [a, b]
				}`),
				Entry("inferred int variable and non-exact-integer float literal", `{
					a := 5
					x := [a, 12.5]
				}`),
			)
		})

		Context("invalid series literals - variable and literal mismatches", func() {
			DescribeTable("should reject",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 variable then string literal", `{
					a i32 := 1
					x := [a, "hello"]
				}`),
				Entry("string literal then i32 variable", `{
					a i32 := 1
					x := ["hello", a]
				}`),
				Entry("str variable then int literal", `{
					a str := "hello"
					x := [a, 1]
				}`),
				Entry("int literal then str variable", `{
					a str := "hello"
					x := [1, a]
				}`),
				Entry("f64 variable then string literal", `{
					a f64 := 1.0
					x := [a, "hello"]
				}`),
				Entry("string literal then f64 variable", `{
					a f64 := 1.0
					x := ["hello", a]
				}`),
			)
		})

		Context(
			"invalid series literals - three or more elements with mismatch",
			func() {
				DescribeTable("should reject",
					func(bCtx SpecContext, code string) {
						block := MustSucceed(parser.ParseBlock(code))
						ctx := context.NewRoot(bCtx, block, NewRoot(nil))
						statement.AnalyzeBlock(ctx)
						Expect(ctx.Diagnostics.Ok()).To(BeFalse())
						Expect(
							(*ctx.Diagnostics)[0].Message,
						).To(ContainSubstring("incompatible type"))
					},
					Entry("two ints then string variable", `{
					s str := "hello"
					x := [1, 2, s]
				}`),
					Entry("string variable then two ints", `{
					s str := "hello"
					x := [s, 1, 2]
				}`),
					Entry("int variable string int", `{
					a i32 := 1
					x := [a, "hello", 2]
				}`),
					Entry("three variables last mismatched", `{
					a i32 := 1
					b i32 := 2
					c str := "hello"
					x := [a, b, c]
				}`),
					Entry("three variables first mismatched", `{
					a str := "hello"
					b i32 := 1
					c i32 := 2
					x := [a, b, c]
				}`),
					Entry("three variables middle mismatched", `{
					a i32 := 1
					b str := "hello"
					c i32 := 2
					x := [a, b, c]
				}`),
				)
			},
		)

		Context("invalid series literals - expression mismatches", func() {
			DescribeTable("should reject",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("int expression then string", `{ x := [1 + 2, "hello"] }`),
				Entry("string then int expression", `{ x := ["hello", 1 + 2] }`),
				Entry("variable expression then string", `{
					a i32 := 1
					x := [a + 1, "hello"]
				}`),
				Entry("string then variable expression", `{
					a i32 := 1
					x := ["hello", a + 1]
				}`),
			)
		})

		Context("series assignment compatibility", func() {
			DescribeTable("should reject structural mismatches",
				func(bCtx SpecContext, code, errorSubstring string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					setupFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring(errorSubstring))
				},
				Entry("series to scalar variable", `{
					x := 1
					x = [1, 2, 3]
				}`, "type mismatch"),
				Entry("scalar to series variable", `{
					x := [1, 2, 3]
					x = 42
				}`, "type mismatch"),
				Entry("series literal to typed scalar", `{
					x i32 := [1, 2, 3]
				}`, "type mismatch"),
				Entry("series variable to scalar variable", `{
					a := [1, 2, 3]
					b := 1
					b = a
				}`, "type mismatch"),
				Entry("scalar variable to series variable", `{
					a := 1
					b := [1, 2, 3]
					b = a
				}`, "type mismatch"),
			)
		})
	})

	Describe("Series Literals", func() {
		Context("valid series literals with literals only", func() {
			DescribeTable("should accept",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("integer literals", `{ x := [1, 2, 3] }`),
				Entry("float literals", `{ x := [1.0, 2.0, 3.0] }`),
				Entry("mixed int and float literals", `{ x := [1, 2.0, 3] }`),
				Entry("int then float literal", `{ x := [1, 2.0] }`),
				Entry("float then int literal", `{ x := [1.0, 2] }`),
				Entry("empty series", `{ x := [] }`),
				Entry("single element", `{ x := [42] }`),
				Entry("unary negation", `{ x := [-1, -2, -3] }`),
			)
		})

		Context("valid series literals with expressions", func() {
			DescribeTable("should accept",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("additive expressions", `{ x := [1 + 2, 3 + 4] }`),
				Entry("multiplicative expressions", `{ x := [2 * 3, 4 * 5] }`),
				Entry("mixed operators", `{ x := [1 + 2, 3 * 4, 10 - 5] }`),
				Entry("nested parentheses", `{ x := [(1 + 2) * 3, 4 ^ 2] }`),
				Entry("division", `{ x := [10 / 2, 20 / 4] }`),
				Entry("modulo", `{ x := [10 % 3, 20 % 7] }`),
			)
		})

		Context("valid series literals with variables", func() {
			DescribeTable("should accept",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("inferred type variables", `{
					a := 1
					b := 2
					x := [a, b]
				}`),
				Entry("typed i32 variables", `{
					a i32 := 1
					b i32 := 2
					x := [a, b]
				}`),
				Entry("typed i64 variables", `{
					a i64 := 1
					b i64 := 2
					x := [a, b]
				}`),
				Entry("typed f32 variables", `{
					a f32 := 1.0
					b f32 := 2.0
					x := [a, b]
				}`),
				Entry("typed f64 variables", `{
					a f64 := 1.0
					b f64 := 2.0
					x := [a, b]
				}`),
				Entry("three variables", `{
					a := 1
					b := 2
					c := 3
					x := [a, b, c]
				}`),
				Entry("variable expressions", `{
					a := 1
					b := 2
					x := [a + b, a - b, a * b]
				}`),
			)
		})

		Context("valid series literals mixing variables and literals", func() {
			DescribeTable("should accept",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("literal first then variable", `{
					a := 1
					x := [1, a]
				}`),
				Entry("variable first then literal", `{
					a := 1
					x := [a, 1]
				}`),
				Entry("typed variable then literal", `{
					a i32 := 1
					x := [a, 2]
				}`),
				Entry("literal then typed variable", `{
					a i32 := 1
					x := [2, a]
				}`),
				Entry("variable sandwiched by literals", `{
					a := 10
					x := [1, a, 100]
				}`),
				Entry("literal sandwiched by variables", `{
					a := 1
					b := 3
					x := [a, 2, b]
				}`),
				Entry("variable expression with literal", `{
					a := 10
					x := [a + 5, 20]
				}`),
				Entry("literal with variable expression", `{
					a := 10
					x := [5, a * 2]
				}`),
				// Inferred type variables with literals - int/float literals can coerce
				Entry("inferred int variable and float literal", `{
					a := 5
					x := [a, 12.0]
				}`),
				Entry("inferred float variable and int literal", `{
					a := 12.0
					x := [a, 5]
				}`),
			)
		})

		Context("series literals with function calls", func() {
			fn := func(name string, t types.Type) symbol.Symbol {
				return symbol.Symbol{
					Name: name,
					Kind: symbol.KindFunction,
					Exec: symbol.ExecBoth,
					Type: types.Function(types.FunctionProperties{
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: t}},
					}),
				}
			}
			var funcs []symbol.Symbol

			BeforeEach(func() {
				funcs = []symbol.Symbol{
					fn("getI32", types.I32()),
					fn("anotherI32", types.I32()),
					fn("getI64", types.I64()),
					fn("getF32", types.F32()),
					fn("getF64", types.F64()),
					fn("getStr", types.String()),
				}
			})

			DescribeTable(
				"valid function call combinations",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, funcs...))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("two same-type function calls", `{ x := [getI32(), getI32()] }`),
				Entry(
					"two different functions same return type",
					`{ x := [getI32(), anotherI32()] }`,
				),
				Entry("function call then literal", `{ x := [getI32(), 42] }`),
				Entry("literal then function call", `{ x := [42, getI32()] }`),
				Entry("function call then expression", `{ x := [getI32(), 1 + 2] }`),
				Entry("expression then function call", `{ x := [1 + 2, getI32()] }`),
				Entry(
					"function call in expression then literal",
					`{ x := [getI32() + 1, 42] }`,
				),
				Entry(
					"literal then function call in expression",
					`{ x := [42, getI32() + 1] }`,
				),
				Entry(
					"function call plus literal in expression",
					`{ x := [1 + getI32(), 42] }`,
				),
				Entry(
					"three function calls",
					`{ x := [getI32(), getI32(), getI32()] }`,
				),
				Entry(
					"function call sandwiched by literals",
					`{ x := [1, getI32(), 2] }`,
				),
				Entry(
					"literal sandwiched by function calls",
					`{ x := [getI32(), 1, getI32()] }`,
				),
			)

			DescribeTable("valid function call with variable combinations",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, funcs...))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("function call then compatible variable", `{
					v i32 := 10
					x := [getI32(), v]
				}`),
				Entry("compatible variable then function call", `{
					v i32 := 10
					x := [v, getI32()]
				}`),
				Entry("function call then variable expression", `{
					v i32 := 10
					x := [getI32(), v + 1]
				}`),
				Entry("variable expression then function call", `{
					v i32 := 10
					x := [v + 1, getI32()]
				}`),
				Entry("function call in expression with variable", `{
					v i32 := 10
					x := [getI32() + v, 42]
				}`),
				Entry("variable in expression with function call", `{
					v i32 := 10
					x := [v + getI32(), 42]
				}`),
				Entry("three elements func var literal", `{
					v i32 := 10
					x := [getI32(), v, 42]
				}`),
				Entry("three elements var func literal", `{
					v i32 := 10
					x := [v, getI32(), 42]
				}`),
				Entry("three elements literal func var", `{
					v i32 := 10
					x := [42, getI32(), v]
				}`),
			)

			DescribeTable(
				"invalid function call combinations",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, funcs...))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 then f64 function calls", `{ x := [getI32(), getF64()] }`),
				Entry("f64 then i32 function calls", `{ x := [getF64(), getI32()] }`),
				Entry("i32 then i64 function calls", `{ x := [getI32(), getI64()] }`),
				Entry(
					"i32 then string function calls",
					`{ x := [getI32(), getStr()] }`,
				),
				Entry(
					"string then i32 function calls",
					`{ x := [getStr(), getI32()] }`,
				),
				Entry(
					"f32 then string function calls",
					`{ x := [getF32(), getStr()] }`,
				),
				Entry(
					"f64 then string function calls",
					`{ x := [getF64(), getStr()] }`,
				),
				Entry(
					"i32 function then string literal",
					`{ x := [getI32(), "hello"] }`,
				),
				Entry(
					"string literal then i32 function",
					`{ x := ["hello", getI32()] }`,
				),
				Entry("string function then int literal", `{ x := [getStr(), 42] }`),
				Entry("int literal then string function", `{ x := [42, getStr()] }`),
				Entry(
					"f64 function then string literal",
					`{ x := [getF64(), "hello"] }`,
				),
				Entry(
					"three functions last mismatched",
					`{ x := [getI32(), getI32(), getStr()] }`,
				),
				Entry(
					"three functions first mismatched",
					`{ x := [getStr(), getI32(), getI32()] }`,
				),
				Entry(
					"three functions middle mismatched",
					`{ x := [getI32(), getStr(), getI32()] }`,
				),
			)

			DescribeTable("invalid function call with variable combinations",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil, funcs...))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 function then string variable", `{
					s str := "hello"
					x := [getI32(), s]
				}`),
				Entry("string variable then i32 function", `{
					s str := "hello"
					x := [s, getI32()]
				}`),
				Entry("f64 function then string variable", `{
					s str := "hello"
					x := [getF64(), s]
				}`),
				Entry("string function then i32 variable", `{
					v i32 := 10
					x := [getStr(), v]
				}`),
				Entry("i32 variable then string function", `{
					v i32 := 10
					x := [v, getStr()]
				}`),
				Entry("i32 function then f64 variable", `{
					v f64 := 10.0
					x := [getI32(), v]
				}`),
				Entry("f64 variable then i32 function", `{
					v f64 := 10.0
					x := [v, getI32()]
				}`),
				Entry("three elements func var mismatch", `{
					s str := "hello"
					x := [getI32(), 42, s]
				}`),
				Entry("three elements var func mismatch", `{
					s str := "hello"
					x := [s, getI32(), 42]
				}`),
				Entry("function expression then string", `{
					x := [getI32() + 1, "hello"]
				}`),
				Entry("string then function expression", `{
					x := ["hello", getI32() + 1]
				}`),
			)
		})

		Context("invalid series literals - type mismatches with literals", func() {
			DescribeTable("should reject",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("int then string", `{ x := [1, "hello"] }`),
				Entry("string then int", `{ x := ["hello", 1] }`),
				Entry("float then string", `{ x := [1.0, "hello"] }`),
				Entry("string then float", `{ x := ["hello", 1.0] }`),
				Entry("int int string", `{ x := [1, 2, "hello"] }`),
				Entry("string int int", `{ x := ["hello", 1, 2] }`),
				Entry("int string int", `{ x := [1, "hello", 2] }`),
			)
		})

		Context("invalid series literals - type mismatches with variables", func() {
			DescribeTable("should reject two mismatched variables",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 and str variables", `{
					a i32 := 1
					b str := "hello"
					x := [a, b]
				}`),
				Entry("str and i32 variables", `{
					a str := "hello"
					b i32 := 1
					x := [a, b]
				}`),
				Entry("f64 and str variables", `{
					a f64 := 1.0
					b str := "hello"
					x := [a, b]
				}`),
				Entry("i32 and f64 variables", `{
					a i32 := 1
					b f64 := 2.0
					x := [a, b]
				}`),
				Entry("f64 and i32 variables", `{
					a f64 := 1.0
					b i32 := 2
					x := [a, b]
				}`),
				Entry("i32 and f32 variables", `{
					a i32 := 1
					b f32 := 2.0
					x := [a, b]
				}`),
				Entry("i64 and f64 variables", `{
					a i64 := 1
					b f64 := 2.0
					x := [a, b]
				}`),
				// Inferred type variables - these should also be rejected for
				// consistency with explicit type annotations above
				Entry("inferred int and inferred float variables", `{
					a := 5
					b := 12.0
					x := [a, b]
				}`),
				Entry("inferred float and inferred int variables", `{
					a := 12.0
					b := 5
					x := [a, b]
				}`),
			)
		})

		Context("invalid series literals - variable and literal mismatches", func() {
			DescribeTable("should reject",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 variable then string literal", `{
					a i32 := 1
					x := [a, "hello"]
				}`),
				Entry("string literal then i32 variable", `{
					a i32 := 1
					x := ["hello", a]
				}`),
				Entry("str variable then int literal", `{
					a str := "hello"
					x := [a, 1]
				}`),
				Entry("int literal then str variable", `{
					a str := "hello"
					x := [1, a]
				}`),
				Entry("f64 variable then string literal", `{
					a f64 := 1.0
					x := [a, "hello"]
				}`),
				Entry("string literal then f64 variable", `{
					a f64 := 1.0
					x := ["hello", a]
				}`),
			)
		})

		Context(
			"invalid series literals - three or more elements with mismatch",
			func() {
				DescribeTable("should reject",
					func(bCtx SpecContext, code string) {
						block := MustSucceed(parser.ParseBlock(code))
						ctx := context.NewRoot(bCtx, block, NewRoot(nil))
						statement.AnalyzeBlock(ctx)
						Expect(ctx.Diagnostics.Ok()).To(BeFalse())
						Expect(
							(*ctx.Diagnostics)[0].Message,
						).To(ContainSubstring("incompatible type"))
					},
					Entry("two ints then string variable", `{
					s str := "hello"
					x := [1, 2, s]
				}`),
					Entry("string variable then two ints", `{
					s str := "hello"
					x := [s, 1, 2]
				}`),
					Entry("int variable string int", `{
					a i32 := 1
					x := [a, "hello", 2]
				}`),
					Entry("three variables last mismatched", `{
					a i32 := 1
					b i32 := 2
					c str := "hello"
					x := [a, b, c]
				}`),
					Entry("three variables first mismatched", `{
					a str := "hello"
					b i32 := 1
					c i32 := 2
					x := [a, b, c]
				}`),
					Entry("three variables middle mismatched", `{
					a i32 := 1
					b str := "hello"
					c i32 := 2
					x := [a, b, c]
				}`),
				)
			},
		)

		Context("invalid series literals - expression mismatches", func() {
			DescribeTable("should reject",
				func(bCtx SpecContext, code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring("incompatible type"))
				},
				Entry("int expression then string", `{ x := [1 + 2, "hello"] }`),
				Entry("string then int expression", `{ x := ["hello", 1 + 2] }`),
				Entry("variable expression then string", `{
					a i32 := 1
					x := [a + 1, "hello"]
				}`),
				Entry("string then variable expression", `{
					a i32 := 1
					x := ["hello", a + 1]
				}`),
			)
		})

		Context("series assignment compatibility", func() {
			DescribeTable("should reject structural mismatches",
				func(bCtx SpecContext, code, errorSubstring string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					setupFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(
						(*ctx.Diagnostics)[0].Message,
					).To(ContainSubstring(errorSubstring))
				},
				Entry("series to scalar variable", `{
					x := 1
					x = [1, 2, 3]
				}`, "type mismatch"),
				Entry("scalar to series variable", `{
					x := [1, 2, 3]
					x = 42
				}`, "type mismatch"),
				Entry("series literal to typed scalar", `{
					x i32 := [1, 2, 3]
				}`, "type mismatch"),
				Entry("series variable to scalar variable", `{
					a := [1, 2, 3]
					b := 1
					b = a
				}`, "type mismatch"),
				Entry("scalar variable to series variable", `{
					a := 1
					b := [1, 2, 3]
					b = a
				}`, "type mismatch"),
			)
		})
	})

	Describe("Indexed Assignment", func() {
		It("should allow indexed assignment to series", func(bCtx SpecContext) {
			block := MustSucceed(parser.ParseBlock(`{
				data series i64 := [1, 2, 3]
				data[0] = 10
			}`))
			ctx := context.NewRoot(bCtx, block, NewRoot(nil))
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It(
			"should detect indexed assignment on non-series type",
			func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
				x i64 := 42
				x[0] = 10
			}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				setupFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("indexed assignment only supported on series types"))
			},
		)

		DescribeTable("should reject a non-integer index",
			func(bCtx SpecContext, stmt string) {
				block := MustSucceed(parser.ParseBlock(`{
				data series i64 := [1, 2, 3]
				i f32 := 1.0
				` + stmt + `
			}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				setupFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(
					"series index must be an integer type, got f32",
				))
			},
			Entry("indexed assignment", "data[i] = 10"),
			Entry("indexed compound assignment", "data[i] += 10"),
		)

		It("should detect slice assignment (not supported)", func(bCtx SpecContext) {
			block := MustSucceed(parser.ParseBlock(`{
				data series i64 := [1, 2, 3]
				data[0:2] = 10
			}`))
			ctx := context.NewRoot(bCtx, block, NewRoot(nil))
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(
				(*ctx.Diagnostics)[0].Message,
			).To(ContainSubstring("slice assignment not supported"))
		})

		It("should detect type mismatch in indexed assignment", func(bCtx SpecContext) {
			block := MustSucceed(parser.ParseBlock(`{
				data series i64 := [1, 2, 3]
				data[0] = "hello"
			}`))
			ctx := context.NewRoot(bCtx, block, NewRoot(nil))
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})

		DescribeTable("should reject a series assigned to a series element",
			func(bCtx SpecContext, value string) {
				block := MustSucceed(parser.ParseBlock(
					"{\ndata series f64 := [1.0, 2.0, 3.0]\ndata[0] = " + value + "\n}",
				))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				setupFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.String()).To(ContainSubstring(
					"cannot assign series f64 to series element of type f64",
				))
			},
			Entry("a slice", "data[1:3]"),
			Entry("an open slice", "data[1:]"),
			Entry("the series", "data"),
		)

		It("should accept an index of a slice as a series element",
			func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
				data series f64 := [1.0, 2.0, 3.0]
				data[0] = data[1:3][0]
			}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				setupFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			},
		)
	})

	Describe("For Loops", func() {
		Context("range loops", func() {
			It("should analyze range with 1 argument", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(10) { x := i }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				loopScope := ctx.Scope.Children()[0]
				sym := MustSucceed(loopScope.Resolve(ctx, "i"))
				Expect(sym.Kind).To(Equal(symbol.KindLoopVariable))
				limitSym := MustSucceed(loopScope.Resolve(ctx, "__for_limit"))
				Expect(limitSym.Kind).To(Equal(symbol.KindVariable))
			})

			It("should analyze range with 2 arguments", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(5, 10) { x := i }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})

			It("should analyze range with 3 arguments", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(0, 10, 2) { x := i }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				loopScope := ctx.Scope.Children()[0]
				stepSym := MustSucceed(loopScope.Resolve(ctx, "__for_step"))
				Expect(stepSym.Kind).To(Equal(symbol.KindVariable))
			})

			It(
				"should analyze range with explicit integer type",
				func(bCtx SpecContext) {
					stmt := MustSucceed(
						parser.ParseStatement(`for i := range(i32(10)) { x := i }`),
					)
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					loopScope := ctx.Scope.Children()[0]
					sym := MustSucceed(loopScope.Resolve(ctx, "i"))
					Expect(sym.Type).To(Equal(types.I32()))
				},
			)

			It("should reject range with no arguments", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range() { x := i }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("range()"))
			})

			It("should reject range with 4 arguments", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(1, 2, 3, 4) { x := i }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("range()"))
			})

			It("should reject an index after range", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(10)[1:] { x := i }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).
					To(Equal("undefined symbol: range"))
			})

			It("should reject range with float arguments", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(f64(3.14)) { x := i }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("integer type"))
			})

			It("should accept range with negative bounds", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(-5, 5) { x := i }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})

			DescribeTable("loop variable type",
				func(bCtx SpecContext, code string, expected types.Type) {
					stmt := MustSucceed(parser.ParseStatement(code))
					ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					loopScope := ctx.Scope.Children()[0]
					sym := MustSucceed(loopScope.Resolve(ctx, "i"))
					Expect(sym.Type).To(Equal(expected))
				},
				Entry(
					"all literals default to i64",
					`for i := range(0, 10) { x := i }`,
					types.I64(),
				),
				Entry(
					"literal start takes the typed end",
					`for i := range(0, i32(10)) { x := i }`,
					types.I32(),
				),
				Entry(
					"literals take the typed step",
					`for i := range(0, 10, u8(2)) { x := i }`,
					types.U8(),
				),
			)

			DescribeTable("mixed integer types",
				func(bCtx SpecContext, code, expected string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.NewRoot(bCtx, block, NewRoot(nil))
					statement.AnalyzeBlock(ctx)
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(expected))
				},
				Entry("i32 start with i64 end", `{
					lo i32 := 1
					hi i64 := 5
					for i := range(lo, hi) { x := i }
				}`, "type mismatch: cannot use i32 and i64 in range()"),
				Entry("i64 step after an i32 end", `{
					n i32 := 3
					s i64 := 1
					for i := range(0, 2 * n, s) { x := i }
				}`, "type mismatch: cannot use i32 and i64 in range()"),
				Entry("unsigned start with signed end", `{
					lo u16 := 1
					hi i16 := 5
					for i := range(lo, hi) { x := i }
				}`, "type mismatch: cannot use u16 and i16 in range()"),
				Entry("cast literal end wider than the start", `{
					n i32 := 5
					for i := range(n, i64(100)) { x := i }
				}`, "type mismatch: cannot use i32 and i64 in range()"),
			)
		})

		Context("series iteration", func() {
			It("should analyze single-ident series iteration", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
					data series i64 := [1, 2, 3]
					for x := data { y := x }
				}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})

			It("should analyze two-ident series iteration", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
					data series f64 := [1.0, 2.0, 3.0]
					for i, x := data { y := x + f64(i) }
				}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})

			It("should reject iteration over non-series", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
					x i32 := 10
					for v := x { y := v }
				}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("cannot iterate"))
			})

			It("should reject two-ident form on non-series", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
					x i32 := 10
					for i, v := x { y := v }
				}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("series"))
			})
		})

		Context("condition and infinite loops", func() {
			It("should analyze conditional for loop", func(bCtx SpecContext) {
				block := MustSucceed(parser.ParseBlock(`{
					running i32 := 1
					for running { running = 0 }
				}`))
				ctx := context.NewRoot(bCtx, block, NewRoot(nil))
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})

			It("should analyze infinite for loop", func(bCtx SpecContext) {
				stmt := MustSucceed(parser.ParseStatement(`for { x := 1 }`))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})
		})

		Context("break and continue", func() {
			It("should accept break inside for loop", func(bCtx SpecContext) {
				stmt := MustSucceed(parser.ParseStatement(`for { break }`))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})

			It("should accept continue inside for loop", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(10) { continue }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})

			It("should reject break outside for loop", func(bCtx SpecContext) {
				stmt := MustSucceed(parser.ParseStatement(`break`))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("break can only be used inside a for loop"))
			})

			It("should reject continue outside for loop", func(bCtx SpecContext) {
				stmt := MustSucceed(parser.ParseStatement(`continue`))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("continue can only be used inside a for loop"))
			})

			It("should accept break in nested if inside loop", func(bCtx SpecContext) {
				stmt := MustSucceed(parser.ParseStatement(`for i := range(10) {
					if i > 5 { break }
				}`))
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			})
		})

		Context("loop variable immutability", func() {
			It("should reject assignment to loop variable", func(bCtx SpecContext) {
				stmt := MustSucceed(
					parser.ParseStatement(`for i := range(10) { i = 5 }`),
				)
				ctx := context.NewRoot(bCtx, stmt, NewRoot(nil))
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("cannot assign to loop variable"))
			})
		})
	})
})

var _ = Describe("Module constants in declarations", func() {
	DescribeTable("Should accept a sequence stateful initializer",
		func(bCtx SpecContext, value string) {
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
				Name: "x",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   1,
			}))
			ast := MustSucceed(parser.Parse("import math\nsequence main {\n" +
				"limit f32 $= " + value + "\nstage a {\nx > 1.0 => next\n}\n" +
				"stage b {\n}\n}\nx => main\nunused := math.inf * x"))
			ctx := context.NewRoot(bCtx, ast, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		},
		Entry("a literal", "1.5"),
		Entry("a constant", "math.inf"),
		Entry("a cast constant", "f32(-math.nan)"),
		Entry("a cast literal", "f32(1.5)"),
		Entry("an expression on a constant", "math.inf * 2.0"),
	)

	It("Should fold a constant cast to its stored value", func(bCtx SpecContext) {
		root := symbol.NewRoot(nil, stl.NewSymbols())
		symbol.AutoImportModules(root)
		expr := MustSucceed(parser.ParseExpression("f32(-math.inf)"))
		ctx := context.NewRoot(bCtx, expr, root)
		Expect(statement.CastConstValue(ctx, expr, types.F32())).To(Equal("-Infinity"))
	})

	DescribeTable("Should keep a sequence variable from a constant out of the flow",
		func(bCtx SpecContext, op string) {
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
				Name: "x",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   1,
			}))
			ast := MustSucceed(parser.Parse(`import math
sequence main {
	limit f64 ` + op + ` math.inf
	stage a {
		x > limit => b
	}
	stage b {
	}
}
x => main`))
			ctx := context.NewRoot(bCtx, ast, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			seq := MustSucceed(ctx.Scope.Resolve(ctx, "main"))
			limit := MustSucceed(seq.Resolve(ctx, "limit"))
			Expect(limit.Type).To(Equal(types.F64()))
			Expect(limit.DefaultValue).To(Equal("Infinity"))
		},
		Entry("a variable", ":="),
		Entry("a stateful variable", "$="),
	)
})

var _ = Describe("Flow variables from module constants", func() {
	DescribeTable(
		"Should accept the initializer",
		func(bCtx SpecContext, source string) {
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
				Name: "x",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   1,
			}))
			ast := MustSucceed(parser.Parse("import math\n" + source))
			ctx := context.NewRoot(bCtx, ast, root)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		},
		Entry("a constant", "limit := math.inf"),
		Entry("an expression on a constant", "limit := math.inf * 2.0"),
		Entry(
			"a sequence variable",
			"sequence main {\n limit := math.nan\n stage a {\n }\n}\nx => main",
		),
		Entry("a channel read", "limit := x * math.inf"),
		Entry(
			"a function",
			"func f() {v f64} f64 {\n limit := math.inf\n return limit\n}",
		),
	)
})

var _ = Describe("Computed initializers", func() {
	It("Should give each computed initializer node its declared type", func(
		bCtx SpecContext,
	) {
		root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
			Name: "go", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1,
		}))
		ast := MustSucceed(parser.Parse(
			"sequence main {\n v u8 $= 2 + 3\n w f32 := 1.5 * 2.0\n stage a {\n }\n}\n" +
				"go => main",
		))
		ctx := context.NewRoot(bCtx, ast, root)
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		var outputs []types.Type
		for _, c := range root.Children() {
			if c.Kind != symbol.KindFunction ||
				!strings.HasPrefix(c.Name, "expression_") {
				continue
			}
			out := MustBeOk(c.Type.Outputs.Get(ir.DefaultOutputParam))
			outputs = append(outputs, ctx.Constraints.ApplySubstitutions(out.Type))
		}
		Expect(outputs).To(ConsistOf(types.U8(), types.F32()))
	})
})
