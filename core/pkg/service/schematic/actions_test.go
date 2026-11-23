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
	"encoding/json/v2"
	"fmt"
	"slices"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

// tankCfg constructs a tank node config with its defaults. hex is optional.
func tankCfg(label, hex string) schematic.NodeConfig {
	GinkgoHelper()
	cfg := schematic.TankNodeConfig{Label: schematic.LabelConfig{Label: label}}
	if hex != "" {
		cfg.StrokeColor = fixedColor(hex)
	}
	nc := schematic.NodeConfig{Variant: cfg}
	nc = defaults.Fill(nc)
	return nc
}

// pipeCfg constructs a pipe edge config. hex is optional.
func pipeCfg(hex string) schematic.EdgeConfig {
	GinkgoHelper()
	cfg := schematic.SegmentedEdgeConfig{Segments: []schematic.Segment{}}
	if hex != "" {
		cfg.StrokeColor = fixedColor(hex)
	}
	return defaults.Fill(schematic.EdgeConfig{
		Variant: schematic.PipeEdgeConfig{SegmentedEdgeConfig: cfg},
	})
}

// electricCfg constructs an electric edge config with no segments.
func electricCfg() schematic.EdgeConfig {
	return defaults.Fill(schematic.EdgeConfig{Variant: schematic.ElectricEdgeConfig{
		SegmentedEdgeConfig: schematic.SegmentedEdgeConfig{
			Segments: []schematic.Segment{},
		},
	}})
}

// groupCfg constructs an unlocked group box config.
func groupCfg() schematic.NodeConfig {
	return schematic.NodeConfig{Variant: schematic.GroupBoxNodeConfig{}}
}

// fields returns the JSON form of v, as a client sends a whole config.
func fields(v any) map[string]any {
	GinkgoHelper()
	var m map[string]any
	Expect(json.Unmarshal(MustSucceed(json.Marshal(v)), &m)).To(Succeed())
	return m
}

// node constructs a node with no config at the given coordinates.
func node(key string, x, y float64) schematic.Node {
	return schematic.Node{Key: key, Position: spatial.XY{X: x, Y: y}}
}

// configured returns n with the given config.
func configured(n schematic.Node, cfg schematic.NodeConfig) schematic.Node {
	n.Config = cfg
	return n
}

// groupBox constructs a group box node at the origin.
func groupBox(key string) schematic.Node {
	return configured(node(key, 0, 0), groupCfg())
}

// in returns n with the given parent.
func in(n schematic.Node, parent string) schematic.Node {
	n.Parent = &parent
	return n
}

// edge constructs an edge with no config from a source handle to a target handle.
func edge(key, srcNode, srcParam, tgtNode, tgtParam string) schematic.Edge {
	return schematic.Edge{
		Key:    key,
		Source: schematic.Handle{Node: srcNode, Param: srcParam},
		Target: schematic.Handle{Node: tgtNode, Param: tgtParam},
	}
}

// pipe constructs a pipe edge from a source handle to a target handle. hex is
// optional.
func pipe(key, srcNode, srcParam, tgtNode, tgtParam, hex string) schematic.Edge {
	GinkgoHelper()
	e := edge(key, srcNode, srcParam, tgtNode, tgtParam)
	e.Config = pipeCfg(hex)
	return e
}

