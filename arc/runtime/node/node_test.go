// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type mockNode struct {
	initCalled bool
	nextCalled int
}

func (m *mockNode) Init(ctx node.Context) {
	m.initCalled = true
}

func (m *mockNode) Next(ctx node.Context) {
	m.nextCalled++
}

type mockFactory struct {
	nodeType     string
	createCalled int
	returnNode   node.Node
	returnError  error
}

func (m *mockFactory) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	m.createCalled++
	if cfg.Node.Type != m.nodeType {
		return nil, query.NotFound
	}
	return m.returnNode, m.returnError
}

var _ = Describe("Node", func() {
	Describe("Context", func() {
		It("Should embed context.Context", func() {
			baseCtx := context.WithValue(context.Background(), "key", "value")
			nodeCtx := node.Context{Context: baseCtx}
			Expect(nodeCtx.Value("key")).To(Equal("value"))
		})
		It("Should call MarkChanged callback", func() {
			var markedOutputs []string
			nodeCtx := node.Context{
				Context: ctx,
				MarkChanged: func(output string) {
					markedOutputs = append(markedOutputs, output)
				},
			}
			nodeCtx.MarkChanged("out1")
			nodeCtx.MarkChanged("out2")
			Expect(markedOutputs).To(Equal([]string{"out1", "out2"}))
		})
		It("Should call ReportError callback", func() {
			var reportedErrors []error
			nodeCtx := node.Context{
				Context: ctx,
				ReportError: func(err error) {
					reportedErrors = append(reportedErrors, err)
				},
			}
			err1 := errors.New("error 1")
			err2 := errors.New("error 2")
			nodeCtx.ReportError(err1)
			nodeCtx.ReportError(err2)
			Expect(reportedErrors).To(HaveLen(2))
			Expect(reportedErrors[0]).To(Equal(err1))
			Expect(reportedErrors[1]).To(Equal(err2))
		})
		It("Should handle nil MarkChanged callback", func() {
			nodeCtx := node.Context{Context: ctx}
			Expect(func() {
				if nodeCtx.MarkChanged != nil {
					nodeCtx.MarkChanged("output")
				}
			}).ToNot(Panic())
		})
		It("Should handle nil ReportError callback", func() {
			nodeCtx := node.Context{Context: ctx}
			Expect(func() {
				if nodeCtx.ReportError != nil {
					nodeCtx.ReportError(errors.New("test"))
				}
			}).ToNot(Panic())
		})
	})
	Describe("Factory", func() {
		It("Should create node", func() {
			expectedNode := &mockNode{}
			factory := &mockFactory{
				nodeType:   "test",
				returnNode: expectedNode,
			}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "test"},
				State: s.Node("n1"),
			}
			n, err := factory.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(expectedNode))
			Expect(factory.createCalled).To(Equal(1))
		})
		It("Should return error when node type does not match", func() {
			factory := &mockFactory{nodeType: "other"}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "test"},
				State: s.Node("n1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
		})
		It("Should propagate factory errors", func() {
			expectedErr := errors.New("creation failed")
			factory := &mockFactory{
				nodeType:    "test",
				returnError: expectedErr,
			}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "test"},
				State: s.Node("n1"),
			}
			_, err := factory.Create(ctx, cfg)
			Expect(err).To(Equal(expectedErr))
		})
	})
	Describe("MultiFactory", func() {
		It("Should try factories in order", func() {
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{nodeType: "type2", returnNode: &mockNode{}}
			factory3 := &mockFactory{nodeType: "type3"}
			multi := node.MultiFactory{factory1, factory2, factory3}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "type2"},
				State: s.Node("n1"),
			}
			n, err := multi.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(0))
		})
		It("Should return NotFound when no factory matches", func() {
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{nodeType: "type2"}
			multi := node.MultiFactory{factory1, factory2}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("n1"),
			}
			_, err := multi.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
		})
		It("Should stop on first non-NotFound error", func() {
			expectedErr := errors.New("factory error")
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{
				nodeType:    "type2",
				returnError: expectedErr,
			}
			factory3 := &mockFactory{nodeType: "type3"}
			multi := node.MultiFactory{factory1, factory2, factory3}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "type2"},
				State: s.Node("n1"),
			}
			_, err := multi.Create(ctx, cfg)
			Expect(err).To(Equal(expectedErr))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(0))
		})
		It("Should handle empty factory list", func() {
			multi := node.MultiFactory{}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "test"},
				State: s.Node("n1"),
			}
			_, err := multi.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
		})
		It("Should handle single factory", func() {
			factory := &mockFactory{
				nodeType:   "test",
				returnNode: &mockNode{},
			}
			multi := node.MultiFactory{factory}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "test"},
				State: s.Node("n1"),
			}
			n, err := multi.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
			Expect(factory.createCalled).To(Equal(1))
		})
		It("Should try all factories when all return NotFound", func() {
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{nodeType: "type2"}
			factory3 := &mockFactory{nodeType: "type3"}
			multi := node.MultiFactory{factory1, factory2, factory3}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("n1"),
			}
			_, err := multi.Create(ctx, cfg)
			Expect(err).To(Equal(query.NotFound))
			Expect(factory1.createCalled).To(Equal(1))
			Expect(factory2.createCalled).To(Equal(1))
			Expect(factory3.createCalled).To(Equal(1))
		})
		It("Should return first successful match", func() {
			expectedNode := &mockNode{}
			factory1 := &mockFactory{nodeType: "type1"}
			factory2 := &mockFactory{
				nodeType:   "test",
				returnNode: expectedNode,
			}
			factory3 := &mockFactory{
				nodeType:   "test",
				returnNode: &mockNode{},
			}
			multi := node.MultiFactory{factory1, factory2, factory3}
			s := state.New(state.Config{
				Nodes: []ir.Node{{Key: "n1"}},
			})
			cfg := node.Config{
				Node:  ir.Node{Type: "test"},
				State: s.Node("n1"),
			}
			n, err := multi.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(expectedNode))
			Expect(factory3.createCalled).To(Equal(0))
		})
	})
	Describe("Config", func() {
		It("Should hold node configuration", func() {
			irNode := ir.Node{
				Key:  "test",
				Type: "constant",
			}
			s := state.New(state.Config{
				Nodes: []ir.Node{irNode},
			})
			cfg := node.Config{
				Node:  irNode,
				State: s.Node("test"),
			}
			Expect(cfg.Node.Key).To(Equal("test"))
			Expect(cfg.Node.Type).To(Equal("constant"))
			Expect(cfg.State).ToNot(BeNil())
		})
	})
})
