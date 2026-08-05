// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AnalyzeArguments hook (call form)", func() {
	It(
		"Should invoke the hook on a func-form call inside a function body",
		func(bCtx SpecContext) {
			var (
				called  int
				gotArgs []symbol.Argument
			)
			hooked := symbol.Symbol{
				Name: "hooked",
				Kind: symbol.KindFunction,
				Exec: symbol.ExecBoth,
				Type: types.Function(types.FunctionProperties{
					Inputs: types.Params{{Name: "a", Type: types.I32()}},
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.I32()},
					},
				}),
				AnalyzeArguments: func(_ *diagnostics.Diagnostics, args []symbol.Argument) {
					called++
					gotArgs = args
				},
			}
			src := `func main() i32 { return hooked(1) }`
			ast := MustSucceed(parser.Parse(src))
			ctx := acontext.NewRoot(bCtx, ast, NewRoot(nil, hooked))
			analyzer.AnalyzeProgram(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(called).To(Equal(1))
			Expect(gotArgs).To(HaveLen(1))
		},
	)

	It(
		"Should not invoke the hook when the symbol does not define one",
		func(bCtx SpecContext) {
			plain := symbol.Symbol{
				Name: "plain",
				Kind: symbol.KindFunction,
				Exec: symbol.ExecBoth,
				Type: types.Function(types.FunctionProperties{
					Inputs: types.Params{{Name: "a", Type: types.I32()}},
					Outputs: types.Params{
						{Name: ir.DefaultOutputParam, Type: types.I32()},
					},
				}),
			}
			src := `func main() i32 { return plain(1) }`
			ast := MustSucceed(parser.Parse(src))
			ctx := acontext.NewRoot(bCtx, ast, NewRoot(nil, plain))
			Expect(func() { analyzer.AnalyzeProgram(ctx) }).ToNot(Panic())
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		},
	)
})
