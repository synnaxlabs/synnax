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
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Type Casts", func() {
	Describe("Integer to Float Casts", func() {
		It("should allow casting i32 to f32", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 10
					y := f32(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting i64 to f64", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i64 := 100
					y := f64(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting integer literal to float", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					y := f32(42)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Float to Integer Casts", func() {
		It("should allow casting f32 to i32", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f32 := 3.14
					y := i32(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting f64 to i64", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 := 3.14159
					y := i64(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting float literal to integer", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					y := i32(3.14)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Integer Width Conversions", func() {
		It("should allow casting i32 to i64 (widening)", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 10
					y := i64(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting i64 to i32 (narrowing)", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i64 := 10
					y := i32(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting u32 to u64 (widening)", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x u32 := 10
					y := u64(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting i8 to i32", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i8 := 10
					y := i32(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Signed/Unsigned Conversions", func() {
		It("should allow casting i32 to u32", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 10
					y := u32(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting u32 to i32", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x u32 := 10
					y := i32(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Float Width Conversions", func() {
		It("should allow casting f32 to f64 (widening)", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f32 := 3.14
					y := f64(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow casting f64 to f32 (narrowing)", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 := 3.14159
					y := f32(x)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Type Casts in Expressions", func() {
		It("should allow type cast in arithmetic expression", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 10
					y f32 := 3.14
					result := f32(x) + y
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow nested type casts", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x f64 := 3.14159
					y := i32(f32(x))
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should allow type cast of complex expression", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 10
					y i32 := 20
					result := f32(x + y)
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Boolean Casts", func() {
		It("should allow casting u8 to boolean context", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x u8 := 1
					y u8 := 0
					result := x && y
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Edge Cases", func() {
		It("should handle type cast of literal in function call", func() {
			ast := MustSucceed(parser.Parse(`
				func takeFloat(x f32) f32 {
					return x
				}

				func testFunc() {
					result := takeFloat(f32(10))
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("should handle type cast in comparison", func() {
			ast := MustSucceed(parser.Parse(`
				func testFunc() {
					x i32 := 10
					result := f32(x) > 5.0
				}
			`))
			ctx := context.CreateRoot(bCtx, ast, nil)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
		})
	})
})