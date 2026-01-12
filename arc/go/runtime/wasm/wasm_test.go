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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// testHarness encapsulates common test setup for wasm module tests.
type testHarness struct {
	factory  node.Factory
	state    *state.State
	wasmMod  *wasm.Module
	mod      module.Module
	analyzed ir.IR
	graph    arc.Graph
}

// newHarness creates a new test harness from a graph definition.
func newHarness(
	ctx context.Context,
	g arc.Graph,
	resolver symbol.Resolver,
	channelDigests []state.ChannelDigest,
	opts ...arc.Option,
) *testHarness {
	if resolver != nil {
		opts = append(opts, arc.WithResolver(resolver))
	}
	mod := MustSucceed(arc.CompileGraph(ctx, g, opts...))
	analyzed, diagnostics := graph.Analyze(ctx, g, resolver)
	Expect(diagnostics.Ok()).To(BeTrue())
	cfg := state.Config{IR: analyzed}
	if len(channelDigests) > 0 {
		cfg.ChannelDigests = channelDigests
	}
	s := state.New(cfg)
	wasmMod := MustSucceed(wasm.OpenModule(ctx, wasm.ModuleConfig{
		Module: mod,
		State:  s,
	}))
	factory := MustSucceed(wasm.NewFactory(wasmMod))
	return &testHarness{
		graph:    g,
		mod:      mod,
		analyzed: analyzed,
		state:    s,
		wasmMod:  wasmMod,
		factory:  factory,
	}
}

func (h *testHarness) Close() {
	Expect(h.wasmMod.Close()).To(Succeed())
}

func (h *testHarness) SetInput(nodeKey string, idx int, data telem.Series, time telem.Series) {
	n := h.state.Node(nodeKey)
	*n.Output(idx) = data
	*n.OutputTime(idx) = time
}

func (h *testHarness) CreateNode(ctx context.Context, nodeKey string) node.Node {
	return MustSucceed(h.factory.Create(ctx, node.Config{
		Node:   h.analyzed.Nodes.Get(nodeKey),
		State:  h.state.Node(nodeKey),
		Module: h.mod,
	}))
}

func (h *testHarness) Execute(ctx context.Context, nodeKey string) set.Set[string] {
	n := h.CreateNode(ctx, nodeKey)
	changed := make(set.Set[string])
	n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
	return changed
}

func (h *testHarness) Output(nodeKey string, idx int) telem.Series {
	return *h.state.Node(nodeKey).Output(idx)
}

func (h *testHarness) OutputTime(nodeKey string, idx int) telem.Series {
	return *h.state.Node(nodeKey).OutputTime(idx)
}

// singleFunctionGraph creates a simple graph with one function and one node.
func singleFunctionGraph(key string, outType types.Type, body string) arc.Graph {
	return arc.Graph{
		Functions: []ir.Function{{
			Key:     key,
			Inputs:  types.Params{},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: outType}},
			Body:    ir.Body{Raw: body},
		}},
		Nodes: []graph.Node{{Key: key, Type: key}},
	}
}

// binaryOpGraph creates a graph with two input nodes and one binary operation node.
func binaryOpGraph(
	opKey string,
	lhsKey, rhsKey string,
	inType, outType types.Type,
	body string,
) arc.Graph {
	dummyBody := ir.Body{Raw: `{ return 1 }`}
	if inType.Kind == types.KindF32 || inType.Kind == types.KindF64 {
		dummyBody = ir.Body{Raw: `{ return 1.0 }`}
	}
	return arc.Graph{
		Functions: []ir.Function{
			{
				Key:     opKey,
				Inputs:  types.Params{{Name: "lhs", Type: inType}, {Name: "rhs", Type: inType}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: outType}},
				Body:    ir.Body{Raw: body},
			},
			{
				Key:     lhsKey,
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: inType}},
				Body:    dummyBody,
			},
			{
				Key:     rhsKey,
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: inType}},
				Body:    dummyBody,
			},
		},
		Nodes: []graph.Node{
			{Key: lhsKey, Type: lhsKey},
			{Key: rhsKey, Type: rhsKey},
			{Key: opKey, Type: opKey},
		},
		Edges: []graph.Edge{
			{Source: ir.Handle{Node: lhsKey, Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: opKey, Param: "lhs"}},
			{Source: ir.Handle{Node: rhsKey, Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: opKey, Param: "rhs"}},
		},
	}
}

