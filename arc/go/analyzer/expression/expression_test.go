// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expressions", func() {
	Describe("Binary Expressions", func() {
		It("Should validate add expressions on numeric types", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z := x + y
						w := x * y
						v := x - y
						u := x / y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject arithmetic operations on strings", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x str := "hello"
						y str := "world"
						z := x + y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Severity).To(Equal(diagnostics.Error))
			Expect(first.Message).To(ContainSubstring("cannot use str in + operation"))
		})

		It("Should reject mixed type arithmetic", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y f32 := 20.5
						z := x + y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))

			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})

		It("Should validate comparison operations", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						a := x < y
						b := x > y
						c := x <= y
						d := x >= y
						e := x == y
						f := x != y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject mixed type comparisons", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 10
					y u32 := 20
					z := x > y
				}`,
			))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("type mismatch: cannot use i32 and u32 in > operation"))
		})

		It("Should allow for comparison of a floating point variable with an integer literal", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f32 := 10
					z := x > 5
				}`,
			))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should allow comparison of an integer variable with a floating point literal", func() {
			ast := MustSucceed(parser.Parse(`
			func testFunc() {
				x i32 := 10
				z := x > 5.0
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			// With literal inference, 5.0 should adapt to i32
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should validate logical operations on booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						a u8 := 0
						b u8 := 1
						c := a and b
						d := a or b
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject logical AND operations on non-booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z := x and y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("cannot use i32 in and operation"))
		})

		It("Should reject logical OR operations on non-booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x or y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
		})

		It("Should validate modulo operation on integers", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 3
						z := x % y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Unary Expressions", func() {
		It("Should validate unary negation on numeric types", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y := -x
						z f32 := 5.5
						w := -z
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject unary negation on non-numeric types", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x str := "hello"
						y := -x
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("operator - not supported for type str"))
		})

		It("Should validate logical not on booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x u8 := 1
						y := not x
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject logical not on non-booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y := not x
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("operator 'not' requires boolean operand"))
		})
	})

	Describe("Literal Expressions", func() {
		It("Should correctly type integer literals", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := 42
						y i32 := 100
						z i64 := 1000000
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should correctly type float literals", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := 3.14
						y f32 := 2.718
						z f64 := 1.414213
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should correctly type string literals", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := "hello world"
						y str := "test"
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should correctly type u8 values used as booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x := 1
						y u8 := 1
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Complex Expressions", func() {
		It("Should handle nested binary expressions with correct precedence", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z i32 := 30
						result := x + y * z
						result2 := (x + y) * z
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should handle mixed arithmetic and comparison operations", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z i32 := 30
						result := x + y < z
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should handle chained logical operations", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						a u8 := 1
						b u8 := 0
						c u8 := 1
						result := a and b or c
						result2 := a or b and c
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should validate complex mixed expressions", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y i32 := 20
						z i32 := 30
						a u8 := x < y
						b u8 := y > z
						result := a and b or (x + y == z)
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect type errors in complex expressions", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y str := "20"
						z u8 := true
						result := x + y * z
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).ToNot(HaveLen(0))
		})
	})

	Describe("Function Call Expressions", func() {
		It("Should validate function calls with correct arguments", func() {
			ast := MustSucceed(parser.Parse(`
					func add(x i32, y i32) i32 {
						return x + y
					}

					func testFunc() {
						result := add(10, 20)
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect undefined function calls", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						result := undefinedFunc(10, 20)
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: undefinedFunc"))
		})

		It("Should handle nested function calls", func() {
			ast := MustSucceed(parser.Parse(`
					func double(x i32) i32 {
						return x * 2
					}

					func add(x i32, y i32) i32 {
						return x + y
					}

					func testFunc() {
						result := add(double(5), double(10))
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Variable Reference Expressions", func() {
		It("Should resolve local variables correctly", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y := x + 5
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect undefined variable references", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						y := undefinedVar + 5
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: undefinedVar"))
		})

		It("Should not allow shadowing", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						if x > 0 {
							x i32 := 20
							y := x + 5
						}
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("name x conflicts with existing symbol at line 3, col 6"))
		})

		It("Should resolve function parameters correctly", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc(x i32, y i32) i32 {
						return x + y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle empty expressions gracefully", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should handle very deeply nested expressions", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i64 := ((((1 + 2) * 3) - 4) / 5) % 6
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Channels in Expressions", func() {
		It("Should correctly resolve an instantaneous channel read an an expression", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() i32 {
					return (ox_pt_1 + ox_pt_2) / 2
				}
			`))
			resolver := symbol.MapResolver{
				"ox_pt_1": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "ox_pt_1",
					Type: types.Chan(types.I32()),
				},
				"ox_pt_2": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "ox_pt_2",
					Type: types.Chan(types.I32()),
				},
			}
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should return an error when channels with mismatched types are used in arithmetic operations", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() i32 {
					return (ox_pt_1 + ox_pt_2) / 2
				}
			`))
			resolver := symbol.MapResolver{
				"ox_pt_1": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "ox_pt_1",
					Type: types.Chan(types.I32()),
				},
				"ox_pt_2": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "ox_pt_1",
					Type: types.Chan(types.F32()),
				},
			}
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			firstDiag := (*ctx.Diagnostics)[0]
			Expect(firstDiag.Message).To(ContainSubstring("type mismatch: cannot use i32 and f32 in + operation"))
		})

		It("Should not return an error when adding a channel to a variable of the same type", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() i32 {
					return ox_pt_1 + 2
				}
			`))
			resolver := symbol.MapResolver{
				"ox_pt_1": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "ox_pt_1",
					Type: types.Chan(types.I32()),
				},
			}
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("IsPureLiteral", func() {
		getExpr := func(code string) parser.IExpressionContext {
			ast := MustSucceed(parser.Parse(code))
			return ast.AllTopLevelItem()[0].FlowStatement().AllFlowNode()[0].Expression()
		}

		It("Should return true for integer literals", func() {
			expr := getExpr(`42 -> out`)
			Expect(expression.IsPureLiteral(expr)).To(BeTrue())
		})

		It("Should return true for float literals", func() {
			expr := getExpr(`3.14 -> out`)
			Expect(expression.IsPureLiteral(expr)).To(BeTrue())
		})

		It("Should return true for string literals", func() {
			expr := getExpr(`"hello" -> out`)
			Expect(expression.IsPureLiteral(expr)).To(BeTrue())
		})

		It("Should return false for identifiers", func() {
			expr := getExpr(`x -> out`)
			Expect(expr).To(BeNil()) // identifiers are parsed as flowNode.identifier, not expression
		})

		It("Should return false for binary expressions", func() {
			expr := getExpr(`1 + 2 -> out`)
			Expect(expression.IsPureLiteral(expr)).To(BeFalse())
		})

		It("Should return false for unary expressions", func() {
			expr := getExpr(`-1 -> out`)
			Expect(expression.IsPureLiteral(expr)).To(BeFalse())
		})

		It("Should return false for parenthesized expressions", func() {
			expr := getExpr(`(42) -> out`)
			Expect(expression.IsPureLiteral(expr)).To(BeFalse())
		})

		It("Should return false for comparison expressions", func() {
			expr := getExpr(`1 > 0 -> out`)
			Expect(expression.IsPureLiteral(expr)).To(BeFalse())
		})

		It("Should return false for logical expressions", func() {
			expr := getExpr(`1 and 0 -> out`)
			Expect(expression.IsPureLiteral(expr)).To(BeFalse())
		})
	})

	Describe("GetLiteral", func() {
		getExpr := func(code string) parser.IExpressionContext {
			ast := MustSucceed(parser.Parse(code))
			return ast.AllTopLevelItem()[0].FlowStatement().AllFlowNode()[0].Expression()
		}

		It("Should extract integer literal", func() {
			expr := getExpr(`42 -> out`)
			lit := expression.GetLiteral(expr)
			Expect(lit).ToNot(BeNil())
			Expect(lit.GetText()).To(Equal("42"))
		})

		It("Should extract float literal", func() {
			expr := getExpr(`3.14 -> out`)
			lit := expression.GetLiteral(expr)
			Expect(lit).ToNot(BeNil())
			Expect(lit.GetText()).To(Equal("3.14"))
		})

		It("Should extract string literal", func() {
			expr := getExpr(`"hello" -> out`)
			lit := expression.GetLiteral(expr)
			Expect(lit).ToNot(BeNil())
			Expect(lit.GetText()).To(Equal(`"hello"`))
		})
	})
})
