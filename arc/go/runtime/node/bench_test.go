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
	"github.com/synnaxlabs/arc/stl/channels"
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

func benchmarkChannelStateForWrites(indexed bool) *channels.ProgramState {
	digest := channels.Digest{Key: 1}
	if indexed {
		digest.Index = 2
	}
	return channels.NewProgramState([]channels.Digest{digest})
}

func BenchmarkWriteChannelU8Indexed(b *testing.B) {
	s := benchmarkChannelStateForWrites(true)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		s.WriteChannelU8(1, uint8(i))
	}
}

func BenchmarkWriteChannelU8NoIndex(b *testing.B) {
	s := benchmarkChannelStateForWrites(false)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		s.WriteChannelU8(1, uint8(i))
	}
}

func BenchmarkWriteChannelU8SameKeyFlush(b *testing.B) {
	const writesPerCycle = 128
	s := benchmarkChannelStateForWrites(true)
	b.ReportAllocs()
	for b.Loop() {
		for j := range writesPerCycle {
			s.WriteChannelU8(1, uint8(j))
		}
		_, _, _ = s.Flush(telem.Frame[uint32]{}, 0)
	}
}

const periodNS = int64(100 * telem.Millisecond)

// literalProgram wires consumer c with a literal "period" and a var-bound "rate"
// (variable node v): the two ways a timer node receives its span.
func literalProgram() *node.ProgramState {
	v := ir.Node{Key: "v", Type: "variable", Outputs: types.Params{
		{Name: ir.DefaultOutputParam, Type: types.I64()},
	}}
	c := ir.Node{Key: "c", Type: "consumer", Inputs: types.Params{
		{Name: "period", Type: types.I64(), Value: periodNS},
		{Name: "rate", Type: types.VarRef(types.I64(), "v"), Value: periodNS},
	}}
	return node.New(ir.IR{Nodes: ir.Nodes{v, c}})
}

// BenchmarkNumericInputAt is the read a timer node performs after the fix, with
// the index resolved once at construction.
func BenchmarkNumericInputAt(b *testing.B) {
	c := literalProgram().Node("c")
	idx, err := c.ResolveInput("period")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	var total int64
	for b.Loop() {
		total += c.NumericInputAt[int64](idx)
	}
	if total == 0 {
		b.Fatal("read no value")
	}
}

// BenchmarkNumericInputLiteral is the read a timer node performed before the
// fix: resolve by name, then cast the boxed literal.
func BenchmarkNumericInputLiteral(b *testing.B) {
	c := literalProgram().Node("c")
	b.ReportAllocs()
	var total int64
	for b.Loop() {
		total += c.NumericInput[int64]("period")
	}
	if total == 0 {
		b.Fatal("read no value")
	}
}

// BenchmarkInputSeriesLiteral reads the same value from the typed series node
// construction already built for the literal, with the index resolved once.
func BenchmarkInputSeriesLiteral(b *testing.B) {
	c := literalProgram().Node("c")
	idx, err := c.ResolveInput("period")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	var total int64
	for b.Loop() {
		total += c.Input(idx).ValueAt[int64](-1)
	}
	if total == 0 {
		b.Fatal("read no value")
	}
}

// BenchmarkNumericInputVar reads a var-bound input from the variable's output
// series. The scheduler fills that slot before the first cycle, so this is the
// only state a real program reaches.
func BenchmarkNumericInputVar(b *testing.B) {
	s := literalProgram()
	*s.Node("v").Output(0) = telem.NewSeriesV[int64](periodNS)
	c := s.Node("c")
	b.ReportAllocs()
	var total int64
	for b.Loop() {
		total += c.NumericInput[int64]("rate")
	}
	if total == 0 {
		b.Fatal("read no value")
	}
}

// BenchmarkResolveInput isolates the by-name lookup that every read repeats.
func BenchmarkResolveInput(b *testing.B) {
	c := literalProgram().Node("c")
	b.ReportAllocs()
	var total int
	for b.Loop() {
		idx, err := c.ResolveInput("rate")
		if err != nil {
			b.Fatal(err)
		}
		total += idx
	}
	if total == 0 {
		b.Fatal("resolved no index")
	}
}

// BenchmarkCastNumeric isolates the boxed cast alone.
func BenchmarkCastNumeric(b *testing.B) {
	v := any(periodNS)
	b.ReportAllocs()
	var total int64
	for b.Loop() {
		total += telem.CastNumeric[int64](v)
	}
	if total == 0 {
		b.Fatal("read no value")
	}
}

func BenchmarkFlushManyKeysSingleWrite(b *testing.B) {
	const keys = 256
	digests := make([]channels.Digest, keys)
	for i := range keys {
		digests[i] = channels.Digest{Key: uint32(i + 1)}
	}
	s := channels.NewProgramState(digests)
	b.ReportAllocs()
	for b.Loop() {
		for k := range keys {
			s.WriteChannelU8(uint32(k+1), uint8(k))
		}
		_, _, _ = s.Flush(telem.Frame[uint32]{}, 0)
	}
}
