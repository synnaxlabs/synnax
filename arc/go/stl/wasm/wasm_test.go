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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channel"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	stltime "github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/stl/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var _ = Describe("ConvertConfigValue", func() {
	DescribeTable("supported numeric and timestamp types",
		func(v any, expected uint64) {
			Expect(wasm.ConvertConfigValue(v)).To(Equal(expected))
		},
		Entry("int8", int8(1), uint64(1)),
		Entry("int16", int16(2), uint64(2)),
		Entry("int32", int32(3), uint64(3)),
		Entry("int64", int64(4), uint64(4)),
		Entry("uint8", uint8(5), uint64(5)),
		Entry("uint16", uint16(6), uint64(6)),
		Entry("uint32", uint32(7), uint64(7)),
		Entry("uint64", uint64(8), uint64(8)),
		Entry("float32", float32(1.5), uint64(math.Float32bits(1.5))),
		Entry("float64", float64(2.5), math.Float64bits(2.5)),
		Entry("telem.TimeStamp", telem.TimeStamp(9), uint64(9)),
	)

	DescribeTable("unsupported types return an error instead of panicking",
		func(v any) {
			_, err := wasm.ConvertConfigValue(v)
			Expect(err).To(HaveOccurred())
		},
		Entry("bool", true),
	)
})

// testHarness encapsulates common test setup for wasm module tests.
type testHarness struct {
	factory      node.Factory
	state        *node.ProgramState
	wasmRT       wazero.Runtime
	guest        api.Module
	channelState *channel.ProgramState
	prog         program.Program
	analyzed     ir.IR
	graph        arc.Graph
}

func (h *testHarness) ChannelState() *channel.ProgramState { return h.channelState }

// newHarness creates a new test harness from a graph definition.
func newHarness(
	g arc.Graph,
	resolver symbol.Resolver,
	channelDigests ...channel.Digest,
) *testHarness {
	var compileResolver symbol.Resolver
	if resolver != nil {
		compileResolver = symbol.CompoundResolver{resolver, stl.SymbolResolver}
	} else {
		compileResolver = stl.SymbolResolver
	}
	prog := MustSucceed(arc.CompileGraph(ctx, g, arc.WithResolver(compileResolver)))
	analyzed, diagnostics := graph.Analyze(ctx, g, resolver)
	Expect(diagnostics.Ok()).To(BeTrue())
	s := node.New(analyzed)

	stringsState := stlstrings.NewProgramState()
	seriesState := series.NewProgramState()
	channelState := channel.NewProgramState(channelDigests)

	wasmRT := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	statefulMod := MustSucceed(stateful.NewModule(ctx, seriesState, stringsState, wasmRT))
	_, _ = series.NewModule(ctx, seriesState, wasmRT)
	stringsMod := MustSucceed(stlstrings.NewModule(ctx, stringsState, wasmRT, nil))
	_, _ = stlmath.NewModule(ctx, wasmRT)
	errorsMod := MustSucceed(stlerrors.NewModule(ctx, nil, wasmRT))
	_, _ = stltime.NewModule(ctx, wasmRT)
	_, _ = channel.NewModule(ctx, channelState, stringsState, wasmRT)

	guest := MustSucceed(wasmRT.Instantiate(ctx, prog.WASM))
	stringsMod.SetMemory(guest.Memory())
	errorsMod.SetMemory(guest.Memory())

	factory := &wasm.Module{
		Module:        guest,
		Memory:        guest.Memory(),
		Strings:       stringsState,
		NodeKeySetter: statefulMod,
	}
	return &testHarness{
		graph:        g,
		prog:         prog,
		analyzed:     analyzed,
		state:        s,
		wasmRT:       wasmRT,
		guest:        guest,
		channelState: channelState,
		factory:      factory,
	}
}

func (h *testHarness) Close() {
	Expect(h.guest.Close(ctx)).To(Succeed())
	Expect(h.wasmRT.Close(ctx)).To(Succeed())
}

func (h *testHarness) SetInput(nodeKey string, idx int, data telem.Series, time telem.Series) {
	n := h.state.Node(nodeKey)
	*n.Output(idx) = data
	*n.OutputTime(idx) = time
}

func (h *testHarness) CreateNode(nodeKey string) node.Node {
	return MustSucceed(h.factory.Create(ctx, node.Config{
		Node:    h.analyzed.Nodes.Get(nodeKey),
		State:   h.state.Node(nodeKey),
		Program: h.prog,
	}))
}

