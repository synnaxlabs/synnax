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
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// benchNode is a minimal node.Node used in scheduler benchmarks. It avoids
// the bookkeeping overhead of MockNode (NextCalled tracking, elapsed slice
// append) so benchmarks measure scheduler cost rather than test harness
// cost. Outputs are identified by their declared position — the scheduler
// pre-seeds its propagation tables from Outputs() so Next can call
// MarkChanged(i) without any string-based resolution.
type benchNode struct {
	outputNames []string
	truthy      set.Set[string]
}

func (b *benchNode) Next(ctx node.Context) {
	for i := range b.outputNames {
		ctx.MarkChanged(i)
	}
}

func (*benchNode) Reset() {}

func (b *benchNode) Outputs() []string { return b.outputNames }

func (b *benchNode) IsOutputTruthy(p string) bool { return b.truthy.Contains(p) }

func newBenchNode(markAndTruthy ...string) *benchNode {
	t := make(set.Set[string], len(markAndTruthy))
	for _, p := range markAndTruthy {
		t.Add(p)
	}
	return &benchNode{outputNames: markAndTruthy, truthy: t}
}

// buildFlatParallel constructs an N-node single-phase parallel root. Each
// node is phase-0 so every node runs every cycle.
func buildFlatParallel(n int) (ir.IR, map[string]node.Node) {
	keys := make([]string, n)
	members := make([]ir.Member, n)
	nodes := make(map[string]node.Node, n)
	for i := range n {
		k := "n" + strconv.Itoa(i)
		keys[i] = k
		members[i] = noderef(k)
		nodes[k] = newBenchNode()
	}
	return programOf(keys, nil, rootScope(members...)), nodes
}

// buildFanoutChain constructs an N-node two-phase parallel root where a
// single source fans out to N-1 targets via continuous edges. Exercises
// MarkChanged propagation with N-1 edge lookups per tick.
func buildFanoutChain(n int) (ir.IR, map[string]node.Node) {
	if n < 2 {
		n = 2
	}
	keys := make([]string, n)
	edges := make([]ir.Edge, 0, n-1)
	p0 := []ir.Member{noderef("src")}
	p1 := make([]ir.Member, 0, n-1)
	nodes := make(map[string]node.Node, n)
	keys[0] = "src"
	nodes["src"] = newBenchNode("out")
	for i := 1; i < n; i++ {
		k := "t" + strconv.Itoa(i)
		keys[i] = k
		p1 = append(p1, noderef(k))
		edges = append(edges, continuousEdge("src", "out", k, "in"))
		nodes[k] = newBenchNode()
	}
	return programOf(keys, edges, rootWithPhases(phase(p0...), phase(p1...))), nodes
}

// buildDeepNested constructs a chain of D nested gated-parallel scopes with
// one node at the leaf. The outer scope is activated once; the walk has to
// descend D levels every cycle.
func buildDeepNested(depth int) (ir.IR, map[string]node.Node) {
	keys := []string{"leaf"}
	nodes := map[string]node.Node{"leaf": newBenchNode()}
	current := parallelScope("s0", phase(noderef("leaf")))
	for i := 1; i < depth; i++ {
		current = ir.Scope{
			Key:      "s" + strconv.Itoa(i),
			Mode:     ir.ScopeModeParallel,
			Liveness: ir.LivenessGated,
			Phases:   []ir.Phase{{Members: []ir.Member{scopeMember(current)}}},
		}
	}
	// Wrap the outermost gated scope with an always-live root and activate
	// via an external source so the whole chain is live each cycle.
	keys = append(keys, "trigger")
	nodes["trigger"] = newBenchNode("go")
	current.Activation = &ir.Handle{Node: "trigger", Param: "go"}
	root := rootWithPhases(phase(noderef("trigger"), scopeMember(current)))
	return programOf(keys, nil, root), nodes
}

// buildSequentialChain constructs a sequential scope of N members with a
// transition from member i to member i+1. On each tick every member fires
// its transition, cascading through the full chain within one Next call.
func buildSequentialChain(n int) (ir.IR, map[string]node.Node) {
	keys := make([]string, 0, n+1)
	members := make([]ir.Member, n)
	transitions := make([]ir.Transition, 0, n)
	nodes := make(map[string]node.Node, n+1)
	keys = append(keys, "trigger")
	nodes["trigger"] = newBenchNode("go")
	for i := range n {
		k := "m" + strconv.Itoa(i)
		keys = append(keys, k)
		members[i] = noderef(k)
		// Each member's output "next" is truthy, so a transition targeting
		// the successor fires on every cycle this member runs.
		nodes[k] = newBenchNode("next")
		if i+1 < n {
			next := "m" + strconv.Itoa(i+1)
			transitions = append(transitions, ir.Transition{
				On:     ir.Handle{Node: k, Param: "next"},
				Target: memberKeyTarget(next),
			})
		} else {
			transitions = append(transitions, ir.Transition{
				On:     ir.Handle{Node: k, Param: "next"},
				Target: exitTarget(),
			})
		}
	}
	seq := sequentialScope("seq", members, transitions...)
	seq.Activation = &ir.Handle{Node: "trigger", Param: "go"}
	return programOf(keys, nil, rootWithPhases(phase(noderef("trigger"), scopeMember(seq)))), nodes
}

func runTickBench(b *testing.B, prog ir.IR, nodes map[string]node.Node) {
	s := scheduler.New(prog, nodes, 0)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
			for i := 0; i < b.N; i++ {
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