var _ = Describe("Reducer", func() {
	Describe("SetNodePosition", func() {
		It("Should move the matching node to the new position", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{
				node("n1", 0, 0),
				node("n2", 5, 5),
			}}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "n1",
						Position: spatial.XY{X: 100, Y: 200},
					}),
				),
			)
			Expect(out.Nodes).To(HaveLen(2))
			Expect(out.Nodes[0]).To(Equal(node("n1", 100, 200)))
			Expect(out.Nodes[1]).To(Equal(node("n2", 5, 5)))
		})
		It("Should keep the config of the node it moves", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{
				configured(node("n1", 0, 0), tankCfg("Pump", "")),
			}}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "n1",
						Position: spatial.XY{X: 1, Y: 2},
					}),
				),
			)
			Expect(out.Nodes).To(Equal([]schematic.Node{
				configured(node("n1", 1, 2), tankCfg("Pump", "")),
			}))
		})
		It("Should be a no-op when the key does not match any node", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "ghost",
						Position: spatial.XY{X: 100, Y: 200},
					}),
				),
			)
			Expect(out.Nodes).To(Equal([]schematic.Node{node("n1", 0, 0)}))
		})
		It("Should only move the first matching node when keys are duplicated", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{
				node("dup", 0, 0),
				node("dup", 1, 1),
			}}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "dup",
						Position: spatial.XY{X: 9, Y: 9},
					}),
				),
			)
			Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 9, Y: 9}))
			Expect(out.Nodes[1].Position).To(Equal(spatial.XY{X: 1, Y: 1}))
		})
	})

	Describe("SetNode", func() {
		It(
			"Should append the node to the end of the slice when no node has the same key",
			func() {
				state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
				out := MustSucceed(
					schematic.Reduce(
						state,
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: node("n2", 1, 2),
						}),
					),
				)
				Expect(
					out.Nodes,
				).To(Equal([]schematic.Node{node("n1", 0, 0), node("n2", 1, 2)}))
			},
		)
		It("Should store the config the node carries", func() {
			out := MustSucceed(
				schematic.Reduce(
					schematic.Schematic{},
					schematic.NewSetNodeAction(schematic.SetNodePayload{
						Node: configured(node("n1", 0, 0), tankCfg("Pump", "#ff0000")),
					}),
				),
			)
			Expect(out.Nodes).To(Equal([]schematic.Node{
				configured(node("n1", 0, 0), tankCfg("Pump", "#ff0000")),
			}))
		})
		It(
			"Should replace an existing node and its config at the same index",
			func() {
				state := schematic.Schematic{Nodes: []schematic.Node{
					node("n1", 0, 0),
					configured(node("n2", 1, 1), tankCfg("Old", "#ff0000")),
					node("n3", 2, 2),
				}}
				out := MustSucceed(
					schematic.Reduce(
						state,
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: configured(node("n2", 9, 9), tankCfg("New", "")),
						}),
					),
				)
				Expect(out.Nodes).To(Equal([]schematic.Node{
					node("n1", 0, 0),
					configured(node("n2", 9, 9), tankCfg("New", "")),
					node("n3", 2, 2),
				}))
			},
		)
	})

	Describe("RemoveNode", func() {
		It("Should remove the matching node with its config", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{
				configured(node("n1", 0, 0), tankCfg("Pump", "")),
				configured(node("n2", 1, 1), tankCfg("Tank", "")),
			}}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{
						Key: "n1",
					}),
				),
			)
			Expect(out.Nodes).To(Equal([]schematic.Node{
				configured(node("n2", 1, 1), tankCfg("Tank", "")),
			}))
		})
		It("Should remove the edges that connect to the node", func() {
			state := schematic.Schematic{
				Nodes: []schematic.Node{
					node("n1", 0, 0),
					node("n2", 1, 1),
					node("n3", 2, 2),
				},
				Edges: []schematic.Edge{
					pipe("e1", "n1", "out", "n2", "in", "#ffffff"),
					pipe("e2", "n3", "out", "n1", "in", "#000000"),
					pipe("e3", "n2", "out", "n3", "in", "#ff0000"),
				},
			}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{
						Key: "n1",
					}),
				),
			)
			Expect(out.Edges).To(Equal([]schematic.Edge{
				pipe("e3", "n2", "out", "n3", "in", "#ff0000"),
			}))
		})
		It("Should be a no-op when the key does not match any node", func() {
			state := schematic.Schematic{
				Nodes: []schematic.Node{
					configured(node("n1", 0, 0), tankCfg("Pump", "")),
				},
				Edges: []schematic.Edge{pipe("e1", "n1", "out", "n1", "in", "")},
			}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{
						Key: "ghost",
					}),
				),
			)
			Expect(out).To(Equal(state))
		})
	})

	Describe("RemoveNode group children", func() {
		DescribeTable("Should promote the children of a removed node",
			func(nodes, expected []schematic.Node, key string) {
				out := MustSucceed(schematic.Reduce(
					schematic.Schematic{Nodes: nodes},
					schematic.NewRemoveNodeAction(
						schematic.RemoveNodePayload{Key: key},
					),
				))
				Expect(out.Nodes).To(Equal(expected))
			},
			Entry(
				"to the top level when the group is at the top level",
				[]schematic.Node{groupBox("g1"), in(node("n1", 0, 0), "g1")},
				[]schematic.Node{node("n1", 0, 0)},
				"g1",
			),
			Entry(
				"to the parent of a nested group",
				[]schematic.Node{
					groupBox("outer"),
					in(groupBox("inner"), "outer"),
					in(node("n1", 0, 0), "inner"),
					in(node("n2", 0, 0), "outer"),
				},
				[]schematic.Node{
					groupBox("outer"),
					in(node("n1", 0, 0), "outer"),
					in(node("n2", 0, 0), "outer"),
				},
				"inner",
			),
			Entry(
				"and leave grandchildren in their own group",
				[]schematic.Node{
					groupBox("outer"),
					in(groupBox("inner"), "outer"),
					in(node("n1", 0, 0), "inner"),
				},
				[]schematic.Node{groupBox("inner"), in(node("n1", 0, 0), "inner")},
				"outer",
			),
			Entry(
				"and leave the group alone when a child is removed",
				[]schematic.Node{
					groupBox("g1"),
					in(node("n1", 0, 0), "g1"),
					in(node("n2", 0, 0), "g1"),
				},
				[]schematic.Node{groupBox("g1"), in(node("n2", 0, 0), "g1")},
				"n1",
			),
		)
	})

	Describe("AddEdge", func() {
		It("Should append an edge whose key is not yet present", func() {
			state := schematic.Schematic{
				Nodes: []schematic.Node{
					node("a", 0, 0),
					node("b", 1, 1),
					node("c", 2, 2),
				},
				Edges: []schematic.Edge{edge("e1", "a", "o", "b", "i")},
			}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewAddEdgeAction(schematic.AddEdgePayload{
						Edge: pipe("e2", "b", "o", "c", "i", "#ff0000"),
					}),
				),
			)
			Expect(out.Edges).To(Equal([]schematic.Edge{
				edge("e1", "a", "o", "b", "i"),
				pipe("e2", "b", "o", "c", "i", "#ff0000"),
			}))
		})
		It("Should be a no-op when an edge with the same key already exists", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{
				edge("e1", "a", "o", "b", "i"),
				edge("e2", "b", "o", "c", "i"),
			}}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewAddEdgeAction(schematic.AddEdgePayload{
						Edge: edge("e1", "x", "y", "z", "w"),
					}),
				),
			)
			Expect(out.Edges).To(Equal(state.Edges))
		})
		DescribeTable(
			"Should be a no-op when a node the edge connects does not exist",
			func(e schematic.Edge) {
				state := schematic.Schematic{Nodes: []schematic.Node{node("a", 0, 0)}}
				out := MustSucceed(
					schematic.Reduce(
						state,
						schematic.NewAddEdgeAction(schematic.AddEdgePayload{Edge: e}),
					),
				)
				Expect(out.Edges).To(BeEmpty())
			},
			Entry("missing source", edge("e1", "ghost", "o", "a", "i")),
			Entry("missing target", edge("e1", "a", "o", "ghost", "i")),
		)
	})

	Describe("RemoveEdge", func() {
		It("Should remove the matching edge with its config", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{
				pipe("e1", "a", "o", "b", "i", "#ffffff"),
				pipe("e2", "b", "o", "c", "i", "#000000"),
			}}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewRemoveEdgeAction(schematic.RemoveEdgePayload{
						Key: "e1",
					}),
				),
			)
			Expect(out.Edges).To(Equal([]schematic.Edge{
				pipe("e2", "b", "o", "c", "i", "#000000"),
			}))
		})
		It("Should be a no-op when the key does not match any edge", func() {
			state := schematic.Schematic{Edges: []schematic.Edge{
				pipe("e1", "a", "o", "b", "i", "#ffffff"),
			}}
			out := MustSucceed(
				schematic.Reduce(
					state,
					schematic.NewRemoveEdgeAction(schematic.RemoveEdgePayload{
						Key: "ghost",
					}),
				),
			)
			Expect(out).To(Equal(state))
		})
	})

	Describe("ReduceAll real-world scenarios", func() {
		It("Should converge to the final position after a drag storm", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("pump", 0, 0)}}
			actions := make([]schematic.Action, 0, 30)
			for i := range 30 {
				actions = append(
					actions,
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "pump",
						Position: spatial.XY{X: float64(i), Y: float64(i * 2)},
					}),
				)
			}
			out := MustSucceed(schematic.Reduce(state, actions...))
			Expect(out.Nodes).To(HaveLen(1))
			Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 29, Y: 58}))
		})

		It("Should build a complete graph from an empty schematic", func() {
			actions := []schematic.Action{
				schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: configured(node("pump", 0, 0), tankCfg("Main Pump", "")),
				}),
				schematic.NewSetNodeAction(
					schematic.SetNodePayload{Node: node("valve", 100, 0)},
				),
				schematic.NewSetNodeAction(
					schematic.SetNodePayload{Node: node("tank", 200, 0)},
				),
				schematic.NewAddEdgeAction(schematic.AddEdgePayload{
					Edge: pipe("e1", "pump", "out", "valve", "in", ""),
				}),
				schematic.NewAddEdgeAction(schematic.AddEdgePayload{
					Edge: edge("e2", "valve", "out", "tank", "in"),
				}),
			}
			out := MustSucceed(schematic.Reduce(schematic.Schematic{}, actions...))
			Expect(out.Nodes).To(HaveLen(3))
			Expect(out.Nodes[0].Config).To(Equal(tankCfg("Main Pump", "")))
			Expect(out.Edges).To(HaveLen(2))
			Expect(out.Edges[0].Config).To(Equal(pipeCfg("")))
		})

		It(
			"Should drop the config and edges of a node that is removed and re-added",
			func() {
				state := schematic.Schematic{
					Nodes: []schematic.Node{
						configured(node("n1", 0, 0), tankCfg("v1", "")),
						node("n2", 1, 1),
					},
					Edges: []schematic.Edge{edge("e1", "n1", "o", "n2", "i")},
				}
				actions := []schematic.Action{
					schematic.NewRemoveNodeAction(
						schematic.RemoveNodePayload{Key: "n1"},
					),
					schematic.NewSetNodeAction(
						schematic.SetNodePayload{Node: node("n1", 50, 50)},
					),
				}
				out := MustSucceed(schematic.Reduce(state, actions...))
				Expect(out.Nodes).To(Equal([]schematic.Node{
					node("n2", 1, 1),
					node("n1", 50, 50),
				}))
				Expect(out.Edges).To(BeEmpty())
			},
		)

		It(
			"Should converge an idempotent action sequence to the same state as a single application",
			func() {
				state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
				single := []schematic.Action{
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "n1",
						Position: spatial.XY{X: 10, Y: 20},
					}),
				}
				doubled := []schematic.Action{single[0], single[0], single[0]}
				Expect(
					MustSucceed(schematic.Reduce(state, single...)),
				).To(Equal(MustSucceed(schematic.Reduce(state, doubled...))))
			},
		)

		It(
			"Should apply a 50-action editor session and converge to a coherent schematic",
			func() {
				actions := make([]schematic.Action, 0, 23)
				for i := range 5 {
					actions = append(
						actions,
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: node("n"+strconv.Itoa(i), float64(i*100), 0),
						}),
					)
				}
				for i := range 5 {
					key := "n" + strconv.Itoa(i)
					for _, y := range []float64{50, 100} {
						actions = append(
							actions,
							schematic.NewSetNodePositionAction(
								schematic.SetNodePositionPayload{
									Key:      key,
									Position: spatial.XY{X: float64(i * 100), Y: y},
								},
							),
						)
					}
				}
				for i := range 4 {
					actions = append(
						actions,
						schematic.NewAddEdgeAction(schematic.AddEdgePayload{
							Edge: pipe(
								"e"+strconv.Itoa(i),
								"n"+strconv.Itoa(i),
								"out",
								"n"+strconv.Itoa(i+1),
								"in",
								"",
							),
						}),
					)
				}
				for i := range 3 {
					actions = append(
						actions,
						schematic.NewSetAction(schematic.SetPayload{
							Path:  []string{"nodes", "n" + strconv.Itoa(i), "config"},
							Value: fields(tankCfg("node "+strconv.Itoa(i), "")),
						}),
					)
				}
				actions = append(
					actions,
					schematic.NewSetAction(schematic.SetPayload{
						Path:  []string{"edges", "e1", "config"},
						Value: fields(electricCfg()),
					}),
				)
				out := MustSucceed(schematic.Reduce(schematic.Schematic{}, actions...))
				Expect(out.Nodes).To(HaveLen(5))
				Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 0, Y: 100}))
				Expect(out.Nodes[4].Position).To(Equal(spatial.XY{X: 400, Y: 100}))
				for i, n := range out.Nodes {
					if i < 3 {
						Expect(n.Config).To(Equal(tankCfg("node "+strconv.Itoa(i), "")))
					} else {
						Expect(n.Config.Variant).To(BeNil())
					}
				}
				Expect(out.Edges).To(HaveLen(4))
				Expect(out.Edges[1].Config).To(Equal(electricCfg()))
			},
		)

		It("Should leave state untouched when given an empty action list", func() {
			state := schematic.Schematic{
				Key:   uuid.New(),
				Name:  "empty",
				Nodes: []schematic.Node{node("n1", 0, 0)},
			}
			Expect(MustSucceed(schematic.Reduce(state))).To(Equal(state))
			Expect(MustSucceed(schematic.Reduce(state))).To(Equal(state))
		})
	})

	Describe("Action envelope", func() {
		It("Should error with a zero state when Type is unrecognized", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			Expect(schematic.Reduce(state, schematic.Action{Type: "made_up"})).
				Error().To(SatisfyAll(
				MatchError(union.ErrUnknownVariant),
				MatchError(ContainSubstring("made_up")),
			))
		})

		It("Should reject a set_config action", func() {
			state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
			Expect(schematic.Reduce(state, schematic.Action{Type: "set_config"})).
				Error().To(SatisfyAll(
				MatchError(union.ErrUnknownVariant),
				MatchError(ContainSubstring("set_config")),
			))
		})

		DescribeTable(
			"Should error when Type names a variant whose payload pointer is nil",
			func(actionType string) {
				state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
				Expect(schematic.Reduce(state, schematic.Action{Type: actionType})).
					Error().To(MatchError(union.ErrMissingPayload))
			},
			Entry("set", schematic.ActionTypeSet),
			Entry("set_node_position", schematic.ActionTypeSetNodePosition),
			Entry("set_node", schematic.ActionTypeSetNode),
			Entry("remove_node", schematic.ActionTypeRemoveNode),
			Entry("add_edge", schematic.ActionTypeAddEdge),
			Entry("remove_edge", schematic.ActionTypeRemoveEdge),
		)

		It(
			"Should dispatch on Type and ignore extra payload pointers when more than one is set",
			func() {
				state := schematic.Schematic{Nodes: []schematic.Node{node("n1", 0, 0)}}
				pos := schematic.SetNodePositionPayload{
					Key:      "n1",
					Position: spatial.XY{X: 50, Y: 60},
				}
				edge := schematic.AddEdgePayload{
					Edge: edge("e_extra", "n1", "o", "n2", "i"),
				}
				out := MustSucceed(schematic.Reduce(state, schematic.Action{
					Type:            schematic.ActionTypeSetNodePosition,
					SetNodePosition: &pos,
					AddEdge:         &edge,
				}))
				Expect(out.Nodes[0].Position).To(Equal(spatial.XY{X: 50, Y: 60}))
				Expect(out.Edges).To(BeEmpty())
			},
		)
	})
})

