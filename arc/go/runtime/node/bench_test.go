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
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

func BenchmarkRefreshInputsSingleInput(b *testing.B) {
	ctx := context.Background()
	g := graph.Graph{
		Nodes: graph.Nodes{
			{Key: "source", Type: "source"},
			{Key: "target", Type: "target"},
		},
		Functions: []graph.Function{
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
		Edges: []ir.Edge{
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		telem.SetValueAt[float32](*sourceNode.Output(0), 0, float32(i))
		telem.SetValueAt[telem.TimeStamp](
			*sourceNode.OutputTime(0),
			0,
			telem.TimeStamp(i+1)*telem.SecondTS,
		)
		if !targetNode.RefreshInputs() {
			b.Fatal("Failed to refresh inputs")
		}
	}
}

func benchmarkChannelStateForWrites(indexed bool) *channel.ProgramState {
	digest := channel.Digest{Key: 1}
	if indexed {
		digest.Index = 2
	}
	return channel.NewProgramState([]channel.Digest{digest})
}

func BenchmarkWriteChannelU8Indexed(b *testing.B) {
	s := benchmarkChannelStateForWrites(true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.WriteChannelU8(1, uint8(i))
	}
}

func BenchmarkWriteChannelU8NoIndex(b *testing.B) {
	s := benchmarkChannelStateForWrites(false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.WriteChannelU8(1, uint8(i))
	}
}

func BenchmarkWriteChannelU8SameKeyFlush(b *testing.B) {
	const writesPerCycle = 128
	s := benchmarkChannelStateForWrites(true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < writesPerCycle; j++ {
			s.WriteChannelU8(1, uint8(j))
		}
		_, _ = s.Flush(telem.Frame[uint32]{})
	}
}

func BenchmarkFlushManyKeysSingleWrite(b *testing.B) {
	const keys = 256
	digests := make([]channel.Digest, keys)
	for i := 0; i < keys; i++ {
		digests[i] = channel.Digest{Key: uint32(i + 1)}
	}
	s := channel.NewProgramState(digests)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k := 0; k < keys; k++ {
			s.WriteChannelU8(uint32(k+1), uint8(k))
		}
		_, _ = s.Flush(telem.Frame[uint32]{})
	}
}
