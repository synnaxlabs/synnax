// Copyright 2025 Synnax Labs, Inc.
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
			Type: types.Function(
				types.Params{
					Keys:   []string{"input"},
					Values: []types.Type{types.U8()},
				},
				types.Params{},
				types.Params{},
			),
		},
		"logger": symbol.Symbol{
			Name: "logger",
			Kind: symbol.KindFunction,
			Type: types.Function(
				types.Params{
					Keys:   []string{"input"},
					Values: []types.Type{types.U8()},
				},
				types.Params{},
				types.Params{},
			),
		},
		"display": symbol.Symbol{
			Name: "display",
			Kind: symbol.KindFunction,
			Type: types.Function(
				types.Params{
					Keys:   []string{"input"},
					Values: []types.Type{types.F64()},
				},
				types.Params{},
				types.Params{},
			),
		},
		"warning": symbol.Symbol{
			Name: "warning",
			Kind: symbol.KindFunction,
			Type: types.Function(types.Params{}, types.Params{}, types.Params{}),
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
			stageSym := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_0"))
			Expect(stageSym.Name).To(Equal("__expr_0"))
			Expect(stageSym.Kind).To(Equal(symbol.KindFunction))
			Expect(stageSym.Type.Kind).To(Equal(types.KindFunction))
			Expect(stageSym.Type.Config.Count()).To(Equal(0))
			output := MustBeOk(stageSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output).To(Equal(types.U8()))
		})

		It("should extract multiple channels from arithmetic expressions", func() {
			ast := MustSucceed(parser.Parse(`
				(ox_pt_1 + ox_pt_2) / 2 -> display{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue(), ctx.Diagnostics.String())
			synthFunc := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_0"))
			Expect(synthFunc).ToNot(BeNil())
			output := MustBeOk(synthFunc.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output).To(Equal(types.F64()))
		})
	})

	Context("Complex logical expressions", func() {
		It("should handle logical AND with multiple channels", func() {
			ast := MustSucceed(parser.Parse(`
				ox_pt_1 > 100 && pressure > 50 -> alarm_ch
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			synthTask := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_0"))
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
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("unknown"))
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
			stage0 := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_0"))
			stage1 := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_1"))
			stage2 := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_2"))
			Expect(stage0).ToNot(BeNil())
			Expect(stage1).ToNot(BeNil())
			Expect(stage2).ToNot(BeNil())
		})
	})

	Context("Parenthesized expressions", func() {
		It("should handle nested parentheses correctly", func() {
			ast := MustSucceed(parser.Parse(`
				((ox_pt_1 + ox_pt_2) * 2) > 100 -> alarm{}
			`))
			ctx := context.CreateRoot(bCtx, ast, testResolver)
			Expect(analyzer.AnalyzeProgram(ctx)).To(BeTrue())
			synthFunc := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_0"))
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
			synthFunc := MustSucceed(ctx.Scope.Resolve(ctx, "__expr_0"))
			Expect(synthFunc).ToNot(BeNil())
		})
	})
})
