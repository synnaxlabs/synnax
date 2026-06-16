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

var _ = Describe("format-string end-to-end runtime", func() {
	lastString := func(fr telem.Frame[uint32], key uint32) string {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[string](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	runFmtTrigger := func(ctx SpecContext, source string) string {
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
		return lastString(out, 101)
	}

	runFmtChannel := func(
		ctx SpecContext,
		source, arcType string,
		valueType types.Type,
		valueDT telem.DataType,
		ingest func(*runtimeHarness),
	) string {
		resolver := channelSymbols(map[string]channelDef{
			"v":   {valueType, 100},
			"log": {types.String(), 101},
		})
		h := newRuntimeHarness(ctx,
			`func f(val `+arcType+`) {
			    log = `+source+`
			}
			v -> f{}`, resolver,
			channels.Digest{Key: 100, DataType: valueDT},
			channels.Digest{Key: 101, DataType: telem.StringT},
		)
		defer h.Close(ctx)
		ingest(h)
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		return lastString(out, 101)
	}

	Describe("Literal raw strings (no placeholders)", func() {
		DescribeTable("emits literal text verbatim",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("plain word", `f"static"`, "static"),
			Entry("single space", `f" "`, " "),
			Entry("multi-word with punctuation", `f"hello, world!"`, "hello, world!"),
			Entry("doubled open brace", `f"{{"`, "{"),
			Entry("doubled close brace", `f"}}"`, "}"),
			Entry("bare close brace is literal", `f"}"`, "}"),
			Entry("doubled braces around literal", `f"{{x}}"`, "{x}"),
			Entry("doubled braces mixed with text", `f"pre {{ mid }} post"`, "pre { mid } post"),
			Entry("embedded double quotes", `f"he said \"hi\""`, `he said "hi"`),
		)
	})

	Describe("Single placeholder, no format spec", func() {
		DescribeTable("renders numeric literal placeholders via str() conversion",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("integer literal", `f"the answer is {42}"`, "the answer is 42"),
			Entry("negative integer literal", `f"negative: {-7}"`, "negative: -7"),
			Entry("zero", `f"zero: {0}"`, "zero: 0"),
			Entry("float literal", `f"pi: {3.14}"`, "pi: 3.14"),
			Entry("float literal trailing zeros stripped", `f"x: {1.0}"`, "x: 1"),
			Entry("explicit f32 cast", `f"x: {f32(3.14)}"`, "x: 3.14"),
			Entry("explicit f64 cast", `f"x: {f64(3.14)}"`, "x: 3.14"),
			Entry("explicit i32 cast", `f"x: {i32(42)}"`, "x: 42"),
			Entry("explicit u32 cast", `f"x: {u32(42)}"`, "x: 42"),
			Entry("explicit u8 cast", `f"x: {u8(255)}"`, "x: 255"),
		)

		DescribeTable("renders channel value of each numeric type",
			func(
				ctx SpecContext,
				arcType string,
				valueType types.Type,
				valueDT telem.DataType,
				ingest func(*runtimeHarness),
				expected string,
			) {
				Expect(runFmtChannel(ctx, `f"x={val}"`, arcType, valueType, valueDT, ingest)).
					To(Equal(expected))
			},
			Entry("u8 channel", "u8", types.U8(), telem.Uint8T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](7)) }, "x=7"),
			Entry("u16 channel", "u16", types.U16(), telem.Uint16T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint16](7)) }, "x=7"),
			Entry("u32 channel", "u32", types.U32(), telem.Uint32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint32](4000000000)) }, "x=4000000000"),
			Entry("u64 channel", "u64", types.U64(), telem.Uint64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint64](18000000000000000000)) }, "x=18000000000000000000"),
			Entry("i32 channel", "i32", types.I32(), telem.Int32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](-42)) }, "x=-42"),
			Entry("i64 channel", "i64", types.I64(), telem.Int64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int64](-1700000000)) }, "x=-1700000000"),
			Entry("f32 channel", "f32", types.F32(), telem.Float32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](3.14)) }, "x=3.14"),
			Entry("f64 channel high precision", "f64", types.F64(), telem.Float64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](0.1234567890123456)) }, "x=0.1234567890123456"),
		)

		It("renders a string-typed placeholder bound to a local string", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trig": {types.U8(), 100},
				"log":  {types.String(), 101},
			})
			h := newRuntimeHarness(ctx, `
				func f() {
				    name := "probe"
				    log = `+`f"hello, {name}"`+`
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
			Expect(lastString(out, 101)).To(Equal("hello, probe"))
		})
	})

	Describe("Single placeholder with valid format spec", func() {
		DescribeTable("integer types format with valid Go fmt verbs",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("decimal", `f"{i32(42):d}"`, "42"),
			Entry("decimal with width", `f"{i32(42):5d}"`, "   42"),
			Entry("decimal zero-padded", `f"{i32(7):05d}"`, "00007"),
			Entry("decimal with sign", `f"{i32(42):+d}"`, "+42"),
			Entry("hex lower", `f"{i32(255):x}"`, "ff"),
			Entry("hex upper", `f"{i32(255):X}"`, "FF"),
			Entry("hex zero-padded", `f"{i32(255):04x}"`, "00ff"),
			Entry("octal", `f"{i32(8):o}"`, "10"),
			Entry("binary", `f"{i32(5):b}"`, "101"),
			Entry("negative decimal", `f"{i32(-42):d}"`, "-42"),
		)

		DescribeTable("integer channel values format with valid specs (i8/i16/i32/i64 promotion)",
			func(
				ctx SpecContext,
				source, arcType string,
				valueType types.Type,
				valueDT telem.DataType,
				ingest func(*runtimeHarness),
				expected string,
			) {
				Expect(runFmtChannel(ctx, source, arcType, valueType, valueDT, ingest)).
					To(Equal(expected))
			},
			Entry("i32 channel :05d", `f"{val:05d}"`, "i32", types.I32(), telem.Int32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](7)) }, "00007"),
			Entry("i32 channel :+d", `f"{val:+d}"`, "i32", types.I32(), telem.Int32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](42)) }, "+42"),
			Entry("i64 channel :d", `f"{val:d}"`, "i64", types.I64(), telem.Int64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int64](1700000000)) }, "1700000000"),
			Entry("i64 channel :x", `f"{val:x}"`, "i64", types.I64(), telem.Int64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int64](255)) }, "ff"),
		)

		DescribeTable("unsigned integer types format with valid Go fmt verbs",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("u8 decimal", `f"{u8(255):d}"`, "255"),
			Entry("u8 hex", `f"{u8(255):x}"`, "ff"),
			Entry("u8 binary", `f"{u8(255):b}"`, "11111111"),
			Entry("u32 decimal", `f"{u32(4000000000):d}"`, "4000000000"),
			Entry("u32 hex zero-padded", `f"{u32(255):08x}"`, "000000ff"),
			Entry("u32 octal", `f"{u32(8):o}"`, "10"),
			Entry("u64 decimal", `f"{u64(12345):d}"`, "12345"),
			Entry("u64 hex", `f"{u64(255):x}"`, "ff"),
		)

		DescribeTable("unsigned integer channel values format with valid specs (u8/u16/u32/u64 promotion)",
			func(
				ctx SpecContext,
				source, arcType string,
				valueType types.Type,
				valueDT telem.DataType,
				ingest func(*runtimeHarness),
				expected string,
			) {
				Expect(runFmtChannel(ctx, source, arcType, valueType, valueDT, ingest)).
					To(Equal(expected))
			},
			Entry("u8 channel :d", `f"{val:d}"`, "u8", types.U8(), telem.Uint8T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](255)) }, "255"),
			Entry("u8 channel :x (promoted to u32)", `f"{val:x}"`, "u8", types.U8(), telem.Uint8T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](255)) }, "ff"),
			Entry("u16 channel :d", `f"{val:d}"`, "u16", types.U16(), telem.Uint16T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint16](65000)) }, "65000"),
			Entry("u32 channel :X", `f"{val:X}"`, "u32", types.U32(), telem.Uint32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint32](255)) }, "FF"),
			Entry("u64 channel :x", `f"{val:x}"`, "u64", types.U64(), telem.Uint64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint64](255)) }, "ff"),
		)

		DescribeTable("float types format with valid Go fmt verbs",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("f64 fixed default", `f"{f64(3.14159):f}"`, "3.141590"),
			Entry("f64 fixed 2 decimals", `f"{f64(3.14159):.2f}"`, "3.14"),
			Entry("f64 fixed 4 decimals", `f"{f64(3.14159):.4f}"`, "3.1416"),
			Entry("f64 fixed width.precision", `f"{f64(3.14):8.3f}"`, "   3.140"),
			Entry("f64 fixed 0 decimals", `f"{f64(3.7):.0f}"`, "4"),
			Entry("f64 scientific lower", `f"{f64(12345.678):e}"`, "1.234568e+04"),
			Entry("f64 scientific upper", `f"{f64(12345.678):E}"`, "1.234568E+04"),
			Entry("f64 general lower", `f"{f64(0.000123):g}"`, "0.000123"),
			Entry("f64 general upper", `f"{f64(0.000123):G}"`, "0.000123"),
			Entry("f32 fixed 2 decimals", `f"{f32(3.14159):.2f}"`, "3.14"),
		)

		DescribeTable("float channel values format with valid specs",
			func(
				ctx SpecContext,
				source, arcType string,
				valueType types.Type,
				valueDT telem.DataType,
				ingest func(*runtimeHarness),
				expected string,
			) {
				Expect(runFmtChannel(ctx, source, arcType, valueType, valueDT, ingest)).
					To(Equal(expected))
			},
			Entry("f32 channel :.2f", `f"{val:.2f}"`, "f32", types.F32(), telem.Float32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](3.14159)) }, "3.14"),
			Entry("f32 channel :e", `f"{val:e}"`, "f32", types.F32(), telem.Float32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](12345.678)) }, "1.234568e+04"),
			Entry("f64 channel :.4f", `f"{val:.4f}"`, "f64", types.F64(), telem.Float64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](3.14159)) }, "3.1416"),
			Entry("f64 channel :g", `f"{val:g}"`, "f64", types.F64(), telem.Float64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](0.000123)) }, "0.000123"),
		)
	})

	Describe("Documented Examples table (syntax.mdx)", func() {
		runFmtExample := func(ctx SpecContext, declarations, body string) string {
			resolver := channelSymbols(map[string]channelDef{
				"trig": {types.U8(), 100},
				"log":  {types.String(), 101},
			})
			src := `func f() {
			    ` + declarations + `
			    log = ` + body + `
			}
			trig -> f{}`
			h := newRuntimeHarness(ctx, src, resolver,
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
			return lastString(out, 101)
		}
		DescribeTable("each documented spec produces the documented output",
			func(ctx SpecContext, declarations, body, expected string) {
				Expect(runFmtExample(ctx, declarations, body)).To(Equal(expected))
			},
			Entry("#b alternate-form binary", "", `f"{i32(5):#b}"`, "0b101"),
			Entry("#o alternate-form octal", "", `f"{i32(8):#o}"`, "010"),
			Entry("#x alternate-form hex", "", `f"{i32(255):#x}"`, "0xff"),
			Entry("X uppercase hex", "", `f"{i32(255):X}"`, "FF"),
			Entry("E uppercase scientific", "", `f"{f64(3.14):E}"`, "3.140000E+00"),
			Entry("G uppercase compact", "", `f"{f64(3.14):G}"`, "3.14"),
			Entry("+d signed decimal", "", `f"{i32(42):+d}"`, "+42"),
			Entry("space d leading space", "", `f"{i32(42): d}"`, " 42"),
			Entry("5d width", "", `f"{i32(42):5d}"`, "   42"),
			Entry("-5d left-aligned width", "", `f"{i32(42):-5d}"`, "42   "),
			Entry("05d zero-padded width", "", `f"{i32(42):05d}"`, "00042"),
			Entry("5s string width", `name := "ok"`, `f"{name:5s}"`, "   ok"),
			Entry("-5s left-aligned string width", `name := "ok"`, `f"{name:-5s}"`, "ok   "),
			Entry(".2f float precision", "", `f"{f64(3.14159):.2f}"`, "3.14"),
			Entry("+f signed float", "", `f"{f64(3.14):+f}"`, "+3.140000"),
			Entry("6.2f width and precision", "", `f"{f64(3.14):6.2f}"`, "  3.14"),
			Entry("+08.2f sign zero-pad width precision", "", `f"{f64(3.14):+08.2f}"`, "+0003.14"),
			Entry("#06x alternate-form zero-pad hex", "", `f"{i32(255):#06x}"`, "0x0000ff"),
		)
	})

	Describe("Multiple placeholders and interleaved escapes", func() {
		DescribeTable("multi-segment concat chains",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("two placeholders with literal between",
				`f"pre {1} mid {2} post"`, "pre 1 mid 2 post"),
			Entry("adjacent placeholders no separator",
				`f"{1}{2}"`, "12"),
			Entry("placeholder at start",
				`f"{42} trailing"`, "42 trailing"),
			Entry("placeholder at end",
				`f"leading {42}"`, "leading 42"),
			Entry("doubled braces interleaved with placeholders",
				`f"{{ {7} }}"`, "{ 7 }"),
			Entry("three placeholders with mixed specs",
				`f"{1}, {i32(2):05d}, {f64(3.14):.2f}"`, "1, 00002, 3.14"),
			Entry("doubled braces around placeholder",
				`f"{{{42}}}"`, "{42}"),
			Entry("raw string with backslash adjacent to placeholder",
				`rf"C:\logs\{42}.txt"`, `C:\logs\42.txt`),
			Entry("raw string with backslash adjacent and doubled-brace literal",
				`rf"C:\out\{{tag}}-{42}.bin"`, `C:\out\{tag}-42.bin`),
			Entry("raw format string with only doubled braces (no placeholder)",
				`rf"C:\logs\{{abc}}.txt"`, `C:\logs\{abc}.txt`),
		)
	})

	Describe("Flow-form synthetic functions", func() {
		It("synthesizes a fmt$ function for a raw string with placeholders in flow form", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`sensor -> f"v={sensor}" -> log`, resolver,
				channels.Digest{Key: 100, DataType: telem.Float32T},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[float32](3.14))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("v=3.14"))
		})

		It("preserves a numeric format spec on a flow-form synthetic", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F64(), 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`sensor -> f"v={sensor:.2f}" -> log`, resolver,
				channels.Digest{Key: 100, DataType: telem.Float64T},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[float64](3.14159))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("v=3.14"))
		})

		It("synthesizes a fmt$ function for a multi-channel placeholder body in flow form", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"t":      {types.I32(), 102},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`sensor -> f"v={sensor} t={t}" -> log`, resolver,
				channels.Digest{Key: 100, DataType: telem.Float32T},
				channels.Digest{Key: 102, DataType: telem.Int32T},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(102, telem.NewSeriesV[int32](7))
			h.Ingest(100, telem.NewSeriesV[float32](3.14))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("v=3.14 t=7"))
		})

		It("synthesizes a fmt$ function for an rf-prefixed multi-line format string preserving backslashes across newlines", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"t":      {types.I32(), 102},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				"sensor -> rf`path\\to: {sensor}\nt={t}` -> log", resolver,
				channels.Digest{Key: 100, DataType: telem.Float32T},
				channels.Digest{Key: 102, DataType: telem.Int32T},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(102, telem.NewSeriesV[int32](7))
			h.Ingest(100, telem.NewSeriesV[float32](3.14))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("path\\to: 3.14\nt=7"))
		})

		It("synthesizes a fmt$ function for an rf-prefixed format string preserving backslashes", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`sensor -> rf"path\to: {sensor}" -> log`, resolver,
				channels.Digest{Key: 100, DataType: telem.Float32T},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[float32](3.14))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal(`path\to: 3.14`))
		})

		It("synthesizes a fmt$ function for a multi-line format string with placeholders across newlines", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"t":      {types.I32(), 102},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				"sensor -> f`v={sensor}\nt={t}` -> log", resolver,
				channels.Digest{Key: 100, DataType: telem.Float32T},
				channels.Digest{Key: 102, DataType: telem.Int32T},
				channels.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(102, telem.NewSeriesV[int32](7))
			h.Ingest(100, telem.NewSeriesV[float32](3.14))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("v=3.14\nt=7"))
		})

		It("flows a literal raw string (no placeholders) without synthesizing a function", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trig": {types.U8(), 100},
				"log":  {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`trig -> f"static" -> log`, resolver,
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
			Expect(lastString(out, 101)).To(Equal("static"))
		})

		It("flows a raw format string with only doubled-brace literals (no placeholders)", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trig": {types.U8(), 100},
				"log":  {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				`trig -> rf"C:\logs\{{abc}}.txt" -> log`, resolver,
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
			Expect(lastString(out, 101)).To(Equal(`C:\logs\{abc}.txt`))
		})
	})

})
