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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

type mockNode struct {
	nextCalled int
}

func (m *mockNode) Next(node.Context) { m.nextCalled++ }

func (m *mockNode) IsOutputTruthy(param string) bool { return false }

func (m *mockNode) Outputs() []string { return nil }

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

func newTestConfig(ctx context.Context, nodeType string) node.Config {
	g := graph.Graph{
		Nodes:     []graph.Node{{Key: "n1", Type: nodeType}},
		Functions: []graph.Function{{Key: nodeType}},
	}
	analyzed, _ := graph.Analyze(ctx, g, nil)
	s := node.New(analyzed)
	return node.Config{
		Node:  ir.Node{Type: nodeType},
		State: s.Node("n1"),
	}
}

var _ = Describe("Node", func() {
	Describe("CompoundFactory", func() {
		It("Should try factories in order", func(ctx SpecContext) {
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{nodeType: "type2", returnNode: &mockNode{}}
			factory3 := &mockFactory{nodeType: "type3"}
			compound := node.CompoundFactory{factory1, factory2, factory3}
			n := MustSucceed(compound.Create(ctx, newTestConfig(ctx, "type2")))
			Expect(n).ToNot(BeNil())
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(0))
		})

		It("Should return NotFound when no factory matches", func(ctx SpecContext) {
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{nodeType: "type2"}
			compound := node.CompoundFactory{factory1, factory2}
			_, err := compound.Create(ctx, newTestConfig(ctx, "unknown"))
			Expect(err).To(MatchError(query.ErrNotFound))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
		})

		It("Should stop on first non-NotFound error", func(ctx SpecContext) {
			expectedErr := errors.New("factory error")
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{nodeType: "type2", returnError: expectedErr}
			factory3 := &mockFactory{nodeType: "type3"}
			compound := node.CompoundFactory{factory1, factory2, factory3}
			_, err := compound.Create(ctx, newTestConfig(ctx, "type2"))
			Expect(err).To(MatchError(expectedErr))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(0))
		})

		It("Should handle empty factory list", func(ctx SpecContext) {
			compound := node.CompoundFactory{}
			Expect(compound.Create(ctx, newTestConfig(ctx, "test"))).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should handle single factory", func(ctx SpecContext) {
			factory := &mockFactory{nodeType: "test", returnNode: &mockNode{}}
			compound := node.CompoundFactory{factory}
			n := MustSucceed(compound.Create(ctx, newTestConfig(ctx, "test")))
			Expect(n).ToNot(BeNil())
			Expect(factory.createCalled).To(Equal(1))
		})

		It("Should try all factories when all return NotFound", func(ctx SpecContext) {
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{nodeType: "type2"}
			factory3 := &mockFactory{nodeType: "type3"}
			compound := node.CompoundFactory{factory1, factory2, factory3}
			Expect(compound.Create(ctx, newTestConfig(ctx, "unknown"))).Error().To(MatchError(query.ErrNotFound))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(1))
		})

		It("Should return first successful match", func(ctx SpecContext) {
			expectedNode := &mockNode{}
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{nodeType: "test", returnNode: expectedNode}
			factory3 := &mockFactory{nodeType: "test", returnNode: &mockNode{}}
			compound := node.CompoundFactory{factory1, factory2, factory3}
			n := MustSucceed(compound.Create(ctx, newTestConfig(ctx, "test")))
			Expect(n).To(Equal(expectedNode))
			Expect(factory3.createCalled).To(Equal(0))
		})
	})

	Describe("Config", func() {
		It("Should hold node configuration", func(ctx SpecContext) {
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
