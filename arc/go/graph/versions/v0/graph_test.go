// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/graph/versions/v0"
	ir "github.com/synnaxlabs/arc/ir/versions/v0"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

// legacyXY is a point under the Go field names Cores before v0.54 stored.
type legacyXY struct{ X, Y float64 }

type legacyNode struct {
	Key      string
	Type     string
	Config   xmsgpack.EncodedJSON
	Position legacyXY
}

type legacyViewport struct {
	Position legacyXY
	Zoom     float64
}

var _ = Describe("Graph", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode a graph stored under lowercase names", func() {
			g := v0.Graph{
				Viewport: v0.Viewport{Position: spatial.XY{X: 1, Y: 2}, Zoom: 1.5},
				Nodes: v0.Nodes{{
					Key:      "n1",
					Type:     "on",
					Position: spatial.XY{X: 3, Y: 4},
				}},
				Edges: ir.Edges{{Source: ir.Handle{Node: "n1", Param: "output"}}},
			}
			var decoded v0.Graph
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(g)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(g))
		})

		It("Should keep the viewport of a graph stored with no nodes", func() {
			g := v0.Graph{
				Viewport: v0.Viewport{Position: spatial.XY{X: 5, Y: 6}, Zoom: 2},
			}
			var decoded v0.Graph
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(g)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(g))
		})

		It("Should decode a graph stored under uppercase Go field names", func() {
			legacy := struct {
				Viewport legacyViewport
				Edges    []struct{ Source, Target struct{ Node, Param string } }
				Nodes    []legacyNode
			}{
				Viewport: legacyViewport{Position: legacyXY{X: 5, Y: 6}, Zoom: 2},
				Edges: []struct{ Source, Target struct{ Node, Param string } }{{
					Source: struct{ Node, Param string }{"n1", "output"},
					Target: struct{ Node, Param string }{"n2", "input"},
				}},
				Nodes: []legacyNode{{
					Key:      "n1",
					Type:     "on",
					Config:   xmsgpack.EncodedJSON{"channel": 12.0},
					Position: legacyXY{X: 10, Y: 20},
				}},
			}
			var decoded v0.Graph
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(legacy)), &decoded)).
				To(Succeed())
			Expect(decoded.Viewport).To(Equal(
				v0.Viewport{Position: spatial.XY{X: 5, Y: 6}, Zoom: 2},
			))
			Expect(decoded.Nodes).To(Equal(v0.Nodes{{
				Key:      "n1",
				Type:     "on",
				Config:   xmsgpack.EncodedJSON{"channel": 12.0},
				Position: spatial.XY{X: 10, Y: 20},
			}}))
			Expect(decoded.Edges).To(Equal(ir.Edges{{
				Source: ir.Handle{Node: "n1", Param: "output"},
				Target: ir.Handle{Node: "n2", Param: "input"},
			}}))
		})
	})
})
