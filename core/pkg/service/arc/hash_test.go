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
				Key: "e1",
				Edge: ir.Edge{
					Source: ir.Handle{Node: "a", Param: "out"},
					Target: ir.Handle{Node: "b", Param: "in"},
				},
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

		It("Should trust a non-empty Raw over the document", func() {
			a := textArc("a -> b")
			a.Text.Raw = "a -> c"
			Expect(MustSucceed(arc.Hash(a))).
				To(Equal(MustSucceed(arc.Hash(textArc("a -> c")))))
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
				Key: "e3",
				Edge: ir.Edge{
					Source: ir.Handle{Node: "b", Param: "out"},
					Target: ir.Handle{Node: "a", Param: "in"},
				},
			}}, reordered.Graph.Edges...)
			base := graphArc()
			base.Graph.Edges = append(base.Graph.Edges, graph.Edge{
				Key: "e4",
				Edge: ir.Edge{
					Source: ir.Handle{Node: "b", Param: "out"},
					Target: ir.Handle{Node: "a", Param: "in"},
				},
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
})