// An older Console dispatches configs without the fields it never knew, so a decoded
// payload fills the schema defaults for those and keeps every value it did carry.
var _ = Describe("Payload decoding", func() {
	decodeConfig := func(config string) schematic.NodeConfig {
		GinkgoHelper()
		var a schematic.Action
		Expect(json.Unmarshal([]byte(`{"type": "set_node", "set_node": {"node":
			{"key": "n1", "position": {"x": 1, "y": 2}, "config": `+config+`}}}`), &a,
		)).To(Succeed())
		Expect(a.SetNode.Node.Position).To(Equal(spatial.XY{X: 1, Y: 2}))
		return a.SetNode.Node.Config
	}

	It("Should fill the defaults a node config omits", func() {
		cfg, ok := decodeConfig(`{"variant": "switch"}`).
			Variant.(schematic.SwitchNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Scale).To(BeEquivalentTo(1))
		Expect(cfg.StalenessTimeout).To(Equal(5 * telem.Second))
		Expect(cfg.Label.Level).To(BeEquivalentTo("h5"))
	})

	It("Should keep a zero the payload carries", func() {
		cfg, ok := decodeConfig(`{"variant": "line", "end": {"x": 0, "y": 40}}`).
			Variant.(schematic.LineNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.End).To(Equal(spatial.XY{X: 0, Y: 40}))
		Expect(cfg.StrokeWidth).To(BeEquivalentTo(2))
	})

	invalidTimeout := MatchError(ContainSubstring("staleness_timeout"))
	DescribeTable("Should validate a span against its bounds",
		func(field string, span int64, matcher OmegaMatcher) {
			cfg := decodeConfig(
				fmt.Sprintf(`{"variant": "valve", %q: %d}`, field, span),
			)
			Expect(cfg.Validate()).To(matcher)
		},
		Entry("staleness timeout at 0", "staleness_timeout", int64(0), invalidTimeout),
		Entry("staleness timeout below 1 s", "staleness_timeout", int64(999999999),
			invalidTimeout),
		Entry("staleness timeout at 1 s", "staleness_timeout", int64(1000000000),
			Succeed()),
		Entry("staleness timeout above 2^53 ns", "staleness_timeout", int64(1)<<53,
			invalidTimeout),
		Entry("click delay at the timer limit", "on_click_delay",
			int64(2147483647000000), Succeed()),
		Entry("click delay above the timer limit", "on_click_delay",
			int64(2147483647000001), MatchError(ContainSubstring("on_click_delay"))),
		Entry("negative click delay", "on_click_delay", int64(-1),
			MatchError(ContainSubstring("on_click_delay"))),
	)

	DescribeTable("Should validate a count against its lower bound",
		func(variant, field string, count int, matcher OmegaMatcher) {
			cfg := decodeConfig(
				fmt.Sprintf(`{"variant": %q, %q: %d}`, variant, field, count),
			)
			Expect(cfg.Validate()).To(matcher)
		},
		Entry("polygon sides below 3", "polygon", "num_sides", 2,
			MatchError(ContainSubstring("num_sides"))),
		Entry("polygon sides at 3", "polygon", "num_sides", 3, Succeed()),
		Entry("value rolling average at 0", "value", "rolling_average", 0,
			MatchError(ContainSubstring("rolling_average"))),
		Entry("value rolling average at 1", "value", "rolling_average", 1, Succeed()),
		Entry("tank rolling average at 0", "tank", "rolling_average", 0,
			MatchError(ContainSubstring("rolling_average"))),
		Entry("tank rolling average at 1", "tank", "rolling_average", 1, Succeed()),
		Entry("value inline size at 0", "value", "inline_size", 0,
			MatchError(ContainSubstring("inline_size"))),
		Entry("value inline size at 1", "value", "inline_size", 1, Succeed()),
		Entry("input inline size at 0", "input", "inline_size", 0,
			MatchError(ContainSubstring("inline_size"))),
		Entry("text box inline size at 0", "text_box", "inline_size", 0,
			MatchError(ContainSubstring("inline_size"))),
		Entry("circle radius at 0", "circle", "radius", 0,
			MatchError(ContainSubstring("radius"))),
		Entry("circle radius at 1", "circle", "radius", 1, Succeed()),
		Entry("polygon side length at 0", "polygon", "side_length", 0,
			MatchError(ContainSubstring("side_length"))),
		Entry("gauge bar width at 0", "gauge", "bar_width", 0,
			MatchError(ContainSubstring("bar_width"))),
		Entry("gauge bar width at 1", "gauge", "bar_width", 1, Succeed()),
	)

	DescribeTable("Should validate a scale above 0",
		func(variant string, scale float64, matcher OmegaMatcher) {
			cfg := decodeConfig(
				fmt.Sprintf(`{"variant": %q, "scale": %g}`, variant, scale),
			)
			Expect(cfg.Validate()).To(matcher)
		},
		Entry("valve at 0", "valve", 0.0, MatchError(ContainSubstring("scale"))),
		Entry("light below 0", "light", -1.0, MatchError(ContainSubstring("scale"))),
		Entry("cylinder above 0", "cylinder", 0.5, Succeed()),
	)

	DescribeTable("Should validate a tank staleness timeout against its bounds",
		func(span int64, matcher OmegaMatcher) {
			cfg := decodeConfig(
				fmt.Sprintf(`{"variant": "tank", "staleness_timeout": %d}`, span),
			)
			Expect(cfg.Validate()).To(matcher)
		},
		Entry("below 1 s", int64(999999999), invalidTimeout),
		Entry("at 1 s", int64(1000000000), Succeed()),
	)

	It("Should fill the wrap width a payload omits", func() {
		cfg, ok := decodeConfig(`{"variant": "button"}`).Variant.(schematic.ButtonNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Label.MaxInlineSize).To(BeEquivalentTo(150))
	})

	DescribeTable("Should validate a wrap width against its bounds",
		func(width int, matcher OmegaMatcher) {
			cfg := decodeConfig(fmt.Sprintf(`{"variant": "button", "label": {
				"label": "P1", "max_inline_size": %d}}`, width))
			Expect(cfg.Validate()).To(matcher)
		},
		Entry("at 0", 0, MatchError(ContainSubstring("max_inline_size"))),
		Entry("at 1", 1, Succeed()),
	)

	It("Should keep an empty string the payload carries", func() {
		cfg, ok := decodeConfig(`{"variant": "value", "units": ""}`).
			Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Units).To(Equal(""))
	})

	It("Should keep a zero precision", func() {
		cfg, ok := decodeConfig(`{"variant": "value", "precision": 0}`).
			Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Precision).To(HaveValue(BeZero()))
	})

	It("Should leave an omitted shape stroke width absent", func() {
		cfg, ok := decodeConfig(`{"variant": "circle"}`).
			Variant.(schematic.CircleNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.StrokeWidth).To(BeNil())
	})

	It("Should give an omitted text box value the empty string", func() {
		cfg, ok := decodeConfig(`{"variant": "text_box"}`).
			Variant.(schematic.TextBoxNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Value).To(BeEmpty())
	})

	It("Should leave an omitted precision absent", func() {
		cfg, ok := decodeConfig(`{"variant": "value"}`).Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Precision).To(BeNil())
	})

	It("Should leave the authority of an omitted control state absent", func() {
		cfg, ok := decodeConfig(`{"variant": "button"}`).
			Variant.(schematic.ButtonNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Control.Authority).To(BeNil())
		Expect(cfg.CommandChannel).To(BeZero())
	})

	DescribeTable("Should reject a precision that is not a byte", func(p string) {
		var a schematic.Action
		Expect(json.Unmarshal([]byte(`{"type": "set_node", "set_node": {"node":
			{"key": "v1", "config": {"variant": "value", "precision": `+p+`}}}}`), &a)).
			To(MatchError(ContainSubstring("precision")))
	},
		Entry("fraction", "2.6"),
		Entry("negative", "-1"),
		Entry("above the byte range", "300"),
	)

	It("Should reject a precision above 17", func() {
		cfg := decodeConfig(`{"variant": "value", "precision": 18}`)
		Expect(cfg.Validate()).To(MatchError(ContainSubstring("precision")))
	})

	It("Should reject a config naming an unknown variant", func() {
		var a schematic.Action
		Expect(json.Unmarshal([]byte(`{"type": "set_node", "set_node": {"node":
			{"key": "n1", "config": {"scale": 2}}}}`), &a)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`NodeConfig: unknown variant ""`)),
		))
	})
})

