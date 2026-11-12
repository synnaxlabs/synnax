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
	"github.com/synnaxlabs/arc/stl/channels"
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

	It("Should send a sequence variable derived from a channel", func(ctx SpecContext) {
		h := newRuntimeHarness(ctx, `
		sequence main {
			a f64 := x + 1.0
			stage s {
				a -> out
			}
		}
		x => main`, channelSymbols(
			map[string]channelDef{"x": {types.F64(), 100}, "out": {types.F64(), 200}},
		),
			channels.Digest{Key: 100, DataType: telem.Float64T},
			channels.Digest{Key: 200, DataType: telem.Float64T},
		)
		defer h.Close(ctx)
		for range 3 {
			h.Ingest(100, telem.NewSeriesV(3.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		series := out.Get(200).Series
		Expect(series).ToNot(BeEmpty())
		Expect(series[len(series)-1].Unmarshal[float64]()).To(ContainElement(4.0))
	})

	DescribeTable("Should send a variable with a computed initializer",
		func(ctx SpecContext, source string, expected []float64) {
			h := newRuntimeHarness(ctx, source, channelSymbols(
				map[string]channelDef{
					"x":   {types.F64(), 100},
					"out": {types.F64(), 200},
				},
			),
				channels.Digest{Key: 100, DataType: telem.Float64T},
				channels.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)
			for _, v := range []float64{3, 5} {
				h.Ingest(100, telem.NewSeriesV(v))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			var values []float64
			for _, s := range out.Get(200).Series {
				values = append(values, s.Unmarshal[float64]()...)
			}
			Expect(values).To(Equal(expected))
		},
		Entry("top-level source", "y := 1.5 + 2.0\ny -> out", []float64{3.5}),
		Entry(
			"stage source",
			"sequence main {\n stage a {\n y := 1.5 + 2.0\n y -> out\n }\n}\nx => main",
			[]float64{3.5},
		),
		Entry(
			"stage sink",
			"sequence main {\n stage a {\n y := 1.5 + 2.0\n x -> y\n y -> out\n }\n}\nx => main",
			[]float64{3.5, 5},
		),
		Entry(
			"value variable derived from a reactive variable",
			"sequence main {\n a f64 := x + 1.0\n y f64 := a + 2.0\n stage s {\n"+
				" y -> out\n }\n}\nx => main",
			[]float64{6, 8},
		),
		Entry(
			"sequence variable written and read",
			"sequence main {\n y := 1.5 + 2.0\n stage a {\n x -> y\n y -> out\n }\n}\n"+
				"x => main",
			[]float64{3.5, 5},
		),
	)
})
