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
				Position: spatial.XY{X: 100, Y: 200},
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded graph.Node
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("node1"))
			Expect(decoded.Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})
		It("Should decode legacy uppercase Go field names, dropping inline type and config", func() {
			legacy := struct {
				Key      string
				Type     string
				Config   xmsgpack.EncodedJSON
				Position spatial.XY
			}{
				Key:      "node1",
				Type:     "fn1",
				Config:   xmsgpack.EncodedJSON{"gain": 1},
				Position: spatial.XY{X: 50, Y: 75},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded graph.Node
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("node1"))
			Expect(decoded.Position).To(Equal(spatial.XY{X: 50, Y: 75}))
		})
	})

	Describe("Graph", func() {
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Functions ir.Functions
				Edges     ir.Edges
				Nodes     graph.Nodes
			}{
				Nodes: graph.Nodes{{Key: "n1"}},
				Edges: ir.Edges{{Source: ir.Handle{Node: "n1", Param: "out"}}},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded graph.Graph
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Nodes).To(HaveLen(1))
			Expect(decoded.Nodes[0].Key).To(Equal("n1"))
			Expect(decoded.Edges).To(HaveLen(1))
		})

		It("Should lift legacy inline node type and config into the configs map", func() {
			legacy := struct {
				Nodes []struct {
					Key      string               `msgpack:"key"`
					Type     string               `msgpack:"type"`
					Config   xmsgpack.EncodedJSON `msgpack:"config"`
					Position spatial.XY           `msgpack:"position"`
				} `msgpack:"nodes"`
			}{
				Nodes: []struct {
					Key      string               `msgpack:"key"`
					Type     string               `msgpack:"type"`
					Config   xmsgpack.EncodedJSON `msgpack:"config"`
					Position spatial.XY           `msgpack:"position"`
				}{
					{Key: "n1", Type: "on", Config: xmsgpack.EncodedJSON{"channel": int8(12)}},
					{Key: "n2", Type: "printer"},
				},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded graph.Graph
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Nodes).To(HaveLen(2))
			Expect(decoded.Configs["n1"]).To(SatisfyAll(
				HaveKeyWithValue("type", "on"),
				HaveKeyWithValue("channel", int8(12)),
			))
			Expect(decoded.Configs["n2"]).To(HaveKeyWithValue("type", "printer"))
		})
	})
})
