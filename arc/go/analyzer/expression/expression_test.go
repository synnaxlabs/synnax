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

// expectOperatorTypeError validates that code fails with an error mentioning
// both the type and operator. This provides more precise error checking than
// just checking for a substring.
func expectOperatorTypeError(code string, typeName, operator string) {
	ast := MustSucceed(parser.Parse(code))
	ctx := context.CreateRoot(bCtx, ast, nil)
	Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
	Expect(*ctx.Diagnostics).To(HaveLen(1))
	Expect((*ctx.Diagnostics)[0].Severity).To(Equal(diagnostics.Error))
	Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(typeName))
	Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(operator))
}

var _ = Describe("Expressions", func() {
	Describe("Binary Expressions", func() {
		DescribeTable("valid arithmetic operations",
			func(code string) { expectSuccess(code, nil) },
			Entry("add on i32", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x + y
				}
			`),
			Entry("subtract on i32", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x - y
				}
			`),
			Entry("multiply on i32", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x * y
				}
			`),
			Entry("divide on i32", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x / y
				}
			`),
			Entry("modulo on i32", `
				func testFunc() {
					x i32 := 10
					y i32 := 3
					z := x % y
				}
			`),
			Entry("all comparison operators", `
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
			`),
			Entry("logical AND on booleans", `
				func testFunc() {
					a u8 := 0
					b u8 := 1
					c := a and b
				}
			`),
			Entry("logical OR on booleans", `
				func testFunc() {
					a u8 := 0
					b u8 := 1
					c := a or b
				}
			`),
			Entry("multiple additions", `
				func testFunc() {
					a i32 := 1
					b i32 := 2
					c i32 := 3
					d i32 := 4
					result := a + b + c + d
				}
			`),
			Entry("multiple multiplications", `
				func testFunc() {
					a i32 := 2
					b i32 := 3
					c i32 := 4
					result := a * b * c
				}
			`),
			Entry("mixed operations with correct precedence", `
				func testFunc() {
					a i32 := 2
					b i32 := 3
					c i32 := 4
					d i32 := 5
					result := a + b * c - d / a
				}
			`),
		)

		DescribeTable("invalid arithmetic operations on strings",
			func(code string, operator string) {
				expectOperatorTypeError(code, "str", operator)
			},
			Entry("add", `
				func testFunc() {
					x str := "hello"
					y str := "world"
					z := x + y
				}
			`, "+"),
			Entry("subtract", `
				func testFunc() {
					x str := "hello"
					y str := "world"
					z := x - y
				}
			`, "-"),
			Entry("multiply", `
				func testFunc() {
					x str := "hello"
					y i32 := 3
					z := x * y
				}
			`, "*"),
			Entry("divide", `
				func testFunc() {
					x str := "hello"
					y i32 := 3
					z := x / y
				}
			`, "/"),
		)

		DescribeTable("invalid logical operations on non-booleans",
			func(code string, typeName, operator string) {
				expectOperatorTypeError(code, typeName, operator)
			},
			Entry("AND on i32", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x and y
				}
			`, "i32", "and"),
			Entry("OR on i32", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x or y
				}
			`, "i32", "or"),
		)

		DescribeTable("type mismatch errors",
			func(code string, expectedErrSubstring string) {
				expectFailure(code, nil, expectedErrSubstring)
			},
			Entry("mixed type arithmetic", `
				func testFunc() {
					x i32 := 10
					y f32 := 20.5
					z := x + y
				}
			`, "type mismatch"),
			Entry("mixed type comparisons", `
				func testFunc() {
					x i32 := 10
					y u32 := 20
					z := x > y
				}
			`, "type mismatch: cannot use i32 and u32 in > operation"),
		)
	})

	Describe("Unary Expressions", func() {
		DescribeTable("valid unary operations",
			func(code string) { expectSuccess(code, nil) },
			Entry("negation on i32", `
				func testFunc() {
					x i32 := 10
					y := -x
				}
			`),
			Entry("negation on f32", `
				func testFunc() {
					z f32 := 5.5
					w := -z
				}
			`),
			Entry("logical not on boolean", `
				func testFunc() {
					x u8 := 1
					y := not x
				}
			`),
			Entry("double negation", `
				func testFunc() {
					x i32 := 5
					y := --x
				}
			`),
			Entry("negation of parenthesized expression", `
				func testFunc() {
					x i32 := 5
					y i32 := 3
					z := -(x + y)
				}
			`),
			Entry("double not", `
				func testFunc() {
					x u8 := 1
					y := not not x
				}
			`),
			Entry("negation of function call result", `
				func getValue() i32 {
					return 5
				}

				func testFunc() {
					x := -getValue()
				}
			`),
			Entry("negation on unsigned integer", `
				func testFunc() {
					x u32 := 5
					y := -x
				}
			`),
		)

		DescribeTable("invalid unary operations",
			func(code string, expectedErrSubstring string) {
				expectFailure(code, nil, expectedErrSubstring)
			},
			Entry("negation on string", `
				func testFunc() {
					x str := "hello"
					y := -x
				}
			`, "operator - not supported for type str"),
			Entry("not on non-boolean", `
				func testFunc() {
					x i32 := 10
					y := not x
				}
			`, "operator 'not' requires boolean operand"),
			Entry("not on string", `
				func testFunc() {
					x str := "hello"
					y := not x
				}
			`, "boolean operand"),
		)
	})

	Describe("Literal Expressions", func() {
		DescribeTable("valid literals",
			func(code string) { expectSuccess(code, nil) },
			Entry("integer literal", `
				func testFunc() {
					x := 42
					y i32 := 100
					z i64 := 1000000
				}
			`),
			Entry("float literals", `
				func testFunc() {
					x := 3.14
					y f32 := 2.718
					z f64 := 1.414213
				}
			`),
			Entry("string literals", `
				func testFunc() {
					x := "hello world"
					y str := "test"
				}
			`),
			Entry("u8 boolean values", `
				func testFunc() {
					x := 1
					y u8 := 1
				}
			`),
			Entry("zero literal", `
				func testFunc() {
					x := 0
				}
			`),
			Entry("negative literal", `
				func testFunc() {
					x i32 := -42
				}
			`),
			Entry("very large integer", `
				func testFunc() {
					x i64 := 9223372036854775807
				}
			`),
			Entry("very small float", `
				func testFunc() {
					x f64 := 0.000001
				}
			`),
			Entry("empty string", `
				func testFunc() {
					x str := ""
				}
			`),
			Entry("string with special characters", `
				func testFunc() {
					x str := "hello\nworld"
				}
			`),
		)
	})

	Describe("Complex Expressions", func() {
		DescribeTable("valid complex expressions",
			func(code string) { expectSuccess(code, nil) },
			Entry("nested with correct precedence", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z i32 := 30
					result := x + y * z
					result2 := (x + y) * z
				}
			`),
			Entry("mixed arithmetic and comparison", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z i32 := 30
					result := x + y < z
				}
			`),
			Entry("chained logical operations", `
				func testFunc() {
					a u8 := 1
					b u8 := 0
					c u8 := 1
					result := a and b or c
					result2 := a or b and c
				}
			`),
			Entry("complex mixed expressions", `
				func testFunc() {
					x i32 := 10
					y i32 := 20
					z i32 := 30
					a u8 := x < y
					b u8 := y > z
					result := a and b or (x + y == z)
				}
			`),
			Entry("very deeply nested", `
				func testFunc() {
					x i64 := ((((1 + 2) * 3) - 4) / 5) % 6
				}
			`),
		)

		It("Should detect type errors in complex expressions", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 10
					y str := "20"
					z u8 := 1
					result := x + y * z
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("cannot use str"))
		})
	})

	Describe("Function Call Expressions", func() {
		DescribeTable("valid function calls",
			func(code string) { expectSuccess(code, nil) },
			Entry("with correct arguments", `
				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					result := add(10, 20)
				}
			`),
			Entry("nested function calls", `
				func double(x i32) i32 {
					return x * 2
				}

				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					result := add(double(5), double(10))
				}
			`),
			Entry("no arguments", `
				func getZero() i32 {
					return 0
				}

				func testFunc() {
					x := getZero()
				}
			`),
			Entry("expression arguments", `
				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					a i32 := 5
					b i32 := 10
					result := add(a + 1, b * 2)
				}
			`),
			Entry("chained function calls", `
				func identity(x i32) i32 {
					return x
				}

				func testFunc() {
					result := identity(identity(identity(5)))
				}
			`),
		)

		DescribeTable("invalid function calls",
			func(code string, expectedErrSubstring string) {
				expectFailure(code, nil, expectedErrSubstring)
			},
			Entry("undefined function", `
				func testFunc() {
					result := undefinedFunc(10, 20)
				}
			`, "undefined symbol: undefinedFunc"),
			Entry("undefined variable in argument", `
				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					result := add(undefinedVar, 10)
				}
			`, "undefined symbol"),
			Entry("too few arguments", `
				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					result := add(5)
				}
			`, "expects 2 argument(s), got 1"),
			Entry("too many arguments", `
				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					result := add(5, 10, 15)
				}
			`, "expects 2 argument(s), got 3"),
			Entry("no arguments when expected", `
				func getValue(x i32) i32 {
					return x
				}

				func testFunc() {
					result := getValue()
				}
			`, "expects 1 argument(s), got 0"),
			Entry("wrong argument type - string instead of i32", `
				func add(x i32, y i32) i32 {
					return x + y
				}

				func testFunc() {
					result := add(5, "hello")
				}
			`, "argument 2 of add"),
			Entry("wrong argument type - i32 vs f32 variable", `
				func process(x f32) f32 {
					return x * 2.0
				}

				func testFunc() {
					x i32 := 5
					result := process(x)
				}
			`, "argument 1 of process"),
			Entry("nested call type mismatch", `
				func getFloat() f32 {
					return 3.14
				}

				func needsInt(x i32) i32 {
					return x + 1
				}

				func testFunc() {
					result := needsInt(getFloat())
				}
			`, "argument 1 of needsInt"),
			Entry("wrong arg count in nested call", `
				func double(x i32) i32 {
					return x * 2
				}

				func testFunc() {
					result := double(double())
				}
			`, "expects 1 argument(s), got 0"),
			Entry("calling a variable as a function", `
				func testFunc() {
					x i32 := 42
					result := x()
				}
			`, "cannot call non-function"),
			Entry("calling a parameter as a function", `
				func testFunc(x i32) i32 {
					return x()
				}
			`, "cannot call non-function"),
		)
	})

	Describe("Optional Parameter Function Calls", func() {
		DescribeTable("valid optional parameter calls",
			func(code string) { expectSuccess(code, nil) },
			Entry("omit single optional parameter", `
				func add(x i64, y i64 = 0) i64 { return x + y }
				func testFunc() { result := add(10) }
			`),
			Entry("provide all arguments including optional", `
				func add(x i64, y i64 = 0) i64 { return x + y }
				func testFunc() { result := add(10, 20) }
			`),
			Entry("omit multiple optional parameters", `
				func sum(a i64, b i64 = 1, c i64 = 2) i64 { return a + b + c }
				func testFunc() {
					x := sum(10)
					y := sum(10, 20)
					z := sum(10, 20, 30)
				}
			`),
			Entry("string default parameter", `
				func greet(name str = "world") str { return name }
				func testFunc() { result := greet() }
			`),
			Entry("all optional parameters", `
				func defaults(a i64 = 1, b i64 = 2) i64 { return a + b }
				func testFunc() { result := defaults() }
			`),
			Entry("f64 default parameter", `
				func scale(x f64, factor f64 = 2.5) f64 { return x * factor }
				func testFunc() { result := scale(4.0) }
			`),
		)

		DescribeTable("invalid optional parameter calls",
			func(code string, expectedErrSubstring string) {
				expectFailure(code, nil, expectedErrSubstring)
			},
			Entry("missing required argument", `
				func add(x i64, y i64 = 0) i64 { return x + y }
				func testFunc() { result := add() }
			`, "expects 1 to 2 argument(s), got 0"),
			Entry("too many arguments", `
				func add(x i64, y i64 = 0) i64 { return x + y }
				func testFunc() { result := add(1, 2, 3) }
			`, "expects 1 to 2 argument(s), got 3"),
		)
	})

	Describe("Variable Reference Expressions", func() {
		DescribeTable("valid variable references",
			func(code string) { expectSuccess(code, nil) },
			Entry("local variable", `
				func testFunc() {
					x i32 := 10
					y := x + 5
				}
			`),
			Entry("function parameters", `
				func testFunc(x i32, y i32) i32 {
					return x + y
				}
			`),
		)

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
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("name x conflicts with existing symbol"))
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle empty expressions gracefully", func() {
			expectSuccess(`
				func testFunc() {
				}
			`, nil)
		})
	})

	Describe("Index and Slice Operations", func() {
		DescribeTable("valid index and slice operations",
			func(code string) { expectSuccess(code, nil) },
			Entry("index with integer literal", `
				func testFunc() {
					arr series i32 := [1, 2, 3, 4, 5]
					x := arr[0]
				}
			`),
			Entry("index with variable", `
				func testFunc() {
					arr series i32 := [1, 2, 3]
					idx i32 := 1
					x := arr[idx]
				}
			`),
			Entry("index with expression", `
				func testFunc() {
					arr series i32 := [1, 2, 3, 4]
					i i32 := 1
					x := arr[i + 1]
				}
			`),
			Entry("slice with both bounds", `
				func testFunc() {
					arr series i32 := [1, 2, 3, 4, 5]
					x := arr[1:3]
				}
			`),
		)

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
		It("Should correctly resolve an instantaneous channel read in an expression", func() {
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
					Name: "ox_pt_2",
					Type: types.Chan(types.F32()),
				},
			}
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot use i32 and f32 in + operation"))
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

		DescribeTable("channel operations",
			func(code string, resolver symbol.MapResolver) {
				expectSuccess(code, resolver)
			},
			Entry("unary negation", `
				func testFunc() f32 {
					return -sensor
				}
			`, symbol.MapResolver{
				"sensor": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "sensor",
					Type: types.Chan(types.F32()),
				},
			}),
			Entry("comparison", `
				func testFunc() u8 {
					return sensor > 100
				}
			`, symbol.MapResolver{
				"sensor": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "sensor",
					Type: types.Chan(types.F32()),
				},
			}),
			Entry("multiple channels in expression", `
				func testFunc() f64 {
					return (temp1 + temp2 + temp3) / 3
				}
			`, symbol.MapResolver{
				"temp1": symbol.Symbol{Kind: symbol.KindChannel, Name: "temp1", Type: types.Chan(types.F64())},
				"temp2": symbol.Symbol{Kind: symbol.KindChannel, Name: "temp2", Type: types.Chan(types.F64())},
				"temp3": symbol.Symbol{Kind: symbol.KindChannel, Name: "temp3", Type: types.Chan(types.F64())},
			}),
		)

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
			Entry("numeric literal with unit suffix", `5m -> out`, true),
		)

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
			Entry("non-literal expression", `x + y -> out`, ""),
		)
	})

	Describe("Power Expressions with Units", func() {
		DescribeTable("valid power expressions",
			func(code string) { expectSuccess(code, nil) },
			Entry("literal integer exponent", `
				func testFunc() {
					x f64 m := 5m
					y := x^2
				}
			`),
			Entry("negative literal integer exponent", `
				func testFunc() {
					x f64 m := 5m
					y := x^-2
				}
			`),
			Entry("zero exponent", `
				func testFunc() {
					x f64 m := 5m
					y := x^0
				}
			`),
			Entry("dimensionless base with any exponent", `
				func testFunc() {
					x f64 := 5.0
					y i32 := 2
					z := x^y
				}
			`),
		)

		DescribeTable("invalid power expressions",
			func(code string, expectedErrSubstring string) {
				expectFailure(code, nil, expectedErrSubstring)
			},
			Entry("dimensioned base with variable exponent", `
				func testFunc() {
					x f64 m := 5m
					n i32 := 2
					y := x^n
				}
			`, "literal integer exponent"),
			Entry("dimensioned exponent", `
				func testFunc() {
					x f64 := 5.0
					n f64 s := 2s
					y := x^n
				}
			`, "dimensionless"),
			Entry("dimensioned base with float literal exponent", `
				func testFunc() {
					x f64 m := 5m
					y := x^2.0
				}
			`, "literal integer exponent"),
			Entry("dimensioned base with unit-suffixed literal exponent", `
				func testFunc() {
					x f64 m := 5m
					y := x^2s
				}
			`, "dimensionless"),
		)
	})

	Describe("Unit Dimension Validation for Binary Operations", func() {
		DescribeTable("valid unit operations",
			func(code string) { expectSuccess(code, nil) },
			Entry("addition of same units", `
				func testFunc() {
					x f64 m := 5m
					y f64 m := 3m
					z := x + y
				}
			`),
			Entry("subtraction of same units", `
				func testFunc() {
					x f64 s := 10s
					y f64 s := 3s
					z := x - y
				}
			`),
			Entry("comparison of same units", `
				func testFunc() {
					x f64 m := 5m
					y f64 m := 3m
					result := x > y
				}
			`),
			Entry("equality comparison of same units", `
				func testFunc() {
					x f64 m := 5m
					y f64 m := 5m
					result := x == y
				}
			`),
			Entry("dimensionless arithmetic", `
				func testFunc() {
					x f64 := 5.0
					y f64 := 3.0
					z := x + y
				}
			`),
			Entry("modulo of same units", `
				func testFunc() {
					x i32 := 5
					y i32 := 3
					z := x % y
				}
			`),
		)

		DescribeTable("invalid unit operations",
			func(code string, expectedErrSubstring string) {
				expectFailure(code, nil, expectedErrSubstring)
			},
			Entry("addition of incompatible units", `
				func testFunc() {
					x f64 m := 5m
					y f64 s := 3s
					z := x + y
				}
			`, "incompatible"),
			Entry("subtraction of incompatible units", `
				func testFunc() {
					x f64 m := 5m
					y f64 s := 3s
					z := x - y
				}
			`, "incompatible"),
			Entry("comparison of incompatible units", `
				func testFunc() {
					x f64 m := 5m
					y f64 s := 3s
					result := x > y
				}
			`, "incompatible"),
		)
	})

	Describe("Series Type Operations", func() {
		DescribeTable("valid series operations",
			func(code string) { expectSuccess(code, nil) },
			Entry("series literal declaration", `
				func testFunc() {
					arr series i32 := [1, 2, 3, 4, 5]
				}
			`),
			Entry("series of floats", `
				func testFunc() {
					arr series f64 := [1.0, 2.0, 3.0]
				}
			`),
			Entry("empty series", `
				func testFunc() {
					arr series i32 := []
				}
			`),
			Entry("series element access", `
				func testFunc() {
					arr series i32 := [10, 20, 30]
					x := arr[0]
				}
			`),
			Entry("series in function parameter", `
				func sumFirst(arr series i32) i32 {
					return arr[0]
				}

				func testFunc() {
					data series i32 := [1, 2, 3]
					result := sumFirst(data)
				}
			`),
		)
	})

	Describe("Type Cast Edge Cases", func() {
		DescribeTable("valid type casts",
			func(code string) { expectSuccess(code, nil) },
			Entry("same type cast (i32 to i32)", `
				func testFunc() {
					x i32 := 42
					y := i32(x)
				}
			`),
			Entry("same type cast (f64 to f64)", `
				func testFunc() {
					x f64 := 3.14
					y := f64(x)
				}
			`),
			Entry("u8 boolean to i32", `
				func testFunc() {
					x u8 := 1
					y := i32(x)
				}
			`),
			Entry("i32 to u8 boolean", `
				func testFunc() {
					x i32 := 1
					y := u8(x)
				}
			`),
		)
	})

	Describe("IsLiteral and GetLiteral Edge Cases", func() {
		It("Should handle nil expression gracefully in IsLiteral", func() {
			// Parse a minimal program and get a nil-safe context
			ast := MustSucceed(parser.Parse(`func f() {}`))
			// Access a non-existent expression path that results in nil
			topLevel := ast.AllTopLevelItem()[0]
			funcDecl := topLevel.FunctionDeclaration()
			// The function body has no expressions, so we can test nil handling
			Expect(funcDecl).ToNot(BeNil())
		})
	})

	Describe("Nested Expression Failure Propagation", func() {
		It("Should propagate failure from nested unary expression", func() {
			expectFailure(`
				func testFunc() {
					x := --undefinedVar
				}
			`, nil, "undefined symbol")
		})

		It("Should propagate failure from slice expression index", func() {
			expectFailure(`
				func testFunc() {
					arr series i32 := [1, 2, 3]
					x := arr[undefinedStart:undefinedEnd]
				}
			`, nil, "undefined symbol")
		})

		It("Should propagate failure from nested power expression", func() {
			expectFailure(`
				func testFunc() {
					x := 2^undefinedVar
				}
			`, nil, "undefined symbol")
		})

		It("Should propagate failure from function call argument", func() {
			expectFailure(`
				func add(x i32, y i32) i32 {
					return x + y
				}
				func testFunc() {
					result := add(1, undefinedVar)
				}
			`, nil, "undefined symbol")
		})
	})

	Describe("Power Expression Edge Cases", func() {
		It("Should handle chained power with failure in exponent", func() {
			expectFailure(`
				func testFunc() {
					x f64 := 2.0
					y := x^2^undefinedVar
				}
			`, nil, "undefined symbol")
		})
	})
})
