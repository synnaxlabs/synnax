// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

// node constructs a node at the given coordinates. zIndex is left zero.
func node(key string, x, y float64) schematic.Node {
	return schematic.Node{Key: key, Position: spatial.XY{X: x, Y: y}}
}

// edge constructs an edge from source node + handle to target node + handle.
func edge(key, srcNode, srcParam, tgtNode, tgtParam string) schematic.Edge {
	return schematic.Edge{
		Key:    key,
		Source: schematic.Handle{Node: srcNode, Param: srcParam},
		Target: schematic.Handle{Node: tgtNode, Param: tgtParam},
	}
}

var _ = Describe("Reducer", func() {
	Describe("SetNodePosition", func() {
		It("Should move the matching node to the new position", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{
				node("n1", 0, 0),
				node("n2", 5, 5),
			}}
			out := MustSucceed(schematic.NewSetNodePositionAction(schematic.SetNodePosition{
				Key:      "n1",
				Position: spatial.XY{X: 100, Y: 200},
			}).Reduce(state))
			Expect(out.Nodes).To(HaveLen(2))
			Expect(out.Nodes[0]).To(Equal(node("n1", 100, 200)))
			Expect(out.Nodes[1]).To(Equal(node("n2", 5, 5)))
		})
		It("Should be a no-op when the key does not match any node", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			out := MustSucceed(schematic.NewSetNodePositionAction(schematic.SetNodePosition{
				Key:      "ghost",
				Position: spatial.XY{X: 100, Y: 200},
			}).Reduce(state))
			Expect(out.Nodes).To(Equal([]schematic.Node{node("n1", 0, 0)}))
		})
		It("Should only move the first matching node when keys are duplicated", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{
				node("dup", 0, 0),
				node("dup", 1, 1),
			}}
			out := MustSucceed(schematic.NewSetNodePositionAction(schematic.SetNodePosition{
				Key:      "dup",
				Position: spatial.XY{X: 9, Y: 9},
			}).Reduce(state))
			Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 9, Y: 9}))
			Expect(out.Nodes[1].Position).To(Equal(spatial.XY{X: 1, Y: 1}))
		})
	})

	Describe("SetNode", func() {
		It("Should append the node to the end of the slice when no node has the same key", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			out := MustSucceed(schematic.NewSetNodeAction(schematic.SetNode{
				Node: node("n2", 1, 2),
			}).Reduce(state))
			Expect(out.Nodes).To(Equal([]schematic.Node{node("n1", 0, 0), node("n2", 1, 2)}))
		})
		It("Should write config under the node's key when config is non-nil", func() {
			state := schematic.Schematic{}
			out := MustSucceed(schematic.NewSetNodeAction(schematic.SetNode{
				Node:   node("n1", 0, 0),
				Config: msgpack.EncodedJSON{"label": "Pump", "color": "#ff0000"},
			}).Reduce(state))
			Expect(out.Configs).To(HaveKey("n1"))
			Expect(out.Configs["n1"]).To(Equal(msgpack.EncodedJSON{
				"label": "Pump",
				"color": "#ff0000",
			}))
		})
		It("Should leave configs untouched when the action's config is nil", func() {
			state := schematic.Schematic{}
			out := MustSucceed(schematic.NewSetNodeAction(schematic.SetNode{
				Node: node("n1", 0, 0),
			}).Reduce(state))
			Expect(out.Configs).To(BeNil())
		})
		It("Should replace an existing node in place when the key already exists, preserving slice index", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{
				node("n1", 0, 0),
				node("n2", 1, 1),
				node("n3", 2, 2),
			}}
			out := MustSucceed(schematic.NewSetNodeAction(schematic.SetNode{
				Node: node("n2", 9, 9),
			}).Reduce(state))
			Expect(out.Nodes).To(HaveLen(3))
			Expect(out.Nodes[0]).To(Equal(node("n1", 0, 0)))
			Expect(out.Nodes[1]).To(Equal(node("n2", 9, 9)))
			Expect(out.Nodes[2]).To(Equal(node("n3", 2, 2)))
		})
	})

	Describe("RemoveNode", func() {
		It("Should remove the matching node and any config stored under its key", func() {
			state := schematic.Schematic{
				Nodes: []schematic.Node{node("n1", 0, 0), node("n2", 1, 1)},
				Configs: map[string]msgpack.EncodedJSON{
					"n1": {"label": "Pump"},
					"n2": {"label": "Tank"},
				},
			}
			out := MustSucceed(schematic.NewRemoveNodeAction(schematic.RemoveNode{
				Key: "n1",
			}).Reduce(state))
			Expect(out.Nodes).To(Equal([]schematic.Node{node("n2", 1, 1)}))
			Expect(out.Configs).ToNot(HaveKey("n1"))
			Expect(out.Configs).To(HaveKey("n2"))
		})
		It("Should leave existing edges intact even when they reference the removed node", func() {
			state := schematic.Schematic{
				Nodes: []schematic.Node{node("n1", 0, 0), node("n2", 1, 1)},
				Edges: []schematic.Edge{edge("e1", "n1", "out", "n2", "in")},
			}
			out := MustSucceed(schematic.NewRemoveNodeAction(schematic.RemoveNode{
				Key: "n1",
			}).Reduce(state))
			Expect(out.Edges).To(HaveLen(1))
			Expect(out.Edges[0].Source.Node).To(Equal("n1"))
		})
		It("Should be a no-op when the key does not match any node", func() {
			state := schematic.Schematic{
				Nodes:   []schematic.Node{node("n1", 0, 0)},
				Configs: map[string]msgpack.EncodedJSON{"n1": {"label": "Pump"}},
			}
			out := MustSucceed(schematic.NewRemoveNodeAction(schematic.RemoveNode{
				Key: "ghost",
			}).Reduce(state))
			Expect(out.Nodes).To(Equal(state.Nodes))
			Expect(out.Configs).To(Equal(state.Configs))
		})
	})

	Describe("AddEdge", func() {
		It("Should append an edge whose key is not yet present", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{edge("e1", "a", "o", "b", "i")}}
			out := MustSucceed(schematic.NewAddEdgeAction(schematic.AddEdge{
				Edge: edge("e2", "b", "o", "c", "i"),
			}).Reduce(state))
			Expect(out.Edges).To(HaveLen(2))
			Expect(out.Edges[1].Key).To(Equal("e2"))
		})
		It("Should be a no-op when an edge with the same key already exists", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{
				edge("e1", "a", "o", "b", "i"),
				edge("e2", "b", "o", "c", "i"),
			}}
			out := MustSucceed(schematic.NewAddEdgeAction(schematic.AddEdge{
				Edge: edge("e1", "x", "y", "z", "w"),
			}).Reduce(state))
			Expect(out.Edges).To(Equal(state.Edges))
		})
	})

	Describe("RemoveEdge", func() {
		It("Should remove the matching edge", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{
				edge("e1", "a", "o", "b", "i"),
				edge("e2", "b", "o", "c", "i"),
			}}
			out := MustSucceed(schematic.NewRemoveEdgeAction(schematic.RemoveEdge{
				Key: "e1",
			}).Reduce(state))
			Expect(out.Edges).To(Equal([]schematic.Edge{edge("e2", "b", "o", "c", "i")}))
		})
		It("Should be a no-op when the key does not match any edge", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{edge("e1", "a", "o", "b", "i")}}
			out := MustSucceed(schematic.NewRemoveEdgeAction(schematic.RemoveEdge{
				Key: "ghost",
			}).Reduce(state))
			Expect(out.Edges).To(Equal(state.Edges))
		})
	})

	Describe("SetConfig", func() {
		It("Should write the config entry under the given key", func() {
			state := schematic.Schematic{}
			out := MustSucceed(schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "n1",
				Config: msgpack.EncodedJSON{"label": "Pump"},
			}).Reduce(state))
			Expect(out.Configs["n1"]).To(Equal(msgpack.EncodedJSON{"label": "Pump"}))
		})
		It("Should merge payload fields into an existing config entry", func() {
			state := schematic.Schematic{Configs: map[string]msgpack.EncodedJSON{
				"n1": {"label": "Old", "color": "#ff0000"},
			}}
			out := MustSucceed(schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "n1",
				Config: msgpack.EncodedJSON{"label": "New"},
			}).Reduce(state))
			Expect(out.Configs["n1"]).To(Equal(msgpack.EncodedJSON{
				"label": "New",
				"color": "#ff0000",
			}))
		})
		It("Should accept a key that does not match any node or edge", func() {
			state := schematic.Schematic{}
			out := MustSucceed(schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "orphan",
				Config: msgpack.EncodedJSON{"data": 1},
			}).Reduce(state))
			Expect(out.Configs["orphan"]).To(Equal(msgpack.EncodedJSON{"data": 1}))
		})
		It("Should override the payload color with the edge's source color on insert", func() {
			state := schematic.Schematic{
				Edges: []schematic.Edge{edge("e1", "src", "o", "tgt", "i")},
				Configs: map[string]msgpack.EncodedJSON{
					"src": {"color": "#00ff00"},
				},
			}
			out := MustSucceed(schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "e1",
				Config: msgpack.EncodedJSON{"variant": "pipe", "color": "#000000"},
			}).Reduce(state))
			Expect(out.Configs["e1"]).To(Equal(msgpack.EncodedJSON{
				"variant": "pipe",
				"color":   "#00ff00",
			}))
		})
		It("Should inherit the edge's source color when the payload omits color", func() {
			state := schematic.Schematic{
				Edges: []schematic.Edge{edge("e1", "src", "o", "tgt", "i")},
				Configs: map[string]msgpack.EncodedJSON{
					"src": {"color": "#00ff00"},
				},
			}
			out := MustSucceed(schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "e1",
				Config: msgpack.EncodedJSON{"variant": "pipe"},
			}).Reduce(state))
			Expect(out.Configs["e1"]).To(Equal(msgpack.EncodedJSON{
				"variant": "pipe",
				"color":   "#00ff00",
			}))
		})
		It("Should leave the payload untouched when the source node has no color", func() {
			state := schematic.Schematic{
				Edges: []schematic.Edge{edge("e1", "src", "o", "tgt", "i")},
				Configs: map[string]msgpack.EncodedJSON{
					"src": {"label": "Pump"},
				},
			}
			out := MustSucceed(schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "e1",
				Config: msgpack.EncodedJSON{"variant": "pipe", "color": "#000000"},
			}).Reduce(state))
			Expect(out.Configs["e1"]).To(Equal(msgpack.EncodedJSON{
				"variant": "pipe",
				"color":   "#000000",
			}))
		})
		It("Should leave the payload untouched when the source node has no config", func() {
			state := schematic.Schematic{
				Edges: []schematic.Edge{edge("e1", "src", "o", "tgt", "i")},
			}
			out := MustSucceed(schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "e1",
				Config: msgpack.EncodedJSON{"variant": "pipe", "color": "#000000"},
			}).Reduce(state))
			Expect(out.Configs["e1"]).To(Equal(msgpack.EncodedJSON{
				"variant": "pipe",
				"color":   "#000000",
			}))
		})
		It("Should not override the color when merging into an existing edge config", func() {
			state := schematic.Schematic{
				Edges: []schematic.Edge{edge("e1", "src", "o", "tgt", "i")},
				Configs: map[string]msgpack.EncodedJSON{
					"src": {"color": "#00ff00"},
					"e1":  {"variant": "pipe", "color": "#000000"},
				},
			}
			out := MustSucceed(schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "e1",
				Config: msgpack.EncodedJSON{"variant": "electric"},
			}).Reduce(state))
			Expect(out.Configs["e1"]).To(Equal(msgpack.EncodedJSON{
				"variant": "electric",
				"color":   "#000000",
			}))
		})
	})

	Describe("ReduceAll real-world scenarios", func() {
		It("Should converge to the final position after a drag storm", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("pump", 0, 0)}}
			actions := make([]schematic.Action, 0, 30)
			for i := range 30 {
				actions = append(actions, schematic.NewSetNodePositionAction(schematic.SetNodePosition{
					Key:      "pump",
					Position: spatial.XY{X: float64(i), Y: float64(i * 2)},
				}))
			}
			out := MustSucceed(schematic.ReduceAll(state, actions))
			Expect(out.Nodes).To(HaveLen(1))
			Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 29, Y: 58}))
		})

		It("Should build a complete graph from an empty schematic", func() {
			state := schematic.Schematic{}
			actions := []schematic.Action{
				schematic.NewSetNodeAction(schematic.SetNode{Node: node("pump", 0, 0)}),
				schematic.NewSetNodeAction(schematic.SetNode{Node: node("valve", 100, 0)}),
				schematic.NewSetNodeAction(schematic.SetNode{Node: node("tank", 200, 0)}),
				schematic.NewAddEdgeAction(schematic.AddEdge{Edge: edge("e1", "pump", "out", "valve", "in")}),
				schematic.NewAddEdgeAction(schematic.AddEdge{Edge: edge("e2", "valve", "out", "tank", "in")}),
				schematic.NewSetConfigAction(schematic.SetConfig{
					Key:    "pump",
					Config: msgpack.EncodedJSON{"label": "Main Pump"},
				}),
				schematic.NewSetConfigAction(schematic.SetConfig{
					Key:    "e1",
					Config: msgpack.EncodedJSON{"variant": "pipe"},
				}),
			}
			out := MustSucceed(schematic.ReduceAll(state, actions))
			Expect(out.Nodes).To(HaveLen(3))
			Expect(out.Edges).To(HaveLen(2))
			Expect(out.Configs).To(HaveLen(2))
			Expect(out.Configs["pump"]).To(Equal(msgpack.EncodedJSON{"label": "Main Pump"}))
			Expect(out.Configs["e1"]).To(Equal(msgpack.EncodedJSON{"variant": "pipe"}))
		})

		It("Should drop config but keep dangling edges when a node is removed and re-added", func() {
			state := schematic.Schematic{
				Nodes:   []schematic.Node{node("n1", 0, 0), node("n2", 1, 1)},
				Edges:   []schematic.Edge{edge("e1", "n1", "o", "n2", "i")},
				Configs: map[string]msgpack.EncodedJSON{"n1": {"label": "v1"}},
			}
			actions := []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNode{Key: "n1"}),
				schematic.NewSetNodeAction(schematic.SetNode{Node: node("n1", 50, 50)}),
			}
			out := MustSucceed(schematic.ReduceAll(state, actions))
			Expect(out.Nodes).To(HaveLen(2))
			Expect(out.Nodes[1]).To(Equal(node("n1", 50, 50)))
			Expect(out.Configs).ToNot(HaveKey("n1"))
			Expect(out.Edges).To(HaveLen(1))
			Expect(out.Edges[0].Source.Node).To(Equal("n1"))
		})

		It("Should converge an idempotent action sequence to the same state as a single application", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			single := []schematic.Action{
				schematic.NewSetNodePositionAction(schematic.SetNodePosition{
					Key:      "n1",
					Position: spatial.XY{X: 10, Y: 20},
				}),
			}
			doubled := []schematic.Action{single[0], single[0], single[0]}
			Expect(MustSucceed(schematic.ReduceAll(state, single))).To(Equal(MustSucceed(schematic.ReduceAll(state, doubled))))
		})

		It("Should apply a 50-action editor session and converge to a coherent schematic", func() {
			state := schematic.Schematic{}
			var actions []schematic.Action
			for i := range 5 {
				actions = append(actions, schematic.NewSetNodeAction(schematic.SetNode{
					Node: node("n"+string(rune('0'+i)), float64(i*100), 0),
				}))
			}
			for i := range 5 {
				key := "n" + string(rune('0'+i))
				actions = append(actions, schematic.NewSetNodePositionAction(schematic.SetNodePosition{
					Key:      key,
					Position: spatial.XY{X: float64(i * 100), Y: 50},
				}))
				actions = append(actions, schematic.NewSetNodePositionAction(schematic.SetNodePosition{
					Key:      key,
					Position: spatial.XY{X: float64(i * 100), Y: 100},
				}))
			}
			for i := range 4 {
				src := "n" + string(rune('0'+i))
				dst := "n" + string(rune('0'+i+1))
				actions = append(actions, schematic.NewAddEdgeAction(schematic.AddEdge{
					Edge: edge("e"+string(rune('0'+i)), src, "out", dst, "in"),
				}))
			}
			for i := range 3 {
				actions = append(actions, schematic.NewSetConfigAction(schematic.SetConfig{
					Key:    "n" + string(rune('0'+i)),
					Config: msgpack.EncodedJSON{"label": "node " + string(rune('0'+i))},
				}))
			}
			actions = append(actions, schematic.NewSetConfigAction(schematic.SetConfig{
				Key:    "e1",
				Config: msgpack.EncodedJSON{"variant": "electric"},
			}))
			out := MustSucceed(schematic.ReduceAll(state, actions))
			Expect(out.Nodes).To(HaveLen(5))
			Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 0, Y: 100}))
			Expect(out.Nodes[4].Position).To(Equal(spatial.XY{X: 400, Y: 100}))
			Expect(out.Edges).To(HaveLen(4))
			Expect(out.Configs).To(HaveLen(4))
		})

		It("Should leave state untouched when given an empty action list", func() {
			state := schematic.Schematic{
				Key:   uuid.New(),
				Name:  "empty",
				Nodes: []schematic.Node{node("n1", 0, 0)},
			}
			Expect(MustSucceed(schematic.ReduceAll(state, nil))).To(Equal(state))
			Expect(MustSucceed(schematic.ReduceAll(state, []schematic.Action{}))).To(Equal(state))
		})
	})

	Describe("Action envelope", func() {
		It("Should return state unchanged when Type is unrecognized", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			Expect(MustSucceed(schematic.Action{Type: "totally_made_up"}.Reduce(state))).To(Equal(state))
		})

		It("Should return state unchanged when Type names a variant whose payload pointer is nil", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			Expect(MustSucceed(schematic.Action{Type: schematic.ActionTypeSetNodePosition}.Reduce(state))).To(Equal(state))
			Expect(MustSucceed(schematic.Action{Type: schematic.ActionTypeSetNode}.Reduce(state))).To(Equal(state))
			Expect(MustSucceed(schematic.Action{Type: schematic.ActionTypeRemoveNode}.Reduce(state))).To(Equal(state))
			Expect(MustSucceed(schematic.Action{Type: schematic.ActionTypeAddEdge}.Reduce(state))).To(Equal(state))
			Expect(MustSucceed(schematic.Action{Type: schematic.ActionTypeRemoveEdge}.Reduce(state))).To(Equal(state))
			Expect(MustSucceed(schematic.Action{Type: schematic.ActionTypeSetConfig}.Reduce(state))).To(Equal(state))
			Expect(MustSucceed(schematic.Action{Type: schematic.ActionTypeSetNodeMeasured}.Reduce(state))).To(Equal(state))
		})

		It("Should dispatch on Type and ignore extra payload pointers when more than one is set", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			pos := schematic.SetNodePosition{Key: "n1", Position: spatial.XY{X: 50, Y: 60}}
			edge := schematic.AddEdge{Edge: edge("e_extra", "n1", "o", "n2", "i")}
			out := MustSucceed(schematic.Action{
				Type:            schematic.ActionTypeSetNodePosition,
				SetNodePosition: &pos,
				AddEdge:         &edge,
			}.Reduce(state))
			Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 50, Y: 60}))
			Expect(out.Edges).To(BeEmpty())
		})
	})
})
