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
	"testing"

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	stltime "github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/stl/wasm"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero"
)

// affineBench is a WASM node computing y = a * x + b over three constant source
// nodes, the transformation behind sensor calibration and unit conversion.
type affineBench struct {
	node    node.Node
	x, a, b *node.State
	ctx     node.Context
}

func affineGraph() arc.Graph {
	constant := func(key string) ir.Function {
		return ir.Function{
			Key: key,
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.F32()},
			},
			Body: ir.Body{Raw: `{ return 1.0 }`},
		}
	}
	edge := func(source string) graph.Edge {
		return graph.Edge{
			Source: ir.Handle{Node: source, Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "affine", Param: source},
		}
	}
	return arc.Graph{
		Functions: []ir.Function{
			{
				Key: "affine",
				Inputs: types.Params{
					{Name: "x", Type: types.F32()},
					{Name: "a", Type: types.F32()},
					{Name: "b", Type: types.F32()},
				},
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.F32()},
				},
				Body: ir.Body{Raw: `{ return a * x + b }`},
			},
			constant("x"),
			constant("a"),
			constant("b"),
		},
		Nodes: []graph.Node{{Key: "x"}, {Key: "a"}, {Key: "b"}, {Key: "affine"}},
		Inputs: map[string]msgpack.EncodedJSON{
			"x":      {"type": "x"},
			"a":      {"type": "a"},
			"b":      {"type": "b"},
			"affine": {"type": "affine"},
		},
		Edges: graph.Edges{edge("x"), edge("a"), edge("b")},
	}
}

func newAffineBench(b *testing.B) affineBench {
	b.Helper()
	ctx := context.Background()
	g := affineGraph()
	mod, err := arc.CompileGraph(ctx, g, NewRoot(nil))
	if err != nil {
		b.Fatalf("Failed to compile graph: %v", err)
	}
	a, diagnostics := graph.Analyze(ctx, g, nil)
	if diagnostics != nil && !diagnostics.Ok() {
		b.Fatalf("Failed to analyze graph: %s", diagnostics.String())
	}
	s := node.New(a)
	factory := newBenchModule(ctx, b, mod.WASM)
	n, err := factory.Create(node.Config{
		Node:    a.Nodes.Get("affine"),
		State:   s.Node("affine"),
		Program: mod,
	})
	if err != nil {
		b.Fatalf("Failed to create WASM node: %v", err)
	}
	return affineBench{
		node: n,
		x:    s.Node("x"),
		a:    s.Node("a"),
		b:    s.Node("b"),
		ctx: node.Context{
			Context:       ctx,
			ReserveStamps: testutil.ReserveStamps(0),
			MarkDone:      func() {},
			MarkChanged:   func(int) {},
			ReportError:   func(err error) {},
		},
	}
}

// newBenchModule instantiates the guest with every STL host module and closes it when
// the benchmark ends.
func newBenchModule(ctx context.Context, b *testing.B, guestWASM []byte) *wasm.Module {
	b.Helper()
	stringsState := stlstrings.NewProgramState()
	seriesState := series.NewProgramState()
	wasmRT := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	statefulMod, err := stateful.NewHost(ctx, wasmRT, seriesState, stringsState)
	if err != nil {
		b.Fatalf("Failed to create stateful module: %v", err)
	}
	if _, err := series.NewHost(ctx, wasmRT, seriesState); err != nil {
		b.Fatal(err)
	}
	stringsMod, err := stlstrings.NewHost(ctx, wasmRT, stringsState, nil)
	if err != nil {
		b.Fatalf("Failed to create strings module: %v", err)
	}
	if _, err := stlmath.NewHost(ctx, wasmRT); err != nil {
		b.Fatal(err)
	}
	errorsMod, err := stlerrors.NewHost(ctx, wasmRT, nil)
	if err != nil {
		b.Fatalf("Failed to create errors module: %v", err)
	}
	if _, err := stltime.NewHost(ctx, wasmRT, stringsState, discardReport); err != nil {
		b.Fatal(err)
	}
	guest, err := wasmRT.Instantiate(ctx, guestWASM)
	if err != nil {
		b.Fatalf("Failed to instantiate WASM module: %v", err)
	}
	stringsMod.SetMemory(guest.Memory())
	errorsMod.SetMemory(guest.Memory())
	b.Cleanup(func() {
		if err := guest.Close(ctx); err != nil {
			b.Errorf("Failed to close guest module: %v", err)
		}
		if err := wasmRT.Close(ctx); err != nil {
			b.Errorf("Failed to close wazero runtime: %v", err)
		}
	})
	return &wasm.Module{
		Module:        guest,
		Memory:        guest.Memory(),
		Strings:       stringsState,
		NodeKeySetter: statefulMod,
	}
}

func (f affineBench) setConstants() {
	*f.a.Output(0) = telem.NewSeriesV[float32](1)
	*f.a.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
	*f.b.Output(0) = telem.NewSeriesV[float32](1)
	*f.b.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
}

// BenchmarkWASMNodeSimpleArithmetic measures the per-sample cost of a WASM node
// computing an affine transformation: call overhead, value conversion, and
// allocations.
func BenchmarkWASMNodeSimpleArithmetic(b *testing.B) {
	f := newAffineBench(b)
	b.ReportAllocs()
	f.setConstants()
	b.ResetTimer()
	for i := range b.N {
		*f.x.OutputTime(0) = telem.NewSeriesSecondsTSV(telem.TimeStamp(i))
		f.node.Next(f.ctx)
	}
	b.StopTimer()
	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "samples/sec")
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/sample")
}

// BenchmarkWASMNodeZeroAlloc is a variant that pre-allocates all series
// to isolate whether nodeImpl.Next itself causes heap allocations
// (specifically ctx boxing into context.Context).
func BenchmarkWASMNodeZeroAlloc(b *testing.B) {
	f := newAffineBench(b)
	f.setConstants()
	*f.x.Output(0) = telem.NewSeriesV[float32](1)
	*f.x.OutputTime(0) = telem.NewSeriesSecondsTSV(1)

	// Warm up to ensure all internal buffers are allocated
	f.node.Next(f.ctx)

	b.ReportAllocs()
	for b.Loop() {
		f.node.Next(f.ctx)
	}
	b.StopTimer()
}
