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
		It(
			"Should route to high output with correct data, alignment, time range, and timestamps",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":   {types.F64(), 100},
					"high_out": {types.F64(), 200},
					"low_out":  {types.F64(), 300},
				})
				h := newRuntimeHarness(ctx, demuxSource+`
				sensor -> demux{threshold=50.0} -> {
					high: 1.0 -> high_out,
					low: 2.0 -> low_out
				}
			`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Float64T},
					channels.Digest{Key: 300, DataType: telem.Float64T},
				)
				defer h.Close(ctx)

				h.Ingest(100, telem.NewSeriesV(75.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				highResult := h.Output("demux_0", 0)
				Expect(highResult.Len()).To(Equal(int64(1)))
				Expect(
					telem.UnmarshalSeries[float64](highResult),
				).To(Equal([]float64{75.0}))

				highTime := h.OutputTime("demux_0", 0)
				Expect(highTime.Len()).To(Equal(int64(1)))

				lowResult := h.Output("demux_0", 1)
				Expect(lowResult.Len()).To(Equal(int64(0)))

				// The branch only gates the entry: the write carries the entry's
				// own constant, not the routed value.
				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(out.Get(200).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[float64](out.Get(200).Series[0]),
				).To(Equal([]float64{1.0}))
				Expect(out.Get(300).Series).To(BeEmpty())
			},
		)

		It(
			"Should route to low output and produce empty high output",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":   {types.F64(), 100},
					"high_out": {types.F64(), 200},
					"low_out":  {types.F64(), 300},
				})
				h := newRuntimeHarness(ctx, demuxSource+`
				sensor -> demux{threshold=50.0} -> {
					high: 1.0 -> high_out,
					low: 2.0 -> low_out
				}
			`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Float64T},
					channels.Digest{Key: 300, DataType: telem.Float64T},
				)
				defer h.Close(ctx)

				h.Ingest(100, telem.NewSeriesV(25.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				highResult := h.Output("demux_0", 0)
				Expect(highResult.Len()).To(Equal(int64(0)))

				lowResult := h.Output("demux_0", 1)
				Expect(lowResult.Len()).To(Equal(int64(1)))
				Expect(
					telem.UnmarshalSeries[float64](lowResult),
				).To(Equal([]float64{25.0}))

				lowTime := h.OutputTime("demux_0", 1)
				Expect(lowTime.Len()).To(Equal(int64(1)))

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(out.Get(200).Series).To(BeEmpty())
				Expect(out.Get(300).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[float64](out.Get(300).Series[0]),
				).To(Equal([]float64{2.0}))
			},
		)

		It(
			"Should split a multi-sample batch across outputs with per-sample timestamps",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":   {types.F64(), 100},
					"high_out": {types.F64(), 200},
					"low_out":  {types.F64(), 300},
				})
				h := newRuntimeHarness(ctx, demuxSource+`
				sensor -> demux{threshold=50.0} -> {
					high: 1.0 -> high_out,
					low: 2.0 -> low_out
				}
			`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Float64T},
					channels.Digest{Key: 300, DataType: telem.Float64T},
				)
				defer h.Close(ctx)

				h.Ingest(100, telem.NewSeriesV(80.0, 20.0, 90.0, 10.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				highResult := h.Output("demux_0", 0)
				Expect(
					telem.UnmarshalSeries[float64](highResult),
				).To(Equal([]float64{80.0, 90.0}))
				Expect(h.OutputTime("demux_0", 0).Len()).To(Equal(int64(2)))

				lowResult := h.Output("demux_0", 1)
				Expect(
					telem.UnmarshalSeries[float64](lowResult),
				).To(Equal([]float64{20.0, 10.0}))
				Expect(h.OutputTime("demux_0", 1).Len()).To(Equal(int64(2)))

				// Each entry's constant fires once per trigger batch, so the
				// writes carry one sample regardless of the batch size.
				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(
					telem.UnmarshalSeries[float64](out.Get(200).Series[0]),
				).To(Equal([]float64{1.0}))
				Expect(
					telem.UnmarshalSeries[float64](out.Get(300).Series[0]),
				).To(Equal([]float64{2.0}))
			},
		)

		It(
			"Should route three named outputs to separate channels",
			func(ctx SpecContext) {
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
				    negative: -1 -> neg_out,
				    zero: 0 -> zero_out,
				    positive: 1 -> pos_out
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Int64T},
					channels.Digest{Key: 200, DataType: telem.Int64T},
					channels.Digest{Key: 300, DataType: telem.Int64T},
					channels.Digest{Key: 400, DataType: telem.Int64T},
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
				Expect(
					telem.UnmarshalSeries[int64](out.Get(200).Series[0]),
				).To(Equal([]int64{-1}))
				Expect(
					telem.UnmarshalSeries[int64](out.Get(300).Series[0]),
				).To(Equal([]int64{0}))
				Expect(
					telem.UnmarshalSeries[int64](out.Get(400).Series[0]),
				).To(Equal([]int64{1}))
			},
		)

		It(
			"Should fire entries for mixed bool and u8 outputs in the same tick",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":   {types.F64(), 100},
					"flag_out": {types.U8(), 200},
					"mode_out": {types.U8(), 300},
				})
				h := newRuntimeHarness(ctx, `
				func split{} (value f64) (ok bool, mode u8) {
				    mode = 2
				    ok = value > 50.0
				}

				sensor -> split{} -> {
				    ok: 1 -> flag_out,
				    mode: 2 -> mode_out
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
					channels.Digest{Key: 300, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				// One input sets both outputs, so both entries run in the same
				// tick regardless of their output types.
				h.Ingest(100, telem.NewSeriesV(75.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				// mode is written first, so a too-wide bool store clobbers it.
				Expect(
					telem.UnmarshalSeries[bool](h.Output("split_0", 0)),
				).To(Equal([]bool{true}))
				Expect(
					telem.UnmarshalSeries[uint8](h.Output("split_0", 1)),
				).To(Equal([]uint8{2}))

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(out.Get(200).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
				).To(Equal([]uint8{1}))
				Expect(out.Get(300).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(300).Series[0]),
				).To(Equal([]uint8{2}))
			},
		)

		It(
			"Should fire entries for mixed numeric and string outputs in the same tick",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":    {types.F64(), 100},
					"count_out": {types.U8(), 200},
					"mode_out":  {types.U8(), 300},
					"total_out": {types.U8(), 400},
					"ratio_out": {types.U8(), 500},
					"mean_out":  {types.U8(), 600},
					"label_out": {types.U8(), 700},
				})
				h := newRuntimeHarness(ctx, `
				func spread{} (value f64) (
				    count u16,
				    mode i32,
				    total i64,
				    ratio f32,
				    mean f64,
				    label str,
				) {
				    label = "ok"
				    mean = value
				    ratio = 1.5
				    total = 5000000000
				    mode = -3
				    count = 300
				}

				sensor -> spread{} -> {
				    count: 1 -> count_out,
				    mode: 2 -> mode_out,
				    total: 3 -> total_out,
				    ratio: 4 -> ratio_out,
				    mean: 5 -> mean_out,
				    label: 6 -> label_out
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
					channels.Digest{Key: 300, DataType: telem.Uint8T},
					channels.Digest{Key: 400, DataType: telem.Uint8T},
					channels.Digest{Key: 500, DataType: telem.Uint8T},
					channels.Digest{Key: 600, DataType: telem.Uint8T},
					channels.Digest{Key: 700, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				// Values need every byte of their width and are written in reverse
				// declaration order, so a wrong store width fails the readbacks.
				h.Ingest(100, telem.NewSeriesV(75.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				Expect(
					telem.UnmarshalSeries[uint16](h.Output("spread_0", 0)),
				).To(Equal([]uint16{300}))
				Expect(
					telem.UnmarshalSeries[int32](h.Output("spread_0", 1)),
				).To(Equal([]int32{-3}))
				Expect(
					telem.UnmarshalSeries[int64](h.Output("spread_0", 2)),
				).To(Equal([]int64{5000000000}))
				Expect(
					telem.UnmarshalSeries[float32](h.Output("spread_0", 3)),
				).To(Equal([]float32{1.5}))
				Expect(
					telem.UnmarshalSeries[float64](h.Output("spread_0", 4)),
				).To(Equal([]float64{75.0}))
				Expect(
					telem.UnmarshalSeries[string](h.Output("spread_0", 5)),
				).To(Equal([]string{"ok"}))

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				for i, key := range []uint32{200, 300, 400, 500, 600, 700} {
					Expect(out.Get(key).Series).To(HaveLen(1))
					Expect(
						telem.UnmarshalSeries[uint8](out.Get(key).Series[0]),
					).To(Equal([]uint8{uint8(i + 1)}))
				}
			},
		)

		It(
			"Should fire a transition entry when the output is set to a falsy value",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":  {types.F64(), 100},
					"vlv_cmd": {types.U8(), 200},
				})
				h := newRuntimeHarness(ctx, `
				func gate{} (value f64) (level u8) {
				    level = 0
				}

				sequence alarm {
				    stage active {
				        1 -> vlv_cmd
				    }
				}

				sensor -> gate{} -> {
				    level: true => alarm
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				// gate sets level to 0. Setting the output fires the entry, and
				// the transition gates on the entry's own constant, so the
				// falsy output value still activates the sequence.
				h.Ingest(100, telem.NewSeriesV(25.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(out.Get(200).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
				).To(Equal([]uint8{1}))
			},
		)
	})

	Describe("Chained Routing", func() {
		It(
			"Should route through a processing function with correct intermediate outputs",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":    {types.F64(), 100},
					"alarm_out": {types.F64(), 200},
				})
				h := newRuntimeHarness(ctx, demuxSource+`
				func amplify{} (signal f64) f64 {
				    return signal * 2.0
				}

				sensor -> demux{threshold=50.0} -> {
				    high: 2.0 -> amplify{} -> alarm_out
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Float64T},
				)
				defer h.Close(ctx)

				h.Ingest(100, telem.NewSeriesV(80.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				demuxHigh := h.Output("demux_0", 0)
				Expect(
					telem.UnmarshalSeries[float64](demuxHigh),
				).To(Equal([]float64{80.0}))

				// amplify computes from the entry's constant, not the routed
				// value: 2.0 * 2.0.
				amplifyResult := h.Output("amplify_0", 0)
				Expect(
					telem.UnmarshalSeries[float64](amplifyResult),
				).To(Equal([]float64{4.0}))
				Expect(h.OutputTime("amplify_0", 0).Len()).To(Equal(int64(1)))

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(
					telem.UnmarshalSeries[float64](out.Get(200).Series[0]),
				).To(Equal([]float64{4.0}))
			},
		)

		It(
			"Should not propagate to chained function when branch receives no data",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":    {types.F64(), 100},
					"alarm_out": {types.F64(), 200},
				})
				h := newRuntimeHarness(ctx, demuxSource+`
				func amplify{} (signal f64) f64 {
				    return signal * 2.0
				}

				sensor -> demux{threshold=50.0} -> {
				    high: 2.0 -> amplify{} -> alarm_out
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Float64T},
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
			},
		)

		It(
			"Should fire the chained function once per multi-sample trigger batch",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"sensor":    {types.F64(), 100},
					"alarm_out": {types.F64(), 200},
				})
				h := newRuntimeHarness(ctx, demuxSource+`
				func amplify{} (signal f64) f64 {
				    return signal * 2.0
				}

				sensor -> demux{threshold=50.0} -> {
				    high: 2.0 -> amplify{} -> alarm_out
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Float64T},
				)
				defer h.Close(ctx)

				h.Ingest(100, telem.NewSeriesV(80.0, 30.0, 90.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				demuxHigh := h.Output("demux_0", 0)
				Expect(
					telem.UnmarshalSeries[float64](demuxHigh),
				).To(Equal([]float64{80.0, 90.0}))
				Expect(h.OutputTime("demux_0", 0).Len()).To(Equal(int64(2)))

				// The entry's constant fires once for the batch, so amplify
				// emits a single sample.
				amplifyResult := h.Output("amplify_0", 0)
				Expect(
					telem.UnmarshalSeries[float64](amplifyResult),
				).To(Equal([]float64{4.0}))
				Expect(h.OutputTime("amplify_0", 0).Len()).To(Equal(int64(1)))

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(
					telem.UnmarshalSeries[float64](out.Get(200).Series[0]),
				).To(Equal([]float64{4.0}))
			},
		)

		It(
			"Should re-fire a chained constant on every upstream trigger with monotonically increasing timestamps",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"trig": {types.U8(), 100},
					"log":  {types.U8(), 200},
				})
				h := newRuntimeHarness(ctx, `trig -> 1 -> log`, resolver,
					channels.Digest{Key: 100, DataType: telem.Uint8T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				const fires = 4
				timestamps := make([]telem.TimeStamp, 0, fires)
				for i := range fires {
					h.Ingest(100, telem.NewSeriesV[uint8](1))
					h.Tick(ctx, telem.TimeSpan(i+1)*telem.Millisecond)
					h.channelState.ClearReads()
					out, _ := h.Flush()
					Expect(out.Get(200).Series).ToNot(BeEmpty(),
						"log should be written on every upstream trigger (fire %d)", i)
					timestamps = append(timestamps,
						telem.ValueAt[telem.TimeStamp](h.OutputTime("const_0", 0), 0))
				}
				for i := 1; i < len(timestamps); i++ {
					Expect(timestamps[i]).To(BeNumerically(">", timestamps[i-1]),
						"constant timestamp at fire %d should be strictly greater than fire %d", i, i-1)
				}
			},
		)
	})

	Describe("Routing to Sequences", func() {
		It(
			"Should activate a stage that writes a constant on the activation tick",
			func(ctx SpecContext) {
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
				    high: true => alarm
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
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
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
				).To(Equal([]uint8{1}))

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
			},
		)

		It(
			"Should not activate the sequence across multiple ticks when the branch never fires",
			func(ctx SpecContext) {
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
				    high: true => alarm
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				for i := range 3 {
					h.Ingest(100, telem.NewSeriesV(25.0))
					h.Tick(ctx, telem.TimeSpan(i+1)*telem.Millisecond)
					h.channelState.ClearReads()

					_, changed := h.Flush()
					Expect(
						changed,
					).To(BeFalse(), "tick %d should not produce writes", i+1)
				}
			},
		)

		It(
			"Should only activate the sequence whose routing branch fires",
			func(ctx SpecContext) {
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
				    above: true => open_valve,
				    below: true => log_event
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
					channels.Digest{Key: 300, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				// Tick 1: above threshold. Only open_valve activates.
				h.Ingest(100, telem.NewSeriesV(150.0))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(out.Get(200).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
				).To(Equal([]uint8{1}))
				Expect(out.Get(300).Series).To(BeEmpty())

				// Tick 2: below threshold. log_event activates. open_valve
				// constant already fired so it doesn't re-emit.
				h.Ingest(100, telem.NewSeriesV(50.0))
				h.Tick(ctx, 2*telem.Millisecond)
				h.channelState.ClearReads()

				out2, changed2 := h.Flush()
				Expect(changed2).To(BeTrue())
				Expect(out2.Get(200).Series).To(BeEmpty())
				Expect(out2.Get(300).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out2.Get(300).Series[0]),
				).To(Equal([]uint8{1}))
			},
		)

		It(
			"Should activate a multi-stage sequence and transition between stages",
			func(ctx SpecContext) {
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

				func check_pressure(p f32) bool {
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
				    high: true => pressurize
				}`, resolver,
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 101, DataType: telem.Float32T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
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
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
				).To(Equal([]uint8{1}))

				// Tick 2: pressure rises above 100. check_pressure returns truthy,
				// firing the one-shot transition to the hold stage. The hold stage's
				// constant writes 0 to vlv_cmd.
				h.Ingest(101, telem.NewSeriesV[float32](150.0))
				h.Tick(ctx, 2*telem.Millisecond)
				h.channelState.ClearReads()

				out2, changed2 := h.Flush()
				Expect(changed2).To(BeTrue())
				Expect(
					telem.UnmarshalSeries[uint8](out2.Get(200).Series[0]),
				).To(Equal([]uint8{0}))

				// Tick 3: no new data. Hold stage constant already fired.
				// No further writes.
				h.Tick(ctx, 3*telem.Millisecond)

				_, changed3 := h.Flush()
				Expect(changed3).To(BeFalse())
			},
		)
	})

	Describe("Routing with select{}", func() {
		It(
			"Should use select to route a boolean channel into different sequence stages",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"flag":     {types.Bool(), 100},
					"open_cmd": {types.U8(), 200},
					"shut_cmd": {types.U8(), 300},
				})
				h := newRuntimeHarness(ctx, `
				flag -> select{} -> {
				    true: true => open_valve,
				    false: true => shut_valve
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
					channels.Digest{Key: 100, DataType: telem.BooleanT},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
					channels.Digest{Key: 300, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				// Tick 1: flag=1 (truthy). select routes to "true" output,
				// activating open_valve.
				h.Ingest(100, telem.NewSeriesV[bool](true))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				selectTrue := h.Output("select_0", 0)
				Expect(selectTrue.Len()).To(Equal(int64(1)))
				selectFalse := h.Output("select_0", 1)
				Expect(selectFalse.Len()).To(Equal(int64(0)))

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(out.Get(200).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
				).To(Equal([]uint8{1}))
				Expect(out.Get(300).Series).To(BeEmpty())

				// Tick 2: flag=0 (falsy). select routes to "false" output,
				// activating shut_valve.
				h.Ingest(100, telem.NewSeriesV[bool](false))
				h.Tick(ctx, 2*telem.Millisecond)
				h.channelState.ClearReads()

				selectTrue2 := h.Output("select_0", 0)
				Expect(selectTrue2.Len()).To(Equal(int64(0)))
				selectFalse2 := h.Output("select_0", 1)
				Expect(selectFalse2.Len()).To(Equal(int64(1)))

				out2, changed2 := h.Flush()
				Expect(changed2).To(BeTrue())
				Expect(out2.Get(200).Series).To(BeEmpty())
				Expect(out2.Get(300).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out2.Get(300).Series[0]),
				).To(Equal([]uint8{1}))
			},
		)
	})

	Describe("Routing with select{}", func() {
		It("Should use select to route a boolean channel", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"flag":     {types.Bool(), 100},
				"open_cmd": {types.U8(), 200},
				"shut_cmd": {types.U8(), 300},
			})
			h := newRuntimeHarness(ctx, `
				flag -> select{} -> {
					true: true => open_valve,
					false: true => shut_valve
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
				}
			`, resolver,
				channels.Digest{Key: 100, DataType: telem.BooleanT},
				channels.Digest{Key: 200, DataType: telem.Uint8T},
				channels.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[bool](true))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			selectTrue := h.Output("select_0", 0)
			Expect(selectTrue.Len()).To(Equal(int64(1)))
			selectFalse := h.Output("select_0", 1)
			Expect(selectFalse.Len()).To(Equal(int64(0)))

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(
				telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
			).To(Equal([]uint8{1}))
			Expect(out.Get(300).Series).To(BeEmpty())
		})

		It("Should write the entry's explicit value per branch", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"flag":    {types.Bool(), 100},
				"vlv_cmd": {types.Bool(), 200},
			})
			h := newRuntimeHarness(ctx, `
				flag -> select{} -> {
					true: true -> vlv_cmd,
					false: false -> vlv_cmd
				}
			`, resolver,
				channels.Digest{Key: 100, DataType: telem.BooleanT},
				channels.Digest{Key: 200, DataType: telem.BooleanT},
			)
			defer h.Close(ctx)

			// True branch: the entry writes its own literal true.
			h.Ingest(100, telem.NewSeriesV[bool](true))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			out, changed := h.Flush()
			Expect(changed).To(BeTrue())
			Expect(out.Get(200).Series).To(HaveLen(1))
			Expect(
				telem.UnmarshalSeries[bool](out.Get(200).Series[0]),
			).To(Equal([]bool{true}))

			// False branch: the entry writes literal false, not the 1-valued
			// pulse the select node emits to gate the branch.
			h.Ingest(100, telem.NewSeriesV[bool](false))
			h.Tick(ctx, 2*telem.Millisecond)
			h.channelState.ClearReads()

			out2, changed2 := h.Flush()
			Expect(changed2).To(BeTrue())
			Expect(out2.Get(200).Series).To(HaveLen(1))
			Expect(
				telem.UnmarshalSeries[bool](out2.Get(200).Series[0]),
			).To(Equal([]bool{false}))
		})

		It(
			"Should run an inline body entry without an upstream flow",
			func(ctx SpecContext) {
				resolver := channelSymbols(map[string]channelDef{
					"flag":      {types.Bool(), 100},
					"stage_out": {types.U8(), 200},
					"seq_out":   {types.U8(), 300},
				})
				h := newRuntimeHarness(ctx, `
				flag -> select{} -> {
				    true: stage { 1 -> stage_out },
				    false: sequence { 1 -> seq_out }
				}
			`, resolver,
					channels.Digest{Key: 100, DataType: telem.BooleanT},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
					channels.Digest{Key: 300, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				// True branch activates the inline stage; its body fires on
				// activation with no upstream flow in the entry.
				h.Ingest(100, telem.NewSeriesV[bool](true))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()

				out, changed := h.Flush()
				Expect(changed).To(BeTrue())
				Expect(out.Get(200).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
				).To(Equal([]uint8{1}))
				Expect(out.Get(300).Series).To(BeEmpty())

				// False branch activates the inline sequence the same way.
				h.Ingest(100, telem.NewSeriesV[bool](false))
				h.Tick(ctx, 2*telem.Millisecond)
				h.channelState.ClearReads()

				out2, changed2 := h.Flush()
				Expect(changed2).To(BeTrue())
				Expect(out2.Get(200).Series).To(BeEmpty())
				Expect(out2.Get(300).Series).To(HaveLen(1))
				Expect(
					telem.UnmarshalSeries[uint8](out2.Get(300).Series[0]),
				).To(Equal([]uint8{1}))
			},
		)
	})

	Describe("Routing to Stages", func() {
		It(
			"Should compile and execute a routing table that targets a stage within the same sequence",
			func(ctx SpecContext) {
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
				            high: true => pressurize,
				        }
				    }
				    stage pressurize {
				        1 -> vlv_cmd
				    }
				}`, resolver,
					channels.Digest{Key: 50, DataType: telem.Uint8T},
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Uint8T},
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
				Expect(
					telem.UnmarshalSeries[uint8](out.Get(200).Series[0]),
				).To(Equal([]uint8{1}))
			},
		)
	})
})
