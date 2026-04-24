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
		for range 5 {
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

		It("Transitions into an anonymous nested sequence step after the preceding flow step", func(ctx SpecContext) {
			// Regression: analyzeSequence stamps the nested scope's Key with
			// an AutoName (seq_N) for anonymous nested sequences, but
			// collectStepKeys and autoWireTransition reference it by the
			// outer's step key (step_N). Without the nested-scope Key
			// override in the step iteration, the transition target lookup
			// misses and the sequence stalls at step 0 forever.
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"a":         {types.U8(), 101},
				"b":         {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    1 -> a
				    sequence {
				        1 -> b
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"outer's first write must fire on trigger")
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"nested anonymous sequence must activate once outer's first step transitions")
		})

		It("Anonymous top-level sequence auto-starts and sequentially runs a nested anonymous valve-timing sub-sequence", func(ctx SpecContext) {
			// The outer is anonymous → LivenessAlways → root cascade activates
			// it at boot (no trigger channel needed). The outer is sequential,
			// so its nested sub-sequence is a step member — activated by step
			// machinery (not cascade) when the outer's wait{2s} fires.
			//
			// Expected timeline (all absolute elapsed):
			//     t = 0..2s    outer holds at step 0 (wait{2s}); no writes.
			//     t ≈ 2s       outer → step 1 (inner activates); press = 1.
			//     t ≈ 2.25s    inner's first wait{250ms} fires; press = 0.
			//     t ≈ 2.5s     inner's second wait{250ms} fires; vent = 1.
			//     t ≈ 2.75s    inner's third wait{250ms} fires; vent = 0.
			resolver := channelSymbols(map[string]channelDef{
				"press_vlv_cmd": {types.U8(), 101},
				"vent_vlv_cmd":  {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence {
				    wait{2s}
				    sequence {
				        1 -> press_vlv_cmd
				        wait{250ms}
				        0 -> press_vlv_cmd
				        wait{250ms}
				        1 -> vent_vlv_cmd
				        wait{250ms}
				        0 -> vent_vlv_cmd
				    }
				}`, resolver,
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			// Prime the scheduler. Outer auto-starts; step 0 (wait{2s}) begins
			// tracking from this tick.
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"press must not fire while outer is holding at wait{2s}")
			Expect(out.Get(102).Series).To(BeEmpty(),
				"vent must not fire while outer is holding at wait{2s}")

			// Still inside outer's wait{2s} — elapsed = 1s, threshold 2s.
			advance(h, ctx, 1*telem.Second)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"press must not fire at t=1s (outer wait{2s} still blocking)")
			Expect(out.Get(102).Series).To(BeEmpty(),
				"vent must not fire at t=1s")

			// At t ≈ 2.01s outer's wait{2s} fires; step machinery activates
			// the nested sequence; inner step 0 runs press = 1.
			advance(h, ctx, 2*telem.Second+10*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"press must open once the outer's 2s wait elapses and step machinery activates the inner sequence")
			Expect(out.Get(102).Series).To(BeEmpty(),
				"vent must not fire yet — inner has only reached its first write")

			// At t ≈ 2.27s inner's first wait{250ms} fires; press = 0.
			advance(h, ctx, 2*telem.Second+270*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(0)),
				"press must close after inner's first wait{250ms} elapses")
			Expect(out.Get(102).Series).To(BeEmpty(),
				"vent must still be untouched at t≈2.27s")

			// At t ≈ 2.53s inner's second wait{250ms} fires; vent = 1.
			advance(h, ctx, 2*telem.Second+530*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"press must not be re-written after it closed")
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"vent must open after inner's second wait{250ms} elapses")

			// At t ≈ 2.79s inner's third wait{250ms} fires; vent = 0.
			advance(h, ctx, 2*telem.Second+790*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(0)),
				"vent must close after inner's third wait{250ms} elapses")
		})

		It("Anonymous outer auto-starts, transitions from a gated stage into a named nested sequence via => next", func(ctx SpecContext) {
			// The outer is anonymous → LivenessAlways → root cascade activates
			// it at boot. It has two sequential steps:
			//   step 0: stage cat { wait{2s} => next }
			//   step 1: sequence puff { wait{2s}; 1 -> ox_mpv_cmd }
			//
			// Stage body = parallel reactive flows, so cat's wait{2s} gates
			// the => next transition. Sequence body = sequential steps, so
			// puff's wait{2s} blocks before the write. Even though puff is
			// named (normally LivenessGated), as a step of the parent it's
			// activated by step machinery — no explicit => puff is needed.
			//
			// Expected timeline:
			//     t = 0..2s    cat holds at wait{2s}; ox_mpv_cmd untouched.
			//     t ≈ 2s       cat's wait fires, => next advances to puff;
			//                  puff's wait{2s} begins. Still no write.
			//     t ≈ 4s       puff's wait fires; 1 -> ox_mpv_cmd.
			resolver := channelSymbols(map[string]channelDef{
				"ox_mpv_cmd": {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence {
				    stage cat {
				        wait{2s} => next
				    }
				    sequence puff {
				        wait{2s}
				        1 -> ox_mpv_cmd
				    }
				}`, resolver,
				channel.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			// Prime the scheduler. Outer auto-starts; cat's wait{2s} begins.
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"ox_mpv_cmd must not fire while cat is holding at wait{2s}")

			// Still inside cat's wait{2s}.
			advance(h, ctx, 1*telem.Second)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"ox_mpv_cmd must not fire at t=1s (cat still blocking)")

			// At t ≈ 2.01s cat's wait fires, => next activates puff, puff's
			// own wait{2s} begins — the write must not fire yet.
			advance(h, ctx, 2*telem.Second+10*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"ox_mpv_cmd must not fire when puff activates — its own wait{2s} gates the write")

			// Halfway through puff's wait.
			advance(h, ctx, 3*telem.Second)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"ox_mpv_cmd must not fire at t≈3s (puff's wait{2s} still blocking)")

			// At t ≈ 4.02s puff's wait fires; 1 -> ox_mpv_cmd.
			advance(h, ctx, 4*telem.Second+20*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"ox_mpv_cmd must open once puff's 2s wait elapses after cat transitions")

			// Sequence has completed; no further writes on subsequent ticks.
			advance(h, ctx, 5*telem.Second)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"ox_mpv_cmd must not be rewritten after the sequence completes")
		})

		It("Resolves => X to an enclosing sequence's member across intermediate stages", func(ctx SpecContext) {
			// `=> after` fires from inside an anonymous nested sequence inside
			// stage cat — two structural layers (stage + nested seq) between
			// the firing flow and the target. The target lives in the outer
			// anonymous sequence's memberKeys, so the stack walk finds it
			// there and the transition lives on the outer frame. When the
			// inner wait fires, the outer advances step 0 (cat) → step 1
			// (after); cat freezes, after runs.
			resolver := channelSymbols(map[string]channelDef{
				"x": {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence {
				    stage cat {
				        sequence {
				            wait{1s} => after
				        }
				    }
				    sequence after {
				        1 -> x
				    }
				}`, resolver,
				channel.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"x must not be written while cat holds at wait{1s}")

			advance(h, ctx, 500*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"x must still be untouched mid-wait")

			advance(h, ctx, 1*telem.Second+10*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"after the wait fires, outer must advance to `after` and write 1")
		})

		It("Resolves => b through a nested sequence to a sibling stage", func(ctx SpecContext) {
			// `=> b` from inside a nested sequence inside stage a must find
			// b as a member of main (the outer frame) and advance main from
			// stage a to stage b.
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"x":         {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage a {
				        sequence {
				            wait{500ms} => b
				        }
				    }
				    stage b {
				        1 -> x
				    }
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(),
				"x must not be written while a holds at wait{500ms}")

			advance(h, ctx, 600*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"main must advance from a to b and write 1")
		})

		It("Applies shadowing: innermost sequence's member wins over outer same-name", func(ctx SpecContext) {
			// Both `inner` and the outer anonymous sequence contain a member
			// named `target`. From inside stage `a` (a step of inner),
			// `=> target` must resolve to inner's target, not outer's. The
			// shadowing rule is lexical — closer wrapping sequence wins —
			// so adding a same-named sibling in an outer scope cannot
			// silently steal a jump that used to resolve locally.
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"x_inner":   {types.U8(), 101},
				"x_outer":   {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence {
				    sequence inner {
				        stage a {
				            start_cmd => target
				        }
				        stage target {
				            1 -> x_inner
				        }
				    }
				    sequence target {
				        1 -> x_outer
				    }
				}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"inner's target must run and write to x_inner")
			Expect(out.Get(102).Series).To(BeEmpty(),
				"outer's target must be shadowed by inner's target and never run")
		})

		It("Preserves top-level activation for => other across sibling sequences", func(ctx SpecContext) {
			// Regression guard: `=> other` from inside main should still
			// activate root-level `other` via the cross-scope activation
			// path. `other` is not in any enclosing frame's memberKeys
			// (main's memberKeys are [step_0], not [other]), so the stack
			// walk misses and we fall through to root activation.
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"trigger":   {types.U8(), 101},
				"x":         {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    trigger => other
				}
				sequence other {
				    1 -> x
				}
				start_cmd => main`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"=> other must activate other and write 1 to x")
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
