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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Stratification", func() {
	Describe("Basic Linear Flows", func() {
		It("Should assign stratum 0 to channel sources", func() {
			var (
				nodes  = []ir.Node{{Key: "source", Type: "on"}}
				edges  []ir.Edge
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(1))
			Expect(strata.Get("source")).To(Equal(0))
		})

		It("Should assign stratum 0 to constant nodes", func() {
			var (
				nodes = []ir.Node{
					{Key: "const1", Type: "constant"},
					{Key: "const2", Type: "constant"},
				}
				edges  []ir.Edge
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(2))
			Expect(strata.Get("const1")).To(Equal(0))
			Expect(strata.Get("const2")).To(Equal(0))
		})

		It("Should create simple linear stratification: channel -> process", func() {
			var (
				nodes = []ir.Node{
					{Key: "sensor", Type: "on"},
					{Key: "process", Type: "filter"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "sensor", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "process", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(2))
			Expect(strata.Get("sensor")).To(Equal(0))
			Expect(strata.Get("process")).To(Equal(1))
		})

		It("Should create three-level stratification: channel -> process1 -> process2", func() {
			var (
				nodes = []ir.Node{
					{Key: "sensor", Type: "on"},
					{Key: "filter", Type: "filter"},
					{Key: "logger", Type: "log"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "sensor", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "filter", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "filter", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "logger", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(3))
			Expect(strata.Get("sensor")).To(Equal(0))
			Expect(strata.Get("filter")).To(Equal(1))
			Expect(strata.Get("logger")).To(Equal(2))
		})

		It("Should handle long chains correctly", func() {
			var (
				nodes = []ir.Node{
					{Key: "source", Type: "on"},
					{Key: "step1", Type: "process"},
					{Key: "step2", Type: "process"},
					{Key: "step3", Type: "process"},
					{Key: "step4", Type: "process"},
					{Key: "sink", Type: "write"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "step1", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "step1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "step2", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "step2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "step3", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "step3", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "step4", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "step4", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(6))
			Expect(strata.Get("source")).To(Equal(0))
			Expect(strata.Get("step1")).To(Equal(1))
			Expect(strata.Get("step2")).To(Equal(2))
			Expect(strata.Get("step3")).To(Equal(3))
			Expect(strata.Get("step4")).To(Equal(4))
			Expect(strata.Get("sink")).To(Equal(5))
		})
	})

	Describe("Diamond Dependencies", func() {
		It("Should handle diamond pattern correctly", func() {
			// Pattern:
			//     source (0)
			//     /    \
			//   left(1) right(1)
			//     \    /
			//      sink(2)
			var (
				nodes = []ir.Node{
					{Key: "source", Type: "on"},
					{Key: "left", Type: "process"},
					{Key: "right", Type: "process"},
					{Key: "sink", Type: "combine"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "left", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "right", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "left", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "right", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "b"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(4))
			Expect(strata.Get("source")).To(Equal(0))
			Expect(strata.Get("left")).To(Equal(1))
			Expect(strata.Get("right")).To(Equal(1))
			Expect(strata.Get("sink")).To(Equal(2))
		})

		It("Should handle asymmetric diamond (different path lengths)", func() {
			// Pattern:
			//       source (0)
			//       /    \
			//   fast(1)  slow1(1) -> slow2(2)
			//       \    /
			//        sink(3)
			var (
				nodes = []ir.Node{
					{Key: "source", Type: "on"},
					{Key: "fast", Type: "process"},
					{Key: "slow1", Type: "process"},
					{Key: "slow2", Type: "process"},
					{Key: "sink", Type: "combine"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "fast", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "slow1", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "slow1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "slow2", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "fast", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "slow2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "b"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(5))
			Expect(strata.Get("source")).To(Equal(0))
			Expect(strata.Get("fast")).To(Equal(1))
			Expect(strata.Get("slow1")).To(Equal(1))
			Expect(strata.Get("slow2")).To(Equal(2))
			// Sink must be at stratum 3 because it depends on slow2 (stratum 2)
			Expect(strata.Get("sink")).To(Equal(3))
		})

		It("Should handle multiple diamonds in series", func() {
			// Pattern:
			//   source -> diamond1 -> diamond2 -> sink
			var (
				nodes = []ir.Node{
					{Key: "source", Type: "on"},
					{Key: "d1_left", Type: "process"},
					{Key: "d1_right", Type: "process"},
					{Key: "d1_merge", Type: "combine"},
					{Key: "d2_left", Type: "process"},
					{Key: "d2_right", Type: "process"},
					{Key: "sink", Type: "write"},
				}
				edges = []ir.Edge{
					// First diamond
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "d1_left", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "d1_right", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "d1_left", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "d1_merge", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "d1_right", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "d1_merge", Param: "b"},
					},
					// Second diamond
					{
						Source: ir.Handle{Node: "d1_merge", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "d2_left", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "d1_merge", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "d2_right", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "d2_left", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "d2_right", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "b"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(7))
			Expect(strata.Get("source")).To(Equal(0))
			Expect(strata.Get("d1_left")).To(Equal(1))
			Expect(strata.Get("d1_right")).To(Equal(1))
			Expect(strata.Get("d1_merge")).To(Equal(2))
			Expect(strata.Get("d2_left")).To(Equal(3))
			Expect(strata.Get("d2_right")).To(Equal(3))
			Expect(strata.Get("sink")).To(Equal(4))
		})
	})

	Describe("Independent Parallel Flows", func() {
		It("Should handle two independent linear flows", func() {
			var (
				nodes = []ir.Node{
					{Key: "sensor1", Type: "on"},
					{Key: "process1", Type: "filter"},
					{Key: "sensor2", Type: "on"},
					{Key: "process2", Type: "filter"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "sensor1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "process1", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "sensor2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "process2", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(4))
			// All sources at stratum 0
			Expect(strata.Get("sensor1")).To(Equal(0))
			Expect(strata.Get("sensor2")).To(Equal(0))
			// All first-level processors at stratum 1
			Expect(strata.Get("process1")).To(Equal(1))
			Expect(strata.Get("process2")).To(Equal(1))
		})

		It("Should handle multiple independent chains of different lengths", func() {
			var (
				nodes = []ir.Node{
					// Short chain
					{Key: "short_src", Type: "on"},
					{Key: "short_sink", Type: "write"},
					// Long chain
					{Key: "long_src", Type: "on"},
					{Key: "long_step1", Type: "process"},
					{Key: "long_step2", Type: "process"},
					{Key: "long_sink", Type: "write"},
				}
				edges = []ir.Edge{
					// Short chain
					{
						Source: ir.Handle{Node: "short_src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "short_sink", Param: "input"},
					},
					// Long chain
					{
						Source: ir.Handle{Node: "long_src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "long_step1", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "long_step1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "long_step2", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "long_step2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "long_sink", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(6))
			// Sources
			Expect(strata.Get("short_src")).To(Equal(0))
			Expect(strata.Get("long_src")).To(Equal(0))
			// Short chain (depth 1)
			Expect(strata.Get("short_sink")).To(Equal(1))
			// Long chain (depth 3)
			Expect(strata.Get("long_step1")).To(Equal(1))
			Expect(strata.Get("long_step2")).To(Equal(2))
			Expect(strata.Get("long_sink")).To(Equal(3))
		})

		It("Should handle fan-out from single source to multiple sinks", func() {
			// Pattern:
			//          source (0)
			//        /   |   \
			//    sink1  sink2  sink3
			//    (1)    (1)    (1)
			var (
				nodes = []ir.Node{
					{Key: "source", Type: "on"},
					{Key: "sink1", Type: "write"},
					{Key: "sink2", Type: "write"},
					{Key: "sink3", Type: "write"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink1", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink2", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink3", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(4))
			Expect(strata.Get("source")).To(Equal(0))
			Expect(strata.Get("sink1")).To(Equal(1))
			Expect(strata.Get("sink2")).To(Equal(1))
			Expect(strata.Get("sink3")).To(Equal(1))
		})

		It("Should handle fan-in from multiple sources to single sink", func() {
			// Pattern:
			//   source1 (0)    source2 (0)    source3 (0)
			//        \          |          /
			//              combiner (1)
			var (
				nodes = []ir.Node{
					{Key: "source1", Type: "on"},
					{Key: "source2", Type: "on"},
					{Key: "source3", Type: "on"},
					{Key: "combiner", Type: "combine"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "source1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "combiner", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "source2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "combiner", Param: "b"},
					},
					{
						Source: ir.Handle{Node: "source3", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "combiner", Param: "c"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(4))
			Expect(strata.Get("source1")).To(Equal(0))
			Expect(strata.Get("source2")).To(Equal(0))
			Expect(strata.Get("source3")).To(Equal(0))
			Expect(strata.Get("combiner")).To(Equal(1))
		})
	})

	Describe("Cycle Detection", func() {
		It("Should detect simple two-node cycle", func() {
			var (
				nodes = []ir.Node{
					{Key: "node1", Type: "process"},
					{Key: "node2", Type: "process"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "node1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "node2", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "node2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "node1", Param: "input"},
					},
				}
				strata, diag = stratifier.Stratify(
					ctx,
					nodes,
					edges,
					nil,
					&diagnostics.Diagnostics{},
				)
			)
			Expect(diag).To(MatchError(ContainSubstring("cycle detected")))
			Expect(strata).To(BeEmpty())
		})

		It("Should detect self-loop", func() {
			var (
				nodes = []ir.Node{{Key: "looper", Type: "process"}}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "looper", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "looper", Param: "input"},
					},
				}
				strata, diag = stratifier.Stratify(
					ctx,
					nodes,
					edges,
					nil,
					&diagnostics.Diagnostics{},
				)
			)
			Expect(diag).To(MatchError(ContainSubstring("cycle detected")))
			Expect(strata).To(BeEmpty())
		})

		It("Should detect three-node cycle", func() {
			var (
				nodes = []ir.Node{
					{Key: "a", Type: "process"},
					{Key: "b", Type: "process"},
					{Key: "c", Type: "process"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "b", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "c", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "c", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "a", Param: "input"},
					},
				}
				strata, diag = stratifier.Stratify(
					ctx,
					nodes,
					edges,
					nil,
					&diagnostics.Diagnostics{},
				)
			)
			Expect(diag).To(MatchError(Or(
				ContainSubstring("cycle detected"),
				ContainSubstring("a"),
				ContainSubstring("b"),
				ContainSubstring("c"),
			)))
			Expect(strata).To(BeEmpty())
		})

		It("Should detect cycle in complex graph with valid paths", func() {
			// Valid path: source -> valid_chain -> sink
			// Cycle: cycleA -> cycleB -> cycleC -> cycleA
			var (
				nodes = []ir.Node{
					{Key: "source", Type: "on"},
					{Key: "valid_chain", Type: "process"},
					{Key: "sink", Type: "write"},
					{Key: "cycleA", Type: "process"},
					{Key: "cycleB", Type: "process"},
					{Key: "cycleC", Type: "process"},
				}
				edges = []ir.Edge{
					// Valid path
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "valid_chain", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "valid_chain", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "input"},
					},
					// Cycle
					{
						Source: ir.Handle{Node: "cycleA", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "cycleB", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "cycleB", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "cycleC", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "cycleC", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "cycleA", Param: "input"},
					},
				}
				strata, diag = stratifier.Stratify(
					ctx,
					nodes,
					edges,
					nil,
					&diagnostics.Diagnostics{},
				)
			)
			Expect(diag).To(MatchError(ContainSubstring("cycle detected")))
			Expect(strata).To(BeEmpty())
		})

		It("Should not falsely detect cycles in diamond patterns", func() {
			var (
				// Diamond is NOT a cycle - both paths converge at sink
				nodes = []ir.Node{
					{Key: "source", Type: "on"},
					{Key: "left", Type: "process"},
					{Key: "right", Type: "process"},
					{Key: "sink", Type: "combine"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "left", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "right", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "left", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "right", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "sink", Param: "b"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(len(strata)).To(BeNumerically(">", 0))
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle empty graph", func() {
			var (
				nodes  []ir.Node
				edges  []ir.Edge
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(0))
		})

		It("Should handle single isolated node", func() {
			var (
				nodes  = []ir.Node{{Key: "isolated", Type: "constant"}}
				edges  []ir.Edge
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(1))
			Expect(strata.Get("isolated")).To(Equal(0))
		})

		It("Should handle multiple isolated nodes", func() {
			var (
				nodes = []ir.Node{
					{Key: "island1", Type: "constant"},
					{Key: "island2", Type: "on"},
					{Key: "island3", Type: "constant"},
				}
				edges  []ir.Edge
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.NodeCount()).To(Equal(3))
			Expect(strata.Get("island1")).To(Equal(0))
			Expect(strata.Get("island2")).To(Equal(0))
			Expect(strata.Get("island3")).To(Equal(0))
		})

		It("Should handle node with only outgoing edges", func() {
			var (
				nodes = []ir.Node{
					{Key: "broadcaster", Type: "on"},
					{Key: "listener1", Type: "write"},
					{Key: "listener2", Type: "write"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "broadcaster", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "listener1", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "broadcaster", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "listener2", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.Get("broadcaster")).To(Equal(0))
			Expect(strata.Get("listener1")).To(Equal(1))
			Expect(strata.Get("listener2")).To(Equal(1))
		})

		It("Should handle node with only incoming edges", func() {
			var (
				nodes = []ir.Node{
					{Key: "source1", Type: "on"},
					{Key: "source2", Type: "on"},
					{Key: "aggregator", Type: "combine"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "source1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "aggregator", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "source2", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "aggregator", Param: "b"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.Get("source1")).To(Equal(0))
			Expect(strata.Get("source2")).To(Equal(0))
			Expect(strata.Get("aggregator")).To(Equal(1))
		})

		It("Should handle non-source nodes without incoming edges as stratum 0", func() {
			// A non-source node (not 'on' or 'constant') with no incoming edges
			// should still be assigned stratum 0 (it's a root in the subgraph)
			var (
				nodes = []ir.Node{
					{Key: "orphan", Type: "process"}, // Not a source type but no dependencies
					{Key: "child", Type: "process"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "orphan", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "child", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.Get("orphan")).To(Equal(0))
			Expect(strata.Get("child")).To(Equal(1))
		})
	})

	Describe("Complex Real-World Scenarios", func() {
		It("Should stratify a typical sensor processing pipeline", func() {
			// Realistic pattern:
			// - Multiple sensors (channels)
			// - Preprocessing stages
			// - Feature extraction
			// - Decision logic
			// - Actuation
			var (
				nodes = []ir.Node{
					// Sensors (stratum 0)
					{Key: "temp_sensor", Type: "on"},
					{Key: "pressure_sensor", Type: "on"},
					{Key: "flow_sensor", Type: "on"},
					// Preprocessing (stratum 1)
					{Key: "temp_filter", Type: "filter"},
					{Key: "pressure_filter", Type: "filter"},
					{Key: "flow_calibrate", Type: "calibrate"},
					// Feature extraction (stratum 2)
					{Key: "temp_derivative", Type: "derivative"},
					{Key: "combined_state", Type: "combine"},
					// Decision (stratum 3)
					{Key: "control_logic", Type: "controller"},
					// Actuation (stratum 4)
					{Key: "valve_cmd", Type: "write"},
				}
				edges = []ir.Edge{
					// Sensor -> Preprocessing
					{
						Source: ir.Handle{Node: "temp_sensor", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "temp_filter", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "pressure_sensor", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "pressure_filter", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "flow_sensor", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "flow_calibrate", Param: "input"},
					},
					// Preprocessing -> Feature extraction
					{
						Source: ir.Handle{Node: "temp_filter", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "temp_derivative", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "temp_filter", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "combined_state", Param: "temp"},
					},
					{
						Source: ir.Handle{Node: "pressure_filter", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "combined_state", Param: "pressure"},
					},
					{
						Source: ir.Handle{Node: "flow_calibrate", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "combined_state", Param: "flow"},
					},
					// Feature extraction -> Decision
					{
						Source: ir.Handle{Node: "temp_derivative", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "control_logic", Param: "rate"},
					},
					{
						Source: ir.Handle{Node: "combined_state", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "control_logic", Param: "state"},
					},
					// Decision -> Actuation
					{
						Source: ir.Handle{Node: "control_logic", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "valve_cmd", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)

			// Verify sensors are at stratum 0
			Expect(strata.Get("temp_sensor")).To(Equal(0))
			Expect(strata.Get("pressure_sensor")).To(Equal(0))
			Expect(strata.Get("flow_sensor")).To(Equal(0))

			// Verify preprocessing at stratum 1
			Expect(strata.Get("temp_filter")).To(Equal(1))
			Expect(strata.Get("pressure_filter")).To(Equal(1))
			Expect(strata.Get("flow_calibrate")).To(Equal(1))

			// Verify feature extraction at stratum 2
			Expect(strata.Get("temp_derivative")).To(Equal(2))
			Expect(strata.Get("combined_state")).To(Equal(2))

			// Verify decision at stratum 3
			Expect(strata.Get("control_logic")).To(Equal(3))

			// Verify actuation at stratum 4
			Expect(strata.Get("valve_cmd")).To(Equal(4))
		})

		It("Should stratify a multi-func alarm system with priorities", func() {
			// Pattern: Multiple alarm conditions with different priorities merging
			var (
				nodes = []ir.Node{
					// Sensors
					{Key: "temp", Type: "on"},
					{Key: "vibration", Type: "on"},
					{Key: "pressure", Type: "on"},
					// Threshold checks (stratum 1)
					{Key: "temp_high", Type: "compare"},
					{Key: "temp_critical", Type: "compare"},
					{Key: "vib_high", Type: "compare"},
					{Key: "pressure_low", Type: "compare"},
					// Priority logic (stratum 2)
					{Key: "critical_alarm", Type: "or"},
					{Key: "warning_alarm", Type: "or"},
					// Alarm manager (stratum 3)
					{Key: "alarm_manager", Type: "select"},
					// Output (stratum 4)
					{Key: "alarm_output", Type: "write"},
				}
				edges = []ir.Edge{
					// Sensors to checks
					{
						Source: ir.Handle{Node: "temp", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "temp_high", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "temp", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "temp_critical", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "vibration", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "vib_high", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "pressure", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "pressure_low", Param: "input"},
					},
					// Checks to priority logic
					{
						Source: ir.Handle{Node: "temp_critical", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "critical_alarm", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "pressure_low", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "critical_alarm", Param: "b"},
					},
					{
						Source: ir.Handle{Node: "temp_high", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "warning_alarm", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "vib_high", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "warning_alarm", Param: "b"},
					},
					// Priority to manager
					{
						Source: ir.Handle{Node: "critical_alarm", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "alarm_manager", Param: "critical"},
					},
					{
						Source: ir.Handle{Node: "warning_alarm", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "alarm_manager", Param: "warning"},
					},
					// Manager to output
					{
						Source: ir.Handle{Node: "alarm_manager", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "alarm_output", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.Get("temp")).To(Equal(0))
			Expect(strata.Get("temp_high")).To(Equal(1))
			Expect(strata.Get("critical_alarm")).To(Equal(2))
			Expect(strata.Get("alarm_manager")).To(Equal(3))
			Expect(strata.Get("alarm_output")).To(Equal(4))
		})
	})

	Describe("Per-Stage Stratification (Two-Tier Model)", func() {
		It("Should not detect cycles in cyclic state machines", func() {
			// This is a valid state machine where stages transition to each other
			// stage first -> second, stage second -> first
			// This should NOT produce a cycle error because stages are stratified independently
			var (
				nodes = []ir.Node{
					{Key: "start_cmd", Type: "on"},
					{Key: "entry_main_first", Type: "entry"},
					{Key: "entry_main_second", Type: "entry"},
					{Key: "first_node", Type: "process"},
					{Key: "second_node", Type: "process"},
				}
				edges = []ir.Edge{
					// Global: start_cmd triggers main sequence
					{
						Source: ir.Handle{Node: "start_cmd", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "entry_main_first", Param: "input"},
					},
					// Stage first: first_node triggers second stage
					{
						Source: ir.Handle{Node: "first_node", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "entry_main_second", Param: "input"},
					},
					// Stage second: second_node triggers first stage (cyclic transition)
					{
						Source: ir.Handle{Node: "second_node", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "entry_main_first", Param: "input"},
					},
				}
				sequences = []ir.Sequence{
					{
						Key: "main",
						Stages: []ir.Stage{
							{Key: "first", Nodes: []string{"first_node"}},
							{Key: "second", Nodes: []string{"second_node"}},
						},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, sequences, diag))
			)
			// Should not produce a cycle error
			Expect(diag.Ok()).To(BeTrue())
			// Global strata should contain start_cmd and entry nodes
			Expect(strata.NodeCount()).To(BeNumerically(">", 0))
		})

		It("Should populate per-stage strata independently", func() {
			var (
				nodes = []ir.Node{
					{Key: "global_source", Type: "on"},
					{Key: "entry_main_first", Type: "entry"},
					{Key: "stage_source", Type: "constant"}, // Stage-local source
					{Key: "stage_process", Type: "process"},
				}
				edges = []ir.Edge{
					// Global: global_source triggers first stage
					{
						Source: ir.Handle{Node: "global_source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "entry_main_first", Param: "input"},
					},
					// Stage first: stage_source -> stage_process
					{
						Source: ir.Handle{Node: "stage_source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "stage_process", Param: "input"},
					},
				}
				sequences = []ir.Sequence{
					{
						Key: "main",
						Stages: []ir.Stage{
							{Key: "first", Nodes: []string{"stage_source", "stage_process"}},
						},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, sequences, diag))
			)
			// Global strata should contain global_source and entry node
			Expect(strata.Get("global_source")).To(Equal(0))
			Expect(strata.Get("entry_main_first")).To(Equal(1))

			// Per-stage strata should have stage_source at stratum 0
			Expect(sequences[0].Stages[0].Strata.Get("stage_source")).To(Equal(0))
			Expect(sequences[0].Stages[0].Strata.Get("stage_process")).To(Equal(1))
		})

		It("Should stratify multiple stages independently", func() {
			var (
				nodes = []ir.Node{
					{Key: "global_src", Type: "on"},
					{Key: "entry_seq_s1", Type: "entry"},
					{Key: "entry_seq_s2", Type: "entry"},
					// Stage s1 nodes
					{Key: "s1_const", Type: "constant"},
					{Key: "s1_proc", Type: "process"},
					// Stage s2 nodes
					{Key: "s2_const", Type: "constant"},
					{Key: "s2_proc1", Type: "process"},
					{Key: "s2_proc2", Type: "process"},
				}
				edges = []ir.Edge{
					// Global edges
					{
						Source: ir.Handle{Node: "global_src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "entry_seq_s1", Param: "input"},
					},
					// Stage s1 edges
					{
						Source: ir.Handle{Node: "s1_const", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "s1_proc", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "s1_proc", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "entry_seq_s2", Param: "input"},
					},
					// Stage s2 edges
					{
						Source: ir.Handle{Node: "s2_const", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "s2_proc1", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "s2_proc1", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "s2_proc2", Param: "input"},
					},
				}
				sequences = []ir.Sequence{
					{
						Key: "seq",
						Stages: []ir.Stage{
							{Key: "s1", Nodes: []string{"s1_const", "s1_proc"}},
							{Key: "s2", Nodes: []string{"s2_const", "s2_proc1", "s2_proc2"}},
						},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, sequences, diag))
			)
			// Global strata should contain entry nodes and global source
			// stratum 0: global_src (source)
			// stratum 1: entry_seq_s1 (depends on global_src)
			// entry_seq_s2 depends on s1_proc which is in a stage, so it may be at stratum 0
			// since it has no incoming edges from global nodes
			Expect(strata.Get("global_src")).To(Equal(0))
			Expect(strata.Get("entry_seq_s1")).To(Equal(1))
			// Stage s1: const at 0, proc at 1
			Expect(sequences[0].Stages[0].Strata.Get("s1_const")).To(Equal(0))
			Expect(sequences[0].Stages[0].Strata.Get("s1_proc")).To(Equal(1))

			// Stage s2: const at 0, proc1 at 1, proc2 at 2
			Expect(sequences[0].Stages[1].Strata.Get("s2_const")).To(Equal(0))
			Expect(sequences[0].Stages[1].Strata.Get("s2_proc1")).To(Equal(1))
			Expect(sequences[0].Stages[1].Strata.Get("s2_proc2")).To(Equal(2))
		})
	})

	Describe("Multiple Output Parameters", func() {
		It("Should handle nodes with multiple named outputs", func() {
			var (
				nodes = []ir.Node{
					{Key: "source", Type: "on"},
					{Key: "splitter", Type: "split"},
					{Key: "high_sink", Type: "write"},
					{Key: "low_sink", Type: "write"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "splitter", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "splitter", Param: "high"},
						Target: ir.Handle{Node: "high_sink", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "splitter", Param: "low"},
						Target: ir.Handle{Node: "low_sink", Param: "input"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.Get("source")).To(Equal(0))
			Expect(strata.Get("splitter")).To(Equal(1))
			Expect(strata.Get("high_sink")).To(Equal(2))
			Expect(strata.Get("low_sink")).To(Equal(2))
		})

		It("Should handle conditional routing through different outputs", func() {
			var (
				nodes = []ir.Node{
					{Key: "input", Type: "on"},
					{Key: "router", Type: "select"},
					{Key: "true_path", Type: "process"},
					{Key: "false_path", Type: "process"},
					{Key: "merge", Type: "combine"},
				}
				edges = []ir.Edge{
					{
						Source: ir.Handle{Node: "input", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "router", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "router", Param: "true"},
						Target: ir.Handle{Node: "true_path", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "router", Param: "false"},
						Target: ir.Handle{Node: "false_path", Param: "input"},
					},
					{
						Source: ir.Handle{Node: "true_path", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "merge", Param: "a"},
					},
					{
						Source: ir.Handle{Node: "false_path", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "merge", Param: "b"},
					},
				}
				diag   = &diagnostics.Diagnostics{}
				strata = MustSucceed(stratifier.Stratify(ctx, nodes, edges, nil, diag))
			)
			Expect(strata.Get("input")).To(Equal(0))
			Expect(strata.Get("router")).To(Equal(1))
			Expect(strata.Get("true_path")).To(Equal(2))
			Expect(strata.Get("false_path")).To(Equal(2))
			Expect(strata.Get("merge")).To(Equal(3))
		})
	})
})
