// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package scheduler_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/x/telem"
)

// benchNode is a minimal node.Node used in scheduler benchmarks. It avoids
// the bookkeeping overhead of MockNode (NextCalled tracking, elapsed slice
// append) so benchmarks measure scheduler cost rather than test harness
// cost. Output names live in the IR; this node fires MarkChanged for every
// declared truthy ordinal each cycle.
type benchNode struct {
	truthy []bool
}

func (b *benchNode) Next(ctx node.Context) {
	for i, t := range b.truthy {
		if t {
			ctx.MarkChanged(i)
		}
	}
}

func (*benchNode) Reset() {}

func (b *benchNode) IsOutputTruthy(idx int) bool {
	if idx < 0 || idx >= len(b.truthy) {
		return false
	}
	return b.truthy[idx]
}

// newBenchNode constructs a benchNode with the given per-ordinal truthy
// values. Pass nothing for a silent node.
func newBenchNode(truthy ...bool) *benchNode { return &benchNode{truthy: truthy} }

// buildFlatParallel constructs an N-node single-phase parallel root. Each
// node is phase-0 so every node runs every cycle.
func buildFlatParallel(n int) (ir.IR, map[string]node.Node) {
	irNodes := make([]ir.Node, n)
	members := make([]ir.Member, n)
	nodes := make(map[string]node.Node, n)
	for i := range n {
		k := "n" + strconv.Itoa(i)
		irNodes[i] = irNode(k)
		members[i] = ir.NodeMember(k)
		nodes[k] = newBenchNode()
	}
	return programOf(irNodes, nil, rootScope(members...)), nodes
}

// buildFanoutChain constructs an N-node two-phase parallel root where a
// single source fans out to N-1 targets via continuous edges. Exercises
// MarkChanged propagation with N-1 edge lookups per tick.
func buildFanoutChain(n int) (ir.IR, map[string]node.Node) {
	if n < 2 {
		n = 2
	}
	irNodes := make([]ir.Node, 0, n)
	edges := make([]ir.Edge, 0, n-1)
	p0 := []ir.Member{ir.NodeMember("src")}
	p1 := make([]ir.Member, 0, n-1)
	nodes := make(map[string]node.Node, n)
	irNodes = append(irNodes, irNode("src", "out"))
	nodes["src"] = newBenchNode(true)
	for i := 1; i < n; i++ {
		k := "t" + strconv.Itoa(i)
		irNodes = append(irNodes, irNode(k))
		p1 = append(p1, ir.NodeMember(k))
		edges = append(edges, continuousEdge("src", "out", k, "in"))
		nodes[k] = newBenchNode()
	}
	return programOf(irNodes, edges, rootWithStrata(stratum(p0...), stratum(p1...))), nodes
}

// buildDeepNested constructs a chain of D nested parallel scopes with one
// node at the leaf. The outermost scope is gated and activated via an
// external handle; all inner wrappers are always-live so the cascade
// descends the full chain on each activation.
func buildDeepNested(depth int) (ir.IR, map[string]node.Node) {
	irNodes := []ir.Node{irNode("leaf")}
	nodes := map[string]node.Node{"leaf": newBenchNode()}
	current := ir.Scope{
		Key:      "s0",
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
		Strata:   []ir.Members{stratum(ir.NodeMember("leaf"))},
	}
	for i := 1; i < depth; i++ {
		current = ir.Scope{
			Key:      "s" + strconv.Itoa(i),
			Mode:     ir.ScopeModeParallel,
			Liveness: ir.LivenessAlways,
			Strata:   []ir.Members{{ir.ScopeMember(current)}},
		}
	}
	irNodes = append(irNodes, irNode("trigger", "go"))
	nodes["trigger"] = newBenchNode(true)
	current.Liveness = ir.LivenessGated
	current.Activation = &ir.Handle{Node: "trigger", Param: "go"}
	root := rootWithStrata(stratum(ir.NodeMember("trigger"), ir.ScopeMember(current)))
	return programOf(irNodes, nil, root), nodes
}

// buildSequentialChain constructs a sequential scope of N members with a
// transition from member i to member i+1. On each tick every member fires
// its transition, cascading through the full chain within one Next call.
func buildSequentialChain(n int) (ir.IR, map[string]node.Node) {
	irNodes := make([]ir.Node, 0, n+1)
	members := make([]ir.Member, n)
	transitions := make([]ir.Transition, 0, n)
	nodes := make(map[string]node.Node, n+1)
	irNodes = append(irNodes, irNode("trigger", "go"))
	nodes["trigger"] = newBenchNode(true)
	for i := range n {
		k := "m" + strconv.Itoa(i)
		// Each member's output "next" is truthy, so a transition targeting
		// the successor fires on every cycle this member runs.
		irNodes = append(irNodes, irNode(k, "next"))
		members[i] = ir.NodeMember(k)
		nodes[k] = newBenchNode(true)
		if i+1 < n {
			next := "m" + strconv.Itoa(i+1)
			transitions = append(transitions, ir.Transition{
				On:        ir.Handle{Node: k, Param: "next"},
				TargetKey: stepKeyTarget(next),
			})
		} else {
			transitions = append(transitions, ir.Transition{
				On:        ir.Handle{Node: k, Param: "next"},
				TargetKey: exitTarget(),
			})
		}
	}
	seq := sequentialScope("seq", members, transitions...)
	seq.Activation = &ir.Handle{Node: "trigger", Param: "go"}
	return programOf(irNodes, nil, rootWithStrata(stratum(ir.NodeMember("trigger"), ir.ScopeMember(seq)))), nodes
}

func runTickBench(b *testing.B, prog ir.IR, nodes map[string]node.Node) {
	s := scheduler.New(prog, nodes, 0)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
	}
}

func BenchmarkTickFlatParallel(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			prog, nodes := buildFlatParallel(n)
			runTickBench(b, prog, nodes)
		})
	}
}

func BenchmarkTickFanoutChain(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			prog, nodes := buildFanoutChain(n)
			runTickBench(b, prog, nodes)
		})
	}
}

func BenchmarkTickDeepNestedScopes(b *testing.B) {
	for _, d := range []int{4, 16, 64} {
		b.Run(fmt.Sprintf("d=%d", d), func(b *testing.B) {
			prog, nodes := buildDeepNested(d)
			runTickBench(b, prog, nodes)
		})
	}
}

func BenchmarkTickSequentialCascade(b *testing.B) {
	for _, n := range []int{4, 16, 64} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			prog, nodes := buildSequentialChain(n)
			runTickBench(b, prog, nodes)
		})
	}
}

func BenchmarkConstruction(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			prog, nodes := buildFanoutChain(n)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = scheduler.New(prog, nodes, 0)
			}
		})
	}
}

func BenchmarkMarkChangedTruthy(b *testing.B) {
	// A single source with a wide fanout of 64 continuous edges. Every tick
	// the source fires MarkChanged once on a truthy output, which walks its
	// outgoing edges and (because the source also drives no sequential
	// scope) writes a hot-path marked_this_cycle entry.
	const fanout = 64
	prog, nodes := buildFanoutChain(fanout + 1)
	runTickBench(b, prog, nodes)
}
