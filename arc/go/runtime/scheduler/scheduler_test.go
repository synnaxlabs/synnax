// Copyright 2026 Synnax Labs, Inc.
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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/ir/testutil"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/x/telem"
)

// MockNode is a configurable mock for testing scheduler behavior.
type MockNode struct {
	NextCalled    int
	ResetCalled   int
	ElapsedValues []telem.TimeSpan
	ParamTruthy   map[string]bool
	OnNext        func(node.Context)
}

func NewMockNode() *MockNode {
	return &MockNode{ParamTruthy: make(map[string]bool)}
}

func (m *MockNode) Next(ctx node.Context) {
	m.NextCalled++
	m.ElapsedValues = append(m.ElapsedValues, ctx.Elapsed)
	if m.OnNext != nil {
		m.OnNext(ctx)
	}
}

func (m *MockNode) Reset() {
	m.ResetCalled++
}

func (m *MockNode) IsOutputTruthy(param string) bool {
	return m.ParamTruthy[param]
}

// MarkOnNext configures the node to mark a parameter as changed when Next() is called.
func (m *MockNode) MarkOnNext(param string) {
	m.OnNext = func(ctx node.Context) { ctx.MarkChanged(param) }
}

// ActivateOnNext configures the node to activate stage when Next() is called.
func (m *MockNode) ActivateOnNext() {
	m.OnNext = func(ctx node.Context) { ctx.ActivateStage() }
}

// ErrorOnNext configures the node to report an error when Next() is called.
func (m *MockNode) ErrorOnNext(err error) {
	m.OnNext = func(ctx node.Context) { ctx.ReportError(err) }
}

// MockErrorHandler collects errors for testing.
type MockErrorHandler struct {
	Errors []struct {
		NodeKey string
		Err     error
	}
}

func (h *MockErrorHandler) HandleError(nodeKey string, err error) {
	h.Errors = append(h.Errors, struct {
		NodeKey string
		Err     error
	}{nodeKey, err})
}

