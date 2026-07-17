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
)

var _ = Describe("Nodes", func() {
	var (
		node1, node2, node3 v2.Node
		nodes               v2.Nodes
	)

	BeforeEach(func() {
		node1 = v2.Node{Key: "node1", Type: "add"}
		node2 = v2.Node{Key: "node2", Type: "multiply"}
		node3 = v2.Node{Key: "node3", Type: "divide"}
		nodes = v2.Nodes{node1, node2, node3}
	})

	Describe("Find", func() {
		It("Should find existing node by key", func() {
			node, found := nodes.Find("node2")
			Expect(found).To(BeTrue())
			Expect(node.Key).To(Equal("node2"))
			Expect(node.Type).To(Equal("multiply"))
		})

		It("Should return false for non-existent key", func() {
			_, found := nodes.Find("nonexistent")
			Expect(found).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("Should get existing node by key", func() {
			node := nodes.Get("node2")
			Expect(node.Key).To(Equal("node2"))
			Expect(node.Type).To(Equal("multiply"))
		})

		It("Should panic for non-existent key", func() {
			Expect(func() {
				_ = nodes.Get("nonexistent")
			}).To(Panic())
		})
	})

	Describe("Empty Collection", func() {
		It("Should handle Find on empty collection", func() {
			empty := v2.Nodes{}
			_, found := empty.Find("anything")
			Expect(found).To(BeFalse())
		})

		It("Should panic on Get with empty collection", func() {
			empty := v2.Nodes{}
			Expect(func() {
				_ = empty.Get("anything")
			}).To(Panic())
		})
	})
})

var _ = Describe("IsEntryNode", func() {
	reads := func(key uint32) types.Channels {
		return types.Channels{Read: map[uint32]string{key: "ch"}}
	}
	edgeInto := func(nodeKey string) v2.Edge {
		return v2.Edge{Target: v2.Handle{Node: nodeKey, Param: v2.DefaultInputParam}}
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
