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
	Describe("Variable Declaration", func() {
		Describe("Local Variables", func() {
			It("Should analyze a local variable with explicit type", func() {
				stmt := MustSucceed(parser.ParseStatement(`x i32 := 42`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeTrue())
				sym := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
				Expect(sym.Type).To(Equal(types.I32()))
			})

			It("Should infer type from initializer", func() {
				stmt := MustSucceed(parser.ParseStatement(`x := 3.14`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				sym := MustSucceed(ctx.Scope.Resolve(ctx, "x"))
				// Literals now infer as type variables with float constraint
				Expect(sym.Type.Kind).To(Equal(types.KindTypeVariable))
				Expect(sym.Type.Constraint).ToNot(BeNil())
				Expect(sym.Type.Constraint.Kind).To(Equal(types.KindFloatConstant))
			})

			It("Should detect type mismatch", func() {
				stmt := MustSucceed(parser.ParseStatement(`x i32 := "hello"`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot assign string to i32"))
			})

			It("Should detect duplicate variable declaration", func() {
				stmt := MustSucceed(parser.ParseBlock(`{
					x := 1
					x := 1
				}`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.AnalyzeBlock(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("name x conflicts with existing symbol at line 2, col 5"))
			})

			It("Should detect undefined variable in initializer", func() {
				stmt := MustSucceed(parser.ParseStatement(`x := y + 1`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: y"))
			})
		})

		Describe("Stateful Variables", func() {
			It("Should analyze a stateful variable", func() {
				stmt := MustSucceed(parser.ParseStatement(`counter $= 0`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				sym := MustSucceed(ctx.Scope.Resolve(ctx, "counter"))
				Expect(sym.Kind).To(Equal(symbol.KindStatefulVariable))
				// Literals now infer as type variables with integer constraint
				Expect(sym.Type.Kind).To(Equal(types.KindTypeVariable))
				Expect(sym.Type.Constraint).ToNot(BeNil())
				Expect(sym.Type.Constraint.Kind).To(Equal(types.KindIntegerConstant))
			})

			It("Should analyze stateful variable with explicit type", func() {
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
		It("Should analyze assignment to existing variable", func() {
			stmt := MustSucceed(parser.ParseStatement(`x = 42`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "x",
				Kind: symbol.KindVariable,
				Type: types.I64(),
			}))
			Expect(statement.Analyze(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should detect assignment to undefined variable", func() {
			stmt := MustSucceed(parser.ParseStatement(`x = 42`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			Expect(statement.Analyze(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: x"))
		})

		It("Should detect type mismatch in assignment", func() {
			stmt := MustSucceed(parser.ParseStatement(`x = "hello"`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "x",
				Kind: symbol.KindVariable,
				Type: types.I32(),
			}))
			Expect(statement.Analyze(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch"))
		})
	})

	Describe("If Statement", func() {
		It("Should analyze simple if statement", func() {
			stmt := MustSucceed(parser.ParseStatement(`if 1 {
				x := 42
			}`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			Expect(statement.Analyze(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should analyze if-else chain", func() {
			stmt := MustSucceed(parser.ParseStatement(`if 0 {
				x := 1
			} else if 1 {
				y := 2
			} else {
				z := 3
			}`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			Expect(statement.Analyze(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should detect undefined variable in condition", func() {
			stmt := MustSucceed(parser.ParseStatement(`if x > 10 {
				y := 1
			}`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			Expect(statement.Analyze(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: x"))
		})

		It("Should handle nested blocks with separate scopes", func() {
			stmt := MustSucceed(parser.ParseStatement(`if 1 {
				x := 42
				if 1 {
					y := x + 1
				}
			}`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			Expect(statement.Analyze(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})
	})

	Describe("Block", func() {
		It("Should analyze multiple statements in a block", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x := 1
				y := 2
				z := x + y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			Expect(statement.AnalyzeBlock(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should maintain variable visibility within block", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x := 1
				y := x + 2
				z := x + y
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			Expect(statement.AnalyzeBlock(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should detect errors in block statements", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x := 1
				y := undefined
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			Expect(statement.AnalyzeBlock(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: undefined"))
		})
	})

	Describe("Expression Statement", func() {
		It("Should analyze standalone expression", func() {
			stmt := MustSucceed(parser.ParseStatement(`x + 1`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "x",
				Kind: symbol.KindVariable,
				Type: types.I64(),
			}))
			Expect(statement.Analyze(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should detect errors in standalone expression", func() {
			stmt := MustSucceed(parser.ParseStatement(`undefined_var + 1`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			Expect(statement.Analyze(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
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

		Describe("Channel Writes", func() {
			It("Should analyze basic channel write with arrow", func() {
				stmt := MustSucceed(parser.ParseStatement(`42.0 -> output`))
				ctx := context.CreateRoot(bCtx, stmt, channelResolver)
				ok := statement.Analyze(ctx)
				if !ok {
					GinkgoWriter.Printf("Diagnostics: %v\n", ctx.Diagnostics)
				}
				Expect(ok).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			})

			It("Should analyze channel write with recv operator", func() {
				stmt := MustSucceed(parser.ParseStatement(`output <- 42.0`))
				ctx := context.CreateRoot(bCtx, stmt, channelResolver)
				Expect(statement.Analyze(ctx)).To(BeTrue())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			})

			It("Should detect type mismatch in channel write", func() {
				stmt := MustSucceed(parser.ParseStatement(`"hello" -> output`))
				ctx := context.CreateRoot(bCtx, stmt, channelResolver)
				Expect(statement.Analyze(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("type mismatch: cannot write string to channel of type f64"))
			})

			It("Should analyze channel write with variable", func() {
				stmt := MustSucceed(parser.ParseStatement(`value -> output`))
				ctx := context.CreateRoot(bCtx, stmt, channelResolver)
				MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
					Name: "value",
					Kind: symbol.KindVariable,
					Type: types.F64(),
				}))
				Expect(statement.Analyze(ctx)).To(BeTrue(), ctx.Diagnostics.String())
				Expect(*ctx.Diagnostics).To(BeEmpty())
			})

			It("Should detect undefined channel in write", func() {
				stmt := MustSucceed(parser.ParseStatement(`42.0 -> undefined_channel`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol: undefined_channel"))
			})
		})

		Describe("Channel Reads", func() {
			It("Should analyze blocking channel read", func() {
				stmt := MustSucceed(parser.ParseStatement(`value := <-sensor`))
				ctx := context.CreateRoot(bCtx, stmt, channelResolver)
				Expect(statement.Analyze(ctx)).To(BeTrue(), ctx.Diagnostics.String())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				varScope := MustSucceed(ctx.Scope.Resolve(ctx, "value"))
				Expect(varScope.Type).To(Equal(types.F64()))
			})

			It("Should analyze non-blocking channel read", func() {
				stmt := MustSucceed(parser.ParseStatement(`current := sensor`))
				ctx := context.CreateRoot(bCtx, stmt, channelResolver)
				Expect(statement.Analyze(ctx)).To(BeTrue(), ctx.Diagnostics.String())
				Expect(*ctx.Diagnostics).To(BeEmpty())
				varScope := MustSucceed(ctx.Scope.Resolve(ctx, "current"))
				Expect(varScope.Type).To(Equal(types.F64()))
			})

			It("Should detect undefined channel in read", func() {
				stmt := MustSucceed(parser.ParseStatement(`value := <-undefined_channel`))
				ctx := context.CreateRoot(bCtx, stmt, nil)
				Expect(statement.Analyze(ctx)).To(BeFalse())
				Expect(*ctx.Diagnostics).To(HaveLen(1))
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined"))
			})
		})

	})

	Describe("Mixed Type Scenarios", func() {
		It("Should handle complex nested structures", func() {
			stmt := MustSucceed(parser.ParseStatement(`if 1 {
				x := 10
				y $= 20
				if x < y {
					z := x + y
					z = z * 2
				}
			}`))
			ctx := context.CreateRoot(bCtx, stmt, nil)
			Expect(statement.Analyze(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should properly track types through assignments", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 10
				y := x
				z := y + 5
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			Expect(statement.AnalyzeBlock(ctx)).To(BeTrue())
			Expect(*ctx.Diagnostics).To(BeEmpty())
		})

		It("Should return an error when a variable of an incorrect type is assigned to another variable", func() {
			block := MustSucceed(parser.ParseBlock(`{
				x i32 := 10
				y f32 := x
			}`))
			ctx := context.CreateRoot(bCtx, block, nil)
			Expect(statement.AnalyzeBlock(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			first := (*ctx.Diagnostics)[0]
			Expect(first.Message).To(ContainSubstring("type mismatch: cannot assign i32 to f32"))
		})
	})
})
