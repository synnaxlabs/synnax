// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
)

// gnode constructs a graph node at the given coordinates.
func gnode(key string, x, y float64) graph.Node {
	return graph.Node{Key: key, Position: spatial.XY{X: x, Y: y}}
}

// gedge constructs a continuous edge between two node parameters.
func gedge(srcNode, srcParam, tgtNode, tgtParam string) ir.Edge {
	return ir.Edge{
		Source: ir.Handle{Node: srcNode, Param: srcParam},
		Target: ir.Handle{Node: tgtNode, Param: tgtParam},
		Kind:   ir.EdgeKindContinuous,
	}
}

// withGraph builds an Arc whose graph holds the given nodes and edges.
func withGraph(nodes graph.Nodes, edges ir.Edges) arc.Arc {
	return arc.Arc{Graph: graph.Graph{Nodes: nodes, Edges: edges}}
}

var _ = Describe("Reducer", func() {
	Describe("Rename", func() {
		It("Should replace the module name", func() {
			out := MustSucceed(arc.Reduce(
				arc.Arc{Name: "old"},
				arc.NewRenameAction(arc.RenamePayload{Name: "new"}),
			))
			Expect(out.Name).To(Equal("new"))
		})
	})

	Describe("SetMode", func() {
		It("Should switch the representation mode", func() {
			out := MustSucceed(arc.Reduce(
				arc.Arc{Mode: arc.ModeText},
				arc.NewSetModeAction(arc.SetModePayload{Mode: arc.ModeGraph}),
			))
			Expect(out.Mode).To(Equal(arc.ModeGraph))
		})
	})

	Describe("SetNode", func() {
		It("Should append the node when no node has the same key", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, nil)
			out := MustSucceed(arc.Reduce(state, arc.NewSetNodeAction(arc.SetNodePayload{
				Node: gnode("n2", 1, 2),
			})))
			Expect(out.Graph.Nodes).To(Equal(graph.Nodes{gnode("n1", 0, 0), gnode("n2", 1, 2)}))
		})
		It("Should replace an existing node in place, preserving slice index", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0), gnode("n2", 1, 1), gnode("n3", 2, 2)}, nil)
			out := MustSucceed(arc.Reduce(state, arc.NewSetNodeAction(arc.SetNodePayload{
				Node: gnode("n2", 9, 9),
			})))
			Expect(out.Graph.Nodes).To(HaveLen(3))
			Expect(out.Graph.Nodes[1]).To(Equal(gnode("n2", 9, 9)))
		})
	})

	Describe("SetNodePosition", func() {
		It("Should move the matching node", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0), gnode("n2", 5, 5)}, nil)
			out := MustSucceed(arc.Reduce(state, arc.NewSetNodePositionAction(arc.SetNodePositionPayload{
				Key:      "n1",
				Position: spatial.XY{X: 100, Y: 200},
			})))
			Expect(out.Graph.Nodes[0].Position).To(Equal(spatial.XY{X: 100, Y: 200}))
			Expect(out.Graph.Nodes[1].Position).To(Equal(spatial.XY{X: 5, Y: 5}))
		})
		It("Should be a no-op when the key does not match any node", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, nil)
			out := MustSucceed(arc.Reduce(state, arc.NewSetNodePositionAction(arc.SetNodePositionPayload{
				Key:      "ghost",
				Position: spatial.XY{X: 100, Y: 200},
			})))
			Expect(out.Graph.Nodes).To(Equal(graph.Nodes{gnode("n1", 0, 0)}))
		})
	})

	Describe("SetNodeConfig", func() {
		It("Should replace the configuration keyed by node key", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, nil)
			state.Graph.Configs = map[string]msgpack.EncodedJSON{"n1": {"gain": 1}}
			out := MustSucceed(arc.Reduce(state, arc.NewSetNodeConfigAction(arc.SetNodeConfigPayload{
				Key:    "n1",
				Config: msgpack.EncodedJSON{"offset": 2},
			})))
			Expect(out.Graph.Configs["n1"]).To(Equal(msgpack.EncodedJSON{"offset": 2}))
		})
		It("Should write the configuration even when no node has the key", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, nil)
			out := MustSucceed(arc.Reduce(state, arc.NewSetNodeConfigAction(arc.SetNodeConfigPayload{
				Key:    "ghost",
				Config: msgpack.EncodedJSON{"offset": 2},
			})))
			Expect(out.Graph.Configs["ghost"]).To(Equal(msgpack.EncodedJSON{"offset": 2}))
		})
	})

	Describe("RemoveNode", func() {
		It("Should remove the matching node, its config, and every edge connected to it", func() {
			state := withGraph(
				graph.Nodes{gnode("n1", 0, 0), gnode("n2", 1, 1), gnode("n3", 2, 2)},
				ir.Edges{gedge("n1", "out", "n2", "in"), gedge("n2", "out", "n3", "in")},
			)
			state.Graph.Configs = map[string]msgpack.EncodedJSON{
				"n2": {"type": "stage"},
				"n3": {"type": "stage"},
			}
			out := MustSucceed(arc.Reduce(state, arc.NewRemoveNodeAction(arc.RemoveNodePayload{
				Key: "n2",
			})))
			Expect(out.Graph.Nodes).To(Equal(graph.Nodes{gnode("n1", 0, 0), gnode("n3", 2, 2)}))
			Expect(out.Graph.Edges).To(BeEmpty())
			Expect(out.Graph.Configs).ToNot(HaveKey("n2"))
			Expect(out.Graph.Configs).To(HaveKey("n3"))
		})
		It("Should keep unrelated edges intact", func() {
			state := withGraph(
				graph.Nodes{gnode("n1", 0, 0), gnode("n2", 1, 1), gnode("n3", 2, 2)},
				ir.Edges{gedge("n1", "out", "n3", "in")},
			)
			out := MustSucceed(arc.Reduce(state, arc.NewRemoveNodeAction(arc.RemoveNodePayload{
				Key: "n2",
			})))
			Expect(out.Graph.Edges).To(Equal(ir.Edges{gedge("n1", "out", "n3", "in")}))
		})
		It("Should be a no-op when the key does not match any node", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, ir.Edges{gedge("n1", "out", "n1", "in")})
			out := MustSucceed(arc.Reduce(state, arc.NewRemoveNodeAction(arc.RemoveNodePayload{
				Key: "ghost",
			})))
			Expect(out.Graph.Nodes).To(Equal(state.Graph.Nodes))
			Expect(out.Graph.Edges).To(Equal(state.Graph.Edges))
		})
	})

	Describe("AddEdge", func() {
		It("Should append an edge whose source and target are not yet present", func() {
			state := withGraph(nil, ir.Edges{gedge("a", "o", "b", "i")})
			out := MustSucceed(arc.Reduce(state, arc.NewAddEdgeAction(arc.AddEdgePayload{
				Edge: gedge("b", "o", "c", "i"),
			})))
			Expect(out.Graph.Edges).To(HaveLen(2))
			Expect(out.Graph.Edges[1]).To(Equal(gedge("b", "o", "c", "i")))
		})
		It("Should be a no-op when an edge with the same source and target exists", func() {
			state := withGraph(nil, ir.Edges{gedge("a", "o", "b", "i")})
			out := MustSucceed(arc.Reduce(state, arc.NewAddEdgeAction(arc.AddEdgePayload{
				Edge: gedge("a", "o", "b", "i"),
			})))
			Expect(out.Graph.Edges).To(Equal(state.Graph.Edges))
		})
	})

	Describe("RemoveEdge", func() {
		It("Should remove the edge matching the source and target handles", func() {
			state := withGraph(nil, ir.Edges{
				gedge("a", "o", "b", "i"),
				gedge("b", "o", "c", "i"),
			})
			out := MustSucceed(arc.Reduce(state, arc.NewRemoveEdgeAction(arc.RemoveEdgePayload{
				Source: ir.Handle{Node: "a", Param: "o"},
				Target: ir.Handle{Node: "b", Param: "i"},
			})))
			Expect(out.Graph.Edges).To(Equal(ir.Edges{gedge("b", "o", "c", "i")}))
		})
		It("Should be a no-op when no edge matches the handles", func() {
			state := withGraph(nil, ir.Edges{gedge("a", "o", "b", "i")})
			out := MustSucceed(arc.Reduce(state, arc.NewRemoveEdgeAction(arc.RemoveEdgePayload{
				Source: ir.Handle{Node: "x", Param: "o"},
				Target: ir.Handle{Node: "y", Param: "i"},
			})))
			Expect(out.Graph.Edges).To(Equal(state.Graph.Edges))
		})
	})

	Describe("ReduceAll scenarios", func() {
		It("Should build a complete graph from an empty module", func() {
			actions := []arc.Action{
				arc.NewSetModeAction(arc.SetModePayload{Mode: arc.ModeGraph}),
				arc.NewSetNodeAction(arc.SetNodePayload{Node: gnode("src", 0, 0)}),
				arc.NewSetNodeAction(arc.SetNodePayload{Node: gnode("op", 100, 0)}),
				arc.NewSetNodeAction(arc.SetNodePayload{Node: gnode("sink", 200, 0)}),
				arc.NewAddEdgeAction(arc.AddEdgePayload{Edge: gedge("src", "out", "op", "in")}),
				arc.NewAddEdgeAction(arc.AddEdgePayload{Edge: gedge("op", "out", "sink", "in")}),
			}
			out := MustSucceed(arc.Reduce(arc.Arc{}, actions...))
			Expect(out.Mode).To(Equal(arc.ModeGraph))
			Expect(out.Graph.Nodes).To(HaveLen(3))
			Expect(out.Graph.Edges).To(HaveLen(2))
		})
		It("Should converge to the final position after a drag storm", func() {
			state := withGraph(graph.Nodes{gnode("n", 0, 0)}, nil)
			actions := make([]arc.Action, 0, 30)
			for i := range 30 {
				actions = append(actions, arc.NewSetNodePositionAction(arc.SetNodePositionPayload{
					Key:      "n",
					Position: spatial.XY{X: float64(i), Y: float64(i * 2)},
				}))
			}
			out := MustSucceed(arc.Reduce(state, actions...))
			Expect(out.Graph.Nodes[0].Position).To(Equal(spatial.XY{X: 29, Y: 58}))
		})
		It("Should leave state untouched when given an empty action list", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, nil)
			Expect(MustSucceed(arc.Reduce(state))).To(Equal(state))
		})
	})

	Describe("Action envelope", func() {
		It("Should return state unchanged when Type is unrecognized", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, nil)
			Expect(MustSucceed(arc.Reduce(state, arc.Action{Type: "totally_made_up"}))).To(Equal(state))
		})

		DescribeTable("Should error when Type names a variant whose payload pointer is nil",
			func(actionType string) {
				state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, nil)
				out, err := arc.Reduce(state, arc.Action{Type: actionType})
				Expect(err).To(MatchError(union.ErrMissingPayload))
				Expect(out).To(Equal(state))
			},
			Entry("rename", arc.ActionTypeRename),
			Entry("set_mode", arc.ActionTypeSetMode),
			Entry("set_node", arc.ActionTypeSetNode),
			Entry("set_node_position", arc.ActionTypeSetNodePosition),
			Entry("set_node_config", arc.ActionTypeSetNodeConfig),
			Entry("remove_node", arc.ActionTypeRemoveNode),
			Entry("add_edge", arc.ActionTypeAddEdge),
			Entry("remove_edge", arc.ActionTypeRemoveEdge),
		)

		It("Should dispatch on Type and ignore extra payload pointers", func() {
			state := withGraph(graph.Nodes{gnode("n1", 0, 0)}, nil)
			pos := arc.SetNodePositionPayload{Key: "n1", Position: spatial.XY{X: 50, Y: 60}}
			edge := arc.AddEdgePayload{Edge: gedge("n1", "o", "n2", "i")}
			out := MustSucceed(arc.Reduce(state, arc.Action{
				Type:            arc.ActionTypeSetNodePosition,
				SetNodePosition: &pos,
				AddEdge:         &edge,
			}))
			Expect(out.Graph.Nodes[0].Position).To(Equal(spatial.XY{X: 50, Y: 60}))
			Expect(out.Graph.Edges).To(BeEmpty())
		})
	})
})
