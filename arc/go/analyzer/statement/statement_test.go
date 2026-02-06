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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Statement", func() {
	// Helper to set up function context for tests that need it
	setupFunctionContext := func(ctx context.Context[parser.IBlockContext]) {
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
			DescribeTable("type inference and validation",
				func(code string, expectOk bool, assertion func(context.Context[parser.IStatementContext])) {
					stmt := MustSucceed(parser.ParseStatement(code))
					ctx := context.CreateRoot(bCtx, stmt, nil)
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(Equal(expectOk))
					if assertion != nil {
						assertion(ctx)
					}
				},
				Entry("explicit type with initializer", `x i32 := 42`, true, func(ctx context.Context[parser.IStatementContext]) {
					sym := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
					Expect(sym.Type).To(Equal(types.I32()))
				}),
				Entry("inferred type from float literal", `x := 3.14`, true, func(ctx context.Context[parser.IStatementContext]) {
					Expect(*ctx.Diagnostics).To(BeEmpty())
					sym := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
					Expect(sym.Type.Kind).To(Equal(types.KindVariable))
					Expect(sym.Type.Constraint).ToNot(BeNil())
					Expect(sym.Type.Constraint.Kind).To(Equal(types.KindFloatConstant))
				}),
			)

			It("should detect type mismatch between declaration and initializer", func() {
				stmt := MustSucceed(parser.ParseStatement(`x i32 := "hello"`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot assign str to 'x' (type i32)"))
			})

			It("should detect duplicate variable declaration", func() {
				stmt := MustSucceed(parser.ParseBlock(`{
					x := 1
					x := 1
				}`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("name x conflicts with existing symbol"))
			})

			It("should detect undefined variable in initializer", func() {
				stmt := MustSucceed(parser.ParseStatement(`x := y + 1`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: y"))
			})
		})

		Context("stateful variables", func() {
			It("should analyze a stateful variable with inferred type", func() {
				stmt := MustSucceed(parser.ParseStatement(`counter $= 0`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				sym := MustSucceed(ctx.Scope.Resolve(ctx, "counter"))
				Expect(sym.Kind).To(Equal(symbol.KindStatefulVariable))
				Expect(sym.Type.Kind).To(Equal(types.KindVariable))
				Expect(sym.Type.Constraint).ToNot(BeNil())
				Expect(sym.Type.Constraint.Kind).To(Equal(types.KindIntegerConstant))
			})

			It("should analyze stateful variable with explicit type", func() {
				stmt := MustSucceed(parser.ParseStatement(`total f32 $= 0.0`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				sym := MustSucceed(ctx.Scope.Resolve(ctx, "total"))
				Expect(sym.Type).To(Equal(types.F32()))
			})
		})
	})

	Describe("Assignment", func() {
		DescribeTable("assignment validation",
			func(code string, expectOk bool, errorSubstring string) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := context.CreateRoot(bCtx, block, nil)
				setupFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(Equal(expectOk))
				if !expectOk && errorSubstring != "" {
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(errorSubstring))
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
	})

	Describe("If Statement", func() {
		DescribeTable("valid if statements",
			func(code string) {
				stmt := MustSucceed(parser.ParseStatement(code))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				statement.Analyze(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			},
			Entry("simple if", `if 1 { x := 42 }`),
			Entry("if-else chain", `if 0 { x := 1 } else if 1 { y := 2 } else { z := 3 }`),
			Entry("nested blocks", `if 1 {
				x := 42
				if 1 { y := x + 1 }
			}`),
		)

		It("should detect undefined variable in condition", func() {
			stmt := MustSucceed(parser.ParseStatement(`if x > 10 { y := 1 }`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			statement.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: x"))
		})
	})

	Describe("Block", func() {
		DescribeTable("block analysis",
			func(code string, expectOk bool, errorSubstring string) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(Equal(expectOk))
				if !expectOk && errorSubstring != "" {
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(errorSubstring))
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
		It("should analyze standalone expression with existing variable", func() {
			stmt := MustSucceed(parser.ParseStatement(`x + 1`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "x",
				Kind: symbol.KindVariable,
				Type: types.I64(),
			}))
			Expect(scope).ToNot(BeNil())
			statement.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("should detect errors in standalone expression", func() {
			stmt := MustSucceed(parser.ParseStatement(`undefined_var + 1`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			statement.Analyze(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol"))
		})
	})

	Describe("Channel Operations", func() {
		var channelResolver symbol.MapResolver

		BeforeEach(func() {
			channelResolver = symbol.MapResolver{
				"sensor": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "sensor",
					Type: types.Chan(types.F64()),
				},
				ir.DefaultOutputParam: symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: ir.DefaultOutputParam,
					Type: types.Chan(types.F64()),
				},
				"int_chan": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "int_chan",
					Type: types.Chan(types.I32()),
				},
				"string_chan": symbol.Symbol{
					Kind: symbol.KindChannel,
					Name: "string_chan",
					Type: types.Chan(types.String()),
				},
			}
		})

		setupChannelFunctionContext := func(ctx context.Context[parser.IBlockContext]) {
			scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "testFunc",
				Kind: symbol.KindFunction,
			}))
			Expect(scope).ToNot(BeNil())
			fn := MustSucceed(ctx.Scope.Resolve(ctx, "testFunc"))
			ctx.Scope = fn
		}

		Context("channel assignment in imperative context", func() {
			DescribeTable("channel write validation",
				func(code string, expectOk bool, errorSubstring string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot[parser.IBlockContext](bCtx, block, channelResolver)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(Equal(expectOk), ctx.Diagnostics.String())
					if !expectOk && errorSubstring != "" {
						Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(errorSubstring))
					}
				},
				Entry("literal value", `{ output = 42.0 }`, true, ""),
				Entry("variable value", `{
					value := 42.0
					output = value
				}`, true, ""),
				Entry("type mismatch", `{ output = "hello" }`, false, "type mismatch"),
			)

			It("should detect undefined channel in assignment", func() {
				block := MustSucceed(parser.ParseBlock(`{ undefined_channel = 42.0 }`))
				ctx := context.CreateRoot(bCtx, block, nil)
				setupChannelFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined"))
			})
		})

		Context("channel reads in imperative context", func() {
			DescribeTable("channel read type inference",
				func(chanName string, expectedType types.Type) {
					code := "current := " + chanName
					stmt := MustSucceed(parser.ParseStatement(code))
					ctx := context.CreateRoot(bCtx, stmt, channelResolver)
					statement.Analyze(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
					Expect(*ctx.Diagnostics).To(BeEmpty())
					varScope := MustSucceed(ctx.Scope.Resolve(ctx, "current"))
					Expect(varScope.Type).To(Equal(expectedType))
				},
				Entry("f64 channel", "sensor", types.Chan(types.F64())),
				Entry("i32 channel", "int_chan", types.Chan(types.I32())),
				Entry("i32 channel addition", "int_chan + 1", types.I32()),
			)
		})

		Context("channel alias assignment to scalar variables", func() {
			DescribeTable("should accept valid channel alias to scalar assignments",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot[parser.IBlockContext](bCtx, block, channelResolver)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("chan f64 alias assigned to f64 scalar", `{
					local_ref := sensor
					value f64 := 0.0
					value = local_ref
				}`),
				Entry("chan f64 alias assigned to stateful f64 scalar", `{
					local_ref := sensor
					value f64 $= 0.0
					value = local_ref
				}`),
				Entry("chan f64 alias assigned to inferred-type variable", `{
					local_ref := sensor
					value := 0.0
					value = local_ref
				}`),
				Entry("chan i32 alias assigned to i32 scalar", `{
					local_ref := int_chan
					value i32 := 0
					value = local_ref
				}`),
				Entry("alias-to-alias preserves chan type", `{
					local_ref := sensor
					other_ref := local_ref
				}`),
				Entry("chan alias in comparison", `{
					local_ref := sensor
					x f64 := 0.0
					if local_ref > 100.0 { x = 1.0 }
				}`),
				Entry("chan alias in arithmetic", `{
					local_ref := sensor
					result f64 := local_ref + 1.0
				}`),
				Entry("chan alias written to channel target", `{
					sensor_ref := sensor
					output = sensor_ref
				}`),
				Entry("arithmetic expr of alias assigned to scalar", `{
					local_ref := sensor
					value f64 := 0.0
					value = local_ref * 2.0
				}`),
			)

			It("should reject type mismatch after unwrapping channel alias", func() {
				block := MustSucceed(parser.ParseBlock(`{
					local_ref := int_chan
					value f64 := 0.0
					value = local_ref
				}`))
				ctx := context.CreateRoot[parser.IBlockContext](bCtx, block, channelResolver)
				setupChannelFunctionContext(ctx)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
			})
		})

		Context("series literals with channel alias elements", func() {
			DescribeTable("should accept valid series literals containing channel aliases",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot[parser.IBlockContext](bCtx, block, channelResolver)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("channel alias then exact-integer float literal", `{
					ref := sensor
					x := [ref, 1.0]
				}`),
				Entry("exact-integer float literal then channel alias", `{
					ref := sensor
					x := [1.0, ref]
				}`),
				Entry("two channel aliases of same type", `{
					ref1 := sensor
					ref2 := sensor
					x := [ref1, ref2]
				}`),
				Entry("channel alias then int literal", `{
					ref := sensor
					x := [ref, 1]
				}`),
				Entry("int literal then channel alias", `{
					ref := sensor
					x := [1, ref]
				}`),
				Entry("channel alias with arithmetic", `{
					ref := sensor
					x := [ref + 1.0, 2.0]
				}`),
				Entry("i32 channel alias then int literal", `{
					ref := int_chan
					x := [ref, 42]
				}`),
				Entry("int literal then i32 channel alias", `{
					ref := int_chan
					x := [42, ref]
				}`),
				Entry("channel alias then non-exact float literal", `{
					ref := sensor
					x := [ref, 1.5]
				}`),
				Entry("non-exact float literal then channel alias", `{
					ref := sensor
					x := [1.5, ref]
				}`),
				Entry("channel alias then pi-like float literal", `{
					ref := sensor
					x := [ref, 3.14]
				}`),
			)

			DescribeTable("should reject invalid series literals containing channel aliases",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot[parser.IBlockContext](bCtx, block, channelResolver)
					setupChannelFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
				},
				Entry("f64 channel alias then string literal", `{
					ref := sensor
					x := [ref, "hello"]
				}`),
				Entry("string literal then f64 channel alias", `{
					ref := sensor
					x := ["hello", ref]
				}`),
				Entry("f64 alias and i32 alias", `{
					f_ref := sensor
					i_ref := int_chan
					x := [f_ref, i_ref]
				}`),
				Entry("i32 alias and f64 alias", `{
					i_ref := int_chan
					f_ref := sensor
					x := [i_ref, f_ref]
				}`),
				Entry("non-exact float literal then i32 channel alias", `{
					ref := int_chan
					x := [1.5, ref]
				}`),
			)
		})
	})

	Describe("Compound Assignment", func() {
		Context("valid compound assignments", func() {
			DescribeTable("numeric types",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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

			It("should allow string concatenation with +=", func() {
				block := MustSucceed(parser.ParseBlock(`{
					s str := "hello"
					s += " world"
				}`))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			})
		})

		Context("invalid compound assignments", func() {
			var channelResolver symbol.MapResolver

			BeforeEach(func() {
				channelResolver = symbol.MapResolver{
					"sensor": symbol.Symbol{
						Kind: symbol.KindChannel,
						Name: "sensor",
						Type: types.Chan(types.F64()),
					},
				}
			})

			It("should reject compound assignment on channels", func() {
				block := MustSucceed(parser.ParseBlock(`{ sensor += 1.0 }`))
				ctx := context.CreateRoot(bCtx, block, channelResolver)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("compound assignment not supported on channels"))
			})

			DescribeTable("strings only support +=",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect(*ctx.Diagnostics).To(HaveLen(1))
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("strings only support += operator"))
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

			It("should reject compound assignment with type mismatch", func() {
				block := MustSucceed(parser.ParseBlock(`{
					x i32 := 10
					x += "hello"
				}`))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
			})

			It("Should reject compound assignments of inferred literal float vs. integer", func() {
				block := MustSucceed(parser.ParseBlock(`{
					x := 10
					y := x + 3.2
				}`))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
			})

			It("should reject compound assignment on undefined variable", func() {
				block := MustSucceed(parser.ParseBlock(`{ undefined_var += 5 }`))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: undefined_var"))
			})

			DescribeTable("valid indexed compound assignments",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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

			It("should reject slice compound assignment", func() {
				block := MustSucceed(parser.ParseBlock(`{
					arr series i32 := [1, 2, 3]
					arr[0:2] += 5
				}`))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("slice compound assignment not supported"))
			})

			It("should reject indexed compound assignment on non-series type", func() {
				block := MustSucceed(parser.ParseBlock(`{
					x i32 := 5
					x[0] += 1
				}`))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("indexed"))
			})

			// Whole-series compound assignment tests
			DescribeTable("valid whole-series compound assignments",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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

			It("should reject series compound assignment with incompatible scalar type", func() {
				block := MustSucceed(parser.ParseBlock(`{
					s series i32 := [1, 2, 3]
					s += "hello"
				}`))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
			})

			It("should reject series compound assignment with mismatched series element types", func() {
				block := MustSucceed(parser.ParseBlock(`{
					a series i32 := [1, 2, 3]
					b series f64 := [1.0, 2.0, 3.0]
					a += b
				}`))
				ctx := context.CreateRoot(bCtx, block, nil)
				statement.AnalyzeBlock(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
			})
		})
	})

	Describe("Mixed Type Scenarios", func() {
		It("should handle complex nested structures", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x := 10
				y $= 20
				if x < y {
					z := x + y
					z = z * 2
				}
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("should properly track types through assignments", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 10
				y := x
				z := y + 5
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("should return an error when assigning incompatible types", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 10
				y f32 := x
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot assign i32 to 'y' (type f32)"))
		})
	})

	Describe("Series Literals", func() {
		Context("valid series literals with literals only", func() {
			DescribeTable("should accept",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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
			var funcResolver symbol.MapResolver

			BeforeEach(func() {
				funcResolver = symbol.MapResolver{
					"getI32": symbol.Symbol{
						Name: "getI32",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.I32()}},
						}),
					},
					"anotherI32": symbol.Symbol{
						Name: "anotherI32",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.I32()}},
						}),
					},
					"getI64": symbol.Symbol{
						Name: "getI64",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.I64()}},
						}),
					},
					"getF32": symbol.Symbol{
						Name: "getF32",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.F32()}},
						}),
					},
					"getF64": symbol.Symbol{
						Name: "getF64",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.F64()}},
						}),
					},
					"getStr": symbol.Symbol{
						Name: "getStr",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.String()}},
						}),
					},
				}
			})

			DescribeTable("valid function call combinations",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, funcResolver)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("two same-type function calls", `{ x := [getI32(), getI32()] }`),
				Entry("two different functions same return type", `{ x := [getI32(), anotherI32()] }`),
				Entry("function call then literal", `{ x := [getI32(), 42] }`),
				Entry("literal then function call", `{ x := [42, getI32()] }`),
				Entry("function call then expression", `{ x := [getI32(), 1 + 2] }`),
				Entry("expression then function call", `{ x := [1 + 2, getI32()] }`),
				Entry("function call in expression then literal", `{ x := [getI32() + 1, 42] }`),
				Entry("literal then function call in expression", `{ x := [42, getI32() + 1] }`),
				Entry("function call plus literal in expression", `{ x := [1 + getI32(), 42] }`),
				Entry("three function calls", `{ x := [getI32(), getI32(), getI32()] }`),
				Entry("function call sandwiched by literals", `{ x := [1, getI32(), 2] }`),
				Entry("literal sandwiched by function calls", `{ x := [getI32(), 1, getI32()] }`),
			)

			DescribeTable("valid function call with variable combinations",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, funcResolver)
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

			DescribeTable("invalid function call combinations",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, funcResolver)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 then f64 function calls", `{ x := [getI32(), getF64()] }`),
				Entry("f64 then i32 function calls", `{ x := [getF64(), getI32()] }`),
				Entry("i32 then i64 function calls", `{ x := [getI32(), getI64()] }`),
				Entry("i32 then string function calls", `{ x := [getI32(), getStr()] }`),
				Entry("string then i32 function calls", `{ x := [getStr(), getI32()] }`),
				Entry("f32 then string function calls", `{ x := [getF32(), getStr()] }`),
				Entry("f64 then string function calls", `{ x := [getF64(), getStr()] }`),
				Entry("i32 function then string literal", `{ x := [getI32(), "hello"] }`),
				Entry("string literal then i32 function", `{ x := ["hello", getI32()] }`),
				Entry("string function then int literal", `{ x := [getStr(), 42] }`),
				Entry("int literal then string function", `{ x := [42, getStr()] }`),
				Entry("f64 function then string literal", `{ x := [getF64(), "hello"] }`),
				Entry("three functions last mismatched", `{ x := [getI32(), getI32(), getStr()] }`),
				Entry("three functions first mismatched", `{ x := [getStr(), getI32(), getI32()] }`),
				Entry("three functions middle mismatched", `{ x := [getI32(), getStr(), getI32()] }`),
			)

			DescribeTable("invalid function call with variable combinations",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, funcResolver)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
				// Inferred type variables - these should also be rejected for consistency
				// with explicit type annotations above
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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

		Context("invalid series literals - three or more elements with mismatch", func() {
			DescribeTable("should reject",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
		})

		Context("invalid series literals - expression mismatches", func() {
			DescribeTable("should reject",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
				func(code string, errorSubstring string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					setupFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(errorSubstring))
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
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
			var funcResolver symbol.MapResolver

			BeforeEach(func() {
				funcResolver = symbol.MapResolver{
					"getI32": symbol.Symbol{
						Name: "getI32",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.I32()}},
						}),
					},
					"anotherI32": symbol.Symbol{
						Name: "anotherI32",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.I32()}},
						}),
					},
					"getI64": symbol.Symbol{
						Name: "getI64",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.I64()}},
						}),
					},
					"getF32": symbol.Symbol{
						Name: "getF32",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.F32()}},
						}),
					},
					"getF64": symbol.Symbol{
						Name: "getF64",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.F64()}},
						}),
					},
					"getStr": symbol.Symbol{
						Name: "getStr",
						Kind: symbol.KindVariable,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Type: types.String()}},
						}),
					},
				}
			})

			DescribeTable("valid function call combinations",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, funcResolver)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				},
				Entry("two same-type function calls", `{ x := [getI32(), getI32()] }`),
				Entry("two different functions same return type", `{ x := [getI32(), anotherI32()] }`),
				Entry("function call then literal", `{ x := [getI32(), 42] }`),
				Entry("literal then function call", `{ x := [42, getI32()] }`),
				Entry("function call then expression", `{ x := [getI32(), 1 + 2] }`),
				Entry("expression then function call", `{ x := [1 + 2, getI32()] }`),
				Entry("function call in expression then literal", `{ x := [getI32() + 1, 42] }`),
				Entry("literal then function call in expression", `{ x := [42, getI32() + 1] }`),
				Entry("function call plus literal in expression", `{ x := [1 + getI32(), 42] }`),
				Entry("three function calls", `{ x := [getI32(), getI32(), getI32()] }`),
				Entry("function call sandwiched by literals", `{ x := [1, getI32(), 2] }`),
				Entry("literal sandwiched by function calls", `{ x := [getI32(), 1, getI32()] }`),
			)

			DescribeTable("valid function call with variable combinations",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, funcResolver)
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

			DescribeTable("invalid function call combinations",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, funcResolver)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
				},
				Entry("i32 then f64 function calls", `{ x := [getI32(), getF64()] }`),
				Entry("f64 then i32 function calls", `{ x := [getF64(), getI32()] }`),
				Entry("i32 then i64 function calls", `{ x := [getI32(), getI64()] }`),
				Entry("i32 then string function calls", `{ x := [getI32(), getStr()] }`),
				Entry("string then i32 function calls", `{ x := [getStr(), getI32()] }`),
				Entry("f32 then string function calls", `{ x := [getF32(), getStr()] }`),
				Entry("f64 then string function calls", `{ x := [getF64(), getStr()] }`),
				Entry("i32 function then string literal", `{ x := [getI32(), "hello"] }`),
				Entry("string literal then i32 function", `{ x := ["hello", getI32()] }`),
				Entry("string function then int literal", `{ x := [getStr(), 42] }`),
				Entry("int literal then string function", `{ x := [42, getStr()] }`),
				Entry("f64 function then string literal", `{ x := [getF64(), "hello"] }`),
				Entry("three functions last mismatched", `{ x := [getI32(), getI32(), getStr()] }`),
				Entry("three functions first mismatched", `{ x := [getStr(), getI32(), getI32()] }`),
				Entry("three functions middle mismatched", `{ x := [getI32(), getStr(), getI32()] }`),
			)

			DescribeTable("invalid function call with variable combinations",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, funcResolver)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
				// Inferred type variables - these should also be rejected for consistency
				// with explicit type annotations above
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
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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

		Context("invalid series literals - three or more elements with mismatch", func() {
			DescribeTable("should reject",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
		})

		Context("invalid series literals - expression mismatches", func() {
			DescribeTable("should reject",
				func(code string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("incompatible type"))
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
				func(code string, errorSubstring string) {
					block := MustSucceed(parser.ParseBlock(code))
					ctx := context.CreateRoot(bCtx, block, nil)
					setupFunctionContext(ctx)
					statement.AnalyzeBlock(ctx)
					Expect(ctx.Diagnostics.Ok()).To(BeFalse())
					Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(errorSubstring))
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
		It("should allow indexed assignment to series", func() {
			block := MustSucceed(parser.ParseBlock(`{
				data series i64 := [1, 2, 3]
				data[0] = 10
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("should detect indexed assignment on non-series type", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i64 := 42
				x[0] = 10
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("indexed assignment only supported on series types"))
		})

		It("should detect slice assignment (not supported)", func() {
			block := MustSucceed(parser.ParseBlock(`{
				data series i64 := [1, 2, 3]
				data[0:2] = 10
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("slice assignment not supported"))
		})

		It("should detect type mismatch in indexed assignment", func() {
			block := MustSucceed(parser.ParseBlock(`{
				data series i64 := [1, 2, 3]
				data[0] = "hello"
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			setupFunctionContext(ctx)
			statement.AnalyzeBlock(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})
	})
})
