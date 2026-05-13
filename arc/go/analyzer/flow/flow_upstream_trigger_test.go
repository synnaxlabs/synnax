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

// execBothFn builds an ExecBoth function symbol with Inputs mirroring Config
// one-for-one, matching the shape required by the symbol.ExecBoth contract.
func execBothFn(name string, params types.Params, output types.Type) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecBoth,
		Type: types.Function(types.FunctionProperties{
			Inputs:  params,
			Config:  params,
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: output}},
		}),
	}
}

func execFlowFn(name string, inputs types.Params, output types.Type) symbol.Symbol {
	props := types.FunctionProperties{Inputs: inputs}
	if output.Kind != types.KindInvalid {
		props.Outputs = types.Params{{Name: ir.DefaultOutputParam, Type: output}}
	}
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Inputs:  props.Inputs,
			Outputs: props.Outputs,
		}),
	}
}

var _ = Describe("upstreamIsTrigger Suppression", func() {
	Describe("ExecBoth with non-empty Config (suppression active)", func() {
		It("Should accept channel upstream whose value type does not match the input", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"trig": {Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1},
				"x":    {Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 2},
				"fmtfn": execBothFn(
					"fmtfn",
					types.Params{{Name: "x", Type: types.Chan(types.I32())}},
					types.String(),
				),
			}
			ast := MustSucceed(parser.Parse(`trig -> fmtfn{x=x}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept expression upstream whose type does not match the input", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"trig": {Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1},
				"x":    {Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 2},
				"fmtfn": execBothFn(
					"fmtfn",
					types.Params{{Name: "x", Type: types.Chan(types.I32())}},
					types.String(),
				),
			}
			ast := MustSucceed(parser.Parse(`trig > 100.0 -> fmtfn{x=x}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept upstream function whose output type does not match the input", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"x":        {Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 1},
				"producer": execFlowFn("producer", nil, types.U8()),
				"fmtfn": execBothFn(
					"fmtfn",
					types.Params{{Name: "x", Type: types.Chan(types.I32())}},
					types.String(),
				),
			}
			ast := MustSucceed(parser.Parse(`producer{} -> fmtfn{x=x}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept upstream function feeding an ExecBoth fn with multiple inputs", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"x":        {Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 1},
				"y":        {Name: "y", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
				"producer": execFlowFn("producer", nil, types.F64()),
				"fmtfn": execBothFn(
					"fmtfn",
					types.Params{
						{Name: "x", Type: types.Chan(types.I32())},
						{Name: "y", Type: types.Chan(types.F64())},
					},
					types.String(),
				),
			}
			ast := MustSucceed(parser.Parse(`producer{} -> fmtfn{x=x, y=y}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})

		It("Should accept void upstream function feeding an ExecBoth fn with inputs", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"x":    {Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 1},
				"void": execFlowFn("void", nil, types.Type{}),
				"fmtfn": execBothFn(
					"fmtfn",
					types.Params{{Name: "x", Type: types.Chan(types.I32())}},
					types.String(),
				),
			}
			ast := MustSucceed(parser.Parse(`void{} -> fmtfn{x=x}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Non-ExecBoth (regression guards: suppression must NOT activate)", func() {
		It("Should reject channel upstream with mismatched value type for an ExecFlow fn", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1},
				"sink":   execFlowFn("sink", types.Params{{Name: "v", Type: types.I32()}}, types.Type{}),
			}
			ast := MustSucceed(parser.Parse(`sensor -> sink{}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
		})

		It("Should reject expression upstream with mismatched type for an ExecFlow fn", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"sensor": {Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1},
				"sink":   execFlowFn("sink", types.Params{{Name: "v", Type: types.String()}}, types.Type{}),
			}
			ast := MustSucceed(parser.Parse(`sensor > 100.0 -> sink{}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("does not match"))
		})

		It("Should reject upstream function feeding an ExecFlow fn with multiple inputs", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"producer": execFlowFn("producer", nil, types.F64()),
				"multi": execFlowFn(
					"multi",
					types.Params{
						{Name: "a", Type: types.F64()},
						{Name: "b", Type: types.F64()},
					},
					types.Type{},
				),
			}
			ast := MustSucceed(parser.Parse(`producer{} -> multi{}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("multi has more than one parameter"))
		})

		It("Should reject upstream function output type mismatch for an ExecFlow fn", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"producer": execFlowFn("producer", nil, types.U8()),
				"sink":     execFlowFn("sink", types.Params{{Name: "v", Type: types.F64()}}, types.Type{}),
			}
			ast := MustSucceed(parser.Parse(`producer{} -> sink{}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("is not equal to argument type"))
		})
	})

	Describe("ExecBoth with empty Config (suppression NOT active)", func() {
		It("Should treat an ExecBoth fn with empty Config like a normal flow fn", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"trig": {Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
				"now": {
					Name: "now",
					Kind: symbol.KindFunction,
					Exec: symbol.ExecBoth,
					Type: types.Function(types.FunctionProperties{
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.TimeStamp()}},
					}),
				},
			}
			ast := MustSucceed(parser.Parse(`trig -> now{}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})
	})

	Describe("Mid-chain channel sinks", func() {
		It("Should accept producer -> channel -> ExecBoth+config consumer", func(bCtx SpecContext) {
			r := symbol.MapResolver{
				"tick":  {Name: "tick", Kind: symbol.KindChannel, Type: types.Chan(types.TimeStamp()), ID: 1},
				"log_b": {Name: "log_b", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 2},
				"emitKey": execBothFn(
					"emitKey",
					types.Params{{Name: "key", Type: types.String()}},
					types.String(),
				),
				"finish": execBothFn(
					"finish",
					types.Params{{Name: "key_or_name", Type: types.String()}},
					types.U8(),
				),
			}
			ast := MustSucceed(parser.Parse(`tick -> emitKey{key="flow"} -> log_b -> finish{key_or_name="flow"}`))
			ctx := context.CreateRoot(bCtx, ast, r)
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		})
	})
})
