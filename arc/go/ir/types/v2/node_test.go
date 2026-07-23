// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/arc/ir/types/v2"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Node", func() {
	Describe("IsEntryNode", func() {
		reads := func(key uint32) types.Channels {
			return types.Channels{Read: map[uint32]string{key: "ch"}}
		}
		edgeInto := func(nodeKey string) v2.Edge {
			return v2.Edge{Target: v2.Handle{Node: nodeKey, Param: "input"}}
		}
		DescribeTable(
			"Classification",
			func(node v2.Node, edges v2.Edges, expected bool) {
				Expect(node.IsEntryNode(edges)).To(Equal(expected))
			},
			Entry("no incoming edges and no channel reads",
				v2.Node{Key: "n"}, v2.Edges{}, true),
			Entry("an incoming edge",
				v2.Node{Key: "n"}, v2.Edges{edgeInto("n")}, false),
			Entry("a channel read",
				v2.Node{Key: "n", Channels: reads(1)}, v2.Edges{}, false),
			Entry("both an incoming edge and a channel read",
				v2.Node{Key: "n", Channels: reads(1)}, v2.Edges{edgeInto("n")}, false),
			Entry("an edge that targets a different node",
				v2.Node{Key: "n"}, v2.Edges{edgeInto("other")}, true),
		)
	})

	Describe("String", func() {
		DescribeTable(
			"Rendering",
			func(node v2.Node, expected string) {
				Expect(node.String()).To(Equal(expected))
			},
			Entry("no inputs, outputs, or channels",
				v2.Node{Key: "n1", Type: "add"},
				"n1 (type: add)\n└── channels: (none)\n"),
			Entry("inputs without outputs",
				v2.Node{
					Key:    "n1",
					Type:   "add",
					Inputs: types.Params{{Name: "x", Type: types.I64()}},
				},
				"n1 (type: add)\n├── channels: (none)\n└── inputs: x (i64)\n"),
			Entry("inputs, outputs, and channels",
				v2.Node{
					Key:  "n1",
					Type: "add",
					Inputs: types.Params{
						{Name: "x", Type: types.I64()},
						{Name: "y", Type: types.I64()},
					},
					Outputs: types.Params{{Name: "output", Type: types.I64()}},
					Channels: types.Channels{
						Read:  map[uint32]string{1: "sensor"},
						Write: map[uint32]string{2: "valve"},
					},
				},
				"n1 (type: add)\n"+
					"├── channels: read [1: sensor], write [2: valve]\n"+
					"├── inputs: x (i64), y (i64)\n"+
					"└── outputs: output (i64)\n"),
		)
	})

	Describe("DecodeMsgpack", func() {
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Key      string
				Type     string
				Inputs   types.Params
				Outputs  types.Params
				Channels types.Channels
			}{
				Key:  "node1",
				Type: "fn1",
				Inputs: types.Params{
					{Name: "rate", Type: types.Type{Kind: types.KindF32}},
				},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v2.Node
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Key).To(Equal("node1"))
			Expect(decoded.Type).To(Equal("fn1"))
			Expect(decoded.Inputs).To(HaveLen(1))
		})
	})
})
