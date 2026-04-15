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
// always-live Scope; Phases sets its phase layering; Sequence appends a nested
// sequential Scope member.
//
// Example usage:
//
//	program := testutil.NewIRBuilder().
//	    Node("A").
//	    Node("B").
//	    Edge("A", "output", "B", "input").
//	    Phases([][]string{{"A"}, {"B"}}).
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

// Phases sets the Root scope's phase layering. Each inner slice is a phase of
// node keys that execute together with no dependency between them; phase N
// depends only on phases 0..N-1.
func (b *IRBuilder) Phases(phases [][]string) *IRBuilder {
	b.prog.Root.Phases = make([]ir.Phase, 0, len(phases))
	for _, phase := range phases {
		members := make([]ir.Member, 0, len(phase))
		for _, key := range phase {
			members = append(members, ir.Member{
				Key:     key,
				NodeRef: &ir.NodeRef{Key: key},
			})
		}
		b.prog.Root.Phases = append(b.prog.Root.Phases, ir.Phase{Members: members})
	}
	return b
}

// ScopeSpec describes a nested Scope for use with IRBuilder.Sequence.
// Each ScopeSpec becomes a sibling member of the sequential Scope that
// Sequence appends.
type ScopeSpec struct {
	// Key is the member key (matches the nested scope's key).
	Key string
	// Phases is the phase layering for a parallel nested scope. Mutually
	// exclusive with Members.
	Phases [][]string
	// Members is the ordered member keys for a sequential nested scope.
	// Mutually exclusive with Phases.
	Members []string
}

// Sequence appends a sequential, gated nested Scope to the Root as a Member of
// its final phase (creating the phase if the Root is empty). Each spec becomes
// one Member of the sequential scope.
func (b *IRBuilder) Sequence(key string, specs []ScopeSpec) *IRBuilder {
	members := make([]ir.Member, 0, len(specs))
	for _, spec := range specs {
		nested := ir.Scope{
			Key:      spec.Key,
			Mode:     ir.ScopeModeParallel,
			Liveness: ir.LivenessGated,
		}
		switch {
		case spec.Members != nil:
			nested.Mode = ir.ScopeModeSequential
			for _, mk := range spec.Members {
				nested.Members = append(nested.Members, ir.Member{
					Key:     mk,
					NodeRef: &ir.NodeRef{Key: mk},
				})
			}
		default:
			for _, phase := range spec.Phases {
				phaseMembers := make([]ir.Member, 0, len(phase))
				for _, k := range phase {
					phaseMembers = append(phaseMembers, ir.Member{
						Key:     k,
						NodeRef: &ir.NodeRef{Key: k},
					})
				}
				nested.Phases = append(nested.Phases, ir.Phase{Members: phaseMembers})
			}
		}
		members = append(members, ir.Member{Key: spec.Key, Scope: &nested})
	}

	sequential := ir.Scope{
		Key:      key,
		Mode:     ir.ScopeModeSequential,
		Liveness: ir.LivenessGated,
		Members:  members,
	}

	if len(b.prog.Root.Phases) == 0 {
		b.prog.Root.Phases = []ir.Phase{{}}
	}
	last := len(b.prog.Root.Phases) - 1
	b.prog.Root.Phases[last].Members = append(
		b.prog.Root.Phases[last].Members,
		ir.Member{Key: key, Scope: &sequential},
	)
	return b
}

// Build returns the constructed IR.
func (b *IRBuilder) Build() ir.IR { return b.prog }
