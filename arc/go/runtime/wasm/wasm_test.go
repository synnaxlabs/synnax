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
	"github.com/synnaxlabs/arc/runtime/builtin"
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

// expectOutput is a helper that executes a single-function graph and checks the first output element.
func expectOutput[T telem.Sample](ctx context.Context, key string, outType types.Type, body string, resolver symbol.Resolver, expected T) {
	g := singleFunctionGraph(key, outType, body)
	h := newHarness(ctx, g, resolver, nil)
	defer h.Close()
	h.Execute(ctx, key)
	result := h.Output(key, 0)
	Expect(telem.UnmarshalSeries[T](result)[0]).To(Equal(expected))
}

var _ = Describe("WASM", func() {
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

			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(telem.UnmarshalSeries[int64](h.Output("counter", 0))[0]).To(Equal(int64(1)))

			n.Reset()
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(telem.UnmarshalSeries[int64](h.Output("counter", 0))[0]).To(Equal(int64(2)))

			n.Reset()
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(telem.UnmarshalSeries[int64](h.Output("counter", 0))[0]).To(Equal(int64(3)))
		})

		It("Should isolate stateful variables between different functions", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "counter1",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
						Body: ir.Body{Raw: `{
							count i64 $= 0
							count = count + 1
							return count
						}`},
					},
					{
						Key:     "counter2",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
						Body: ir.Body{Raw: `{
							count i64 $= 0
							count = count + 10
							return count
						}`},
					},
				},
				Nodes: []graph.Node{
					{Key: "c1", Type: "counter1"},
					{Key: "c2", Type: "counter2"},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			n1 := h.CreateNode(ctx, "c1")
			n2 := h.CreateNode(ctx, "c2")
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

			n1.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("c1", 0))[0]).To(Equal(int64(1)))

			n2.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("c2", 0))[0]).To(Equal(int64(10)))

			n1.Reset()
			n1.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("c1", 0))[0]).To(Equal(int64(2)))

			n2.Reset()
			n2.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("c2", 0))[0]).To(Equal(int64(20)))
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

	Describe("Series Literal Edge Cases", func() {
		DescribeTable("empty series",
			func(elemType string, outType types.Type) {
				expectOutput(ctx, "empty_series", types.I32(), `{
				s series `+elemType+` := []
				return len(s)
			}`, builtin.SymbolResolver, int32(0))
			},
			Entry("i32", "i32", types.I32()),
			Entry("f64", "f64", types.F64()),
		)

		DescribeTable("single element series",
			expectOutput[int32],
			Entry("i32", ctx, "single_elem", types.I32(), `{
				s series i32 := [42]
				return s[0]
			}`, nil, int32(42)),
		)

		It("Should handle single element f64 series", func() {
			expectOutput(ctx, "single_elem", types.F64(), `{
				s series f64 := [3.14]
				return s[0]
			}`, nil, 3.14)
		})

		It("Should handle 10 element series access", func() {
			expectOutput(ctx, "ten_elem", types.I32(), `{
				s series i32 := [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
				return s[9]
			}`, nil, int32(9))
		})

		It("Should sum elements from 10 element series", func() {
			expectOutput(ctx, "sum_ten", types.I32(), `{
				s series i32 := [1, 1, 1, 1, 1, 1, 1, 1, 1, 1]
				return s[0] + s[1] + s[2] + s[3] + s[4] + s[5] + s[6] + s[7] + s[8] + s[9]
			}`, nil, int32(10))
		})
	})

	Describe("Series Arithmetic Per Type", func() {
		DescribeTable("i32 arithmetic",
			expectOutput[int32],
			Entry("add", ctx, "add_i32", types.I32(), `{
				a series i32 := [1, 2, 3]
				b series i32 := [10, 20, 30]
				c series i32 := a + b
				return c[0]
			}`, nil, int32(11)),
			Entry("subtract", ctx, "sub_i32", types.I32(), `{
				a series i32 := [10, 20, 30]
				b series i32 := [1, 2, 3]
				c series i32 := a - b
				return c[2]
			}`, nil, int32(27)),
			Entry("multiply", ctx, "mul_i32", types.I32(), `{
				a series i32 := [2, 3, 4]
				b series i32 := [5, 6, 7]
				c series i32 := a * b
				return c[1]
			}`, nil, int32(18)),
			Entry("divide", ctx, "div_i32", types.I32(), `{
				a series i32 := [10, 20, 30]
				b series i32 := [2, 4, 5]
				c series i32 := a / b
				return c[0]
			}`, nil, int32(5)),
		)

		It("Should add i64 series", func() {
			expectOutput(ctx, "add_i64", types.I64(), `{
				a series i64 := [1, 2, 3]
				b series i64 := [10, 20, 30]
				c series i64 := a + b
				return c[0]
			}`, nil, int64(11))
		})

		DescribeTable("f32 arithmetic",
			expectOutput[float32],
			Entry("add", ctx, "add_f32", types.F32(), `{
				a series f32 := [1.0, 2.0, 3.0]
				b series f32 := [0.5, 0.5, 0.5]
				c series f32 := a + b
				return c[0]
			}`, nil, float32(1.5)),
			Entry("multiply", ctx, "mul_f32", types.F32(), `{
				a series f32 := [2.0, 3.0]
				b series f32 := [1.5, 2.5]
				c series f32 := a * b
				return c[0]
			}`, nil, float32(3.0)),
		)

		It("Should mod f64 series", func() {
			expectOutput(ctx, "mod_f64", types.F64(), `{
				a series f64 := [10.0, 20.0]
				b series f64 := [3.0, 7.0]
				c series f64 := a % b
				return c[0]
			}`, nil, 1.0)
		})
	})

	Describe("Series-Scalar Operations", func() {
		DescribeTable("series op scalar (i32)",
			expectOutput[int32],
			Entry("add", ctx, "series_scalar_add", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := s + 10
				return r[0]
			}`, nil, int32(11)),
			Entry("subtract", ctx, "series_scalar_sub", types.I32(), `{
				s series i32 := [10, 20, 30]
				r series i32 := s - 5
				return r[1]
			}`, nil, int32(15)),
		)

		DescribeTable("series op scalar (f64)",
			expectOutput[float64],
			Entry("multiply", ctx, "series_scalar_mul", types.F64(), `{
				s series f64 := [1.0, 2.0, 3.0]
				r series f64 := s * 2.0
				return r[2]
			}`, nil, 6.0),
			Entry("divide", ctx, "series_scalar_div", types.F64(), `{
				s series f64 := [10.0, 20.0]
				r series f64 := s / 2.0
				return r[0]
			}`, nil, 5.0),
		)

		DescribeTable("scalar op series (i32)",
			expectOutput[int32],
			Entry("add", ctx, "scalar_series_add", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := 10 + s
				return r[0]
			}`, nil, int32(11)),
			Entry("subtract", ctx, "scalar_series_sub", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := 10 - s
				return r[0]
			}`, nil, int32(9)),
		)

		DescribeTable("scalar op series (f64)",
			expectOutput[float64],
			Entry("multiply", ctx, "scalar_series_mul", types.F64(), `{
				s series f64 := [1.0, 2.0, 3.0]
				r series f64 := 2.0 * s
				return r[1]
			}`, nil, 4.0),
			Entry("divide", ctx, "scalar_series_div", types.F64(), `{
				s series f64 := [2.0, 4.0, 5.0]
				r series f64 := 10.0 / s
				return r[0]
			}`, nil, 5.0),
		)
	})

	Describe("Series Comparison Operations", func() {
		DescribeTable("comparison operators",
			expectOutput[uint8],
			Entry("less than", ctx, "series_lt", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a < b
				return c[0]
			}`, nil, uint8(1)),
			Entry("greater than", ctx, "series_gt", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a > b
				return c[1]
			}`, nil, uint8(1)),
			Entry("equal", ctx, "series_eq", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a == b
				return c[2]
			}`, nil, uint8(1)),
			Entry("not equal", ctx, "series_ne", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [1.0, 4.0, 3.0]
				c series u8 := a != b
				return c[1]
			}`, nil, uint8(1)),
			Entry("less than or equal", ctx, "series_le", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a <= b
				return c[2]
			}`, nil, uint8(1)),
			Entry("greater than or equal", ctx, "series_ge", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a >= b
				return c[1]
			}`, nil, uint8(1)),
		)
	})

	Describe("Series Length Operations", func() {
		DescribeTable("len() function",
			expectOutput[int32],
			Entry("empty series", ctx, "len_empty", types.I32(), `{
				s series f64 := []
				return len(s)
			}`, builtin.SymbolResolver, int32(0)),
			Entry("single element", ctx, "len_one", types.I32(), `{
				s series f64 := [1.0]
				return len(s)
			}`, builtin.SymbolResolver, int32(1)),
			Entry("five elements", ctx, "len_five", types.I32(), `{
				s series f64 := [1.0, 2.0, 3.0, 4.0, 5.0]
				return len(s)
			}`, builtin.SymbolResolver, int32(5)),
			Entry("after operation", ctx, "len_after_op", types.I32(), `{
				a series f64 := [1.0, 2.0, 3.0]
				b series f64 := [4.0, 5.0, 6.0]
				c series f64 := a + b
				return len(c)
			}`, builtin.SymbolResolver, int32(3)),
		)
	})

	Describe("String Operations Extended", func() {
		DescribeTable("string len() function",
			expectOutput[int32],
			Entry("empty string", ctx, "len_empty_str", types.I32(), `{
				return len("")
			}`, builtin.SymbolResolver, int32(0)),
			Entry("simple string", ctx, "len_str", types.I32(), `{
				return len("hello")
			}`, builtin.SymbolResolver, int32(5)),
			Entry("concatenated strings", ctx, "len_concat", types.I32(), `{
				return len("ab" + "cd")
			}`, builtin.SymbolResolver, int32(4)),
			Entry("triple concatenation", ctx, "len_triple", types.I32(), `{
				return len("a" + "b" + "c")
			}`, builtin.SymbolResolver, int32(3)),
			Entry("variable concatenation", ctx, "len_var_concat", types.I32(), `{
				a str := "hello"
				b str := " world"
				return len(a + b)
			}`, builtin.SymbolResolver, int32(11)),
		)
	})

	Describe("No-Input Node Initialization", func() {
		It("Should execute only once per stage entry for nodes with no inputs", func() {
			// Create a stateful counter function with no inputs
			g := singleFunctionGraph("init_counter", types.I64(), `{
				count i64 $= 0
				count = count + 1
				return count
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			n := h.CreateNode(ctx, "init_counter")

			// First call - should execute and return 1
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			Expect(telem.UnmarshalSeries[int64](h.Output("init_counter", 0))[0]).To(Equal(int64(1)))

			// Second call - should NOT execute again (initialized flag)
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			// No output should be marked as changed since we didn't execute
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeFalse())

			// Reset the node (simulating stage re-entry)
			n.Reset()

			// Third call - should execute again after reset
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			// Counter persists so it should be 2 now
			Expect(telem.UnmarshalSeries[int64](h.Output("init_counter", 0))[0]).To(Equal(int64(2)))
		})

		It("Should execute every time for non-entry nodes with inputs", func() {
			g := binaryOpGraph("add", "lhs", "rhs", types.I64(), types.I64(), `{ return lhs + rhs }`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			n := h.CreateNode(ctx, "add")

			h.SetInput("lhs", 0, telem.NewSeriesV[int64](1), telem.NewSeriesSecondsTSV(1))
			h.SetInput("rhs", 0, telem.NewSeriesV[int64](2), telem.NewSeriesSecondsTSV(1))

			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			Expect(telem.UnmarshalSeries[int64](h.Output("add", 0))[0]).To(Equal(int64(3)))

			h.SetInput("lhs", 0, telem.NewSeriesV[int64](10), telem.NewSeriesSecondsTSV(2))
			h.SetInput("rhs", 0, telem.NewSeriesV[int64](20), telem.NewSeriesSecondsTSV(2))

			// Nodes with incoming edges should execute every time they have new input
			changed = make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) { changed.Add(output) }})
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())
			Expect(telem.UnmarshalSeries[int64](h.Output("add", 0))[0]).To(Equal(int64(30)))
		})
	})

	Describe("Config Parameters", func() {
		It("Should pass config values to WASM function", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "add_config",
						Config:  types.Params{{Name: "x", Type: types.I64()}},
						Inputs:  types.Params{{Name: "y", Type: types.I64()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
						Body:    ir.Body{Raw: `{ return x + y }`},
					},
					{
						Key:     "input_source",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
						Body:    ir.Body{Raw: `{ return 1 }`},
					},
				},
				Nodes: []graph.Node{
					{Key: "input_source", Type: "input_source"},
					{Key: "add_config", Type: "add_config", Config: map[string]any{"x": int64(10)}},
				},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "input_source", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "add_config", Param: "y"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			// Set up input source output
			h.SetInput("input_source", 0, telem.NewSeriesV[int64](5), telem.NewSeriesSecondsTSV(1))

			n := h.CreateNode(ctx, "add_config")
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(s string) { changed.Add(s) }})

			output := h.Output("add_config", 0)
			Expect(output.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[int64](output)[0]).To(Equal(int64(15)))
		})

		It("Should handle multiple config parameters", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "multi_config",
						Config:  types.Params{{Name: "a", Type: types.I32()}, {Name: "b", Type: types.I32()}},
						Inputs:  types.Params{{Name: "c", Type: types.I32()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						Body:    ir.Body{Raw: `{ return a + b + c }`},
					},
					{
						Key:     "input_source",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						Body:    ir.Body{Raw: `{ return 1 }`},
					},
				},
				Nodes: []graph.Node{
					{Key: "input_source", Type: "input_source"},
					{Key: "multi_config", Type: "multi_config", Config: map[string]any{"a": int32(5), "b": int32(10)}},
				},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "input_source", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "multi_config", Param: "c"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("input_source", 0, telem.NewSeriesV[int32](3), telem.NewSeriesSecondsTSV(1))

			n := h.CreateNode(ctx, "multi_config")
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(s string) { changed.Add(s) }})

			output := h.Output("multi_config", 0)
			Expect(output.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[int32](output)[0]).To(Equal(int32(18)))
		})

		It("Should handle float64 config parameters", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "scale_config",
						Config:  types.Params{{Name: "factor", Type: types.F64()}},
						Inputs:  types.Params{{Name: "value", Type: types.F64()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
						Body:    ir.Body{Raw: `{ return value * factor }`},
					},
					{
						Key:     "input_source",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
						Body:    ir.Body{Raw: `{ return 1.0 }`},
					},
				},
				Nodes: []graph.Node{
					{Key: "input_source", Type: "input_source"},
					{Key: "scale_config", Type: "scale_config", Config: map[string]any{"factor": 2.5}},
				},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "input_source", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "scale_config", Param: "value"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("input_source", 0, telem.NewSeriesV[float64](10.0), telem.NewSeriesSecondsTSV(1))

			n := h.CreateNode(ctx, "scale_config")
			changed := make(set.Set[string])
			n.Next(node.Context{Context: ctx, MarkChanged: func(s string) { changed.Add(s) }})

			output := h.Output("scale_config", 0)
			Expect(output.Len()).To(Equal(int64(1)))
			Expect(telem.UnmarshalSeries[float64](output)[0]).To(Equal(25.0))
		})
	})

	Describe("Imperative Channel Writes", func() {
		Describe("Writing to Non-Indexed Channels", func() {
			It("Should write only data channel when index is not configured", func() {
				resolver := symbol.MapResolver{
					"output_ch": {
						Name: "output_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   100,
					},
				}
				g := singleFunctionGraph("write_test", types.I32(), `{
					output_ch = 42
					return 42
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 100, DataType: telem.Int32T},
				})
				defer h.Close()

				h.Execute(ctx, "write_test")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(100).Series).To(HaveLen(1))
				Expect(fr.Get(100).Series[0]).To(telem.MatchSeriesDataV[int32](42))
			})
		})

		Describe("Writing to Indexed Channels", func() {
			It("Should write both data and index channel when index is configured", func() {
				resolver := symbol.MapResolver{
					"output_ch": {
						Name: "output_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   100,
					},
				}
				g := singleFunctionGraph("write_indexed", types.I32(), `{
					output_ch = 99
					return 99
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 100, Index: 101, DataType: telem.Int32T},
				})
				defer h.Close()

				h.Execute(ctx, "write_indexed")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(100).Series).To(HaveLen(1))
				Expect(fr.Get(100).Series[0]).To(telem.MatchSeriesDataV[int32](99))
				Expect(fr.Get(101).Series).To(HaveLen(1))
				Expect(fr.Get(101).Series[0].Len()).To(Equal(int64(1)))
				ts := telem.UnmarshalSeries[telem.TimeStamp](fr.Get(101).Series[0])
				Expect(ts[0]).To(BeNumerically(">", 0))
			})

			It("Should write timestamp that is approximately now", func() {
				resolver := symbol.MapResolver{
					"sensor_out": {
						Name: "sensor_out",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   200,
					},
				}
				g := singleFunctionGraph("write_ts", types.I32(), `{
					sensor_out = 42
					return 42
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 200, Index: 201, DataType: telem.Int32T},
				})
				defer h.Close()

				before := telem.Now()
				h.Execute(ctx, "write_ts")
				after := telem.Now()

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(201).Series).To(HaveLen(1))
				ts := telem.UnmarshalSeries[telem.TimeStamp](fr.Get(201).Series[0])
				Expect(ts[0]).To(BeNumerically(">=", before))
				Expect(ts[0]).To(BeNumerically("<=", after))
			})

			It("Should write to multiple indexed channels independently", func() {
				resolver := symbol.MapResolver{
					"ch_a": {
						Name: "ch_a",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   10,
					},
					"ch_b": {
						Name: "ch_b",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   20,
					},
				}
				g := singleFunctionGraph("multi_write", types.I32(), `{
					ch_a = 15
					ch_b = 100
					return 0
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 10, Index: 11, DataType: telem.Int32T},
					{Key: 20, Index: 21, DataType: telem.Int32T},
				})
				defer h.Close()

				h.Execute(ctx, "multi_write")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(10).Series).To(HaveLen(1))
				Expect(fr.Get(10).Series[0]).To(telem.MatchSeriesDataV[int32](15))
				Expect(fr.Get(11).Series).To(HaveLen(1))
				Expect(fr.Get(20).Series).To(HaveLen(1))
				Expect(fr.Get(20).Series[0]).To(telem.MatchSeriesDataV[int32](100))
				Expect(fr.Get(21).Series).To(HaveLen(1))
			})
		})

		Describe("Sequential Writes with Timestamps", func() {
			It("Should produce increasing timestamps for sequential writes", func() {
				resolver := symbol.MapResolver{
					"counter_ch": {
						Name: "counter_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   300,
					},
				}
				g := singleFunctionGraph("seq_write", types.I32(), `{
					count i32 $= 0
					count = count + 1
					counter_ch = count
					return count
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 300, Index: 301, DataType: telem.Int32T},
				})
				defer h.Close()

				n := h.CreateNode(ctx, "seq_write")
				timestamps := make([]telem.TimeStamp, 3)

				for i := range 3 {
					n.Reset()
					n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
					fr, changed := h.state.Flush(telem.Frame[uint32]{})
					Expect(changed).To(BeTrue())
					ts := telem.UnmarshalSeries[telem.TimeStamp](fr.Get(301).Series[0])
					timestamps[i] = ts[0]
				}

				Expect(timestamps[1]).To(BeNumerically(">=", timestamps[0]))
				Expect(timestamps[2]).To(BeNumerically(">=", timestamps[1]))
			})
		})

		Describe("Integer Type Channel Writes", func() {
			It("Should write i32 with index", func() {
				resolver := symbol.MapResolver{
					"i32_ch": {
						Name: "i32_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   700,
					},
				}
				g := singleFunctionGraph("i32_write", types.I32(), `{
					i32_ch = -50000
					return -50000
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 700, Index: 701, DataType: telem.Int32T},
				})
				defer h.Close()

				h.Execute(ctx, "i32_write")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(700).Series).To(HaveLen(1))
				Expect(fr.Get(700).Series[0]).To(telem.MatchSeriesDataV[int32](-50000))
				Expect(fr.Get(701).Series).To(HaveLen(1))
			})

			It("Should write u8 with index", func() {
				resolver := symbol.MapResolver{
					"u8_ch": {
						Name: "u8_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U8()),
						ID:   800,
					},
				}
				g := singleFunctionGraph("u8_write", types.U8(), `{
					u8_ch = 255
					return 255
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 800, Index: 801, DataType: telem.Uint8T},
				})
				defer h.Close()

				h.Execute(ctx, "u8_write")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(800).Series).To(HaveLen(1))
				Expect(fr.Get(800).Series[0]).To(telem.MatchSeriesDataV[uint8](255))
				Expect(fr.Get(801).Series).To(HaveLen(1))
			})
		})

		Describe("Float Type Channel Writes", func() {
			It("Should write f64 with index", func() {
				resolver := symbol.MapResolver{
					"f64_ch": {
						Name: "f64_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   1100,
					},
				}
				g := singleFunctionGraph("f64_write", types.F64(), `{
					f64_ch = 3.14159
					return 3.14159
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 1100, Index: 1101, DataType: telem.Float64T},
				})
				defer h.Close()

				h.Execute(ctx, "f64_write")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(1100).Series).To(HaveLen(1))
				Expect(fr.Get(1100).Series[0]).To(telem.MatchSeriesDataV(3.14159))
				Expect(fr.Get(1101).Series).To(HaveLen(1))
			})

			It("Should write f32 with index", func() {
				resolver := symbol.MapResolver{
					"f32_ch": {
						Name: "f32_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   1200,
					},
				}
				g := singleFunctionGraph("f32_write", types.F32(), `{
					f32_ch = 2.718
					return 2.718
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 1200, Index: 1201, DataType: telem.Float32T},
				})
				defer h.Close()

				h.Execute(ctx, "f32_write")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(1200).Series).To(HaveLen(1))
				Expect(fr.Get(1200).Series[0]).To(telem.MatchSeriesDataV[float32](2.718))
				Expect(fr.Get(1201).Series).To(HaveLen(1))
			})
		})

		Describe("Edge Cases", func() {
			It("Should handle empty flush when no writes occur", func() {
				g := singleFunctionGraph("no_write", types.I32(), `{ return 42 }`)
				h := newHarness(ctx, g, nil, nil)
				defer h.Close()

				h.Execute(ctx, "no_write")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeFalse())
				Expect(fr.RawKeys()).To(BeEmpty())
			})

			It("Should handle channel with zero as index (no index)", func() {
				resolver := symbol.MapResolver{
					"no_idx_ch": {
						Name: "no_idx_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   900,
					},
				}
				g := singleFunctionGraph("zero_idx", types.I32(), `{
					no_idx_ch = 123
					return 123
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 900, Index: 0, DataType: telem.Int32T},
				})
				defer h.Close()

				h.Execute(ctx, "zero_idx")

				fr, changed := h.state.Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(900).Series).To(HaveLen(1))
				Expect(fr.Get(0).Series).To(BeEmpty())
			})
		})

		Describe("Comparison with Declarative Writes", func() {
			It("Should produce same output structure as WriteChan for indexed channels", func() {
				resolver := symbol.MapResolver{
					"test_ch": {
						Name: "test_ch",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   1000,
					},
				}
				g := singleFunctionGraph("imperative_vs_decl", types.I32(), `{
					test_ch = 123
					return 123
				}`)

				h := newHarness(ctx, g, resolver, []state.ChannelDigest{
					{Key: 1000, Index: 1001, DataType: telem.Int32T},
				})
				defer h.Close()

				h.Execute(ctx, "imperative_vs_decl")

				fr, _ := h.state.Flush(telem.Frame[uint32]{})
				dataKeys := make(set.Set[uint32])
				for _, key := range fr.RawKeys() {
					dataKeys.Add(key)
				}
				Expect(dataKeys.Contains(1000)).To(BeTrue())
				Expect(dataKeys.Contains(1001)).To(BeTrue())
			})
		})
	})

	Describe("Void Functions (No Outputs)", func() {
		It("Should execute without panic when function has no outputs", func() {
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "void_func",
						Inputs:  types.Params{{Name: "trigger", Type: types.U8()}},
						Outputs: types.Params{},
						Body:    ir.Body{Raw: `{}`},
					},
					{
						Key:     "trigger_source",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
						Body:    ir.Body{Raw: `{ return 1 }`},
					},
				},
				Nodes: []graph.Node{
					{Key: "trigger_source", Type: "trigger_source"},
					{Key: "void_func", Type: "void_func"},
				},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "trigger_source", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "void_func", Param: "trigger"}},
				},
			}
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			h.SetInput("trigger_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(1))

			changed := h.Execute(ctx, "void_func")
			Expect(changed).To(BeEmpty())
		})

		It("Should execute void function with stateful variables", func() {
			resolver := symbol.MapResolver{
				"output_ch": {
					Name: "output_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.I32()),
					ID:   100,
				},
			}
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "void_with_state",
						Inputs:  types.Params{{Name: "trigger", Type: types.U8()}},
						Outputs: types.Params{},
						Body: ir.Body{Raw: `{
							count i32 $= 0
							count = count + 1
							output_ch = count
						}`},
					},
					{
						Key:     "trigger_source",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
						Body:    ir.Body{Raw: `{ return 1 }`},
					},
				},
				Nodes: []graph.Node{
					{Key: "trigger_source", Type: "trigger_source"},
					{Key: "void_with_state", Type: "void_with_state"},
				},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "trigger_source", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "void_with_state", Param: "trigger"}},
				},
			}
			h := newHarness(ctx, g, resolver, []state.ChannelDigest{{Key: 100, DataType: telem.Int32T}})
			defer h.Close()

			h.SetInput("trigger_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(1))

			n := h.CreateNode(ctx, "void_with_state")
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

			n.Reset()
			n.Next(nCtx)
			fr, changed := h.state.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(100).Series[0]).To(telem.MatchSeriesDataV[int32](1))

			h.SetInput("trigger_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(2))
			n.Next(nCtx)
			fr, changed = h.state.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(100).Series[0]).To(telem.MatchSeriesDataV[int32](2))
		})
	})

	Describe("Flow Expression Execution", func() {
		It("Should execute every time for flow expression nodes", func() {
			g := singleFunctionGraph("expression_0", types.I64(), `{
				count i64 $= 0
				count = count + 1
				return count
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			n := h.CreateNode(ctx, "expression_0")
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expression_0", 0))[0]).To(Equal(int64(1)))

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expression_0", 0))[0]).To(Equal(int64(2)))

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expression_0", 0))[0]).To(Equal(int64(3)))
		})

		It("Should continue executing after reset for expression nodes", func() {
			g := singleFunctionGraph("expression_0", types.I64(), `{
				count i64 $= 0
				count = count + 1
				return count
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			n := h.CreateNode(ctx, "expression_0")
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expression_0", 0))[0]).To(Equal(int64(1)))

			n.Reset()

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expression_0", 0))[0]).To(Equal(int64(2)))

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expression_0", 0))[0]).To(Equal(int64(3)))
		})

		It("Should not treat non-expression nodes as expressions", func() {
			g := singleFunctionGraph("expr_0", types.I64(), `{
				count i64 $= 0
				count = count + 1
				return count
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()

			n := h.CreateNode(ctx, "expr_0")
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expr_0", 0))[0]).To(Equal(int64(1)))

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expr_0", 0))[0]).To(Equal(int64(1)))
		})
	})
})
