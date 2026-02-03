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

// analyzeAndExpect is a helper that parses source, analyzes it, and returns the context.
// It asserts the analysis succeeds when expectSuccess is true.
func analyzeAndExpect(source string) context.Context[parser.IProgramContext] {
	prog := MustSucceed(parser.Parse(source))
	ctx := context.CreateRoot(bCtx, prog, nil)
	analyzer.AnalyzeProgram(ctx)
	Expect(ctx.Diagnostics.Ok()).To(BeTrue())
	return ctx
}

var _ = Describe("Analyzer Integration", func() {

	Describe("Cross-Scope Symbol Resolution", func() {
		It("Should diagnose a variable declaration that shadows a function name", func() {
			prog := MustSucceed(parser.Parse(`
				func dog() {
					dog := 1
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			diagnostic := (*ctx.Diagnostics)[0]
			Expect(diagnostic.Message).To(Equal("name dog conflicts with existing symbol at line 2, col 4"))
		})

		It("Should allow variable declaration from a function parameter", func() {
			_ = analyzeAndExpect(`
				func testFunc(a i64) {
					b := a
				}
			`)
		})

		It("Should resolve variables across nested scopes", func() {
			_ = analyzeAndExpect(`
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
		It("Should allow shadowing built-in function names", func() {
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
			analyzer.AnalyzeProgram(ctx, nil)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
		})

		It("Should resolve to local variable when shadowing global", func() {
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
			analyzer.AnalyzeProgram(ctx, nil)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "test"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			varScope := MustSucceed(blockScope.Resolve(ctx, "value"))
			Expect(varScope.Type).To(Equal(types.I32()))
		})

		It("Should use shadowed local in expressions", func() {
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
			analyzer.AnalyzeProgram(ctx, nil)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
			funcScope := MustSucceed(ctx.Scope.Resolve(ctx, "test"))
			blockScope := MustSucceed(funcScope.FirstChildOfKind(symbol.KindBlock))
			yScope := MustSucceed(blockScope.Resolve(ctx, "y"))
			Expect(yScope.Type).To(Equal(types.I64()))
		})
	})

	Describe("Type Unification Results", func() {
		type unificationCase struct {
			source       string
			funcName     string
			varName      string
			expectedType types.Type
		}

		DescribeTable("literal type inference",
			func(tc unificationCase) {
				ctx := analyzeAndExpect(tc.source)
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
		It("Should build correct symbol table for if-else statement", func() {
			ctx := analyzeAndExpect(`
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

		It("Should build correct symbol table for variables in nested blocks", func() {
			ctx := analyzeAndExpect(`
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

		It("Should build correct symbol table for if-else-if chain", func() {
			ctx := analyzeAndExpect(`
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
		It("Should analyze program with multiple functions", func() {
			ctx := analyzeAndExpect(`
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

		It("Should detect undefined function in call expression", func() {
			prog := MustSucceed(parser.Parse(`
				func caller() i64 {
					return unknownFunc(5)
				}
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: unknownFunc"))
		})
	})

	Describe("AnalyzeStatement", func() {
		It("Should analyze a valid variable declaration statement", func() {
			stmt := MustSucceed(parser.ParseStatement("x := 42"))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())

			varScope := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
			Expect(varScope.Type).To(Equal(types.I64()))
		})

		It("Should diagnose undefined symbol in statement", func() {
			stmt := MustSucceed(parser.ParseStatement("x := undefined_var"))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: undefined_var"))
		})

		It("Should handle type unification in statement analysis", func() {
			stmt := MustSucceed(parser.ParseStatement("x f32 := 100"))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			analyzer.AnalyzeStatement(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())

			varScope := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
			Expect(varScope.Type).To(Equal(types.F32()))
		})
	})

	Describe("AnalyzeBlock", func() {
		It("Should analyze a valid block without return statements", func() {
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

		It("Should diagnose error in block and stop analysis", func() {
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

		It("Should handle type unification across block statements", func() {
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

	Describe("Complete Analysis", func() {
		It("Should report multiple independent errors in different functions", func() {
			prog := MustSucceed(parser.Parse(`
				func a() { x := undefined1 }
				func b() { y := undefined2 }
			`))
			ctx := context.CreateRoot(bCtx, prog, nil)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(2))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined1"))
			Expect((*ctx.Diagnostics)[1].Message).To(ContainSubstring("undefined2"))
		})

		It("Should not cascade undefined errors for poisoned symbols", func() {
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
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined_var"))
		})

		It("Should not cascade type errors when operands are Invalid", func() {
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
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined_var"))
		})

		It("Should report all errors in if/else branches", func() {
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
			Expect(*ctx.Diagnostics).To(HaveLen(2))
		})
	})
})
