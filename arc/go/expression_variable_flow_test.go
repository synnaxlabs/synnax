// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Expression variables in flows", func() {
	It("Should send a channel expression variable", func(ctx SpecContext) {
		h := newRuntimeHarness(ctx, "y := x * 2.0\ny -> out", channelSymbols(
			map[string]channelDef{"x": {types.F64(), 100}, "out": {types.F64(), 200}},
		),
			channels.Digest{Key: 100, DataType: telem.Float64T},
			channels.Digest{Key: 200, DataType: telem.Float64T},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV(3.0))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		Expect(out.Get(200).Series).ToNot(BeEmpty())
		Expect(out.Get(200).Series[0].Unmarshal[float64]()[0]).To(Equal(6.0))
	})

	DescribeTable("Should reject a variable with a non-constant initializer",
		func(ctx SpecContext, source string) {
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
				Name: "x",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   100,
			}, &symbol.Symbol{
				Name: "out",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   200,
			}))
			Expect(arc.CompileText(ctx, arc.Text{Raw: source}, root)).Error().To(
				MatchError(ContainSubstring(
					"cannot use 'y' in a flow: its initializer is not a compile-time constant",
				)),
			)
		},
		Entry("top-level source", "y := 1.5 + 2.0\ny -> out"),
		Entry(
			"stage source",
			"sequence main {\n stage a {\n y := 1.5 + 2.0\n y -> out\n }\n}\nx => main",
		),
		Entry(
			"stage sink",
			"sequence main {\n stage a {\n y := 1.5 + 2.0\n x -> y\n }\n}\nx => main",
		),
		Entry(
			"value variable derived from a reactive variable",
			"sequence main {\n a f64 := x + 1.0\n y f64 := a + 2.0\n stage s {\n"+
				" y -> out\n }\n}\nx => main",
		),
		Entry(
			"sequence variable written and read",
			"sequence main {\n y := 1.5 + 2.0\n stage a {\n x -> y\n y -> out\n }\n}\n"+
				"x => main",
		),
	)
})
