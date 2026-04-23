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

// Behavioral coverage for sequence/stage execution. Each test compiles a
// minimal Arc program through the full text → IR → runtime pipeline and
// asserts observable behavior via channel writes.
var _ = Describe("Sequence", func() {
	// lastU8 returns the final u8 value written to a channel in a flushed
	// frame. Fails if the channel was not written.
	lastU8 := func(fr telem.Frame[uint32], key uint32) uint8 {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[uint8](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	// lastF32 returns the final f32 value written to a channel.
	lastF32 := func(fr telem.Frame[uint32], key uint32) float32 {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[float32](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	// lastString returns the final string value written to a channel.
	lastString := func(fr telem.Frame[uint32], key uint32) string {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[string](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	// trigger ingests a u8=1 onto the given channel and ticks the scheduler
	// long enough for the on-channel-read → entry → step cascade to settle.
	trigger := func(h *runtimeHarness, ctx SpecContext, key uint32) {
		h.Ingest(key, telem.NewSeriesV[uint8](1))
		for i := 0; i < 5; i++ {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
	}

	// advance ticks the scheduler with the given elapsed time.
	advance := func(h *runtimeHarness, ctx SpecContext, elapsed telem.TimeSpan) {
		h.Tick(ctx, elapsed)
		h.channelState.ClearReads()
	}

	Describe("Sequential execution", func() {
		It("Executes writes in declaration order, gated by wait", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"valve_cmd": {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    1 -> valve_cmd
				    wait{500ms}
				    0 -> valve_cmd
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			advance(h, ctx, 600*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
		})

		It("Cascades consecutive writes within a single tick", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"a":         {types.U8(), 101},
				"b":         {types.U8(), 102},
				"c":         {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    1 -> a
				    1 -> b
				    1 -> c
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))
			Expect(lastU8(out, 102)).To(Equal(uint8(1)))
			Expect(lastU8(out, 103)).To(Equal(uint8(1)))
		})

		It("Blocks at a bare expression gate until truthy", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"press_cmd": {types.U8(), 101},
				"pressure":  {types.F32(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    1 -> press_cmd
				    pressure > 50
				    0 -> press_cmd
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			h.Ingest(102, telem.NewSeriesV[float32](10))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(), "press_cmd should not be re-written while gate is falsy")

			h.Ingest(102, telem.NewSeriesV[float32](75))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
		})
	})

	Describe("Stage transitions", func() {
		It("Transitions to next stage when comparison becomes truthy", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"press_cmd": {types.U8(), 101},
				"pressure":  {types.F32(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage on {
				        1 -> press_cmd
				        pressure > 50 => off
				    }
				    stage off {
				        0 -> press_cmd
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			h.Ingest(102, telem.NewSeriesV[float32](75))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
		})

		It("Transitions to next stage after wait timeout", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"toggle_cmd": {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage on {
				        1 -> toggle_cmd
				        wait{500ms} => off
				    }
				    stage off {
				        0 -> toggle_cmd
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			advance(h, ctx, 600*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
		})

		It("Jumps to a named stage via => name from a sibling stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"normal_cmd":    {types.U8(), 101},
				"emergency_cmd": {types.U8(), 102},
				"pressure":      {types.F32(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage normal {
				        1 -> normal_cmd
				        pressure > 100 => emergency
				    }
				    stage emergency {
				        0 -> normal_cmd
				        1 -> emergency_cmd
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			h.Ingest(103, telem.NewSeriesV[float32](150))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
			Expect(lastU8(out, 102)).To(Equal(uint8(1)))
		})

		It("Resolves multi-stage sequences through a chain of comparison transitions", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"press_vlv_cmd": {types.U8(), 101},
				"vent_vlv_cmd":  {types.U8(), 102},
				"pressure":      {types.F32(), 103},
			})
			h := newRuntimeHarness(ctx, `
				HIGH f32 := 25
				LOW f32 := 5

				sequence main {
				    stage press {
				        1 -> press_vlv_cmd
				        pressure > HIGH => vent
				    }
				    stage vent {
				        0 -> press_vlv_cmd
				        1 -> vent_vlv_cmd
				        pressure < LOW => done
				    }
				    stage done {
				        0 -> vent_vlv_cmd
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			h.Ingest(103, telem.NewSeriesV[float32](50))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
			Expect(lastU8(out, 102)).To(Equal(uint8(1)))

			h.Ingest(103, telem.NewSeriesV[float32](2))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(0)))
		})
	})

	Describe("Composition", func() {
		It("Inline stage in sequence resumes after => next", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"ox_cmd":    {types.U8(), 101},
				"pressure":  {types.F32(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    1 -> ox_cmd
				    stage {
				        pressure < 15 => next
				    }
				    0 -> ox_cmd
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			h.Ingest(102, telem.NewSeriesV[float32](100))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty())

			h.Ingest(102, telem.NewSeriesV[float32](10))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
		})

		It("Inline sequence in stage runs alongside reactive flows", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"ox_cmd":    {types.U8(), 101},
				"vent_cmd":  {types.U8(), 102},
				"pressure":  {types.F32(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage fire {
				        sequence {
				            1 -> ox_cmd
				        }
				        pressure < 15 => exit
				    }
				    stage exit {
				        0 -> ox_cmd
				        1 -> vent_cmd
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"inline sub-sequence's first write should fire on stage entry")
			h.channelState.ClearReads()

			h.Ingest(103, telem.NewSeriesV[float32](10))
			h.Tick(ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"reactive exit transition should fire alongside the sub-sequence")
		})

		It("Anonymous stages in a sequence address steps by position", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"a":         {types.U8(), 101},
				"b":         {types.F32(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage {
				        1 -> a
				        b > 5 => next
				    }
				    stage {
				        0 -> a
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			h.Ingest(102, telem.NewSeriesV[float32](10))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
		})
	})

	Describe("Reactive flows", func() {
		It("Top-level interval drives a function call repeatedly", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"out": {types.F32(), 100},
			})
			h := newRuntimeHarness(ctx, `
				func emit() {
				    out = 7
				}
				interval{50ms} -> emit{}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			advance(h, ctx, 60*telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastF32(out, 100)).To(BeNumerically("~", 7.0, 0.001))
		})

		It("Channel-driven function executes when the source channel updates", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"out":    {types.F32(), 101},
			})
			h := newRuntimeHarness(ctx, `
				func dbl(v f32) f32 {
				    return v * 2
				}
				sensor -> dbl{} -> out`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[float32](21))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastF32(out, 101)).To(BeNumerically("~", 42.0, 0.001))
		})
	})

	Describe("Channel writes", func() {
		It("Writes a global constant to a channel", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":    {types.U8(), 100},
				"const_output": {types.F32(), 101},
			})
			h := newRuntimeHarness(ctx, `
				SOME_CONST f32 := 42.0

				sequence main {
				    SOME_CONST => const_output
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastF32(out, 101)).To(BeNumerically("~", 42.0, 0.001))
		})

		It("Writes a computed arithmetic expression to a channel", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":    {types.U8(), 100},
				"const_output": {types.F32(), 101},
			})
			h := newRuntimeHarness(ctx, `
				SOME_CONST f32 := -49.5

				sequence main {
				    SOME_CONST * 2 => const_output
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastF32(out, 101)).To(BeNumerically("~", -99.0, 0.001))
		})

		It("Writes a string literal to a string channel", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"log":       {types.String(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage a {
				        "hello" -> log
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("hello"))
		})
	})

	Describe("Function semantics", func() {
		It("Stateful variable persists across calls", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"trigger":   {types.U8(), 101},
				"count":     {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				func bump() {
				    n u8 $= 0
				    n = n + 1
				    count = n
				}
				trigger -> bump{}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)))

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(2)))

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(3)))
		})

		It("Resolves forward-declared function references through nested calls", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"out":    {types.F32(), 101},
			})
			// Functions are defined in scrambled order (1, 3, 2) to verify
			// the analyzer resolves cross-function references regardless of
			// declaration order.
			h := newRuntimeHarness(ctx, `
				func nested_1() {
				    nested_2(sensor)
				}

				func nested_3(val f32) {
				    out = val
				}

				func nested_2(val f32) {
				    nested_3(val)
				}

				interval{20ms} -> nested_1{}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[float32](3.14))
			advance(h, ctx, 25*telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastF32(out, 101)).To(BeNumerically("~", 3.14, 0.001))
		})

		It("Function with string config writes the configured value", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"log":       {types.String(), 101},
			})
			h := newRuntimeHarness(ctx, `
				func event_log{msg str} () {
				    log = msg
				}

				sequence main {
				    stage a {
				        event_log{"pressurizing"}
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("pressurizing"))
		})
	})

	Describe("Signal-triggered control", func() {
		It("Cycles through stages in response to repeated signal channel writes", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"stop_cmd":  {types.U8(), 101},
				"log":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence ctrl {
				    stage start {
				        "start" -> log
				        stop_cmd => stop
				    }
				    stage stop {
				        "stop" -> log
				    }
				}
				start_cmd => ctrl`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("start"))

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastString(out, 102)).To(Equal("stop"))
		})
	})

	Describe("Compound flow conditions", func() {
		It("Routes a transition only when a compound boolean condition is satisfied", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"heater_cmd": {types.U8(), 101},
				"temp_a":     {types.F32(), 102},
				"temp_b":     {types.F32(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage on {
				        1 -> heater_cmd
				(temp_a > 90 and temp_b > 90) => off
				    }
				    stage off {
				        0 -> heater_cmd
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Float32T},
				channel.Digest{Key: 103, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))

			// Only one channel above threshold — should not transition.
			h.Ingest(102, telem.NewSeriesV[float32](100))
			h.Ingest(103, telem.NewSeriesV[float32](50))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty())

			// Both above threshold — should transition.
			h.Ingest(102, telem.NewSeriesV[float32](100))
			h.Ingest(103, telem.NewSeriesV[float32](100))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
		})
	})

	// Re-entry covers stage deactivation and re-activation within a running
	// sequence. Stages that transition away and are later re-entered via a
	// named transition start fresh: the constant writes resume, any reactive
	// flows re-arm, and (per spec) stateful state resets.
	Describe("Re-entry", func() {
		It("Re-activates a stage when a sibling transitions back via => name", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"go_b":      {types.U8(), 101},
				"go_a":      {types.U8(), 102},
				"a_out":     {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage a {
				        1 -> a_out
				        go_b => b
				    }
				    stage b {
				        0 -> a_out
				        go_a => a
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(out.Get(103).Series).ToNot(BeEmpty(), "stage a should write a_out on first activation")
			Expect(lastU8(out, 103)).To(Equal(uint8(1)))

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(103).Series).ToNot(BeEmpty(), "stage b should write a_out=0 on transition from a")
			Expect(lastU8(out, 103)).To(Equal(uint8(0)))

			h.Ingest(102, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(103).Series).ToNot(BeEmpty(), "stage a should re-activate and re-write a_out=1")
			Expect(lastU8(out, 103)).To(Equal(uint8(1)))
		})

		// wait{} countdown restarts when its enclosing stage is re-entered
		// via a => name transition from a sibling stage. Threshold math:
		//   BaseInterval = 500ms (only timer in the program)
		//   tolerance    = BaseInterval / 2 = 250ms
		//   fire when    = elapsed - startTime >= duration - tolerance = 250ms
		// The probe enters a, detours away before 250ms, re-enters a, and
		// advances 200ms of fresh time. If wait reset on re-entry, its
		// startTime tracks the re-entry elapsed and wait has NOT fired. If
		// wait did not reset, its startTime is stale and elapsed-startTime
		// already exceeds 250ms, so wait fires immediately on re-entry.
		It("wait{} countdown restarts when its stage is re-entered", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"abort_cmd":  {types.U8(), 101},
				"resume_cmd": {types.U8(), 102},
				"heartbeat":  {types.U8(), 103},
				"done_out":   {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage a {
				        1 -> heartbeat
				        wait{500ms} => done
				        abort_cmd => parked
				    }
				    stage parked {
				        resume_cmd => a
				    }
				    stage done {
				        1 -> done_out
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Uint8T},
				channel.Digest{Key: 104, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			var elapsed telem.TimeSpan = 0
			tickTo := func(t telem.TimeSpan) {
				elapsed = t
				h.Tick(ctx, elapsed)
				h.channelState.ClearReads()
			}

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			tickTo(telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)), "stage a should activate and write heartbeat")
			Expect(out.Get(104).Series).To(BeEmpty(), "done_out must not be written yet")

			tickTo(100 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(104).Series).To(BeEmpty(), "at 100ms wait must not have fired (below 250ms threshold)")

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			tickTo(101 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(103).Series).To(BeEmpty(), "after abort, stage a is inactive so heartbeat stops")
			Expect(out.Get(104).Series).To(BeEmpty(), "done_out must not fire during detour")

			h.Ingest(102, telem.NewSeriesV[uint8](1))
			tickTo(102 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)), "re-entering stage a should resume the heartbeat")
			Expect(out.Get(104).Series).To(BeEmpty(), "done_out must not fire on re-entry tick")

			tickTo(300 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(104).Series).To(BeEmpty(),
				"at 300ms cumulative (198ms post-re-entry), wait must NOT have fired; "+
					"if it did, wait did not reset on re-entry (startTime stuck at 1ms → elapsed-startTime=299ms > 250ms threshold)")

			tickTo(500 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(1)),
				"at 500ms cumulative (398ms post-re-entry), wait should have fired after its reset cycle")
		})

		// interval{} cadence restarts when its enclosing stage is re-entered.
		// Threshold math:
		//   BaseInterval = 100ms (only timer in the program)
		//   tolerance    = BaseInterval / 2 = 50ms
		//   fire when    = elapsed - lastFired >= period - tolerance = 50ms
		//   lastFired    = -period initially, so first tick fires immediately
		// The probe fires interval once at elapsed=50ms (lastFired becomes
		// 50ms), detours to parked within 10ms, and re-enters at elapsed=60ms
		// — only 10ms past the last fire, well below the 50ms threshold.
		//   If interval reset on re-entry (lastFired = -period): elapsed
		//   - lastFired = 160ms, fires immediately on the re-entry tick.
		//   If interval did NOT reset: lastFired = 50ms (stale), elapsed
		//   - lastFired = 10ms < 50ms, no fire.
		// The assertion on the re-entry tick distinguishes the two cases.
		It("interval{} cadence restarts when its stage is re-entered", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"abort_cmd":  {types.U8(), 101},
				"resume_cmd": {types.U8(), 102},
				"pulse":      {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage a {
				        interval{100ms} -> pulse
				        abort_cmd => parked
				    }
				    stage parked {
				        resume_cmd => a
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			var elapsed telem.TimeSpan = 0
			tickTo := func(t telem.TimeSpan) {
				elapsed = t
				h.Tick(ctx, elapsed)
				h.channelState.ClearReads()
			}

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			tickTo(50 * telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"interval should fire on the first tick after stage a activates (lastFired=-period)")

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			tickTo(55 * telem.Millisecond)
			_, _ = h.Flush()

			h.Ingest(102, telem.NewSeriesV[uint8](1))
			tickTo(60 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"interval should fire immediately on stage re-entry (reset restores lastFired=-period); "+
					"if it did not fire, lastFired is stale (10ms since last fire < 50ms threshold)")

			tickTo(70 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(103).Series).To(BeEmpty(),
				"interval should NOT fire at elapsed=70ms (only 10ms since the re-entry fire)")

			tickTo(115 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"interval should fire at elapsed=115ms (55ms since re-entry fire, past the 50ms threshold)")
		})

		// Channel transitions in a newly-activated stage only fire on writes
		// that arrive AFTER activation. Data buffered before the stage
		// becomes active (high-water mark is ahead of that data) is gated
		// out, so a stale safety command sitting in the buffer from a prior
		// run cannot trigger the transition on entry.
		It("Stale channel writes do not fire transitions on stage activation", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"abort_cmd": {types.U8(), 101},
				"running":   {types.U8(), 102},
				"halted":    {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage run {
				        1 -> running
				        abort_cmd => halt
				    }
				    stage halt {
				        0 -> running
				        1 -> halted
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(101, telem.NewSeriesV[uint8](1))

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"stage run should activate and write running=1")
			Expect(out.Get(103).Series).To(BeEmpty(),
				"halt must NOT activate from the stale pre-activation abort_cmd=1 "+
					"(if it did, the abort_cmd read was not gated to post-activation writes)")

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(0)),
				"halt should now activate and stop running")
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"halt should write halted=1 on the fresh abort_cmd write")
		})

		// When two => transitions in the same stage go truthy on the same
		// tick, the one declared first in the stage body wins. This is how
		// users encode safety priority: put the abort condition before the
		// success condition, and an abort beats a simultaneous success.
		// The reverse entry (success declared first) confirms the winner
		// follows declaration order, not some fixed heuristic.
		DescribeTable("Transition priority follows declaration order",
			func(ctx SpecContext, source string, winKey, loseKey uint32) {
				resolver := channelSymbols(map[string]channelDef{
					"start_cmd":      {types.U8(), 100},
					"abort_signal":   {types.U8(), 101},
					"success_signal": {types.U8(), 102},
					"valve":          {types.U8(), 103},
					"aborted":        {types.U8(), 104},
					"succeeded":      {types.U8(), 105},
				})
				h := newRuntimeHarness(ctx, source, resolver,
					channel.Digest{Key: 100, DataType: telem.Uint8T},
					channel.Digest{Key: 101, DataType: telem.Uint8T},
					channel.Digest{Key: 102, DataType: telem.Uint8T},
					channel.Digest{Key: 103, DataType: telem.Uint8T},
					channel.Digest{Key: 104, DataType: telem.Uint8T},
					channel.Digest{Key: 105, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				trigger(h, ctx, 100)
				out, _ := h.Flush()
				Expect(lastU8(out, 103)).To(Equal(uint8(1)),
					"stage run should activate and hold valve open")
				Expect(out.Get(104).Series).To(BeEmpty())
				Expect(out.Get(105).Series).To(BeEmpty())

				h.Ingest(101, telem.NewSeriesV[uint8](1))
				h.Ingest(102, telem.NewSeriesV[uint8](1))
				advance(h, ctx, telem.Millisecond)
				out, _ = h.Flush()

				Expect(lastU8(out, winKey)).To(Equal(uint8(1)),
					"declaration-order-first transition should win when both are truthy same tick")
				Expect(out.Get(loseKey).Series).To(BeEmpty(),
					"the loser's target stage must not have activated at all")
				Expect(lastU8(out, 103)).To(Equal(uint8(0)),
					"whichever target activated, run deactivated so valve drops to 0")
			},
			Entry("abort declared first beats success", `
				sequence main {
				    stage run {
				        1 -> valve
				        abort_signal => abort
				        success_signal => done
				    }
				    stage abort {
				        0 -> valve
				        1 -> aborted
				    }
				    stage done {
				        0 -> valve
				        1 -> succeeded
				    }
				}
				start_cmd => main`, uint32(104), uint32(105)),
			Entry("success declared first beats abort", `
				sequence main {
				    stage run {
				        1 -> valve
				        success_signal => done
				        abort_signal => abort
				    }
				    stage abort {
				        0 -> valve
				        1 -> aborted
				    }
				    stage done {
				        0 -> valve
				        1 -> succeeded
				    }
				}
				start_cmd => main`, uint32(105), uint32(104)),
		)
	})
})