var _ = Describe("Path actions", func() {
	var state schematic.Schematic
	BeforeEach(func() {
		state = schematic.Schematic{
			Name: "Engine",
			Nodes: []schematic.Node{
				configured(node("n1", 0, 0), tankCfg("T1", "")),
				configured(node("n2", 1, 1), tankCfg("T2", "")),
			},
			Edges: []schematic.Edge{pipe("e1", "n1", "out", "n2", "in", "")},
		}
	})
	set := func(path []string, value any, guards ...patch.Guard) schematic.Action {
		return schematic.NewSetAction(
			schematic.SetPayload{Path: path, Value: value, Guards: guards},
		)
	}
	tank := func(s schematic.Schematic) schematic.TankNodeConfig {
		GinkgoHelper()
		cfg, ok := s.Nodes[0].Config.Variant.(schematic.TankNodeConfig)
		Expect(ok).To(BeTrue())
		return cfg
	}

	Describe("SetPayload", func() {
		It("Should set a field of a node config", func() {
			next := MustSucceed(schematic.Reduce(
				state,
				set([]string{"nodes", "n1", "config", "label", "label"}, "T3"),
			))
			Expect(tank(next).Label.Label).To(Equal("T3"))
			Expect(next.Nodes[1]).To(Equal(state.Nodes[1]))
			Expect(next.Name).To(Equal("Engine"))
		})

		It("Should set a boolean field", func() {
			next := MustSucceed(schematic.Reduce(
				state,
				set([]string{"nodes", "n1", "config", "scale_visible"}, true),
			))
			Expect(tank(next).ScaleVisible).To(BeTrue())
		})

		It("Should set a fixed color from its hex form", func() {
			next := MustSucceed(schematic.Reduce(
				state,
				set(
					[]string{"nodes", "n1", "config", "stroke_color"},
					map[string]any{"type": "fixed", "value": "#ff0000"},
				),
			))
			Expect(tank(next).StrokeColor).To(Equal(fixedColor("#ff0000")))
		})

		It("Should set a field of an edge config", func() {
			next := MustSucceed(schematic.Reduce(
				state,
				set(
					[]string{"edges", "e1", "config", "stroke_color"},
					map[string]any{"type": "fixed", "value": "#00ff00"},
				),
			))
			Expect(next.Edges).To(Equal([]schematic.Edge{
				pipe("e1", "n1", "out", "n2", "in", "#00ff00"),
			}))
		})

		It("Should replace a config whole with one of another variant", func() {
			next := MustSucceed(schematic.Reduce(
				state,
				set([]string{"edges", "e1", "config"}, fields(electricCfg())),
			))
			Expect(next.Edges[0].Config).To(Equal(electricCfg()))
		})

		It("Should do nothing when the variant guard fails", func() {
			next := MustSucceed(schematic.Reduce(state, set(
				[]string{"nodes", "n1", "config", "label", "label"},
				"T3",
				patch.Guard{
					Path:  []string{"nodes", "n1", "config", "variant"},
					Value: "valve",
				},
			)))
			Expect(tank(next).Label.Label).To(Equal("T1"))
		})

		It("Should set the name", func() {
			next := MustSucceed(schematic.Reduce(state, set([]string{"name"}, "Other")))
			Expect(next.Name).To(Equal("Other"))
			Expect(next.Nodes).To(Equal(state.Nodes))
		})

		It("Should reject a name that is not a string", func() {
			Expect(schematic.Reduce(state, set([]string{"name"}, 1))).
				Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[name]: decoding value")),
			))
		})

		It("Should reject a path to a field the schematic does not have", func() {
			Expect(schematic.Reduce(state, set([]string{"configs", "n1"}, nil))).
				Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("[configs]: the path names no editable field"),
				),
			))
		})

		It("Should set the parent of a node", func() {
			next := MustSucceed(schematic.Reduce(
				state,
				set([]string{"nodes", "n1", "parent"}, "g1"),
			))
			Expect(next.Nodes[0].Parent).To(HaveValue(Equal("g1")))
		})

		It("Should remove the parent of a node", func() {
			grouped := MustSucceed(schematic.Reduce(
				state,
				set([]string{"nodes", "n1", "parent"}, "g1"),
			))
			next := MustSucceed(schematic.Reduce(
				grouped,
				schematic.NewRemoveAction(schematic.RemovePayload{
					Path: []string{"nodes", "n1"},
					Key:  "parent",
				}),
			))
			Expect(next.Nodes[0].Parent).To(BeNil())
		})

		It("Should reject a field the config does not declare", func() {
			Expect(schematic.Reduce(
				state,
				set([]string{"nodes", "n1", "config", "tags"}, []any{}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"[nodes, n1, config, tags]: the path names no editable field",
				)),
			))
		})
	})

	DescribeTable(
		"Should reject a path into a node or edge outside its config",
		func(a schematic.Action, msg string) {
			Expect(schematic.Reduce(state, a)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry(
			"a set of every node",
			set([]string{"nodes"}, []any{}),
			"[nodes]: the path must go into an editable field of one entry",
		),
		Entry(
			"a set of one node",
			set([]string{"nodes", "n1"}, map[string]any{"key": "n1"}),
			"[nodes, n1]: the path must go into an editable field of one entry",
		),
		Entry(
			"a set of a node's position",
			set([]string{"nodes", "n1", "position"}, map[string]any{"x": 1, "y": 1}),
			"[nodes, n1, position]: the path names no editable field",
		),
		Entry(
			"a set of a node's key",
			set([]string{"nodes", "n1", "key"}, "n9"),
			"[nodes, n1, key]: the path names no editable field",
		),
		Entry(
			"a set of an edge's source",
			set(
				[]string{"edges", "e1", "source"},
				map[string]any{"node": "n2", "param": "out"},
			),
			"[edges, e1, source]: the path names no editable field",
		),
		Entry(
			"a set of every edge",
			set([]string{"edges"}, []any{}),
			"[edges]: the path must go into an editable field of one entry",
		),
		Entry(
			"a set of an edge's parent",
			set([]string{"edges", "e1", "parent"}, "g1"),
			"[edges, e1, parent]: the path names no editable field",
		),
		Entry(
			"a remove of a node",
			schematic.NewRemoveAction(schematic.RemovePayload{
				Path: []string{"nodes"},
				Key:  "n1",
			}),
			"[nodes, n1]: the path must go into an editable field of one entry",
		),
		Entry(
			"an insert of a node",
			schematic.NewInsertAction(schematic.InsertPayload{
				Path:  []string{"nodes"},
				Value: map[string]any{"key": "n3"},
			}),
			"[nodes]: the path must go into an editable field of one entry",
		),
		Entry(
			"a move of an edge",
			schematic.NewMoveAction(schematic.MovePayload{
				Path: []string{"edges"},
				Key:  "e1",
			}),
			"[edges]: the path must go into an editable field of one entry",
		),
	)

	Describe("RemovePayload", func() {
		It("Should remove an optional field of a config", func() {
			state.Nodes[0] = configured(node("n1", 0, 0), tankCfg("T1", "#ff0000"))
			next := MustSucceed(schematic.Reduce(state, schematic.NewRemoveAction(
				schematic.RemovePayload{
					Path: []string{"nodes", "n1", "config"},
					Key:  "stroke_color",
				},
			)))
			Expect(tank(next).StrokeColor).To(BeNil())
		})

		It("Should reject a remove of the name", func() {
			Expect(schematic.Reduce(state, schematic.NewRemoveAction(
				schematic.RemovePayload{Key: "name"},
			))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[name]: the document requires a value")),
			))
		})
	})

	DescribeTable(
		"Should do nothing to a node or edge that does not exist",
		func(a schematic.Action) {
			Expect(MustSucceed(schematic.Reduce(state, a))).To(Equal(state))
		},
		Entry(
			"a set of a field",
			set([]string{"nodes", "ghost", "config", "label", "label"}, "T2"),
		),
		Entry(
			"a set of the whole config",
			set(
				[]string{"nodes", "ghost", "config"},
				map[string]any{"variant": "tank"},
			),
		),
		Entry(
			"a set of an edge config",
			set(
				[]string{"edges", "ghost", "config"},
				map[string]any{"variant": "pipe"},
			),
		),
		Entry(
			"a remove of a field",
			schematic.NewRemoveAction(schematic.RemovePayload{
				Path: []string{"nodes", "ghost", "config", "label"},
				Key:  "label",
			}),
		),
		Entry(
			"an insert",
			schematic.NewInsertAction(schematic.InsertPayload{
				Path:  []string{"nodes", "ghost", "config", "segments"},
				Value: map[string]any{"key": "a"},
			}),
		),
		Entry(
			"a move",
			schematic.NewMoveAction(schematic.MovePayload{
				Path: []string{"edges", "ghost", "config", "segments"},
				Key:  "a",
			}),
		),
		Entry(
			"a set of a field that is not editable",
			set([]string{"nodes", "ghost", "position"}, map[string]any{"x": 1, "y": 1}),
		),
		Entry(
			"a set of an edge field that is not editable",
			set(
				[]string{"edges", "ghost", "source"},
				map[string]any{"node": "n2", "param": "out"},
			),
		),
	)
})

