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
				start_cmd => main
			`, resolver,
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
				start_cmd => main
			`, resolver,
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
				start_cmd => main
			`, resolver,
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
						1 -> press_cmd,
						pressure > 50 => off
					}
					stage off {
						0 -> press_cmd,
					}
				}
				start_cmd => main
			`, resolver,
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
						1 -> toggle_cmd,
						wait{500ms} => off,
					}
					stage off {
						0 -> toggle_cmd,
					}
				}
				start_cmd => main
			`, resolver,
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
						1 -> normal_cmd,
						pressure > 100 => emergency
					}
					stage emergency {
						0 -> normal_cmd,
						1 -> emergency_cmd
					}
				}
				start_cmd => main
			`, resolver,
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
						1 -> press_vlv_cmd,
						pressure > HIGH => vent
					}
					stage vent {
						0 -> press_vlv_cmd,
						1 -> vent_vlv_cmd,
						pressure < LOW => done
					}
					stage done {
						0 -> vent_vlv_cmd,
					}
				}
				start_cmd => main
			`, resolver,
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
						pressure < 15 => next,
					}
					0 -> ox_cmd
				}
				start_cmd => main
			`, resolver,
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

		// Pending: anonymous inline sequences inside stages currently fail symbol
		// resolution at analyze time ("could not find symbol matching parser
		// rule"). RFC 0034 §3.2 requires the analyzer to register inline
		// sub-sequences in the enclosing scope before text-builder runs.
		PIt("Inline sequence in stage progresses through its steps in parallel with reactive flows", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"ox_cmd":    {types.U8(), 101},
				"fuel_cmd":  {types.U8(), 102},
				"pressure":  {types.F32(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
					stage fire {
						sequence {
							1 -> ox_cmd
							wait{200ms}
							1 -> fuel_cmd
						},
						pressure < 15 => exit,
					}
					stage exit {
						0 -> ox_cmd,
						0 -> fuel_cmd,
					}
				}
				start_cmd => main
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))
			Expect(out.Get(102).Series).To(BeEmpty(), "fuel_cmd should not fire before wait elapses")

			advance(h, ctx, 300*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)))

			h.Ingest(103, telem.NewSeriesV[float32](10))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)))
			Expect(lastU8(out, 102)).To(Equal(uint8(0)))
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
						1 -> a,
						b > 5 => next,
					}
					stage {
						0 -> a,
					}
				}
				start_cmd => main
			`, resolver,
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
				interval{50ms} -> emit{}
			`, resolver,
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
				sensor -> dbl{} -> out
			`, resolver,
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
				start_cmd => main
			`, resolver,
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
				start_cmd => main
			`, resolver,
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
						"hello" -> log,
					}
				}
				start_cmd => main
			`, resolver,
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
				trigger -> bump{}
			`, resolver,
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

				interval{20ms} -> nested_1{}
			`, resolver,
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
						event_log{"pressurizing"},
					}
				}
				start_cmd => main
			`, resolver,
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
						"start" -> log,
						stop_cmd => stop
					}
					stage stop {
						"stop" -> log,
					}
				}
				start_cmd => ctrl
			`, resolver,
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
						1 -> heater_cmd,
						(temp_a > 90 and temp_b > 90) => off,
					}
					stage off {
						0 -> heater_cmd,
					}
				}
				start_cmd => main
			`, resolver,
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
})
