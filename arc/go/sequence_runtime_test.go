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
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

// Sequence/stage end-to-end shapes pulled from integration/tests/arc. Each shape
// exercises a distinct composable-execution pattern introduced by RFC 0034.
// Harness construction alone is enough to exercise ProgramState.Node for every
// emitted node, which is where wrapper-kind unbound-input bugs surface.
var _ = Describe("Sequence/Stage Runtime Shapes", func() {
	tick := func(h *runtimeHarness, ctx SpecContext) {
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()
	}

	Describe("wait_timing shape: two-stage sequence with wait transition", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"toggle_cmd": {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				sequence main {
					stage stage1 {
						1 -> toggle_cmd,
						wait{1s} => next,
					}
					stage stage2 {
						0 -> toggle_cmd,
					}
				}

				start_cmd => main
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("lifecycle shape: multi-stage sequence with comparison transitions", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":     {types.U8(), 100},
				"press_pt":      {types.F32(), 101},
				"press_vlv_cmd": {types.U8(), 102},
				"vent_vlv_cmd":  {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				HIGH f32 := 25
				LOW f32 := 5

				sequence main {
					stage press {
						1 -> press_vlv_cmd,
						press_pt > HIGH => vent
					}
					stage vent {
						1 -> vent_vlv_cmd,
						press_pt < LOW => complete
					}
					stage complete {
						0 -> press_vlv_cmd,
						0 -> vent_vlv_cmd
					}
				}

				start_cmd => main
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("short_circuit shape: flow with compound condition + noop chain", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":  {types.U8(), 100},
				"heater_cmd": {types.U8(), 101},
				"temp_a":     {types.F32(), 102},
				"temp_b":     {types.F32(), 103},
			})
			h := newRuntimeHarness(ctx, `
				func noop{}(input u8) u8 {
					return input
				}

				start_cmd => main

				sequence main {
					stage on {
						1 -> heater_cmd,
						interval{1s} -> (temp_a > 290 and temp_b > 290) -> noop{} => off,
					}
					stage off {
						0 -> heater_cmd,
					}
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Float32T},
				channel.Digest{Key: 103, DataType: telem.Float32T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("interval_press shape: top-level interval-driven function", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"press_pt":      {types.F32(), 100},
				"press_vlv_cmd": {types.U8(), 101},
				"vent_vlv_cmd":  {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				func open_press() {
					if (press_pt > 30) {
						press_vlv_cmd = 0
						vent_vlv_cmd = 1
					} else if (press_pt < 1) {
						press_vlv_cmd = 1
						vent_vlv_cmd = 0
					}
				}

				interval{50ms} -> open_press{}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("stateful_var shape: persistent counter across ticks", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"count_out": {types.U8(), 101},
			})
			h := newRuntimeHarness(ctx, `
				func count{c_chan chan u8}() {
					n u8 $= 0
					n = n + 1
					c_chan = n
				}

				start_cmd => main

				sequence main {
					stage run {
						count{c_chan = count_out},
					}
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("signal_ctrl shape: separate signal-triggered sequence", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"stop_cmd":  {types.U8(), 101},
				"log_out":   {types.U8(), 102},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => signal_ctrl

				sequence signal_ctrl {
					stage start {
						1 -> log_out,
						stop_cmd => stop
					}
					stage stop {
						0 -> log_out
					}
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("string_channel_write shape: string literal -> str channel", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"log":       {types.String(), 101},
			})
			h := newRuntimeHarness(ctx, `
				start_cmd => main

				sequence main {
					stage a {
						"hello" -> log,
					}
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("event_log shape: function with str config + void return", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"log":       {types.String(), 101},
			})
			h := newRuntimeHarness(ctx, `
				func event_log{msg str} () {
					log = msg
				}

				start_cmd => main

				sequence main {
					stage a {
						event_log{"pressurizing"},
					}
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	// TODO: stable_for + select + branching writes. Tripping over Arc literal typing
	// ("expected i64, got u8"). Add back once the syntax is pinned down.
	PDescribe("stable_select shape: sensor -> fn -> stable_for -> select -> branches", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {})
	})

	Describe("const_to_channel shape: global const => channel", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":    {types.U8(), 100},
				"const_output": {types.F32(), 101},
			})
			h := newRuntimeHarness(ctx, `
				SOME_CONST f32 := 42.0

				start_cmd => main

				sequence main {
					stage a {
						SOME_CONST => const_output,
					}
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("arith_to_channel shape: const expr => channel", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"start_cmd":    {types.U8(), 100},
				"const_output": {types.F32(), 101},
			})
			h := newRuntimeHarness(ctx, `
				SOME_CONST f32 := -49.5

				start_cmd => main

				sequence main {
					stage a {
						SOME_CONST * 2 => const_output,
					}
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("nested_forward_fn shape: function calling forward-declared sibling", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"out":    {types.F32(), 101},
			})
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

				interval{100ms} -> nested_1{}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})

	Describe("flow_then_stage shape: top-level flow + separate sequence", func() {
		It("Should construct and tick without panic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor":    {types.F32(), 100},
				"doubled":   {types.F32(), 101},
				"start_cmd": {types.U8(), 102},
				"cmd":       {types.U8(), 103},
			})
			h := newRuntimeHarness(ctx, `
				func dbl(v f32) f32 {
					return v * 2
				}

				sensor -> dbl{} -> doubled

				start_cmd => main

				sequence main {
					stage only {
						1 -> cmd,
					}
				}
			`, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 103, DataType: telem.Uint8T},
			)
			defer h.Close(ctx)
			tick(h, ctx)
		})
	})
})
