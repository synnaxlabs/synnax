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
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Stat Flow Chains", func() {
	Describe("avg", func() {
		It("Should compute the average through a flow chain", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"my_sensor":     {types.F64(), 100},
				"output_sensor": {types.F64(), 200},
			})
			h := newRuntimeHarness(ctx, `my_sensor -> avg{} -> output_sensor`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(10.0, 20.0, 30.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			result := h.Output("avg_0", 0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(BeNumerically("~", 20.0, 0.01))

			resultTime := h.OutputTime("avg_0", 0)
			Expect(resultTime.Len()).To(Equal(int64(1)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float64](out.Get(200).Series[0])[0]).To(BeNumerically("~", 20.0, 0.01))
		})

		It("Should compute the average with int32 type", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":  {types.I32(), 100},
				"avg_out": {types.I32(), 200},
			})
			h := newRuntimeHarness(ctx, `sensor -> avg{} -> avg_out`, resolver,
				channel.Digest{Key: 100, DataType: telem.Int32T},
				channel.Digest{Key: 200, DataType: telem.Int32T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[int32](10, 20, 30))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			result := h.Output("avg_0", 0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(BeNumerically("~", 20, 1))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(telem.UnmarshalSeries[int32](out.Get(200).Series[0])[0]).To(BeNumerically("~", 20, 1))
		})
	})

	Describe("min", func() {
		It("Should compute the minimum through a flow chain", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"my_sensor":     {types.F64(), 100},
				"output_sensor": {types.F64(), 200},
			})
			h := newRuntimeHarness(ctx, `my_sensor -> min{} -> output_sensor`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(50.0, 10.0, 30.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			result := h.Output("min_0", 0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(BeNumerically("~", 10.0, 0.01))
			Expect(h.OutputTime("min_0", 0).Len()).To(Equal(int64(1)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(telem.UnmarshalSeries[float64](out.Get(200).Series[0])[0]).To(BeNumerically("~", 10.0, 0.01))
		})
	})

	Describe("max", func() {
		It("Should compute the maximum through a flow chain", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"my_sensor":     {types.F64(), 100},
				"output_sensor": {types.F64(), 200},
			})
			h := newRuntimeHarness(ctx, `my_sensor -> max{} -> output_sensor`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(10.0, 50.0, 30.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			result := h.Output("max_0", 0)
			Expect(result.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(BeNumerically("~", 50.0, 0.01))
			Expect(h.OutputTime("max_0", 0).Len()).To(Equal(int64(1)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(telem.UnmarshalSeries[float64](out.Get(200).Series[0])[0]).To(BeNumerically("~", 50.0, 0.01))
		})
	})

	Describe("derivative", func() {
		It("Should compute pointwise derivative through a flow chain", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"my_sensor": {types.F64(), 100},
				"rate_out":  {types.F64(), 200},
			})
			h := newRuntimeHarness(ctx, `my_sensor -> derivative{} -> rate_out`, resolver,
				channel.Digest{Key: 99, DataType: telem.TimeStampT},
				channel.Digest{Key: 100, DataType: telem.Float64T, Index: 99},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.IngestIndexed(99, telem.NewSeriesSecondsTSV(1, 2, 4), 100, telem.NewSeriesV(10.0, 20.0, 40.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			result := h.Output("derivative_0", 0)
			Expect(result.Len()).To(Equal(int64(3)))
			Expect(h.OutputTime("derivative_0", 0).Len()).To(Equal(int64(3)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			vals := telem.UnmarshalSeries[float64](out.Get(200).Series[0])
			Expect(vals).To(HaveLen(3))
			Expect(vals[0]).To(BeNumerically("~", 0.0, 0.01))
			Expect(vals[1]).To(BeNumerically("~", 10.0, 0.01))
			Expect(vals[2]).To(BeNumerically("~", 10.0, 0.01))
		})
	})
})
