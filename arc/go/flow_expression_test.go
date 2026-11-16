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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Literal expressions in flows", func() {
	DescribeTable("Should write a literal expression at the sink's type",
		func(ctx SpecContext, source string, dt types.Type, expected float64) {
			dataType := dt.ToTelem()
			h := newRuntimeHarness(ctx, source, channelSymbols(map[string]channelDef{
				"x":   {types.U8(), 100},
				"out": {dt, 200},
			}),
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 200, DataType: dataType},
			)
			defer h.Close(ctx)
			for range 3 {
				h.Ingest(100, telem.NewSeriesV[uint8](1))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			series := out.Get(200).Series
			Expect(series).ToNot(BeEmpty())
			want := telem.NewSeriesFromAny(expected, dataType).Data
			last := series[len(series)-1].Data
			Expect(last[len(last)-len(want):]).To(Equal(want))
		},
		Entry("integers to i32", "x -> 1 + 2 -> out", types.I32(), 3.0),
		Entry("integers to u8", "x -> 1 + 2 -> out", types.U8(), 3.0),
		Entry("floats to f32", "x -> 1.5 + 2.5 -> out", types.F32(), 4.0),
		Entry(
			"integers to i32 in a stage",
			"sequence main {\n stage a {\n 1 + 2 -> out\n }\n}\nx => main",
			types.I32(),
			3.0,
		),
	)

	DescribeTable("Should write a stage reassignment at the variable's type",
		func(ctx SpecContext, decl, value string, dt types.Type, expected float64) {
			dataType := dt.ToTelem()
			h := newRuntimeHarness(ctx, "sequence main {\n "+decl+"\n stage a {\n v = "+
				value+"\n v -> out\n }\n}\ngo => main",
				channelSymbols(map[string]channelDef{
					"go":  {types.Bool(), 100},
					"out": {dt, 200},
				}),
				channels.Digest{Key: 100, DataType: telem.BooleanT},
				channels.Digest{Key: 200, DataType: dataType},
			)
			defer h.Close(ctx)
			for range 3 {
				h.Ingest(100, telem.NewSeriesV(true))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			series := out.Get(200).Series
			Expect(series).ToNot(BeEmpty())
			want := telem.NewSeriesFromAny(expected, dataType).Data
			last := series[len(series)-1].Data
			Expect(last[len(last)-len(want):]).To(Equal(want))
		},
		Entry("i32", "v i32 := 0", "1 + 2", types.I32(), 3.0),
		Entry("f32", "v f32 := 0.0", "1.5 * 2.0", types.F32(), 3.0),
		Entry("stateful f32", "v f32 $= 0.0", "1.5 * 2.0", types.F32(), 3.0),
	)

	DescribeTable("Should write a computed value at its declared type",
		func(ctx SpecContext, source string, dt types.Type, expected float64) {
			dataType := dt.ToTelem()
			h := newRuntimeHarness(ctx, strings.ReplaceAll(source, "T", dt.String()),
				channelSymbols(map[string]channelDef{
					"go":  {types.Bool(), 100},
					"out": {dt, 200},
				}),
				channels.Digest{Key: 100, DataType: telem.BooleanT},
				channels.Digest{Key: 200, DataType: dataType},
			)
			defer h.Close(ctx)
			for range 3 {
				h.Ingest(100, telem.NewSeriesV(true))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			series := out.Get(200).Series
			Expect(series).ToNot(BeEmpty())
			want := telem.NewSeriesFromAny(expected, dataType).Data
			last := series[len(series)-1].Data
			Expect(last[len(last)-len(want):]).To(Equal(want))
		},
		Entry(
			"a stateful initializer",
			"sequence main {\n v T $= 1.5 * 2.0\n stage a {\n v -> out\n }\n}\ngo => main",
			types.F32(),
			3.0,
		),
		Entry(
			"an integer stateful initializer",
			"sequence main {\n v T $= 1 + 2\n stage a {\n v -> out\n }\n}\ngo => main",
			types.U8(),
			3.0,
		),
		Entry(
			"a sequence initializer",
			"sequence main {\n v T := 1.5 * 2.0\n stage a {\n v -> out\n }\n}\ngo => main",
			types.F32(),
			3.0,
		),
		Entry(
			"a call input",
			"func f(a T) {v bool} T {\n return a\n}\ngo -> f(a=1.5 * 2.0) -> out",
			types.F32(),
			3.0,
		),
	)

	It("Should type each placeholder literal on its own", func(ctx SpecContext) {
		h := newRuntimeHarness(ctx, `
		func f() {a f32} {
			b f64 := 2.0
			log = f"{a + 1.5} {b + 1.5}"
		}
		x -> f()`, channelSymbols(map[string]channelDef{
			"x":   {types.F32(), 100},
			"log": {types.String(), 101},
		}),
			channels.Digest{Key: 100, DataType: telem.Float32T},
			channels.Digest{Key: 101, DataType: telem.StringT},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV[float32](1))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		series := out.Get(101).Series
		Expect(series).ToNot(BeEmpty())
		Expect(series[len(series)-1].Unmarshal[string]()).To(ConsistOf("2.5 3.5"))
	})
})
