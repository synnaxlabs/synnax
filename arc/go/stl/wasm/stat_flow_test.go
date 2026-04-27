// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

func channelResolver(channels map[string]struct {
	dt types.Type
	id int
},
) symbol.MapResolver {
	r := symbol.MapResolver{}
	for name, ch := range channels {
		r[name] = symbol.Symbol{
			Name: name,
			Kind: symbol.KindChannel,
			Type: types.Chan(ch.dt),
			ID:   ch.id,
		}
	}
	return r
}

var _ = Describe("Stat Flow Chains", func() {
	Describe("avg", func() {
		It("Should compute the average through a flow chain", func(ctx SpecContext) {
			resolver := channelResolver(map[string]struct {
				dt types.Type
				id int
			}{
				"my_sensor":     {types.F64(), 100},
				"output_sensor": {types.F64(), 200},
			})
			h := newTextHarness(ctx, `my_sensor -> avg{} -> output_sensor`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			inputData := telem.NewSeriesV(10.0, 20.0, 30.0)
			inputData.Alignment = 7
			inputData.TimeRange = telem.TimeRange{
				Start: 1 * telem.SecondTS,
				End:   3 * telem.SecondTS,
			}
			h.SetInput("on_my_sensor_0", 0,
				inputData,
				telem.NewSeriesSecondsTSV(1, 2, 3),
			)
			h.Execute(ctx, "avg_0")

			result := h.Output("avg_0", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(BeNumerically("~", 20.0, 0.01))
			Expect(result.Alignment).To(Equal(telem.Alignment(7)))
			Expect(result.TimeRange.Start).To(Equal(1 * telem.SecondTS))

			resultTime := h.OutputTime("avg_0", 0)
			Expect(resultTime.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[telem.TimeStamp](resultTime)[0]).To(Equal(3 * telem.SecondTS))
		})

		It("Should compute the average with int32 type", func(ctx SpecContext) {
			resolver := channelResolver(map[string]struct {
				dt types.Type
				id int
			}{
				"sensor":  {types.I32(), 100},
				"avg_out": {types.I32(), 200},
			})
			h := newTextHarness(ctx, `sensor -> avg{} -> avg_out`, resolver,
				channel.Digest{Key: 100, DataType: telem.Int32T},
				channel.Digest{Key: 200, DataType: telem.Int32T},
			)
			defer h.Close(ctx)

			h.SetInput("on_sensor_0", 0,
				telem.NewSeriesV[int32](10, 20, 30),
				telem.NewSeriesSecondsTSV(1, 2, 3),
			)
			h.Execute(ctx, "avg_0")

			result := h.Output("avg_0", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(BeNumerically("~", 20, 1))
		})
	})

	Describe("min", func() {
		It("Should compute the minimum through a flow chain", func(ctx SpecContext) {
			resolver := channelResolver(map[string]struct {
				dt types.Type
				id int
			}{
				"my_sensor":     {types.F64(), 100},
				"output_sensor": {types.F64(), 200},
			})
			h := newTextHarness(ctx, `my_sensor -> min{} -> output_sensor`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.SetInput("on_my_sensor_0", 0,
				telem.NewSeriesV(50.0, 10.0, 30.0),
				telem.NewSeriesSecondsTSV(1, 2, 3),
			)
			h.Execute(ctx, "min_0")

			result := h.Output("min_0", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(BeNumerically("~", 10.0, 0.01))
		})
	})

	Describe("max", func() {
		It("Should compute the maximum through a flow chain", func(ctx SpecContext) {
			resolver := channelResolver(map[string]struct {
				dt types.Type
				id int
			}{
				"my_sensor":     {types.F64(), 100},
				"output_sensor": {types.F64(), 200},
			})
			h := newTextHarness(ctx, `my_sensor -> max{} -> output_sensor`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.SetInput("on_my_sensor_0", 0,
				telem.NewSeriesV(10.0, 50.0, 30.0),
				telem.NewSeriesSecondsTSV(1, 2, 3),
			)
			h.Execute(ctx, "max_0")

			result := h.Output("max_0", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(BeNumerically("~", 50.0, 0.01))
		})
	})

	Describe("derivative", func() {
		It("Should compute pointwise derivative through a flow chain", func(ctx SpecContext) {
			resolver := channelResolver(map[string]struct {
				dt types.Type
				id int
			}{
				"my_sensor": {types.F64(), 100},
				"rate_out":  {types.F64(), 200},
			})
			h := newTextHarness(ctx, `my_sensor -> derivative{} -> rate_out`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.SetInput("on_my_sensor_0", 0,
				telem.NewSeriesV(10.0, 20.0, 40.0),
				telem.NewSeriesSecondsTSV(1, 2, 4),
			)
			h.Execute(ctx, "derivative_0")

			result := h.Output("derivative_0", 0)
			vals := telem.UnmarshalSeries[float64](result)
			Expect(vals).To(HaveLen(3))
			Expect(vals[0]).To(BeNumerically("~", 0.0, 0.01))
			Expect(vals[1]).To(BeNumerically("~", 10.0, 0.01))
			Expect(vals[2]).To(BeNumerically("~", 10.0, 0.01))
		})
	})

	Describe("Full chain", func() {
		It("Should execute source -> avg -> sink and flush channel writes", func(ctx SpecContext) {
			resolver := channelResolver(map[string]struct {
				dt types.Type
				id int
			}{
				"my_sensor":     {types.F64(), 100},
				"output_sensor": {types.F64(), 200},
			})
			h := newTextHarness(ctx, `my_sensor -> avg{} -> output_sensor`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			fr := telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV(10.0, 20.0, 30.0))
			h.ChannelState().Ingest(fr)

			h.Execute(ctx, "on_my_sensor_0")
			h.Execute(ctx, "avg_0")
			h.Execute(ctx, "write_output_sensor_0")

			out, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(out.Get(200).Series[0]).To(telem.MatchSeriesDataV[float64](20.0))
		})

		It("Should execute source -> min -> sink and flush channel writes", func(ctx SpecContext) {
			resolver := channelResolver(map[string]struct {
				dt types.Type
				id int
			}{
				"pressure": {types.F64(), 100},
				"min_psi":  {types.F64(), 200},
			})
			h := newTextHarness(ctx, `pressure -> min{} -> min_psi`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			fr := telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV(50.0, 10.0, 30.0))
			h.ChannelState().Ingest(fr)

			h.Execute(ctx, "on_pressure_0")
			h.Execute(ctx, "min_0")
			h.Execute(ctx, "write_min_psi_0")

			out, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series[0]).To(telem.MatchSeriesDataV[float64](10.0))
		})
	})
})
