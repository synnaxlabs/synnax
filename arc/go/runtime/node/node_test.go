// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
)

var _ = Describe("Node", func() {
	Describe("Config", func() {
		It("Should hold node configuration", func() {
			var (
				irNode = ir.Node{Key: "test", Type: "constant"}
				g      = graph.Graph{
					Nodes:     []graph.Node{{Key: "test", Type: "constant"}},
					Functions: []graph.Function{{Key: "constant"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = node.New(analyzed)
				cfg         = node.Config{Node: irNode, State: s.Node("test")}
			)
			Expect(cfg.Node.Key).To(Equal("test"))
			Expect(cfg.Node.Type).To(Equal("constant"))
			Expect(cfg.State).ToNot(BeNil())
		})
	})
})
