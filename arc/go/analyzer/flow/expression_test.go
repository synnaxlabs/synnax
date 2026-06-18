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
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AnalyzeSingleExpression", func() {
	var root *symbol.Symbol
	BeforeEach(func() {
		testChannels := []symbol.Symbol{
			{Name: "temp_sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10},
			{Name: "pressure", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 11},
			{Name: "ox_pt_1", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 12},
			{Name: "ox_pt_2", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 13},
		}
		root = NewRoot(nil, testChannels...)
	})

	Describe("Pure Literals", func() {
		It("should create KindConstant for integer literal", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`42`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Inputs.Get("value"))
			Expect(valueCfg.Type.IsNumeric() || valueCfg.Type.Kind == types.KindVariable).To(BeTrue())
			output := MustBeOk(constSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type.IsNumeric() || output.Type.Kind == types.KindVariable).To(BeTrue())
		})

		It("should create KindConstant for float literal", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`3.14`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Inputs.Get("value"))
			Expect(valueCfg.Type.IsFloat() || valueCfg.Type.Kind == types.KindVariable).To(BeTrue())
		})

		It("should create KindConstant for string literal", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`"hello"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Inputs.Get("value"))
			Expect(valueCfg.Type).To(Equal(types.String()))
		})

		It("should auto-increment constant names for multiple literals", func(bCtx SpecContext) {
			expr0 := MustSucceed(parser.ParseExpression(`42`))
			ctx := context.NewRoot(bCtx, expr0, root)
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
		It("should create KindFunction for binary expression with channel", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			Expect(fnSym.Type.Kind).To(Equal(types.KindFunction))
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.U8()))
		})

		It("should accumulate read channels from expression", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Channels.Read).To(HaveLen(1))
			Expect(fnSym.Channels.Read[12]).To(Equal("ox_pt_1"))
		})

		It("should accumulate multiple channels from arithmetic expression", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 + ox_pt_2`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Channels.Read).To(HaveLen(2))
			Expect(fnSym.Channels.Read[12]).To(Equal("ox_pt_1"))
			Expect(fnSym.Channels.Read[13]).To(Equal("ox_pt_2"))
		})

		It("should create KindFunction for logical AND expression", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100 and pressure > 50`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.U8()))
		})

		It("should auto-increment expression names", func(bCtx SpecContext) {
			expr0 := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
			ctx := context.NewRoot(bCtx, expr0, root)
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

		It("should handle parenthesized expressions", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`(ox_pt_1 + ox_pt_2) * 2`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
		})

		It("should handle type cast expressions", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f64(temp_sensor)`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.F64()))
		})
	})

	Describe("Raw String Format Literals", func() {
		It("should create KindConstant for raw string without placeholders", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"static"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
		})

		It("should create fmt_str synthetic function for raw string with placeholder", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"v={ox_pt_1}"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			Expect(fnSym.Type.Kind).To(Equal(types.KindFunction))
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.String()))
		})

		It("should track placeholder channel reads on the synthetic function", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"v={ox_pt_1}"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
			Expect(fnSym.Channels.Read).To(HaveLen(1))
			Expect(fnSym.Channels.Read[12]).To(Equal("ox_pt_1"))
		})

		It("should track multiple placeholder channel reads", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"a={ox_pt_1} b={ox_pt_2}"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
			Expect(fnSym.Channels.Read).To(HaveLen(2))
			Expect(fnSym.Channels.Read[12]).To(Equal("ox_pt_1"))
			Expect(fnSym.Channels.Read[13]).To(Equal("ox_pt_2"))
		})

		It("should accept a numeric literal placeholder with format spec", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"x={42%05d}"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
		})

		It("should auto-increment fmt_str names across multiple raw strings", func(bCtx SpecContext) {
			expr0 := MustSucceed(parser.ParseExpression(`f"{ox_pt_1}"`))
			ctx := context.NewRoot(bCtx, expr0, root)
			flow.AnalyzeSingleExpression(ctx)

			expr1 := MustSucceed(parser.ParseExpression(`f"{ox_pt_2}"`))
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
			MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
			MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_1"))
		})

		It("should report unmatched opening brace in raw string body", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"{x"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("unmatched"))
		})

		It("should report empty placeholder", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"pre {} post"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("must contain an expression"))
		})

		It("should report undefined identifier in placeholder", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"x={unknown_ch}"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("undefined symbol"))
		})

		It("should report invalid format spec for placeholder type", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"x={ox_pt_1:s}"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("invalid format spec"))
		})

	})

	Describe("Error Cases", func() {
		It("should report undefined symbol in expression", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`unknown_channel > 100`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("undefined symbol: unknown_channel"))
		})

		It("should report multiple undefined symbols", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`foo + bar`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(2))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("undefined symbol: foo"))
			Expect((*ctx.Diagnostics)[1].Message).To(Equal("undefined symbol: bar"))
		})

	})
})