var _ = Describe("Wasm", func() {
	Describe("Next with mismatched input lengths", func() {
		It("Should repeat shorter input values to match longest input", func() {
			g := binaryOpGraph("add", "lhs", "rhs", types.I64(), types.I64(), `{ return lhs + rhs }`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("lhs", 0, telem.NewSeriesV[int64](1, 2, 3, 4, 5), telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5))
			h.SetInput("rhs", 0, telem.NewSeriesV[int64](10, 20), telem.NewSeriesSecondsTSV(1, 2))

			changed := h.Execute(ctx, "add")
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())

			result := h.Output("add", 0)
			Expect(result.Len()).To(Equal(int64(5)))
			Expect(telem.UnmarshalSeries[int64](result)).To(Equal([]int64{11, 22, 13, 24, 15}))
			Expect(h.OutputTime("add", 0)).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)))
		})

		It("Should handle equal length inputs correctly", func() {
			g := binaryOpGraph("multiply", "a", "b", types.I32(), types.I32(), `{ return lhs * rhs }`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("a", 0, telem.NewSeriesV[int32](2, 3, 4), telem.NewSeriesSecondsTSV(10, 20, 30))
			h.SetInput("b", 0, telem.NewSeriesV[int32](5, 6, 7), telem.NewSeriesSecondsTSV(10, 20, 30))

			h.Execute(ctx, "multiply")
			result := h.Output("multiply", 0)
			Expect(result.Len()).To(Equal(int64(3)))
			Expect(telem.UnmarshalSeries[int32](result)).To(Equal([]int32{10, 18, 28}))
		})

		It("Should repeat single value input across all iterations", func() {
			g := binaryOpGraph("subtract", "x", "y", types.F32(), types.F32(), `{ return lhs - rhs }`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("x", 0, telem.NewSeriesV[float32](100.0, 200.0, 300.0, 400.0), telem.NewSeriesSecondsTSV(5, 10, 15, 20))
			h.SetInput("y", 0, telem.NewSeriesV[float32](25.0), telem.NewSeriesSecondsTSV(5))

			h.Execute(ctx, "subtract")
			result := h.Output("subtract", 0)
			Expect(result.Len()).To(Equal(int64(4)))
			Expect(telem.UnmarshalSeries[float32](result)).To(Equal([]float32{75.0, 175.0, 275.0, 375.0}))
		})
	})

	Describe("Next with multiple outputs", func() {
		It("Should handle functions with multiple outputs and mismatched input lengths", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "math_ops",
						Inputs:  types.Params{{Name: "a", Type: types.I64()}, {Name: "b", Type: types.I64()}},
						Outputs: types.Params{{Name: "sum", Type: types.I64()}, {Name: "product", Type: types.I64()}},
						Body: ir.Body{Raw: `{
							sum = a + b
							product = a * b
						}`},
					},
					{Key: "a", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}, Body: ir.Body{Raw: `{ return 1 }`}},
					{Key: "b", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}, Body: ir.Body{Raw: `{ return 1 }`}},
				},
				Nodes: []graph.Node{{Key: "a", Type: "a"}, {Key: "b", Type: "b"}, {Key: "math_ops", Type: "math_ops"}},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "math_ops", Param: "a"}},
					{Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "math_ops", Param: "b"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("a", 0, telem.NewSeriesV[int64](10, 20, 30), telem.NewSeriesSecondsTSV(1, 2, 3))
			h.SetInput("b", 0, telem.NewSeriesV[int64](5), telem.NewSeriesSecondsTSV(1))

			changed := h.Execute(ctx, "math_ops")
			Expect(changed.Contains("sum")).To(BeTrue())
			Expect(changed.Contains("product")).To(BeTrue())

			sumResult := h.Output("math_ops", 0)
			Expect(telem.UnmarshalSeries[int64](sumResult)).To(Equal([]int64{15, 25, 35}))

			productResult := h.Output("math_ops", 1)
			Expect(telem.UnmarshalSeries[int64](productResult)).To(Equal([]int64{50, 100, 150}))
		})
	})

	Describe("Runtime Operations - Channel Read", func() {
		It("Should read from channels using runtime bindings", func() {
			resolver := symbol.MapResolver{
				"sensor": symbol.Symbol{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 0},
			}
			g := singleFunctionGraph("read_channel", types.I32(), `{
				value i32 := sensor
				return value * 2
			}`)

			h := newHarness(ctx, g, resolver, []state.ChannelDigest{{Key: 0, DataType: telem.Int32T}})
			defer h.Close()

			fr := telem.Frame[uint32]{}
			fr = fr.Append(0, telem.NewSeriesV[int32](21))
			h.state.Ingest(fr)

			h.Execute(ctx, "read_channel")
			result := h.Output("read_channel", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(42)))
		})
	})

	Describe("Runtime Operations - State Persistence", func() {
		It("Should persist stateful variables across function calls", func() {
			g := singleFunctionGraph("counter", types.I64(), `{
				count i64 $= 0
				count = count + 1
				return count
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			n := h.CreateNode(ctx, "counter")

			// First call - should return 1
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(telem.UnmarshalSeries[int64](h.Output("counter", 0))[0]).To(Equal(int64(1)))

			// Second call - should return 2
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(telem.UnmarshalSeries[int64](h.Output("counter", 0))[0]).To(Equal(int64(2)))

			// Third call - should return 3
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(telem.UnmarshalSeries[int64](h.Output("counter", 0))[0]).To(Equal(int64(3)))
		})
	})

	Describe("Optional Parameters", func() {
		It("Should use default value for unconnected optional input", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "add",
						Inputs:  types.Params{{Name: "x", Type: types.I64()}, {Name: "y", Type: types.I64(), Value: int64(10)}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
						Body:    ir.Body{Raw: `{ return x + y }`},
					},
					{Key: "x", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}, Body: ir.Body{Raw: `{ return 1 }`}},
				},
				Nodes: []graph.Node{{Key: "x", Type: "x"}, {Key: "add", Type: "add"}},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "x", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "add", Param: "x"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("x", 0, telem.NewSeriesV[int64](5, 15, 25), telem.NewSeriesSecondsTSV(1, 2, 3))
			h.Execute(ctx, "add")
			Expect(telem.UnmarshalSeries[int64](h.Output("add", 0))).To(Equal([]int64{15, 25, 35}))
		})

		It("Should handle multiple optional parameters", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "compute",
						Inputs:  types.Params{{Name: "a", Type: types.I32()}, {Name: "b", Type: types.I32(), Value: int32(2)}, {Name: "c", Type: types.I32(), Value: int32(3)}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						Body:    ir.Body{Raw: `{ return a * b + c }`},
					},
					{Key: "a", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}}, Body: ir.Body{Raw: `{ return 1 }`}},
				},
				Nodes: []graph.Node{{Key: "a", Type: "a"}, {Key: "compute", Type: "compute"}},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "compute", Param: "a"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("a", 0, telem.NewSeriesV[int32](5, 10), telem.NewSeriesSecondsTSV(1, 2))
			h.Execute(ctx, "compute")
			Expect(telem.UnmarshalSeries[int32](h.Output("compute", 0))).To(Equal([]int32{13, 23}))
		})

		It("Should handle float64 optional parameters", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "scale",
						Inputs:  types.Params{{Name: "value", Type: types.F64()}, {Name: "factor", Type: types.F64(), Value: 2.5}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
						Body:    ir.Body{Raw: `{ return value * factor }`},
					},
					{Key: "value", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}}, Body: ir.Body{Raw: `{ return 1.0 }`}},
				},
				Nodes: []graph.Node{{Key: "value", Type: "value"}, {Key: "scale", Type: "scale"}},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "value", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "scale", Param: "value"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("value", 0, telem.NewSeriesV[float64](10.0, 20.0), telem.NewSeriesSecondsTSV(1, 2))
			h.Execute(ctx, "scale")
			Expect(telem.UnmarshalSeries[float64](h.Output("scale", 0))).To(Equal([]float64{25.0, 50.0}))
		})

		It("Should allow overriding optional parameter with connected edge", func() {
			g := binaryOpGraph("add", "x", "y", types.I64(), types.I64(), `{ return lhs + rhs }`)
			// Modify to add optional value
			g.Functions[0].Inputs[1].Value = int64(10)

			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("x", 0, telem.NewSeriesV[int64](5), telem.NewSeriesSecondsTSV(1))
			h.SetInput("y", 0, telem.NewSeriesV[int64](100), telem.NewSeriesSecondsTSV(1))
			h.Execute(ctx, "add")
			Expect(telem.UnmarshalSeries[int64](h.Output("add", 0))).To(Equal([]int64{105}))
		})
	})

	Describe("Alignment and TimeRange Propagation", func() {
		It("Should sum alignments from multiple inputs and propagate to outputs", func() {
			g := binaryOpGraph("add", "lhs", "rhs", types.I64(), types.I64(), `{ return lhs + rhs }`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			lhsSeries := telem.NewSeriesV[int64](1, 2, 3)
			lhsSeries.Alignment = 100
			lhsSeries.TimeRange = telem.TimeRange{Start: 10 * telem.SecondTS, End: 30 * telem.SecondTS}
			h.SetInput("lhs", 0, lhsSeries, telem.NewSeriesSecondsTSV(10, 20, 30))

			rhsSeries := telem.NewSeriesV[int64](10, 20, 30)
			rhsSeries.Alignment = 50
			rhsSeries.TimeRange = telem.TimeRange{Start: 5 * telem.SecondTS, End: 25 * telem.SecondTS}
			h.SetInput("rhs", 0, rhsSeries, telem.NewSeriesSecondsTSV(5, 15, 25))

			h.Execute(ctx, "add")

			result := h.Output("add", 0)
			Expect(telem.UnmarshalSeries[int64](result)).To(Equal([]int64{11, 22, 33}))
			Expect(result.Alignment).To(Equal(telem.Alignment(150)))
			Expect(result.TimeRange.Start).To(Equal(5 * telem.SecondTS))
			Expect(result.TimeRange.End).To(Equal(30 * telem.SecondTS))
		})

		It("Should propagate alignment and time range to multiple outputs", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "math_ops",
						Inputs:  types.Params{{Name: "a", Type: types.I64()}, {Name: "b", Type: types.I64()}},
						Outputs: types.Params{{Name: "sum", Type: types.I64()}, {Name: "product", Type: types.I64()}},
						Body: ir.Body{Raw: `{
							sum = a + b
							product = a * b
						}`},
					},
					{Key: "a", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}, Body: ir.Body{Raw: `{ return 1 }`}},
					{Key: "b", Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}, Body: ir.Body{Raw: `{ return 1 }`}},
				},
				Nodes: []graph.Node{{Key: "a", Type: "a"}, {Key: "b", Type: "b"}, {Key: "math_ops", Type: "math_ops"}},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "math_ops", Param: "a"}},
					{Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "math_ops", Param: "b"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			aSeries := telem.NewSeriesV[int64](2, 3)
			aSeries.Alignment = 200
			aSeries.TimeRange = telem.TimeRange{Start: 100 * telem.SecondTS, End: 200 * telem.SecondTS}
			h.SetInput("a", 0, aSeries, telem.NewSeriesSecondsTSV(100, 150))

			bSeries := telem.NewSeriesV[int64](5, 10)
			bSeries.Alignment = 300
			bSeries.TimeRange = telem.TimeRange{Start: 50 * telem.SecondTS, End: 250 * telem.SecondTS}
			h.SetInput("b", 0, bSeries, telem.NewSeriesSecondsTSV(50, 200))

			h.Execute(ctx, "math_ops")

			expectedAlignment := telem.Alignment(500)
			expectedTimeRange := telem.TimeRange{Start: 50 * telem.SecondTS, End: 250 * telem.SecondTS}

			sumResult := h.Output("math_ops", 0)
			Expect(sumResult.Alignment).To(Equal(expectedAlignment))
			Expect(sumResult.TimeRange).To(Equal(expectedTimeRange))

			productResult := h.Output("math_ops", 1)
			Expect(productResult.Alignment).To(Equal(expectedAlignment))
			Expect(productResult.TimeRange).To(Equal(expectedTimeRange))
		})
	})

	Describe("Stateful Series Variables", func() {
		It("Should persist stateful series variables across calls", func() {
			g := singleFunctionGraph("series_state", types.F64(), `{ history series f64 $= [0.0]
							return history[0] }`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			n := h.CreateNode(ctx, "series_state")
			changed := make(set.Set[string])

			// First call
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(telem.UnmarshalSeries[float64](h.Output("series_state", 0))[0]).To(Equal(float64(0.0)))

			// Second call - state persists
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(telem.UnmarshalSeries[float64](h.Output("series_state", 0))[0]).To(Equal(float64(0.0)))
		})
	})
})
