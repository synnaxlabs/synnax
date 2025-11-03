// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkWASMNodeSimpleArithmetic measures the performance of a WASM node executing
// simple arithmetic operations (affine transformation: y = a * x + b).
//
// This benchmark establishes baseline performance characteristics for:
// - WASM function call overhead
// - Value extraction/conversion (valueAt/setValueAt)
// - Memory allocation patterns
// - Scaling behavior with input data size
//
// The pattern tested is representative of common control system transformations like
// sensor calibration, unit conversion, and simple signal processing.
func BenchmarkWASMNodeSimpleArithmetic(b *testing.B) {
	ctx := context.Background()

	// Create graph: y = a * x + b
	// This represents an affine transformation commonly used for sensor calibration
	g := arc.Graph{
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
				Body: ir.Body{Raw: `{
							return a * x + b
						}`},
			},
			{
				Key: "x",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.F32()},
				},
				Body: ir.Body{Raw: `{ return 1.0 }`},
			},
			{
				Key: "a",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.F32()},
				},
				Body: ir.Body{Raw: `{ return 1.0 }`},
			},
			{
				Key: "b",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.F32()},
				},
				Body: ir.Body{Raw: `{ return 1.0 }`},
			},
		},
		Nodes: []graph.Node{
			{Key: "x", Type: "x"},
			{Key: "a", Type: "a"},
			{Key: "b", Type: "b"},
			{Key: "affine", Type: "affine"},
		},
		Edges: []graph.Edge{
			{
				Source: ir.Handle{Node: "x", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "affine", Param: "x"},
			},
			{
				Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "affine", Param: "a"},
			},
			{
				Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "affine", Param: "b"},
			},
		},
	}

	// Compile graph to WASM
	mod, err := arc.CompileGraph(ctx, g)
	if err != nil {
		b.Fatalf("Failed to compile graph: %v", err)
	}

	// Analyze graph for IR
	a, diagnostics := graph.Analyze(ctx, g, nil)
	if diagnostics != nil && !diagnostics.Ok() {
		b.Fatalf("Failed to analyze graph: %s", diagnostics.String())
	}

	// Create state manager
	s := state.New(state.Config{IR: a})

	// Seed input data - create realistic sensor data pattern
	xNode := s.Node("x")
	aNode := s.Node("a")
	bNode := s.Node("b")

	// Open WASM module
	wasmMod, err := wasm.OpenModule(ctx, wasm.ModuleConfig{
		Module: mod,
		State:  s,
	})
	if err != nil {
		b.Fatalf("Failed to open WASM module: %v", err)
	}
	defer func() {
		if err := wasmMod.Close(); err != nil {
			b.Errorf("Failed to close WASM module: %v", err)
		}
	}()

	// Create WASM node factory
	factory, err := wasm.NewFactory(wasmMod)
	if err != nil {
		b.Fatalf("Failed to create WASM factory: %v", err)
	}

	// Create the affine transformation node
	affineNode := s.Node("affine")
	n, err := factory.Create(ctx, node.Config{
		Node:   a.Nodes.Get("affine"),
		State:  affineNode,
		Module: mod,
	})
	if err != nil {
		b.Fatalf("Failed to create WASM node: %v", err)
	}

	nodeCtx := node.Context{
		Context:     ctx,
		MarkChanged: func(output string) {},
	}

	// Benchmark the Next() execution
	b.ReportAllocs()
	b.ResetTimer()

	*aNode.Output(0) = telem.NewSeriesV[float32](1)
	*aNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
	*bNode.Output(0) = telem.NewSeriesV[float32](1)
	*bNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)

	for i := 0; i < b.N; i++ {
		*xNode.Output(0) = telem.NewSeriesV[float32](1)
		*xNode.OutputTime(0) = telem.NewSeriesSecondsTSV(telem.TimeStamp(i))
		n.Next(nodeCtx)
	}

	b.StopTimer()

	// Report custom metrics
	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "samples/sec")
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/sample")
}
