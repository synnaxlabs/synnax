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
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("backtick format-string end-to-end runtime", func() {
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
			channel.Digest{Key: 100, DataType: telem.Uint8T},
			channel.Digest{Key: 101, DataType: telem.StringT},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV[uint8](1))
		for i := 0; i < 5; i++ {
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
			channel.Digest{Key: 100, DataType: valueDT},
			channel.Digest{Key: 101, DataType: telem.StringT},
		)
		defer h.Close(ctx)
		ingest(h)
		for i := 0; i < 5; i++ {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		return lastString(out, 101)
	}

	compileErrorResolver := func() symbol.CompoundResolver {
		return symbol.CompoundResolver{
			stl.SymbolResolver,
			channelSymbols(map[string]channelDef{
				"trig": {types.U8(), 100},
				"log":  {types.String(), 101},
				"vI32": {types.I32(), 102},
				"vU32": {types.U32(), 103},
				"vF32": {types.F32(), 104},
				"vF64": {types.F64(), 105},
				"vStr": {types.String(), 106},
			}),
		}
	}

	Describe("Literal raw strings (no placeholders)", func() {
		DescribeTable("emits literal text verbatim",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("plain word", "`static`", "static"),
			Entry("single space", "` `", " "),
			Entry("multi-word with punctuation", "`hello, world!`", "hello, world!"),
			Entry("escaped open brace", "`\\{`", "{"),
			Entry("bare close brace is literal", "`}`", "}"),
			Entry("open escaped, close bare around literal", "`\\{x}`", "{x}"),
			Entry("escape mixed with text", "`pre \\{ mid } post`", "pre { mid } post"),
			Entry("embedded double quotes", "`he said \"hi\"`", `he said "hi"`),
			Entry("escaped backtick", "`a\\`b`", "a`b"),
		)
	})

	Describe("Single placeholder, no format spec", func() {
		DescribeTable("renders numeric literal placeholders via str() conversion",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("integer literal", "`the answer is {42}`", "the answer is 42"),
			Entry("negative integer literal", "`negative: {-7}`", "negative: -7"),
			Entry("zero", "`zero: {0}`", "zero: 0"),
			Entry("float literal", "`pi: {3.14}`", "pi: 3.14"),
			Entry("float literal trailing zeros stripped", "`x: {1.0}`", "x: 1"),
			Entry("explicit f32 cast", "`x: {f32(3.14)}`", "x: 3.14"),
			Entry("explicit f64 cast", "`x: {f64(3.14)}`", "x: 3.14"),
			Entry("explicit i32 cast", "`x: {i32(42)}`", "x: 42"),
			Entry("explicit u32 cast", "`x: {u32(42)}`", "x: 42"),
			Entry("explicit u8 cast", "`x: {u8(255)}`", "x: 255"),
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
				Expect(runFmtChannel(ctx, "`x={val}`", arcType, valueType, valueDT, ingest)).
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
				    log = `+"`hello, {name}`"+`
				}
				trig -> f{}`, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[uint8](1))
			for i := 0; i < 5; i++ {
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
			Entry("decimal", "`{i32(42):d}`", "42"),
			Entry("decimal with width", "`{i32(42):5d}`", "   42"),
			Entry("decimal zero-padded", "`{i32(7):05d}`", "00007"),
			Entry("decimal with sign", "`{i32(42):+d}`", "+42"),
			Entry("hex lower", "`{i32(255):x}`", "ff"),
			Entry("hex upper", "`{i32(255):X}`", "FF"),
			Entry("hex zero-padded", "`{i32(255):04x}`", "00ff"),
			Entry("octal", "`{i32(8):o}`", "10"),
			Entry("binary", "`{i32(5):b}`", "101"),
			Entry("negative decimal", "`{i32(-42):d}`", "-42"),
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
			Entry("i32 channel :05d", "`{val:05d}`", "i32", types.I32(), telem.Int32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](7)) }, "00007"),
			Entry("i32 channel :+d", "`{val:+d}`", "i32", types.I32(), telem.Int32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int32](42)) }, "+42"),
			Entry("i64 channel :d", "`{val:d}`", "i64", types.I64(), telem.Int64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int64](1700000000)) }, "1700000000"),
			Entry("i64 channel :x", "`{val:x}`", "i64", types.I64(), telem.Int64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[int64](255)) }, "ff"),
		)

		DescribeTable("unsigned integer types format with valid Go fmt verbs",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("u8 decimal", "`{u8(255):d}`", "255"),
			Entry("u8 hex", "`{u8(255):x}`", "ff"),
			Entry("u8 binary", "`{u8(255):b}`", "11111111"),
			Entry("u32 decimal", "`{u32(4000000000):d}`", "4000000000"),
			Entry("u32 hex zero-padded", "`{u32(255):08x}`", "000000ff"),
			Entry("u32 octal", "`{u32(8):o}`", "10"),
			Entry("u64 decimal", "`{u64(12345):d}`", "12345"),
			Entry("u64 hex", "`{u64(255):x}`", "ff"),
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
			Entry("u8 channel :d", "`{val:d}`", "u8", types.U8(), telem.Uint8T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](255)) }, "255"),
			Entry("u8 channel :x (promoted to u32)", "`{val:x}`", "u8", types.U8(), telem.Uint8T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint8](255)) }, "ff"),
			Entry("u16 channel :d", "`{val:d}`", "u16", types.U16(), telem.Uint16T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint16](65000)) }, "65000"),
			Entry("u32 channel :X", "`{val:X}`", "u32", types.U32(), telem.Uint32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint32](255)) }, "FF"),
			Entry("u64 channel :x", "`{val:x}`", "u64", types.U64(), telem.Uint64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[uint64](255)) }, "ff"),
		)

		DescribeTable("float types format with valid Go fmt verbs",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("f64 fixed default", "`{f64(3.14159):f}`", "3.141590"),
			Entry("f64 fixed 2 decimals", "`{f64(3.14159):.2f}`", "3.14"),
			Entry("f64 fixed 4 decimals", "`{f64(3.14159):.4f}`", "3.1416"),
			Entry("f64 fixed width.precision", "`{f64(3.14):8.3f}`", "   3.140"),
			Entry("f64 fixed 0 decimals", "`{f64(3.7):.0f}`", "4"),
			Entry("f64 scientific lower", "`{f64(12345.678):e}`", "1.234568e+04"),
			Entry("f64 scientific upper", "`{f64(12345.678):E}`", "1.234568E+04"),
			Entry("f64 general lower", "`{f64(0.000123):g}`", "0.000123"),
			Entry("f64 general upper", "`{f64(0.000123):G}`", "0.000123"),
			Entry("f32 fixed 2 decimals", "`{f32(3.14159):.2f}`", "3.14"),
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
			Entry("f32 channel :.2f", "`{val:.2f}`", "f32", types.F32(), telem.Float32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](3.14159)) }, "3.14"),
			Entry("f32 channel :e", "`{val:e}`", "f32", types.F32(), telem.Float32T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float32](12345.678)) }, "1.234568e+04"),
			Entry("f64 channel :.4f", "`{val:.4f}`", "f64", types.F64(), telem.Float64T,
				func(h *runtimeHarness) { h.Ingest(100, telem.NewSeriesV[float64](3.14159)) }, "3.1416"),
			Entry("f64 channel :g", "`{val:g}`", "f64", types.F64(), telem.Float64T,
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
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[uint8](1))
			for i := 0; i < 5; i++ {
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
			Entry("#b alternate-form binary", "", "`{i32(5):#b}`", "0b101"),
			Entry("#o alternate-form octal", "", "`{i32(8):#o}`", "010"),
			Entry("#x alternate-form hex", "", "`{i32(255):#x}`", "0xff"),
			Entry("X uppercase hex", "", "`{i32(255):X}`", "FF"),
			Entry("E uppercase scientific", "", "`{f64(3.14):E}`", "3.140000E+00"),
			Entry("G uppercase compact", "", "`{f64(3.14):G}`", "3.14"),
			Entry("+d signed decimal", "", "`{i32(42):+d}`", "+42"),
			Entry("space d leading space", "", "`{i32(42): d}`", " 42"),
			Entry("5d width", "", "`{i32(42):5d}`", "   42"),
			Entry("-5d left-aligned width", "", "`{i32(42):-5d}`", "42   "),
			Entry("05d zero-padded width", "", "`{i32(42):05d}`", "00042"),
			Entry("5s string width", `name := "ok"`, "`{name:5s}`", "   ok"),
			Entry("-5s left-aligned string width", `name := "ok"`, "`{name:-5s}`", "ok   "),
			Entry(".2f float precision", "", "`{f64(3.14159):.2f}`", "3.14"),
			Entry("+f signed float", "", "`{f64(3.14):+f}`", "+3.140000"),
			Entry("6.2f width and precision", "", "`{f64(3.14):6.2f}`", "  3.14"),
			Entry("+08.2f sign zero-pad width precision", "", "`{f64(3.14):+08.2f}`", "+0003.14"),
			Entry("#06x alternate-form zero-pad hex", "", "`{i32(255):#06x}`", "0x0000ff"),
		)
	})

	Describe("Multiple placeholders and interleaved escapes", func() {
		DescribeTable("multi-segment concat chains",
			func(ctx SpecContext, source, expected string) {
				Expect(runFmtTrigger(ctx, source)).To(Equal(expected))
			},
			Entry("two placeholders with literal between",
				"`pre {1} mid {2} post`", "pre 1 mid 2 post"),
			Entry("adjacent placeholders no separator",
				"`{1}{2}`", "12"),
			Entry("placeholder at start",
				"`{42} trailing`", "42 trailing"),
			Entry("placeholder at end",
				"`leading {42}`", "leading 42"),
			Entry("escapes interleaved with placeholders",
				"`\\{ {7} }`", "{ 7 }"),
			Entry("three placeholders with mixed specs",
				"`{1}, {i32(2):05d}, {f64(3.14):.2f}`", "1, 00002, 3.14"),
			Entry("escaped open and bare close around placeholder",
				"`\\{{42}}`", "{42}"),
		)
	})

	Describe("Flow-form synthetic functions", func() {
		It("synthesizes a fmt$ function for a raw string with placeholders in flow form", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"sensor": {types.F32(), 100},
				"log":    {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				"sensor -> `v={sensor}` -> log", resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[float32](3.14))
			for i := 0; i < 5; i++ {
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
				"sensor -> `v={sensor:.2f}` -> log", resolver,
				channel.Digest{Key: 100, DataType: telem.Float64T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[float64](3.14159))
			for i := 0; i < 5; i++ {
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
				"sensor -> `v={sensor} t={t}` -> log", resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 102, DataType: telem.Int32T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(102, telem.NewSeriesV[int32](7))
			h.Ingest(100, telem.NewSeriesV[float32](3.14))
			for i := 0; i < 5; i++ {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("v=3.14 t=7"))
		})

		It("flows a literal raw string (no placeholders) without synthesizing a function", func(ctx SpecContext) {
			resolver := channelSymbols(map[string]channelDef{
				"trig": {types.U8(), 100},
				"log":  {types.String(), 101},
			})
			h := newRuntimeHarness(ctx,
				"trig -> `static` -> log", resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[uint8](1))
			for i := 0; i < 5; i++ {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			Expect(lastString(out, 101)).To(Equal("static"))
		})
	})

	Describe("Compile-time diagnostics for invalid placeholder usage", func() {
		DescribeTable("rejects invalid format spec on a string-typed placeholder",
			func(ctx SpecContext, source, errSubstr string) {
				program := `func f() {
				    name := "probe"
				    log = ` + source + `
				}
				trig -> f{}`
				Expect(arc.CompileText(ctx, arc.Text{Raw: program}, arc.WithResolver(compileErrorResolver()))).
					Error().To(MatchError(ContainSubstring(errSubstr)))
			},
			Entry("string with :d spec", "`{name:d}`", "invalid format spec"),
			Entry("string with :.2f spec", "`{name:.2f}`", "invalid format spec"),
		)

		DescribeTable("rejects float-only specs on integer-typed placeholders",
			func(ctx SpecContext, source, errSubstr string) {
				program := `func f(val i32) {
				    log = ` + source + `
				}
				vI32 -> f{}`
				Expect(arc.CompileText(ctx, arc.Text{Raw: program}, arc.WithResolver(compileErrorResolver()))).
					Error().To(MatchError(ContainSubstring(errSubstr)))
			},
			Entry("i32 with :f", "`{val:f}`", "invalid format spec"),
			Entry("i32 with :.2f", "`{val:.2f}`", "invalid format spec"),
			Entry("i32 with :e", "`{val:e}`", "invalid format spec"),
			Entry("i32 with :g", "`{val:g}`", "invalid format spec"),
		)

		DescribeTable("rejects integer-only specs on float-typed placeholders",
			func(ctx SpecContext, source, errSubstr string) {
				program := `func f(val f32) {
				    log = ` + source + `
				}
				vF32 -> f{}`
				Expect(arc.CompileText(ctx, arc.Text{Raw: program}, arc.WithResolver(compileErrorResolver()))).
					Error().To(MatchError(ContainSubstring(errSubstr)))
			},
			Entry("f32 with :d", "`{val:d}`", "invalid format spec"),
			Entry("f32 with :o", "`{val:o}`", "invalid format spec"),
		)

		DescribeTable("rejects unsupported placeholder types",
			func(ctx SpecContext, source, errSubstr string) {
				program := `func f() {
				    log = ` + source + `
				}
				trig -> f{}`
				Expect(arc.CompileText(ctx, arc.Text{Raw: program}, arc.WithResolver(compileErrorResolver()))).
					Error().To(MatchError(ContainSubstring(errSubstr)))
			},
			Entry("undefined identifier", "`{undeclared}`", "undeclared"),
		)

		DescribeTable("rejects malformed format-string bodies",
			func(ctx SpecContext, source, errSubstr string) {
				program := `func f() {
				    log = ` + source + `
				}
				trig -> f{}`
				Expect(arc.CompileText(ctx, arc.Text{Raw: program}, arc.WithResolver(compileErrorResolver()))).
					Error().To(MatchError(ContainSubstring(errSubstr)))
			},
			Entry("unterminated placeholder", "`{x`", "unmatched"),
			Entry("empty placeholder body", "`{}`", "must contain"),
			Entry("empty spec after percent", "`{x:}`", "format spec"),
		)
	})
})
