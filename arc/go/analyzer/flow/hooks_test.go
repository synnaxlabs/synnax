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
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/symbol/hooks"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AnalyzeFlowConfig hook", func() {
	It("Should invoke the hook on a flow-form invocation that carries config", func(bCtx SpecContext) {
		var (
			called    int
			configAST parser.IConfigValuesContext
		)
		params := types.Params{{Name: "x", Type: types.I32()}}
		r := symbol.MapResolver{
			"trig": {Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
			"hooked": {
				Name: "hooked",
				Kind: symbol.KindFunction,
				Exec: symbol.ExecBoth,
				Type: types.Function(types.FunctionProperties{Config: params, Inputs: params}),
				AnalyzeFlowConfig: hooks.FlowConfigHook(func(_ any, c parser.IConfigValuesContext) {
					called++
					configAST = c
				}),
			},
		}
		ast := MustSucceed(parser.Parse(`trig -> hooked{x=1}`))
		ctx := acontext.CreateRoot(bCtx, ast, r)
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		Expect(called).To(Equal(1))
		Expect(configAST).ToNot(BeNil())
	})

	It("Should not invoke the hook when the symbol does not define one", func(bCtx SpecContext) {
		params := types.Params{{Name: "x", Type: types.I32()}}
		r := symbol.MapResolver{
			"trig": {Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
			"plain": {
				Name: "plain",
				Kind: symbol.KindFunction,
				Exec: symbol.ExecBoth,
				Type: types.Function(types.FunctionProperties{Config: params, Inputs: params}),
			},
		}
		ast := MustSucceed(parser.Parse(`trig -> plain{x=1}`))
		ctx := acontext.CreateRoot(bCtx, ast, r)
		Expect(func() { analyzer.AnalyzeProgram(ctx) }).ToNot(Panic())
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
	})
})