var _ = Describe("Path actions on defaulted fields", func() {
	symbol := func(variant string) schematic.Schematic {
		GinkgoHelper()
		var cfg schematic.NodeConfig
		Expect(json.Unmarshal([]byte(`{"variant": "`+variant+`"}`), &cfg)).
			To(Succeed())
		return schematic.Schematic{
			Name:  "Engine",
			Nodes: []schematic.Node{configured(node("n1", 0, 0), cfg)},
		}
	}
	configPath := []string{"nodes", "n1", "config"}
	set := func(path []string, value any) schematic.Action {
		return schematic.NewSetAction(schematic.SetPayload{
			Path:  slices.Concat(configPath, path),
			Value: value,
		})
	}
	remove := func(key string) schematic.Action {
		return schematic.NewRemoveAction(
			schematic.RemovePayload{Path: configPath, Key: key},
		)
	}
	DescribeTable("Should reject emptying a field that has a schema default",
		func(variant string, a schematic.Action) {
			Expect(schematic.Reduce(symbol(variant), a)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("the document requires a value")),
			))
		},
		Entry("a removed page", "off_page_reference", remove("page")),
		Entry("a page set to no value", "off_page_reference",
			set([]string{"page"}, nil)),
		Entry("a removed control state", "button", remove("control")),
	)

	DescribeTable("Should clear an optional field to absent",
		func(
			variant, key string,
			value any,
			clearAction schematic.Action,
			field string,
		) {
			state := MustSucceed(
				schematic.Reduce(symbol(variant), set([]string{key}, value)),
			)
			Expect(state.Nodes[0].Config.Variant).To(HaveField(field, Not(BeNil())))
			next := MustSucceed(schematic.Reduce(state, clearAction))
			Expect(next.Nodes[0].Config.Variant).To(HaveField(field, BeNil()))
		},
		Entry("a removed precision", "value", "precision", 5, remove("precision"),
			"Precision"),
		Entry("a precision set to no value", "value", "precision", 5,
			set([]string{"precision"}, nil), "Precision"),
		Entry("a removed light threshold", "light", "threshold",
			map[string]any{"lower": 1, "upper": 2}, remove("threshold"), "Threshold"),
		Entry("a removed cylinder border radius", "cylinder", "border_radius", 4,
			remove("border_radius"), "BorderRadius"),
		Entry("a removed tank border radius", "tank", "border_radius", 4,
			remove("border_radius"), "BorderRadius"),
		Entry("a removed box border radius", "box", "border_radius", 5,
			remove("border_radius"), "BorderRadius"),
		Entry("a removed box stroke width", "box", "stroke_width", 0,
			remove("stroke_width"), "StrokeWidth"),
		Entry("a polygon stroke width set to no value", "polygon", "stroke_width", 4,
			set([]string{"stroke_width"}, nil), "StrokeWidth"),
		Entry("a removed control authority", "button", "control",
			map[string]any{
				"authority":        5,
				"hidden":           false,
				"chip_hidden":      false,
				"indicator_hidden": false,
				"orientation":      "bottom",
			},
			schematic.NewRemoveAction(schematic.RemovePayload{
				Path: slices.Concat(configPath, []string{"control"}),
				Key:  "authority",
			}),
			"Control.Authority"),
	)

	DescribeTable("Should set a field inside a struct the symbol did not store",
		func(variant string, path []string, value any, want OmegaMatcher) {
			next := MustSucceed(schematic.Reduce(symbol(variant), set(path, value)))
			Expect(next.Nodes[0].Config.Variant).To(want)
		},
		Entry("a control state field", "button", []string{"control", "hidden"}, true,
			SatisfyAll(
				HaveField("Control.Hidden", BeTrue()),
				HaveField("Control.Authority", BeNil()),
			)),
		Entry("a position field", "value", []string{"position", "x"}, 3,
			SatisfyAll(
				HaveField("Position.X", BeEquivalentTo(3)),
				HaveField("Position.Y", BeEquivalentTo(0)),
			)),
	)
})