var _ = Describe("Scheduler", func() {
	var (
		ctx   context.Context
		nodes map[string]node.Node
		mocks map[string]*MockNode
	)

	mock := func(key string) *MockNode {
		m := NewMockNode()
		nodes[key] = m
		mocks[key] = m
		return m
	}

	build := func(prog ir.IR) *scheduler.Scheduler {
		return scheduler.New(prog, nodes)
	}

	BeforeEach(func() {
		ctx = context.Background()
		nodes = make(map[string]node.Node)
		mocks = make(map[string]*MockNode)
	})

	Describe("Construction & Initialization", func() {
		It("Should construct with empty program", func() {
			prog := ir.IR{}
			s := build(prog)
			s.Next(ctx, telem.Microsecond)
		})

		It("Should construct with single stratum", func() {
			mock("A")
			mock("B")
			mock("C")

			prog := testutil.NewIRBuilder().
				Node("A").Node("B").Node("C").
				Strata([][]string{{"A", "B", "C"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(mocks["A"].NextCalled).To(Equal(1))
			Expect(mocks["B"].NextCalled).To(Equal(1))
			Expect(mocks["C"].NextCalled).To(Equal(1))
		})

		It("Should build transition table", func() {
			triggerA := mock("trigger_a")
			mock("trigger_b")
			entryA := mock("entry_seq_stage_a")
			mock("entry_seq_stage_b")
			mock("A")
			mock("B")

			triggerA.MarkOnNext("activate")
			triggerA.ParamTruthy["activate"] = true
			entryA.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger_a").
				Node("trigger_b").
				Node("entry_seq_stage_a").
				Node("entry_seq_stage_b").
				Node("A").
				Node("B").
				OneShot("trigger_a", "activate", "entry_seq_stage_a", "input").
				OneShot("trigger_b", "activate", "entry_seq_stage_b", "input").
				Strata([][]string{{"trigger_a", "trigger_b"}, {"entry_seq_stage_a", "entry_seq_stage_b"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage_a", Strata: [][]string{{"A"}}},
					{Key: "stage_b", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			// If transition table built correctly, stage_a should be active
			Expect(mocks["A"].NextCalled).To(Equal(1))
		})
	})

	Describe("Basic Execution", func() {
		It("Should always execute stratum 0", func() {
			nodeA := mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)

			s.Next(ctx, telem.Microsecond)
			s.Next(ctx, 2*telem.Microsecond)
			s.Next(ctx, 3*telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(3))
		})

		It("Should skip higher strata without changes", func() {
			nodeA := mock("A")
			nodeB := mock("B")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Edge("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should pass elapsed time to context", func() {
			nodeA := mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)

			s.Next(ctx, 5*telem.Microsecond)
			s.Next(ctx, 10*telem.Microsecond)

			Expect(nodeA.ElapsedValues).To(HaveLen(2))
			Expect(nodeA.ElapsedValues[0]).To(Equal(5 * telem.Microsecond))
			Expect(nodeA.ElapsedValues[1]).To(Equal(10 * telem.Microsecond))
		})

		It("Should accumulate multiple next() calls", func() {
			nodeA := mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)

			for i := 0; i < 100; i++ {
				s.Next(ctx, telem.TimeSpan(i)*telem.Microsecond)
			}

			Expect(nodeA.NextCalled).To(Equal(100))
		})

		It("Should handle empty strata without crashing", func() {
			mock("A")
			mock("B")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Strata([][]string{{"A"}, {}, {"B"}}). // Empty middle stratum
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(mocks["A"].NextCalled).To(Equal(1))
			Expect(mocks["B"].NextCalled).To(Equal(0))
		})

		It("Should clear changed set per strata execution", func() {
			nodeA := mock("A")
			nodeB := mock("B")
			mock("C")

			firstCall := true
			nodeA.OnNext = func(ctx node.Context) {
				if firstCall {
					ctx.MarkChanged("output")
					firstCall = false
				}
			}

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Node("C").
				Edge("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}, {"C"}}).
				Build()

			s := build(prog)

			s.Next(ctx, telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))

			s.Next(ctx, 2*telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))
		})
	})

	Describe("Change Propagation", func() {
		It("Should propagate changes via continuous edge", func() {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Edge("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should not propagate without edge", func() {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should handle multiple outputs from single node", func() {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")

			nodeA.MarkOnNext("output_x")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Node("C").
				Edge("A", "output_x", "B", "input").
				Edge("A", "output_y", "C", "input").
				Strata([][]string{{"A"}, {"B", "C"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(0))
		})

		It("Should handle multiple inputs to single node", func() {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")

			nodeA.MarkOnNext("output")
			nodeB.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Node("C").
				Edge("A", "output", "C", "input_a").
				Edge("B", "output", "C", "input_b").
				Strata([][]string{{"A", "B"}, {"C"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should respect parameter-specific edges", func() {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")

			nodeA.MarkOnNext("param_a")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Node("C").
				Edge("A", "param_a", "B", "input").
				Edge("A", "param_b", "C", "input").
				Strata([][]string{{"A"}, {"B", "C"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(0))
		})

		It("Should handle chained propagation", func() {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")

			nodeA.MarkOnNext("output")
			nodeB.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Node("C").
				Edge("A", "output", "B", "input").
				Edge("B", "output", "C", "input").
				Strata([][]string{{"A"}, {"B"}, {"C"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should handle diamond dependency", func() {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			nodeD := mock("D")

			nodeA.MarkOnNext("output")
			nodeB.MarkOnNext("output")
			nodeC.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Node("C").
				Node("D").
				Edge("A", "output", "B", "input").
				Edge("A", "output", "C", "input").
				Edge("B", "output", "D", "input_b").
				Edge("C", "output", "D", "input_c").
				Strata([][]string{{"A"}, {"B", "C"}, {"D"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeD.NextCalled).To(Equal(1))
		})

		It("Should handle wide graph", func() {
			for i := 0; i < 10; i++ {
				mock(fmt.Sprintf("N%d", i))
			}

			stratum0 := make([]string, 10)
			for i := 0; i < 10; i++ {
				stratum0[i] = fmt.Sprintf("N%d", i)
			}

			builder := testutil.NewIRBuilder()
			for i := 0; i < 10; i++ {
				builder.Node(fmt.Sprintf("N%d", i))
			}
			prog := builder.Strata([][]string{stratum0}).Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			for i := 0; i < 10; i++ {
				Expect(mocks[fmt.Sprintf("N%d", i)].NextCalled).To(Equal(1))
			}
		})
	})

	Describe("One-Shot Edge Semantics", func() {
		It("Should fire one-shot when truthy", func() {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.MarkOnNext("output")
			nodeA.ParamTruthy["output"] = true

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				OneShot("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should not fire one-shot when falsy", func() {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.MarkOnNext("output")
			nodeA.ParamTruthy["output"] = false

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				OneShot("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should fire one-shot only once per stage", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()
			nodeA.MarkOnNext("output")
			nodeA.ParamTruthy["output"] = true

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				Node("B").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				OneShot("A", "output", "B", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}, {"B"}}},
				}).
				Build()

			s := build(prog)

			s.Next(ctx, telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))

			s.Next(ctx, 2*telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should fire one-shot once ever in global strata", func() {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.MarkOnNext("output")
			nodeA.ParamTruthy["output"] = true

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				OneShot("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)

			s.Next(ctx, telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))

			s.Next(ctx, 2*telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))

			s.Next(ctx, 3*telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should reset one-shot on stage entry", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()
			nodeA.MarkOnNext("output")
			nodeA.ParamTruthy["output"] = true

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				Node("B").
				// Global: continuous edge so it triggers every time
				Edge("trigger", "activate", "entry_seq_stage", "input").
				// Stage: A→B one-shot
				OneShot("A", "output", "B", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}, {"B"}}},
				}).
				Build()

			s := build(prog)

			s.Next(ctx, telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeA.ResetCalled).To(Equal(1))

			// Stage re-activates via continuous edge, clearing fired_one_shots
			s.Next(ctx, 2*telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(2))
			Expect(nodeA.ResetCalled).To(Equal(2))
		})

		It("Should not affect continuous edge by truthiness", func() {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.MarkOnNext("output")
			nodeA.ParamTruthy["output"] = false // Falsy, but continuous edge

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Edge("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeB.NextCalled).To(Equal(1))
		})
	})

	Describe("Stage Filtering & Transitions", func() {
		It("Should not execute when no stage is active", func() {
			mock("A")
			nodeB := mock("B")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Strata([][]string{{"A"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should execute staged nodes when active", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should always execute global strata", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				Node("B").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				Strata([][]string{{"trigger", "A"}, {"entry_seq_stage"}}). // A is global at stratum 0
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1)) // Global
			Expect(nodeB.NextCalled).To(Equal(1)) // Stage
		})

		It("Should activate stage via entry node", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}}},
				}).
				Build()

			s := build(prog)
			Expect(nodeA.NextCalled).To(Equal(0))
			s.Next(ctx, telem.Microsecond)
			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should deactivate previous stage on transition", func() {
			trigger := mock("trigger")
			entryA := mock("entry_seq_stage_a")
			entryB := mock("entry_seq_stage_b")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryA.ActivateOnNext()
			entryB.ActivateOnNext()
			nodeA.MarkOnNext("to_b")
			nodeA.ParamTruthy["to_b"] = true

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage_a").
				Node("entry_seq_stage_b").
				Node("A").
				Node("B").
				OneShot("trigger", "activate", "entry_seq_stage_a", "input").
				OneShot("A", "to_b", "entry_seq_stage_b", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage_a", "entry_seq_stage_b"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage_a", Strata: [][]string{{"A"}, {"entry_seq_stage_b"}}},
					{Key: "stage_b", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))

			// Stage_b remains active, stage_a deactivated
			s.Next(ctx, 2*telem.Microsecond)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(2))
		})

		It("Should reset nodes on stage transition", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}}},
				}).
				Build()

			s := build(prog)

			Expect(nodeA.ResetCalled).To(Equal(0))
			s.Next(ctx, telem.Microsecond)
			Expect(nodeA.ResetCalled).To(Equal(1))
		})

		It("Should maintain cross-sequence independence", func() {
			trigger1 := mock("trigger1")
			trigger2 := mock("trigger2")
			entry1 := mock("entry_seq1_stage")
			entry2 := mock("entry_seq2_stage")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger1.MarkOnNext("activate")
			trigger1.ParamTruthy["activate"] = true
			trigger2.MarkOnNext("activate")
			trigger2.ParamTruthy["activate"] = true
			entry1.ActivateOnNext()
			entry2.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger1").
				Node("trigger2").
				Node("entry_seq1_stage").
				Node("entry_seq2_stage").
				Node("A").
				Node("B").
				OneShot("trigger1", "activate", "entry_seq1_stage", "input").
				OneShot("trigger2", "activate", "entry_seq2_stage", "input").
				Strata([][]string{{"trigger1", "trigger2"}, {"entry_seq1_stage", "entry_seq2_stage"}}).
				Sequence("seq1", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}}},
				}).
				Sequence("seq2", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should handle multiple stages in sequence", func() {
			trigger := mock("trigger")
			entryA := mock("entry_seq_stage_a")
			entryB := mock("entry_seq_stage_b")
			entryC := mock("entry_seq_stage_c")
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryA.ActivateOnNext()
			entryB.ActivateOnNext()
			entryC.ActivateOnNext()
			nodeA.MarkOnNext("to_b")
			nodeA.ParamTruthy["to_b"] = true
			nodeB.MarkOnNext("to_c")
			nodeB.ParamTruthy["to_c"] = true

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage_a").
				Node("entry_seq_stage_b").
				Node("entry_seq_stage_c").
				Node("A").
				Node("B").
				Node("C").
				OneShot("trigger", "activate", "entry_seq_stage_a", "input").
				OneShot("A", "to_b", "entry_seq_stage_b", "input").
				OneShot("B", "to_c", "entry_seq_stage_c", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage_a", "entry_seq_stage_b", "entry_seq_stage_c"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage_a", Strata: [][]string{{"A"}, {"entry_seq_stage_b"}}},
					{Key: "stage_b", Strata: [][]string{{"B"}, {"entry_seq_stage_c"}}},
					{Key: "stage_c", Strata: [][]string{{"C"}}},
				}).
				Build()

			s := build(prog)

			// Single next() cascades through all stages: A→B→C
			s.Next(ctx, telem.Microsecond)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})
	})

	Describe("Convergence Loop", func() {
		It("Should converge single transition", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)
			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should complete cascading transitions", func() {
			trigger := mock("trigger")
			entryA := mock("entry_seq_stage_a")
			entryB := mock("entry_seq_stage_b")
			entryC := mock("entry_seq_stage_c")
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryA.ActivateOnNext()
			entryB.ActivateOnNext()
			entryC.ActivateOnNext()
			nodeA.MarkOnNext("output")
			nodeB.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage_a").
				Node("entry_seq_stage_b").
				Node("entry_seq_stage_c").
				Node("A").
				Node("B").
				Node("C").
				OneShot("trigger", "activate", "entry_seq_stage_a", "input").
				Edge("A", "output", "entry_seq_stage_b", "input").
				Edge("B", "output", "entry_seq_stage_c", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage_a"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage_a", Strata: [][]string{{"A"}, {"entry_seq_stage_b"}}},
					{Key: "stage_b", Strata: [][]string{{"B"}, {"entry_seq_stage_c"}}},
					{Key: "stage_c", Strata: [][]string{{"C"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should stop when stable", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)
			s.Next(ctx, 2*telem.Microsecond)
			Expect(nodeA.NextCalled).To(Equal(2))
		})

		It("Should prevent infinite loop with max iterations", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			nodeA := mock("A")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()
			// A triggers entry which re-activates the stage (infinite loop attempt)
			nodeA.MarkOnNext("reenter")
			nodeA.ParamTruthy["reenter"] = true

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("A").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				OneShot("A", "reenter", "entry_seq_stage", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"A"}, {"entry_seq_stage"}}},
				}).
				Build()

			s := build(prog)
			// Should not hang
			s.Next(ctx, telem.Microsecond)
			Expect(true).To(BeTrue())
		})

		It("Should detect transition during execution", func() {
			trigger := mock("trigger")
			entryA := mock("entry_seq_stage_a")
			entryB := mock("entry_seq_stage_b")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryA.ActivateOnNext()
			entryB.ActivateOnNext()
			nodeA.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage_a").
				Node("entry_seq_stage_b").
				Node("A").
				Node("B").
				OneShot("trigger", "activate", "entry_seq_stage_a", "input").
				Edge("A", "output", "entry_seq_stage_b", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage_a"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage_a", Strata: [][]string{{"A"}, {"entry_seq_stage_b"}}},
					{Key: "stage_b", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
		})
	})

	Describe("Error Handling", func() {
		It("Should pass errors to error handler", func() {
			nodeA := mock("A")
			testErr := errors.Newf("test error")
			nodeA.ErrorOnNext(testErr)

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			handler := &MockErrorHandler{}
			s.SetErrorHandler(handler)
			s.Next(ctx, telem.Microsecond)

			Expect(handler.Errors).To(HaveLen(1))
			Expect(handler.Errors[0].NodeKey).To(Equal("A"))
			Expect(handler.Errors[0].Err).To(Equal(testErr))
		})

		It("Should continue execution after error", func() {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.OnNext = func(ctx node.Context) {
				ctx.ReportError(errors.Newf("error from A"))
				ctx.MarkChanged("output")
			}

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Edge("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should return normally after error", func() {
			nodeA := mock("A")
			nodeA.ErrorOnNext(errors.Newf("node error"))

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			// Should not panic
			s.Next(ctx, telem.Microsecond)
			Expect(nodeA.NextCalled).To(Equal(1))
		})
	})

	Describe("Complex Graph Structures", func() {
		It("Should handle deep strata chain", func() {
			for i := 0; i < 10; i++ {
				m := mock(fmt.Sprintf("N%d", i))
				if i < 9 {
					m.MarkOnNext("output")
				}
			}

			builder := testutil.NewIRBuilder()
			for i := 0; i < 10; i++ {
				builder.Node(fmt.Sprintf("N%d", i))
			}

			for i := 0; i < 9; i++ {
				builder.Edge(fmt.Sprintf("N%d", i), "output", fmt.Sprintf("N%d", i+1), "input")
			}

			strata := make([][]string, 10)
			for i := 0; i < 10; i++ {
				strata[i] = []string{fmt.Sprintf("N%d", i)}
			}

			prog := builder.Strata(strata).Build()
			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			for i := 0; i < 10; i++ {
				Expect(mocks[fmt.Sprintf("N%d", i)].NextCalled).To(Equal(1))
			}
		})

		It("Should handle mixed continuous and one-shot", func() {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")

			nodeA.MarkOnNext("output")
			nodeB.MarkOnNext("output")
			nodeB.ParamTruthy["output"] = true

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Node("C").
				Edge("A", "output", "B", "input").
				OneShot("B", "output", "C", "input").
				Strata([][]string{{"A"}, {"B"}, {"C"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should handle global and staged mixed", func() {
			trigger := mock("trigger")
			entry := mock("entry_seq_stage")
			globalNode := mock("G")
			stagedNode := mock("S")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()
			globalNode.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage").
				Node("G").
				Node("S").
				OneShot("trigger", "activate", "entry_seq_stage", "input").
				Edge("G", "output", "S", "input").
				Strata([][]string{{"trigger", "G"}, {"entry_seq_stage"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"S"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(globalNode.NextCalled).To(Equal(1))
			Expect(stagedNode.NextCalled).To(Equal(1))
		})

		It("Should handle multi-sequence with shared global", func() {
			trigger1 := mock("trigger1")
			trigger2 := mock("trigger2")
			entry1 := mock("entry_seq1_stage")
			entry2 := mock("entry_seq2_stage")
			globalNode := mock("G")
			staged1 := mock("S1")
			staged2 := mock("S2")

			trigger1.MarkOnNext("activate")
			trigger1.ParamTruthy["activate"] = true
			trigger2.MarkOnNext("activate")
			trigger2.ParamTruthy["activate"] = true
			entry1.ActivateOnNext()
			entry2.ActivateOnNext()
			globalNode.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("trigger1").
				Node("trigger2").
				Node("entry_seq1_stage").
				Node("entry_seq2_stage").
				Node("G").
				Node("S1").
				Node("S2").
				OneShot("trigger1", "activate", "entry_seq1_stage", "input").
				OneShot("trigger2", "activate", "entry_seq2_stage", "input").
				Edge("G", "output", "S1", "input").
				Edge("G", "output", "S2", "input").
				Strata([][]string{{"trigger1", "trigger2", "G"}, {"entry_seq1_stage", "entry_seq2_stage"}}).
				Sequence("seq1", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"S1"}}},
				}).
				Sequence("seq2", []testutil.StageSpec{
					{Key: "stage", Strata: [][]string{{"S2"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)

			Expect(globalNode.NextCalled).To(Equal(1))
			Expect(staged1.NextCalled).To(Equal(1))
			Expect(staged2.NextCalled).To(Equal(1))
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle zero elapsed time", func() {
			nodeA := mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			s.Next(ctx, 0)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeA.ElapsedValues[0]).To(Equal(telem.TimeSpan(0)))
		})

		It("Should handle self-loop", func() {
			nodeA := mock("A")
			nodeA.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("A").
				Edge("A", "output", "A", "input").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)
			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should handle empty sequence", func() {
			mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Sequence("empty_seq", []testutil.StageSpec{}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond)
			Expect(mocks["A"].NextCalled).To(Equal(1))
		})
	})
})
