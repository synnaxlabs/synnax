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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

type mockNode struct {
	initCalled bool
	nextCalled int
}

func (m *mockNode) Init(node.Context) { m.initCalled = true }

func (m *mockNode) Next(node.Context) { m.nextCalled++ }

func (m *mockNode) IsOutputTruthy(param string) bool {
	return false
}

func (m *mockNode) Reset() {}

type mockFactory struct {
	returnNode   node.Node
	returnError  error
	nodeType     string
	createCalled int
}

func (m *mockFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	m.createCalled++
	if cfg.Node.Type != m.nodeType {
		return nil, query.ErrNotFound
	}
	return m.returnNode, m.returnError
}

var _ = Describe("Node", func() {
	Describe("MultiFactory", func() {
		It("Should try factories in order", func() {
			var (
				factory1 = &mockFactory{nodeType: "type1"}
				factory2 = &mockFactory{nodeType: "type2", returnNode: &mockNode{}}
				factory3 = &mockFactory{nodeType: "type3"}
				multi    = node.MultiFactory{factory1, factory2, factory3}
				g        = graph.Graph{
					Nodes:     []graph.Node{{Key: "n1", Type: "type2"}},
					Functions: []graph.Function{{Key: "type2"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = state.New(state.Config{IR: analyzed})
				cfg         = node.Config{
					Node:  ir.Node{Type: "type2"},
					State: s.Node("n1"),
				}
				n = MustSucceed(multi.Create(ctx, cfg))
			)
			Expect(n).ToNot(BeNil())
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(0))
		})
		It("Should return NotFound when no factory matches", func() {
			var (
				factory1 = &mockFactory{nodeType: "type1"}
				factory2 = &mockFactory{nodeType: "type2"}
				multi    = node.MultiFactory{factory1, factory2}
				g        = graph.Graph{
					Nodes:     []graph.Node{{Key: "n1", Type: "type2"}},
					Functions: []graph.Function{{Key: "type2"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = state.New(state.Config{IR: analyzed})
				cfg         = node.Config{
					Node:  ir.Node{Type: "unknown"},
					State: s.Node("n1"),
				}
				_, err = multi.Create(ctx, cfg)
			)
			Expect(err).To(HaveOccurredAs(query.ErrNotFound))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
		})
		It("Should stop on first non-NotFound error", func() {
			var (
				expectedErr = errors.New("factory error")
				factory1    = &mockFactory{nodeType: "type1"}
				factory2    = &mockFactory{
					nodeType:    "type2",
					returnError: expectedErr,
				}
				factory3 = &mockFactory{nodeType: "type3"}
				multi    = node.MultiFactory{factory1, factory2, factory3}
				g        = graph.Graph{
					Nodes:     []graph.Node{{Key: "n1", Type: "type2"}},
					Functions: []graph.Function{{Key: "type2"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = state.New(state.Config{IR: analyzed})
				cfg         = node.Config{
					Node:  ir.Node{Type: "type2"},
					State: s.Node("n1"),
				}
				_, err = multi.Create(ctx, cfg)
			)
			Expect(err).To(HaveOccurredAs(expectedErr))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(0))
		})
		It("Should handle empty factory list", func() {
			var (
				multi = node.MultiFactory{}
				g     = graph.Graph{
					Nodes:     []graph.Node{{Key: "n1", Type: "test"}},
					Functions: []graph.Function{{Key: "test"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = state.New(state.Config{IR: analyzed})
				cfg         = node.Config{
					Node:  ir.Node{Type: "test"},
					State: s.Node("n1"),
				}
				_, err = multi.Create(ctx, cfg)
			)
			Expect(err).To(HaveOccurredAs(query.ErrNotFound))
		})
		It("Should handle single factory", func() {
			var (
				factory = &mockFactory{
					nodeType:   "test",
					returnNode: &mockNode{},
				}
				multi = node.MultiFactory{factory}
				g     = graph.Graph{
					Nodes:     []graph.Node{{Key: "n1", Type: "test"}},
					Functions: []graph.Function{{Key: "test"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = state.New(state.Config{IR: analyzed})
				cfg         = node.Config{
					Node:  ir.Node{Type: "test"},
					State: s.Node("n1"),
				}
				n = MustSucceed(multi.Create(ctx, cfg))
			)
			Expect(n).ToNot(BeNil())
			Expect(factory.createCalled).To(Equal(1))
		})
		It("Should try all factories when all return NotFound", func() {
			var (
				factory1 = &mockFactory{nodeType: "type1"}
				factory2 = &mockFactory{nodeType: "type2"}
				factory3 = &mockFactory{nodeType: "type3"}
				multi    = node.MultiFactory{factory1, factory2, factory3}
				g        = graph.Graph{
					Nodes:     []graph.Node{{Key: "n1", Type: "type2"}},
					Functions: []graph.Function{{Key: "type2"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = state.New(state.Config{IR: analyzed})
				cfg         = node.Config{
					Node:  ir.Node{Type: "unknown"},
					State: s.Node("n1"),
				}
				_, err = multi.Create(ctx, cfg)
			)
			Expect(err).To(HaveOccurredAs(query.ErrNotFound))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(1))
		})
		It("Should return first successful match", func() {
			var (
				expectedNode = &mockNode{}
				factory1     = &mockFactory{nodeType: "type1"}
				factory2     = &mockFactory{
					nodeType:   "test",
					returnNode: expectedNode,
				}
				factory3 = &mockFactory{
					nodeType:   "test",
					returnNode: &mockNode{},
				}
				multi = node.MultiFactory{factory1, factory2, factory3}
				g     = graph.Graph{
					Nodes:     []graph.Node{{Key: "n1", Type: "test"}},
					Functions: []graph.Function{{Key: "test"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = state.New(state.Config{IR: analyzed})
				cfg         = node.Config{
					Node:  ir.Node{Type: "test"},
					State: s.Node("n1"),
				}
				n = MustSucceed(multi.Create(ctx, cfg))
			)
			Expect(n).To(Equal(expectedNode))
			Expect(factory3.createCalled).To(Equal(0))
		})
	})
	Describe("Config", func() {
		It("Should hold node configuration", func() {
			var (
				irNode = ir.Node{Key: "test", Type: "constant"}
				g      = graph.Graph{
					Nodes:     []graph.Node{{Key: "test", Type: "constant"}},
					Functions: []graph.Function{{Key: "constant"}},
				}
				analyzed, _ = graph.Analyze(ctx, g, nil)
				s           = state.New(state.Config{IR: analyzed})
				cfg         = node.Config{Node: irNode, State: s.Node("test")}
			)
			Expect(cfg.Node.Key).To(Equal("test"))
			Expect(cfg.Node.Type).To(Equal("constant"))
			Expect(cfg.State).ToNot(BeNil())
		})
	})
})
