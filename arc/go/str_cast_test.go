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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("str() typecast end-to-end runtime", func() {
	lastString := func(fr telem.Frame[uint32], key uint32) string {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[string](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	DescribeTable(
		"compiles, runs, and writes the expected string to a string channel",
		func(ctx SpecContext, source, expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"trig": {types.U8(), 100},
				"log":  {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`func f() {
				    log = `+source+`
				}
				trig -> f{}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal(expected))
		},
		Entry("float literal 3.1 (natural f64)", "str(3.1)", "3.1"),
		Entry("float literal 3.14 (natural f64)", "str(3.14)", "3.14"),
		Entry("float literal 123.456 (natural f64)", "str(123.456)", "123.456"),
		Entry(
			"float literal 123.456000 (trailing fractional zeros)",
			"str(123.456000)",
			"123.456",
		),
		Entry("float literal 123. (whole-valued, no fraction)", "str(123.)", "123"),
		Entry("float literal 1.0 (integer-valued)", "str(1.0)", "1"),
		Entry("float literal 100.000 (trailing zeros)", "str(100.000)", "100"),
		Entry("float literal -0.0 (negative zero)", "str(-0.0)", "-0"),
		Entry(
			"float literal -0.0000 (negative zero with trailing zeros)",
			"str(-0.0000)",
			"-0",
		),
		Entry("explicit f32(3.14)", "str(f32(3.14))", "3.14"),
		Entry("explicit f64(3.14)", "str(f64(3.14))", "3.14"),
		Entry("integer literal 42", "str(42)", "42"),
		Entry("explicit i32(42)", "str(i32(42))", "42"),
		Entry("explicit u32(42)", "str(u32(42))", "42"),
		Entry("explicit u8(255)", "str(u8(255))", "255"),
		Entry("string literal", `str("hello")`, "hello"),
	)

	It(
		"Writes str(3.1) to a string channel via interval trigger (matches Console scenario)",
		func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"log_mem": {types.String(), 101},
			})
			h := newRuntimeHarness(ctx, `
			func example_func() {
			    log_mem = str(3.1)
			}
			interval{1s} -> example_func{}`, resolver,
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			h.Tick(ctx, 2*telem.Second)
			h.channelState.ClearReads()
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("3.1"))
		},
	)

	DescribeTable(
		"reads a numeric channel value, converts via str(), and writes the result to a string channel",
		func(ctx SpecContext, arcType string, sensorType types.Type, telemDT telem.DataType, ingestFn func(*runtimeHarness), expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {sensorType, 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`func emit(val `+arcType+`) {
				    log = str(val)
				}
				sensor -> emit{}`, resolver,
				channels.Digest{Key: 100, DataType: telemDT},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			ingestFn(h)
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal(expected))
		},
		Entry(
			"f32 channel 3.1 (ghost precision case)",
			"f32",
			types.F32(),
			telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](3.1)) },
			"3.1",
		),
		Entry(
			"f32 channel 0.1 (ghost precision case)",
			"f32",
			types.F32(),
			telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](0.1)) },
			"0.1",
		),
		Entry(
			"f32 channel 1.0 (integer-valued)",
			"f32",
			types.F32(),
			telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](1.0)) },
			"1",
		),
		Entry(
			"f32 channel 100.000 (trailing zeros)",
			"f32",
			types.F32(),
			telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](100.000)) },
			"100",
		),
		Entry(
			"f32 channel -2.5 (negative)",
			"f32",
			types.F32(),
			telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](-2.5)) },
			"-2.5",
		),
		Entry(
			"f64 channel 3.1",
			"f64",
			types.F64(),
			telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](3.1)) },
			"3.1",
		),
		Entry(
			"f64 channel 0.1234567890123456 (high precision)",
			"f64",
			types.F64(),
			telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](0.1234567890123456)) },
			"0.1234567890123456",
		),
		Entry(
			"f64 channel NaN",
			"f64",
			types.F64(),
			telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](math.NaN())) },
			"NaN",
		),
		Entry(
			"f64 channel +Inf",
			"f64",
			types.F64(),
			telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](math.Inf(1))) },
			"+Inf",
		),
		Entry(
			"f64 channel -Inf",
			"f64",
			types.F64(),
			telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](math.Inf(-1))) },
			"-Inf",
		),
		Entry(
			"i32 channel -42 (negative)",
			"i32",
			types.I32(),
			telem.Int32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](-42)) },
			"-42",
		),
		Entry(
			"u32 channel 4000000000",
			"u32",
			types.U32(),
			telem.Uint32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint32](4000000000)) },
			"4000000000",
		),
	)

	DescribeTable(
		"concatenates str() of a numeric channel value with a string suffix",
		func(ctx SpecContext, arcType string, sensorType types.Type, telemDT telem.DataType, suffix string, ingestFn func(*runtimeHarness), expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {sensorType, 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`func emit(val `+arcType+`) {
				    log = str(val) + "`+suffix+`"
				}
				sensor -> emit{}`, resolver,
				channels.Digest{Key: 100, DataType: telemDT},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			ingestFn(h)
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal(expected))
		},
		Entry(
			"f32 channel 42.5 + ' psi'",
			"f32",
			types.F32(),
			telem.Float32T,
			" psi",
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](42.5)) },
			"42.5 psi",
		),
		Entry(
			"f32 channel 3.1 + ' psi' (ghost precision case)",
			"f32",
			types.F32(),
			telem.Float32T,
			" psi",
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](3.1)) },
			"3.1 psi",
		),
		Entry(
			"f64 channel 3.14 + ' degrees'",
			"f64",
			types.F64(),
			telem.Float64T,
			" degrees",
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](3.14)) },
			"3.14 degrees",
		),
		Entry(
			"i32 channel 42 + ' items'",
			"i32",
			types.I32(),
			telem.Int32T,
			" items",
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](42)) },
			"42 items",
		),
	)

	DescribeTable(
		"concatenates str() of two float channels with a string literal between them",
		func(ctx SpecContext, _, _ string, dt1, dt2 types.Type, telemDT1, telemDT2 telem.DataType, ingestFn func(*runtimeHarness), expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"p1":  {dt1, 100},
				"p2":  {dt2, 101},
				"log": {types.String(), 102},
			})
			h := newRuntimeHarness(ctx,
				`func emit() {
				    log = str(p1) + " some_words " + str(p2)
				}
				interval{50ms} -> emit{}`, resolver,
				channels.Digest{Key: 100, DataType: telemDT1},
				channels.Digest{Key: 101, DataType: telemDT2},
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			ingestFn(h)
			h.Tick(ctx, 75*telem.Millisecond)
			h.channelState.ClearReads()
			out, _ := h.Flush()
			Expect(lastString(out, 102)).To(Equal(expected))
		},
		Entry("f32 42.5 and f32 3.1",
			"f32", "f32", types.F32(), types.F32(), telem.Float32T, telem.Float32T,
			func(h *runtimeHarness) {
				h.Ingest(100, telem.NewSeriesV[float32](42.5))
				h.Ingest(101, telem.NewSeriesV[float32](3.1))
			},
			"42.5 some_words 3.1"),
		Entry("f32 0.1 and f32 100.0 (ghost precision + trailing zeros)",
			"f32", "f32", types.F32(), types.F32(), telem.Float32T, telem.Float32T,
			func(h *runtimeHarness) {
				h.Ingest(100, telem.NewSeriesV[float32](0.1))
				h.Ingest(101, telem.NewSeriesV[float32](100.0))
			},
			"0.1 some_words 100"),
		Entry("f64 3.14 and f64 -2.5",
			"f64", "f64", types.F64(), types.F64(), telem.Float64T, telem.Float64T,
			func(h *runtimeHarness) {
				h.Ingest(100, telem.NewSeriesV[float64](3.14))
				h.Ingest(101, telem.NewSeriesV[float64](-2.5))
			},
			"3.14 some_words -2.5"),
		Entry("f32 3.1 and f64 0.1234567890123456 (mixed precision)",
			"f32", "f64", types.F32(), types.F64(), telem.Float32T, telem.Float64T,
			func(h *runtimeHarness) {
				h.Ingest(100, telem.NewSeriesV[float32](3.1))
				h.Ingest(101, telem.NewSeriesV[float64](0.1234567890123456))
			},
			"3.1 some_words 0.1234567890123456"),
	)

	DescribeTable(
		"uses str() as an inline typecast stage in a flow chain",
		func(ctx SpecContext, source, expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"log_mem": {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`import time
time.interval{50ms} -> `+source+` -> log_mem`, resolver,
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)

			h.Tick(ctx, 75*telem.Millisecond)
			h.channelState.ClearReads()
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal(expected))
		},
		Entry("str(1.234) (literal float)", "str(1.234)", "1.234"),
		Entry("str(3.1) (ghost-precision literal)", "str(3.1)", "3.1"),
		Entry("str(42) (integer literal)", "str(42)", "42"),
		Entry("str(f32(3.14)) (explicit f32)", "str(f32(3.14))", "3.14"),
		Entry(
			"str(f64(0.1234567890123456)) (high-precision f64)",
			"str(f64(0.1234567890123456))",
			"0.1234567890123456",
		),
		Entry(`str("hello") (string literal no-op)`, `str("hello")`, "hello"),
	)

	DescribeTable(
		"uses str() as a flow stage to convert a numeric channel to a string channel",
		func(ctx SpecContext, sensorType types.Type, telemDT telem.DataType, ingestFn func(*runtimeHarness), expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {sensorType, 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`sensor -> str(sensor) -> log`, resolver,
				channels.Digest{Key: 100, DataType: telemDT},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			ingestFn(h)
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal(expected))
		},
		Entry(
			"f32 channel 3.1",
			types.F32(),
			telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](3.1)) },
			"3.1",
		),
		Entry(
			"f64 channel -2.5",
			types.F64(),
			telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](-2.5)) },
			"-2.5",
		),
		Entry(
			"i32 channel -42",
			types.I32(),
			telem.Int32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](-42)) },
			"-42",
		),
		Entry(
			"u8 channel 255",
			types.U8(),
			telem.Uint8T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](255)) },
			"255",
		),
	)

	DescribeTable(
		"concatenates a prefix and suffix around str() of a literal in a flow chain",
		func(ctx SpecContext, expr, expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"log_mem": {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`import time
time.interval{50ms} -> `+expr+` -> log_mem`, resolver,
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Tick(ctx, 75*telem.Millisecond)
			h.channelState.ClearReads()
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal(expected))
		},
		Entry(`"prefix " + str(1.234) + " suffix"`,
			`"prefix " + str(1.234) + " suffix"`, "prefix 1.234 suffix"),
		Entry(`"value=" + str(42)`,
			`"value=" + str(42)`, "value=42"),
		Entry(`str(3.14) + " radians"`,
			`str(3.14) + " radians"`, "3.14 radians"),
		Entry(`"[" + str(f32(0.1)) + "]" (ghost-precision in middle)`,
			`"[" + str(f32(0.1)) + "]"`, "[0.1]"),
	)

	DescribeTable(
		"concatenates a prefix and suffix around str() of a channel value in a flow chain",
		func(ctx SpecContext, sensorType types.Type, telemDT telem.DataType, ingestFn func(*runtimeHarness), expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {sensorType, 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`sensor -> "prefix " + str(sensor) + " suffix" -> log`, resolver,
				channels.Digest{Key: 100, DataType: telemDT},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			ingestFn(h)
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal(expected))
		},
		Entry("f32 channel 3.1", types.F32(), telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](3.1)) },
			"prefix 3.1 suffix"),
		Entry(
			"f32 channel 100.0 (trailing zeros stripped)",
			types.F32(),
			telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](100.0)) },
			"prefix 100 suffix",
		),
		Entry(
			"f64 channel 0.1234567890123456 (high precision)",
			types.F64(),
			telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](0.1234567890123456)) },
			"prefix 0.1234567890123456 suffix",
		),
		Entry("i32 channel -42 (negative)", types.I32(), telem.Int32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](-42)) },
			"prefix -42 suffix"),
		Entry(
			"u32 channel 4000000000 (large unsigned)",
			types.U32(),
			telem.Uint32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint32](4000000000)) },
			"prefix 4000000000 suffix",
		),
		Entry("u8 channel 255 (max byte)", types.U8(), telem.Uint8T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](255)) },
			"prefix 255 suffix"),
	)
})
