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

import "github.com/synnaxlabs/arc/ir"

// IRBuilder provides a fluent API for constructing IR in tests. Avoids verbose
// struct construction for simple test graphs.
//
// The builder models the new unified execution shell: Root is a parallel,
// always-live Scope; Strata sets its stratum layering; Sequence appends a nested
// sequential Scope member.
//
// Example usage:
//
//	program := testutil.NewIRBuilder().
//	    Node("A").
//	    Node("B").
//	    Edge("A", "output", "B", "input").
//	    Strata([][]string{{"A"}, {"B"}}).
//	    Build()
type IRBuilder struct {
	prog ir.IR
}

// NewIRBuilder creates a new IRBuilder with a parallel, always-live Root scope.
func NewIRBuilder() *IRBuilder {
	return &IRBuilder{prog: ir.IR{Root: ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
	}}}
}

// Node adds a node with the given key. The node is created with just the key
// set; tests that need richer node configuration should assemble it directly.
func (b *IRBuilder) Node(key string) *IRBuilder {
	b.prog.Nodes = append(b.prog.Nodes, ir.Node{Key: key})
	return b
}

// Edge adds a continuous edge: source.param -> target.param. Continuous edges
// propagate changes every time the source output changes.
func (b *IRBuilder) Edge(srcNode, srcParam, tgtNode, tgtParam string) *IRBuilder {
	b.prog.Edges = append(b.prog.Edges, ir.Edge{
		Source: ir.Handle{Node: srcNode, Param: srcParam},
		Target: ir.Handle{Node: tgtNode, Param: tgtParam},
		Kind:   ir.EdgeKindContinuous,
	})
	return b
}

// Conditional adds a conditional edge: source.param => target.param.
// Conditional edges only propagate when the source output is truthy.
func (b *IRBuilder) Conditional(srcNode, srcParam, tgtNode, tgtParam string) *IRBuilder {
	b.prog.Edges = append(b.prog.Edges, ir.Edge{
		Source: ir.Handle{Node: srcNode, Param: srcParam},
		Target: ir.Handle{Node: tgtNode, Param: tgtParam},
		Kind:   ir.EdgeKindConditional,
	})
	return b
}

// Strata sets the Root scope's stratum layering. Each inner slice is a stratum
// of node keys that execute together with no dependency between them; stratum N
// depends only on strata 0..N-1.
func (b *IRBuilder) Strata(strata [][]string) *IRBuilder {
	b.prog.Root.Strata = make([]ir.Members, 0, len(strata))
	for _, stratum := range strata {
		members := make(ir.Members, 0, len(stratum))
		for _, key := range stratum {
			members = append(members, ir.Member{NodeKey: new(key)})
		}
		b.prog.Root.Strata = append(b.prog.Root.Strata, members)
	}
	return b
}

// ScopeSpec describes a nested Scope for use with IRBuilder.Sequence.
// Each ScopeSpec becomes a sibling member of the sequential Scope that
// Sequence appends.
type ScopeSpec struct {
	// Key is the member key (matches the nested scope's key).
	Key string
	// Strata is the stratum layering for a parallel nested scope. Mutually
	// exclusive with Steps.
	Strata [][]string
	// Steps is the ordered step keys for a sequential nested scope.
	// Mutually exclusive with Strata.
	Steps []string
}

// Sequence appends a sequential, gated nested Scope to the Root as a Member of
// its final stratum (creating the stratum if the Root is empty). Each spec
// becomes one step of the sequential scope.
func (b *IRBuilder) Sequence(key string, specs []ScopeSpec) *IRBuilder {
	members := make(ir.Members, 0, len(specs))
	for _, spec := range specs {
		nested := ir.Scope{
			Key:      spec.Key,
			Mode:     ir.ScopeModeParallel,
			Liveness: ir.LivenessGated,
		}
		switch {
		case spec.Steps != nil:
			nested.Mode = ir.ScopeModeSequential
			for _, sk := range spec.Steps {
				nested.Steps = append(nested.Steps, ir.Member{NodeKey: new(sk)})
			}
		default:
			for _, stratum := range spec.Strata {
				stratumMembers := make(ir.Members, 0, len(stratum))
				for _, key := range stratum {
					stratumMembers = append(stratumMembers, ir.Member{NodeKey: new(key)})
				}
				nested.Strata = append(nested.Strata, stratumMembers)
			}
		}
		members = append(members, ir.Member{Scope: &nested})
	}

	sequential := ir.Scope{
		Key:      key,
		Mode:     ir.ScopeModeSequential,
		Liveness: ir.LivenessGated,
		Steps:    members,
	}

	if len(b.prog.Root.Strata) == 0 {
		b.prog.Root.Strata = []ir.Members{{}}
	}
	last := len(b.prog.Root.Strata) - 1
	b.prog.Root.Strata[last] = append(
		b.prog.Root.Strata[last],
		ir.Member{Scope: &sequential},
	)
	return b
}

// Build returns the constructed IR.
func (b *IRBuilder) Build() ir.IR { return b.prog }
