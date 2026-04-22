// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stratifier_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/x/diagnostics"
)

// edge builds a continuous dataflow edge between two node keys on default
// parameters.
func edge(src, tgt string) ir.Edge {
	return ir.Edge{
		Source: ir.Handle{Node: src, Param: ir.DefaultOutputParam},
		Target: ir.Handle{Node: tgt, Param: ir.DefaultInputParam},
		Kind:   ir.EdgeKindContinuous,
	}
}

// programOf constructs a minimal IR whose root scope is parallel+always-live
// with all given members in a single catch-all stratum — the shape the
// analyzer emits before stratification. edges becomes IR.Edges.
func programOf(members []ir.Member, edges []ir.Edge) ir.IR {
	nodes := make([]ir.Node, 0)
	collectNodes(members, &nodes)
	root := ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
	}
	if len(members) > 0 {
		root.Strata = []ir.Members{members}
	}
	return ir.IR{Nodes: nodes, Edges: edges, Root: root}
}

// collectNodes synthesizes a minimal ir.Node for every leaf-node member
// reachable through the given members. Stratification doesn't care about
// node Type or params, only keys.
func collectNodes(members []ir.Member, out *[]ir.Node) {
	for _, m := range members {
		if m.NodeKey != nil {
			*out = append(*out, ir.Node{Key: *m.NodeKey})
			continue
		}
		if m.Scope != nil {
			for _, stratum := range m.Scope.Strata {
				collectNodes(stratum, out)
			}
			collectNodes(m.Scope.Steps, out)
		}
	}
}

// run stratifies prog and returns the resulting root scope. Fails the spec
// on any diagnostic. The program is mutated in place; the returned value is
// prog.Root after stratification.
func run(ctx context.Context, prog ir.IR) ir.Scope {
	diag := &diagnostics.Diagnostics{}
	diag = stratifier.Stratify(ctx, &prog, diag)
	ExpectWithOffset(1, diag.Ok()).To(BeTrue(), diag.String())
	return prog.Root
}

// stratumOf returns the stratum index of a member identified by its key
// within the given scope's Strata, or -1 if not present.
func stratumOf(s ir.Scope, key string) int {
	for i, stratum := range s.Strata {
		for _, m := range stratum {
			if m.Key() == key {
				return i
			}
		}
	}
	return -1
}

// totalMembers counts the number of members across all strata of the scope.
func totalMembers(s ir.Scope) int {
	var count int
	for _, stratum := range s.Strata {
		count += len(stratum)
	}
	return count
}

