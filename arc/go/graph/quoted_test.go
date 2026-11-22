// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var _ = Describe("Quoted channel names in a graph function", func() {
	// importedModules compiles a graph of one function with the param x and body,
	// beside a channel named x, and returns the module of each function its WASM
	// imports.
	importedModules := func(ctx SpecContext, body string) []string {
		GinkgoHelper()
		g := arc.Graph{Functions: []ir.Function{{
			Key:     "f",
			Inputs:  types.Params{{Name: "x", Type: types.F64()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
			Body:    ir.Body{Raw: body},
		}}}
		root := symbol.NewRoot(nil, stl.NewSymbols())
		root.Parent.AddChild(&symbol.Symbol{
			Name: "x",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   7,
		})
		prog := MustSucceed(arc.CompileGraph(ctx, g, root))
		r := wazero.NewRuntime(ctx)
		DeferCleanup(r.Close)
		compiled := MustSucceed(r.CompileModule(ctx, prog.WASM))
		DeferCleanup(compiled.Close)
		return lo.Map(
			compiled.ImportedFunctions(),
			func(f api.FunctionDefinition, _ int) string {
				module, _, _ := f.Import()
				return module
			},
		)
	}

	It("Should read the channel, not a param that shares its name", func(
		ctx SpecContext,
	) {
		Expect(importedModules(ctx, "{\n\treturn 'x' * 2.0\n}")).
			To(ContainElement("channels"))
	})

	It("Should read the param by its bare name", func(ctx SpecContext) {
		Expect(importedModules(ctx, "{\n\treturn x * 2.0\n}")).
			ToNot(ContainElement("channels"))
	})
})
