// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/graph/versions/v1"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Nodes", func() {
	var (
		node1, node2, node3 v1.Node
		nodes               v1.Nodes
	)

	BeforeEach(func() {
		node1 = v1.Node{Key: "node1", Position: spatial.XY{X: 1, Y: 2}}
		node2 = v1.Node{Key: "node2", Position: spatial.XY{X: 3, Y: 4}}
		node3 = v1.Node{Key: "node3", Position: spatial.XY{X: 5, Y: 6}}
		nodes = v1.Nodes{node1, node2, node3}
	})

	Describe("Find", func() {
		It("Should find existing node by key", func() {
			node := MustBeOk(nodes.Find("node2"))
			Expect(node.Key).To(Equal("node2"))
			Expect(node.Position).To(Equal(spatial.XY{X: 3, Y: 4}))
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
			Expect(node.Position).To(Equal(spatial.XY{X: 3, Y: 4}))
		})

		It("Should panic for non-existent key", func() {
			Expect(func() {
				_ = nodes.Get("nonexistent")
			}).To(Panic())
		})
	})

	Describe("Empty Collection", func() {
		It("Should handle Find on empty collection", func() {
			empty := v1.Nodes{}
			_, found := empty.Find("anything")
			Expect(found).To(BeFalse())
		})

		It("Should panic on Get with empty collection", func() {
			empty := v1.Nodes{}
			Expect(func() {
				_ = empty.Get("anything")
			}).To(Panic())
		})
	})
})
