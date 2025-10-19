// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
)

var _ = Describe("Nodes", func() {
	var (
		node1, node2, node3 ir.Node
		nodes               ir.Nodes
	)

	BeforeEach(func() {
		node1 = ir.Node{Key: "node1", Type: "add"}
		node2 = ir.Node{Key: "node2", Type: "multiply"}
		node3 = ir.Node{Key: "node3", Type: "divide"}
		nodes = ir.Nodes{node1, node2, node3}
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
})
