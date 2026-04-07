// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("DecodeMsgpack", func() {
	Describe("Node", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := graph.Node{
				Key:      "node1",
				Type:     "fn1",
				Position: spatial.XY{X: 100, Y: 200},
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded graph.Node
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("node1"))
			Expect(decoded.Type).To(Equal("fn1"))
			Expect(decoded.Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Key      string
				Type     string
				Config   xmsgpack.EncodedJSON
				Position spatial.XY
			}{
				Key:      "node1",
				Type:     "fn1",
				Position: spatial.XY{X: 50, Y: 75},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded graph.Node
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("node1"))
			Expect(decoded.Type).To(Equal("fn1"))
			Expect(decoded.Position).To(Equal(spatial.XY{X: 50, Y: 75}))
		})
	})

	Describe("Viewport", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := graph.Viewport{
				Position: spatial.XY{X: 10, Y: 20},
				Zoom:     1.5,
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded graph.Viewport
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Position).To(Equal(spatial.XY{X: 10, Y: 20}))
			Expect(decoded.Zoom).To(Equal(1.5))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Position spatial.XY
				Zoom     float64
			}{
				Position: spatial.XY{X: 5, Y: 10},
				Zoom:     2.0,
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded graph.Viewport
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Position).To(Equal(spatial.XY{X: 5, Y: 10}))
			Expect(decoded.Zoom).To(Equal(2.0))
		})
	})

	Describe("Graph", func() {
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Viewport  graph.Viewport
				Functions ir.Functions
				Edges     ir.Edges
				Nodes     graph.Nodes
			}{
				Viewport: graph.Viewport{Zoom: 1.0},
				Nodes:    graph.Nodes{{Key: "n1", Type: "fn1"}},
				Edges:    ir.Edges{{Source: ir.Handle{Node: "n1", Param: "out"}}},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded graph.Graph
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Nodes).To(HaveLen(1))
			Expect(decoded.Nodes[0].Key).To(Equal("n1"))
			Expect(decoded.Edges).To(HaveLen(1))
			Expect(decoded.Viewport.Zoom).To(Equal(1.0))
		})
	})
})