func (h *testHarness) Execute(nodeKey string) set.Set[string] {
	n := h.CreateNode(nodeKey)
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

// newTextHarness creates a test harness from text source code.
func newTextHarness(
	source string,
	resolver symbol.Resolver,
	channelDigests ...channel.Digest,
) *testHarness {
	var compileResolver symbol.Resolver
	if resolver != nil {
		compileResolver = symbol.CompoundResolver{resolver, stl.SymbolResolver}
	} else {
		compileResolver = stl.SymbolResolver
	}
	parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
	analyzed, diagnostics := text.Analyze(ctx, parsedText, compileResolver)
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	prog := MustSucceed(text.Compile(ctx, analyzed, compiler.WithHostSymbols(compileResolver)))
	s := node.New(analyzed)

	stringsState := stlstrings.NewProgramState()
	seriesState := series.NewProgramState()
	channelState := channel.NewProgramState(channelDigests)

	wasmRT := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	statefulMod := MustSucceed(stateful.NewModule(ctx, seriesState, stringsState, wasmRT))
	_, _ = series.NewModule(ctx, seriesState, wasmRT)
	stringsMod := MustSucceed(stlstrings.NewModule(ctx, stringsState, wasmRT, nil))
	_, _ = stlmath.NewModule(ctx, wasmRT)
	errorsMod := MustSucceed(stlerrors.NewModule(ctx, nil, wasmRT))
	_, _ = stltime.NewModule(ctx, wasmRT)
	_, _ = channel.NewModule(ctx, channelState, stringsState, wasmRT)

	guest := MustSucceed(wasmRT.Instantiate(ctx, prog.WASM))
	stringsMod.SetMemory(guest.Memory())
	errorsMod.SetMemory(guest.Memory())

	factory := &wasm.Module{
		Module:        guest,
		Memory:        guest.Memory(),
		Strings:       stringsState,
		NodeKeySetter: statefulMod,
	}
	return &testHarness{
		prog:         prog,
		analyzed:     analyzed,
		state:        s,
		wasmRT:       wasmRT,
		guest:        guest,
		channelState: channelState,
		factory:      factory,
	}
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
func expectOutput[T telem.Sample](key string, outType types.Type, body string, resolver symbol.Resolver, expected T) {
	g := singleFunctionGraph(key, outType, body)
	h := newHarness(g, resolver)
	defer h.Close()
	h.Execute(key)
	result := h.Output(key, 0)
	Expect(telem.UnmarshalSeries[T](result)[0]).To(Equal(expected))
}

var _ = Describe("WASM", func() {
	Describe("Next with mismatched input lengths", func() {
		It("Should repeat shorter input values to match longest input", func() {
			g := binaryOpGraph("add", "lhs", "rhs", types.I64(), types.I64(), `{ return lhs + rhs }`)
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("lhs", 0, telem.NewSeriesV[int64](1, 2, 3, 4, 5), telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5))
			h.SetInput("rhs", 0, telem.NewSeriesV[int64](10, 20), telem.NewSeriesSecondsTSV(1, 2))

			changed := h.Execute("add")
			Expect(changed.Contains(ir.DefaultOutputParam)).To(BeTrue())

			result := h.Output("add", 0)
			Expect(result.Len()).To(Equal(int64(5)))
			Expect(telem.UnmarshalSeries[int64](result)).To(Equal([]int64{11, 22, 13, 24, 15}))
			Expect(h.OutputTime("add", 0)).To(telem.MatchSeries(telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5)))
		})

		It("Should handle equal length inputs correctly", func() {
			g := binaryOpGraph("multiply", "a", "b", types.I32(), types.I32(), `{ return lhs * rhs }`)
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("a", 0, telem.NewSeriesV[int32](2, 3, 4), telem.NewSeriesSecondsTSV(10, 20, 30))
			h.SetInput("b", 0, telem.NewSeriesV[int32](5, 6, 7), telem.NewSeriesSecondsTSV(10, 20, 30))

			h.Execute("multiply")
			result := h.Output("multiply", 0)
			Expect(result.Len()).To(Equal(int64(3)))
			Expect(telem.UnmarshalSeries[int32](result)).To(Equal([]int32{10, 18, 28}))
		})

		It("Should repeat single value input across all iterations", func() {
			g := binaryOpGraph("subtract", "x", "y", types.F32(), types.F32(), `{ return lhs - rhs }`)
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("x", 0, telem.NewSeriesV[float32](100.0, 200.0, 300.0, 400.0), telem.NewSeriesSecondsTSV(5, 10, 15, 20))
			h.SetInput("y", 0, telem.NewSeriesV[float32](25.0), telem.NewSeriesSecondsTSV(5))

			h.Execute("subtract")
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
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("a", 0, telem.NewSeriesV[int64](10, 20, 30), telem.NewSeriesSecondsTSV(1, 2, 3))
			h.SetInput("b", 0, telem.NewSeriesV[int64](5), telem.NewSeriesSecondsTSV(1))

			changed := h.Execute("math_ops")
			Expect(changed.Contains("sum")).To(BeTrue())
			Expect(changed.Contains("product")).To(BeTrue())

			sumResult := h.Output("math_ops", 0)
			Expect(telem.UnmarshalSeries[int64](sumResult)).To(Equal([]int64{15, 25, 35}))

			productResult := h.Output("math_ops", 1)
			Expect(telem.UnmarshalSeries[int64](productResult)).To(Equal([]int64{50, 100, 150}))
		})
	})

	Describe("Runtime Operations - ProgramState Persistence", func() {
		It("Should persist stateful variables across function calls", func() {
			g := singleFunctionGraph("counter", types.I64(), `{
				count i64 $= 0
				count = count + 1
				return count
			}`)
			h := newHarness(g, nil)
			defer h.Close()

			n := h.CreateNode("counter")

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
			h := newHarness(g, nil)
			defer h.Close()

			n1 := h.CreateNode("c1")
			n2 := h.CreateNode("c2")
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

		It("Should isolate stateful variables between nodes of the same function", func() {
			// This test verifies that two node instances of the same function type
			// have separate state storage (important fix for node instance isolation)
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "counter",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
						Body: ir.Body{Raw: `{
							count i64 $= 0
							count = count + 1
							return count
						}`},
					},
				},
				Nodes: []graph.Node{
					{Key: "counter_a", Type: "counter"},
					{Key: "counter_b", Type: "counter"},
				},
			}
			h := newHarness(g, nil)
			defer h.Close()

			n1 := h.CreateNode("counter_a")
			n2 := h.CreateNode("counter_b")
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

			// First execution of counter_a should return 1
			n1.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("counter_a", 0))[0]).To(Equal(int64(1)))

			// First execution of counter_b should ALSO return 1 (not 2!)
			// because it has its own separate state
			n2.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("counter_b", 0))[0]).To(Equal(int64(1)))

			// Second execution of counter_a should return 2
			n1.Reset()
			n1.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("counter_a", 0))[0]).To(Equal(int64(2)))

			// Second execution of counter_b should return 2 (its own count)
			n2.Reset()
			n2.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("counter_b", 0))[0]).To(Equal(int64(2)))

			// Third execution of counter_a should return 3
			n1.Reset()
			n1.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("counter_a", 0))[0]).To(Equal(int64(3)))

			// counter_b should still be at 2 (we didn't call it again)
			Expect(telem.UnmarshalSeries[int64](h.Output("counter_b", 0))[0]).To(Equal(int64(2)))
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
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("x", 0, telem.NewSeriesV[int64](5, 15, 25), telem.NewSeriesSecondsTSV(1, 2, 3))
			h.Execute("add")
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
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("a", 0, telem.NewSeriesV[int32](5, 10), telem.NewSeriesSecondsTSV(1, 2))
			h.Execute("compute")
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
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("value", 0, telem.NewSeriesV[float64](10.0, 20.0), telem.NewSeriesSecondsTSV(1, 2))
			h.Execute("scale")
			Expect(telem.UnmarshalSeries[float64](h.Output("scale", 0))).To(Equal([]float64{25.0, 50.0}))
		})

		It("Should allow overriding optional parameter with connected edge", func() {
			g := binaryOpGraph("add", "x", "y", types.I64(), types.I64(), `{ return lhs + rhs }`)
			// Modify to add optional value
			g.Functions[0].Inputs[1].Value = int64(10)

			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("x", 0, telem.NewSeriesV[int64](5), telem.NewSeriesSecondsTSV(1))
			h.SetInput("y", 0, telem.NewSeriesV[int64](100), telem.NewSeriesSecondsTSV(1))
			h.Execute("add")
			Expect(telem.UnmarshalSeries[int64](h.Output("add", 0))).To(Equal([]int64{105}))
		})
	})

	Describe("Alignment and TimeRange Propagation", func() {
		It("Should sum alignments from multiple inputs and propagate to outputs", func() {
			g := binaryOpGraph("add", "lhs", "rhs", types.I64(), types.I64(), `{ return lhs + rhs }`)
			h := newHarness(g, nil)
			defer h.Close()

			lhsSeries := telem.NewSeriesV[int64](1, 2, 3)
			lhsSeries.Alignment = 100
			lhsSeries.TimeRange = telem.TimeRange{Start: 10 * telem.SecondTS, End: 30 * telem.SecondTS}
			h.SetInput("lhs", 0, lhsSeries, telem.NewSeriesSecondsTSV(10, 20, 30))

			rhsSeries := telem.NewSeriesV[int64](10, 20, 30)
			rhsSeries.Alignment = 50
			rhsSeries.TimeRange = telem.TimeRange{Start: 5 * telem.SecondTS, End: 25 * telem.SecondTS}
			h.SetInput("rhs", 0, rhsSeries, telem.NewSeriesSecondsTSV(5, 15, 25))

			h.Execute("add")

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
			h := newHarness(g, nil)
			defer h.Close()

			aSeries := telem.NewSeriesV[int64](2, 3)
			aSeries.Alignment = 200
			aSeries.TimeRange = telem.TimeRange{Start: 100 * telem.SecondTS, End: 200 * telem.SecondTS}
			h.SetInput("a", 0, aSeries, telem.NewSeriesSecondsTSV(100, 150))

			bSeries := telem.NewSeriesV[int64](5, 10)
			bSeries.Alignment = 300
			bSeries.TimeRange = telem.TimeRange{Start: 50 * telem.SecondTS, End: 250 * telem.SecondTS}
			h.SetInput("b", 0, bSeries, telem.NewSeriesSecondsTSV(50, 200))

			h.Execute("math_ops")

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
			h := newHarness(g, nil)
			defer h.Close()

			n := h.CreateNode("series_state")
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
				expectOutput("empty_series", types.I64(), `{
				s series `+elemType+` := []
				return len(s)
			}`, stl.SymbolResolver, int32(0))
			},
			Entry("i32", "i32", types.I32()),
			Entry("f64", "f64", types.F64()),
		)

		DescribeTable("single element series",
			expectOutput[int32],
			Entry("i32", "single_elem", types.I32(), `{
				s series i32 := [42]
				return s[0]
			}`, nil, int32(42)),
		)

		It("Should handle single element f64 series", func() {
			expectOutput("single_elem", types.F64(), `{
				s series f64 := [3.14]
				return s[0]
			}`, nil, 3.14)
		})

		It("Should handle 10 element series access", func() {
			expectOutput("ten_elem", types.I32(), `{
				s series i32 := [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
				return s[9]
			}`, nil, int32(9))
		})

		It("Should sum elements from 10 element series", func() {
			expectOutput("sum_ten", types.I32(), `{
				s series i32 := [1, 1, 1, 1, 1, 1, 1, 1, 1, 1]
				return s[0] + s[1] + s[2] + s[3] + s[4] + s[5] + s[6] + s[7] + s[8] + s[9]
			}`, nil, int32(10))
		})
	})

	Describe("Series Arithmetic Per Type", func() {
		DescribeTable("i32 arithmetic",
			expectOutput[int32],
			Entry("add", "add_i32", types.I32(), `{
				a series i32 := [1, 2, 3]
				b series i32 := [10, 20, 30]
				c series i32 := a + b
				return c[0]
			}`, nil, int32(11)),
			Entry("subtract", "sub_i32", types.I32(), `{
				a series i32 := [10, 20, 30]
				b series i32 := [1, 2, 3]
				c series i32 := a - b
				return c[2]
			}`, nil, int32(27)),
			Entry("multiply", "mul_i32", types.I32(), `{
				a series i32 := [2, 3, 4]
				b series i32 := [5, 6, 7]
				c series i32 := a * b
				return c[1]
			}`, nil, int32(18)),
			Entry("divide", "div_i32", types.I32(), `{
				a series i32 := [10, 20, 30]
				b series i32 := [2, 4, 5]
				c series i32 := a / b
				return c[0]
			}`, nil, int32(5)),
		)

		It("Should add i64 series", func() {
			expectOutput("add_i64", types.I64(), `{
				a series i64 := [1, 2, 3]
				b series i64 := [10, 20, 30]
				c series i64 := a + b
				return c[0]
			}`, nil, int64(11))
		})

		DescribeTable("f32 arithmetic",
			expectOutput[float32],
			Entry("add", "add_f32", types.F32(), `{
				a series f32 := [1.0, 2.0, 3.0]
				b series f32 := [0.5, 0.5, 0.5]
				c series f32 := a + b
				return c[0]
			}`, nil, float32(1.5)),
			Entry("multiply", "mul_f32", types.F32(), `{
				a series f32 := [2.0, 3.0]
				b series f32 := [1.5, 2.5]
				c series f32 := a * b
				return c[0]
			}`, nil, float32(3.0)),
		)

		It("Should mod f64 series", func() {
			expectOutput("mod_f64", types.F64(), `{
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
			Entry("add", "series_scalar_add", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := s + 10
				return r[0]
			}`, nil, int32(11)),
			Entry("subtract", "series_scalar_sub", types.I32(), `{
				s series i32 := [10, 20, 30]
				r series i32 := s - 5
				return r[1]
			}`, nil, int32(15)),
		)

		DescribeTable("series op scalar (f64)",
			expectOutput[float64],
			Entry("multiply", "series_scalar_mul", types.F64(), `{
				s series f64 := [1.0, 2.0, 3.0]
				r series f64 := s * 2.0
				return r[2]
			}`, nil, 6.0),
			Entry("divide", "series_scalar_div", types.F64(), `{
				s series f64 := [10.0, 20.0]
				r series f64 := s / 2.0
				return r[0]
			}`, nil, 5.0),
		)

		DescribeTable("scalar op series (i32)",
			expectOutput[int32],
			Entry("add", "scalar_series_add", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := 10 + s
				return r[0]
			}`, nil, int32(11)),
			Entry("subtract", "scalar_series_sub", types.I32(), `{
				s series i32 := [1, 2, 3]
				r series i32 := 10 - s
				return r[0]
			}`, nil, int32(9)),
		)

		DescribeTable("scalar op series (f64)",
			expectOutput[float64],
			Entry("multiply", "scalar_series_mul", types.F64(), `{
				s series f64 := [1.0, 2.0, 3.0]
				r series f64 := 2.0 * s
				return r[1]
			}`, nil, 4.0),
			Entry("divide", "scalar_series_div", types.F64(), `{
				s series f64 := [2.0, 4.0, 5.0]
				r series f64 := 10.0 / s
				return r[0]
			}`, nil, 5.0),
		)
	})

	Describe("Series Comparison Operations", func() {
		DescribeTable("comparison operators",
			expectOutput[uint8],
			Entry("less than", "series_lt", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a < b
				return c[0]
			}`, nil, uint8(1)),
			Entry("greater than", "series_gt", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a > b
				return c[1]
			}`, nil, uint8(1)),
			Entry("equal", "series_eq", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a == b
				return c[2]
			}`, nil, uint8(1)),
			Entry("not equal", "series_ne", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [1.0, 4.0, 3.0]
				c series u8 := a != b
				return c[1]
			}`, nil, uint8(1)),
			Entry("less than or equal", "series_le", types.U8(), `{
				a series f64 := [1.0, 5.0, 3.0]
				b series f64 := [2.0, 4.0, 3.0]
				c series u8 := a <= b
				return c[2]
			}`, nil, uint8(1)),
			Entry("greater than or equal", "series_ge", types.U8(), `{
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
			Entry("empty series", "len_empty", types.I64(), `{
				s series f64 := []
				return len(s)
			}`, stl.SymbolResolver, int32(0)),
			Entry("single element", "len_one", types.I64(), `{
				s series f64 := [1.0]
				return len(s)
			}`, stl.SymbolResolver, int32(1)),
			Entry("five elements", "len_five", types.I64(), `{
				s series f64 := [1.0, 2.0, 3.0, 4.0, 5.0]
				return len(s)
			}`, stl.SymbolResolver, int32(5)),
			Entry("after operation", "len_after_op", types.I64(), `{
				a series f64 := [1.0, 2.0, 3.0]
				b series f64 := [4.0, 5.0, 6.0]
				c series f64 := a + b
				return len(c)
			}`, stl.SymbolResolver, int32(3)),
		)
	})

	Describe("String Operations Extended", func() {
		DescribeTable("string len() function",
			expectOutput[int32],
			Entry("empty string", "len_empty_str", types.I64(), `{
				return len("")
			}`, stl.SymbolResolver, int32(0)),
			Entry("simple string", "len_str", types.I64(), `{
				return len("hello")
			}`, stl.SymbolResolver, int32(5)),
			Entry("concatenated strings", "len_concat", types.I64(), `{
				return len("ab" + "cd")
			}`, stl.SymbolResolver, int32(4)),
			Entry("triple concatenation", "len_triple", types.I64(), `{
				return len("a" + "b" + "c")
			}`, stl.SymbolResolver, int32(3)),
			Entry("variable concatenation", "len_var_concat", types.I64(), `{
				a str := "hello"
				b str := " world"
				return len(a + b)
			}`, stl.SymbolResolver, int32(11)),
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
			h := newHarness(g, nil)
			defer h.Close()

			n := h.CreateNode("init_counter")

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
			h := newHarness(g, nil)
			defer h.Close()

			n := h.CreateNode("add")

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
			h := newHarness(g, nil)
			defer h.Close()

			// Set up input source output
			h.SetInput("input_source", 0, telem.NewSeriesV[int64](5), telem.NewSeriesSecondsTSV(1))

			n := h.CreateNode("add_config")
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
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("input_source", 0, telem.NewSeriesV[int32](3), telem.NewSeriesSecondsTSV(1))

			n := h.CreateNode("multi_config")
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
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("input_source", 0, telem.NewSeriesV[float64](10.0), telem.NewSeriesSecondsTSV(1))

			n := h.CreateNode("scale_config")
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 100, DataType: telem.Int32T},
				)
				defer h.Close()

				h.Execute("write_test")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 100, Index: 101, DataType: telem.Int32T},
				)
				defer h.Close()

				h.Execute("write_indexed")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 200, Index: 201, DataType: telem.Int32T},
				)
				defer h.Close()

				before := telem.Now()
				h.Execute("write_ts")
				after := telem.Now()

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 10, Index: 11, DataType: telem.Int32T},
					channel.Digest{Key: 20, Index: 21, DataType: telem.Int32T},
				)
				defer h.Close()

				h.Execute("multi_write")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 300, Index: 301, DataType: telem.Int32T},
				)
				defer h.Close()

				n := h.CreateNode("seq_write")
				timestamps := make([]telem.TimeStamp, 3)

				for i := range 3 {
					n.Reset()
					n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
					fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 700, Index: 701, DataType: telem.Int32T},
				)
				defer h.Close()

				h.Execute("i32_write")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 800, Index: 801, DataType: telem.Uint8T},
				)
				defer h.Close()

				h.Execute("u8_write")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 1100, Index: 1101, DataType: telem.Float64T},
				)
				defer h.Close()

				h.Execute("f64_write")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 1200, Index: 1201, DataType: telem.Float32T},
				)
				defer h.Close()

				h.Execute("f32_write")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
				Expect(changed).To(BeTrue())
				Expect(fr.Get(1200).Series).To(HaveLen(1))
				Expect(fr.Get(1200).Series[0]).To(telem.MatchSeriesDataV[float32](2.718))
				Expect(fr.Get(1201).Series).To(HaveLen(1))
			})
		})

		Describe("Edge Cases", func() {
			It("Should handle empty flush when no writes occur", func() {
				g := singleFunctionGraph("no_write", types.I32(), `{ return 42 }`)
				h := newHarness(g, nil)
				defer h.Close()

				h.Execute("no_write")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 900, Index: 0, DataType: telem.Int32T},
				)
				defer h.Close()

				h.Execute("zero_idx")

				fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
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

				h := newHarness(g, resolver,
					channel.Digest{Key: 1000, Index: 1001, DataType: telem.Int32T},
				)
				defer h.Close()

				h.Execute("imperative_vs_decl")

				fr, _ := h.ChannelState().Flush(telem.Frame[uint32]{})
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
			h := newHarness(g, nil)
			defer h.Close()

			h.SetInput("trigger_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(1))

			changed := h.Execute("void_func")
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
			h := newHarness(g, resolver, channel.Digest{Key: 100, DataType: telem.Int32T})
			defer h.Close()

			h.SetInput("trigger_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(1))

			n := h.CreateNode("void_with_state")
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

			n.Reset()
			n.Next(nCtx)
			fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(100).Series[0]).To(telem.MatchSeriesDataV[int32](1))

			h.SetInput("trigger_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(2))
			n.Next(nCtx)
			fr, changed = h.ChannelState().Flush(telem.Frame[uint32]{})
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
			h := newHarness(g, nil)
			defer h.Close()

			n := h.CreateNode("expression_0")
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
			h := newHarness(g, nil)
			defer h.Close()

			n := h.CreateNode("expression_0")
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
			h := newHarness(g, nil)
			defer h.Close()

			n := h.CreateNode("expr_0")
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expr_0", 0))[0]).To(Equal(int64(1)))

			n.Next(nCtx)
			Expect(telem.UnmarshalSeries[int64](h.Output("expr_0", 0))[0]).To(Equal(int64(1)))
		})
	})

	Describe("Channel Config Parameter Arithmetic", func() {
		// Regression test for: "cannot pop the 2nd f32 operand for f32.add:
		// type mismatch: expected f32, but was i32"
		// Bug occurred when reading from a channel config parameter and performing arithmetic.
		It("Should read from channel config param and perform f32 arithmetic", func() {
			resolver := symbol.MapResolver{
				"do_0_counter": {
					Name: "do_0_counter",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
			}

			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "increment_counter",
						Config:  types.Params{{Name: "counter", Type: types.Chan(types.F32())}},
						Inputs:  types.Params{},
						Outputs: types.Params{},
						Body: ir.Body{Raw: `{
							counter = counter + 1.0
						}`},
					},
				},
				Nodes: []graph.Node{
					{Key: "increment_counter", Type: "increment_counter", Config: map[string]any{"counter": uint32(100)}},
				},
				Edges: []graph.Edge{},
			}

			h := newHarness(g, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
			)
			defer h.Close()

			// Ingest initial channel value (5.0), execute, expect write (6.0)
			fr := telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV[float32](5.0))
			h.ChannelState().Ingest(fr)
			h.Execute("increment_counter")
			outFr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outFr.Get(100).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float32](outFr.Get(100).Series[0])[0]).To(Equal(float32(6.0)))
		})

		// Test matching the user's original example with stateful variable and conditional
		It("Should handle channel config param with stateful variable and conditional", func() {
			resolver := symbol.MapResolver{
				"do_0_counter": {
					Name: "do_0_counter",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
			}

			// Original example: func count_rising{counter chan f32}(input u8) {
			//     prev u8 $= input
			//     if input != 0 && prev == 0 { counter = counter + 1.0 }
			//     prev = input
			// }
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key:     "count_rising",
						Config:  types.Params{{Name: "counter", Type: types.Chan(types.F32())}},
						Inputs:  types.Params{{Name: "input", Type: types.U8()}},
						Outputs: types.Params{},
						Body: ir.Body{Raw: `{
							prev u8 $= input
							if input != 0 and prev == 0 {
								counter = counter + 1.0
							}
							prev = input
						}`},
					},
					{
						Key:     "input_source",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
						Body:    ir.Body{Raw: `{ return 0 }`},
					},
				},
				Nodes: []graph.Node{
					{Key: "input_source", Type: "input_source"},
					{Key: "count_rising", Type: "count_rising", Config: map[string]any{"counter": uint32(100)}},
				},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "input_source", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "count_rising", Param: "input"}},
				},
			}

			h := newHarness(g, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
			)
			defer h.Close()

			// Initial state: counter=0, input=0, prev initializes to 0
			fr := telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV[float32](0.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("input_source", 0, telem.NewSeriesV[uint8](0), telem.NewSeriesSecondsTSV(1))
			h.Execute("count_rising")
			outFr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeFalse()) // input=0, no rising edge
			Expect(outFr.Get(100).Series).To(HaveLen(0))

			// Rising edge: input goes 0->1, prev=0, should increment
			fr = telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV[float32](0.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("input_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(2))
			h.Execute("count_rising")
			outFr, changed = h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outFr.Get(100).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float32](outFr.Get(100).Series[0])[0]).To(Equal(float32(1.0)))

			// Stay high: input=1, prev=1, no rising edge
			fr = telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV[float32](1.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("input_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(3))
			h.Execute("count_rising")
			outFr, changed = h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeFalse()) // No rising edge
			Expect(outFr.Get(100).Series).To(HaveLen(0))

			// Falling edge then rising: input 1->0->1
			fr = telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV[float32](1.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("input_source", 0, telem.NewSeriesV[uint8](0), telem.NewSeriesSecondsTSV(4))
			h.Execute("count_rising")
			outFr, changed = h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeFalse()) // Falling edge, no increment
			Expect(outFr.Get(100).Series).To(HaveLen(0))

			// Another rising edge
			fr = telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV[float32](1.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("input_source", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(5))
			h.Execute("count_rising")
			outFr, changed = h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outFr.Get(100).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float32](outFr.Get(100).Series[0])[0]).To(Equal(float32(2.0)))
		})

		It("Should handle multiple channel config parameters", func() {
			resolver := symbol.MapResolver{
				"temp_sensor": {
					Name: "temp_sensor",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
				"pressure_sensor": {
					Name: "pressure_sensor",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   101,
				},
				"output_sum": {
					Name: "output_sum",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   102,
				},
			}

			// Function that reads from two channel config params and writes their sum to a third
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "combine_sensors",
						Config: types.Params{
							{Name: "temp", Type: types.Chan(types.F32())},
							{Name: "pressure", Type: types.Chan(types.F32())},
							{Name: "result", Type: types.Chan(types.F32())},
						},
						Inputs:  types.Params{},
						Outputs: types.Params{},
						Body: ir.Body{Raw: `{
							result = temp + pressure
						}`},
					},
				},
				Nodes: []graph.Node{
					{
						Key:  "combine_sensors",
						Type: "combine_sensors",
						Config: map[string]any{
							"temp":     uint32(100),
							"pressure": uint32(101),
							"result":   uint32(102),
						},
					},
				},
				Edges: []graph.Edge{},
			}

			h := newHarness(g, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 101, DataType: telem.Float32T},
				channel.Digest{Key: 102, DataType: telem.Float32T},
			)
			defer h.Close()

			// Test: temp=25.5, pressure=101.3, expect result=126.8
			fr := telem.Frame[uint32]{}
			fr = fr.Append(100, telem.NewSeriesV[float32](25.5))
			fr = fr.Append(101, telem.NewSeriesV[float32](101.3))
			h.ChannelState().Ingest(fr)
			h.Execute("combine_sensors")
			outFr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outFr.Get(102).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float32](outFr.Get(102).Series[0])[0]).To(BeNumerically("~", float32(126.8), 0.01))
		})

		It("Should handle multiple channel config params with different operations", func() {
			resolver := symbol.MapResolver{
				"input_a": {
					Name: "input_a",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
					ID:   200,
				},
				"input_b": {
					Name: "input_b",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
					ID:   201,
				},
				"out_sum": {
					Name: "out_sum",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
					ID:   202,
				},
				"out_diff": {
					Name: "out_diff",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
					ID:   203,
				},
				"out_product": {
					Name: "out_product",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
					ID:   204,
				},
			}

			// Function that performs multiple operations on channel config params
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "multi_op",
						Config: types.Params{
							{Name: "a", Type: types.Chan(types.F64())},
							{Name: "b", Type: types.Chan(types.F64())},
							{Name: "sum", Type: types.Chan(types.F64())},
							{Name: "diff", Type: types.Chan(types.F64())},
							{Name: "product", Type: types.Chan(types.F64())},
						},
						Inputs:  types.Params{},
						Outputs: types.Params{},
						Body: ir.Body{Raw: `{
							sum = a + b
							diff = a - b
							product = a * b
						}`},
					},
				},
				Nodes: []graph.Node{
					{
						Key:  "multi_op",
						Type: "multi_op",
						Config: map[string]any{
							"a":       uint32(200),
							"b":       uint32(201),
							"sum":     uint32(202),
							"diff":    uint32(203),
							"product": uint32(204),
						},
					},
				},
				Edges: []graph.Edge{},
			}

			h := newHarness(g, resolver,
				channel.Digest{Key: 200, DataType: telem.Float64T},
				channel.Digest{Key: 201, DataType: telem.Float64T},
				channel.Digest{Key: 202, DataType: telem.Float64T},
				channel.Digest{Key: 203, DataType: telem.Float64T},
				channel.Digest{Key: 204, DataType: telem.Float64T},
			)
			defer h.Close()

			// Test: a=10.0, b=3.0
			// Expected: sum=13.0, diff=7.0, product=30.0
			fr := telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float64](10.0))
			fr = fr.Append(201, telem.NewSeriesV[float64](3.0))
			h.ChannelState().Ingest(fr)
			h.Execute("multi_op")
			outFr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())

			Expect(outFr.Get(202).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float64](outFr.Get(202).Series[0])[0]).To(Equal(float64(13.0)))

			Expect(outFr.Get(203).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float64](outFr.Get(203).Series[0])[0]).To(Equal(float64(7.0)))

			Expect(outFr.Get(204).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float64](outFr.Get(204).Series[0])[0]).To(Equal(float64(30.0)))
		})

		It("Should handle channel config param used multiple times in expression", func() {
			resolver := symbol.MapResolver{
				"value_ch": {
					Name: "value_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   300,
				},
				"squared_ch": {
					Name: "squared_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   301,
				},
			}

			// Function that reads from a channel config param twice (squaring it)
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "square_value",
						Config: types.Params{
							{Name: "value", Type: types.Chan(types.F32())},
							{Name: "squared", Type: types.Chan(types.F32())},
						},
						Inputs:  types.Params{},
						Outputs: types.Params{},
						Body: ir.Body{Raw: `{
							squared = value * value
						}`},
					},
				},
				Nodes: []graph.Node{
					{
						Key:  "square_value",
						Type: "square_value",
						Config: map[string]any{
							"value":   uint32(300),
							"squared": uint32(301),
						},
					},
				},
				Edges: []graph.Edge{},
			}

			h := newHarness(g, resolver,
				channel.Digest{Key: 300, DataType: telem.Float32T},
				channel.Digest{Key: 301, DataType: telem.Float32T},
			)
			defer h.Close()

			// Test: value=7.0, expect squared=49.0
			fr := telem.Frame[uint32]{}
			fr = fr.Append(300, telem.NewSeriesV[float32](7.0))
			h.ChannelState().Ingest(fr)
			h.Execute("square_value")
			outFr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outFr.Get(301).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float32](outFr.Get(301).Series[0])[0]).To(Equal(float32(49.0)))

			// Test: value=0.5, expect squared=0.25
			fr = telem.Frame[uint32]{}
			fr = fr.Append(300, telem.NewSeriesV[float32](0.5))
			h.ChannelState().Ingest(fr)
			h.Execute("square_value")
			outFr, changed = h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outFr.Get(301).Series).To(HaveLen(1))
			Expect(telem.UnmarshalSeries[float32](outFr.Get(301).Series[0])[0]).To(Equal(float32(0.25)))
		})

		It("Should handle mixed channel and non-channel config params", func() {
			// This tests the tolerance_alarm pattern: some config params are channels,
			// others are plain values (f32, i64)
			resolver := symbol.MapResolver{
				"set_point_ch": {
					Name: "set_point_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   400,
				},
			}

			// Simplified tolerance alarm: checks if value is above set_point + tolerance
			g := arc.Graph{
				Functions: []ir.Function{
					{
						Key: "tolerance_check",
						Config: types.Params{
							{Name: "tolerance_upper", Type: types.F32()},
							{Name: "tolerance_lower", Type: types.F32()},
							{Name: "set_point", Type: types.Chan(types.F32())},
							{Name: "samples", Type: types.I64()},
						},
						Inputs:  types.Params{{Name: "value", Type: types.F32()}},
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
						Body: ir.Body{Raw: `{
							count i64 $= 0
							upper_limit f32 := set_point + tolerance_upper
							lower_limit f32 := set_point - tolerance_lower
							if value >= upper_limit {
								count = count + 1
							} else if value <= lower_limit {
								count = count + 1
							} else {
								count = 0
							}
							if count >= samples {
								return 1
							}
							return 0
						}`},
					},
					{
						Key:     "value_source",
						Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F32()}},
						Body:    ir.Body{Raw: `{ return 0.0 }`},
					},
				},
				Nodes: []graph.Node{
					{Key: "value_source", Type: "value_source"},
					{
						Key:  "tolerance_check",
						Type: "tolerance_check",
						Config: map[string]any{
							"tolerance_upper": float32(10.0),
							"tolerance_lower": float32(5.0),
							"set_point":       uint32(400),
							"samples":         int64(3),
						},
					},
				},
				Edges: []graph.Edge{
					{Source: ir.Handle{Node: "value_source", Param: ir.DefaultOutputParam}, Target: ir.Handle{Node: "tolerance_check", Param: "value"}},
				},
			}

			h := newHarness(g, resolver,
				channel.Digest{Key: 400, DataType: telem.Float32T},
			)
			defer h.Close()

			// set_point=100.0, tolerance_upper=10.0, tolerance_lower=5.0
			// upper_limit = 110.0, lower_limit = 95.0
			// samples=3 means we need 3 consecutive violations to alarm

			// Test 1: value=105 (within limits), should return 0
			fr := telem.Frame[uint32]{}
			fr = fr.Append(400, telem.NewSeriesV[float32](100.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("value_source", 0, telem.NewSeriesV[float32](105.0), telem.NewSeriesSecondsTSV(1))
			h.Execute("tolerance_check")
			result := h.Output("tolerance_check", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(0)))

			// Test 2: value=115 (above upper limit), count=1, should return 0
			fr = telem.Frame[uint32]{}
			fr = fr.Append(400, telem.NewSeriesV[float32](100.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("value_source", 0, telem.NewSeriesV[float32](115.0), telem.NewSeriesSecondsTSV(2))
			h.Execute("tolerance_check")
			result = h.Output("tolerance_check", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(0)))

			// Test 3: value=115 again, count=2, should return 0
			fr = telem.Frame[uint32]{}
			fr = fr.Append(400, telem.NewSeriesV[float32](100.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("value_source", 0, telem.NewSeriesV[float32](115.0), telem.NewSeriesSecondsTSV(3))
			h.Execute("tolerance_check")
			result = h.Output("tolerance_check", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(0)))

			// Test 4: value=115 again, count=3 >= samples, should return 1 (alarm!)
			fr = telem.Frame[uint32]{}
			fr = fr.Append(400, telem.NewSeriesV[float32](100.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("value_source", 0, telem.NewSeriesV[float32](115.0), telem.NewSeriesSecondsTSV(4))
			h.Execute("tolerance_check")
			result = h.Output("tolerance_check", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))
		})

		It("Should handle intermediate variable assignment from channel config param", func() {
			// This is the EXACT user code that was failing.
			// The key pattern is: sp := set_point (where set_point is chan f32)
			resolver := symbol.MapResolver{
				"input_val": {
					Name: "input_val",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
				"set_point_ch": {
					Name: "set_point_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   200,
				},
				"output_ch": {
					Name: "output_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   300,
				},
			}

			source := `
func tolerance_alarm{
    tolerance_upper f32,
    tolerance_lower f32,
    set_point chan f32,
    samples i64
} (value f32) u8 {
    count i64 $= 0
    sp := set_point

    if value >= (sp + tolerance_upper) {
        count = count + 1
    } else if value <= (sp - tolerance_lower) {
        count = count + 1
    } else {
        count = 0
    }

    if count >= samples {
        return 1
    }
    return 0
}

input_val -> tolerance_alarm{tolerance_upper=10.0, tolerance_lower=5.0, set_point=set_point_ch, samples=3} -> output_ch
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 200, DataType: telem.Float32T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close()

			// set_point_ch has value 100.0
			// tolerance_upper=10.0, tolerance_lower=5.0, samples=3
			// upper = sp + tolerance_upper = 100.0 + 10.0 = 110.0
			// lower = sp - tolerance_lower = 100.0 - 5.0 = 95.0

			// Test 1: value=105 (within limits), should return 0
			fr := telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float32](100.0)) // set_point_ch = 100.0
			h.ChannelState().Ingest(fr)
			h.SetInput("on_input_val_0", 0, telem.NewSeriesV[float32](105.0), telem.NewSeriesSecondsTSV(1))
			h.Execute("tolerance_alarm_0")
			result := h.Output("tolerance_alarm_0", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(0)))

			// Test 2: value=115 (above upper=110), count=1, should return 0
			fr = telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float32](100.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("on_input_val_0", 0, telem.NewSeriesV[float32](115.0), telem.NewSeriesSecondsTSV(2))
			h.Execute("tolerance_alarm_0")
			result = h.Output("tolerance_alarm_0", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(0)))

			// Test 3: value=115 again, count=2, should return 0
			fr = telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float32](100.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("on_input_val_0", 0, telem.NewSeriesV[float32](115.0), telem.NewSeriesSecondsTSV(3))
			h.Execute("tolerance_alarm_0")
			result = h.Output("tolerance_alarm_0", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(0)))

			// Test 4: value=115 again, count=3 >= samples, should return 1 (alarm!)
			fr = telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float32](100.0))
			h.ChannelState().Ingest(fr)
			h.SetInput(
				"on_input_val_0",
				0,
				telem.NewSeriesV[float32](115.0),
				telem.NewSeriesSecondsTSV(4),
			)
			h.Execute("tolerance_alarm_0")
			result = h.Output("tolerance_alarm_0", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))

			// Test 5: Change set_point to 200.0, value=198 now within limits
			// upper = 200.0 + 10.0 = 210.0
			// lower = 200.0 - 5.0 = 195.0
			// value = 198.0 is between 195 and 210, so count resets to 0
			fr = telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float32](200.0)) // set_point_ch = 200.0
			h.ChannelState().Ingest(fr)
			h.SetInput(
				"on_input_val_0",
				0,
				telem.NewSeriesV[float32](198.0),
				telem.NewSeriesSecondsTSV(5),
			)
			h.Execute("tolerance_alarm_0")
			result = h.Output("tolerance_alarm_0", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(0)))
		})

		It("Should handle writing to channel through intermediate variable", func() {
			// Test that writing to an intermediate variable correctly writes to the channel
			// out := output (config param with channel type)
			// out = value * 2.0 (write to channel through intermediate variable)
			resolver := symbol.MapResolver{
				"input_ch": {
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
				"write_target": {
					Name: "write_target",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   200,
				},
				"sink_ch": {
					Name: "sink_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   300,
				},
			}

			source := `
func writer{
    output chan f32
} (value f32) u8 {
    out := output
    out = value * 2.0
    return 0
}

input_ch -> writer{output=write_target} -> sink_ch
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 200, DataType: telem.Float32T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close()

			// Set input value to 25.0, expect write_target to receive 50.0 (25 * 2)
			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[float32](25.0), telem.NewSeriesSecondsTSV(1))
			h.Execute("writer_0")

			// Check that the channel was written to with the correct value
			fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(200).Series).To(HaveLen(1))
			Expect(fr.Get(200).Series[0]).To(telem.MatchSeriesDataV[float32](50.0))
		})

		It("Should handle nested intermediate variable assignments from channel config param", func() {
			// Test that we can chain intermediate variable assignments:
			// out := output      (from config param)
			// out2 := out        (from intermediate variable)
			// out2 = value * 3.0 (write through second intermediate)
			resolver := symbol.MapResolver{
				"input_ch": {
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
				"write_target": {
					Name: "write_target",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   200,
				},
				"sink_ch": {
					Name: "sink_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   300,
				},
			}

			source := `
func writer{
    output chan f32
} (value f32) u8 {
    out := output
    out2 := out
    out2 = value * 3.0
    return 0
}

input_ch -> writer{output=write_target} -> sink_ch
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 200, DataType: telem.Float32T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close()

			// Set input value to 10.0, expect write_target to receive 30.0 (10 * 3)
			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[float32](10.0), telem.NewSeriesSecondsTSV(1))
			h.Execute("writer_0")

			// Check that the channel was written to with the correct value
			fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(200).Series).To(HaveLen(1))
			Expect(fr.Get(200).Series[0]).To(telem.MatchSeriesDataV[float32](30.0))
		})

		It("Should handle writing to channel through global channel alias", func() {
			// Test that writing through an alias of a global channel works correctly
			// out := output_ch   (global channel alias)
			// out = value * 4.0  (write to channel through alias)
			resolver := symbol.MapResolver{
				"input_ch": {
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
				"output_ch": {
					Name: "output_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   200,
				},
				"sink_ch": {
					Name: "sink_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   300,
				},
			}

			source := `
func writer{} (value f32) u8 {
    out := output_ch
    out = value * 4.0
    return 0
}

input_ch -> writer{} -> sink_ch
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 200, DataType: telem.Float32T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close()

			// Set input value to 5.0, expect output_ch to receive 20.0 (5 * 4)
			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[float32](5.0), telem.NewSeriesSecondsTSV(1))
			h.Execute("writer_0")

			// Check that the channel was written to with the correct value
			fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(200).Series).To(HaveLen(1))
			Expect(fr.Get(200).Series[0]).To(telem.MatchSeriesDataV[float32](20.0))
		})

		It("Should handle triple nested aliases from global channel", func() {
			// Test deeply nested alias chain:
			// a := global_ch
			// b := a
			// c := b
			// c = value * 5.0
			resolver := symbol.MapResolver{
				"input_ch": {
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
				"output_ch": {
					Name: "output_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   200,
				},
				"sink_ch": {
					Name: "sink_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   300,
				},
			}

			source := `
func writer{} (value f32) u8 {
    a := output_ch
    b := a
    c := b
    c = value * 5.0
    return 0
}

input_ch -> writer{} -> sink_ch
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 200, DataType: telem.Float32T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close()

			// Set input value to 4.0, expect output_ch to receive 20.0 (4 * 5)
			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[float32](4.0), telem.NewSeriesSecondsTSV(1))
			h.Execute("writer_0")

			fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(200).Series).To(HaveLen(1))
			Expect(fr.Get(200).Series[0]).To(telem.MatchSeriesDataV[float32](20.0))
		})

		It("Should handle multiple aliases to same global channel", func() {
			// Test multiple independent aliases to the same channel:
			// out1 := output_ch
			// out2 := output_ch
			// out1 = value * 2.0  (first write)
			// out2 = value * 3.0  (second write, overwrites)
			resolver := symbol.MapResolver{
				"input_ch": {
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
				"output_ch": {
					Name: "output_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   200,
				},
				"sink_ch": {
					Name: "sink_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   300,
				},
			}

			source := `
func writer{} (value f32) u8 {
    out1 := output_ch
    out2 := output_ch
    out1 = value * 2.0
    out2 = value * 3.0
    return 0
}

input_ch -> writer{} -> sink_ch
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 200, DataType: telem.Float32T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close()

			// Set input value to 10.0
			// First write: 10 * 2 = 20
			// Second write: 10 * 3 = 30
			// Both writes go to the same channel - the runtime accumulates them
			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[float32](10.0), telem.NewSeriesSecondsTSV(1))
			h.Execute("writer_0")

			fr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			// Both writes are preserved in a single output series for the channel.
			Expect(fr.Get(200).Series).To(HaveLen(1))
			Expect(fr.Get(200).Series[0]).To(telem.MatchSeriesDataV[float32](20.0, 30.0))
		})

		It("Should handle reading from global channel alias", func() {
			// Test reading through a global channel alias:
			// sp := set_point_ch  (alias to global channel)
			// threshold := sp     (read from the channel through alias)
			resolver := symbol.MapResolver{
				"input_ch": {
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   100,
				},
				"set_point_ch": {
					Name: "set_point_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   200,
				},
				"output_ch": {
					Name: "output_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   300,
				},
			}

			source := `
func checker{} (value f32) u8 {
    sp := set_point_ch
    threshold := sp
    if (value > threshold) {
        return 1
    }
    return 0
}

input_ch -> checker{} -> output_ch
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 100, DataType: telem.Float32T},
				channel.Digest{Key: 200, DataType: telem.Float32T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close()

			// Set set_point_ch to 50.0
			fr := telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float32](50.0))
			h.ChannelState().Ingest(fr)

			// Test with value=60 (above threshold), should return 1
			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[float32](60.0), telem.NewSeriesSecondsTSV(1))
			h.Execute("checker_0")
			result := h.Output("checker_0", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(1)))

			// Test with value=40 (below threshold), should return 0
			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[float32](40.0), telem.NewSeriesSecondsTSV(2))
			h.Execute("checker_0")
			result = h.Output("checker_0", 0)
			Expect(telem.UnmarshalSeries[uint8](result)[0]).To(Equal(uint8(0)))
		})

		It("Should write to separate channels when function with channel config is used multiple times", func() {
			resolver := symbol.MapResolver{
				"input_1": {
					Name: "input_1",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   101,
				},
				"input_2": {
					Name: "input_2",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   102,
				},
				"counter_1": {
					Name: "counter_1",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   201,
				},
				"counter_2": {
					Name: "counter_2",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   202,
				},
				"sink_1": {
					Name: "sink_1",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   301,
				},
				"sink_2": {
					Name: "sink_2",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   302,
				},
			}

			source := `
func increment{
    counter chan f32
} (trigger u8) u8 {
    if trigger != 0 {
        counter = counter + 1.0
    }
    return 0
}

input_1 -> increment{counter=counter_1} -> sink_1
input_2 -> increment{counter=counter_2} -> sink_2
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 101, DataType: telem.Uint8T},
				channel.Digest{Key: 102, DataType: telem.Uint8T},
				channel.Digest{Key: 201, DataType: telem.Float32T},
				channel.Digest{Key: 202, DataType: telem.Float32T},
				channel.Digest{Key: 301, DataType: telem.Uint8T},
				channel.Digest{Key: 302, DataType: telem.Uint8T},
			)
			defer h.Close()

			fr := telem.Frame[uint32]{}
			fr = fr.Append(201, telem.NewSeriesV[float32](0.0))
			fr = fr.Append(202, telem.NewSeriesV[float32](0.0))
			h.ChannelState().Ingest(fr)

			h.SetInput("on_input_1_0", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(1))
			h.SetInput("on_input_2_0", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(1))
			h.Execute("increment_0")
			h.Execute("increment_1")

			outFr, changed := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(outFr.Get(201).Series).To(HaveLen(1), "counter_1 should have been written")
			Expect(telem.UnmarshalSeries[float32](outFr.Get(201).Series[0])[0]).To(Equal(float32(1.0)))
			Expect(outFr.Get(202).Series).To(HaveLen(1), "counter_2 should have been written")
			Expect(telem.UnmarshalSeries[float32](outFr.Get(202).Series[0])[0]).To(Equal(float32(1.0)))
		})

		It("Should not write to channel when stateful variable initialized from channel is modified", func() {
			resolver := symbol.MapResolver{
				"input_ch": {
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   100,
				},
				"counter_ch": {
					Name: "counter_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   200,
				},
				"sink_ch": {
					Name: "sink_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   300,
				},
			}

			source := `
func count_local (trigger u8) u8 {
    counter $= counter_ch
    prev $= trigger
    if trigger != 0 and prev == 0 {
        counter = counter + 1.0
    }
    prev = trigger
    return 0
}

input_ch -> count_local{} -> sink_ch
`
			h := newTextHarness(source, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 200, DataType: telem.Float32T},
				channel.Digest{Key: 300, DataType: telem.Uint8T},
			)
			defer h.Close()

			fr := telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float32](5.0))
			h.ChannelState().Ingest(fr)

			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[uint8](0), telem.NewSeriesSecondsTSV(1))
			h.Execute("count_local_0")

			outFr, _ := h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(outFr.Get(200).Series).To(BeEmpty())

			fr = telem.Frame[uint32]{}
			fr = fr.Append(200, telem.NewSeriesV[float32](5.0))
			h.ChannelState().Ingest(fr)
			h.SetInput("on_input_ch_0", 0, telem.NewSeriesV[uint8](1), telem.NewSeriesSecondsTSV(2))
			h.Execute("count_local_0")

			outFr, _ = h.ChannelState().Flush(telem.Frame[uint32]{})
			Expect(outFr.Get(200).Series).To(BeEmpty())
		})
	})

	Describe("String config params", func() {
		It("should create and execute a node with a string config param without error", func() {
			g := arc.Graph{
				Functions: []ir.Function{{
					Key:     "log_fn",
					Config:  types.Params{{Name: "msg", Type: types.String()}},
					Outputs: types.Params{},
					Body:    ir.Body{Raw: `{}`},
				}},
				Nodes: []graph.Node{{
					Key:    "log_fn",
					Type:   "log_fn",
					Config: map[string]any{"msg": "hello"},
				}},
			}
			h := newHarness(g, nil)
			defer h.Close()
			h.Execute("log_fn")
		})
	})

	Describe("For Loops", func() {
		Describe("Range loops", func() {
			DescribeTable("should compute correct sums",
				expectOutput[int64],
				Entry("1-arg range", "range1", types.I64(), `{
					sum i64 := 0
					for i := range(5) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(10)),
				Entry("2-arg range", "range2", types.I64(), `{
					sum i64 := 0
					for i := range(5, 10) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(35)),
				Entry("3-arg range with step", "range3", types.I64(), `{
					sum i64 := 0
					for i := range(0, 10, 2) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(20)),
				Entry("empty range", "range_empty", types.I64(), `{
					sum i64 := 99
					for i := range(0) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(99)),
				Entry("reversed bounds", "range_rev", types.I64(), `{
					sum i64 := 99
					for i := range(10, 5) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(99)),
				Entry("step of 3", "range_step3", types.I64(), `{
					sum i64 := 0
					for i := range(0, 10, 3) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(18)),
				Entry("explicit step of 1", "range_step1", types.I64(), `{
					sum i64 := 0
					for i := range(0, 5, 1) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(10)),
				Entry("step larger than range", "range_big_step", types.I64(), `{
					sum i64 := 0
					for i := range(0, 5, 10) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(0)),
				Entry("step equals range", "range_step_eq", types.I64(), `{
					sum i64 := 0
					for i := range(0, 10, 10) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(0)),
				Entry("negative step", "range_neg_step", types.I64(), `{
					sum i64 := 0
					for i := range(10, 0, -2) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(30)),
				Entry("negative step of -1", "range_neg1", types.I64(), `{
					sum i64 := 0
					for i := range(10, 0, -1) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(55)),
				Entry("negative step with large gap", "range_neg_big", types.I64(), `{
					sum i64 := 0
					for i := range(20, 0, -5) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(50)),
				Entry("negative step empty", "range_neg_empty", types.I64(), `{
					sum i64 := 99
					for i := range(0, 10, -1) {
						sum = sum + i
					}
					return sum
				}`, nil, int64(99)),
			)
		})

		Describe("Series iteration", func() {
			It("Should sum elements with single-ident form", func() {
				expectOutput("series_sum", types.I32(), `{
					data series i32 := [1, 2, 3, 4, 5]
					sum i32 := 0
					for x := data {
						sum = sum + x
					}
					return sum
				}`, nil, int32(15))
			})

			It("Should compute weighted sum with two-ident form", func() {
				expectOutput("series_weighted", types.I32(), `{
					data series i32 := [10, 20, 30]
					sum i32 := 0
					for i, x := data {
						sum = sum + x * (i + 1)
					}
					return sum
				}`, nil, int32(140))
			})
		})

		Describe("Break and continue", func() {
			It("Should break on threshold in series iteration", func() {
				expectOutput("break_thresh", types.I32(), `{
					data series i32 := [1, 2, 3, 100, 5]
					sum i32 := 0
					for x := data {
						if x > 50 {
							break
						}
						sum = sum + x
					}
					return sum
				}`, nil, int32(6))
			})

			It("Should continue to skip odd indices in range loop", func() {
				expectOutput("cont_skip", types.I32(), `{
					sum i32 := 0
					for i := range(i32(6)) {
						if i % 2 != 0 {
							continue
						}
						sum = sum + i
					}
					return sum
				}`, nil, int32(6))
			})

			It("Should continue to skip elements in series iteration", func() {
				expectOutput("cont_series", types.I32(), `{
					data series i32 := [10, -1, 20, -1, 30]
					sum i32 := 0
					for x := data {
						if x < 0 {
							continue
						}
						sum = sum + x
					}
					return sum
				}`, nil, int32(60))
			})

			It("Should continue only the inner loop when nested", func() {
				expectOutput("cont_nested", types.I64(), `{
					sum i64 := 0
					for i := range(3) {
						for j := range(4) {
							if j == 2 {
								continue
							}
							sum = sum + 1
						}
					}
					return sum
				}`, nil, int64(9))
			})

			It("Should break only the inner loop when nested", func() {
				expectOutput("break_inner_only", types.I64(), `{
					sum i64 := 0
					for i := range(3) {
						for j := range(10) {
							if j >= 2 {
								break
							}
							sum = sum + 1
						}
						sum = sum + 100
					}
					return sum
				}`, nil, int64(306))
			})
		})

		Describe("Conditional and infinite", func() {
			It("Should execute while-style countdown", func() {
				expectOutput("while_count", types.I32(), `{
					n i32 := 5
					sum i32 := 0
					for n > 0 {
						sum = sum + n
						n = n - 1
					}
					return sum
				}`, nil, int32(15))
			})

			It("Should execute infinite loop with break", func() {
				expectOutput("inf_break", types.I32(), `{
					val i32 := 1
					for {
						val = val * 2
						if val > 100 {
							break
						}
					}
					return val
				}`, nil, int32(128))
			})
		})

		Describe("Nested loops", func() {
			It("Should compute matrix iteration count", func() {
				expectOutput("nested_count", types.I64(), `{
					count i64 := 0
					for i := range(3) {
						for j := range(4) {
							count = count + 1
						}
					}
					return count
				}`, nil, int64(12))
			})

			It("Should handle inner break with outer running fully", func() {
				expectOutput("inner_break", types.I64(), `{
					count i64 := 0
					for i := range(3) {
						for j := range(4) {
							if j > 1 {
								break
							}
							count = count + 1
						}
					}
					return count
				}`, nil, int64(6))
			})

			It("Should handle 3-deep nested range loops", func() {
				expectOutput("nested_3deep", types.I64(), `{
					count i64 := 0
					for i := range(2) {
						for j := range(3) {
							for k := range(4) {
								count = count + 1
							}
						}
					}
					return count
				}`, nil, int64(24))
			})

			It("Should nest range loop inside series iteration", func() {
				expectOutput("series_range_nested", types.I32(), `{
					data series i32 := [10, 20, 30]
					sum i32 := 0
					for x := data {
						for j := range(i32(x / 10)) {
							sum = sum + 1
						}
					}
					return sum
				}`, nil, int32(6))
			})
		})

		Describe("Stateful across calls", func() {
			It("Should accumulate across reactive executions", func() {
				g := singleFunctionGraph("loop_state", types.I64(), `{
					total i64 $= 0
					for i := range(3) {
						total = total + 1
					}
					return total
				}`)
				h := newHarness(g, nil)
				defer h.Close()

				n := h.CreateNode("loop_state")
				nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}

				n.Next(nCtx)
				Expect(telem.UnmarshalSeries[int64](h.Output("loop_state", 0))[0]).To(Equal(int64(3)))

				n.Reset()
				n.Next(nCtx)
				Expect(telem.UnmarshalSeries[int64](h.Output("loop_state", 0))[0]).To(Equal(int64(6)))

				n.Reset()
				n.Next(nCtx)
				Expect(telem.UnmarshalSeries[int64](h.Output("loop_state", 0))[0]).To(Equal(int64(9)))
			})
		})
	})
})
