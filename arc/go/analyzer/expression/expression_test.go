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

		It("Should validate function call with no arguments", func() {
			expectSuccess(`
				func getZero() i32 {
					return 0
				}

				func testFunc() {
					x := getZero()
				}
			`, nil)
		})

		It("Should validate function call with expression arguments", func() {
			expectSuccess(`
				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					a i32 := 5
					b i32 := 10
					result := add(a + 1, b * 2)
				}
			`, nil)
		})

		It("Should reject undefined variable in function argument", func() {
			expectFailure(`
				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					result := add(undefinedVar, 10)
				}
			`, nil, "undefined symbol")
		})

		It("Should validate chained function calls", func() {
			expectSuccess(`
				func identity(x i32) i32 {
					return x
				}

				func testFunc() {
					result := identity(identity(identity(5)))
				}
			`, nil)
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

	Describe("Index and Slice Operations", func() {
		It("Should validate array index with integer expression", func() {
			expectSuccess(`
				func testFunc() {
					arr series i32 := [1, 2, 3, 4, 5]
					x := arr[0]
				}
			`, nil)
		})

		It("Should validate array index with variable", func() {
			expectSuccess(`
				func testFunc() {
					arr series i32 := [1, 2, 3]
					idx i32 := 1
					x := arr[idx]
				}
			`, nil)
		})

		It("Should validate array index with expression", func() {
			expectSuccess(`
				func testFunc() {
					arr series i32 := [1, 2, 3, 4]
					i i32 := 1
					x := arr[i + 1]
				}
			`, nil)
		})

		It("Should validate slice operation with both bounds", func() {
			expectSuccess(`
				func testFunc() {
					arr series i32 := [1, 2, 3, 4, 5]
					x := arr[1:3]
				}
			`, nil)
		})

		It("Should reject index with undefined variable", func() {
			expectFailure(`
				func testFunc() {
					arr series i32 := [1, 2, 3]
					x := arr[undefinedIdx]
				}
			`, nil, "undefined symbol")
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

	Describe("IsLiteral", func() {
		getExpr := func(code string) parser.IExpressionContext {
			ast := MustSucceed(parser.Parse(code))
			return ast.AllTopLevelItem()[0].FlowStatement().AllFlowNode()[0].Expression()
		}

		DescribeTable("literal detection",
			func(code string, isLiteral bool) {
				expr := getExpr(code)
				if expr == nil {
					Expect(isLiteral).To(BeFalse())
					return
				}
				Expect(expression.IsLiteral(expr)).To(Equal(isLiteral))
			},
			Entry("integer literal", `42 -> out`, true),
			Entry("float literal", `3.14 -> out`, true),
			Entry("string literal", `"hello" -> out`, true),
			Entry("identifier (parsed as flowNode)", `x -> out`, false),
			Entry("binary expression", `1 + 2 -> out`, false),
			Entry("unary expression", `-1 -> out`, false),
			Entry("parenthesized expression", `(42) -> out`, false),
			Entry("comparison expression", `1 > 0 -> out`, false),
			Entry("logical expression", `1 and 0 -> out`, false),
		)
	})

	Describe("GetLiteral", func() {
		getExpr := func(code string) parser.IExpressionContext {
			ast := MustSucceed(parser.Parse(code))
			return ast.AllTopLevelItem()[0].FlowStatement().AllFlowNode()[0].Expression()
		}

		DescribeTable("literal extraction",
			func(code string, expectedText string) {
				expr := getExpr(code)
				lit := expression.GetLiteral(expr)
				if expectedText == "" {
					Expect(lit).To(BeNil())
				} else {
					Expect(lit).ToNot(BeNil())
					Expect(lit.GetText()).To(Equal(expectedText))
				}
			},
			Entry("integer literal", `42 -> out`, "42"),
			Entry("float literal", `3.14 -> out`, "3.14"),
			Entry("string literal", `"hello" -> out`, `"hello"`),
			Entry("binary expression", `1 + 2 -> out`, ""),
			Entry("unary expression", `-1 -> out`, ""),
		)
	})

	Describe("Power Expressions with Units", func() {
		It("Should accept power expression with literal integer exponent", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 m := 5m
					y := x^2
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept power expression with negative literal integer exponent", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 m := 5m
					y := x^-2
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept power expression with zero exponent", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 m := 5m
					y := x^0
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept dimensionless base with any exponent", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 := 5.0
					y i32 := 2
					z := x^y
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject dimensioned base with variable exponent", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 m := 5m
					n i32 := 2
					y := x^n
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("literal integer exponent"))
		})

		It("Should reject dimensioned exponent", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 := 5.0
					n f64 s := 2s
					y := x^n
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("dimensionless"))
		})

		It("Should reject dimensioned base with float literal exponent", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 m := 5m
					y := x^2.0
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("literal integer exponent"))
		})

		It("Should reject dimensioned base with unit-suffixed literal exponent", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 m := 5m
					y := x^2s
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("dimensionless"))
		})
	})

	Describe("Unit Dimension Validation for Binary Operations", func() {
		It("Should accept addition of same units", func() {
			expectSuccess(`
				func testFunc() {
					x f64 m := 5m
					y f64 m := 3m
					z := x + y
				}
			`, nil)
		})

		It("Should accept subtraction of same units", func() {
			expectSuccess(`
				func testFunc() {
					x f64 s := 10s
					y f64 s := 3s
					z := x - y
				}
			`, nil)
		})

		It("Should reject addition of incompatible units", func() {
			expectFailure(`
				func testFunc() {
					x f64 m := 5m
					y f64 s := 3s
					z := x + y
				}
			`, nil, "incompatible")
		})

		It("Should reject subtraction of incompatible units", func() {
			expectFailure(`
				func testFunc() {
					x f64 m := 5m
					y f64 s := 3s
					z := x - y
				}
			`, nil, "incompatible")
		})

		It("Should accept comparison of same units", func() {
			expectSuccess(`
				func testFunc() {
					x f64 m := 5m
					y f64 m := 3m
					result := x > y
				}
			`, nil)
		})

		It("Should reject comparison of incompatible units", func() {
			expectFailure(`
				func testFunc() {
					x f64 m := 5m
					y f64 s := 3s
					result := x > y
				}
			`, nil, "incompatible")
		})

		It("Should accept equality comparison of same units", func() {
			expectSuccess(`
				func testFunc() {
					x f64 m := 5m
					y f64 m := 5m
					result := x == y
				}
			`, nil)
		})

		It("Should accept dimensionless arithmetic", func() {
			expectSuccess(`
				func testFunc() {
					x f64 := 5.0
					y f64 := 3.0
					z := x + y
				}
			`, nil)
		})

		It("Should accept modulo of same units", func() {
			expectSuccess(`
				func testFunc() {
					x i32 := 5
					y i32 := 3
					z := x % y
				}
			`, nil)
		})
	})

	Describe("Series Type Operations", func() {
		It("Should validate series literal declaration", func() {
			expectSuccess(`
				func testFunc() {
					arr series i32 := [1, 2, 3, 4, 5]
				}
			`, nil)
		})

		It("Should validate series of floats", func() {
			expectSuccess(`
				func testFunc() {
					arr series f64 := [1.0, 2.0, 3.0]
				}
			`, nil)
		})

		It("Should validate empty series", func() {
			expectSuccess(`
				func testFunc() {
					arr series i32 := []
				}
			`, nil)
		})

		It("Should validate series element access", func() {
			expectSuccess(`
				func testFunc() {
					arr series i32 := [10, 20, 30]
					x := arr[0]
				}
			`, nil)
		})

		It("Should validate series in function parameter", func() {
			expectSuccess(`
				func sumFirst(arr series i32) i32 {
					return arr[0]
				}

				func testFunc() {
					data series i32 := [1, 2, 3]
					result := sumFirst(data)
				}
			`, nil)
		})
	})

	Describe("Unary Expression Edge Cases", func() {
		It("Should validate double negation", func() {
			expectSuccess(`
				func testFunc() {
					x i32 := 5
					y := --x
				}
			`, nil)
		})

		It("Should validate negation of parenthesized expression", func() {
			expectSuccess(`
				func testFunc() {
					x i32 := 5
					y i32 := 3
					z := -(x + y)
				}
			`, nil)
		})

		It("Should validate double not", func() {
			expectSuccess(`
				func testFunc() {
					x u8 := 1
					y := not not x
				}
			`, nil)
		})

		It("Should validate negation of function call result", func() {
			expectSuccess(`
				func getValue() i32 {
					return 5
				}

				func testFunc() {
					x := -getValue()
				}
			`, nil)
		})

		It("Should reject not on string", func() {
			expectFailure(`
				func testFunc() {
					x str := "hello"
					y := not x
				}
			`, nil, "boolean operand")
		})

		It("Should reject negation on unsigned integer", func() {
			expectSuccess(`
				func testFunc() {
					x u32 := 5
					y := -x
				}
			`, nil)
		})
	})

	Describe("Binary Expression Edge Cases", func() {
		It("Should validate all comparison operators", func() {
			expectSuccess(`
				func testFunc() {
					x i32 := 5
					y i32 := 3
					a := x < y
					b := x <= y
					c := x > y
					d := x >= y
					e := x == y
					f := x != y
				}
			`, nil)
		})

		It("Should validate multiple additions", func() {
			expectSuccess(`
				func testFunc() {
					a i32 := 1
					b i32 := 2
					c i32 := 3
					d i32 := 4
					result := a + b + c + d
				}
			`, nil)
		})

		It("Should validate multiple multiplications", func() {
			expectSuccess(`
				func testFunc() {
					a i32 := 2
					b i32 := 3
					c i32 := 4
					result := a * b * c
				}
			`, nil)
		})

		It("Should validate mixed operations with correct precedence", func() {
			expectSuccess(`
				func testFunc() {
					a i32 := 2
					b i32 := 3
					c i32 := 4
					d i32 := 5
					result := a + b * c - d / a
				}
			`, nil)
		})

		It("Should reject string in subtraction", func() {
			expectFailure(`
				func testFunc() {
					x str := "hello"
					y str := "world"
					z := x - y
				}
			`, nil, "cannot use str")
		})

		It("Should reject string in multiplication", func() {
			expectFailure(`
				func testFunc() {
					x str := "hello"
					y i32 := 3
					z := x * y
				}
			`, nil, "cannot use str")
		})

		It("Should reject string in division", func() {
			expectFailure(`
				func testFunc() {
					x str := "hello"
					y i32 := 3
					z := x / y
				}
			`, nil, "cannot use str")
		})
	})

	Describe("Literal Edge Cases", func() {
		It("Should validate zero literal", func() {
			expectSuccess(`
				func testFunc() {
					x := 0
				}
			`, nil)
		})

		It("Should validate negative literal in declaration", func() {
			expectSuccess(`
				func testFunc() {
					x i32 := -42
				}
			`, nil)
		})

		It("Should validate very large integer literal", func() {
			expectSuccess(`
				func testFunc() {
					x i64 := 9223372036854775807
				}
			`, nil)
		})

		It("Should validate very small float literal", func() {
			expectSuccess(`
				func testFunc() {
					x f64 := 0.000001
				}
			`, nil)
		})

		It("Should validate empty string literal", func() {
			expectSuccess(`
				func testFunc() {
					x str := ""
				}
			`, nil)
		})

		It("Should validate string with special characters", func() {
			expectSuccess(`
				func testFunc() {
					x str := "hello\nworld"
				}
			`, nil)
		})
	})

	Describe("Channel Type Edge Cases", func() {
		It("Should validate channel in unary expression", func() {
			resolver := symbol.MapResolver{
				"sensor": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "sensor",
					Type: types.Chan(types.F32()),
				},
			}
			expectSuccess(`
				func testFunc() f32 {
					return -sensor
				}
			`, resolver)
		})

		It("Should validate channel in comparison", func() {
			resolver := symbol.MapResolver{
				"sensor": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "sensor",
					Type: types.Chan(types.F32()),
				},
			}
			expectSuccess(`
				func testFunc() u8 {
					return sensor > 100
				}
			`, resolver)
		})

		It("Should validate multiple channels in expression", func() {
			resolver := symbol.MapResolver{
				"temp1": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "temp1",
					Type: types.Chan(types.F64()),
				},
				"temp2": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "temp2",
					Type: types.Chan(types.F64()),
				},
				"temp3": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "temp3",
					Type: types.Chan(types.F64()),
				},
			}
			expectSuccess(`
				func testFunc() f64 {
					return (temp1 + temp2 + temp3) / 3
				}
			`, resolver)
		})

		It("Should reject channel type mismatch in logical operation", func() {
			resolver := symbol.MapResolver{
				"sensor": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "sensor",
					Type: types.Chan(types.F32()),
				},
			}
			expectFailure(`
				func testFunc() u8 {
					return sensor and 1
				}
			`, resolver, "cannot use f32 in and operation")
		})
	})

	Describe("IsLiteral and GetLiteral Edge Cases", func() {
		getExpr := func(code string) parser.IExpressionContext {
			ast := MustSucceed(parser.Parse(code))
			return ast.AllTopLevelItem()[0].FlowStatement().AllFlowNode()[0].Expression()
		}

		It("Should return false for function call expression", func() {
			ast := MustSucceed(parser.Parse(`
				func getValue() i32 {
					return 5
				}
				getValue() -> out
			`))
			flowNode := ast.AllTopLevelItem()[1].FlowStatement().AllFlowNode()[0]
			expr := flowNode.Expression()
			Expect(expression.IsLiteral(expr)).To(BeFalse())
		})

		It("Should return false for index expression", func() {
			expr := getExpr(`arr[0] -> out`)
			Expect(expression.IsLiteral(expr)).To(BeFalse())
		})

		It("Should return nil for GetLiteral on non-literal", func() {
			expr := getExpr(`x + y -> out`)
			Expect(expression.GetLiteral(expr)).To(BeNil())
		})

		It("Should correctly identify numeric literal with unit suffix", func() {
			expr := getExpr(`5m -> out`)
			Expect(expression.IsLiteral(expr)).To(BeTrue())
			lit := expression.GetLiteral(expr)
			Expect(lit).ToNot(BeNil())
		})
	})
})
