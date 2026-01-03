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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AnalyzeFunctionBody", func() {
	Describe("Single return statement", func() {
		It("Should infer i32 from single explicit i32 return", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 42
				return x
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I32()))
		})

		It("Should infer i64 from single integer literal return", func() {
			block := MustSucceed(parser.ParseBlock(`{
				return 42
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.Kind).To(Equal(types.KindI64))
		})

		It("Should infer f64 from single float literal return", func() {
			block := MustSucceed(parser.ParseBlock(`{
				return 3.14
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.Kind).To(Equal(types.KindF64))
		})

		It("Should infer string from single string return", func() {
			block := MustSucceed(parser.ParseBlock(`{
				return "hello"
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.String()))
		})

		It("Should infer f32 from single explicit f32 return", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x f32 := 1.5
				return x
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("Should infer f32 from an integer constant and an f32 channel", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (u8_channel == 1) {
					return 0
				}
				return f32_chan
			}`))
			globalResolver := symbol.MapResolver{
				"u8_channel": symbol.Symbol{
					Name: "u8_channel",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
				},
				"f32_chan": symbol.Symbol{
					Name: "f32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			}
			ctx := context.CreateRoot(bCtx, block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("Should infer f32 from integer constant and f32 series", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (condition == 1) {
					return 0
				}
				return f32_series
			}`))
			globalResolver := symbol.MapResolver{
				"condition": symbol.Symbol{
					Name: "condition",
					Kind: symbol.KindVariable,
					Type: types.U8(),
				},
				"f32_series": symbol.Symbol{
					Name: "f32_series",
					Kind: symbol.KindVariable,
					Type: types.Series(types.F32()),
				},
			}
			ctx := context.CreateRoot(bCtx, block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})
		It("Should infer f32 from integer constant and plain f32 variable", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x f32 := 3.5
				if (x > 0) {
					return 0
				}
				return x
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("Should infer f64 from integer constant and f64 channel", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (condition == 1) {
					return 0
				}
				return f64_chan
			}`))
			globalResolver := symbol.MapResolver{
				"condition": symbol.Symbol{
					Name: "condition",
					Kind: symbol.KindVariable,
					Type: types.U8(),
				},
				"f64_chan": symbol.Symbol{
					Name: "f64_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
				},
			}
			ctx := context.CreateRoot(bCtx, block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F64()))
		})
		It("Should infer i32 from integer constant and i32 channel", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (condition == 1) {
					return 0
				}
				return i32_chan
			}`))
			globalResolver := symbol.MapResolver{
				"condition": symbol.Symbol{
					Name: "condition",
					Kind: symbol.KindVariable,
					Type: types.U8(),
				},
				"i32_chan": symbol.Symbol{
					Name: "i32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.I32()),
				},
			}
			ctx := context.CreateRoot(bCtx, block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I32()))
		})

		It("Should infer f32 from multiple integer constants and one f32 channel", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (a == 1) {
					return 0
				} else if (a == 2) {
					return 1
				} else if (a == 3) {
					return 2
				}
				return f32_chan
			}`))
			globalResolver := symbol.MapResolver{
				"a": symbol.Symbol{
					Name: "a",
					Kind: symbol.KindVariable,
					Type: types.U8(),
				},
				"f32_chan": symbol.Symbol{
					Name: "f32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			}
			ctx := context.CreateRoot(bCtx, block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("Should infer the correct type for channel and literal operations in power expressions", func() {
			block := MustSucceed(parser.ParseBlock(`{
				return f32_chan ^ 2
			}`))
			globalResolver := symbol.MapResolver{
				"f32_chan": symbol.Symbol{
					Name: "f32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			}
			ctx := context.CreateRoot(bCtx, block, globalResolver)
			inferredType, ok := statement.AnalyzeFunctionBody(ctx)
			Expect(ok).To(BeTrue(), ctx.Diagnostics.String())
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("Should infer f32 from float constant and f32 channel", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (condition == 1) {
					return 3.14
				}
				return f32_chan
			}`))
			globalResolver := symbol.MapResolver{
				"condition": symbol.Symbol{
					Name: "condition",
					Kind: symbol.KindVariable,
					Type: types.U8(),
				},
				"f32_chan": symbol.Symbol{
					Name: "f32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			}
			ctx := context.CreateRoot(bCtx, block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})
		It("Should infer f32 from channel and plain variable of same type", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x f32 := 2.5
				if (x > 0) {
					return f32_chan
				}
				return x
			}`))
			globalResolver := symbol.MapResolver{
				"f32_chan": symbol.Symbol{
					Name: "f32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			}
			ctx := context.CreateRoot(bCtx, block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})
	})

	Describe("Multiple return statements", func() {
		It("Should accept matching i32 types", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				y i32 := 2
				if x > 0 {
					return x
				}
				return y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I32()))
		})

		It("Should unify compatible integer types (i8 and i32)", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i8 := 1
				y i32 := 100
				if x > 0 {
					return x
				}
				return y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I32()))
		})

		It("Should unify compatible unsigned types (u8 and u16)", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x u8 := 1
				y u16 := 100
				if x > 0 {
					return x
				}
				return y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.U16()))
		})

		It("Should unify signed and unsigned to larger signed type", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i8 := 1
				y u8 := 2
				if x > 0 {
					return x
				}
				return y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I16()))
		})

		It("Should unify f32 and f64 to f64", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x f32 := 1.0
				y f64 := 2.0
				if x > 0 {
					return x
				}
				return y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F64()))
		})

		It("Should error on incompatible types (i32 and string)", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				if x > 0 {
					return x
				}
				return "hello"
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			_, ok := statement.AnalyzeFunctionBody(ctx)
			Expect(ok).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible return types"))
		})

		It("Should error on int/float mismatch", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				y f32 := 1.0
				if x > 0 {
					return x
				}
				return y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			_, ok := statement.AnalyzeFunctionBody(ctx)
			Expect(ok).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("mixed integer and floating-point"))
		})
	})

	Describe("Nested control flow", func() {
		It("Should recursively infer from nested if statements", func() {
			block := MustSucceed(parser.ParseBlock(`{
				a i32 := 1
				b i32 := 2
				if a > 0 {
					if b > 0 {
						return a
					} else {
						return b
					}
				} else {
					return 3
				}
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I32()))
		})

		It("Should handle if-else-if chains", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				if x > 10 {
					return 1
				} else if x > 5 {
					return 2
				} else {
					return 3
				}
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.Kind).To(Equal(types.KindI64))
		})

		It("Should handle deeply nested if statements", func() {
			block := MustSucceed(parser.ParseBlock(`{
				a u8 := 1
				b u16 := 2
				c u32 := 3
				if a > 0 {
					if b > 0 {
						if c > 0 {
							return a
						} else {
							return b
						}
					} else {
						return c
					}
				}
				return 4
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.U32()))
		})
	})

	Describe("No return statements", func() {
		It("Should return invalid type for empty block", func() {
			block := MustSucceed(parser.ParseBlock(`{}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.IsValid()).To(BeFalse())
		})

		It("Should return invalid type for block with only declarations", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				y i32 := 2
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.IsValid()).To(BeFalse())
		})

		It("Should return invalid type for block with only expressions", func() {
			block := MustSucceed(parser.ParseBlock(`{
				1 + 2
				3 * 4
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.IsValid()).To(BeFalse())
		})
	})

	Describe("Partial returns", func() {
		It("Should infer type even if not all paths return", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				if x > 0 {
					return x
				}
				y i32 := 2
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I32()))
		})

		It("Should collect returns from multiple non-exhaustive branches", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i8 := 1
				y i16 := 2
				if x > 0 {
					return x
				}
				if y > 0 {
					return y
				}
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I16()))
		})

		It("Should handle mixed complete and partial paths", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				if x > 10 {
					return 100
				} else if x > 5 {
					y i32 := x * 2
				}
				return x
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I32()))
		})
	})

	Describe("Edge cases", func() {
		It("Should handle return with no expression (void return)", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				if x > 0 {
					return
				}
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.IsValid()).To(BeFalse())
		})

		It("Should handle all same type returns", func() {
			block := MustSucceed(parser.ParseBlock(`{
				return 1
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.Kind).To(Or(Equal(types.KindVariable), Equal(types.KindI64)))
		})

		It("Should handle complex integer size unification", func() {
			block := MustSucceed(parser.ParseBlock(`{
				a i8 := 1
				b i16 := 2
				c i32 := 3
				d i64 := 4
				if a > 0 {
					return a
				} else if b > 0 {
					return b
				} else if c > 0 {
					return c
				} else {
					return d
				}
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I64()))
		})

		It("Should handle large unsigned values", func() {
			block := MustSucceed(parser.ParseBlock(`{
				a u32 := 1
				b u64 := 2
				if a > 0 {
					return a
				}
				return b
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.U64()))
		})
	})

	Describe("Error handling", func() {
		It("Should fail gracefully on analysis errors", func() {
			block := MustSucceed(parser.ParseBlock(`{
				return undefined_var
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			_, ok := statement.AnalyzeFunctionBody(ctx)
			Expect(ok).To(BeFalse())
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
		})

		It("Should report type incompatibility clearly", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				y str := "hello"
				if x > 0 {
					return x
				}
				return y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			_, ok := statement.AnalyzeFunctionBody(ctx)
			Expect(ok).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible return types"))
		})
	})

	Describe("Type preservation", func() {
		It("Should preserve exact types when possible", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i8 := 1
				y i8 := 2
				if x > 0 {
					return x
				}
				return y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.I8()))
		})

		It("Should use smallest reasonable type for mixed unsigned", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x u8 := 1
				y u8 := 2
				z u16 := 300
				if x > 0 {
					return x
				} else if y > 0 {
					return y
				}
				return z
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.U16()))
		})

	})
})
