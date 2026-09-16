// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm_test

import (
	"bytes"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
)

func stamps(n int64) telem.Series {
	ts := make([]telem.TimeStamp, n)
	for i := range ts {
		ts[i] = telem.TimeStamp(i+1) * telem.SecondTS
	}
	return telem.NewSeries(ts)
}

func sampleAt(s telem.Series, i int) telem.Series {
	return telem.Series{DataType: s.DataType, Data: s.At(i)}
}

var _ = Describe("Batched execution", func() {
	// A series longer than one sample takes the batched guest call; a single sample
	// stays on the per-sample path.
	runAdd := func(
		ctx context.Context,
		t types.Type,
		lhs, rhs telem.Series,
	) telem.Series {
		GinkgoHelper()
		h := newHarness(ctx, binaryOpGraph(
			"add", "lhs", "rhs", t, t, `{ return lhs + rhs }`,
		), nil)
		DeferCleanup(h.Close)
		h.SetInput("lhs", 0, lhs, stamps(lhs.Len()))
		h.SetInput("rhs", 0, rhs, stamps(rhs.Len()))
		Expect(h.Execute(ctx, "add").Contains(ir.DefaultOutputParam)).To(BeTrue())
		return h.Output("add", 0)
	}

	DescribeTable("Should produce what the per-sample path produces",
		func(ctx SpecContext, t types.Type, lhs, rhs telem.Series) {
			batched := runAdd(ctx, t, lhs, rhs)
			Expect(batched.Len()).To(Equal(lhs.Len()))
			var perSample bytes.Buffer
			for i := range int(lhs.Len()) {
				one := runAdd(
					ctx, t,
					sampleAt(lhs, i),
					sampleAt(rhs, i%int(rhs.Len())),
				)
				perSample.Write(one.Data)
			}
			Expect(batched.Data).To(Equal(perSample.Bytes()))
		},
		Entry("f64", types.F64(),
			telem.NewSeriesV[float64](-1.5, 0, 1.5, 2.25),
			telem.NewSeriesV[float64](0.25)),
		Entry("f32", types.F32(),
			telem.NewSeriesV[float32](-1.5, 0, 1.5, 2.25),
			telem.NewSeriesV[float32](0.25, 0.5)),
		Entry("i64", types.I64(),
			telem.NewSeriesV[int64](-2, -1, 0, 1),
			telem.NewSeriesV[int64](3)),
		Entry("i32", types.I32(),
			telem.NewSeriesV[int32](-2, -1, 0, 1),
			telem.NewSeriesV[int32](3, 4)),
		Entry("i16", types.I16(),
			telem.NewSeriesV[int16](-2, -1, 0, 1),
			telem.NewSeriesV[int16](3)),
		Entry("i8", types.I8(),
			telem.NewSeriesV[int8](-2, -1, 0, 1),
			telem.NewSeriesV[int8](3)),
		Entry("u64", types.U64(),
			telem.NewSeriesV[uint64](1, 2, 3, 4),
			telem.NewSeriesV[uint64](5)),
		Entry("u32", types.U32(),
			telem.NewSeriesV[uint32](1, 2, 3, 4),
			telem.NewSeriesV[uint32](5, 6)),
		Entry("u16", types.U16(),
			telem.NewSeriesV[uint16](1, 2, 3, 4),
			telem.NewSeriesV[uint16](5)),
		Entry("u8", types.U8(),
			telem.NewSeriesV[uint8](250, 251, 252, 253),
			telem.NewSeriesV[uint8](5)),
	)

	// A negative i8 literal must reach the guest as -1 on both paths, and so must a
	// negative edge-fed sample.
	DescribeTable("Should sign-extend narrow signed inputs on both paths",
		func(ctx SpecContext, in telem.Series, expected []int8) {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "div",
						Inputs: types.Params{
							{Name: "x", Type: types.I8()},
							{Name: "d", Type: types.I8(), Value: int8(-1)},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I8()},
						},
						Body: ir.Body{Raw: `{ return x / d }`},
					},
					{
						Key: "src",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I8()},
						},
						Body: ir.Body{Raw: `{ return 1 }`},
					},
				},
				Nodes: []graph.Node{{Key: "src"}, {Key: "div"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"src": {"type": "src"},
					"div": {"type": "div"},
				},
				Edges: graph.Edges{{
					Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "div", Param: "x"},
				}},
			}
			h := newHarness(ctx, g, nil)
			DeferCleanup(h.Close)
			h.SetInput("src", 0, in, stamps(in.Len()))
			h.Execute(ctx, "div")
			Expect(h.Output("div", 0).Unmarshal[int8]()).To(Equal(expected))
		},
		Entry("per-sample", telem.NewSeriesV[int8](-4), []int8{4}),
		Entry("batched", telem.NewSeriesV[int8](-4, 6, 0), []int8{4, -6, 0}),
	)

	It("Should grow guest memory to hold a long series", func(ctx SpecContext) {
		const count = 20000
		lhs := make([]float64, count)
		for i := range lhs {
			lhs[i] = float64(i)
		}
		out := runAdd(
			ctx,
			types.F64(),
			telem.NewSeries(lhs),
			telem.NewSeriesV[float64](0.5),
		)
		Expect(out.Len()).To(Equal(int64(count)))
		values := out.Unmarshal[float64]()
		Expect(values[0]).To(Equal(0.5))
		Expect(values[count-1]).To(Equal(float64(count-1) + 0.5))
	})

	// Both paths owe the same contract: a trap ends the cycle and the node emits
	// nothing, so output never depends on where in the series the trap landed.
	divGraph := binaryOpGraph(
		"div", "num", "den", types.I64(), types.I64(), `{ return lhs / rhs }`,
	)
	// Two outputs never qualify for the wrapper, so this graph stays per-sample even
	// over a long series.
	divPairGraph := binaryTwoOutputGraph(
		"div", "num", "den", types.I64(), types.I64(), "quot", "same",
		`{
			quot = lhs / rhs
			same = lhs
		}`,
	)

	DescribeTable(
		"Should emit nothing when a sample traps",
		func(ctx SpecContext, g arc.Graph, num, den telem.Series) {
			h := newHarness(ctx, g, nil)
			DeferCleanup(h.Close)
			h.SetInput("num", 0, num, stamps(num.Len()))
			h.SetInput("den", 0, den, stamps(den.Len()))
			Expect(h.Execute(ctx, "div")).To(BeEmpty())
			for i := range h.Outputs("div") {
				Expect(h.Output("div", i).Len()).To(Equal(int64(0)))
				Expect(h.OutputTime("div", i).Len()).To(Equal(int64(0)))
			}
			Expect(h.Errors()).To(HaveLen(1))
			Expect(h.Errors()[0]).To(MatchError(ContainSubstring("node div")))
		},
		Entry("batched, trap on the first sample", divGraph,
			telem.NewSeriesV[int64](1, 2, 3, 4),
			telem.NewSeriesV[int64](0, 1, 1, 1)),
		Entry("batched, trap mid-series", divGraph,
			telem.NewSeriesV[int64](1, 2, 3, 4),
			telem.NewSeriesV[int64](1, 1, 0, 1)),
		Entry("batched, trap on the last sample", divGraph,
			telem.NewSeriesV[int64](1, 2, 3, 4),
			telem.NewSeriesV[int64](1, 1, 1, 0)),
		Entry("per-sample, single sample", divGraph,
			telem.NewSeriesV[int64](1),
			telem.NewSeriesV[int64](0)),
		Entry(
			"per-sample, trap mid-series discards the samples before it",
			divPairGraph,
			telem.NewSeriesV[int64](1, 2, 3, 4),
			telem.NewSeriesV[int64](1, 1, 0, 1),
		),
		Entry("per-sample, trap on the last sample", divPairGraph,
			telem.NewSeriesV[int64](1, 2, 3, 4),
			telem.NewSeriesV[int64](1, 1, 1, 0)),
	)

	It("Should stamp each output sample with its input time", func(ctx SpecContext) {
		h := newHarness(ctx, binaryOpGraph(
			"add", "lhs", "rhs", types.I64(), types.I64(), `{ return lhs + rhs }`,
		), nil)
		DeferCleanup(h.Close)
		h.SetInput("lhs", 0, telem.NewSeriesV[int64](1, 2, 3), stamps(3))
		h.SetInput("rhs", 0, telem.NewSeriesV[int64](10), stamps(1))
		h.Execute(ctx, "add")
		Expect(h.OutputTime("add", 0)).To(telem.MatchSeries(stamps(3)))
	})
})
