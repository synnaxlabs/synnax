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
	"encoding/json"
	"math"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("NaN and infinity", func() {
	var (
		nan = math.NaN()
		inf = math.Inf(1)
	)

	importFor := func(expr string) string {
		if strings.Contains(expr, "math.") {
			return "import math"
		}
		return ""
	}

	// evaluate runs expr over v in a func and returns the bool result.
	evaluate := func(
		ctx SpecContext,
		dt types.Type,
		dataType telem.DataType,
		expr string,
		in telem.Series,
	) bool {
		GinkgoHelper()
		h := newRuntimeHarness(ctx, importFor(expr)+`
		func check() {v `+dt.String()+`} bool {
			return `+expr+`
		}
		x -> check() -> out`, channelSymbols(map[string]channelDef{
			"x":   {dt, 100},
			"out": {types.Bool(), 200},
		}),
			channels.Digest{Key: 100, DataType: dataType},
			channels.Digest{Key: 200, DataType: telem.BooleanT},
		)
		defer h.Close(ctx)
		h.Ingest(100, in)
		h.Tick(ctx, telem.Millisecond)
		out, changed := h.Flush()
		Expect(changed).To(BeTrue())
		series := out.Get(200).Series
		Expect(series).To(HaveLen(1))
		return series[0].Unmarshal[bool]()[0]
	}

	DescribeTable(
		"Should give the same result for f32 and f64",
		func(ctx SpecContext, expr string, value float64, expected bool) {
			Expect(evaluate(
				ctx, types.F64(), telem.Float64T, expr, telem.NewSeriesV(value),
			)).To(Equal(expected), "f64")
			Expect(evaluate(
				ctx, types.F32(), telem.Float32T, expr,
				telem.NewSeriesV(float32(value)),
			)).To(Equal(expected), "f32")
		},
		Entry("is_nan of NaN", "math.is_nan(v)", nan, true),
		Entry("is_nan of +Inf", "math.is_nan(v)", inf, false),
		Entry("is_nan of a number", "math.is_nan(v)", 1.5, false),
		Entry("is_infinite of NaN", "math.is_infinite(v)", nan, false),
		Entry("is_infinite of +Inf", "math.is_infinite(v)", inf, true),
		Entry("is_infinite of -Inf", "math.is_infinite(v)", -inf, true),
		Entry("is_infinite of a number", "math.is_infinite(v)", 1.5, false),
		Entry("is_finite of NaN", "math.is_finite(v)", nan, false),
		Entry("is_finite of +Inf", "math.is_finite(v)", inf, false),
		Entry("is_finite of -Inf", "math.is_finite(v)", -inf, false),
		Entry("is_finite of a number", "math.is_finite(v)", 1.5, true),
		Entry("v != v of NaN", "v != v", nan, true),
		Entry("v != v of +Inf", "v != v", inf, false),
		Entry("v != v of a number", "v != v", 1.5, false),
		Entry("math.nan literal", "math.is_nan(math.nan)", 0.0, true),
		Entry("v == math.inf", "v == math.inf", inf, true),
		Entry("v == -math.inf", "v == -math.inf", -inf, true),
		Entry("math.inf first", "math.inf == v", inf, true),
		Entry("v < math.inf", "v < math.inf", 1.5, true),
		Entry("-math.inf is infinite", "math.is_infinite(-math.inf)", 0.0, true),
		Entry(
			"math.inf in arithmetic",
			"math.is_nan(v + math.inf - math.inf)",
			1.5,
			true,
		),
		Entry("v == math.nan", "v == math.nan", nan, false),
		Entry("v != math.nan", "v != math.nan", nan, true),
		Entry("is_finite of a float literal", "math.is_finite(1.0)", 0.0, true),
		Entry("is_nan of an integer literal", "math.is_nan(3)", 0.0, false),
		Entry("is_infinite of an integer literal", "math.is_infinite(3)", 0.0, false),
		Entry("is_finite of an integer literal", "math.is_finite(3)", 0.0, true),
		Entry("is_finite of a negated integer", "math.is_finite(-3)", 0.0, true),
		Entry("is_finite of integer arithmetic", "math.is_finite(1 + 2)", 0.0, true),
	)

	It("Should use a constant as f32 and f64 in one program", func(ctx SpecContext) {
		h := newRuntimeHarness(ctx, `
		import math
		func check() {a f32} bool {
			c f32 := -math.inf
			d f64 := math.inf
			last f64 $= math.nan
			prior := last
			last = f64(a)
			return a > c and f64(a) < d and math.is_nan(prior)
		}
		a -> check() -> out`, channelSymbols(map[string]channelDef{
			"a":   {types.F32(), 100},
			"out": {types.Bool(), 200},
		}),
			channels.Digest{Key: 100, DataType: telem.Float32T},
			channels.Digest{Key: 200, DataType: telem.BooleanT},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV[float32](1))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		Expect(out.Get(200).Series[0].Unmarshal[bool]()[0]).To(BeTrue())
	})

	It("Should store constants in an f32 series", func(ctx SpecContext) {
		h := newRuntimeHarness(ctx, `
		import math
		func check() {v f32} bool {
			data series f32 := [math.inf, -math.inf, math.nan]
			return data[0] > v and data[1] < v and math.is_nan(data[2])
		}
		x -> check() -> out`, channelSymbols(map[string]channelDef{
			"x":   {types.F32(), 100},
			"out": {types.Bool(), 200},
		}),
			channels.Digest{Key: 100, DataType: telem.Float32T},
			channels.Digest{Key: 200, DataType: telem.BooleanT},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV[float32](1))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		Expect(out.Get(200).Series[0].Unmarshal[bool]()[0]).To(BeTrue())
	})

	DescribeTable("Should send a bare constant through a flow",
		func(ctx SpecContext, dt types.Type, dataType telem.DataType) {
			h := newRuntimeHarness(ctx, `
			import math
			x -> math.nan -> bad
			x -> math.inf -> high
			math.inf -> first`, channelSymbols(map[string]channelDef{
				"x":     {dt, 100},
				"bad":   {dt, 200},
				"high":  {dt, 201},
				"first": {dt, 202},
			}),
				channels.Digest{Key: 100, DataType: dataType},
				channels.Digest{Key: 200, DataType: dataType},
				channels.Digest{Key: 201, DataType: dataType},
				channels.Digest{Key: 202, DataType: dataType},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesFromAny(1.0, dataType))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			value := func(key uint32) float64 {
				s := out.Get(key).Series[0]
				if dataType == telem.Float32T {
					return float64(s.Unmarshal[float32]()[0])
				}
				return s.Unmarshal[float64]()[0]
			}
			Expect(math.IsNaN(value(200))).To(BeTrue())
			Expect(math.IsInf(value(201), 1)).To(BeTrue())
			Expect(math.IsInf(value(202), 1)).To(BeTrue())
		},
		Entry("f32", types.F32(), telem.Float32T),
		Entry("f64", types.F64(), telem.Float64T),
	)

	DescribeTable("Should resolve module members in a flow",
		func(ctx SpecContext, source, message string) {
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
				Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.F64()),
			}, &symbol.Symbol{
				Name: "y",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   1,
			}))
			_, err := arc.CompileText(
				ctx,
				arc.Text{Raw: "import math\n" + source},
				root,
			)
			Expect(err).To(MatchError(ContainSubstring(message)))
		},
		Entry("constant as a sink", "x -> math.nan", "math.nan is not a channel"),
		Entry("unknown member", "x -> math.foo -> y", "undefined symbol: foo"),
	)

	It("Should resolve a bare channel named like a constant", func(ctx SpecContext) {
		h := newRuntimeHarness(ctx, `
		import math
		nan -> y
		nan -> math.inf -> z`, channelSymbols(map[string]channelDef{
			"nan": {types.F64(), 100},
			"y":   {types.F64(), 200},
			"z":   {types.F64(), 201},
		}),
			channels.Digest{Key: 100, DataType: telem.Float64T},
			channels.Digest{Key: 200, DataType: telem.Float64T},
			channels.Digest{Key: 201, DataType: telem.Float64T},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV(2.5))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		Expect(out.Get(200).Series[0].Unmarshal[float64]()[0]).To(Equal(2.5))
	})

	It("Should write a constant to a channel from a flow", func(ctx SpecContext) {
		h := newRuntimeHarness(ctx, `
		import math
		x -> -math.inf -> low
		x -> (math.nan) -> bad`, channelSymbols(map[string]channelDef{
			"x":   {types.F32(), 100},
			"low": {types.F32(), 200},
			"bad": {types.F32(), 201},
		}),
			channels.Digest{Key: 100, DataType: telem.Float32T},
			channels.Digest{Key: 200, DataType: telem.Float32T},
			channels.Digest{Key: 201, DataType: telem.Float32T},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV[float32](1))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		low := out.Get(200).Series[0].Unmarshal[float32]()[0]
		Expect(math.IsInf(float64(low), -1)).To(BeTrue())
		bad := out.Get(201).Series[0].Unmarshal[float32]()[0]
		Expect(math.IsNaN(float64(bad))).To(BeTrue())
	})

	DescribeTable("Should mix constants and literals in a series",
		func(ctx SpecContext, dt types.Type, body string, last int, expected float64) {
			h := newRuntimeHarness(ctx, `
			import math
			func check() {v f64} f64 {
				data series `+dt.String()+` := `+body+`
				return f64(data[`+strconv.Itoa(last)+`])
			}
			x -> check() -> out`, channelSymbols(map[string]channelDef{
				"x":   {types.F64(), 100},
				"out": {types.F64(), 200},
			}),
				channels.Digest{Key: 100, DataType: telem.Float64T},
				channels.Digest{Key: 200, DataType: telem.Float64T},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV(1.0))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(200).Series[0].Unmarshal[float64]()[0]).To(Equal(expected))
		},
		Entry("integer then inf, f64", types.F64(), "[1, math.inf]", 1, inf),
		Entry("integer then inf, f32", types.F32(), "[1, math.inf]", 1, inf),
		Entry("nan then integer, f64", types.F64(), "[math.nan, 2]", 1, 2.0),
		Entry("nan then integer, f32", types.F32(), "[math.nan, 2]", 1, 2.0),
		Entry("float, inf, integer, f64", types.F64(), "[1.5, math.inf, 3]", 1, inf),
		Entry("float, inf, integer, f32", types.F32(), "[1.5, math.inf, 3]", 1, inf),
		Entry("negated inf first, f64", types.F64(), "[-math.inf, 1]", 0, -inf),
	)

	Describe("Mixed float and integer literals", func() {
		run := func(ctx SpecContext, dt types.Type, expr string) telem.Series {
			GinkgoHelper()
			dataType := telem.Float64T
			if dt.Kind == types.KindF32 {
				dataType = telem.Float32T
			}
			h := newRuntimeHarness(ctx, `
			func check() {v `+dt.String()+`} `+dt.String()+` {
				r `+dt.String()+` := `+expr+`
				return r
			}
			x -> check() -> out`, channelSymbols(map[string]channelDef{
				"x":   {dt, 100},
				"out": {dt, 200},
			}),
				channels.Digest{Key: 100, DataType: dataType},
				channels.Digest{Key: 200, DataType: dataType},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesFromAny(1.0, dataType))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			return out.Get(200).Series[0]
		}

		DescribeTable("Should give the exact f64 value",
			func(ctx SpecContext, expr string, expected float64) {
				Expect(
					run(ctx, types.F64(), expr).Unmarshal[float64]()[0],
				).To(Equal(expected))
			},
			Entry("float first", "3.2 + 2", 5.2),
			Entry("integer first", "2 + 3.2", 5.2),
			Entry("three operands", "2 + 3.2 + 1", 6.2),
		)

		DescribeTable("Should give the exact f32 value",
			func(ctx SpecContext, expr string, expected float32) {
				Expect(
					run(ctx, types.F32(), expr).Unmarshal[float32]()[0],
				).To(Equal(expected))
			},
			Entry("float first", "3.2 + 2", float32(5.2)),
			Entry("integer first", "2 + 3.2", float32(5.2)),
			Entry("three operands", "2 + 3.2 + 1", float32(6.2)),
		)

		DescribeTable("Should infer f64 without a declared type",
			func(ctx SpecContext, expr string, expected float64) {
				h := newRuntimeHarness(ctx, `
				func check() {v f64} f64 {
					r := `+expr+`
					return r
				}
				x -> check() -> out`, channelSymbols(map[string]channelDef{
					"x":   {types.F64(), 100},
					"out": {types.F64(), 200},
				}),
					channels.Digest{Key: 100, DataType: telem.Float64T},
					channels.Digest{Key: 200, DataType: telem.Float64T},
				)
				defer h.Close(ctx)
				h.Ingest(100, telem.NewSeriesV(1.0))
				h.Tick(ctx, telem.Millisecond)
				out, _ := h.Flush()
				Expect(
					out.Get(200).Series[0].Unmarshal[float64]()[0],
				).To(Equal(expected))
			},
			Entry("float first", "3.2 + 2", 5.2),
			Entry("integer first", "2 + 3.2", 5.2),
			Entry("three operands", "2 + 3.2 + 1", 6.2),
		)
	})

	Describe("Equality of two NaN channels", func() {
		compare := func(
			ctx SpecContext,
			dt types.Type,
			dataType telem.DataType,
			expr string,
			a, b telem.Series,
		) bool {
			GinkgoHelper()
			h := newRuntimeHarness(ctx, `
			func check() {v `+dt.String()+`} bool {
				y := right
				return `+expr+`
			}
			left -> check() -> out`, channelSymbols(map[string]channelDef{
				"left":  {dt, 100},
				"right": {dt, 101},
				"out":   {types.Bool(), 200},
			}),
				channels.Digest{Key: 100, DataType: dataType},
				channels.Digest{Key: 101, DataType: dataType},
				channels.Digest{Key: 200, DataType: telem.BooleanT},
			)
			defer h.Close(ctx)
			h.Ingest(101, b)
			h.Ingest(100, a)
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			return out.Get(200).Series[0].Unmarshal[bool]()[0]
		}

		DescribeTable("Should follow IEEE 754",
			func(ctx SpecContext, expr string, expected bool) {
				Expect(compare(
					ctx, types.F64(), telem.Float64T, expr,
					telem.NewSeriesV(nan), telem.NewSeriesV(nan),
				)).To(Equal(expected), "f64")
				Expect(compare(
					ctx, types.F32(), telem.Float32T, expr,
					telem.NewSeriesV(float32(nan)), telem.NewSeriesV(float32(nan)),
				)).To(Equal(expected), "f32")
			},
			Entry("x == y", "v == y", false),
			Entry("x != y", "v != y", true),
			Entry("x != x", "v != v", true),
			Entry("x == x", "v == v", false),
		)
	})

	Describe("Stored values", func() {
		// run compiles source, ingests a trigger on x, and returns the sample
		// written to out.
		run := func(ctx SpecContext, source string, dt types.Type) float64 {
			GinkgoHelper()
			dataType := telem.Float64T
			if dt.Kind == types.KindF32 {
				dataType = telem.Float32T
			}
			h := newRuntimeHarness(ctx, "import math\n"+source,
				channelSymbols(map[string]channelDef{
					"x":   {dt, 100},
					"out": {dt, 200},
				}),
				channels.Digest{Key: 100, DataType: dataType},
				channels.Digest{Key: 200, DataType: dataType},
			)
			defer h.Close(ctx)
			for range 5 {
				h.Ingest(100, telem.NewSeriesFromAny(1.5, dataType))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			series := out.Get(200).Series
			Expect(series).ToNot(BeEmpty())
			last := series[len(series)-1]
			if dt.Kind == types.KindF32 {
				return float64(last.Unmarshal[float32]()[0])
			}
			return last.Unmarshal[float64]()[0]
		}
		sources := []struct{ name, text string }{
			{"a flow variable read in a func", `
			limit T := V
			func f() {v T} T { return limit }
			x -> f() -> out`},
			{"a flow variable in a sequence", `
			sequence main {
				limit T := V
				stage a {
					limit -> out
				}
			}
			x => main`},
			{"a stateful variable in a sequence", `
			sequence main {
				limit T $= V
				stage a {
					limit -> out
				}
			}
			x => main`},
			{"a function input value", `
			func f(a T) {v T} T { return a }
			x -> f(a=V) -> out`},
		}
		for _, source := range sources {
			DescribeTable(
				"Should keep the value of "+source.name,
				func(ctx SpecContext, dt types.Type, constant string, expected float64) {
					replacer := strings.NewReplacer("T", dt.String(), "V", constant)
					got := run(ctx, replacer.Replace(source.text), dt)
					if math.IsNaN(expected) {
						Expect(math.IsNaN(got)).To(BeTrue())
						return
					}
					Expect(got).To(Equal(expected))
				},
				Entry("f32 NaN", types.F32(), "math.nan", nan),
				Entry("f32 +Inf", types.F32(), "math.inf", inf),
				Entry("f32 -Inf", types.F32(), "-math.inf", -inf),
				Entry("f64 NaN", types.F64(), "math.nan", nan),
				Entry("f64 +Inf", types.F64(), "math.inf", inf),
				Entry("f64 -Inf", types.F64(), "-math.inf", -inf),
			)
		}

		It("Should encode every stored value for JSON", func(ctx SpecContext) {
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
				Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.F64()),
			}))
			prog := MustSucceed(arc.CompileText(ctx, arc.Text{Raw: `
			import math
			func f(a f64) {v f64} f64 { return a }
			x -> f(a=-math.inf)`}, root))
			Expect(MustSucceed(json.Marshal(prog.IR.Nodes))).To(
				ContainSubstring(`"-Infinity"`),
			)
		})

		It("Should reject a constant for an integer input", func(ctx SpecContext) {
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
				Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.F64()),
			}))
			Expect(arc.CompileText(ctx, arc.Text{Raw: `
			import math
			func f(a i32) {v f64} f64 { return v }
			x -> f(a=math.inf)`}, root)).Error().To(
				MatchError(ContainSubstring("not compatible with i32")),
			)
		})
	})

	Describe("Stage transitions", func() {
		// logged runs a stage guarded by guard, feeds it speed, and returns whether
		// the transition fired.
		logged := func(ctx SpecContext, guard string, speed float64) bool {
			GinkgoHelper()
			h := newRuntimeHarness(ctx, importFor(guard)+`
			sequence main {
				stage wait {
					`+guard+` => done
				}
				stage done {
					"passed" -> log
				}
			}
			start_cmd => main`, channelSymbols(map[string]channelDef{
				"start_cmd": {types.U8(), 100},
				"x":         {types.F64(), 101},
				"log":       {types.String(), 102},
			}),
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 101, DataType: telem.Float64T},
				channels.Digest{Key: 102, DataType: telem.StringT},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[uint8](1))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			h.Ingest(101, telem.NewSeriesV(speed))
			for range 5 {
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			return len(out.Get(102).Series) > 0
		}

		const guarded = "not math.is_finite(x) or x < 15"

		DescribeTable("Should fire on the value", func(
			ctx SpecContext,
			guard string,
			speed float64,
			expected bool,
		) {
			Expect(logged(ctx, guard, speed)).To(Equal(expected))
		},
			Entry("guarded with NaN", guarded, nan, true),
			Entry("guarded with +Inf", guarded, inf, true),
			Entry("guarded under the limit", guarded, 10.0, true),
			Entry("guarded over the limit", guarded, 20.0, false),
			Entry("unguarded with NaN", "x < 15", nan, false),
			Entry("is_nan call with NaN", "math.is_nan(x)", nan, true),
			Entry("is_nan call with a number", "math.is_nan(x)", 1.5, false),
		)
	})
})
