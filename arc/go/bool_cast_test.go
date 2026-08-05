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

var _ = Describe("bool() typecast end-to-end runtime", func() {
	lastBool := func(fr telem.Frame[uint32], key uint32) bool {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[bool](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	DescribeTable(
		"reads a numeric channel value, converts via bool(), and writes the result to a bool channel",
		func(ctx SpecContext, arcType string, sensorType types.Type, telemDT telem.DataType, ingestFn func(*runtimeHarness), expected bool) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {sensorType, 100},
				"out":    {types.Bool(), 101},
			})
			h := newRuntimeHarness(ctx,
				`func emit(val `+arcType+`) {
				    out = bool(val)
				}
				sensor -> emit{}`, resolver,
				channels.Digest{Key: 100, DataType: telemDT},
				channels.Digest{Key: 101, DataType: telem.BoolT},
			)
			defer h.Close(ctx)
			ingestFn(h)
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastBool(out, 101)).To(Equal(expected))
		},
		Entry("u8 0 → false", "u8", types.U8(), telem.Uint8T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](0)) }, false),
		Entry("u8 255 → true", "u8", types.U8(), telem.Uint8T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](255)) }, true),
		Entry("u16 0 → false", "u16", types.U16(), telem.Uint16T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint16](0)) }, false),
		Entry("u16 65535 → true", "u16", types.U16(), telem.Uint16T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint16](65535)) }, true),
		Entry("u32 0 → false", "u32", types.U32(), telem.Uint32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint32](0)) }, false),
		Entry("u32 4000000000 → true", "u32", types.U32(), telem.Uint32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint32](4000000000)) }, true),
		Entry("u64 0 → false", "u64", types.U64(), telem.Uint64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint64](0)) }, false),
		Entry("u64 max → true", "u64", types.U64(), telem.Uint64T,
			func(h *runtimeHarness) {
				h.Ingest(100, telem.NewSeriesV[uint64](18446744073709551615))
			}, true),
		Entry("i8 0 → false", "i8", types.I8(), telem.Int8T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int8](0)) }, false),
		Entry("i8 127 → true", "i8", types.I8(), telem.Int8T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int8](127)) }, true),
		Entry("i8 -128 → true (negative)", "i8", types.I8(), telem.Int8T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int8](-128)) }, true),
		Entry("i16 0 → false", "i16", types.I16(), telem.Int16T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int16](0)) }, false),
		Entry("i16 -32768 → true (negative)", "i16", types.I16(), telem.Int16T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int16](-32768)) }, true),
		Entry("i32 0 → false", "i32", types.I32(), telem.Int32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](0)) }, false),
		Entry("i32 -42 → true (negative)", "i32", types.I32(), telem.Int32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](-42)) }, true),
		Entry("i64 0 → false", "i64", types.I64(), telem.Int64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int64](0)) }, false),
		Entry("i64 -9223372036854775807 → true (negative)", "i64", types.I64(), telem.Int64T,
			func(h *runtimeHarness) {
				h.Ingest(100, telem.NewSeriesV[int64](-9223372036854775807))
			}, true),
		Entry("f32 0.0 → false", "f32", types.F32(), telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](0)) }, false),
		Entry("f32 3.14 → true", "f32", types.F32(), telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](3.14)) }, true),
		Entry("f32 -2.5 → true (negative)", "f32", types.F32(), telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](-2.5)) }, true),
		Entry("f32 -0.0 → false (negative zero)", "f32", types.F32(), telem.Float32T,
			func(h *runtimeHarness) {
				h.Ingest(100, telem.NewSeriesV[float32](float32(math.Copysign(0, -1))))
			}, false),
		Entry("f32 -0.1 → true", "f32", types.F32(), telem.Float32T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](-0.1)) }, true),
		Entry("f64 0.0 → false", "f64", types.F64(), telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](0)) }, false),
		Entry("f64 3.14 → true", "f64", types.F64(), telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](3.14)) }, true),
		Entry("f64 -2.5 → true (negative)", "f64", types.F64(), telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](-2.5)) }, true),
		Entry("f64 -0.0 → false (negative zero)", "f64", types.F64(), telem.Float64T,
			func(h *runtimeHarness) {
				h.Ingest(100, telem.NewSeriesV[float64](math.Copysign(0, -1)))
			}, false),
		Entry("f64 -0.1 → true", "f64", types.F64(), telem.Float64T,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](-0.1)) }, true),
	)

	DescribeTable(
		"widens a bool to a numeric type and writes the result to a numeric channel",
		func(ctx SpecContext, source string, outType types.Type, telemDT telem.DataType, verify func(telem.Series)) {
			resolver := channelSymbols(map[string]channelDef{
				"trig": {types.U8(), 100},
				"out":  {outType, 101},
			})
			h := newRuntimeHarness(ctx,
				`func f() {
				    out = `+source+`
				}
				trig -> f{}`, resolver,
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telemDT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[uint8](1))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			ch := out.Get(101)
			Expect(ch.Series).ToNot(BeEmpty(), "channel 101 not written")
			s := ch.Series[len(ch.Series)-1]
			Expect(s.Len()).To(BeNumerically(">", 0), "channel 101 has no samples")
			verify(s)
		},
		Entry("u8(true) → 1", "u8(true)", types.U8(), telem.Uint8T, func(s telem.Series) {
			v := telem.UnmarshalSeries[uint8](s)
			Expect(v[len(v)-1]).To(Equal(uint8(1)))
		}),
		Entry("u8(false) → 0", "u8(false)", types.U8(), telem.Uint8T, func(s telem.Series) {
			v := telem.UnmarshalSeries[uint8](s)
			Expect(v[len(v)-1]).To(Equal(uint8(0)))
		}),
		Entry("i64(true) → 1", "i64(true)", types.I64(), telem.Int64T, func(s telem.Series) {
			v := telem.UnmarshalSeries[int64](s)
			Expect(v[len(v)-1]).To(Equal(int64(1)))
		}),
		Entry("f32(true) → 1.0", "f32(true)", types.F32(), telem.Float32T, func(s telem.Series) {
			v := telem.UnmarshalSeries[float32](s)
			Expect(v[len(v)-1]).To(Equal(float32(1)))
		}),
		Entry("f64(false) → 0.0", "f64(false)", types.F64(), telem.Float64T, func(s telem.Series) {
			v := telem.UnmarshalSeries[float64](s)
			Expect(v[len(v)-1]).To(Equal(float64(0)))
		}),
	)

	lastString := func(fr telem.Frame[uint32], key uint32) string {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[string](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	DescribeTable(
		"stringifies bools and bool/numeric cast chains via str()",
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
		// Direct bool → str: renders "true"/"false".
		Entry(`str(true) → "true"`, "str(true)", "true"),
		Entry(`str(false) → "false"`, "str(false)", "false"),
		Entry(`str(bool(0)) → "false"`, "str(bool(0))", "false"),
		Entry(`str(bool(5)) → "true"`, "str(bool(5))", "true"),
		Entry(`str(bool(0.0)) → "false"`, "str(bool(0.0))", "false"),
		Entry(`str(bool(-0.1)) → "true"`, "str(bool(-0.1))", "true"),
		Entry(`str(bool(u8(1))) → "true"`, "str(bool(u8(1)))", "true"),
		// Bool widened to numeric first, then stringified: renders "1"/"0",
		// NOT "true"/"false".
		Entry(`str(u8(true)) → "1"`, "str(u8(true))", "1"),
		Entry(`str(u8(false)) → "0"`, "str(u8(false))", "0"),
		Entry(`str(i32(true)) → "1"`, "str(i32(true))", "1"),
		Entry(`str(f64(true)) → "1"`, "str(f64(true))", "1"),
		Entry(`str(f64(false)) → "0"`, "str(f64(false))", "0"),
	)

	DescribeTable(
		"reads a bool channel value, converts via str(), and writes the result to a string channel",
		func(ctx SpecContext, ingestFn func(*runtimeHarness), expected string) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.Bool(), 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`func emit(val bool) {
				    log = str(val)
				}
				sensor -> emit{}`, resolver,
				channels.Digest{Key: 100, DataType: telem.BoolT},
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
		Entry(`bool channel true → "true"`,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[bool](true)) }, "true"),
		Entry(`bool channel false → "false"`,
			func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[bool](false)) }, "false"),
	)
})
