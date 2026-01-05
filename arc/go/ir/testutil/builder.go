// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides utilities for constructing IR in tests.
package testutil

import (
	"github.com/synnaxlabs/arc/ir"
)

// StageSpec defines a stage for use with IRBuilder.Sequence.
type StageSpec struct {
	Key    string
	Strata ir.Strata
}

// IRBuilder provides a fluent API for constructing IR in tests.
// Avoids verbose struct construction for simple test graphs.
//
// Example usage:
//
//	ir := testutil.NewIRBuilder().
//	    Node("A").
//	    Node("B").
//	    Edge("A", "output", "B", "input").
//	    Strata([][]string{{"A"}, {"B"}}).
//	    Build()
type IRBuilder struct {
	prog ir.IR
}

// NewIRBuilder creates a new IRBuilder.
func NewIRBuilder() *IRBuilder {
	return &IRBuilder{prog: ir.IR{}}
}

// Node adds a node with the given key.
// Creates a minimal node with just the key set.
func (b *IRBuilder) Node(key string) *IRBuilder {
	b.prog.Nodes = append(b.prog.Nodes, ir.Node{Key: key})
	return b
}

// Edge adds a continuous edge: source.param -> target.param.
// Continuous edges propagate changes every time the source output changes.
func (b *IRBuilder) Edge(srcNode, srcParam, tgtNode, tgtParam string) *IRBuilder {
	b.prog.Edges = append(b.prog.Edges, ir.Edge{
		Source: ir.Handle{Node: srcNode, Param: srcParam},
		Target: ir.Handle{Node: tgtNode, Param: tgtParam},
		Kind:   ir.Continuous,
	})
	return b
}

// OneShot adds a one-shot edge: source.param => target.param.
// One-shot edges only fire when the source output is truthy,
// and only once per stage activation (or once ever for global strata).
func (b *IRBuilder) OneShot(srcNode, srcParam, tgtNode, tgtParam string) *IRBuilder {
	b.prog.Edges = append(b.prog.Edges, ir.Edge{
		Source: ir.Handle{Node: srcNode, Param: srcParam},
		Target: ir.Handle{Node: tgtNode, Param: tgtParam},
		Kind:   ir.OneShot,
	})
	return b
}

// Strata sets the global strata (topological execution order for non-staged nodes).
// Each inner slice is a stratum; nodes in the same stratum are independent.
func (b *IRBuilder) Strata(s [][]string) *IRBuilder {
	b.prog.Strata = s
	return b
}

// Sequence adds a sequence with stages.
//
// Example:
//
//	.Sequence("my_seq", []testutil.StageSpec{
//	    {Key: "stage_a", Strata: [][]string{{"A"}, {"B"}}},
//	    {Key: "stage_b", Strata: [][]string{{"C"}}},
//	})
func (b *IRBuilder) Sequence(key string, stages []StageSpec) *IRBuilder {
	seq := ir.Sequence{Key: key}
	for _, spec := range stages {
		nodes := collectNodes(spec.Strata)
		seq.Stages = append(seq.Stages, ir.Stage{
			Key:    spec.Key,
			Nodes:  nodes,
			Strata: spec.Strata,
		})
	}
	b.prog.Sequences = append(b.prog.Sequences, seq)
	return b
}

// Build returns the constructed IR.
func (b *IRBuilder) Build() ir.IR {
	return b.prog
}

// collectNodes extracts all node keys from strata.
func collectNodes(strata ir.Strata) []string {
	var nodes []string
	for _, stratum := range strata {
		nodes = append(nodes, stratum...)
	}
	return nodes
}
