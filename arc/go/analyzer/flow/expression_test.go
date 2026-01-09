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
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expression func Conversion", func() {
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
		"alarm": symbol.Symbol{
			Name: "alarm",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{
						Name: ir.DefaultInputParam,
						Type: types.U8(),
					},
				},
			}),
		},
		"logger": symbol.Symbol{
			Name: "logger",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{
						Name: ir.DefaultInputParam,
						Type: types.U8(),
					},
				},
			}),
		},
		"display": symbol.Symbol{
			Name: "display",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{
						Name: ir.DefaultInputParam,
						Type: types.F64(),
					},
				},
			}),
		},
		"warning": symbol.Symbol{
			Name: "warning",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{}),
		},
		"alarm_ch": symbol.Symbol{
			Name: "alarm_ch",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.U8()),
			ID:   14,
		},
	}

	Context("Binary expression with channels", func() {
		It("should convert comparison expression to synthetic function", func() {
			ast := MustSucceed(parser.Parse(`ox_pt_1 > 100 -> alarm{}`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Name).To(Equal("expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			Expect(fnSym.Type.Kind).To(Equal(types.KindFunction))
			Expect(fnSym.Type.Config).To(BeEmpty())
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.U8()))
			Expect(fnSym.Channels.Read).To(HaveLen(1))
			Expect(fnSym.Channels.Read).To(HaveKey(uint32(12)))
		})

		It("should extract multiple channels from arithmetic expressions", func() {
			ast := MustSucceed(parser.Parse(`
				(ox_pt_1 + ox_pt_2) / 2 -> display{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
			synthFunc := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(synthFunc).ToNot(BeNil())
			output := MustBeOk(synthFunc.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.F64()))
		})
	})

	Context("Complex logical expressions", func() {
		It("should handle logical AND with multiple channels", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 and pressure > 50 -> alarm_ch
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			synthTask := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(synthTask).ToNot(BeNil())
			Expect(synthTask.Type.Kind).To(Equal(types.KindFunction))
		})
	})

	Context("Error cases", func() {
		It("should reject unknown channels in expressions", func() {
			ast := MustSucceed(parser.Parse(`
				unknown_channel > 100 -> alarm{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("undefined symbol: unknown_channel"))
		})
	})

	Context("Multiple expressions in flow", func() {
		It("should create separate tasks for each expression", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 -> alarm{}
				pressure < 50 -> warning{}
				f64(temp_sensor) * 1.8 + 32 -> display{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
			fn0 := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			fn1 := MustSucceed(ctx.Scope.Resolve(ctx, "expression_1"))
			fn2 := MustSucceed(ctx.Scope.Resolve(ctx, "expression_2"))
			Expect(fn0).ToNot(BeNil())
			Expect(fn1).ToNot(BeNil())
			Expect(fn2).ToNot(BeNil())
		})
	})

	Context("Parenthesized expressions", func() {
		It("should handle nested parentheses correctly", func() {
			ast := MustSucceed(parser.Parse(`
				((ox_pt_1 + ox_pt_2) * 2) > 100 -> alarm{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			synthFunc := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(synthFunc).ToNot(BeNil())
		})
	})

	Context("Channel type preservation", func() {
		It("should preserve channel types in config", func() {
			ast := MustSucceed(parser.Parse(`
				f64(temp_sensor) -> display{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
			synthFunc := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(synthFunc).ToNot(BeNil())
		})
	})
})
