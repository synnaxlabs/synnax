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
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

func newText(raw string) text.Text { return text.Text{Doc: text.Create(raw)} }

func textArc(raw string) arc.Arc {
	return arc.Arc{Mode: arc.ModeText, Text: newText(raw)}
}

func graphArc() arc.Arc {
	return arc.Arc{
		Mode: arc.ModeGraph,
		Graph: graph.Graph{
			Nodes: graph.Nodes{
				{Key: "a", Position: spatial.XY{X: 0, Y: 0}},
				{Key: "b", Position: spatial.XY{X: 10, Y: 10}},
			},
			Edges: graph.Edges{{
				Key:    "e1",
				Source: ir.Handle{Node: "a", Param: "out"},
				Target: ir.Handle{Node: "b", Param: "in"},
			}},
			Inputs: map[string]msgpack.EncodedJSON{
				"a": {"type": "on", "channel": 1},
				"b": {"type": "set", "channel": 2},
			},
		},
	}
}

var _ = Describe("Hash", func() {
	Describe("Text mode", func() {
		It("Should hash equal sources equally", func() {
			Expect(MustSucceed(arc.Hash(textArc("a -> b")))).
				To(Equal(MustSucceed(arc.Hash(textArc("a -> b")))))
		})

		It("Should hash different sources differently", func() {
			Expect(MustSucceed(arc.Hash(textArc("a -> b")))).
				ToNot(Equal(MustSucceed(arc.Hash(textArc("a -> c")))))
		})

		It("Should ignore graph content", func() {
			a := textArc("a -> b")
			b := textArc("a -> b")
			b.Graph = graphArc().Graph
			Expect(MustSucceed(arc.Hash(a))).To(Equal(MustSucceed(arc.Hash(b))))
		})

		It("Should hash a pre-materialized arc identically", func() {
			a := textArc("a -> b")
			materialized := a
			materialized.Text = materialized.Text.Materialize()
			Expect(MustSucceed(arc.Hash(materialized))).
				To(Equal(MustSucceed(arc.Hash(a))))
		})

		It("Should ignore a stale Raw and hash the document", func() {
			a := textArc("a -> b")
			a.Text.Raw = "a -> c"
			Expect(MustSucceed(arc.Hash(a))).
				To(Equal(MustSucceed(arc.Hash(textArc("a -> b")))))
		})
	})

	Describe("Graph mode", func() {
		It("Should hash equal graphs equally", func() {
			Expect(MustSucceed(arc.Hash(graphArc()))).
				To(Equal(MustSucceed(arc.Hash(graphArc()))))
		})

		It("Should ignore node positions", func() {
			moved := graphArc()
			moved.Graph.Nodes[0].Position = spatial.XY{X: 500, Y: 500}
			Expect(MustSucceed(arc.Hash(graphArc()))).
				To(Equal(MustSucceed(arc.Hash(moved))))
		})

		It("Should ignore edge identity", func() {
			rekeyed := graphArc()
			rekeyed.Graph.Edges[0].Key = "e2"
			Expect(MustSucceed(arc.Hash(graphArc()))).
				To(Equal(MustSucceed(arc.Hash(rekeyed))))
		})

		It("Should ignore node and edge ordering", func() {
			reordered := graphArc()
			reordered.Graph.Nodes[0], reordered.Graph.Nodes[1] = reordered.Graph.Nodes[1], reordered.Graph.Nodes[0]
			reordered.Graph.Edges = append(graph.Edges{{
				Key:    "e3",
				Source: ir.Handle{Node: "b", Param: "out"},
				Target: ir.Handle{Node: "a", Param: "in"},
			}}, reordered.Graph.Edges...)
			base := graphArc()
			base.Graph.Edges = append(base.Graph.Edges, graph.Edge{
				Key:    "e4",
				Source: ir.Handle{Node: "b", Param: "out"},
				Target: ir.Handle{Node: "a", Param: "in"},
			})
			Expect(MustSucceed(arc.Hash(base))).
				To(Equal(MustSucceed(arc.Hash(reordered))))
		})

		It("Should change when an edge is reconnected", func() {
			reconnected := graphArc()
			reconnected.Graph.Edges[0].Target = ir.Handle{Node: "b", Param: "other"}
			Expect(MustSucceed(arc.Hash(graphArc()))).
				ToNot(Equal(MustSucceed(arc.Hash(reconnected))))
		})

		It("Should change when node inputs change", func() {
			changed := graphArc()
			changed.Graph.Inputs["a"] = msgpack.EncodedJSON{"type": "on", "channel": 9}
			Expect(MustSucceed(arc.Hash(graphArc()))).
				ToNot(Equal(MustSucceed(arc.Hash(changed))))
		})

		It("Should change when a node is added", func() {
			grown := graphArc()
			grown.Graph.Nodes = append(grown.Graph.Nodes, graph.Node{Key: "c"})
			Expect(MustSucceed(arc.Hash(graphArc()))).
				ToNot(Equal(MustSucceed(arc.Hash(grown))))
		})
	})

	It("Should hash the two modes differently", func() {
		a := arc.Arc{Mode: arc.ModeText}
		b := arc.Arc{Mode: arc.ModeGraph}
		Expect(MustSucceed(arc.Hash(a))).ToNot(Equal(MustSucceed(arc.Hash(b))))
	})

	// A v0.58.2 Core stamped these hashes into the tasks of rack-bound Arcs. A hash
	// that drifts from them rewrites every such task on its first edit, even a layout
	// move.
	Describe("v0.58.2 stamps", func() {
		It("Should match the hash a v0.58.2 Core stamped for a Console graph", func() {
			a := arc.Arc{
				Mode: arc.ModeGraph,
				Graph: graph.Graph{
					Functions: ir.Functions{},
					Nodes: graph.Nodes{
						{Key: "n_on", Position: spatial.XY{X: 10, Y: 20}},
						{Key: "n_wr", Position: spatial.XY{X: 200, Y: 20}},
					},
					Edges: graph.Edges{{
						Key:    "e1",
						Source: ir.Handle{Node: "n_on", Param: "output"},
						Target: ir.Handle{Node: "n_wr", Param: "input"},
						Kind:   ir.EdgeKindContinuous,
					}},
					Inputs: map[string]msgpack.EncodedJSON{
						"n_on": {"channel": 1048584.0, "type": "on"},
						"n_wr": {"channel": 1048584.0, "type": "write"},
					},
				},
			}
			Expect(MustSucceed(arc.Hash(a))).To(Equal("beeffcc46faf0fbd"))
		})

		It("Should match the hash a v0.58.2 Core stamped for a graph with functions",
			func() {
				scalar := func(k types.Kind) types.Type {
					return types.Type{
						Inputs:  types.Params{},
						Outputs: types.Params{},
						Kind:    k,
					}
				}
				a := arc.Arc{
					Mode: arc.ModeGraph,
					Graph: graph.Graph{
						Functions: ir.Functions{{
							Key: "above",
							Body: ir.Body{
								Raw: "func above(x f32) u8 {\n  return x > 2\n}\n",
							},
							Inputs: types.Params{
								{Name: "x", Type: scalar(types.KindF32)},
							},
							Outputs: types.Params{
								{Name: "output", Type: scalar(types.KindU8)},
							},
							Channels: types.Channels{
								Read:  map[uint32]string{1048584: "x"},
								Write: map[uint32]string{},
							},
						}},
						Nodes: graph.Nodes{
							{Key: "n_on", Position: spatial.XY{X: 10, Y: 20}},
							{Key: "n_f", Position: spatial.XY{X: 100, Y: 20}},
							{Key: "n_wr", Position: spatial.XY{X: 200, Y: 20}},
						},
						Edges: graph.Edges{
							{
								Key:    "e1",
								Source: ir.Handle{Node: "n_on", Param: "output"},
								Target: ir.Handle{Node: "n_f", Param: "x"},
								Kind:   ir.EdgeKindContinuous,
							},
							{
								Key:    "e2",
								Source: ir.Handle{Node: "n_f", Param: "output"},
								Target: ir.Handle{Node: "n_wr", Param: "input"},
								Kind:   ir.EdgeKindConditional,
							},
						},
						Inputs: map[string]msgpack.EncodedJSON{
							"n_on": {"channel": 1048584.0, "type": "on"},
							"n_f":  {"type": "above"},
							"n_wr": {"channel": 1048584.0, "type": "write"},
						},
					},
				}
				Expect(MustSucceed(arc.Hash(a))).To(Equal("f8ec1fc1c12a87e4"))
			})

		It("Should match the hash a v0.58.2 Core stamped for a text Arc", func() {
			Expect(MustSucceed(arc.Hash(textArc(
				"func f(x f32) u8 {\n  return x < 5 && x > 2\n}\n",
			)))).To(Equal("64624b63a2e74829"))
		})
	})
})
