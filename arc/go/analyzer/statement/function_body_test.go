// Copyright 2025 Synnax Labs, Inc.
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
	// Helper to create a context with a global resolver
	createContextWithResolver := func(
		block parser.IBlockContext,
		resolver symbol.Resolver,
	) context.Context[parser.IBlockContext] {
		return context.CreateRoot(bCtx, block, resolver)
	}

	// Helper to create a context without a resolver
	createContext := func(block parser.IBlockContext) context.Context[parser.IBlockContext] {
		return context.CreateRoot(bCtx, block, nil)
	}

	Context("when inferring return type from single return statement", func() {
		DescribeTable("explicit variable types",
			func(code string, expected types.Type) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType).To(Equal(expected))
			},
			Entry("i32 variable", `{
				x i32 := 42
				return x
			}`, types.I32()),
			Entry("f32 variable", `{
				x f32 := 1.5
				return x
			}`, types.F32()),
		)

		DescribeTable("literal returns",
			func(code string, expectedKind types.TypeKind) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType.Kind).To(Equal(expectedKind))
			},
			Entry("integer literal", `{ return 42 }`, types.KindI64),
			Entry("float literal", `{ return 3.14 }`, types.KindF64),
		)

		It("should infer string from single string return", func() {
			block := MustSucceed(parser.ParseBlock(`{ return "hello" }`))
			ctx := createContext(block)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.String()))
		})

		DescribeTable("channel type inference with integer constants",
			func(code string, chanName string, chanType types.Type, expected types.Type) {
				globalResolver := symbol.MapResolver{
					"condition": symbol.Symbol{
						Name: "condition",
						Kind: symbol.KindVariable,
						Type: types.U8(),
					},
					"u8_channel": symbol.Symbol{
						Name: "u8_channel",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U8()),
					},
					chanName: symbol.Symbol{
						Name: chanName,
						Kind: symbol.KindChannel,
						Type: chanType,
					},
				}
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContextWithResolver(block, globalResolver)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType).To(Equal(expected))
			},
			Entry("f32 channel with integer constant", `{
				if (u8_channel == 1) { return 0 }
				return f32_chan
			}`, "f32_chan", types.Chan(types.F32()), types.F32()),
			Entry("f64 channel with integer constant", `{
				if (condition == 1) { return 0 }
				return f64_chan
			}`, "f64_chan", types.Chan(types.F64()), types.F64()),
			Entry("i32 channel with integer constant", `{
				if (condition == 1) { return 0 }
				return i32_chan
			}`, "i32_chan", types.Chan(types.I32()), types.I32()),
		)

		It("should infer f32 from integer constant and f32 series", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (condition == 1) { return 0 }
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
			ctx := createContextWithResolver(block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("should infer f32 from integer constant and plain f32 variable", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x f32 := 3.5
				if (x > 0) { return 0 }
				return x
			}`))
			ctx := createContext(block)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("should infer f32 from multiple integer constants and one f32 channel", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (a == 1) { return 0 }
				else if (a == 2) { return 1 }
				else if (a == 3) { return 2 }
				return f32_chan
			}`))
			globalResolver := symbol.MapResolver{
				"a": symbol.Symbol{Name: "a", Kind: symbol.KindVariable, Type: types.U8()},
				"f32_chan": symbol.Symbol{
					Name: "f32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			}
			ctx := createContextWithResolver(block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("should infer the correct type for channel and literal operations in power expressions", func() {
			block := MustSucceed(parser.ParseBlock(`{ return f32_chan ^ 2 }`))
			globalResolver := symbol.MapResolver{
				"f32_chan": symbol.Symbol{
					Name: "f32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			}
			ctx := createContextWithResolver(block, globalResolver)
			inferredType, ok := statement.AnalyzeFunctionBody(ctx)
			Expect(ok).To(BeTrue(), ctx.Diagnostics.String())
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("should infer f32 from float constant and f32 channel", func() {
			block := MustSucceed(parser.ParseBlock(`{
				if (condition == 1) { return 3.14 }
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
			ctx := createContextWithResolver(block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})

		It("should infer f32 from channel and plain variable of same type", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x f32 := 2.5
				if (x > 0) { return f32_chan }
				return x
			}`))
			globalResolver := symbol.MapResolver{
				"f32_chan": symbol.Symbol{
					Name: "f32_chan",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
				},
			}
			ctx := createContextWithResolver(block, globalResolver)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType).To(Equal(types.F32()))
		})
	})

	Context("when unifying multiple return types", func() {
		DescribeTable("compatible numeric types",
			func(code string, expected types.Type) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType).To(Equal(expected))
			},
			Entry("matching i32 types", `{
				x i32 := 1
				y i32 := 2
				if x > 0 { return x }
				return y
			}`, types.I32()),
			Entry("i8 and i32 unify to i32", `{
				x i8 := 1
				y i32 := 100
				if x > 0 { return x }
				return y
			}`, types.I32()),
			Entry("u8 and u16 unify to u16", `{
				x u8 := 1
				y u16 := 100
				if x > 0 { return x }
				return y
			}`, types.U16()),
			Entry("i8 and u8 unify to i16 (signed and unsigned)", `{
				x i8 := 1
				y u8 := 2
				if x > 0 { return x }
				return y
			}`, types.I16()),
			Entry("f32 and f64 unify to f64", `{
				x f32 := 1.0
				y f64 := 2.0
				if x > 0 { return x }
				return y
			}`, types.F64()),
		)

		DescribeTable("incompatible types produce errors",
			func(code string, expectedErrSubstring string) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				_, ok := statement.AnalyzeFunctionBody(ctx)
				Expect(ok).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(expectedErrSubstring))
			},
			Entry("i32 and string", `{
				x i32 := 1
				if x > 0 { return x }
				return "hello"
			}`, "incompatible return types"),
			Entry("i32 and f32 (mixed integer/float)", `{
				x i32 := 1
				y f32 := 1.0
				if x > 0 { return x }
				return y
			}`, "mixed integer and floating-point"),
		)
	})

	Context("when handling nested control flow", func() {
		DescribeTable("nested if statements",
			func(code string, expected types.Type) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType).To(Equal(expected))
			},
			Entry("nested if-else", `{
				a i32 := 1
				b i32 := 2
				if a > 0 {
					if b > 0 { return a }
					else { return b }
				} else { return 3 }
			}`, types.I32()),
			Entry("deeply nested if", `{
				a u8 := 1
				b u16 := 2
				c u32 := 3
				if a > 0 {
					if b > 0 {
						if c > 0 { return a }
						else { return b }
					} else { return c }
				}
				return 4
			}`, types.U32()),
		)

		It("should handle if-else-if chains", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				if x > 10 { return 1 }
				else if x > 5 { return 2 }
				else { return 3 }
			}`))
			ctx := createContext(block)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.Kind).To(Equal(types.KindI64))
		})
	})

	Context("when no return statements exist", func() {
		DescribeTable("returns invalid type",
			func(code string) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType.IsValid()).To(BeFalse())
			},
			Entry("empty block", `{}`),
			Entry("block with only declarations", `{
				x i32 := 1
				y i32 := 2
			}`),
			Entry("block with only expressions", `{
				1 + 2
				3 * 4
			}`),
		)
	})

	Context("when some paths return but not all", func() {
		DescribeTable("partial returns",
			func(code string, expected types.Type) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType).To(Equal(expected))
			},
			Entry("only if branch returns", `{
				x i32 := 1
				if x > 0 { return x }
				y i32 := 2
			}`, types.I32()),
			Entry("multiple non-exhaustive branches", `{
				x i8 := 1
				y i16 := 2
				if x > 0 { return x }
				if y > 0 { return y }
			}`, types.I16()),
			Entry("mixed complete and partial paths", `{
				x i32 := 1
				if x > 10 { return 100 }
				else if x > 5 { y i32 := x * 2 }
				return x
			}`, types.I32()),
		)
	})

	Context("edge cases", func() {
		It("should handle return with no expression (void return)", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				if x > 0 { return }
			}`))
			ctx := createContext(block)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.IsValid()).To(BeFalse())
		})

		It("should handle all same type returns", func() {
			block := MustSucceed(parser.ParseBlock(`{ return 1 }`))
			ctx := createContext(block)
			inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
			Expect(inferredType.Kind).To(Or(Equal(types.KindVariable), Equal(types.KindI64)))
		})

		DescribeTable("integer size unification",
			func(code string, expected types.Type) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType).To(Equal(expected))
			},
			Entry("complex signed integer sizes", `{
				a i8 := 1
				b i16 := 2
				c i32 := 3
				d i64 := 4
				if a > 0 { return a }
				else if b > 0 { return b }
				else if c > 0 { return c }
				else { return d }
			}`, types.I64()),
			Entry("large unsigned values", `{
				a u32 := 1
				b u64 := 2
				if a > 0 { return a }
				return b
			}`, types.U64()),
		)
	})

	Context("error handling", func() {
		It("should fail gracefully on undefined variable", func() {
			block := MustSucceed(parser.ParseBlock(`{ return undefined_var }`))
			ctx := createContext(block)
			_, ok := statement.AnalyzeFunctionBody(ctx)
			Expect(ok).To(BeFalse())
			Expect(*ctx.Diagnostics).ToNot(BeEmpty())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol"))
		})

		It("should report type incompatibility clearly", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 1
				y str := "hello"
				if x > 0 { return x }
				return y
			}`))
			ctx := createContext(block)
			_, ok := statement.AnalyzeFunctionBody(ctx)
			Expect(ok).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible return types"))
		})
	})

	Context("type preservation", func() {
		DescribeTable("preserves exact types when possible",
			func(code string, expected types.Type) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := createContext(block)
				inferredType := MustBeOk(statement.AnalyzeFunctionBody(ctx))
				Expect(inferredType).To(Equal(expected))
			},
			Entry("matching i8 types", `{
				x i8 := 1
				y i8 := 2
				if x > 0 { return x }
				return y
			}`, types.I8()),
			Entry("smallest reasonable unsigned type", `{
				x u8 := 1
				y u8 := 2
				z u16 := 300
				if x > 0 { return x }
				else if y > 0 { return y }
				return z
			}`, types.U16()),
		)
	})
})
