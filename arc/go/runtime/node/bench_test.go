// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node_test

import (
	"context"
	"testing"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
)

func BenchmarkRefreshInputsSingleInput(b *testing.B) {
	ctx := context.Background()
	g := graph.Graph{
		Nodes: graph.Nodes{
			{Key: "source"},
			{Key: "target"},
		},
		Inputs: map[string]msgpack.EncodedJSON{
			"source": {"type": "source"},
			"target": {"type": "target"},
		},
		Functions: []ir.Function{
			{
				Key: "source",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.F32()},
				},
			},
			{
				Key: "target",
				Inputs: types.Params{
					{Name: ir.DefaultInputParam, Type: types.F32()},
				},
			},
		},
		Edges: graph.Edges{
			{
				Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "target", Param: ir.DefaultInputParam},
			},
		},
	}
	inter, diagnostics := graph.Analyze(ctx, g, nil)
	if !diagnostics.Ok() {
		b.Fatalf("Failed to analyze graph: %s", diagnostics.String())
	}
	s := node.New(inter)
	sourceNode := s.Node("source")
	targetNode := s.Node("target")
	*sourceNode.Output(0) = telem.NewSeriesV[float32](0)
	*sourceNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		sourceNode.Output(0).SetValueAt(0, float32(i))
		sourceNode.OutputTime(0).SetValueAt(0, telem.TimeStamp(i+1)*telem.SecondTS)
		sourceNode.MarkFresh(0)
		if !targetNode.RefreshInputs() {
			b.Fatal("Failed to refresh inputs")
		}
	}
}

func BenchmarkRefreshInputsStaleInput(b *testing.B) {
	ctx := context.Background()
	g := graph.Graph{
		Nodes: graph.Nodes{{Key: "fresh"}, {Key: "stale"}, {Key: "target"}},
		Inputs: map[string]msgpack.EncodedJSON{
			"fresh":  {"type": "fresh"},
			"stale":  {"type": "stale"},
			"target": {"type": "target"},
		},
		Functions: []ir.Function{
			{
				Key: "fresh",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.F32()},
				},
			},
			{
				Key: "stale",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.F32()},
				},
			},
			{
				Key: "target",
				Inputs: types.Params{
					{Name: "fresh", Type: types.F32()},
					{Name: "stale", Type: types.F32()},
				},
			},
		},
		Edges: graph.Edges{
			{
				Source: ir.Handle{Node: "fresh", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "target", Param: "fresh"},
			},
			{
				Source: ir.Handle{Node: "stale", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "target", Param: "stale"},
			},
		},
	}
	inter, diagnostics := graph.Analyze(ctx, g, nil)
	if !diagnostics.Ok() {
		b.Fatalf("Failed to analyze graph: %s", diagnostics.String())
	}
	s := node.New(inter)
	freshNode, staleNode := s.Node("fresh"), s.Node("stale")
	targetNode := s.Node("target")
	*freshNode.Output(0) = telem.NewSeriesV[float32](0)
	*freshNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
	*staleNode.Output(0) = telem.MakeSeries(telem.Float32T, 1000)
	*staleNode.OutputTime(0) = telem.MakeSeries(telem.TimestampT, 1000)
	staleNode.MarkFresh(0)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		freshNode.Output(0).SetValueAt(0, float32(i))
		freshNode.OutputTime(0).SetValueAt(0, telem.TimeStamp(i+1)*telem.SecondTS)
		freshNode.MarkFresh(0)
		if !targetNode.RefreshInputs() {
			b.Fatal("Failed to refresh inputs")
		}
	}
}
