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
	truthy     bool
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

func (m *mockNode) IsOutputTruthy(param string) bool {
	return m.truthy
}

func (m *mockNode) Reset() {}

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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
			Expect(s).ToNot(BeNil())
		})
	})
	Describe("Next", func() {
		It("Should execute stratum 0 nodes on every call", func() {
			prog = ir.IR{
				Nodes:  []ir.Node{{Key: "a"}},
				Strata: ir.Strata{{"a"}},
			}
			nodes["a"] = nodeA
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(0))
		})
		It("Should clear changed set at start of each strata execution", func() {
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, nodes)
			s.SetErrorHandler(handler)
			s.Next(ctx)
			Expect(handler.errors).To(HaveLen(2))
			Expect(handler.keys).To(ConsistOf("a", "b"))
		})
	})
	Describe("Stage Filtering", func() {
		It("Should execute all nodes when no sequences defined", func() {
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"a", "b"}},
				// No sequences
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(1))
			Expect(nodeB.nextCalled).To(Equal(1))
		})
		It("Should not execute staged nodes when no stage is active", func() {
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{}, // Global strata is empty (all nodes are in stages)
				Sequences: ir.Sequences{
					{
						Key: "seq1",
						Stages: []ir.Stage{
							{Key: "stage1", Nodes: []string{"a"}, Strata: ir.Strata{{"a"}}},
							{Key: "stage2", Nodes: []string{"b"}, Strata: ir.Strata{{"b"}}},
						},
					},
				},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)
			s.Next(ctx)
			// No stage is active, so staged nodes should not execute
			Expect(nodeA.nextCalled).To(Equal(0))
			Expect(nodeB.nextCalled).To(Equal(0))
		})
		It("Should execute nodes in active stage via entry node", func() {
			// Create an entry node that triggers stage activation
			entryNode := &mockNode{
				onNext: func(ctx node.Context) {
					ctx.ActivateStage()
				},
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "entry_seq1_stage1"}, // Entry node in global strata
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"entry_seq1_stage1"}}, // Entry node executes globally
				Sequences: ir.Sequences{
					{
						Key: "seq1",
						Stages: []ir.Stage{
							{Key: "stage1", Nodes: []string{"a"}, Strata: ir.Strata{{"a"}}},
							{Key: "stage2", Nodes: []string{"b"}, Strata: ir.Strata{{"b"}}},
						},
					},
				},
			}
			nodes["entry_seq1_stage1"] = entryNode
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)
			s.Next(ctx)
			// Entry node triggered stage1 activation
			Expect(nodeA.nextCalled).To(Equal(1))
			Expect(nodeB.nextCalled).To(Equal(0))
		})
		It("Should switch active stage via entry node", func() {
			// Entry node for stage1 always activates
			entryStage1 := &mockNode{
				onNext: func(ctx node.Context) {
					ctx.ActivateStage()
				},
			}
			// Entry node for stage2 - initially does nothing, then activates
			activateStage2 := false
			entryStage2 := &mockNode{
				onNext: func(ctx node.Context) {
					if activateStage2 {
						ctx.ActivateStage()
					}
				},
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "entry_seq1_stage1"},
					{Key: "entry_seq1_stage2"},
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"entry_seq1_stage1", "entry_seq1_stage2"}},
				Sequences: ir.Sequences{
					{
						Key: "seq1",
						Stages: []ir.Stage{
							{Key: "stage1", Nodes: []string{"a"}, Strata: ir.Strata{{"a"}}},
							{Key: "stage2", Nodes: []string{"b"}, Strata: ir.Strata{{"b"}}},
						},
					},
				},
			}
			nodes["entry_seq1_stage1"] = entryStage1
			nodes["entry_seq1_stage2"] = entryStage2
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)

			// First Next: stage1 activates
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(1))
			Expect(nodeB.nextCalled).To(Equal(0))

			// Disable stage1 entry, enable stage2 entry
			entryStage1.onNext = nil
			activateStage2 = true
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(1)) // Still 1 (not in stage2)
			Expect(nodeB.nextCalled).To(Equal(1)) // Now executes
		})
		It("Should always execute nodes in global strata", func() {
			entryNode := &mockNode{
				onNext: func(ctx node.Context) {
					ctx.ActivateStage()
				},
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "entry_seq1_stage1"},
					{Key: "a"},
					{Key: "b"},
					{Key: "c"}, // Not in any stage - in global strata
				},
				Strata: ir.Strata{{"entry_seq1_stage1", "c"}}, // c is in global strata
				Sequences: ir.Sequences{
					{
						Key: "seq1",
						Stages: []ir.Stage{
							{Key: "stage1", Nodes: []string{"a"}, Strata: ir.Strata{{"a"}}},
							{Key: "stage2", Nodes: []string{"b"}, Strata: ir.Strata{{"b"}}},
						},
					},
				},
			}
			nodes["entry_seq1_stage1"] = entryNode
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			nodes["c"] = nodeC
			s = scheduler.New(prog, nodes)

			// Activate stage1 - "c" should still execute
			s.Next(ctx)
			Expect(nodeA.nextCalled).To(Equal(1))
			Expect(nodeB.nextCalled).To(Equal(0))
			Expect(nodeC.nextCalled).To(Equal(1)) // Always runs

			// Next cycle
			s.Next(ctx)
			Expect(nodeC.nextCalled).To(Equal(2)) // Still runs
		})
	})
	Describe("One-Shot Edges", func() {
		It("Should fire one-shot edge when output is truthy", func() {
			nodeA.truthy = true
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
						Kind:   ir.OneShot,
					},
				},
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))
		})
		It("Should not fire one-shot edge when output is not truthy", func() {
			nodeA.truthy = false
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
						Kind:   ir.OneShot,
					},
				},
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(0))
		})
		It("Should only fire one-shot once per stage activation", func() {
			entryNode := &mockNode{
				onNext: func(ctx node.Context) {
					ctx.ActivateStage()
				},
			}
			nodeA.truthy = true
			nodeA.onNext = func(ctx node.Context) {
				ctx.MarkChanged("out")
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "entry_seq1_stage1"},
					{Key: "a"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"entry_seq1_stage1"}},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "b", Param: "in"},
						Kind:   ir.OneShot,
					},
				},
				Sequences: ir.Sequences{
					{
						Key: "seq1",
						Stages: []ir.Stage{
							{
								Key:    "stage1",
								Nodes:  []string{"a", "b"},
								Strata: ir.Strata{{"a"}, {"b"}},
							},
						},
					},
				},
			}
			nodes["entry_seq1_stage1"] = entryNode
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)

			// First Next: stage activates, a runs, one-shot fires to b
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))

			// Second Next: a runs again, but one-shot already fired
			entryNode.onNext = nil // Don't re-activate
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1)) // Still 1, one-shot already fired
		})
		It("Should always fire one-shot for global nodes (not in stage)", func() {
			nodeA.truthy = true
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
						Kind:   ir.OneShot,
					},
				},
				Strata: ir.Strata{{"a"}, {"b"}},
				// No sequences - nodes are global
			}
			nodes["a"] = nodeA
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)

			// First Next: one-shot fires
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(1))

			// Second Next: one-shot fires again (no tracking for global nodes)
			s.Next(ctx)
			Expect(nodeB.nextCalled).To(Equal(2))
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
			s = scheduler.New(prog, nodes)
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
			s = scheduler.New(prog, wideNodes)
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
			s = scheduler.New(prog, deepNodes)
			s.Next(ctx)
			for _, n := range deepNodes {
				Expect(n.(*mockNode).nextCalled).To(Equal(1))
			}
		})
	})
	Describe("Stage Convergence", func() {
		It("Should detect stage transition and continue until stable", func() {
			// Entry node for stage1 activates once then stops
			stage1Activated := false
			entryStage1 := &mockNode{
				onNext: func(ctx node.Context) {
					if !stage1Activated {
						stage1Activated = true
						ctx.ActivateStage()
					}
				},
			}
			// Stage1 node triggers entry to stage2 on first execution
			stage2Triggered := false
			stageANode := &mockNode{
				onNext: func(ctx node.Context) {
					if !stage2Triggered {
						stage2Triggered = true
						ctx.MarkChanged("out")
					}
				},
				truthy: true,
			}
			entryStage2 := &mockNode{
				onNext: func(ctx node.Context) {
					ctx.ActivateStage()
				},
			}
			prog = ir.IR{
				Nodes: []ir.Node{
					{Key: "entry_seq1_stage1"},
					{Key: "a"},
					{Key: "entry_seq1_stage2"},
					{Key: "b"},
				},
				Strata: ir.Strata{{"entry_seq1_stage1"}},
				Edges: []ir.Edge{
					{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "entry_seq1_stage2", Param: "in"},
						Kind:   ir.OneShot,
					},
				},
				Sequences: ir.Sequences{
					{
						Key: "seq1",
						Stages: []ir.Stage{
							{
								Key:    "stage1",
								Nodes:  []string{"a", "entry_seq1_stage2"},
								Strata: ir.Strata{{"a"}, {"entry_seq1_stage2"}},
							},
							{
								Key:    "stage2",
								Nodes:  []string{"b"},
								Strata: ir.Strata{{"b"}},
							},
						},
					},
				},
			}
			nodes["entry_seq1_stage1"] = entryStage1
			nodes["a"] = stageANode
			nodes["entry_seq1_stage2"] = entryStage2
			nodes["b"] = nodeB
			s = scheduler.New(prog, nodes)

			// Single Next call should:
			// 1. Execute entry_seq1_stage1 -> activates stage1
			// 2. Execute stage1 strata -> a runs, triggers entry_seq1_stage2
			// 3. Convergence loop detects transition to stage2
			// 4. Execute stage2 strata -> b runs
			s.Next(ctx)
			Expect(stageANode.nextCalled).To(Equal(1))
			Expect(nodeB.nextCalled).To(Equal(1)) // Stage2 executed via convergence
		})
	})
})
