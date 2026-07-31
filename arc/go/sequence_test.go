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

	// countOf returns how many samples on a channel equal want.
	countOf := func(fr telem.Frame[uint32], key uint32, want string) int {
		n := 0
		for _, s := range fr.Get(key).Series {
			for _, v := range telem.UnmarshalSeries[string](s) {
				if v == want {
					n++
				}
			}
		}
		return n
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

	// A reassignment takes effect when its stage runs, even reached out of source
	// order or after a skipped stage; guards against the old source-order chain.
	Describe("Reactive re-expression", func() {
		src := `
    sequence main {
        rx f32 := rx_src + 1
        stage rx_entry {
            rx -> rx_out
            e_to_b >= 1 => rx_b
            e_to_c >= 1 => rx_c
        }
        stage rx_a {
            rx = rx_src + 10
            rx -> rx_out
            a_to_d >= 1 => rx_d
        }
        stage rx_b {
            rx = rx_src + 20
            rx -> rx_out
        }
        stage rx_c {
            rx = rx_src + 30
            rx -> rx_out
            c_to_a >= 1 => rx_a
        }
        stage rx_d {
            rx = rx_src + 40
            rx -> rx_out
        }
    }
    start_cmd => main`
		newH := func(ctx SpecContext) *runtimeHarness {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"rx_src":    {types.F32(), 101},
				"rx_out":    {types.F32(), 102},
				"e_to_c":    {types.U8(), 103},
				"e_to_b":    {types.U8(), 104},
				"c_to_a":    {types.U8(), 105},
				"a_to_d":    {types.U8(), 106},
			})
			return newRuntimeHarness(ctx, src, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Float32T},
				channels.Digest{Key: 102, DataType: telem.Float32T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
				channels.Digest{Key: 105, DataType: telem.Uint8T},
				channels.Digest{Key: 106, DataType: telem.Uint8T},
			)
		}
		pushSrc := func(h *runtimeHarness, ctx SpecContext, v float32) {
			h.Ingest(101, telem.NewSeriesV[float32](v))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
		}

		It("jumps to a re-expression reached by skipping earlier stages", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 100)
			trigger(h, ctx, 103) // entry => rx_c, skipping rx_a and rx_b
			pushSrc(h, ctx, 2)
			out, _ := h.Flush()
			Expect(lastF32(out, 102)).To(Equal(float32(32))) // rx_c: rx_src + 30
		})

		It("re-expresses to an earlier-source stage after a later one", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 100)
			trigger(h, ctx, 103) // entry => rx_c
			pushSrc(h, ctx, 2)
			out, _ := h.Flush()
			Expect(lastF32(out, 102)).To(Equal(float32(32)))
			trigger(h, ctx, 105) // rx_c => rx_a (earlier in source order)
			pushSrc(h, ctx, 3)
			out, _ = h.Flush()
			Expect(lastF32(out, 102)).To(Equal(float32(13))) // rx_a: rx_src + 10
		})

		It("does not emit on a rebind until the next source value", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 100)
			pushSrc(h, ctx, 2)
			out, _ := h.Flush()
			Expect(lastF32(out, 102)).To(Equal(float32(3))) // entry: rx_src + 1
			trigger(h, ctx, 104)                            // entry => rx_b rebinds rx
			out, _ = h.Flush()
			Expect(out.Get(102).Series).To(BeEmpty(), "a rebind alone must not emit")
			pushSrc(h, ctx, 5)
			out, _ = h.Flush()
			Expect(lastF32(out, 102)).To(Equal(float32(25))) // rx_b: rx_src + 20
		})

		It("re-emits an unchanged recompute on a fresh sample", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 100)
			pushSrc(h, ctx, 2)
			out, _ := h.Flush()
			Expect(lastF32(out, 102)).To(Equal(float32(3)))
			pushSrc(h, ctx, 2)
			out, _ = h.Flush()
			Expect(lastF32(out, 102)).To(Equal(float32(3)),
				"a fresh sample fires even when the value is unchanged")
		})
	})

	Describe("Channel alias re-expression", func() {
		src := `
    sequence main {
        ra := ch_init
        stage r_entry {
            ra -> ra_out
            e_to_c >= 1 => r_c
        }
        stage r_a {
            ra -> ra_out
        }
        stage r_c {
            ra = chc
            ra -> ra_out
            c_to_a >= 1 => r_a
        }
    }
    start_cmd => main`
		newH := func(ctx SpecContext) *runtimeHarness {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"ch_init":   {types.U8(), 201},
				"chc":       {types.U8(), 202},
				"ra_out":    {types.U8(), 102},
				"e_to_c":    {types.U8(), 103},
				"c_to_a":    {types.U8(), 104},
			})
			return newRuntimeHarness(ctx, src, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
			)
		}
		pushC := func(h *runtimeHarness, ctx SpecContext, v uint8) {
			h.Ingest(202, telem.NewSeriesV[uint8](v))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
		}

		It("reads a rebind made in a stage reached by skipping earlier stages", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 100)
			trigger(h, ctx, 103) // r_entry => r_c, skipping r_a
			pushC(h, ctx, 42)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(42))) // r_c: ra = chc
		})

		It("reads a rebind from an earlier-source stage reached after it", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 100)
			trigger(h, ctx, 103) // r_entry => r_c
			pushC(h, ctx, 42)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(42)))
			trigger(h, ctx, 104) // r_c => r_a, compiled before the rebind
			pushC(h, ctx, 99)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(99))) // r_a reads chc's latest, not stale ch_init
		})

		It("writes through an alias from an earlier-source stage reached after a rebind", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"ch_init":   {types.U8(), 201},
				"chc":       {types.U8(), 202},
				"e_to_c":    {types.U8(), 103},
				"c_to_a":    {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        ra := ch_init
        stage w_entry {
            e_to_c >= 1 => w_c
        }
        stage w_a {
            u8(9) -> ra
        }
        stage w_c {
            ra = chc
            c_to_a >= 1 => w_a
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			trigger(h, ctx, 103) // w_entry => w_c (ra = chc)
			trigger(h, ctx, 104) // w_c => w_a, compiled before the rebind
			out, _ := h.Flush()
			Expect(lastU8(out, 202)).To(Equal(uint8(9))) // w_a writes the current binding chc, not ch_init
		})

		It("reads the latest of two rebinds made across stages before a jump back", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"ch_init":   {types.U8(), 201},
				"chc":       {types.U8(), 202},
				"chd":       {types.U8(), 203},
				"ra_out":    {types.U8(), 102},
				"e_to_c":    {types.U8(), 103},
				"c_to_d":    {types.U8(), 104},
				"d_to_a":    {types.U8(), 105},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        ra := ch_init
        stage r_entry {
            e_to_c >= 1 => r_c
        }
        stage r_a {
            ra -> ra_out
        }
        stage r_c {
            ra = chc
            c_to_d >= 1 => r_d
        }
        stage r_d {
            ra = chd
            d_to_a >= 1 => r_a
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
				channels.Digest{Key: 203, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
				channels.Digest{Key: 105, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			trigger(h, ctx, 103) // r_entry => r_c (ra = chc)
			trigger(h, ctx, 104) // r_c => r_d (ra = chd)
			trigger(h, ctx, 105) // r_d => r_a, earliest in source
			h.Ingest(203, telem.NewSeriesV[uint8](7))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(7))) // r_a reads chd, the last rebind, not chc or ch_init
		})
	})

	Describe("Negative literal variable initializers", func() {
		chanFor := func(expected any) (types.Type, telem.DataType) {
			switch expected.(type) {
			case int8:
				return types.I8(), telem.Int8T
			case int16:
				return types.I16(), telem.Int16T
			case int32:
				return types.I32(), telem.Int32T
			case int64:
				return types.I64(), telem.Int64T
			case float32:
				return types.F32(), telem.Float32T
			default:
				return types.F64(), telem.Float64T
			}
		}
		assertLast := func(s telem.Series, expected any) {
			switch e := expected.(type) {
			case int8:
				Expect(telem.UnmarshalSeries[int8](s)).To(ContainElement(e))
			case int16:
				Expect(telem.UnmarshalSeries[int16](s)).To(ContainElement(e))
			case int32:
				Expect(telem.UnmarshalSeries[int32](s)).To(ContainElement(e))
			case int64:
				Expect(telem.UnmarshalSeries[int64](s)).To(ContainElement(e))
			case float32:
				Expect(telem.UnmarshalSeries[float32](s)).To(ContainElement(e))
			case float64:
				Expect(telem.UnmarshalSeries[float64](s)).To(ContainElement(e))
			}
		}

		DescribeTable("Reads back the declared negative value across data types",
			func(ctx SpecContext, arcType, lit string, expected any) {
				vt, dt := chanFor(expected)
				resolver := channelSymbols(map[string]channelDef{
					"start_cmd": {types.U8(), 100},
					"out":       {vt, 101},
				})
				h := newRuntimeHarness(ctx, `
					sequence main {
					    a `+arcType+` := `+lit+`
					    a -> out
					}
					start_cmd => main`, resolver,
					channels.Digest{Key: 100, DataType: telem.Uint8T},
					channels.Digest{Key: 101, DataType: dt},
				)
				defer h.Close(ctx)
				trigger(h, ctx, 100)
				out, _ := h.Flush()
				s := out.Get(101).Series
				Expect(s).ToNot(BeEmpty(), "var channel not written")
				assertLast(s[len(s)-1], expected)
			},
			Entry("i8", "i8", "-5", int8(-5)),
			Entry("i16", "i16", "-5", int16(-5)),
			Entry("i32", "i32", "-5", int32(-5)),
			Entry("i64", "i64", "-5", int64(-5)),
			Entry("i8 type minimum", "i8", "-128", int8(-128)),
			Entry("i16 type minimum", "i16", "-32768", int16(-32768)),
			Entry("f32", "f32", "-2.5", float32(-2.5)),
			Entry("f64", "f64", "-2.5", float64(-2.5)),
		)

		DescribeTable("Initializes the constant so a stage reading it never surfaces the zero value",
			func(ctx SpecContext, lit string, want int64) {
				resolver := channelSymbols(map[string]channelDef{
					"start_cmd": {types.U8(), 100},
					"out":       {types.I64(), 101},
				})
				h := newRuntimeHarness(ctx, `
					sequence main {
					    a i64 := `+lit+`
					    stage s {
					        a -> out
					    }
					}
					start_cmd => main`, resolver,
					channels.Digest{Key: 100, DataType: telem.Uint8T},
					channels.Digest{Key: 101, DataType: telem.Int64T},
				)
				defer h.Close(ctx)
				trigger(h, ctx, 100)
				out, _ := h.Flush()
				s := out.Get(101).Series
				Expect(s).ToNot(BeEmpty(), "var channel not written")
				for _, ser := range s {
					for _, v := range telem.UnmarshalSeries[int64](ser) {
						Expect(v).To(Equal(want), "initialized constant must not glitch through its zero value")
					}
				}
			},
			Entry("negated literal", "-5", int64(-5)),
		)

		DescribeTable("Reassigns a literal variable to a negative value",
			func(ctx SpecContext, lit string, want int64) {
				resolver := channelSymbols(map[string]channelDef{
					"start_cmd": {types.U8(), 100},
					"out":       {types.I64(), 101},
				})
				h := newRuntimeHarness(ctx, `
					sequence main {
					    a i64 := 0
					    stage s {
					        a = `+lit+`
					        a -> out
					    }
					}
					start_cmd => main`, resolver,
					channels.Digest{Key: 100, DataType: telem.Uint8T},
					channels.Digest{Key: 101, DataType: telem.Int64T},
				)
				defer h.Close(ctx)
				trigger(h, ctx, 100)
				out, _ := h.Flush()
				s := out.Get(101).Series
				Expect(s).ToNot(BeEmpty(), "var channel not written")
				Expect(telem.UnmarshalSeries[int64](s[len(s)-1])).To(ContainElement(want))
			},
			Entry("negated literal", "-100", int64(-100)),
		)
	})

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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))
			Expect(lastU8(out, 102)).To(Equal(uint8(1)))
			Expect(lastU8(out, 103)).To(Equal(uint8(1)))
		})

		It("Advances past steps whose terminal node outputs a string", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"done":      {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				func make_key() str {
				    return "ox_alarm"
				}
				sequence main {
				    make_key{}
				    make_key{}
				    make_key{}
				    1 -> done
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Float32T},
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

		It("Re-runs an interval-driven func flow when re-entering start from yield", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"stop_cmd":      {types.U8(), 101},
				"press_vlv_cmd": {types.U8(), 102},
				"press_pt":      {types.F32(), 103},
			})
			h := newRuntimeHarness(ctx, `
				func bang{ sensor chan f32, set_point f32 } () u8 {
				    state u8 $= 0
				    if sensor < set_point {
				        state = 1
				    } else {
				        state = 0
				    }
				    return state
				}
				sequence main {
				    stage start {
				        interval{200ms} -> bang{sensor=press_pt, set_point=50} -> press_vlv_cmd
				        stop_cmd => yield
				    }
				    stage yield {
				        start_cmd => start
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			var now telem.TimeSpan
			adv := func(d telem.TimeSpan) {
				now += d
				h.Tick(ctx, now)
				h.channelState.ClearReads()
			}
			fire := func(key uint32) {
				h.Ingest(key, telem.NewSeriesV[uint8](1))
				adv(telem.Millisecond)
			}

			h.Ingest(103, telem.NewSeriesV[float32](0))
			fire(100)
			adv(250 * telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"bang should drive the valve open while pressure is low")

			fire(101)

			h.Ingest(103, telem.NewSeriesV[float32](0))
			fire(100)
			adv(250 * telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"interval-driven func flow must re-run and drive the valve open after re-entry")
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
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
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
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
				channels.Digest{Key: 101, DataType: telem.Uint8T},
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
				channels.Digest{Key: 101, DataType: telem.Uint8T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
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

	Describe("Inline routing case body lifetime", func() {
		// mk builds `trigger -> select{} -> { true: <body> }` inside a stage,
		// so <body> is entered once when trigger fires select's true output.
		mk := func(ctx SpecContext, body string) *runtimeHarness {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"trigger":   {types.U8(), 101},
				"cpu":       {types.F64(), 102},
				"out":       {types.U8(), 103},
			})
			prog := "\nsequence WU {\n stage watcher {\n trigger -> select{} -> {\n true: " + body + "\n }\n }\n}\nstart_cmd => WU"
			return newRuntimeHarness(ctx, prog, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Float64T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
		}
		enter := func(h *runtimeHarness, ctx SpecContext) {
			h.Ingest(100, telem.NewSeriesV[uint8](1)) // activate WU -> watcher
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(102, telem.NewSeriesV[float64](5)) // cpu > 0
			h.Ingest(101, telem.NewSeriesV[uint8](1))   // trigger -> select true -> enter body
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		// firesPerUpdate counts total out samples across entry + 4 later cpu
		// updates. A once-and-done body writes once; a stage fires each update.
		firesPerUpdate := func(ctx SpecContext, body string) int {
			h := mk(ctx, body)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(102, telem.NewSeriesV[float64](5))
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
			out, _ := h.Flush()
			n := len(out.Get(103).Series)
			for range 4 {
				h.Ingest(102, telem.NewSeriesV[float64](5))
				h.Tick(ctx, telem.Millisecond)
				out, _ := h.Flush()
				n += len(out.Get(103).Series)
				h.channelState.ClearReads()
			}
			return n
		}

		It("Re-triggering an active inline stage does not stack instances", func(ctx SpecContext) {
			h := mk(ctx, "stage { cpu > 0 => out }")
			defer h.Close(ctx)
			enter(h, ctx)
			for range 9 { // spam trigger -> re-activate the already-active stage
				h.Ingest(101, telem.NewSeriesV[uint8](1))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			h.Ingest(102, telem.NewSeriesV[float64](5)) // one cpu update
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			samples := int64(0)
			for _, s := range out.Get(103).Series {
				samples += s.Len()
			}
			// One instance only: a cpu update writes once, not once per trigger.
			Expect(samples).To(Equal(int64(1)))
		})

		DescribeTable("bare inline flow and sequence fire once; stage fires per update",
			func(ctx SpecContext, body string, want int) {
				Expect(firesPerUpdate(ctx, body)).To(Equal(want))
			},
			Entry("bare inline flow fires once and done", "cpu > 0 => out", 1),
			Entry("inline sequence fires once and done", "sequence { cpu > 0 => out }", 1),
			Entry("inline stage fires on every cpu update", "stage { cpu > 0 => out }", 4),
		)
	})

	Describe("Routing table => transitions", func() {
		// select{} routes flag to one of two sibling stages via `=>`.
		selectProg := `
			sequence WU {
			    stage router {
			        flag -> select{} -> {
			            true: 1 => high_stage,
			            false: 1 => low_stage
			        }
			    }
			    stage high_stage { 1 -> high_out }
			    stage low_stage { 1 -> low_out }
			}
			start_cmd => WU`

		DescribeTable("select routes the => transition to the stage picked by the input",
			func(ctx SpecContext, flag uint8, hit, miss uint32) {
				resolver := channelSymbols(map[string]channelDef{
					"start_cmd": {types.U8(), 100},
					"flag":      {types.U8(), 101},
					"high_out":  {types.U8(), 102},
					"low_out":   {types.U8(), 103},
				})
				h := newRuntimeHarness(ctx, selectProg, resolver,
					channels.Digest{Key: 100, DataType: telem.Uint8T},
					channels.Digest{Key: 101, DataType: telem.Uint8T},
					channels.Digest{Key: 102, DataType: telem.Uint8T},
					channels.Digest{Key: 103, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				h.Ingest(100, telem.NewSeriesV[uint8](1)) // activate WU -> router
				h.Tick(ctx, telem.Millisecond)
				h.Ingest(101, telem.NewSeriesV[uint8](flag))
				h.Tick(ctx, telem.Millisecond)
				h.Tick(ctx, telem.Millisecond)
				out, _ := h.Flush()
				Expect(lastU8(out, hit)).To(Equal(uint8(1)))
				Expect(out.Get(miss).Series).To(BeEmpty())
			},
			Entry("flag 1 jumps to high_stage", uint8(1), uint32(102), uint32(103)),
			Entry("flag 0 jumps to low_stage", uint8(0), uint32(103), uint32(102)),
		)

		// A custom 3-output router jumps to one of three sibling stages.
		routerProg := `
			func route{} (v f64) (lo u8, mid u8, hi u8) {
			    if v < 10.0 {
			        lo = 1
			    } else if v < 20.0 {
			        mid = 1
			    } else {
			        hi = 1
			    }
			}
			sequence WU {
			    stage router {
			        signal -> route{} -> {
			            lo: 1 => lo_stage,
			            mid: 1 => mid_stage,
			            hi: 1 => hi_stage
			        }
			    }
			    stage lo_stage { 1 -> lo_out }
			    stage mid_stage { 1 -> mid_out }
			    stage hi_stage { 1 -> hi_out }
			}
			start_cmd => WU`

		DescribeTable("custom multi-output router jumps to the matching stage",
			func(ctx SpecContext, signal float64, hit uint32) {
				resolver := channelSymbols(map[string]channelDef{
					"start_cmd": {types.U8(), 100},
					"signal":    {types.F64(), 101},
					"lo_out":    {types.U8(), 102},
					"mid_out":   {types.U8(), 103},
					"hi_out":    {types.U8(), 104},
				})
				h := newRuntimeHarness(ctx, routerProg, resolver,
					channels.Digest{Key: 100, DataType: telem.Uint8T},
					channels.Digest{Key: 101, DataType: telem.Float64T},
					channels.Digest{Key: 102, DataType: telem.Uint8T},
					channels.Digest{Key: 103, DataType: telem.Uint8T},
					channels.Digest{Key: 104, DataType: telem.Uint8T},
				)
				defer h.Close(ctx)

				h.Ingest(100, telem.NewSeriesV[uint8](1)) // activate WU -> router
				h.Tick(ctx, telem.Millisecond)
				h.Ingest(101, telem.NewSeriesV[float64](signal))
				h.Tick(ctx, telem.Millisecond)
				h.Tick(ctx, telem.Millisecond)
				out, _ := h.Flush()
				Expect(lastU8(out, hit)).To(Equal(uint8(1)))
				for _, k := range []uint32{102, 103, 104} {
					if k != hit {
						Expect(out.Get(k).Series).To(BeEmpty())
					}
				}
			},
			Entry("signal < 10 jumps to lo_stage", 5.0, uint32(102)),
			Entry("signal in [10,20) jumps to mid_stage", 15.0, uint32(103)),
			Entry("signal >= 20 jumps to hi_stage", 25.0, uint32(104)),
		)
	})

	Describe("Inline routing case bodies", func() {
		It("Anonymous top-level stage dispatches into the inline branch matching the routed output", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"flag":      {types.U8(), 100},
				"true_out":  {types.U8(), 101},
				"false_out": {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    flag -> select{} -> {
				        true: stage { 1 -> true_out },
				        false: stage { 1 -> false_out }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"true branch inline must fire when flag is truthy")
			Expect(out.Get(102).Series).To(BeEmpty(),
				"false branch inline must not fire when flag is truthy")
		})

		It("Named gated top-level stage dispatches inline branch only after activation", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"flag":      {types.U8(), 101},
				"true_out":  {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				stage main {
				    flag -> select{} -> {
				        true: stage { 1 -> true_out }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(102).Series).To(BeEmpty(),
				"inline must not fire before main is activated")

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline must fire once main is activated and flag is truthy")
		})

		It("Inline sequence body in a top-level stage runs its sequential steps in order", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"flag":  {types.U8(), 100},
				"out_a": {types.U8(), 101},
				"out_b": {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    flag -> select{} -> {
				        true: sequence {
				            1 -> out_a
				            2 -> out_b
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"first sequential step of inline sequence must fire")
			Expect(lastU8(out, 102)).To(Equal(uint8(2)),
				"second sequential step of inline sequence must fire")
		})

		It("Inline routing directly in a sequence body dispatches without duplicate transitions", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"flag":      {types.U8(), 101},
				"true_out":  {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    flag -> select{} -> {
				        true: stage { 1 -> true_out }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline must fire once when routed directly from a sequence body")
		})

		It("Inline body transition to a sibling stage advances the enclosing sequence", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"flag":       {types.U8(), 101},
				"first_out":  {types.U8(), 102},
				"second_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage first_stage {
				        flag -> select{} -> {
				            true: stage {
				                1 -> first_out,
				                1 -> second_stage
				            }
				        }
				    }
				    stage second_stage {
				        1 -> second_out
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline first_out write must fire")
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"transition from inline body to sibling stage must advance main")
		})

		It("Inline body navigates to a named sibling stage as its sole write", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":   {types.U8(), 100},
				"flag":        {types.U8(), 101},
				"reached_out": {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage first {
				        flag -> select{} -> {
				            true: stage { 1 -> following_stage }
				        }
				    }
				    stage following_stage {
				        1 -> reached_out
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"following_stage must activate after inline body fires its cross-stage write")
		})

		It("Completion of inline routing does not auto-advance the enclosing sequence", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"flag":       {types.U8(), 101},
				"inline_out": {types.U8(), 102},
				"second_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage first {
				        flag -> select{} -> {
				            true: stage { 1 -> inline_out }
				        }
				    }
				    stage second {
				        1 -> second_out
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline body must fire its write")
			Expect(out.Get(103).Series).To(BeEmpty(),
				"main must remain on 'first' after inline routing completes; "+
					"'second' must not auto-advance")
		})

		It("Doubly-nested inline activation fires the inner body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"inner_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> select{} -> {
				            true: stage {
				                1 -> select{} -> {
				                    true: stage { 1 -> inner_out }
				                }
				            }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"doubly-nested inline body must activate when outer body fires its inner select")
		})

		It("Triply-nested inline activation fires the innermost body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"innermost_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> select{} -> {
				            true: stage {
				                1 -> select{} -> {
				                    true: stage {
				                        1 -> select{} -> {
				                            true: stage { 1 -> innermost_out }
				                        }
				                    }
				                }
				            }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"triply-nested inline body must activate via the chain of inline selects")
		})

		It("Quadruply-nested inline activation fires the innermost body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"innermost_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> select{} -> {
				            true: stage {
				                1 -> select{} -> {
				                    true: stage {
				                        1 -> select{} -> {
				                            true: stage {
				                                1 -> select{} -> {
				                                    true: stage { 1 -> innermost_out }
				                                }
				                            }
				                        }
				                    }
				                }
				            }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"quadruply-nested inline body must activate via the chain of inline selects")
		})

	})

	Describe("Inline flow target bodies", func() {
		It("Anonymous top-level stage fires its inline flow body when the source fires", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"flag":     {types.U8(), 100},
				"true_out": {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    flag -> stage { 1 -> true_out }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"inline flow body must fire when its source channel fires")
		})

		It("Named gated top-level stage fires its inline flow body only after activation", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"flag":      {types.U8(), 101},
				"true_out":  {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				stage main {
				    flag -> stage { 1 -> true_out }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(102).Series).To(BeEmpty(),
				"inline must not fire before main is activated")

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline must fire once main is activated and flag is truthy")
		})

		It("Inline sequence flow body runs its sequential steps in order", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"flag":  {types.U8(), 100},
				"out_a": {types.U8(), 101},
				"out_b": {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    flag -> sequence {
				        1 -> out_a
				        2 -> out_b
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"first sequential step of inline sequence must fire")
			Expect(lastU8(out, 102)).To(Equal(uint8(2)),
				"second sequential step of inline sequence must fire")
		})

		It("Inline flow body directly in a sequence body dispatches without duplicate transitions", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"flag":      {types.U8(), 101},
				"true_out":  {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    flag -> stage { 1 -> true_out }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline must fire once when routed directly from a sequence body")
		})

		It("Inline flow body transition to a sibling stage advances the enclosing sequence", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"flag":       {types.U8(), 101},
				"first_out":  {types.U8(), 102},
				"second_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage first_stage {
				        flag -> stage {
				            1 -> first_out,
				            1 -> second_stage
				        }
				    }
				    stage second_stage {
				        1 -> second_out
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline first_out write must fire")
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"transition from inline body to sibling stage must advance main")
		})

		It("Inline flow body navigates to a named sibling stage as its sole write", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":   {types.U8(), 100},
				"flag":        {types.U8(), 101},
				"reached_out": {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage first {
				        flag -> stage { 1 -> following_stage }
				    }
				    stage following_stage {
				        1 -> reached_out
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"following_stage must activate after inline body fires its cross-stage write")
		})

		It("Completion of an inline flow body does not auto-advance the enclosing sequence", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"flag":       {types.U8(), 101},
				"inline_out": {types.U8(), 102},
				"second_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage first {
				        flag -> stage { 1 -> inline_out }
				    }
				    stage second {
				        1 -> second_out
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Ingest(101, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline body must fire its write")
			Expect(out.Get(103).Series).To(BeEmpty(),
				"main must remain on 'first' after the inline body completes; "+
					"'second' must not auto-advance")
		})

		It("Doubly-nested inline flow body fires the inner body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"inner_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> stage {
				            1 -> stage { 1 -> inner_out }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"doubly-nested inline flow body must activate when the outer body fires")
		})

		It("Triply-nested inline flow body fires the innermost body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"innermost_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> stage {
				            1 -> stage {
				                1 -> stage { 1 -> innermost_out }
				            }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"triply-nested inline flow body must activate via the chain of inline bodies")
		})

		It("Quadruply-nested inline flow body fires the innermost body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"innermost_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> stage {
				            1 -> stage {
				                1 -> stage {
				                    1 -> stage { 1 -> innermost_out }
				                }
				            }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"quadruply-nested inline flow body must activate via the chain of inline bodies")
		})
	})

	Describe("Mixed inline dispatch nesting", func() {
		It("Flow body wrapping an inline routing body fires the inner body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"inner_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> stage {
				            1 -> select{} -> {
				                true: stage { 1 -> inner_out }
				            }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"inline routing body nested in a flow body must fire in the same cycle")
		})

		It("Routing body wrapping an inline flow body fires the inner body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"inner_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> select{} -> {
				            true: stage {
				                1 -> stage { 1 -> inner_out }
				            }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"inline flow body nested in a routing body must fire in the same cycle")
		})

		It("Alternating flow/routing/flow nesting fires the innermost body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"innermost_out": {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
				    stage hold {
				        1 -> stage {
				            1 -> select{} -> {
				                true: stage {
				                    1 -> stage { 1 -> innermost_out }
				                }
				            }
				        }
				    }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"innermost body must fire through alternating flow/routing/flow dispatch")
		})
	})

	Describe("Top-level inline flow bodies", func() {
		It("Module-scope inline stage flow body fires when its source fires", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trigger": {types.U8(), 100},
				"out":     {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				trigger -> stage { 1 -> out }`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"module-scope inline flow body must fire when its source channel fires")
		})

		It("Module-scope inline stage flow body fires on a single trigger edge", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trigger":    {types.U8(), 100},
				"direct_out": {types.U8(), 101},
				"stage_out":  {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				trigger -> direct_out
				trigger -> stage { 1 -> stage_out }`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()

			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"plain flow must fire on the triggering edge")
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"inline stage flow body must fire on the same cycle as the "+
					"triggering edge, not one cycle later")
		})

		It("Module-scope inline stage flow body fires every parallel write", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trigger": {types.U8(), 100},
				"out_a":   {types.U8(), 101},
				"out_b":   {types.U8(), 102},
				"out_c":   {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				trigger -> stage {
				    1 -> out_a
				    2 -> out_b
				    3 -> out_c
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"first parallel write of module-scope inline stage must fire")
			Expect(lastU8(out, 102)).To(Equal(uint8(2)),
				"second parallel write of module-scope inline stage must fire")
			Expect(lastU8(out, 103)).To(Equal(uint8(3)),
				"third parallel write of module-scope inline stage must fire")
		})

		It("Module-scope inline sequence flow body runs three steps in order", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trigger": {types.U8(), 100},
				"out_a":   {types.U8(), 101},
				"out_b":   {types.U8(), 102},
				"out_c":   {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				trigger -> sequence {
				    1 -> out_a
				    2 -> out_b
				    3 -> out_c
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"first step of module-scope inline sequence must fire")
			Expect(lastU8(out, 102)).To(Equal(uint8(2)),
				"second step of module-scope inline sequence must fire")
			Expect(lastU8(out, 103)).To(Equal(uint8(3)),
				"third step of module-scope inline sequence must fire")
		})

		It("Module-scope nested inline flow bodies fire the inner body in the same cycle", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trigger":   {types.U8(), 100},
				"inner_out": {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				trigger -> stage {
				    1 -> stage { 1 -> inner_out }
				}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"nested module-scope inline flow body must activate in the same cycle")
		})
	})

	Describe("Sequence-scope inline flow bodies", func() {
		It("Fires constant-gated inline bodies on stage entry and cascades", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out_a":     {types.U8(), 101},
				"out_b":     {types.U8(), 102},
				"out_c":     {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage s1 {
				        1 -> stage {
				            1 -> out_a
				            1 => s2
				        }
				    }
				    stage s2 {
				        1 -> sequence {
				            1 -> out_b
				            1 => s3
				        }
				    }
				    stage s3 {
				        1 -> out_c
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"first stage's constant-gated inline stage body must fire on entry")
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"second stage's constant-gated inline sequence body must fire on entry")
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"third stage must fire after the inline-body transitions cascade")
		})

		It("Cascades when the first stage's inline body is gated by an external edge", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out_a":     {types.U8(), 101},
				"out_b":     {types.U8(), 102},
				"out_c":     {types.U8(), 103},
				"ext":       {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage s1 {
				        ext -> stage {
				            1 -> out_a
				            1 => s2
				        }
				    }
				    stage s2 {
				        1 -> sequence {
				            1 -> out_b
				            1 => s3
				        }
				    }
				    stage s3 {
				        1 -> out_c
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			trigger(h, ctx, 104)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"externally-gated inline stage body must fire on its edge")
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"second stage's constant-gated inline body must fire on entry, "+
					"not wait for another external edge")
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"third stage must fire after the cascade completes")
		})

		It("Cascades constant-gated inline bodies within a single tick", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out_a":     {types.U8(), 101},
				"out_b":     {types.U8(), 102},
				"out_c":     {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    stage s1 {
				        1 -> stage {
				            1 -> out_a
				            1 => s2
				        }
				    }
				    stage s2 {
				        1 -> sequence {
				            1 -> out_b
				            1 => s3
				        }
				    }
				    stage s3 {
				        1 -> out_c
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)),
				"first inline body must fire in the triggering tick")
			Expect(lastU8(out, 102)).To(Equal(uint8(1)),
				"second stage's inline body must fire in the same tick, not the next")
			Expect(lastU8(out, 103)).To(Equal(uint8(1)),
				"third stage must fire in the same tick")
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
				channels.Digest{Key: 100, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Float32T},
				channels.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[float32](21))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastF32(out, 101)).To(BeNumerically("~", 42.0, 0.001))
		})
	})

	Describe("Channel writes", func() {
		It("Writes a top-level variable to a channel", func(ctx SpecContext) {
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastF32(out, 101)).To(BeNumerically("~", 42.0, 0.001))
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.StringT},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
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
				channels.Digest{Key: 100, DataType: telem.Float32T},
				channels.Digest{Key: 101, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.StringT},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.StringT},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Float32T},
				channels.Digest{Key: 103, DataType: telem.Float32T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
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

		It("Resets a function's stateful variable when its stage is re-entered", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"go_b":      {types.U8(), 101},
				"go_a":      {types.U8(), 102},
				"tick":      {types.U8(), 103},
				"count":     {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
				func bump(v u8) u8 {
				    n u8 $= 0
				    n = n + 1
				    return n
				}
				sequence main {
				    stage a {
				        tick -> bump{} -> count
				        go_b => b
				    }
				    stage b {
				        go_a => a
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(1)))

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(2)))

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()

			h.Ingest(102, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(1)), "stateful variable should reset when stage a is re-entered")
		})

		It("Holds a function's stateful variable while its stage stays active", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"tick":      {types.U8(), 101},
				"count":     {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				func bump(v u8) u8 {
				    n u8 $= 0
				    n = n + 1
				    return n
				}
				sequence main {
				    stage a {
				        tick -> bump{} -> count
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			for i := uint8(1); i <= 5; i++ {
				h.Ingest(101, telem.NewSeriesV[uint8](1))
				advance(h, ctx, telem.Millisecond)
				out, _ := h.Flush()
				Expect(lastU8(out, 102)).To(Equal(i), "stateful variable must accumulate while stage a stays active")
			}
		})

		It("Resets a function's stateful variable on every re-entry across repeated cycles", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"go_b":      {types.U8(), 101},
				"go_a":      {types.U8(), 102},
				"tick":      {types.U8(), 103},
				"count":     {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
				func bump(v u8) u8 {
				    n u8 $= 0
				    n = n + 1
				    return n
				}
				sequence main {
				    stage a {
				        tick -> bump{} -> count
				        go_b => b
				    }
				    stage b {
				        go_a => a
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)

			for cycle := range 4 {
				h.Ingest(103, telem.NewSeriesV[uint8](1))
				advance(h, ctx, telem.Millisecond)
				out, _ := h.Flush()
				Expect(lastU8(out, 104)).To(Equal(uint8(1)), "cycle %d: stateful variable should start from 1", cycle)

				h.Ingest(103, telem.NewSeriesV[uint8](1))
				advance(h, ctx, telem.Millisecond)
				out, _ = h.Flush()
				Expect(lastU8(out, 104)).To(Equal(uint8(2)), "cycle %d: stateful variable should accumulate within the stage", cycle)

				h.Ingest(101, telem.NewSeriesV[uint8](1))
				advance(h, ctx, telem.Millisecond)
				h.Flush()

				h.Ingest(102, telem.NewSeriesV[uint8](1))
				advance(h, ctx, telem.Millisecond)
				h.Flush()
			}
		})

		It("Resets every stateful variable in a function when its stage is re-entered", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"go_b":      {types.U8(), 101},
				"go_a":      {types.U8(), 102},
				"tick":      {types.U8(), 103},
				"sum_out":   {types.F32(), 104},
			})
			h := newRuntimeHarness(ctx, `
				func accumulate(v u8) f32 {
				    count u8 $= 0
				    total f32 $= 0.0
				    scale i64 $= 10
				    count = count + 1
				    total = total + f32(count) * f32(scale)
				    return total
				}
				sequence main {
				    stage a {
				        tick -> accumulate{} -> sum_out
				        go_b => b
				    }
				    stage b {
				        go_a => a
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Float32T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastF32(out, 104)).To(Equal(float32(10)))

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastF32(out, 104)).To(Equal(float32(30)))

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()
			h.Ingest(102, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastF32(out, 104)).To(Equal(float32(10)), "count and total must both reset when stage a is re-entered")
		})

		It("Resets a function's stateful string variable when its stage is re-entered", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"go_b":      {types.U8(), 101},
				"go_a":      {types.U8(), 102},
				"tick":      {types.U8(), 103},
				"log":       {types.String(), 104},
			})
			h := newRuntimeHarness(ctx, `
				func trail(v u8) str {
				    acc str $= "x"
				    acc = acc + "x"
				    return acc
				}
				sequence main {
				    stage a {
				        tick -> trail{} -> log
				        go_b => b
				    }
				    stage b {
				        go_a => a
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastString(out, 104)).To(Equal("xx"))

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastString(out, 104)).To(Equal("xxx"))

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()
			h.Ingest(102, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastString(out, 104)).To(Equal("xx"), "stateful string should reset when stage a is re-entered")
		})

		It("Scopes the reset to the re-entered stage's own call site", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"go_b":       {types.U8(), 101},
				"go_a":       {types.U8(), 102},
				"tick":       {types.U8(), 103},
				"staged_out": {types.U8(), 104},
				"free_out":   {types.U8(), 105},
			})
			h := newRuntimeHarness(ctx, `
				func bump(v u8) u8 {
				    n u8 $= 0
				    n = n + 1
				    return n
				}
				sequence main {
				    stage a {
				        tick -> bump{} -> staged_out
				        go_b => b
				    }
				    stage b {
				        go_a => a
				    }
				}
				tick -> bump{} -> free_out
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
				channels.Digest{Key: 105, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(1)))
			free1 := lastU8(out, 105)

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()
			h.Ingest(102, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(1)), "the staged call site should reset on re-entry")
			Expect(lastU8(out, 105)).To(BeNumerically(">", free1),
				"the root-scope call site is not inside the re-entered stage and must keep accumulating")
		})

		It("Resets a stateful variable in a nested sequence's stage on inner re-entry", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"go_inner2": {types.U8(), 101},
				"go_inner1": {types.U8(), 102},
				"tick":      {types.U8(), 103},
				"count":     {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
				func bump(v u8) u8 {
				    n u8 $= 0
				    n = n + 1
				    return n
				}
				sequence main {
				    stage outer {
				        sequence {
				            stage inner1 {
				                tick -> bump{} -> count
				                go_inner2 => inner2
				            }
				            stage inner2 {
				                go_inner1 => inner1
				            }
				        }
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(1)))

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(2)))

			h.Ingest(101, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()
			h.Ingest(102, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			h.Flush()

			h.Ingest(103, telem.NewSeriesV[uint8](1))
			advance(h, ctx, telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 104)).To(Equal(uint8(1)), "stateful variable should reset when the inner stage is re-entered")
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
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
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
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
					channels.Digest{Key: 100, DataType: telem.Uint8T},
					channels.Digest{Key: 101, DataType: telem.Uint8T},
					channels.Digest{Key: 102, DataType: telem.Uint8T},
					channels.Digest{Key: 103, DataType: telem.Uint8T},
					channels.Digest{Key: 104, DataType: telem.Uint8T},
					channels.Digest{Key: 105, DataType: telem.Uint8T},
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

	Describe("Stateful flow variables", func() {
		It("Folds an unwritten stateful's initial value into flow reads", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.U8(), 101},
				"out2":      {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    x u8 $= 5
				    stage s1 {
				        x -> out
				        x * 2 -> out2
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(5)))
			Expect(lastU8(out, 102)).To(Equal(uint8(10)))
		})

		It("Interpolates an unwritten stateful's initial value", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    x u8 $= 5
				    stage s1 {
				        `+`f"x: {x}"`+` -> out
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("x: 5"))
		})

		It("Persists a written stateful across stage re-entries", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.U8(), 101},
				"go2":       {types.U8(), 102},
				"go1":       {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    x u8 $= 0
				    stage s1 {
				        x = x + 1
				        x -> out
				        go2 => next
				    }
				    stage s2 {
				        go1 => s1
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))
			trigger(h, ctx, 102)
			trigger(h, ctx, 103)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(2)))
		})
	})

	Describe("Variable initialization on declaration", func() {
		It("Initializes a variable declared in a sequence body", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    my_var := "hello"
				    my_var -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("hello"))
		})

		It("Initializes a variable declared in a stage body", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    stage read {
				        my_var := "hello"
				        my_var -> out
				    }
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("hello"))
		})

		It("Reflects a flow write to an initialized variable", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    my_var := "hello"
				    "updated" -> my_var
				    my_var -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("updated"))
		})

		It("Shares a sequence-scoped variable with a nested stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    my_var := "initial"
				    stage write {
				        "updated" -> my_var
				        my_var -> out
				    }
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("updated"))
		})

		// Declarations are bindings, not steps: they must not block step advance.
		It("Advances past sequence-scoped declarations", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    my_var := "my_var"
				    1 -> out
				    other := "other"
				    2 -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(2)))
		})
	})

	Describe("Reactive variable reads", func() {
		It("Logs a variable once per change across stage re-activations", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{"log": {types.String(), 101}})
			h := newRuntimeHarness(ctx, `import time
				sequence main {
				    stage s1 {
				        counter $= 0
				        1 => counter + 1 => counter
				        str(counter) => log
				        time.wait{100ms} => next
				    }
				    stage s2 {
				        1 => s1
				    }
				}
				1 => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			// Two scheduler passes settle each wait→s2→s1 re-entry; the logged value
			// lags the increment by one activation, so activations log 0, 1, 2, ...
			step := func(now telem.TimeSpan) {
				h.Tick(ctx, now)
				h.channelState.ClearReads()
				h.Tick(ctx, now)
				h.channelState.ClearReads()
			}
			step(0)
			step(100 * telem.Millisecond)
			step(200 * telem.Millisecond)
			step(300 * telem.Millisecond)

			out, _ := h.Flush()
			var logged []string
			for _, ser := range out.Get(101).Series {
				logged = append(logged, telem.UnmarshalSeries[string](ser)...)
			}
			Expect(logged).To(Equal([]string{"0", "1", "2", "3", "4"}))
		})

		It("Re-emits repeated derivations of the same value", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"cpu": {types.U8(), 201},
				"out": {types.U8(), 202},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    x := cpu * 2
				    x -> out
				}`, resolver,
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			for range 4 {
				h.Ingest(201, telem.NewSeriesV[uint8](5))
			}
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			total := int64(0)
			for _, s := range out.Get(202).Series {
				total += s.Len()
			}
			Expect(total).To(Equal(int64(4)),
				"every fresh sample fires even when the value is unchanged")
			Expect(lastU8(out, 202)).To(Equal(uint8(10)))
		})

		It("Re-fires a derivation on every fresh sample", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"cpu": {types.U8(), 201},
				"out": {types.U8(), 202},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    x := cpu * 2
				    x -> out
				}`, resolver,
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			var got []uint8
			feed := func(v uint8) {
				h.Ingest(201, telem.NewSeriesV[uint8](v))
				for range 5 {
					advance(h, ctx, telem.Millisecond)
				}
				out, _ := h.Flush()
				for _, s := range out.Get(202).Series {
					got = append(got, telem.UnmarshalSeries[uint8](s)...)
				}
			}
			feed(5)
			feed(5)
			feed(7)
			feed(7)
			feed(5)
			Expect(got).To(Equal([]uint8{10, 10, 14, 14, 10}))
		})

		It("Re-emits repeated format-string derivations", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"cpu": {types.U8(), 201},
				"out": {types.String(), 202},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    x := "v: " + str(cpu)
				    x -> out
				}`, resolver,
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			var got []string
			feed := func(v uint8) {
				h.Ingest(201, telem.NewSeriesV[uint8](v))
				for range 5 {
					advance(h, ctx, telem.Millisecond)
				}
				out, _ := h.Flush()
				for _, s := range out.Get(202).Series {
					got = append(got, telem.UnmarshalSeries[string](s)...)
				}
			}
			feed(5)
			feed(5)
			feed(7)
			Expect(got).To(Equal([]string{"v: 5", "v: 5", "v: 7"}))
		})

		It("Derives a channel-read variable and reads it", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"cpu":  {types.U8(), 201},
				"out":  {types.U8(), 202},
				"out2": {types.U8(), 203},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    x := cpu * 2
				    x -> out
				    x + 1 -> out2
				}`, resolver,
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
				channels.Digest{Key: 203, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(201, telem.NewSeriesV[uint8](5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 202)).To(Equal(uint8(10)))
			Expect(lastU8(out, 203)).To(Equal(uint8(11)))

			h.Ingest(201, telem.NewSeriesV[uint8](7))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(lastU8(out, 202)).To(Equal(uint8(14)))
			Expect(lastU8(out, 203)).To(Equal(uint8(15)))
		})
	})

	Describe("Variable reassignment", func() {
		It("Reflects a reassignment to an initialized variable", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    my_var := "hello"
				    my_var = "updated"
				    my_var -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("updated"))
		})

		It("Reassigns a variable inside a stage body", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    my_var := "initial"
				    stage write {
				        my_var = "updated"
				        my_var -> out
				    }
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("updated"))
		})

		It("Reassigns a sequence-declared variable from inside a stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    my_var := "top"
				    stage write {
				        my_var = "updated"
				        my_var -> out
				    }
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("updated"))
		})

		It("Increments a variable via self-referential reassignment exactly once", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    counter u8 := 5
				    counter = counter + 1
				    counter -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(6)))
		})

		It("Redirects writes through an alias across a rebind", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out_a":     {types.U8(), 201},
				"out_b":     {types.U8(), 202},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    sink := out_a
				    u8(1) -> sink
				    sink = out_b
				    u8(2) -> sink
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 201)).To(Equal(uint8(1)))
			Expect(lastU8(out, 202)).To(Equal(uint8(2)))
		})

		It("Reads through an alias that follows a rebind", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"sensor_a":  {types.U8(), 201},
				"sensor_b":  {types.U8(), 202},
				"out":       {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    src := sensor_a
				    src = sensor_b
				    src -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(201, telem.NewSeriesV[uint8](3))
			h.Ingest(202, telem.NewSeriesV[uint8](7))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(7)))
		})

		It("Rebinds an alias inside a stage body", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"trig":      {types.U8(), 105},
				"out_a":     {types.U8(), 201},
				"out_b":     {types.U8(), 202},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    stage w {
				        sink := out_a
				        sink = out_b
				        trig -> sink
				    }
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(105, telem.NewSeriesV[uint8](2))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 202)).To(Equal(uint8(2)))
		})

		It("Walks multiple sequential read-alias rebinds to the final binding", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"sensor_a":  {types.U8(), 201},
				"sensor_b":  {types.U8(), 202},
				"sensor_c":  {types.U8(), 203},
				"out":       {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    src := sensor_a
				    src = sensor_b
				    src = sensor_c
				    src -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(201, telem.NewSeriesV[uint8](1))
			h.Ingest(202, telem.NewSeriesV[uint8](2))
			h.Ingest(203, telem.NewSeriesV[uint8](3))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(3)))
		})

		It("Walks multiple sequential write-alias rebinds to the final binding", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out_a":     {types.U8(), 201},
				"out_b":     {types.U8(), 202},
				"out_c":     {types.U8(), 203},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    sink := out_a
				    sink = out_b
				    sink = out_c
				    u8(9) -> sink
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 201, DataType: telem.Uint8T},
				channels.Digest{Key: 202, DataType: telem.Uint8T},
				channels.Digest{Key: 203, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 203)).To(Equal(uint8(9)))
			Expect(out.Get(201).Series).To(BeEmpty(), "original target must not be written")
			Expect(out.Get(202).Series).To(BeEmpty(), "intermediate target must not be written")
		})

		It("Rebinds an alias to the same channel without crashing", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"sensor":    {types.U8(), 201},
				"out":       {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    src := sensor
				    src = sensor
				    src -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(201, telem.NewSeriesV[uint8](5))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(5)))
		})

		It("Reads through an alias before its source has data without crashing", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"sensor":    {types.U8(), 201},
				"out":       {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    src := sensor
				    src -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			advance(h, ctx, telem.Millisecond)
			_, ok := h.Flush()
			Expect(ok).To(BeFalse(), "no source data means nothing should be emitted")
		})

		It("Writes through an alias bound to an unbacked channel without crashing", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"ghost":     {types.U8(), 299},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    sink := ghost
				    u8(1) -> sink
				}
				start_cmd => s`, resolver,
			)
			defer h.Close(ctx)

			Expect(func() { trigger(h, ctx, 100) }).ToNot(Panic())
		})

		It("Switches a channel-read variable's expression on re-expression", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_val":    {types.U8(), 200},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    r := in_val + u8(1)
				    r = in_val + u8(100)
				    r -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(200, telem.NewSeriesV[uint8](5))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(105)))
		})

		It("Advances through multiple re-expressions to the final feeder", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_val":    {types.U8(), 200},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    r := in_val + u8(1)
				    r = in_val + u8(10)
				    r = in_val + u8(100)
				    r -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(200, telem.NewSeriesV[uint8](5))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(105)))
		})

		It("Re-gates a reactive reader on a fresh input after re-expression", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_val":    {types.U8(), 200},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    r := in_val + u8(1)
				    r -> out
				    r = in_val + u8(100)
				    r -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			var got []uint8
			drain := func() {
				out, _ := h.Flush()
				for _, s := range out.Get(101).Series {
					got = append(got, telem.UnmarshalSeries[uint8](s)...)
				}
			}
			step := func(val uint8) {
				h.Ingest(200, telem.NewSeriesV[uint8](val))
				for range 5 {
					advance(h, ctx, telem.Millisecond)
				}
				drain()
			}

			trigger(h, ctx, 100)
			drain()
			step(10)  // first reader: in=10 -> r=11
			step(100) // second reader (post re-expr): in=100 -> r=200, must not surface stale 11
			Expect(got).To(Equal([]uint8{11, 200}))
		})

		It("Re-fires on an equal input value after re-expression", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_val":    {types.U8(), 200},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    r := in_val + u8(1)
				    r -> out
				    r = in_val + u8(100)
				    r -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(200, telem.NewSeriesV[uint8](2))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			h.Ingest(200, telem.NewSeriesV[uint8](2))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(102)))
		})

		It("Computes an expression over a rebound alias", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"sensor_a":  {types.U8(), 201},
				"sensor_b":  {types.U8(), 202},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    src := sensor_a
				    src = sensor_b
				    src + u8(1) -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(201, telem.NewSeriesV[uint8](3))
			h.Ingest(202, telem.NewSeriesV[uint8](7))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(8)))
		})

		It("Computes an expression mixing a static and a rebound alias", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"sensor_a":  {types.U8(), 201},
				"sensor_b":  {types.U8(), 202},
				"sensor_c":  {types.U8(), 203},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    a := sensor_a
				    b := sensor_b
				    b = sensor_c
				    a + b -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(201, telem.NewSeriesV[uint8](1))
			h.Ingest(203, telem.NewSeriesV[uint8](5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(6)))
		})

		It("Does not replay a derivation into a re-entered stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_val":    {types.U8(), 200},
				"out":       {types.U8(), 101},
				"go2":       {types.U8(), 102},
				"go1":       {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    r := in_val + u8(1)
				    stage s1 {
				        r -> out
				        go2 => next
				    }
				    stage s2 {
				        go1 => s1
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			var got []uint8
			drain := func() {
				out, _ := h.Flush()
				for _, s := range out.Get(101).Series {
					got = append(got, telem.UnmarshalSeries[uint8](s)...)
				}
			}
			trigger(h, ctx, 100)
			h.Ingest(200, telem.NewSeriesV[uint8](5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			drain()
			Expect(got).To(Equal([]uint8{6}))

			trigger(h, ctx, 102) // s1 -> s2
			trigger(h, ctx, 103) // s2 -> s1 re-entry
			drain()
			Expect(got).To(Equal([]uint8{6}), "re-entry must not replay the stale value")

			h.Ingest(200, telem.NewSeriesV[uint8](9))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			drain()
			Expect(got).To(Equal([]uint8{6, 10}))
		})

		It("Swallows a derivation value pending at the re-point", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_val":    {types.U8(), 200},
				"gate":      {types.U8(), 104},
				"out":       {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    r := in_val + u8(1)
				    u8(1) -> gate
				    r = in_val + u8(100)
				    r -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 104, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			// Data lands on the same tick the sequence enters, so the pre-rebind
			// derivation value is pending exactly when the re-point arrives.
			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Ingest(200, telem.NewSeriesV[uint8](5))
			for range 6 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(out.Get(101).Series).To(BeEmpty(), "a value predating the re-point must not fire")

			h.Ingest(200, telem.NewSeriesV[uint8](7))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(107)))
		})

		It("Reflects a stage reassignment on re-entering a reader stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.U8(), 101},
				"go2":       {types.U8(), 102},
				"go1":       {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    my_var u8 := 1
				    stage s1 {
				        my_var -> out
				        go2 => next
				    }
				    stage s2 {
				        my_var = 2
				        go1 => s1
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))
			trigger(h, ctx, 102)
			trigger(h, ctx, 103)
			out, _ = h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(2)))
		})

		It("Increments a variable once per stage entry", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.U8(), 101},
				"go2":       {types.U8(), 102},
				"go1":       {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    count u8 := 0
				    stage s1 {
				        count + 1 -> count
				        count -> out
				        go2 => next
				    }
				    stage s2 {
				        count = count + 1
				        go1 => s1
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			var got []uint8
			drain := func() {
				out, _ := h.Flush()
				for _, s := range out.Get(101).Series {
					got = append(got, telem.UnmarshalSeries[uint8](s)...)
				}
			}
			// Increments fire once per scope entry; reads fire only on unconsumed values.
			trigger(h, ctx, 100)
			drain()
			Expect(got).To(Equal([]uint8{0, 1}))
			trigger(h, ctx, 102)
			trigger(h, ctx, 103)
			drain()
			Expect(got).To(Equal([]uint8{0, 1, 2, 3}))
		})

		// Re-entry without a reassignment while away: the read must stay silent on
		// entry (already-seen value) and fire once with the new increment.
		It("Emits once per re-entry of an increment loop", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.U8(), 101},
				"go2":       {types.U8(), 102},
				"go1":       {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
				    count u8 := 0
				    stage s1 {
				        count + 1 -> count
				        count -> out
				        go2 => next
				    }
				    stage s2 {
				        go1 => s1
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			var got []uint8
			drain := func() {
				out, _ := h.Flush()
				for _, s := range out.Get(101).Series {
					got = append(got, telem.UnmarshalSeries[uint8](s)...)
				}
			}
			trigger(h, ctx, 100)
			drain()
			Expect(got).To(Equal([]uint8{0, 1}))
			trigger(h, ctx, 102)
			trigger(h, ctx, 103)
			drain()
			Expect(got).To(Equal([]uint8{0, 1, 2}))
		})

		It("Routes alias writes to the rebound channel across stages", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out_a":     {types.U8(), 101},
				"out_b":     {types.U8(), 102},
				"go2":       {types.U8(), 103},
				"go1":       {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
				l := out_a
				sequence main {
				    stage s1 {
				        u8(1) -> l
				        go2 => next
				    }
				    stage s2 {
				        l = out_b
				        go1 => s1
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.Uint8T},
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastU8(out, 101)).To(Equal(uint8(1)))
			Expect(out.Get(102).Series).To(BeEmpty())
			trigger(h, ctx, 103)
			trigger(h, ctx, 104)
			out, _ = h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(1)))
		})

		It("Routes alias reads to the rebound channel across stages", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_a":      {types.U8(), 101},
				"in_b":      {types.U8(), 102},
				"out":       {types.U8(), 105},
				"go2":       {types.U8(), 103},
				"go1":       {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
				r := in_a
				sequence main {
				    stage s1 {
				        r -> out
				        go2 => next
				    }
				    stage s2 {
				        r = in_b
				        go1 => s1
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 105, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(101, telem.NewSeriesV[uint8](10))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 105)).To(Equal(uint8(10)))

			trigger(h, ctx, 103)
			trigger(h, ctx, 104)
			h.Ingest(102, telem.NewSeriesV[uint8](20))
			h.Ingest(101, telem.NewSeriesV[uint8](99))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(lastU8(out, 105)).To(Equal(uint8(20)))
		})

		It("Routes alias expression reads to the rebound channel across stages", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_a":      {types.U8(), 101},
				"in_b":      {types.U8(), 102},
				"out":       {types.U8(), 105},
				"go2":       {types.U8(), 103},
				"go1":       {types.U8(), 104},
			})
			h := newRuntimeHarness(ctx, `
				r := in_a
				sequence main {
				    stage s1 {
				        r * 2 -> out
				        go2 => next
				    }
				    stage s2 {
				        r = in_b
				        go1 => s1
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 105, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(101, telem.NewSeriesV[uint8](10))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 105)).To(Equal(uint8(20)))

			trigger(h, ctx, 103)
			trigger(h, ctx, 104)
			h.Ingest(102, telem.NewSeriesV[uint8](30))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(lastU8(out, 105)).To(Equal(uint8(60)))
		})

		It("Routes a top-level alias expression read across a stage rebind", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_a":      {types.U8(), 101},
				"in_b":      {types.U8(), 102},
				"out":       {types.U8(), 105},
				"go2":       {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				r := in_a
				r * 2 -> out
				sequence main {
				    stage s1 {
				        go2 => next
				    }
				    stage s2 {
				        r = in_b
				    }
				}
				start_cmd => main`, resolver,
				channels.Digest{Key: 105, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(101, telem.NewSeriesV[uint8](10))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 105)).To(Equal(uint8(20)))

			trigger(h, ctx, 103)
			h.Ingest(102, telem.NewSeriesV[uint8](30))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(lastU8(out, 105)).To(Equal(uint8(60)))
		})

		It("Re-points a channel-read variable from an inline sequence", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"cpu": {types.U8(), 201},
				"out": {types.U8(), 202},
				"go2": {types.U8(), 203},
			})
			h := newRuntimeHarness(ctx, `
				stage {
				    x := cpu * 2
				    x -> out
				    go2 -> sequence {
				        x = cpu * 3
				    }
				}`, resolver,
				channels.Digest{Key: 202, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			h.Ingest(201, telem.NewSeriesV[uint8](5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 202)).To(Equal(uint8(10)))

			trigger(h, ctx, 203)
			out, _ = h.Flush()
			Expect(out.Get(202).Series).To(BeEmpty(), "a re-point alone must not emit")

			h.Ingest(201, telem.NewSeriesV[uint8](7))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(lastU8(out, 202)).To(Equal(uint8(21)))
		})
	})

	Describe("Scope-entry variable reset", func() {
		// drainStrings collects every string value flushed to a channel in
		// order across the loop's re-entries.
		drainStrings := func(fr telem.Frame[uint32], key uint32) []string {
			var out []string
			for _, ser := range fr.Get(key).Series {
				out = append(out, telem.UnmarshalSeries[string](ser)...)
			}
			return out
		}

		// loop drives four s1->s2->s1 re-entries via a 100ms wait, two
		// scheduler passes settling each re-entry.
		loop := func(h *runtimeHarness, ctx SpecContext) {
			step := func(now telem.TimeSpan) {
				h.Tick(ctx, now)
				h.channelState.ClearReads()
				h.Tick(ctx, now)
				h.channelState.ClearReads()
			}
			step(0)
			step(100 * telem.Millisecond)
			step(200 * telem.Millisecond)
			step(300 * telem.Millisecond)
		}

		It("Resets a := variable declared in the re-entered stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{"log": {types.String(), 101}})
			h := newRuntimeHarness(ctx, `import time
				sequence main {
				    stage s1 {
				        counter := 0
				        1 => counter + 1 => counter
				        str(counter) => log
				        time.wait{100ms} => next
				    }
				    stage s2 {
				        1 => s1
				    }
				}
				1 => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			loop(h, ctx)
			out, _ := h.Flush()
			Expect(drainStrings(out, 101)).To(Equal([]string{"0", "1", "0", "1", "0", "1", "0", "1"}))
		})

		It("Persists a $= variable declared in the re-entered stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{"log": {types.String(), 101}})
			h := newRuntimeHarness(ctx, `import time
				sequence main {
				    stage s1 {
				        counter $= 0
				        1 => counter + 1 => counter
				        str(counter) => log
				        time.wait{100ms} => next
				    }
				    stage s2 {
				        1 => s1
				    }
				}
				1 => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			loop(h, ctx)
			out, _ := h.Flush()
			Expect(drainStrings(out, 101)).To(Equal([]string{"0", "1", "2", "3", "4"}))
		})

		It("Does not reset a variable declared above the sub-scope that writes it", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"log_c": {types.String(), 101},
				"log_s": {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `import time
				sequence main {
				    counter_c := 0
				    counter_s $= 0
				    stage s1 {
				        1 => counter_c + 1 => counter_c
				        1 => counter_s + 1 => counter_s
				        str(counter_c) => log_c
				        str(counter_s) => log_s
				        time.wait{100ms} => next
				    }
				    stage s2 {
				        1 => s1
				    }
				}
				1 => main`, resolver,
				channels.Digest{Key: 101, DataType: telem.StringT},
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			loop(h, ctx)
			out, _ := h.Flush()
			Expect(drainStrings(out, 101)).To(Equal([]string{"0", "1", "2", "3", "4"}))
			Expect(drainStrings(out, 102)).To(Equal([]string{"0", "1", "2", "3", "4"}))
		})
	})

	Describe("Format-string interpolation of variables", func() {
		It("Interpolates a reassigned literal variable", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    my_var := "hello"
				    my_var = "updated"
				    `+`f"val={my_var}"`+` -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("val=updated"))
		})

		It("Interpolates a self-referentially incremented variable", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    counter u8 := 5
				    counter = counter + 1
				    `+`f"n={counter}"`+` -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("n=6"))
		})

		It("Interpolates a channel read/write that follows a rebind", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"sensor_a":  {types.U8(), 201},
				"sensor_b":  {types.U8(), 202},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    src := sensor_a
				    src = sensor_b
				    `+`f"v={src}"`+` -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(201, telem.NewSeriesV[uint8](3))
			h.Ingest(202, telem.NewSeriesV[uint8](7))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("v=7"))
		})

		It("Interpolates a re-expressed channel-read variable", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_val":    {types.U8(), 200},
				"out":       {types.String(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    r := in_val + u8(1)
				    r = in_val + u8(100)
				    `+`f"r={r}"`+` -> out
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(200, telem.NewSeriesV[uint8](5))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal("r=105"))
		})

		It("Reads and interpolates an inherited channel read/write from a nested stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"sensor":     {types.U8(), 201},
				"out_direct": {types.U8(), 102},
				"out_fmt":    {types.String(), 103},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    ia := sensor
				    stage {
				        ia -> out_direct
				        `+`f"a={ia}"`+` -> out_fmt
				    }
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.Uint8T},
				channels.Digest{Key: 103, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(201, telem.NewSeriesV[uint8](7))
			advance(h, ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(7)))
			Expect(lastString(out, 103)).To(Equal("a=7"))
		})

		It("Reads an inherited channel-read variable from a nested stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"in_val":    {types.U8(), 200},
				"out":       {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				sequence s {
				    r := in_val + u8(1)
				    stage {
				        r -> out
				    }
				}
				start_cmd => s`, resolver,
				channels.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)

			trigger(h, ctx, 100)
			h.Ingest(200, telem.NewSeriesV[uint8](5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastU8(out, 102)).To(Equal(uint8(6)))
		})
	})

	// A triggered format string or expression samples a variable's latest value
	// at each fire; writes alone never fire it.
	Describe("Triggered expression variable reads", func() {
		fmtSrc := `
    sequence main {
        v u8 := 3
        stage s1 {
            set_ch -> v
            go_ch -> ` + `f"v={v}"` + ` -> out
        }
    }
    start_cmd => main`
		newH := func(ctx SpecContext, src string) *runtimeHarness {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 300},
				"set_ch":    {types.U8(), 301},
				"go_ch":     {types.U8(), 302},
				"out":       {types.String(), 303},
			})
			return newRuntimeHarness(ctx, src, resolver,
				channels.Digest{Key: 300, DataType: telem.Uint8T},
				channels.Digest{Key: 301, DataType: telem.Uint8T},
				channels.Digest{Key: 302, DataType: telem.Uint8T},
				channels.Digest{Key: 303, DataType: telem.StringT},
			)
		}
		pushVal := func(h *runtimeHarness, ctx SpecContext, v uint8) {
			h.Ingest(301, telem.NewSeriesV[uint8](v))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
		}

		It("reads the latest value at each trigger", func(ctx SpecContext) {
			h := newH(ctx, fmtSrc)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(lastString(out, 303)).To(Equal("v=3"))
			pushVal(h, ctx, 9)
			trigger(h, ctx, 302)
			out, _ = h.Flush()
			Expect(lastString(out, 303)).To(Equal("v=9"))
		})

		It("fires exactly once per trigger and never on idle passes", func(ctx SpecContext) {
			h := newH(ctx, fmtSrc)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(countOf(out, 303, "v=3")).To(Equal(1))
			for range 10 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 303, "v=3")).To(BeZero())
		})

		It("does not fire on a variable write alone", func(ctx SpecContext) {
			h := newH(ctx, fmtSrc)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			_, _ = h.Flush()
			pushVal(h, ctx, 9)
			out, _ := h.Flush()
			Expect(out.Get(303).Series).To(BeEmpty(),
				"a write must not fire the triggered read")
		})

		It("re-emits an unchanged value on every trigger", func(ctx SpecContext) {
			h := newH(ctx, fmtSrc)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			trigger(h, ctx, 302)
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(countOf(out, 303, "v=3")).To(Equal(2))
		})

		It("reads the latest value through a triggered expression", func(ctx SpecContext) {
			h := newH(ctx, `
    sequence main {
        v u8 := 3
        stage s1 {
            set_ch -> v
            go_ch -> str(v) -> out
        }
    }
    start_cmd => main`)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(lastString(out, 303)).To(Equal("3"))
			pushVal(h, ctx, 9)
			trigger(h, ctx, 302)
			out, _ = h.Flush()
			Expect(lastString(out, 303)).To(Equal("9"))
		})

		It("reads the latest value of a stateful variable", func(ctx SpecContext) {
			h := newH(ctx, `
    sequence main {
        v u8 $= 3
        stage s1 {
            set_ch -> v
            go_ch -> `+`f"v={v}"`+` -> out
        }
    }
    start_cmd => main`)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(lastString(out, 303)).To(Equal("v=3"))
			pushVal(h, ctx, 9)
			trigger(h, ctx, 302)
			out, _ = h.Flush()
			Expect(lastString(out, 303)).To(Equal("v=9"))
		})

		It("samples every variable in a multi-placeholder format string", func(ctx SpecContext) {
			h := newH(ctx, `
    sequence main {
        a u8 := 3
        b u8 := 3
        stage s1 {
            set_ch -> b
            go_ch -> `+`f"{a}-{b}"`+` -> out
        }
    }
    start_cmd => main`)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(lastString(out, 303)).To(Equal("3-3"))
			pushVal(h, ctx, 9)
			trigger(h, ctx, 302)
			out, _ = h.Flush()
			Expect(lastString(out, 303)).To(Equal("3-9"))
		})

		It("reads the reset initial after stage re-entry", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 300},
				"set_ch":    {types.U8(), 301},
				"go_ch":     {types.U8(), 302},
				"out":       {types.String(), 303},
				"next_ch":   {types.U8(), 304},
				"back_ch":   {types.U8(), 305},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        stage s1 {
            v u8 := 3
            set_ch -> v
            go_ch -> `+`f"v={v}"`+` -> out
            next_ch => next
        }
        stage s2 {
            back_ch => s1
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 300, DataType: telem.Uint8T},
				channels.Digest{Key: 301, DataType: telem.Uint8T},
				channels.Digest{Key: 302, DataType: telem.Uint8T},
				channels.Digest{Key: 303, DataType: telem.StringT},
				channels.Digest{Key: 304, DataType: telem.Uint8T},
				channels.Digest{Key: 305, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			pushVal(h, ctx, 9)
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(lastString(out, 303)).To(Equal("v=9"))
			trigger(h, ctx, 304)
			trigger(h, ctx, 305)
			trigger(h, ctx, 302)
			out, _ = h.Flush()
			Expect(lastString(out, 303)).To(Equal("v=3"))
		})

		It("fires per interval tick with the live value", func(ctx SpecContext) {
			h := newH(ctx, `import time
    stage main {
        x u8 := 3
        set_ch -> x
        time.interval{50ms} -> `+`f"v={x}"`+` -> out
    }
    1 -> main`)
			defer h.Close(ctx)
			advance(h, ctx, 0)
			advance(h, ctx, 60*telem.Millisecond)
			out, _ := h.Flush()
			Expect(countOf(out, 303, "v=3")).To(Equal(1))
			h.Ingest(301, telem.NewSeriesV[uint8](9))
			advance(h, ctx, 65*telem.Millisecond)
			advance(h, ctx, 115*telem.Millisecond)
			out, _ = h.Flush()
			Expect(countOf(out, 303, "v=9")).To(Equal(1))
			Expect(countOf(out, 303, "v=3")).To(BeZero())
		})
	})

	// A func brace input bound to a variable reads the variable's value at
	// each fire; the node fires only on its own trigger.
	Describe("Variable brace inputs", func() {
		src := `
    func echo{tag str} (n u8) str {
        return tag
    }
    sequence main {
        v str := "initial"
        stage s1 {
            tag_ch -> v
            go_ch -> echo{tag=v} -> echo_out
        }
    }
    start_cmd => main`
		newH := func(ctx SpecContext) *runtimeHarness {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 300},
				"tag_ch":    {types.String(), 301},
				"go_ch":     {types.U8(), 302},
				"echo_out":  {types.String(), 303},
			})
			return newRuntimeHarness(ctx, src, resolver,
				channels.Digest{Key: 300, DataType: telem.Uint8T},
				channels.Digest{Key: 301, DataType: telem.StringT},
				channels.Digest{Key: 302, DataType: telem.Uint8T},
				channels.Digest{Key: 303, DataType: telem.StringT},
			)
		}
		pushTag := func(h *runtimeHarness, ctx SpecContext, v string) {
			h.Ingest(301, telem.NewSeriesV[string](v))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
		}

		It("passes the variable's latest value at each fire", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			pushTag(h, ctx, "alpha")
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(lastString(out, 303)).To(Equal("alpha"))
			pushTag(h, ctx, "beta")
			trigger(h, ctx, 302)
			out, _ = h.Flush()
			Expect(lastString(out, 303)).To(Equal("beta"))
		})

		It("uses the declared initial before any write", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(lastString(out, 303)).To(Equal("initial"))
		})

		It("fires the consumer exactly once per trigger", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 300)
			pushTag(h, ctx, "alpha")
			trigger(h, ctx, 302)
			out, _ := h.Flush()
			Expect(countOf(out, 303, "alpha")).To(Equal(1))
			// Idle cycles must not re-fire the echo.
			for range 10 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 303, "alpha")).To(BeZero())
			// A fresh trigger still fires, exactly once.
			trigger(h, ctx, 302)
			out, _ = h.Flush()
			Expect(countOf(out, 303, "alpha")).To(Equal(1))
		})
	})

	Describe("Stateful variable brace inputs", func() {
		src := `
    func echo{tag str} (n u8) str {
        return tag
    }
    sequence main {
        stage s1 {
            acc str $= "initial"
            tag_ch -> acc
            go_ch -> echo{tag=acc} -> echo_out
            hop_ch >= 1 => s2
        }
        stage s2 {
            back_ch >= 1 => s1
        }
    }
    start_cmd => main`
		newH := func(ctx SpecContext) *runtimeHarness {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 310},
				"tag_ch":    {types.String(), 311},
				"go_ch":     {types.U8(), 312},
				"echo_out":  {types.String(), 313},
				"hop_ch":    {types.U8(), 314},
				"back_ch":   {types.U8(), 315},
			})
			return newRuntimeHarness(ctx, src, resolver,
				channels.Digest{Key: 310, DataType: telem.Uint8T},
				channels.Digest{Key: 311, DataType: telem.StringT},
				channels.Digest{Key: 312, DataType: telem.Uint8T},
				channels.Digest{Key: 313, DataType: telem.StringT},
				channels.Digest{Key: 314, DataType: telem.Uint8T},
				channels.Digest{Key: 315, DataType: telem.Uint8T},
			)
		}
		pushTag := func(h *runtimeHarness, ctx SpecContext, v string) {
			h.Ingest(311, telem.NewSeriesV[string](v))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
		}

		It("passes the stateful variable's latest value at each fire", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 310)
			pushTag(h, ctx, "alpha")
			trigger(h, ctx, 312)
			out, _ := h.Flush()
			Expect(lastString(out, 313)).To(Equal("alpha"))
			pushTag(h, ctx, "beta")
			trigger(h, ctx, 312)
			out, _ = h.Flush()
			Expect(lastString(out, 313)).To(Equal("beta"))
		})

		It("uses the declared initial before any write", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 310)
			trigger(h, ctx, 312)
			out, _ := h.Flush()
			Expect(lastString(out, 313)).To(Equal("initial"))
		})

		It("fires the consumer exactly once per trigger", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 310)
			pushTag(h, ctx, "alpha")
			trigger(h, ctx, 312)
			out, _ := h.Flush()
			Expect(countOf(out, 313, "alpha")).To(Equal(1))
			for range 10 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 313, "alpha")).To(BeZero())
			trigger(h, ctx, 312)
			out, _ = h.Flush()
			Expect(countOf(out, 313, "alpha")).To(Equal(1))
		})

		It("retains the written value across a stage re-entry", func(ctx SpecContext) {
			h := newH(ctx)
			defer h.Close(ctx)
			trigger(h, ctx, 310)
			pushTag(h, ctx, "alpha")
			trigger(h, ctx, 312)
			out, _ := h.Flush()
			Expect(lastString(out, 313)).To(Equal("alpha"))
			// Hop s1 -> s2 -> s1; a := variable would reset to "initial" here.
			trigger(h, ctx, 314)
			trigger(h, ctx, 315)
			trigger(h, ctx, 312)
			out, _ = h.Flush()
			Expect(lastString(out, 313)).To(Equal("alpha"))
		})
	})

	Describe("Channel brace inputs", func() {
		newH := func(ctx SpecContext, src string) *runtimeHarness {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 320},
				"data_ch":    {types.F32(), 321},
				"reader_out": {types.F32(), 322},
				"go_ch":      {types.U8(), 323},
				"sink_ch":    {types.F32(), 324},
				"data2_ch":   {types.F32(), 325},
			})
			return newRuntimeHarness(ctx, src, resolver,
				channels.Digest{Key: 320, DataType: telem.Uint8T},
				channels.Digest{Key: 321, DataType: telem.Float32T},
				channels.Digest{Key: 322, DataType: telem.Float32T},
				channels.Digest{Key: 323, DataType: telem.Uint8T},
				channels.Digest{Key: 324, DataType: telem.Float32T},
				channels.Digest{Key: 325, DataType: telem.Float32T},
			)
		}
		pushData := func(h *runtimeHarness, ctx SpecContext, v float32) {
			h.Ingest(321, telem.NewSeriesV[float32](v))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
		}
		run := func(ctx SpecContext, src string) {
			h := newH(ctx, src)
			defer h.Close(ctx)
			trigger(h, ctx, 320)
			pushData(h, ctx, 1.5)
			trigger(h, ctx, 323)
			out, _ := h.Flush()
			Expect(lastF32(out, 322)).To(Equal(float32(1.5)))
			pushData(h, ctx, 2.5)
			trigger(h, ctx, 323)
			out, _ = h.Flush()
			Expect(lastF32(out, 322)).To(Equal(float32(2.5)))
		}

		It("reads the latest channel data at each fire", func(ctx SpecContext) {
			run(ctx, `
    func reader{channel chan f32} (n u8) f32 {
        return channel
    }
    sequence main {
        stage s1 {
            go_ch -> reader{channel=data_ch} -> reader_out
        }
    }
    start_cmd => main`)
		})

		It("reads the latest data through a channel alias", func(ctx SpecContext) {
			run(ctx, `
    func reader{channel chan f32} (n u8) f32 {
        return channel
    }
    sequence main {
        a := data_ch
        stage s1 {
            go_ch -> reader{channel=a} -> reader_out
        }
    }
    start_cmd => main`)
		})

		runWrite := func(ctx SpecContext, src string) {
			h := newH(ctx, src)
			defer h.Close(ctx)
			trigger(h, ctx, 320)
			pushData(h, ctx, 1.5)
			out, _ := h.Flush()
			Expect(lastF32(out, 324)).To(Equal(float32(1.5)))
			pushData(h, ctx, 2.5)
			out, _ = h.Flush()
			Expect(lastF32(out, 324)).To(Equal(float32(2.5)))
		}

		It("writes each value to the channel", func(ctx SpecContext) {
			runWrite(ctx, `
    func writer{channel chan f32} (value f32) {
        channel = value
    }
    sequence main {
        stage s1 {
            data_ch -> writer{channel=sink_ch}
        }
    }
    start_cmd => main`)
		})

		It("writes each value through a channel alias", func(ctx SpecContext) {
			runWrite(ctx, `
    func writer{channel chan f32} (value f32) {
        channel = value
    }
    sequence main {
        w := sink_ch
        stage s1 {
            data_ch -> writer{channel=w}
        }
    }
    start_cmd => main`)
		})

		It("writes through the declared binding, then the rebound one", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 330},
				"data_ch":   {types.F32(), 331},
				"sink_a":    {types.F32(), 332},
				"sink_b":    {types.F32(), 333},
				"hop_ch":    {types.U8(), 334},
			})
			h := newRuntimeHarness(ctx, `
    func writer{channel chan f32} (value f32) {
        channel = value
    }
    sequence main {
        w := sink_a
        stage s1 {
            data_ch -> writer{channel=w}
            hop_ch >= 1 => s2
        }
        stage s2 {
            w = sink_b
            data_ch -> writer{channel=w}
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 330, DataType: telem.Uint8T},
				channels.Digest{Key: 331, DataType: telem.Float32T},
				channels.Digest{Key: 332, DataType: telem.Float32T},
				channels.Digest{Key: 333, DataType: telem.Float32T},
				channels.Digest{Key: 334, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 330)
			h.Ingest(331, telem.NewSeriesV[float32](1.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(lastF32(out, 332)).To(Equal(float32(1.5)))
			trigger(h, ctx, 334)
			h.Ingest(331, telem.NewSeriesV[float32](2.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(lastF32(out, 333)).To(Equal(float32(2.5)))
		})

		It("reads through the rebound channel, not the declared one", func(ctx SpecContext) {
			h := newH(ctx, `
    func reader{channel chan f32} (n u8) f32 {
        return channel
    }
    sequence main {
        a := data_ch
        stage s1 {
            a = data2_ch
            go_ch -> reader{channel=a} -> reader_out
        }
    }
    start_cmd => main`)
			defer h.Close(ctx)
			trigger(h, ctx, 320)
			h.Ingest(325, telem.NewSeriesV[float32](7.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			trigger(h, ctx, 323)
			out, _ := h.Flush()
			Expect(lastF32(out, 322)).To(Equal(float32(7.5)))
		})
	})

	Describe("Timer brace inputs", func() {
		It("adopts the reassigned interval period at the next fire", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 340},
				"tick_ch":   {types.U8(), 341},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        rate := i64 ns(100ms)
        stage s1 {
            interval{rate} -> tick_ch
            wait{1s} -> sequence {
                rate = i64 ns(2s)
            }
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 340, DataType: telem.Uint8T},
				channels.Digest{Key: 341, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			// Stage entry: the interval fires immediately.
			trigger(h, ctx, 340)
			out, _ := h.Flush()
			Expect(out.Get(341).Series).ToNot(BeEmpty())
			// 100ms cadence while the declared period holds.
			advance(h, ctx, 141*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(341).Series).ToNot(BeEmpty())
			advance(h, ctx, 171*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(341).Series).To(BeEmpty(),
				"30ms after the last fire must not tick at a 100ms period")
			// The 1s wait fires and reassigns rate to 2s; the interval also
			// ticks here, which the flush drains.
			advance(h, ctx, 1101*telem.Millisecond)
			_, _ = h.Flush()
			advance(h, ctx, 1400*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(341).Series).To(BeEmpty(),
				"the interval must adopt the reassigned 2s period")
			advance(h, ctx, 3*telem.Second)
			out, _ = h.Flush()
			Expect(out.Get(341).Series).To(BeEmpty())
			// 2s after the last fire: ticks again.
			advance(h, ctx, 3200*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(341).Series).ToNot(BeEmpty())
		})

		It("fires the wait earlier when its duration is shortened mid-wait", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 342},
				"done_ch":   {types.U8(), 343},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        hold := i64 ns(5s)
        stage s1 {
            wait{300ms} -> sequence {
                hold = i64 ns(1s)
            }
            wait{duration=hold} -> done_ch
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 342, DataType: telem.Uint8T},
				channels.Digest{Key: 343, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 342)
			// The 300ms wait fires and shortens hold to 1s.
			advance(h, ctx, 400*telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(343).Series).To(BeEmpty())
			advance(h, ctx, 700*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(343).Series).To(BeEmpty())
			// 1s after entry: fires under the shortened duration, not at 5s.
			advance(h, ctx, 1100*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 343)).To(Equal(uint8(1)))
		})

		It("holds the wait longer when its duration is lengthened mid-wait", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 344},
				"done_ch":   {types.U8(), 345},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        hold := i64 ns(500ms)
        stage s1 {
            wait{200ms} -> sequence {
                hold = i64 ns(3s)
            }
            wait{duration=hold} -> done_ch
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 344, DataType: telem.Uint8T},
				channels.Digest{Key: 345, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 344)
			// The 200ms wait fires and lengthens hold to 3s.
			advance(h, ctx, 260*telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(345).Series).To(BeEmpty())
			// 600ms after entry: the declared 500ms has passed, but the live
			// duration is now 3s.
			advance(h, ctx, 600*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(345).Series).To(BeEmpty(),
				"the wait must adopt the lengthened duration")
			advance(h, ctx, 3100*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 345)).To(Equal(uint8(1)))
		})
	})

	// A triggered mid-chain variable read emits the variable's live value on
	// each fire; the write itself never fires the chain.
	Describe("Mid-chain variable reads", func() {
		It("emits the live string value at each interval fire", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 350},
				"log_ch":    {types.String(), 351},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        text := "hello"
        stage s1 {
            interval{300ms} -> text -> log_ch
            wait{100ms} -> sequence {
                text = "goodbye"
            }
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 350, DataType: telem.Uint8T},
				channels.Digest{Key: 351, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			// Stage entry: the interval fires and the read emits the initial.
			trigger(h, ctx, 350)
			out, _ := h.Flush()
			Expect(lastString(out, 351)).To(Equal("hello"))
			// The wait fires and reassigns text; the write alone must not
			// re-fire the chain.
			advance(h, ctx, 171*telem.Millisecond)
			advance(h, ctx, 172*telem.Millisecond)
			advance(h, ctx, 173*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(351).Series).To(BeEmpty(),
				"a variable write must not fire the reading chain")
			// Next interval fire: the read emits the reassigned value.
			advance(h, ctx, 331*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastString(out, 351)).To(Equal("goodbye"))
			Expect(countOf(out, 351, "hello")).To(Equal(0),
				"the stale initial must not be emitted after the write")
			// The reassigned value persists on later fires.
			advance(h, ctx, 700*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastString(out, 351)).To(Equal("goodbye"))
		})

		It("emits the live numeric value at each interval fire", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 352},
				"out_ch":    {types.U8(), 353},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        k u8 := 5
        stage s1 {
            interval{300ms} -> k -> out_ch
            wait{100ms} -> sequence {
                k = 9
            }
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 352, DataType: telem.Uint8T},
				channels.Digest{Key: 353, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 352)
			out, _ := h.Flush()
			Expect(lastU8(out, 353)).To(Equal(uint8(5)))
			advance(h, ctx, 171*telem.Millisecond)
			advance(h, ctx, 172*telem.Millisecond)
			advance(h, ctx, 173*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(353).Series).To(BeEmpty(),
				"a variable write must not fire the reading chain")
			advance(h, ctx, 331*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 353)).To(Equal(uint8(9)))
		})

		It("keeps head reads write-driven while mid-chain reads stay trigger-driven", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 354},
				"slow_out":  {types.U8(), 355},
				"live_out":  {types.U8(), 356},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        k u8 := 5
        stage s1 {
            interval{10s} -> k -> slow_out
            k -> live_out
            wait{100ms} -> sequence {
                k = 9
            }
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 354, DataType: telem.Uint8T},
				channels.Digest{Key: 355, DataType: telem.Uint8T},
				channels.Digest{Key: 356, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			// Entry: the interval fires once and reads the initial.
			trigger(h, ctx, 354)
			out, _ := h.Flush()
			Expect(lastU8(out, 355)).To(Equal(uint8(5)))
			// The reassignment fires the head read but not the interval chain.
			advance(h, ctx, 171*telem.Millisecond)
			advance(h, ctx, 172*telem.Millisecond)
			advance(h, ctx, 173*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 356)).To(Equal(uint8(9)),
				"the head read must fire on the write")
			Expect(out.Get(355).Series).To(BeEmpty(),
				"the write must not fire the interval chain")
			advance(h, ctx, 500*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(355).Series).To(BeEmpty(),
				"a 10s interval must stay quiet at 500ms")
		})

		It("emits the live stateful variable value at each interval fire", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 357},
				"out_ch":    {types.U8(), 358},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        k u8 $= 5
        stage s1 {
            interval{300ms} -> k -> out_ch
            wait{100ms} -> sequence {
                k = 9
            }
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 357, DataType: telem.Uint8T},
				channels.Digest{Key: 358, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 357)
			out, _ := h.Flush()
			Expect(lastU8(out, 358)).To(Equal(uint8(5)))
			advance(h, ctx, 171*telem.Millisecond)
			advance(h, ctx, 172*telem.Millisecond)
			advance(h, ctx, 173*telem.Millisecond)
			out, _ = h.Flush()
			Expect(out.Get(358).Series).To(BeEmpty(),
				"a variable write must not fire the reading chain")
			advance(h, ctx, 331*telem.Millisecond)
			out, _ = h.Flush()
			Expect(lastU8(out, 358)).To(Equal(uint8(9)))
		})
	})

	// A reactive read variable heads a chain; each source sample must fire
	// the chain exactly once.
	Describe("Reactive read chain cadence", func() {
		It("fires once per source sample", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"src_ch":  {types.F32(), 360},
				"out_str": {types.String(), 361},
			})
			h := newRuntimeHarness(ctx, `
    v := src_ch - f32(1)
    v -> "tick" -> out_str`, resolver,
				channels.Digest{Key: 360, DataType: telem.Float32T},
				channels.Digest{Key: 361, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(360, telem.NewSeriesV[float32](2.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(countOf(out, 361, "tick")).To(Equal(1),
				"one sample must fire the chain exactly once")
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 361, "tick")).To(Equal(0),
				"idle cycles must not re-fire the chain")
			h.Ingest(360, telem.NewSeriesV[float32](3.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 361, "tick")).To(Equal(1),
				"the next sample must fire the chain exactly once")
		})

		It("keeps the cadence when an unrelated timer chain adds cycles", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"src_ch":  {types.F32(), 362},
				"out_str": {types.String(), 363},
				"beat_ch": {types.U8(), 364},
			})
			h := newRuntimeHarness(ctx, `
    v := src_ch - f32(1)
    v -> "tick" -> out_str
    interval{50ms} -> beat_ch`, resolver,
				channels.Digest{Key: 362, DataType: telem.Float32T},
				channels.Digest{Key: 363, DataType: telem.StringT},
				channels.Digest{Key: 364, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			h.Ingest(362, telem.NewSeriesV[float32](2.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			advance(h, ctx, 60*telem.Millisecond)
			advance(h, ctx, 120*telem.Millisecond)
			advance(h, ctx, 180*telem.Millisecond)
			out, _ := h.Flush()
			Expect(lastU8(out, 364)).To(Equal(uint8(1)),
				"the timer chain must be adding cycles")
			Expect(countOf(out, 363, "tick")).To(Equal(1),
				"timer cycles must not re-fire the reactive chain")
		})

		It("fires once per distinct sample", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"src_ch":  {types.F32(), 365},
				"out_str": {types.String(), 366},
			})
			h := newRuntimeHarness(ctx, `
    v := src_ch - f32(1)
    v -> "tick" -> out_str`, resolver,
				channels.Digest{Key: 365, DataType: telem.Float32T},
				channels.Digest{Key: 366, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			for _, sample := range []float32{2.5, 3.5, 4.5} {
				h.Ingest(365, telem.NewSeriesV[float32](sample))
				for range 5 {
					advance(h, ctx, telem.Millisecond)
				}
			}
			out, _ := h.Flush()
			Expect(countOf(out, 366, "tick")).To(Equal(3))
		})

		It("re-fires on a recomputed identical value", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"src_ch":  {types.F32(), 367},
				"out_str": {types.String(), 368},
			})
			h := newRuntimeHarness(ctx, `
    v := src_ch - f32(1)
    v -> "tick" -> out_str`, resolver,
				channels.Digest{Key: 367, DataType: telem.Float32T},
				channels.Digest{Key: 368, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(367, telem.NewSeriesV[float32](2.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(countOf(out, 368, "tick")).To(Equal(1))
			h.Ingest(367, telem.NewSeriesV[float32](2.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 368, "tick")).To(Equal(1),
				"a fresh sample fires even when the value is unchanged")
		})

		It("fires once per sample when the chain lives in a stage", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 369},
				"src_ch":    {types.F32(), 370},
				"out_str":   {types.String(), 371},
			})
			h := newRuntimeHarness(ctx, `
    sequence main {
        v := src_ch - f32(1)
        stage s1 {
            v -> "tick" -> out_str
        }
    }
    start_cmd => main`, resolver,
				channels.Digest{Key: 369, DataType: telem.Uint8T},
				channels.Digest{Key: 370, DataType: telem.Float32T},
				channels.Digest{Key: 371, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 369)
			h.Ingest(370, telem.NewSeriesV[float32](2.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ := h.Flush()
			Expect(countOf(out, 371, "tick")).To(Equal(1),
				"one sample must fire the staged chain exactly once")
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 371, "tick")).To(Equal(0),
				"idle cycles must not re-fire the staged chain")
			h.Ingest(370, telem.NewSeriesV[float32](3.5))
			for range 5 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 371, "tick")).To(Equal(1))
		})
	})

	// A one-shot entry drives a func whose output rebinds a variable;
	// the loopback must not re-fire the chain.
	Describe("Variable capture dispatch", func() {
		src := `
    func stamp{tag str} (n u8) str {
        return tag
    }
    sequence main {
        fp str := ""
        stage s1 {
            1 -> stamp{tag="parent"} -> fp
            fp -> fp_out
        }
    }
    start_cmd => main`

		It("executes a one-shot capture chain exactly once", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 400},
				"fp_out":    {types.String(), 401},
			})
			h := newRuntimeHarness(ctx, src, resolver,
				channels.Digest{Key: 400, DataType: telem.Uint8T},
				channels.Digest{Key: 401, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			trigger(h, ctx, 400)
			out, _ := h.Flush()
			Expect(countOf(out, 401, "parent")).To(Equal(1))
			// Settle passes and later cycles must not re-run the capture.
			for range 10 {
				advance(h, ctx, telem.Millisecond)
			}
			out, _ = h.Flush()
			Expect(countOf(out, 401, "parent")).To(BeZero())
		})
	})
})
