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

	Describe("Series Literal Edge Cases", func() {
		It("Should handle empty i32 series", func() {
			g := singleFunctionGraph("empty_series", types.I32(), `{
				s series i32 := []
				return len(s)
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "empty_series")
			result := h.Output("empty_series", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(0)))
		})

		It("Should handle empty f64 series", func() {
			g := singleFunctionGraph("empty_series", types.I32(), `{
				s series f64 := []
				return len(s)
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "empty_series")
			result := h.Output("empty_series", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(0)))
		})

		It("Should handle single element i32 series", func() {
			g := singleFunctionGraph("single_elem", types.I32(), `{
				s series i32 := [42]
				return s[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "single_elem")
			result := h.Output("single_elem", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(42)))
		})

		It("Should handle single element f64 series", func() {
			g := singleFunctionGraph("single_elem", types.F64(), `{
				s series f64 := [3.14]
				return s[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "single_elem")
			result := h.Output("single_elem", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(Equal(3.14))
		})

		It("Should handle 10 element series access", func() {
			g := singleFunctionGraph("ten_elem", types.I32(), `{
				s series i32 := [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
				return s[9]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "ten_elem")
			result := h.Output("ten_elem", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(9)))
		})

		It("Should sum elements from 10 element series", func() {
			g := singleFunctionGraph("sum_ten", types.I32(), `{
				s series i32 := [1, 1, 1, 1, 1, 1, 1, 1, 1, 1]
				return s[0] + s[1] + s[2] + s[3] + s[4] + s[5] + s[6] + s[7] + s[8] + s[9]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "sum_ten")
			result := h.Output("sum_ten", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(10)))
		})
	})

	Describe("Series Arithmetic Per Type", func() {
		It("Should add i32 series", func() {
			g := singleFunctionGraph("add_i32", types.I32(), `{
				a series i32 := [1, 2, 3]
				b series i32 := [10, 20, 30]
				c series i32 := a + b
				return c[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "add_i32")
			result := h.Output("add_i32", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(11)))
		})

		It("Should subtract i32 series", func() {
			g := singleFunctionGraph("sub_i32", types.I32(), `{
				a series i32 := [10, 20, 30]
				b series i32 := [1, 2, 3]
				c series i32 := a - b
				return c[2]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "sub_i32")
			result := h.Output("sub_i32", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(27)))
		})

		It("Should multiply i32 series", func() {
			g := singleFunctionGraph("mul_i32", types.I32(), `{
				a series i32 := [2, 3, 4]
				b series i32 := [5, 6, 7]
				c series i32 := a * b
				return c[1]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "mul_i32")
			result := h.Output("mul_i32", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(18)))
		})

		It("Should divide i32 series", func() {
			g := singleFunctionGraph("div_i32", types.I32(), `{
				a series i32 := [10, 20, 30]
				b series i32 := [2, 4, 5]
				c series i32 := a / b
				return c[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "div_i32")
			result := h.Output("div_i32", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(5)))
		})

		It("Should add i64 series", func() {
			g := singleFunctionGraph("add_i64", types.I64(), `{
				a series i64 := [1, 2, 3]
				b series i64 := [10, 20, 30]
				c series i64 := a + b
				return c[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "add_i64")
			result := h.Output("add_i64", 0)
			Expect(telem.UnmarshalSeries[int64](result)[0]).To(Equal(int64(11)))
		})

		It("Should add f32 series", func() {
			g := singleFunctionGraph("add_f32", types.F32(), `{
				a series f32 := [1.0, 2.0, 3.0]
				b series f32 := [0.5, 0.5, 0.5]
				c series f32 := a + b
				return c[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "add_f32")
			result := h.Output("add_f32", 0)
			Expect(telem.UnmarshalSeries[float32](result)[0]).To(Equal(float32(1.5)))
		})

		It("Should multiply f32 series", func() {
			g := singleFunctionGraph("mul_f32", types.F32(), `{
				a series f32 := [2.0, 3.0]
				b series f32 := [1.5, 2.5]
				c series f32 := a * b
				return c[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "mul_f32")
			result := h.Output("mul_f32", 0)
			Expect(telem.UnmarshalSeries[float32](result)[0]).To(Equal(float32(3.0)))
		})

		It("Should mod f64 series", func() {
			g := singleFunctionGraph("mod_f64", types.F64(), `{
				a series f64 := [10.0, 20.0]
				b series f64 := [3.0, 7.0]
				c series f64 := a % b
				return c[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "mod_f64")
			result := h.Output("mod_f64", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(Equal(1.0))
		})
	})

	Describe("Series-Scalar Operations", func() {
		It("Should add series + scalar i32", func() {
			g := singleFunctionGraph("series_scalar_add", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := s + 10
				return r[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_scalar_add")
			result := h.Output("series_scalar_add", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(11)))
		})

		It("Should subtract series - scalar i32", func() {
			g := singleFunctionGraph("series_scalar_sub", types.I32(), `{
				s series i32 := [10, 20, 30]
				r series i32 := s - 5
				return r[1]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_scalar_sub")
			result := h.Output("series_scalar_sub", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(15)))
		})

		It("Should multiply series * scalar f64", func() {
			g := singleFunctionGraph("series_scalar_mul", types.F64(), `{
				s series f64 := [1.0, 2.0, 3.0]
				r series f64 := s * 2.0
				return r[2]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_scalar_mul")
			result := h.Output("series_scalar_mul", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(Equal(6.0))
		})

		It("Should divide series / scalar f64", func() {
			g := singleFunctionGraph("series_scalar_div", types.F64(), `{
				s series f64 := [10.0, 20.0]
				r series f64 := s / 2.0
				return r[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_scalar_div")
			result := h.Output("series_scalar_div", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(Equal(5.0))
		})

		It("Should add scalar + series i32", func() {
			g := singleFunctionGraph("scalar_series_add", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := 10 + s
				return r[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "scalar_series_add")
			result := h.Output("scalar_series_add", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(11)))
		})

		It("Should subtract scalar - series i32", func() {
			g := singleFunctionGraph("scalar_series_sub", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := 10 - s
				return r[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "scalar_series_sub")
			result := h.Output("scalar_series_sub", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(9)))
		})

		It("Should multiply scalar * series f64", func() {
			g := singleFunctionGraph("scalar_series_mul", types.F64(), `{
				s series f64 := [1.0, 2.0, 3.0]
				r series f64 := 2.0 * s
				return r[1]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "scalar_series_mul")
			result := h.Output("scalar_series_mul", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(Equal(4.0))
		})

		It("Should divide scalar / series f64", func() {
			g := singleFunctionGraph("scalar_series_div", types.F64(), `{
				s series f64 := [2.0, 4.0, 5.0]
				r series f64 := 10.0 / s
				return r[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "scalar_series_div")
			result := h.Output("scalar_series_div", 0)
			Expect(telem.UnmarshalSeries[float64](result)[0]).To(Equal(5.0))
		})
	})

	Describe("Series Comparison Operations", func() {
		It("Should compare series < series", func() {
			g := singleFunctionGraph("series_lt", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a < b
				return c[0]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_lt")
			result := h.Output("series_lt", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))
		})

		It("Should compare series > series", func() {
			g := singleFunctionGraph("series_gt", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a > b
				return c[1]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_gt")
			result := h.Output("series_gt", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))
		})

		It("Should compare series == series", func() {
			g := singleFunctionGraph("series_eq", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a == b
				return c[2]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_eq")
			result := h.Output("series_eq", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))
		})

		It("Should compare series != series", func() {
			g := singleFunctionGraph("series_ne", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [1.0, 4.0, 3.0]
				c series u8 := a != b
				return c[1]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_ne")
			result := h.Output("series_ne", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))
		})

		It("Should compare series <= series", func() {
			g := singleFunctionGraph("series_le", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a <= b
				return c[2]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_le")
			result := h.Output("series_le", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))
		})

		It("Should compare series >= series", func() {
			g := singleFunctionGraph("series_ge", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a >= b
				return c[1]
			}`)
			h := newHarness(ctx, g, nil, nil)
			defer h.Close()
			h.Execute(ctx, "series_ge")
			result := h.Output("series_ge", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))
		})
	})

	Describe("Series Length Operations", func() {
		It("Should return 0 for empty series", func() {
			g := singleFunctionGraph("len_empty", types.I32(), `{
				s series f64 := []
				return len(s)
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_empty")
			result := h.Output("len_empty", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(0)))
		})

		It("Should return 1 for single element series", func() {
			g := singleFunctionGraph("len_one", types.I32(), `{
				s series f64 := [1.0]
				return len(s)
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_one")
			result := h.Output("len_one", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(1)))
		})

		It("Should return 5 for five element series", func() {
			g := singleFunctionGraph("len_five", types.I32(), `{
				s series f64 := [1.0, 2.0, 3.0, 4.0, 5.0]
				return len(s)
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_five")
			result := h.Output("len_five", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(5)))
		})

		It("Should return correct length after series operation", func() {
			g := singleFunctionGraph("len_after_op", types.I32(), `{
				a series f64 := [1.0, 2.0, 3.0]
				b series f64 := [4.0, 5.0, 6.0]
				c series f64 := a + b
				return len(c)
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_after_op")
			result := h.Output("len_after_op", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(3)))
		})
	})

	Describe("String Operations Extended", func() {
		It("Should return 0 for empty string length", func() {
			g := singleFunctionGraph("len_empty_str", types.I32(), `{
				return len("")
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_empty_str")
			result := h.Output("len_empty_str", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(0)))
		})

		It("Should return correct length for string", func() {
			g := singleFunctionGraph("len_str", types.I32(), `{
				return len("hello")
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_str")
			result := h.Output("len_str", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(5)))
		})

		It("Should return correct length for concatenated strings", func() {
			g := singleFunctionGraph("len_concat", types.I32(), `{
				return len("ab" + "cd")
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_concat")
			result := h.Output("len_concat", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(4)))
		})

		It("Should return correct length for triple concatenation", func() {
			g := singleFunctionGraph("len_triple", types.I32(), `{
				return len("a" + "b" + "c")
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_triple")
			result := h.Output("len_triple", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(3)))
		})

		It("Should return correct length for variable concatenation", func() {
			g := singleFunctionGraph("len_var_concat", types.I32(), `{
				a str := "hello"
				b str := " world"
				return len(a + b)
			}`)
			h := newHarness(ctx, g, builtin.SymbolResolver, nil)
			defer h.Close()
			h.Execute(ctx, "len_var_concat")
			result := h.Output("len_var_concat", 0)
			Expect(telem.UnmarshalSeries[int32](result)[0]).To(Equal(int32(11)))
		})
	})
})
