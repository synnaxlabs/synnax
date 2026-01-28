// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flow_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AnalyzeSingleExpression", func() {
	testResolver := symbol.MapResolver{
		"temp_sensor": symbol.Symbol{
			Name: "temp_sensor",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
			ID:   10,
		},
		"pressure": symbol.Symbol{
			Name: "pressure",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   11,
		},
		"ox_pt_1": symbol.Symbol{
			Name: "ox_pt_1",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   12,
		},
		"ox_pt_2": symbol.Symbol{
			Name: "ox_pt_2",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   13,
		},
	}

	Describe("Pure Literals", func() {
		It("should create KindConstant for integer literal", func() {
			expr := MustSucceed(parser.ParseExpression(`42`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Config.Get("value"))
			Expect(valueCfg.Type.IsNumeric() || valueCfg.Type.Kind == types.KindVariable).To(BeTrue())
			output := MustBeOk(constSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type.IsNumeric() || output.Type.Kind == types.KindVariable).To(BeTrue())
		})

		It("should create KindConstant for float literal", func() {
			expr := MustSucceed(parser.ParseExpression(`3.14`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Config.Get("value"))
			Expect(valueCfg.Type.IsFloat() || valueCfg.Type.Kind == types.KindVariable).To(BeTrue())
		})

		It("should create KindConstant for string literal", func() {
			expr := MustSucceed(parser.ParseExpression(`"hello"`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Config.Get("value"))
			Expect(valueCfg.Type).To(Equal(types.String()))
		})

		It("should auto-increment constant names for multiple literals", func() {
			expr0 := MustSucceed(parser.ParseExpression(`42`))
			ctx := context.CreateRoot(bCtx, expr0, testResolver)
			flow.AnalyzeSingleExpression(ctx)

			expr1 := MustSucceed(parser.ParseExpression(`100`))
			ctx1 := context.Context[parser.IExpressionContext]{
				Context:     bCtx,
				Scope:       ctx.Scope,
				Diagnostics: ctx.Diagnostics,
				Constraints: ctx.Constraints,
				TypeMap:     ctx.TypeMap,
				AST:         expr1,
			}
			flow.AnalyzeSingleExpression(ctx1)

			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			const0 := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			const1 := MustSucceed(ctx.Scope.Resolve(ctx, "constant_1"))
			Expect(const0.Kind).To(Equal(symbol.KindConstant))
			Expect(const1.Kind).To(Equal(symbol.KindConstant))
		})
	})

	Describe("Complex Expressions", func() {
		It("should create KindFunction for binary expression with channel", func() {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			Expect(fnSym.Type.Kind).To(Equal(types.KindFunction))
			Expect(fnSym.Type.Config).To(BeEmpty())
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.U8()))
		})

		It("should accumulate read channels from expression", func() {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Channels.Read).To(HaveLen(1))
			Expect(fnSym.Channels.Read).To(HaveKey(uint32(12)))
		})

		It("should accumulate multiple channels from arithmetic expression", func() {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 + ox_pt_2`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Channels.Read).To(HaveLen(2))
			Expect(fnSym.Channels.Read).To(HaveKey(uint32(12)))
			Expect(fnSym.Channels.Read).To(HaveKey(uint32(13)))
		})

		It("should create KindFunction for logical AND expression", func() {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100 and pressure > 50`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.U8()))
		})

		It("should auto-increment expression names", func() {
			expr0 := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
			ctx := context.CreateRoot(bCtx, expr0, testResolver)
			flow.AnalyzeSingleExpression(ctx)

			expr1 := MustSucceed(parser.ParseExpression(`pressure < 50`))
			ctx1 := context.Context[parser.IExpressionContext]{
				Context:     bCtx,
				Scope:       ctx.Scope,
				Diagnostics: ctx.Diagnostics,
				Constraints: ctx.Constraints,
				TypeMap:     ctx.TypeMap,
				AST:         expr1,
			}
			flow.AnalyzeSingleExpression(ctx1)

			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fn0 := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			fn1 := MustSucceed(ctx.Scope.Resolve(ctx, "expression_1"))
			Expect(fn0.Kind).To(Equal(symbol.KindFunction))
			Expect(fn1.Kind).To(Equal(symbol.KindFunction))
		})

		It("should handle parenthesized expressions", func() {
			expr := MustSucceed(parser.ParseExpression(`(ox_pt_1 + ox_pt_2) * 2`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
		})

		It("should handle type cast expressions", func() {
			expr := MustSucceed(parser.ParseExpression(`f64(temp_sensor)`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.F64()))
		})
	})

	Describe("Error Cases", func() {
		It("should report undefined symbol in expression", func() {
			expr := MustSucceed(parser.ParseExpression(`unknown_channel > 100`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("undefined symbol: unknown_channel"))
		})

		It("should report multiple undefined symbols", func() {
			expr := MustSucceed(parser.ParseExpression(`foo + bar`))
			ctx := context.CreateRoot(bCtx, expr, testResolver)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(2))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("undefined symbol: foo"))
			Expect((*ctx.Diagnostics)[1].Message).To(Equal("undefined symbol: bar"))
		})
	})
})
