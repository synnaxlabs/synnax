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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
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
		Entry("is_inf of NaN", "math.is_inf(v)", nan, false),
		Entry("is_inf of +Inf", "math.is_inf(v)", inf, true),
		Entry("is_inf of -Inf", "math.is_inf(v)", -inf, true),
		Entry("is_inf of a number", "math.is_inf(v)", 1.5, false),
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
		Entry("-math.inf is infinite", "math.is_inf(-math.inf)", 0.0, true),
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
		Entry("is_inf of an integer literal", "math.is_inf(3)", 0.0, false),
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

	It("Should write a constant to a channel from a flow", func(ctx SpecContext) {
		h := newRuntimeHarness(ctx, `
		import math
		x -> -math.inf -> low
		x -> math.nan -> bad`, channelSymbols(map[string]channelDef{
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

	Describe("Compile errors", func() {
		compile := func(ctx SpecContext, source string) error {
			GinkgoHelper()
			root := symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
				Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.F64()),
			}))
			_, err := arc.CompileText(
				ctx,
				arc.Text{Raw: "import math\n" + source},
				root,
			)
			return err
		}

		DescribeTable("Should keep constants out of IR values",
			func(ctx SpecContext, source, message string) {
				Expect(compile(ctx, source)).To(MatchError(ContainSubstring(message)))
			},
			Entry(
				"flow variable read in a func",
				"limit := math.inf\nfunc f() {v f64} f64 { return v + limit }\nx -> f()",
				"not a compile-time constant",
			),
			Entry(
				"stateful flow variable",
				"sequence main {\nlimit f64 $= math.inf\nstage a {\nx > limit => next\n}\n}",
				"initializer must be a literal",
			),
			Entry("function config",
				"func f(a f64) {v f64} f64 { return v + a }\nx -> f(a=math.inf)",
				"must be a literal"),
		)
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
