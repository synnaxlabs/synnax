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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

// chResolver is the most common resolver used across tests: a single float32 channel "ch".
var chResolver = symbol.MapResolver{
	"ch": {Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
}

func analyzeAndExpect(bCtx SpecContext, source string) context.Context[parser.IProgramContext] {
	return analyzeAndExpectWithResolver(bCtx, source, nil)
}

func analyzeAndExpectWithResolver(bCtx SpecContext, source string, resolver symbol.Resolver) context.Context[parser.IProgramContext] {
	prog := MustSucceed(parser.Parse(source))
	ctx := context.CreateRoot(bCtx, prog, resolver)
	analyzer.AnalyzeProgram(ctx)
	ExpectWithOffset(1, ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
	return ctx
}

func analyzeAndExpectErrorWithResolver(bCtx SpecContext, source string, resolver symbol.Resolver) context.Context[parser.IProgramContext] {
	prog := MustSucceed(parser.Parse(source))
	ctx := context.CreateRoot(bCtx, prog, resolver)
	analyzer.AnalyzeProgram(ctx)
	ExpectWithOffset(1, ctx.Diagnostics.Ok()).To(BeFalse())
	return ctx
}

var _ = Describe("Analyzer Integration", func() {

	Describe("Cross-Scope Symbol Resolution", func() {
		It("Should diagnose a variable declaration that shadows a function name", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					dog := 1
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			errs := ctx.Diagnostics.Errors()
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Message).To(Equal("name dog conflicts with existing symbol at line 2, col 4"))
		})

		It("Should allow variable declaration from a function parameter", func(bCtx SpecContext) {
			_ = analyzeAndExpect(bCtx, `
				func testFunc(a i64) {
					b := a
				}
			`)
		})

		It("Should resolve variables across nested scopes", func(bCtx SpecContext) {
			_ = analyzeAndExpect(bCtx, `
				func outer() i64 {
					x i64 := 10
					if x > 5 {
						y := x + 1
						return y
					}
					return x
				}
			`)
		})
	})

	Describe("Global Shadowing Resolution", func() {
		It("Should allow shadowing built-in function names", func(bCtx SpecContext) {
			globalResolver := symbol.MapResolver{
				"min": symbol.Symbol{Name: "min", Kind: symbol.KindFunction, Type: types.F64()},
			}
			prog := MustSucceed(parser.Parse(`
				func test() i64 {
					min i64 := 10
					return min
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, globalResolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
		})

		It("Should resolve to local variable when shadowing global", func(bCtx SpecContext) {
			globalResolver := symbol.MapResolver{
				"value": symbol.Symbol{Name: "value", Kind: symbol.KindConfig, Type: types.F64()},
			}
			prog := MustSucceed(parser.Parse(`
				func test() i32 {
					value i32 := 42
					return value
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, globalResolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "test"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			varScope := MustSucceed(blockScope.Resolve(ctx, "value"))
			Expect(varScope.Type).To(Equal(types.I32()))
		})

		It("Should use shadowed local in expressions", func(bCtx SpecContext) {
			globalResolver := symbol.MapResolver{
				"x": symbol.Symbol{Name: "x", Kind: symbol.KindConfig, Type: types.F64()},
			}
			prog := MustSucceed(parser.Parse(`
				func test() i64 {
					x i64 := 5
					y := x + 10
					return y
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, globalResolver)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "test"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			yScope := MustSucceed(blockScope.Resolve(ctx, "y"))
			Expect(yScope.Type).To(Equal(types.I64()))
		})
	})

	Describe("Type Unification Results", func() {
		type unificationCase struct {
			expectedType types.Type
			source       string
			funcName     string
			varName      string
		}

		DescribeTable("literal type inference",
			func(bCtx SpecContext, tc unificationCase) {
				ctx := analyzeAndExpect(bCtx, tc.source)
				funcScope := MustSucceed(ctx.Scope.Resolve(ctx, tc.funcName))
				blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
				varScope := MustSucceed(blockScope.Resolve(ctx, tc.varName))
				Expect(varScope.Type).To(Equal(tc.expectedType))
			},
			Entry("integer literal resolves to i64",
				unificationCase{
					source: `
						func testFunc() {
							x := 42
						}
					`,
					funcName:     "testFunc",
					varName:      "x",
					expectedType: types.I64(),
				}),
			Entry("float literal resolves to f64",
				unificationCase{
					source: `
						func testFunc() {
							x := 42.0
						}
					`,
					funcName:     "testFunc",
					varName:      "x",
					expectedType: types.F64(),
				}),
			Entry("integer literal unifies with explicit f32 type",
				unificationCase{
					source: `
						func testFunc() {
							x f32 := 42
						}
					`,
					funcName:     "testFunc",
					varName:      "x",
					expectedType: types.F32(),
				}),
		)
	})

	Describe("Symbol Table Structure", func() {
		It("Should build correct symbol table for if-else statement", func(bCtx SpecContext) {
			ctx := analyzeAndExpect(bCtx, `
				func dog() i64 {
					if 3 > 5 {
						return 1
					} else {
						return 2
					}
				}
			`)

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			Expect(funcScope.Name).To(Equal("dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(2))
			Expect(blocks[0].Children).To(BeEmpty())
			Expect(blocks[1].Children).To(BeEmpty())
		})

		It("Should build correct symbol table for variables in nested blocks", func(bCtx SpecContext) {
			ctx := analyzeAndExpect(bCtx, `
				func dog() i64 {
					a f32 := 2.0
					if (a > 5) {
						b := 2
						return b
					}
					return 2
				}
			`)

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(1))
			Expect(blocks[0].Children).To(HaveLen(1))
			Expect(blocks[0].Children[0].Name).To(Equal("b"))
		})

		It("Should build correct symbol table for if-else-if chain", func(bCtx SpecContext) {
			ctx := analyzeAndExpect(bCtx, `
				func dog(b i64) i64 {
					a i64 := 2
					if b == a {
						return 2
					} else if a > b {
						c i64 := 5
						return c
					} else {
						return 3
					}
				}
			`)

			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "dog"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			blocks := blockScope.FilterChildrenByKind(symbol.KindBlock)
			Expect(blocks).To(HaveLen(3))
			Expect(blocks[0].Children).To(BeEmpty())
			Expect(blocks[1].Children).To(HaveLen(1))
			Expect(blocks[1].Children[0].Name).To(Equal("c"))
			Expect(blocks[2].Children).To(BeEmpty())
		})
	})

	Describe("Multi-Function Programs", func() {
		It("Should analyze program with multiple functions", func(bCtx SpecContext) {
			ctx := analyzeAndExpect(bCtx, `
				func add(x i64, y i64) i64 {
					return x + y
				}

				func multiply(a i64, b i64) i64 {
					return a * b
				}
			`)

			addFunc := MustSucceed(ctx.Scope.Resolve(ctx, "add"))
			Expect(addFunc.Name).To(Equal("add"))
			mulFunc := MustSucceed(ctx.Scope.Resolve(ctx, "multiply"))
			Expect(mulFunc.Name).To(Equal("multiply"))
		})

		It("Should detect undefined function in call expression", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				func caller() i64 {
					return unknownFunc(5)
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			errs := ctx.Diagnostics.Errors()
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Message).To(ContainSubstring("undefined symbol: unknownFunc"))
		})
	})

	Describe("AnalyzeStatement", func() {
		It("Should analyze a valid variable declaration statement", func(bCtx SpecContext) {
			stmt := MustSucceed(parser.ParseStatement("x := 42"))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())

			varScope := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
			Expect(varScope.Type).To(Equal(types.I64()))
		})

		It("Should diagnose undefined symbol in statement", func(bCtx SpecContext) {
			stmt := MustSucceed(parser.ParseStatement("x := undefined_var"))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: undefined_var"))
		})

		It("Should handle type unification in statement analysis", func(bCtx SpecContext) {
			stmt := MustSucceed(parser.ParseStatement("x f32 := 100"))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())

			varScope := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
			Expect(varScope.Type).To(Equal(types.F32()))
		})
	})

	Describe("AnalyzeBlock", func() {
		It("Should analyze a valid block without return statements", func(bCtx SpecContext) {
			// Note: AnalyzeBlock analyzes blocks in isolation without function context,
			// so return statements may fail without proper scope setup.
			prog := MustSucceed(parser.Parse(`
				func test() {
					x := 1
					y := 2
					z := x + y
				}
			`))
			funcDecl := prog.TopLevelItem(0).FunctionDeclaration()
			block := funcDecl.Block()
			progCtx := context.CreateRoot(bCtx, prog, nil)
			blockCtx := context.Child(progCtx, block)
			analyzer.AnalyzeBlock(blockCtx)
			Expect(blockCtx.Diagnostics.Ok()).To(BeTrue())
		})

		It("Should diagnose error in block and stop analysis", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				func test() {
					x := undefined_var
					y := 2
				}
			`))
			funcDecl := prog.TopLevelItem(0).FunctionDeclaration()
			block := funcDecl.Block()
			progCtx := context.CreateRoot(bCtx, prog, nil)
			blockCtx := context.Child(progCtx, block)
			analyzer.AnalyzeBlock(blockCtx)
			Expect(blockCtx.Diagnostics.Ok()).To(BeFalse())
			Expect(*blockCtx.Diagnostics).To(HaveLen(1))
			Expect((*blockCtx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: undefined_var"))
		})

		It("Should handle type unification across block statements", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				func test() {
					x f64 := 1
					y := x + 2.0
				}
			`))
			funcDecl := prog.TopLevelItem(0).FunctionDeclaration()
			block := funcDecl.Block()
			progCtx := context.CreateRoot(bCtx, prog, nil)
			blockCtx := context.Child(progCtx, block)
			analyzer.AnalyzeBlock(blockCtx)
			Expect(blockCtx.Diagnostics.Ok()).To(BeTrue())
		})
	})

	Describe("Global Constants", func() {
		It("Should analyze a constant with inferred type", func(bCtx SpecContext) {
			ctx := analyzeAndExpect(bCtx, `
				MAX_VALUE := 100
			`)
			constScope := MustSucceed(ctx.Scope.Resolve(ctx, "MAX_VALUE"))
			Expect(constScope.Kind).To(Equal(symbol.KindGlobalConstant))
			Expect(constScope.Type).To(Equal(types.I64()))
		})

		It("Should analyze a constant with explicit type", func(bCtx SpecContext) {
			ctx := analyzeAndExpect(bCtx, `
				THRESHOLD f32 := 50.5
			`)
			constScope := MustSucceed(ctx.Scope.Resolve(ctx, "THRESHOLD"))
			Expect(constScope.Kind).To(Equal(symbol.KindGlobalConstant))
			Expect(constScope.Type).To(Equal(types.F32()))
		})

		It("Should allow using a constant inside a function", func(bCtx SpecContext) {
			ctx := analyzeAndExpect(bCtx, `
				LIMIT := 10

				func check(x i64) i64 {
					return x + LIMIT
				}
			`)
			constScope := MustSucceed(ctx.Scope.Resolve(ctx, "LIMIT"))
			Expect(constScope.Kind).To(Equal(symbol.KindGlobalConstant))
			Expect(constScope.Type).To(Equal(types.I64()))
		})
	})

	Describe("Channel Propagation Through Function Calls", func() {
		It("Should propagate callee channels to caller when callee is declared first", func(bCtx SpecContext) {
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func callee() {
					ch = 1.0
				}
				func caller() {
					callee()
				}
			`, chResolver)
			caller := MustSucceed(ctx.Scope.Resolve(ctx, "caller"))
			Expect(caller.Channels.Write[10]).To(Equal("ch"))
		})

		It("Should propagate callee channels to caller for forward references", func(bCtx SpecContext) {
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func caller() {
					callee()
				}
				func callee() {
					ch = 1.0
				}
			`, chResolver)
			caller := MustSucceed(ctx.Scope.Resolve(ctx, "caller"))
			Expect(caller.Channels.Write[10]).To(Equal("ch"))
		})

		It("Should propagate channels through multi-level forward reference chains", func(bCtx SpecContext) {
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func a() {
					b()
				}
				func b() {
					c()
				}
				func c() {
					ch = 1.0
				}
			`, chResolver)
			a := MustSucceed(ctx.Scope.Resolve(ctx, "a"))
			Expect(a.Channels.Write[10]).To(Equal("ch"))
			b := MustSucceed(ctx.Scope.Resolve(ctx, "b"))
			Expect(b.Channels.Write[10]).To(Equal("ch"))
		})

		It("Should propagate both read and write channels", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
				"valve":  {Name: "valve", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 20},
			}
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func caller() {
					callee()
				}
				func callee() {
					valve = sensor * 2
				}
			`, resolver)
			caller := MustSucceed(ctx.Scope.Resolve(ctx, "caller"))
			Expect(caller.Channels.Read[10]).To(Equal("sensor"))
			Expect(caller.Channels.Write[20]).To(Equal("valve"))
		})

		It("Should error on mutual recursion", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch1": {Name: "ch1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
				"ch2": {Name: "ch2", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 20},
			}
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					ch1 = 1.0
					b()
				}
				func b() {
					ch2 = 2.0
					a()
				}
			`, resolver)
			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("a"))
			Expect(diag.Message).To(ContainSubstring("b"))
			Expect(diag.Start.Line).To(Equal(8))
		})

		It("Should error on self-recursion", func(bCtx SpecContext) {
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					ch = 1.0
					a()
				}
			`, chResolver)
			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("a -> a"))
			Expect(diag.Start.Line).To(Equal(4))
		})

		It("Should error on circular dependency chain", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch_a": {Name: "ch_a", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
				"ch_d": {Name: "ch_d", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 40},
			}
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					ch_a = 1.0
					b()
				}
				func b() {
					c()
				}
				func c() {
					d()
				}
				func d() {
					ch_d = 4.0
					a()
				}
			`, resolver)
			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Message).To(ContainSubstring("a"))
			Expect(diag.Message).To(ContainSubstring("b"))
			Expect(diag.Message).To(ContainSubstring("c"))
			Expect(diag.Message).To(ContainSubstring("d"))
			Expect(diag.Start.Line).To(Equal(14))
		})

		It("Should error on diamond with back edge to root", func(bCtx SpecContext) {
			analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					ch = 1.0
					b()
					c()
				}
				func b() {
					a()
				}
				func c() {
					a()
				}
			`, chResolver)
		})

		It("Should error on cycle buried in a larger call tree", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch1": {Name: "ch1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
				"ch2": {Name: "ch2", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 20},
				"ch3": {Name: "ch3", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 30},
			}
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func leaf1() { ch1 = 1.0 }
				func leaf2() { ch2 = 2.0 }
				func leaf3() { ch3 = 3.0 }
				func safe_mid() {
					leaf1()
					leaf2()
				}
				func cycle_a() {
					ch3 = 3.0
					cycle_b()
				}
				func cycle_b() {
					cycle_a()
					leaf3()
				}
				func top() {
					safe_mid()
					cycle_a()
				}
			`, resolver)
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("cycle_a"))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("cycle_b"))
		})

		It("Should error on multiple independent cycles", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch1": {Name: "ch1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
				"ch2": {Name: "ch2", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 20},
			}
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func ping() {
					ch1 = 1.0
					pong()
				}
				func pong() {
					ping()
				}
				func tick() {
					ch2 = 2.0
					tock()
				}
				func tock() {
					tick()
				}
			`, resolver)
			Expect(len(*ctx.Diagnostics)).To(BeNumerically(">=", 2))
		})

		It("Should not false-positive on a deep acyclic call tree", func(bCtx SpecContext) {
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func d() { ch = 1.0 }
				func c() { d() }
				func b() { c() }
				func a() { b() }
			`, chResolver)
			a := MustSucceed(ctx.Scope.Resolve(ctx, "a"))
			Expect(a.Channels.Write[10]).To(Equal("ch"))
		})

		It("Should error when a function both self-recurses and participates in a mutual cycle", func(bCtx SpecContext) {
			analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					ch = 1.0
					a()
					b()
				}
				func b() {
					a()
				}
			`, chResolver)
		})

		It("Should not false-positive on a non-cyclic caller that reaches a cycle", func(bCtx SpecContext) {
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func cycle_a() {
					ch = 1.0
					cycle_b()
				}
				func cycle_b() {
					cycle_a()
				}
				func wrapper() {
					cycle_a()
				}
			`, chResolver)
			for _, d := range ctx.Diagnostics.Errors() {
				Expect(d.Message).ToNot(ContainSubstring("wrapper"))
			}
		})

		It("Should allow self-recursion with if guard", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 0 {
						ch = 1.0
						a()
					}
				}
			`, chResolver)
		})

		It("Should allow mutual recursion when one edge is guarded", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					ch = 1.0
					if ch > 0 {
						b()
					}
				}
				func b() {
					a()
				}
			`, chResolver)
		})

		It("Should allow self-recursion in else-if block", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 0 {
						ch = 1.0
					} else if ch < 0 {
						a()
					}
				}
			`, chResolver)
		})

		It("Should allow self-recursion when else branch is safe in if/else-if/else", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 10 {
						a()
					} else if ch > 0 {
						a()
					} else {
						ch = 1.0
					}
				}
			`, chResolver)
		})

		It("Should allow self-recursion when else-if branch is safe in if/else-if/else", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 10 {
						a()
					} else if ch > 0 {
						ch = 1.0
					} else {
						a()
					}
				}
			`, chResolver)
		})

		It("Should allow self-recursion when if branch is safe in if/else-if/else", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 10 {
						ch = 1.0
					} else if ch > 0 {
						a()
					} else {
						a()
					}
				}
			`, chResolver)
		})

		It("Should error when call is in all branches of if-else", func(bCtx SpecContext) {
			analyzeAndExpectErrorWithResolver(bCtx, `
				func ping() {
					ch = 1.0
					if ch > 0 {
						pong()
					} else {
						pong()
					}
				}
				func pong() {
					ping()
				}
			`, chResolver)
		})

		It("Should allow recursion when nested if has outer guard without else", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 0 {
						if ch > 10 {
							a()
						} else {
							a()
						}
					}
				}
			`, chResolver)
		})

		It("Should error when all paths through nested ifs call callee", func(bCtx SpecContext) {
			analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					ch = 1.0
					if ch > 0 {
						if ch > 10 {
							a()
						} else {
							a()
						}
					} else {
						a()
					}
				}
			`, chResolver)
		})

		It("Should error on tangled web of 5 functions with no exit path", func(bCtx SpecContext) {
			analyzeAndExpectErrorWithResolver(bCtx, `
				func init_seq() {
					ch = 1.0
					if ch > 50 {
						ch = ch + 20.0
					} else if ch > 20 {
						ch = ch + 5.0
					}
					proc_alpha()
				}
				func proc_alpha() {
					if ch > 80 {
						ch = ch - 30.0
					} else {
						ch = ch + 10.0
					}
					xform()
				}
				func xform() {
					ch = ch + 15.0
					if ch > 120 {
						ch = 100.0
					} else {
						ch = ch + 20.0
					}
					route_beta()
				}
				func route_beta() {
					if ch > 90 {
						ch = ch - 40.0
					} else if ch > 60 {
						ch = ch + 15.0
					}
					commit()
				}
				func commit() {
					ch = 50.0
					init_seq()
				}
			`, chResolver)
		})

		It("Should allow tangled web when route_beta has a single exit condition", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func init_seq() {
					ch = 1.0
					if ch > 50 {
						ch = ch + 20.0
					} else if ch > 20 {
						ch = ch + 5.0
					}
					proc_alpha()
				}
				func proc_alpha() {
					if ch > 80 {
						ch = ch - 30.0
					} else {
						ch = ch + 10.0
					}
					xform()
				}
				func xform() {
					ch = ch + 15.0
					if ch > 120 {
						ch = 100.0
					} else {
						ch = ch + 20.0
					}
					route_beta()
				}
				func route_beta() {
					if ch > 0 {
						if ch > 90 {
							ch = ch - 40.0
						} else if ch > 60 {
							ch = ch + 15.0
						}
						commit()
					}
				}
				func commit() {
					ch = 50.0
					init_seq()
				}
			`, chResolver)
		})

		It("Should error when a guarded call coexists with an unconditional recursive cycle", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch1": {Name: "ch1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
			}
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					if ch1 > 0 {
						b()
					} else {
						ch1 = 0
					}
					c()
				}
				func b() {
					a()
				}
				func c() {
					a()
				}
			`, resolver)
			errs := ctx.Diagnostics.Errors()
			Expect(errs).To(HaveLen(1))
			msg := errs[0].Message
			Expect(msg).To(ContainSubstring("circular function call"))
			Expect(msg).To(ContainSubstring("a"))
			Expect(msg).To(ContainSubstring("c"))
		})

		It("Should error when guard only covers one of two recursive paths from the same function", func(bCtx SpecContext) {
			analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					if ch > 100 {
						b()
					}
					c()
				}
				func b() {
					ch = ch + 1.0
					c()
				}
				func c() {
					ch = ch - 1.0
					a()
				}
			`, chResolver)
		})

		It("Should error when every branch calls a different function but all lead back", func(bCtx SpecContext) {
			analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					if ch > 0 {
						b()
					} else {
						c()
					}
				}
				func b() {
					ch = ch + 1.0
					a()
				}
				func c() {
					ch = ch - 1.0
					a()
				}
			`, chResolver)
		})

		It("Should error when deep chain has one guarded link but another path bypasses it", func(bCtx SpecContext) {
			analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					if ch > 50 {
						b()
					}
					d()
				}
				func b() {
					ch = ch + 10.0
					c()
				}
				func c() {
					ch = ch - 5.0
					d()
				}
				func d() {
					ch = ch + 1.0
					a()
				}
			`, chResolver)
		})

		It("Should error on self-recursion hidden after a guarded call to another function", func(bCtx SpecContext) {
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					if ch > 0 {
						helper()
					}
					a()
				}
				func helper() {
					ch = 0
				}
			`, chResolver)
			msg := (*ctx.Diagnostics)[0].Message
			Expect(msg).To(ContainSubstring("a"))
		})

		It("Should allow self-recursion guarded by early return", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch <= 0 {
						return
					}
					a()
				}
			`, chResolver)
		})

		It("Should allow recursion guarded by early return with value", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func factorial(n i64) i64 {
					if n <= 1 {
						return 1
					}
					return n * factorial(n - 1)
				}
				func main() i64 {
					return factorial(5)
				}
			`, nil)
		})

		It("Should allow recursion when if/else always returns before call", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 0 {
						ch = 1.0
						return
					} else {
						ch = 2.0
						return
					}
					a()
				}
			`, chResolver)
		})

		It("Should allow recursion when if/else-if/else always returns before call", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 10 {
						return
					} else if ch > 0 {
						return
					} else {
						return
					}
					a()
				}
			`, chResolver)
		})

		It("Should allow mutual recursion guarded by early return", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch <= 0 {
						return
					}
					b()
				}
				func b() {
					a()
				}
			`, chResolver)
		})

		It("Should allow recursion when nested if/else always returns before call", func(bCtx SpecContext) {
			analyzeAndExpectWithResolver(bCtx, `
				func a() {
					if ch > 100 {
						if ch > 200 {
							return
						} else {
							return
						}
					} else {
						return
					}
					a()
				}
			`, chResolver)
		})

		It("Should error when return comes after recursive call", func(bCtx SpecContext) {
			ctx := analyzeAndExpectErrorWithResolver(bCtx, `
				func a() {
					a()
					return
				}
			`, chResolver)
			diag := (*ctx.Diagnostics)[0]
			Expect(diag.Start.Line).To(Equal(3))
		})

		It("Should handle diamond dependency without duplication", func(bCtx SpecContext) {
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func d() {
					ch = 1.0
				}
				func b() {
					d()
				}
				func c() {
					d()
				}
				func a() {
					b()
					c()
				}
			`, chResolver)
			a := MustSucceed(ctx.Scope.Resolve(ctx, "a"))
			Expect(a.Channels.Write).To(HaveLen(1))
			Expect(a.Channels.Write[10]).To(Equal("ch"))
		})

		It("Should merge channels from multiple callees", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch1": {Name: "ch1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
				"ch2": {Name: "ch2", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 20},
			}
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func helper1() {
					ch1 = 1.0
				}
				func helper2() {
					ch2 = 2.0
				}
				func caller() {
					helper1()
					helper2()
				}
			`, resolver)
			caller := MustSucceed(ctx.Scope.Resolve(ctx, "caller"))
			Expect(caller.Channels.Write).To(HaveLen(2))
			Expect(caller.Channels.Write[10]).To(Equal("ch1"))
			Expect(caller.Channels.Write[20]).To(Equal("ch2"))
		})

		It("Should deduplicate when multiple callees write the same channel", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"ch1": {Name: "ch1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
			}
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func helper1() {
					ch1 = 1.0
				}
				func helper2() {
					ch1 = 2.0
				}
				func caller() {
					helper1()
					helper2()
				}
			`, resolver)
			caller := MustSucceed(ctx.Scope.Resolve(ctx, "caller"))
			Expect(caller.Channels.Write).To(HaveLen(1))
			Expect(caller.Channels.Write[10]).To(Equal("ch1"))
		})

		It("Should propagate when callee both reads and writes the same channel", func(bCtx SpecContext) {
			resolver := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
			}
			ctx := analyzeAndExpectWithResolver(bCtx, `
				func callee() {
					sensor = sensor + 1
				}
				func caller() {
					callee()
				}
			`, resolver)
			caller := MustSucceed(ctx.Scope.Resolve(ctx, "caller"))
			Expect(caller.Channels.Read[10]).To(Equal("sensor"))
			Expect(caller.Channels.Write[10]).To(Equal("sensor"))
		})
	})

	Describe("Complete Analysis", func() {
		It("Should report multiple independent errors in different functions", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				func a() { x := undefined1 }
				func b() { y := undefined2 }
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			errs := ctx.Diagnostics.Errors()
			Expect(errs).To(HaveLen(2))
			Expect(errs[0].Message).To(ContainSubstring("undefined1"))
			Expect(errs[1].Message).To(ContainSubstring("undefined2"))
		})

		It("Should not cascade undefined errors for poisoned symbols", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				func test() {
					x := undefined_var
					y := x + 1
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			// Only the original error - no "undefined x" cascade
			errs := ctx.Diagnostics.Errors()
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Message).To(ContainSubstring("undefined_var"))
		})

		It("Should not cascade type errors when operands are Invalid", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				func test() {
					x := undefined_var
					y := x + "string"
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			// Only the original error - no type mismatch cascade
			errs := ctx.Diagnostics.Errors()
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Message).To(ContainSubstring("undefined_var"))
		})

		It("Should report all errors in if/else branches", func(bCtx SpecContext) {
			prog := MustSucceed(parser.Parse(`
				func test() {
					if 1 > 0 {
						x := undefined1
					} else {
						y := undefined2
					}
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(ctx.Diagnostics.Errors()).To(HaveLen(2))
		})
	})
})
