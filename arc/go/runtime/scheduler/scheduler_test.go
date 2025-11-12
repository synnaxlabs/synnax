// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package scheduler_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
)

type mockNode struct {
	initCalled bool
	nextCalled int
	onInit     func(ctx node.Context)
	onNext     func(ctx node.Context)
}

func (m *mockNode) Init(ctx node.Context) {
	m.initCalled = true
	if m.onInit != nil {
		m.onInit(ctx)
	}
}

func (m *mockNode) Next(ctx node.Context) {
	m.nextCalled++
	if m.onNext != nil {
		m.onNext(ctx)
	}
}

type mockErrorHandler struct {
	errors []error
	keys   []string
}

func (m *mockErrorHandler) HandleError(nodeKey string, err error) {
	m.keys = append(m.keys, nodeKey)
	m.errors = append(m.errors, err)
}

var _ = Describe("Scheduler", func() {
	var (
		prog  ir.IR
		nodes map[string]node.Node
		s     *scheduler.Scheduler
		nodeA *mockNode
		nodeB *mockNode
		nodeC *mockNode
		nodeD *mockNode
	)

	BeforeEach(func() {
		nodes = make(map[string]node.Node)
		nodeA = &mockNode{}
		nodeB = &mockNode{}
		nodeC = &mockNode{}
		nodeD = &mockNode{}
	})

	Describe("New", func() {
		It("Should create scheduler with single stratum", func() {
			prog = ir.IR{
				Nodes:  []ir.Node{{Key: "a"}},
				Strata: ir.Strata{{"a"}},
			}
			nodes["a"] = nodeA
			s = scheduler.New(ctx, prog, nodes)
			Expect(s).ToNot(BeNil())
		})
		It("Should create scheduler with multiple strata", func() {
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
					{Key: "c"},
				},
				Strata: ir.Strata{{"a"}, {"b"}, {"c"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			nodes["c"] = nodeC
			s = scheduler.New(ctx, prog, nodes)
			Expect(s).ToNot(BeNil())
		})
	})
	Describe("Init", func() {
		It("Should execute all nodes in stratum 0", func() {
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"a", "b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(ctx, prog, nodes)
			s.Init(ctx)
			Expect(nodeA.initCalled).To(BeTrue())
			Expect(nodeB.initCalled).To(BeTrue())
		})
		It("Should execute nodes in higher strata", func() {
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(ctx, prog, nodes)
			s.Init(ctx)
			Expect(nodeA.initCalled).To(BeTrue())
			Expect(nodeB.initCalled).To(BeTrue())
		})
		It("Should provide context with MarkChanged callback", func() {
			markedParams := []string{}
			nodeA.onInit = func(ctx node.Context) {
				ctx.MarkChanged("output")
			}
			prog = ir.IR{
				Nodes: []ir.Node{{Key: "a"}},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "output"},
						Target: ir.Handle{Node: "b", Param: "input"},
					},
				},
				Strata: ir.Strata{{"a"}},
			}
			nodes["a"] = nodeA
			s = scheduler.New(ctx, prog, nodes)
			s.Init(ctx)
			nodeA.onInit = func(ctx node.Context) {
				markedParams = append(markedParams, "output")
			}
			s.Init(ctx)
			Expect(markedParams).To(HaveLen(1))
		})
	})
	Describe("Next", func() {
		It("Should execute stratum 0 nodes on every call", func() {
			prog = ir.IR{
				Nodes:  []ir.Node{{Key: "a"}},
				Strata: ir.Strata{{"a"}},
			}
			nodes["a"] = nodeA
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(1))
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(2))
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(3))
		})
		It("Should execute downstream node when marked changed", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "b", Param: "in"},
					},
				},
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))
		})
		It("Should not execute downstream node when not marked changed", func() {
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "b", Param: "in"},
					},
				},
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(0))
		})
		It("Should clear changed set after execution", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "b", Param: "in"},
					},
				},
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))
			nodeA.onNext = nil
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))
		})
		It("Should propagate changes through multiple strata", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			nodeB.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
					{Key: "c"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "b", Param: "in"},
					},
					{
						Source: ir.Handle{Node: "b", Param: "out"},
						Target: ir.Handle{Node: "c", Param: "in"},
					},
				},
				Strata: ir.Strata{{"a"}, {"b"}, {"c"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			nodes["c"] = nodeC
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(1))
			Expect(nodeB.nextCalled).To(Equal(1))
			Expect(nodeC.nextCalled).To(Equal(1))
		})
		It("Should handle multiple outputs from single node", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out1")
				ctx.MarkChanged("out2")
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
					{Key: "c"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out1"},
						Target: ir.Handle{Node: "b", Param: "in"},
					},
					{
						Source: ir.Handle{Node: "a", Param: "out2"},
						Target: ir.Handle{Node: "c", Param: "in"},
					},
				},
				Strata: ir.Strata{{"a"}, {"b", "c"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			nodes["c"] = nodeC
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))
			Expect(nodeC.nextCalled).To(Equal(1))
		})
		It("Should handle multiple inputs to single node", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			nodeB.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
					{Key: "c"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "c", Param: "in1"},
					},
					{
						Source: ir.Handle{Node: "b", Param: "out"},
						Target: ir.Handle{Node: "c", Param: "in2"},
					},
				},
				Strata: ir.Strata{{"a", "b"}, {"c"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			nodes["c"] = nodeC
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeC.nextCalled).To(Equal(1))
		})
		It("Should only mark changed when specific param changes", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out1")
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
					{Key: "c"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out1"},
						Target: ir.Handle{Node: "b", Param: "in"},
					},
					{
						Source: ir.Handle{Node: "a", Param: "out2"},
						Target: ir.Handle{Node: "c", Param: "in"},
					},
				},
				Strata: ir.Strata{{"a"}, {"b", "c"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			nodes["c"] = nodeC
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))
			Expect(nodeC.nextCalled).To(Equal(0))
		})
	})
	Describe("MarkNodeChanged", func() {
		It("Should mark node as changed", func() {
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(ctx, prog, nodes)
			s.MarkNodeChanged("b")
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))
		})
		It("Should allow external marking of stratum 0 nodes", func() {
			prog = ir.IR{
				Nodes:  []ir.Node{{Key: "a"}},
				Strata: ir.Strata{{"a"}},
			}
			nodes["a"] = nodeA
			s = scheduler.New(ctx, prog, nodes)
			s.MarkNodeChanged("a")
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(1))
		})
	})
	Describe("ErrorHandler", func() {
		var handler *mockErrorHandler

		BeforeEach(func() {
			handler = &mockErrorHandler{}
		})

		It("Should call error handler when error reported", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.ReportError(context.Canceled)
			}
			prog = ir.IR{
				Nodes:  []ir.Node{{Key: "a"}},
				Strata: ir.Strata{{"a"}},
			}
			nodes["a"] = nodeA
			s = scheduler.New(ctx, prog, nodes)
			s.SetErrorHandler(handler)
			s.Next(ctx)
			Expect(handler.errors).To(HaveLen(1))
			Expect(handler.errors[0]).To(Equal(context.Canceled))
			Expect(handler.keys[0]).To(Equal("a"))
		})
		It("Should not panic when error handler is nil", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.ReportError(context.Canceled)
			}
			prog = ir.IR{
				Nodes:  []ir.Node{{Key: "a"}},
				Strata: ir.Strata{{"a"}},
			}
			nodes["a"] = nodeA
			s = scheduler.New(ctx, prog, nodes)
			Expect(func() {
				s.Next(ctx)
			}).ToNot(Panic())
		})
		It("Should report multiple errors from same node", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.ReportError(context.Canceled)
				ctx.ReportError(context.DeadlineExceeded)
			}
			prog = ir.IR{
				Nodes:  []ir.Node{{Key: "a"}},
				Strata: ir.Strata{{"a"}},
			}
			nodes["a"] = nodeA
			s = scheduler.New(ctx, prog, nodes)
			s.SetErrorHandler(handler)
			s.Next(ctx)
			Expect(handler.errors).To(HaveLen(2))
			Expect(handler.keys).To(HaveLen(2))
		})
		It("Should report errors from different nodes", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.ReportError(context.Canceled)
			}
			nodeB.onNext = func(ctx node.Context) {
				ctx.ReportError(context.DeadlineExceeded)
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"a", "b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(ctx, prog, nodes)
			s.SetErrorHandler(handler)
			s.Next(ctx)
			Expect(handler.errors).To(HaveLen(2))
			Expect(handler.keys).To(ConsistOf("a", "b"))
		})
	})
	Describe("Complex Graphs", func() {
		It("Should handle diamond dependency", func() {
			nodeA.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			nodeB.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			nodeC.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
					{Key: "c"},
					{Key: "d"},
				},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "b", Param: "in"},
					},
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "c", Param: "in"},
					},
					{
						Source: ir.Handle{Node: "b", Param: "out"},
						Target: ir.Handle{Node: "d", Param: "in1"},
					},
					{
						Source: ir.Handle{Node: "c", Param: "out"},
						Target: ir.Handle{Node: "d", Param: "in2"},
					},
				},
				Strata: ir.Strata{{"a"}, {"b", "c"}, {"d"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			nodes["c"] = nodeC
			nodes["d"] = nodeD
			s = scheduler.New(ctx, prog, nodes)
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(1))
			Expect(nodeB.nextCalled).To(Equal(1))
			Expect(nodeC.nextCalled).To(Equal(1))
			Expect(nodeD.nextCalled).To(Equal(1))
		})
		It("Should handle wide graph with many nodes per stratum", func() {
			wideNodes := make(map[string]node.Node)
			for i := 0; i < 10; i++ {
				wideNodes[string(rune('a'+i))] = &mockNode{}
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"}, {Key: "b"}, {Key: "c"}, {Key: "d"}, {Key: "e"},
					{Key: "f"}, {Key: "g"}, {Key: "h"}, {Key: "i"}, {Key: "j"},
				},
				Strata: ir.Strata{{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
			}
			s = scheduler.New(ctx, prog, wideNodes)
			s.Next(ctx)
			for _, n := range wideNodes {
				Expect(n.(*mockNode).nextCalled).To(Equal(1))
			}
		})
		It("Should handle deep graph with many strata", func() {
			deepNodes := make(map[string]node.Node)
			strata := make(ir.Strata, 10)
			irNodes := make([]ir.Node, 10)
			edges := make([]ir.Edge, 9)
			for i := 0; i < 10; i++ {
				key := string(rune('a' + i))
				deepNodes[key] = &mockNode{
					onNext: func(ctx node.Context) {
						ctx.MarkChanged("out")
					},
				}
				strata[i] = []string{key}
				irNodes[i] = ir.Node{Key: key}
				if i > 0 {
					edges[i-1] = ir.Edge{
						Source: ir.Handle{Node: string(rune('a' + i - 1)), Param: "out"},
						Target: ir.Handle{Node: key, Param: "in"},
					}
				}
			}
			prog = ir.IR{
				Nodes:  irNodes,
				Edges:  edges,
				Strata: strata,
			}
			s = scheduler.New(ctx, prog, deepNodes)
			s.Next(ctx)
			for _, n := range deepNodes {
				Expect(n.(*mockNode).nextCalled).To(Equal(1))
			}
		})
	})
})
