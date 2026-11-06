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
	"github.com/synnaxlabs/arc/stl/channels"
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
		Entry("is_finite of a float literal", "math.is_finite(1.0)", 0.0, true),
		Entry("is_nan of an integer literal", "math.is_nan(3)", 0.0, false),
		Entry("is_infinite of an integer literal", "math.is_infinite(3)", 0.0, false),
		Entry("is_finite of an integer literal", "math.is_finite(3)", 0.0, true),
		Entry("is_finite of a negated integer", "math.is_finite(-3)", 0.0, true),
		Entry("is_finite of integer arithmetic", "math.is_finite(1 + 2)", 0.0, true),
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
