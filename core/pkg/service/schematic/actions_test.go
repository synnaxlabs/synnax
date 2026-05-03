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
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

// emptyLegend returns the zero-value Legend used when a test does not care
// about legend state. It is non-nil so equality comparisons against a default
// schematic don't trip on a nil colors map.
func emptyLegend() schematic.Legend {
	return schematic.Legend{Colors: map[string]color.Color{}}
}

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

	Describe("AddNode", func() {
		It("Should append the node to the end of the slice", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			out := MustSucceed(schematic.NewAddNodeAction(schematic.AddNode{
				Node: node("n2", 1, 2),
			}).Reduce(state))
			Expect(out.Nodes).To(Equal([]schematic.Node{node("n1", 0, 0), node("n2", 1, 2)}))
		})
		It("Should write props under the node's key when props is non-nil", func() {
			state := schematic.Schematic{}
			out := MustSucceed(schematic.NewAddNodeAction(schematic.AddNode{
				Node:  node("n1", 0, 0),
				Props: msgpack.EncodedJSON{"label": "Pump", "color": "#ff0000"},
			}).Reduce(state))
			Expect(out.Props).To(HaveKey("n1"))
			Expect(out.Props["n1"]).To(Equal(msgpack.EncodedJSON{
				"label": "Pump",
				"color": "#ff0000",
			}))
		})
		It("Should leave props untouched when the action's props is nil", func() {
			state := schematic.Schematic{}
			out := MustSucceed(schematic.NewAddNodeAction(schematic.AddNode{
				Node: node("n1", 0, 0),
			}).Reduce(state))
			Expect(out.Props).To(BeNil())
		})
		It("Should not append a duplicate-key node into the slice as a guard - it appends, locking current behavior", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			out := MustSucceed(schematic.NewAddNodeAction(schematic.AddNode{
				Node: node("n1", 9, 9),
			}).Reduce(state))
			Expect(out.Nodes).To(HaveLen(2))
			Expect(out.Nodes[0]).To(Equal(node("n1", 0, 0)))
			Expect(out.Nodes[1]).To(Equal(node("n1", 9, 9)))
		})
	})

	Describe("RemoveNode", func() {
		It("Should remove the matching node and any props stored under its key", func() {
			state := schematic.Schematic{
				Nodes: []schematic.Node{node("n1", 0, 0), node("n2", 1, 1)},
				Props: map[string]msgpack.EncodedJSON{
					"n1": {"label": "Pump"},
					"n2": {"label": "Tank"},
				},
			}
			out := MustSucceed(schematic.NewRemoveNodeAction(schematic.RemoveNode{
				Key: "n1",
			}).Reduce(state))
			Expect(out.Nodes).To(Equal([]schematic.Node{node("n2", 1, 1)}))
			Expect(out.Props).ToNot(HaveKey("n1"))
			Expect(out.Props).To(HaveKey("n2"))
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
				Nodes: []schematic.Node{node("n1", 0, 0)},
				Props: map[string]msgpack.EncodedJSON{"n1": {"label": "Pump"}},
			}
			out := MustSucceed(schematic.NewRemoveNodeAction(schematic.RemoveNode{
				Key: "ghost",
			}).Reduce(state))
			Expect(out.Nodes).To(Equal(state.Nodes))
			Expect(out.Props).To(Equal(state.Props))
		})
	})

	Describe("SetEdge", func() {
		It("Should append an edge whose key is not yet present", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{edge("e1", "a", "o", "b", "i")}}
			out := MustSucceed(schematic.NewSetEdgeAction(schematic.SetEdge{
				Edge: edge("e2", "b", "o", "c", "i"),
			}).Reduce(state))
			Expect(out.Edges).To(HaveLen(2))
			Expect(out.Edges[1].Key).To(Equal("e2"))
		})
		It("Should replace an existing edge in place, preserving slice index", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{
				edge("e1", "a", "o", "b", "i"),
				edge("e2", "b", "o", "c", "i"),
				edge("e3", "c", "o", "d", "i"),
			}}
			out := MustSucceed(schematic.NewSetEdgeAction(schematic.SetEdge{
				Edge: edge("e2", "x", "y", "z", "w"),
			}).Reduce(state))
			Expect(out.Edges).To(HaveLen(3))
			Expect(out.Edges[0].Key).To(Equal("e1"))
			Expect(out.Edges[1]).To(Equal(edge("e2", "x", "y", "z", "w")))
			Expect(out.Edges[2].Key).To(Equal("e3"))
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

	Describe("SetProps", func() {
		It("Should write the props entry under the given key", func() {
			state := schematic.Schematic{}
			out := MustSucceed(schematic.NewSetPropsAction(schematic.SetProps{
				Key:   "n1",
				Props: msgpack.EncodedJSON{"label": "Pump"},
			}).Reduce(state))
			Expect(out.Props["n1"]).To(Equal(msgpack.EncodedJSON{"label": "Pump"}))
		})
		It("Should overwrite an existing props entry", func() {
			state := schematic.Schematic{Props: map[string]msgpack.EncodedJSON{
				"n1": {"label": "Old"},
			}}
			out := MustSucceed(schematic.NewSetPropsAction(schematic.SetProps{
				Key:   "n1",
				Props: msgpack.EncodedJSON{"label": "New"},
			}).Reduce(state))
			Expect(out.Props["n1"]).To(Equal(msgpack.EncodedJSON{"label": "New"}))
		})
		It("Should accept a key that does not match any node or edge", func() {
			state := schematic.Schematic{}
			out := MustSucceed(schematic.NewSetPropsAction(schematic.SetProps{
				Key:   "orphan",
				Props: msgpack.EncodedJSON{"data": 1},
			}).Reduce(state))
			Expect(out.Props["orphan"]).To(Equal(msgpack.EncodedJSON{"data": 1}))
		})
	})

	Describe("SetAuthority", func() {
		It("Should replace the authority value", func() {
			state := schematic.Schematic{Authority: 1}
			out := MustSucceed(schematic.NewSetAuthorityAction(schematic.SetAuthority{
				Value: 200,
			}).Reduce(state))
			Expect(out.Authority).To(BeEquivalentTo(200))
		})
	})

	Describe("SetLegend", func() {
		It("Should replace the legend wholesale", func() {
			state := schematic.Schematic{Legend: schematic.Legend{Visible: false}}
			newLegend := schematic.Legend{
				Visible: true,
				Colors:  map[string]color.Color{"on": {R: 255, G: 0, B: 0, A: 1}},
			}
			out := MustSucceed(schematic.NewSetLegendAction(schematic.SetLegend{
				Legend: newLegend,
			}).Reduce(state))
			Expect(out.Legend).To(Equal(newLegend))
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
			state := schematic.Schematic{Legend: emptyLegend()}
			actions := []schematic.Action{
				schematic.NewAddNodeAction(schematic.AddNode{Node: node("pump", 0, 0)}),
				schematic.NewAddNodeAction(schematic.AddNode{Node: node("valve", 100, 0)}),
				schematic.NewAddNodeAction(schematic.AddNode{Node: node("tank", 200, 0)}),
				schematic.NewSetEdgeAction(schematic.SetEdge{Edge: edge("e1", "pump", "out", "valve", "in")}),
				schematic.NewSetEdgeAction(schematic.SetEdge{Edge: edge("e2", "valve", "out", "tank", "in")}),
				schematic.NewSetPropsAction(schematic.SetProps{
					Key:   "pump",
					Props: msgpack.EncodedJSON{"label": "Main Pump"},
				}),
				schematic.NewSetPropsAction(schematic.SetProps{
					Key:   "e1",
					Props: msgpack.EncodedJSON{"variant": "pipe"},
				}),
			}
			out := MustSucceed(schematic.ReduceAll(state, actions))
			Expect(out.Nodes).To(HaveLen(3))
			Expect(out.Edges).To(HaveLen(2))
			Expect(out.Props).To(HaveLen(2))
			Expect(out.Props["pump"]).To(Equal(msgpack.EncodedJSON{"label": "Main Pump"}))
			Expect(out.Props["e1"]).To(Equal(msgpack.EncodedJSON{"variant": "pipe"}))
		})

		It("Should drop props but keep dangling edges when a node is removed and re-added", func() {
			state := schematic.Schematic{
				Nodes: []schematic.Node{node("n1", 0, 0), node("n2", 1, 1)},
				Edges: []schematic.Edge{edge("e1", "n1", "o", "n2", "i")},
				Props: map[string]msgpack.EncodedJSON{"n1": {"label": "v1"}},
			}
			actions := []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNode{Key: "n1"}),
				schematic.NewAddNodeAction(schematic.AddNode{Node: node("n1", 50, 50)}),
			}
			out := MustSucceed(schematic.ReduceAll(state, actions))
			Expect(out.Nodes).To(HaveLen(2))
			Expect(out.Nodes[1]).To(Equal(node("n1", 50, 50)))
			Expect(out.Props).ToNot(HaveKey("n1"))
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
			state := schematic.Schematic{Legend: emptyLegend()}
			var actions []schematic.Action
			for i := range 5 {
				actions = append(actions, schematic.NewAddNodeAction(schematic.AddNode{
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
				actions = append(actions, schematic.NewSetEdgeAction(schematic.SetEdge{
					Edge: edge("e"+string(rune('0'+i)), src, "out", dst, "in"),
				}))
			}
			for i := range 3 {
				actions = append(actions, schematic.NewSetPropsAction(schematic.SetProps{
					Key:   "n" + string(rune('0'+i)),
					Props: msgpack.EncodedJSON{"label": "node " + string(rune('0'+i))},
				}))
			}
			actions = append(actions, schematic.NewSetPropsAction(schematic.SetProps{
				Key:   "e1",
				Props: msgpack.EncodedJSON{"variant": "electric"},
			}))
			actions = append(actions, schematic.NewSetAuthorityAction(schematic.SetAuthority{Value: 255}))
			actions = append(actions, schematic.NewSetLegendAction(schematic.SetLegend{
				Legend: schematic.Legend{Visible: true, Colors: map[string]color.Color{}},
			}))
			out := MustSucceed(schematic.ReduceAll(state, actions))
			Expect(out.Nodes).To(HaveLen(5))
			Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 0, Y: 100}))
			Expect(out.Nodes[4].Position).To(Equal(spatial.XY{X: 400, Y: 100}))
			Expect(out.Edges).To(HaveLen(4))
			Expect(out.Props).To(HaveLen(4))
			Expect(out.Authority).To(BeEquivalentTo(255))
			Expect(out.Legend.Visible).To(BeTrue())
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

})
