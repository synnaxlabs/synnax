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
	"github.com/synnaxlabs/arc/analyzer/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
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
						x string := "hello"
						y string := "world"
						z := x + y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Severity).To(Equal(diagnostics.Error))
			Expect(first.Message).To(ContainSubstring("cannot use string in + operation"))
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

		It("Should not allow comparison of an integer variable with a floating point literal", func() {
			ast := MustSucceed(parser.Parse(`
			func testFunc() {
				x i32 := 10
				z := x > 5.0
			}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(Equal("type mismatch: cannot use i32 and f64 in > operation"))
		})

		It("Should validate logical operations on booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						a u8 := 0
						b u8 := 1
						c := a && b
						d := a || b
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
						z := x && y
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("cannot use i32 in && operation"))
		})

		It("Should reject logical OR operations on non-booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
					x i32 := 10
					y i32 := 20
					z := x || y
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
						x string := "hello"
						y := -x
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("operator - not supported for type string"))
		})

		It("Should validate logical not on booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x u8 := 1
						y := !x
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should reject logical not on non-booleans", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y := !x
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("operator ! requires boolean operand"))
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
						y string := "test"
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should correctly type boolean literals", func() {
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
						result := a && b || c
						result2 := a || b && c
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
						result := a && b || (x + y == z)
					}
				`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should detect type errors in complex expressions", func() {
			ast := MustSucceed(parser.Parse(`
					func testFunc() {
						x i32 := 10
						y string := "20"
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
			resolver := ir.MapResolver{
				"ox_pt_1": ir.Symbol{
					Kind: ir.KindChannel,
					Name: "ox_pt_1",
					Type: ir.Chan{ValueType: ir.I32{}},
				},
				"ox_pt_2": ir.Symbol{
					Kind: ir.KindChannel,
					Name: "ox_pt_2",
					Type: ir.Chan{ValueType: ir.I32{}},
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
			resolver := ir.MapResolver{
				"ox_pt_1": ir.Symbol{
					Kind: ir.KindChannel,
					Name: "ox_pt_1",
					Type: ir.Chan{ValueType: ir.I32{}},
				},
				"ox_pt_2": ir.Symbol{
					Kind: ir.KindChannel,
					Name: "ox_pt_1",
					Type: ir.Chan{ValueType: ir.F32{}},
				},
			}
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			firstDiag := (*ctx.Diagnostics)[0]
			Expect(firstDiag.Message).To(ContainSubstring("type mismatch: cannot use chan i32 and chan f32 in + operation"))
		})

		It("Should not return an error when adding a channel to a variable of the same type", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() i32 {
					return ox_pt_1 + 2
				}
			`))
			resolver := ir.MapResolver{
				"ox_pt_1": ir.Symbol{
					Kind: ir.KindChannel,
					Name: "ox_pt_1",
					Type: ir.Chan{ValueType: ir.I32{}},
				},
			}
			ctx := context.CreateRoot(bCtx, ast, resolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})
})
