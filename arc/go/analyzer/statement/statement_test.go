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
					result := statement.Analyze(ctx)
					Expect(result).To(Equal(expectOk))
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
				Expect(statement.Analyze(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot assign str to i32"))
			})

			It("should detect duplicate variable declaration", func() {
				stmt := MustSucceed(parser.ParseBlock(`{
					x := 1
					x := 1
				}`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.AnalyzeBlock(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("name x conflicts with existing symbol"))
			})

			It("should detect undefined variable in initializer", func() {
				stmt := MustSucceed(parser.ParseStatement(`x := y + 1`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: y"))
			})
		})

		Context("stateful variables", func() {
			It("should analyze a stateful variable with inferred type", func() {
				stmt := MustSucceed(parser.ParseStatement(`counter $= 0`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeTrue())
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
				Expect(statement.Analyze(ctx)).To(BeTrue())
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
				result := statement.AnalyzeBlock(ctx)
				Expect(result).To(Equal(expectOk))
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
		)
	})

	Describe("If Statement", func() {
		DescribeTable("valid if statements",
			func(code string) {
				stmt := MustSucceed(parser.ParseStatement(code))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeTrue())
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
			Expect(statement.Analyze(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: x"))
		})
	})

	Describe("Block", func() {
		DescribeTable("block analysis",
			func(code string, expectOk bool, errorSubstring string) {
				block := MustSucceed(parser.ParseBlock(code))
				ctx := context.CreateRoot(bCtx, block, nil)
				result := statement.AnalyzeBlock(ctx)
				Expect(result).To(Equal(expectOk))
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
			Expect(statement.Analyze(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("should detect errors in standalone expression", func() {
			stmt := MustSucceed(parser.ParseStatement(`undefined_var + 1`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			Expect(statement.Analyze(ctx)).To(BeFalse())
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
					result := statement.AnalyzeBlock(ctx)
					Expect(result).To(Equal(expectOk), ctx.Diagnostics.String())
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
				Expect(statement.AnalyzeBlock(ctx)).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined"))
			})
		})

		Context("channel reads in imperative context", func() {
			DescribeTable("channel read type inference",
				func(chanName string, expectedType types.Type) {
					code := "current := " + chanName
					stmt := MustSucceed(parser.ParseStatement(code))
					ctx := context.CreateRoot(bCtx, stmt, channelResolver)
					Expect(statement.Analyze(ctx)).To(BeTrue(), ctx.Diagnostics.String())
					Expect(*ctx.Diagnostics).To(BeEmpty())
					varScope := MustSucceed(ctx.Scope.Resolve(ctx, "current"))
					Expect(varScope.Type).To(Equal(expectedType))
				},
				Entry("f64 channel", "sensor", types.F64()),
				Entry("i32 channel", "int_chan", types.I32()),
			)
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
			Expect(statement.AnalyzeBlock(ctx)).To(BeTrue(), ctx.Diagnostics.String())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("should properly track types through assignments", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 10
				y := x
				z := y + 5
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			Expect(statement.AnalyzeBlock(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("should return an error when assigning incompatible types", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 10
				y f32 := x
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			Expect(statement.AnalyzeBlock(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot assign i32 to f32"))
		})
	})
})
