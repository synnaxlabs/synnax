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

var demuxSource = `
func demux{threshold f64} (value f64) (high f64, low f64) {
    if (value > threshold) {
        high = value
    } else {
        low = value
    }
}
`

var _ = Describe("Routing Table Runtime", func() {
	Describe("Output Routing", func() {
		It("Should route to high output with correct data, alignment, time range, and timestamps", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":   {types.F64(), 100},
				"high_out": {types.F64(), 200},
				"low_out":  {types.F64(), 300},
			})
			h := newRuntimeHarness(ctx, demuxSource+`
				sensor -> demux{threshold=50.0} -> {
					high: high_out,
					low: low_out
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
				channel.Digest{Key: 300, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(75.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			highResult := h.Output("demux_0", 0)
			Expect(highResult.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[float64](highResult)).To(Equal([]float64{75.0}))

			highTime := h.OutputTime("demux_0", 0)
			Expect(highTime.Len()).To(Equal(int64(1)))

			lowResult := h.Output("demux_0", 1)
			Expect(lowResult.Len()).To(Equal(int64(0)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float64](out.Get(200).Series[0])).To(Equal([]float64{75.0}))
			Expect(out.Get(300).Series).To(HaveLen(0))
		})

		It("Should route to low output and produce empty high output", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":   {types.F64(), 100},
				"high_out": {types.F64(), 200},
				"low_out":  {types.F64(), 300},
			})
			h := newRuntimeHarness(ctx, demuxSource+`
				sensor -> demux{threshold=50.0} -> {
					high: high_out,
					low: low_out
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
				channel.Digest{Key: 300, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(25.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			highResult := h.Output("demux_0", 0)
			Expect(highResult.Len()).To(Equal(int64(0)))

			lowResult := h.Output("demux_0", 1)
			Expect(lowResult.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[float64](lowResult)).To(Equal([]float64{25.0}))

			lowTime := h.OutputTime("demux_0", 1)
			Expect(lowTime.Len()).To(Equal(int64(1)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(0))
			Expect(out.Get(300).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float64](out.Get(300).Series[0])).To(Equal([]float64{25.0}))
		})

		It("Should split a multi-sample batch across outputs with per-sample timestamps", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":   {types.F64(), 100},
				"high_out": {types.F64(), 200},
				"low_out":  {types.F64(), 300},
			})
			h := newRuntimeHarness(ctx, demuxSource+`
				sensor -> demux{threshold=50.0} -> {
					high: high_out,
					low: low_out
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
				channel.Digest{Key: 300, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(80.0, 20.0, 90.0, 10.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			highResult := h.Output("demux_0", 0)
			Expect(telem.UnmarshalSeries[float64](highResult)).To(Equal([]float64{80.0, 90.0}))
			Expect(h.OutputTime("demux_0", 0).Len()).To(Equal(int64(2)))

			lowResult := h.Output("demux_0", 1)
			Expect(telem.UnmarshalSeries[float64](lowResult)).To(Equal([]float64{20.0, 10.0}))
			Expect(h.OutputTime("demux_0", 1).Len()).To(Equal(int64(2)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(telem.UnmarshalSeries[float64](out.Get(200).Series[0])).To(Equal([]float64{80.0, 90.0}))
			Expect(telem.UnmarshalSeries[float64](out.Get(300).Series[0])).To(Equal([]float64{20.0, 10.0}))
		})

		It("Should route three named outputs to separate channels", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":   {types.I64(), 100},
				"neg_out":  {types.I64(), 200},
				"zero_out": {types.I64(), 300},
				"pos_out":  {types.I64(), 400},
			})
			h := newRuntimeHarness(ctx, `
				func classify{} (value i64) (negative i64, zero i64, positive i64) {
				    if (value < 0) {
				        negative = value
				    } else if (value > 0) {
				        positive = value
				    } else {
				        zero = value
				    }
				}

				sensor -> classify{} -> {
				    negative: neg_out,
				    zero: zero_out,
				    positive: pos_out
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Int64T},
				channel.Digest{Key: 200, DataType: telem.Int64T},
				channel.Digest{Key: 300, DataType: telem.Int64T},
				channel.Digest{Key: 400, DataType: telem.Int64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[int64](-5, 0, 42))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			negResult := h.Output("classify_0", 0)
			Expect(telem.UnmarshalSeries[int64](negResult)).To(Equal([]int64{-5}))

			zeroResult := h.Output("classify_0", 1)
			Expect(telem.UnmarshalSeries[int64](zeroResult)).To(Equal([]int64{0}))

			posResult := h.Output("classify_0", 2)
			Expect(telem.UnmarshalSeries[int64](posResult)).To(Equal([]int64{42}))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(telem.UnmarshalSeries[int64](out.Get(200).Series[0])).To(Equal([]int64{-5}))
			Expect(telem.UnmarshalSeries[int64](out.Get(300).Series[0])).To(Equal([]int64{0}))
			Expect(telem.UnmarshalSeries[int64](out.Get(400).Series[0])).To(Equal([]int64{42}))
		})
	})

	Describe("Chained Routing", func() {
		It("Should route through a processing function with correct intermediate outputs", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":    {types.F64(), 100},
				"alarm_out": {types.F64(), 200},
			})
			h := newRuntimeHarness(ctx, demuxSource+`
				func amplify{} (signal f64) f64 {
				    return signal * 2.0
				}

				sensor -> demux{threshold=50.0} -> {
				    high: amplify{} -> alarm_out
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(80.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			demuxHigh := h.Output("demux_0", 0)
			Expect(telem.UnmarshalSeries[float64](demuxHigh)).To(Equal([]float64{80.0}))

			amplifyResult := h.Output("amplify_0", 0)
			Expect(telem.UnmarshalSeries[float64](amplifyResult)).To(Equal([]float64{160.0}))
			Expect(h.OutputTime("amplify_0", 0).Len()).To(Equal(int64(1)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(telem.UnmarshalSeries[float64](out.Get(200).Series[0])).To(Equal([]float64{160.0}))
		})

		It("Should not propagate to chained function when branch receives no data", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":    {types.F64(), 100},
				"alarm_out": {types.F64(), 200},
			})
			h := newRuntimeHarness(ctx, demuxSource+`
				func amplify{} (signal f64) f64 {
				    return signal * 2.0
				}

				sensor -> demux{threshold=50.0} -> {
				    high: amplify{} -> alarm_out
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(25.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			demuxHigh := h.Output("demux_0", 0)
			Expect(demuxHigh.Len()).To(Equal(int64(0)))

			amplifyResult := h.Output("amplify_0", 0)
			Expect(amplifyResult.Len()).To(Equal(int64(0)))

			_, changed := h.Flush()
			Expect(changed).To(BeFalse())
		})

		It("Should chain multiple samples through amplify preserving per-sample timestamps", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":    {types.F64(), 100},
				"alarm_out": {types.F64(), 200},
			})
			h := newRuntimeHarness(ctx, demuxSource+`
				func amplify{} (signal f64) f64 {
				    return signal * 2.0
				}

				sensor -> demux{threshold=50.0} -> {
				    high: amplify{} -> alarm_out
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV(80.0, 30.0, 90.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			demuxHigh := h.Output("demux_0", 0)
			Expect(telem.UnmarshalSeries[float64](demuxHigh)).To(Equal([]float64{80.0, 90.0}))
			Expect(h.OutputTime("demux_0", 0).Len()).To(Equal(int64(2)))

			amplifyResult := h.Output("amplify_0", 0)
			Expect(telem.UnmarshalSeries[float64](amplifyResult)).To(Equal([]float64{160.0, 180.0}))
			Expect(h.OutputTime("amplify_0", 0).Len()).To(Equal(int64(2)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(telem.UnmarshalSeries[float64](out.Get(200).Series[0])).To(Equal([]float64{160.0, 180.0}))
		})
	})

	Describe("Routing to Sequences", func() {
		It("Should activate a stage that writes a constant on the activation tick", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":  {types.F64(), 100},
				"vlv_cmd": {types.U8(), 200},
			})
			h := newRuntimeHarness(ctx, `
				func demux{threshold f64} (value f64) (high u8, low f64) {
				    if (value > threshold) {
				        high = 1
				    } else {
				        low = value
				    }
				}

				sequence alarm {
				    stage active {
				        1 -> vlv_cmd
				    }
				}

				sensor -> demux{threshold=50.0} -> {
				    high: alarm
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			// Tick 1: above threshold. demux fires high, one-shot activates
			// the alarm sequence's active stage. The constant node writes
			// 1 to vlv_cmd on the activation tick.
			h.Ingest(100, telem.NewSeriesV(75.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[uint8](out.Get(200).Series[0])).To(Equal([]uint8{1}))

			// Tick 2: the constant node already fired. No new writes
			// are produced because constant nodes fire once per stage activation.
			h.Tick(ctx, 2*telem.Millisecond)

			_, changed2 := h.Flush()
			Expect(changed2).To(BeFalse())

			// Tick 3: below-threshold value. The demux high branch doesn't
			// fire, so no re-activation occurs. Still no writes.
			h.Ingest(100, telem.NewSeriesV(25.0))
			h.Tick(ctx, 3*telem.Millisecond)
			h.channelState.ClearReads()

			_, changed3 := h.Flush()
			Expect(changed3).To(BeFalse())
		})

		It("Should not activate the sequence across multiple ticks when the branch never fires", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":  {types.F64(), 100},
				"vlv_cmd": {types.U8(), 200},
			})
			h := newRuntimeHarness(ctx, `
				func demux{threshold f64} (value f64) (high u8, low f64) {
				    if (value > threshold) {
				        high = 1
				    } else {
				        low = value
				    }
				}

				sequence alarm {
				    stage active {
				        1 -> vlv_cmd
				    }
				}

				sensor -> demux{threshold=50.0} -> {
				    high: alarm
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			for i := range 3 {
				h.Ingest(100, telem.NewSeriesV(25.0))
				h.Tick(ctx, telem.TimeSpan(i+1)*telem.Millisecond)
				h.channelState.ClearReads()

				_, changed := h.Flush()
				Expect(changed).To(BeFalse(), "tick %d should not produce writes", i+1)
			}
		})

		It("Should only activate the sequence whose routing branch fires", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":   {types.F64(), 100},
				"open_cmd": {types.U8(), 200},
				"log_cmd":  {types.U8(), 300},
			})
			h := newRuntimeHarness(ctx, `
				func classify{threshold f64} (value f64) (above u8, below u8) {
				    if (value > threshold) {
				        above = 1
				    } else {
				        below = 1
				    }
				}

				sequence open_valve {
				    stage active {
				        1 -> open_cmd
				    }
				}

				sequence log_event {
				    stage active {
				        1 -> log_cmd
				    }
				}

				sensor -> classify{threshold=100.0} -> {
				    above: open_valve,
				    below: log_event
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Uint8T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			// Tick 1: above threshold. Only open_valve activates.
			h.Ingest(100, telem.NewSeriesV(150.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[uint8](out.Get(200).Series[0])).To(Equal([]uint8{1}))
			Expect(out.Get(300).Series).To(HaveLen(0))

			// Tick 2: below threshold. log_event activates. open_valve
			// constant already fired so it doesn't re-emit.
			h.Ingest(100, telem.NewSeriesV(50.0))
			h.Tick(ctx, 2*telem.Millisecond)
			h.channelState.ClearReads()

			out2, changed2 := h.Flush()
			Expect(changed2).To(BeTrue())
			Expect(out2.Get(200).Series).To(HaveLen(0))
			Expect(out2.Get(300).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[uint8](out2.Get(300).Series[0])).To(Equal([]uint8{1}))
		})

		It("Should activate a multi-stage sequence and transition between stages", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":   {types.F64(), 100},
				"press_pt": {types.F32(), 101},
				"vlv_cmd":  {types.U8(), 200},
			})
			h := newRuntimeHarness(ctx, `
				func demux{threshold f64} (value f64) (high u8, low f64) {
				    if (value > threshold) {
				        high = 1
				    } else {
				        low = value
				    }
				}

				func check_pressure(p f32) u8 {
				    return p > 100
				}

				sequence pressurize {
				    stage fill {
				        1 -> vlv_cmd
				        press_pt -> check_pressure{} => next
				    }
				    stage hold {
				        0 -> vlv_cmd
				    }
				}

				sensor -> demux{threshold=50.0} -> {
				    high: pressurize
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
				channel.Digest{Key: 200, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			// Tick 1: activate the sequence. The fill stage opens the valve
			// (constant 1) and evaluates the pressure check. Pressure is
			// below 100 so no transition occurs.
			h.Ingest(100, telem.NewSeriesV(75.0))
			h.Ingest(101, telem.NewSeriesV[float32](50.0))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(telem.UnmarshalSeries[uint8](out.Get(200).Series[0])).To(Equal([]uint8{1}))

			// Tick 2: pressure rises above 100. check_pressure returns truthy,
			// firing the one-shot transition to the hold stage. The hold stage's
			// constant writes 0 to vlv_cmd.
			h.Ingest(101, telem.NewSeriesV[float32](150.0))
			h.Tick(ctx, 2*telem.Millisecond)
			h.channelState.ClearReads()

			out2, changed2 := h.Flush()
			Expect(changed2).To(BeTrue())
			Expect(telem.UnmarshalSeries[uint8](out2.Get(200).Series[0])).To(Equal([]uint8{0}))

			// Tick 3: no new data. Hold stage constant already fired.
			// No further writes.
			h.Tick(ctx, 3*telem.Millisecond)

			_, changed3 := h.Flush()
			Expect(changed3).To(BeFalse())
		})
	})

	Describe("Routing with select{}", func() {
		It("Should use select to route a boolean channel into different sequence stages", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"flag":     {types.U8(), 100},
				"open_cmd": {types.U8(), 200},
				"shut_cmd": {types.U8(), 300},
			})
			h := newRuntimeHarness(ctx, `
				flag -> select{} -> {
				    true: open_valve,
				    false: shut_valve
				}

				sequence open_valve {
				    stage active {
				        1 -> open_cmd
				    }
				}

				sequence shut_valve {
				    stage active {
				        1 -> shut_cmd
				    }
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 200, DataType: telem.Uint8T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			// Tick 1: flag=1 (truthy). select routes to "true" output,
			// activating open_valve.
			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			selectTrue := h.Output("select_0", 0)
			Expect(selectTrue.Len()).To(Equal(int64(1)))
			selectFalse := h.Output("select_0", 1)
			Expect(selectFalse.Len()).To(Equal(int64(0)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[uint8](out.Get(200).Series[0])).To(Equal([]uint8{1}))
			Expect(out.Get(300).Series).To(HaveLen(0))

			// Tick 2: flag=0 (falsy). select routes to "false" output,
			// activating shut_valve.
			h.Ingest(100, telem.NewSeriesV[uint8](0))
			h.Tick(ctx, 2*telem.Millisecond)
			h.channelState.ClearReads()

			selectTrue2 := h.Output("select_0", 0)
			Expect(selectTrue2.Len()).To(Equal(int64(0)))
			selectFalse2 := h.Output("select_0", 1)
			Expect(selectFalse2.Len()).To(Equal(int64(1)))

			out2, changed2 := h.Flush()
			Expect(changed2).To(BeTrue())
			Expect(out2.Get(200).Series).To(HaveLen(0))
			Expect(out2.Get(300).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[uint8](out2.Get(300).Series[0])).To(Equal([]uint8{1}))
		})
	})

	Describe("Routing to Stages", func() {
		It("Should compile and execute a routing table that targets a stage within the same sequence", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trigger": {types.U8(), 50},
				"sensor":  {types.F64(), 100},
				"vlv_cmd": {types.U8(), 200},
			})
			h := newRuntimeHarness(ctx, `
				func demux{threshold f64} (value f64) (high u8, low u8) {
				    if (value > threshold) {
				        high = 1
				    } else {
				        low = 1
				    }
				}

				trigger => main

				sequence main {
				    stage first {
				        sensor -> demux{threshold=50.0} -> {
				            high: pressurize,
				        }
				    }
				    stage pressurize {
				        1 -> vlv_cmd
				    }
				}`, resolver,
				channel.Digest{Key: 50, DataType: telem.Uint8T},
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 200, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			// Tick 1: activate the sequence via trigger. The first stage
			// becomes active but has no sensor data yet.
			h.Ingest(50, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			_, changed := h.Flush()
			Expect(changed).To(BeFalse())

			// Tick 2: provide above-threshold sensor data. The first stage's
			// source node sees fresh data (arriving after stage activation),
			// demux routes high to pressurize stage, which writes 1 to vlv_cmd.
			h.Ingest(100, telem.NewSeriesV(75.0))
			h.Tick(ctx, 2*telem.Millisecond)
			h.channelState.ClearReads()

			out, changed2 := h.Flush()
			Expect(changed2).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[uint8](out.Get(200).Series[0])).To(Equal([]uint8{1}))
		})
	})
})