var _ = Describe("Stratify", func() {
	Describe("Empty input", func() {
		It("Should return a zero scope when the root is empty", func(ctx SpecContext) {
			prog := ir.IR{}
			root := run(ctx, prog)
			Expect(root.IsZero()).To(BeTrue())
		})

		It("Should preserve a parallel scope with no members", func(ctx SpecContext) {
			root := run(ctx, programOf(nil, nil))
			Expect(root.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(root.Strata).To(BeEmpty())
		})
	})

	Describe("Flat parallel scope", func() {
		It("Should place a single source node in phase 0", func(ctx SpecContext) {
			root := run(ctx, programOf([]ir.Member{ir.NodeMember("source")}, nil))
			Expect(totalMembers(root)).To(Equal(1))
			Expect(stratumOf(root, "source")).To(Equal(0))
		})

		It("Should keep independent sources in phase 0 together", func(ctx SpecContext) {
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("a"), ir.NodeMember("b"), ir.NodeMember("c")},
				nil,
			))
			Expect(root.Strata).To(HaveLen(1))
			Expect(root.Strata[0]).To(HaveLen(3))
		})

		It("Should chain two dependent nodes across phases", func(ctx SpecContext) {
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("sensor"), ir.NodeMember("process")},
				[]ir.Edge{edge("sensor", "process")},
			))
			Expect(stratumOf(root, "sensor")).To(Equal(0))
			Expect(stratumOf(root, "process")).To(Equal(1))
		})

		It("Should respect the longest path through a three-link chain", func(ctx SpecContext) {
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("a"), ir.NodeMember("b"), ir.NodeMember("c"), ir.NodeMember("d")},
				[]ir.Edge{edge("a", "b"), edge("b", "c"), edge("c", "d")},
			))
			Expect(stratumOf(root, "a")).To(Equal(0))
			Expect(stratumOf(root, "b")).To(Equal(1))
			Expect(stratumOf(root, "c")).To(Equal(2))
			Expect(stratumOf(root, "d")).To(Equal(3))
		})

		It("Should fan out from a single source into sibling phase-1 members", func(ctx SpecContext) {
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("src"), ir.NodeMember("a"), ir.NodeMember("b")},
				[]ir.Edge{edge("src", "a"), edge("src", "b")},
			))
			Expect(stratumOf(root, "src")).To(Equal(0))
			Expect(stratumOf(root, "a")).To(Equal(1))
			Expect(stratumOf(root, "b")).To(Equal(1))
			Expect(root.Strata[1]).To(HaveLen(2))
		})

		It("Should use the longest path in a diamond graph", func(ctx SpecContext) {
			// a -> b -> d and a -> c -> d: both paths have length 2, so d
			// lands in phase 2 regardless of how the algorithm schedules the
			// middle hops.
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("a"), ir.NodeMember("b"), ir.NodeMember("c"), ir.NodeMember("d")},
				[]ir.Edge{edge("a", "b"), edge("a", "c"), edge("b", "d"), edge("c", "d")},
			))
			Expect(stratumOf(root, "a")).To(Equal(0))
			Expect(stratumOf(root, "b")).To(Equal(1))
			Expect(stratumOf(root, "c")).To(Equal(1))
			Expect(stratumOf(root, "d")).To(Equal(2))
		})

		It("Should push late joins to the deeper phase", func(ctx SpecContext) {
			// a -> c; b -> c; c depends on both, lands after the deeper of
			// the two sources (both at phase 0).
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("a"), ir.NodeMember("b"), ir.NodeMember("c")},
				[]ir.Edge{edge("a", "c"), edge("b", "c")},
			))
			Expect(stratumOf(root, "c")).To(Equal(1))
		})

		It("Should ignore dataflow edges whose endpoints fall outside the scope", func(ctx SpecContext) {
			// Edges reference nodes not in the scope; they must not force
			// any reordering.
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("a"), ir.NodeMember("b")},
				[]ir.Edge{edge("ghost", "a"), edge("b", "phantom")},
			))
			Expect(root.Strata).To(HaveLen(1))
			Expect(root.Strata[0]).To(HaveLen(2))
		})
	})

	Describe("Cycle detection", func() {
		It("Should report a cycle between two members", func(ctx SpecContext) {
			prog := programOf(
				[]ir.Member{ir.NodeMember("a"), ir.NodeMember("b")},
				[]ir.Edge{edge("a", "b"), edge("b", "a")},
			)
			diag := &diagnostics.Diagnostics{}
			diag = stratifier.Stratify(ctx, &prog, diag)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("cycle detected"))
		})

		It("Should report a cycle across three members", func(ctx SpecContext) {
			prog := programOf(
				[]ir.Member{ir.NodeMember("a"), ir.NodeMember("b"), ir.NodeMember("c")},
				[]ir.Edge{edge("a", "b"), edge("b", "c"), edge("c", "a")},
			)
			diag := &diagnostics.Diagnostics{}
			diag = stratifier.Stratify(ctx, &prog, diag)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring("cycle detected"))
		})
	})

	Describe("Nested scopes", func() {
		It("Should treat a nested scope as atomic when phasing its parent", func(ctx SpecContext) {
			// sensor -> gated_stage (containing inner nodes). The stage sits
			// atomically in phase 1 of the root scope.
			stage := ir.Scope{
				Key:      "gated_stage",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessGated,
				Strata: []ir.Members{{
					ir.NodeMember("inner_a"),
					ir.NodeMember("inner_b"),
				}},
			}
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("sensor"), ir.ScopeMember(stage)},
				[]ir.Edge{edge("sensor", "inner_a")},
			))
			Expect(stratumOf(root, "sensor")).To(Equal(0))
			Expect(stratumOf(root, "gated_stage")).To(Equal(1))
		})

		It("Should re-phase a nested parallel scope's members independently", func(ctx SpecContext) {
			// Inner stage has two dependent nodes; the parent can't see the
			// inner dependency, but the stratifier recurses.
			inner := ir.Scope{
				Key:      "stage",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessGated,
				Strata: []ir.Members{{
					ir.NodeMember("a"),
					ir.NodeMember("b"),
				}},
			}
			prog := programOf(
				[]ir.Member{ir.ScopeMember(inner)},
				[]ir.Edge{edge("a", "b")},
			)
			root := run(ctx, prog)
			stage := root.Strata[0][0].Scope
			Expect(stage).ToNot(BeNil())
			Expect(stratumOf(*stage, "a")).To(Equal(0))
			Expect(stratumOf(*stage, "b")).To(Equal(1))
		})

		It("Should leave sequential scopes' member order untouched", func(ctx SpecContext) {
			// Sequential scopes are ordered by source position, not by
			// dataflow. The stratifier must not reorder their Members.
			seq := ir.Scope{
				Key:      "seq",
				Mode:     ir.ScopeModeSequential,
				Liveness: ir.LivenessGated,
				Steps: ir.Members{
					ir.NodeMember("step_0"),
					ir.NodeMember("step_1"),
					ir.NodeMember("step_2"),
				},
			}
			prog := programOf(
				[]ir.Member{ir.ScopeMember(seq)},
				// Edges in reverse order; if stratifier naively tried to
				// re-phase sequential scopes it would violate source order.
				[]ir.Edge{edge("step_2", "step_0"), edge("step_1", "step_0")},
			)
			root := run(ctx, prog)
			seqOut := root.Strata[0][0].Scope
			Expect(seqOut).ToNot(BeNil())
			Expect(seqOut.Steps).To(HaveLen(3))
			Expect(seqOut.Steps[0].Key()).To(Equal("step_0"))
			Expect(seqOut.Steps[1].Key()).To(Equal("step_1"))
			Expect(seqOut.Steps[2].Key()).To(Equal("step_2"))
		})

		It("Should recurse into sequential scope's nested parallel children", func(ctx SpecContext) {
			// seq contains a parallel child whose members have an internal
			// dependency. The stratifier must descend through the sequence.
			parallelChild := ir.Scope{
				Key:      "stage_a",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessGated,
				Strata: []ir.Members{{
					ir.NodeMember("n1"),
					ir.NodeMember("n2"),
				}},
			}
			seq := ir.Scope{
				Key:      "seq",
				Mode:     ir.ScopeModeSequential,
				Liveness: ir.LivenessGated,
				Steps:    ir.Members{ir.ScopeMember(parallelChild)},
			}
			prog := programOf(
				[]ir.Member{ir.ScopeMember(seq)},
				[]ir.Edge{edge("n1", "n2")},
			)
			root := run(ctx, prog)
			stageScope := root.Strata[0][0].Scope.Steps[0].Scope
			Expect(stageScope).ToNot(BeNil())
			Expect(stratumOf(*stageScope, "n1")).To(Equal(0))
			Expect(stratumOf(*stageScope, "n2")).To(Equal(1))
		})

		It("Should project cross-boundary edges onto the outer scope's members", func(ctx SpecContext) {
			// A root-level sensor feeds a node buried inside a nested scope.
			// At the outer level, the edge is projected onto the nested
			// scope as a whole, pushing it into phase 1.
			inner := ir.Scope{
				Key:      "inner",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessGated,
				Strata: []ir.Members{{
					ir.NodeMember("deep"),
				}},
			}
			root := run(ctx, programOf(
				[]ir.Member{ir.NodeMember("sensor"), ir.ScopeMember(inner)},
				[]ir.Edge{edge("sensor", "deep")},
			))
			Expect(stratumOf(root, "sensor")).To(Equal(0))
			Expect(stratumOf(root, "inner")).To(Equal(1))
		})
	})

	Describe("Re-phasing existing layouts", func() {
		It("Should discard a pre-existing phase split and rebuild from edges", func(ctx SpecContext) {
			// The input presents sensor and process already in separate
			// phases (but swapped); the stratifier should rewrite from the
			// actual dataflow.
			prog := ir.IR{
				Nodes: []ir.Node{{Key: "sensor"}, {Key: "process"}},
				Edges: []ir.Edge{edge("sensor", "process")},
				Root: ir.Scope{
					Mode:     ir.ScopeModeParallel,
					Liveness: ir.LivenessAlways,
					Strata: []ir.Members{
						{ir.NodeMember("process")},
						{ir.NodeMember("sensor")},
					},
				},
			}
			root := run(ctx, prog)
			Expect(stratumOf(root, "sensor")).To(Equal(0))
			Expect(stratumOf(root, "process")).To(Equal(1))
		})
	})
})
