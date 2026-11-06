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

func textBase(t text.Text) arc.Base {
	return arc.Base{Source: arc.Source{Variant: arc.TextSource{Text: t}}}
}

func textArc(raw string) arc.Base { return textBase(newText(raw)) }

func graphBase(g graph.Graph) arc.Base {
	return arc.Base{Source: arc.Source{Variant: arc.GraphSource{Graph: g}}}
}

func testGraph() graph.Graph {
	return graph.Graph{
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
	}
}

func graphArc() arc.Base { return graphBase(testGraph()) }

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

		It("Should hash a pre-materialized arc identically", func() {
			a := textArc("a -> b")
			materialized := textBase(newText("a -> b").Materialize())
			Expect(MustSucceed(arc.Hash(materialized))).
				To(Equal(MustSucceed(arc.Hash(a))))
		})

		It("Should ignore a stale Raw and hash the document", func() {
			t := newText("a -> b")
			t.Raw = "a -> c"
			Expect(MustSucceed(arc.Hash(textBase(t)))).
				To(Equal(MustSucceed(arc.Hash(textArc("a -> b")))))
		})
	})

	Describe("Graph mode", func() {
		It("Should hash equal graphs equally", func() {
			Expect(MustSucceed(arc.Hash(graphArc()))).
				To(Equal(MustSucceed(arc.Hash(graphArc()))))
		})

		It("Should ignore node positions", func() {
			moved := testGraph()
			moved.Nodes[0].Position = spatial.XY{X: 500, Y: 500}
			Expect(MustSucceed(arc.Hash(graphArc()))).
				To(Equal(MustSucceed(arc.Hash(graphBase(moved)))))
		})

		It("Should ignore edge identity", func() {
			rekeyed := testGraph()
			rekeyed.Edges[0].Key = "e2"
			Expect(MustSucceed(arc.Hash(graphArc()))).
				To(Equal(MustSucceed(arc.Hash(graphBase(rekeyed)))))
		})

		It("Should ignore node and edge ordering", func() {
			reordered := testGraph()
			reordered.Nodes[0], reordered.Nodes[1] = reordered.Nodes[1], reordered.Nodes[0]
			reordered.Edges = append(graph.Edges{{
				Key:    "e3",
				Source: ir.Handle{Node: "b", Param: "out"},
				Target: ir.Handle{Node: "a", Param: "in"},
			}}, reordered.Edges...)
			base := testGraph()
			base.Edges = append(base.Edges, graph.Edge{
				Key:    "e4",
				Source: ir.Handle{Node: "b", Param: "out"},
				Target: ir.Handle{Node: "a", Param: "in"},
			})
			Expect(MustSucceed(arc.Hash(graphBase(base)))).
				To(Equal(MustSucceed(arc.Hash(graphBase(reordered)))))
		})

		It("Should change when an edge is reconnected", func() {
			reconnected := testGraph()
			reconnected.Edges[0].Target = ir.Handle{Node: "b", Param: "other"}
			Expect(MustSucceed(arc.Hash(graphArc()))).
				ToNot(Equal(MustSucceed(arc.Hash(graphBase(reconnected)))))
		})

		It("Should change when node inputs change", func() {
			changed := testGraph()
			changed.Inputs["a"] = msgpack.EncodedJSON{"type": "on", "channel": 9}
			Expect(MustSucceed(arc.Hash(graphArc()))).
				ToNot(Equal(MustSucceed(arc.Hash(graphBase(changed)))))
		})

		It("Should change when a node is added", func() {
			grown := testGraph()
			grown.Nodes = append(grown.Nodes, graph.Node{Key: "c"})
			Expect(MustSucceed(arc.Hash(graphArc()))).
				ToNot(Equal(MustSucceed(arc.Hash(graphBase(grown)))))
		})
	})

	It("Should hash the two modes differently", func() {
		a := textBase(text.Text{})
		b := graphBase(graph.Graph{})
		Expect(MustSucceed(arc.Hash(a))).ToNot(Equal(MustSucceed(arc.Hash(b))))
	})
})
