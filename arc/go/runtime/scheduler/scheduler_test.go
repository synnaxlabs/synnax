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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// MockNode is a configurable mock for testing scheduler behavior.
type MockNode struct {
	ParamTruthy   map[string]bool
	OnNext        func(node.Context)
	ElapsedValues []telem.TimeSpan
	NextCalled    int
	ResetCalled   int
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
		Err     error
		NodeKey string
	}
}

func (h *MockErrorHandler) HandleError(_ context.Context, nodeKey string, err error) {
	h.Errors = append(h.Errors, struct {
		Err     error
		NodeKey string
	}{err, nodeKey})
}

var _ = Describe("Scheduler", func() {
	var (
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
		return scheduler.New(prog, nodes, 0)
	}

	BeforeEach(func() {
		nodes = make(map[string]node.Node)
		mocks = make(map[string]*MockNode)
	})

	Describe("Construction & Initialization", func() {
		It("Should construct with empty program", func(ctx SpecContext) {
			prog := ir.IR{}
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
		})

		It("Should construct with single stratum", func(ctx SpecContext) {
			mock("A")
			mock("B")
			mock("C")

			prog := testutil.NewIRBuilder().
				Node("A").Node("B").Node("C").
				Strata([][]string{{"A", "B", "C"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(mocks["A"].NextCalled).To(Equal(1))
			Expect(mocks["B"].NextCalled).To(Equal(1))
			Expect(mocks["C"].NextCalled).To(Equal(1))
		})

		It("Should build transition table", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			// If transition table built correctly, stage_a should be active
			Expect(mocks["A"].NextCalled).To(Equal(1))
		})
	})

	Describe("Basic Execution", func() {
		It("Should always execute stratum 0", func(ctx SpecContext) {
			nodeA := mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)

			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(3))
		})

		It("Should skip higher strata without changes", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Edge("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should pass elapsed time to context", func(ctx SpecContext) {
			nodeA := mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)

			s.Next(ctx, 5*telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 10*telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.ElapsedValues).To(HaveLen(2))
			Expect(nodeA.ElapsedValues[0]).To(Equal(5 * telem.Microsecond))
			Expect(nodeA.ElapsedValues[1]).To(Equal(10 * telem.Microsecond))
		})

		It("Should accumulate multiple next() calls", func(ctx SpecContext) {
			nodeA := mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)

			for i := range 100 {
				s.Next(ctx, telem.TimeSpan(i)*telem.Microsecond, node.ReasonTimerTick)
			}

			Expect(nodeA.NextCalled).To(Equal(100))
		})

		It("Should handle empty strata without crashing", func(ctx SpecContext) {
			mock("A")
			mock("B")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Strata([][]string{{"A"}, {}, {"B"}}). // Empty middle stratum
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(mocks["A"].NextCalled).To(Equal(1))
			Expect(mocks["B"].NextCalled).To(Equal(0))
		})

		It("Should clear changed set per strata execution", func(ctx SpecContext) {
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

			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))

			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})
	})

	Describe("Change Propagation", func() {
		It("Should propagate changes via continuous edge", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should not propagate without edge", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should handle multiple outputs from single node", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(0))
		})

		It("Should handle multiple inputs to single node", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should respect parameter-specific edges", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(0))
		})

		It("Should handle chained propagation", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should handle diamond dependency", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeD.NextCalled).To(Equal(1))
		})

		It("Should handle wide graph", func(ctx SpecContext) {
			for i := range 10 {
				mock(fmt.Sprintf("N%d", i))
			}

			stratum0 := make([]string, 10)
			for i := range 10 {
				stratum0[i] = fmt.Sprintf("N%d", i)
			}

			builder := testutil.NewIRBuilder()
			for i := range 10 {
				builder.Node(fmt.Sprintf("N%d", i))
			}
			prog := builder.Strata([][]string{stratum0}).Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			for i := range 10 {
				Expect(mocks[fmt.Sprintf("N%d", i)].NextCalled).To(Equal(1))
			}
		})
	})

	Describe("One-Shot Edge Semantics", func() {
		It("Should fire one-shot when truthy", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should not fire one-shot when falsy", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should fire one-shot only once per stage", func(ctx SpecContext) {
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

			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))

			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should fire one-shot once ever in global strata", func(ctx SpecContext) {
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

			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))

			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))

			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should reset one-shot on stage entry", func(ctx SpecContext) {
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

			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeA.ResetCalled).To(Equal(1))

			// Stage re-activates via continuous edge, clearing fired_one_shots
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(2))
			Expect(nodeA.ResetCalled).To(Equal(2))
		})

		It("Should not affect continuous edge by truthiness", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeB.NextCalled).To(Equal(1))
		})
	})

	Describe("Stage Filtering & Transitions", func() {
		It("Should not execute when no stage is active", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should execute staged nodes when active", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should always execute global strata", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1)) // Global
			Expect(nodeB.NextCalled).To(Equal(1)) // Stage
		})

		It("Should activate stage via entry node", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should deactivate previous stage on transition", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))

			// Stage_b remains active, stage_a deactivated
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(2))
		})

		It("Should reset nodes on stage transition", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.ResetCalled).To(Equal(1))
		})

		It("Should maintain cross-sequence independence", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should handle multiple stages in sequence", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should give priority to the first-written transition when multiple conditions are true", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryActive := mock("entry_seq_active")
			// Two condition nodes in the active stage, both true
			condA := mock("condA")
			condB := mock("condB")
			entryStageA := mock("entry_seq_stage_a")
			entryStageB := mock("entry_seq_stage_b")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryActive.ActivateOnNext()
			// Both conditions output truthy and trigger their entry nodes
			condA.MarkOnNext("transition")
			condA.ParamTruthy["transition"] = true
			condB.MarkOnNext("transition")
			condB.ParamTruthy["transition"] = true
			entryStageA.ActivateOnNext()
			entryStageB.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_active").
				Node("condA").
				Node("condB").
				Node("entry_seq_stage_a").
				Node("entry_seq_stage_b").
				Node("A").
				Node("B").
				OneShot("trigger", "activate", "entry_seq_active", "input").
				// Both conditions trigger their respective entry nodes
				OneShot("condA", "transition", "entry_seq_stage_a", "input").
				OneShot("condB", "transition", "entry_seq_stage_b", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_active"}}).
				Sequence("seq", []testutil.StageSpec{
					// Active stage has both conditions; condA is first in the stratum
					{Key: "active", Strata: [][]string{
						{"condA", "condB"},
						{"entry_seq_stage_a", "entry_seq_stage_b"},
					}},
					{Key: "stage_a", Strata: [][]string{{"A"}}},
					{Key: "stage_b", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			// First transition (condA → stage_a) wins
			Expect(nodeA.NextCalled).To(Equal(1))
			// Second transition (condB → stage_b) should NOT fire
			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should skip later write statements after a transition fires", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryOn := mock("entry_seq_stage_on")
			toAbort := mock("to_abort")
			writeCmd := mock("write_ox_tpc_cmd")
			entryAbort := mock("entry_seq_stage_abort")
			abortNode := mock("abort_node")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryOn.ActivateOnNext()
			entryAbort.ActivateOnNext()

			toAbort.MarkOnNext("check")
			toAbort.ParamTruthy["check"] = true

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_stage_on").
				Node("to_abort").
				Node("write_ox_tpc_cmd").
				Node("entry_seq_stage_abort").
				Node("abort_node").
				OneShot("trigger", "activate", "entry_seq_stage_on", "input").
				OneShot("to_abort", "check", "entry_seq_stage_abort", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_stage_on"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "stage_on", Strata: [][]string{{"to_abort"}, {"entry_seq_stage_abort", "write_ox_tpc_cmd"}}},
					{Key: "stage_abort", Strata: [][]string{{"abort_node"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			// Transition should fire and move into abort stage.
			Expect(abortNode.NextCalled).To(Equal(1))
			// Statement after transition in the same stage pass should be skipped.
			Expect(writeCmd.NextCalled).To(Equal(0))
		})
	})

	Describe("Source-Order Transition Priority", func() {
		It("Should select the shallower entry when entries are at different strata", func(ctx SpecContext) {
			// Documents the pre-fix behavior: when entry nodes are at different
			// strata, the shallower one wins due to short-circuit, regardless of
			// source order. This is the bug we're fixing in the stratifier.
			trigger := mock("trigger")
			entryActive := mock("entry_seq_active")
			condA := mock("condA")
			condB := mock("condB")
			entryA := mock("entry_seq_stage_a")
			entryB := mock("entry_seq_stage_b")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryActive.ActivateOnNext()

			condA.MarkOnNext("check")
			condA.ParamTruthy["check"] = true
			condB.MarkOnNext("check")
			condB.ParamTruthy["check"] = true
			entryA.ActivateOnNext()
			entryB.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_active").
				Node("condA").
				Node("condB").
				Node("entry_seq_stage_a").
				Node("entry_seq_stage_b").
				Node("A").
				Node("B").
				OneShot("trigger", "activate", "entry_seq_active", "input").
				OneShot("condA", "check", "entry_seq_stage_a", "input").
				OneShot("condB", "check", "entry_seq_stage_b", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_active"}}).
				Sequence("seq", []testutil.StageSpec{
					// entry_b at stratum 1, entry_a at stratum 2 (different strata)
					{Key: "active", Strata: [][]string{
						{"condA", "condB"},
						{"entry_seq_stage_b"},
						{"entry_seq_stage_a"},
					}},
					{Key: "stage_a", Strata: [][]string{{"A"}}},
					{Key: "stage_b", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			// Short-circuit picks the shallower entry (stage_b at stratum 1)
			// even though stage_a should win by source order.
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeA.NextCalled).To(Equal(0))
		})

		It("Should respect source order when entries are at the same stratum", func(ctx SpecContext) {
			// Post-fix behavior: when the stratifier flattens entry nodes to the
			// same stratum, source order (position within the stratum) determines
			// which transition wins.
			trigger := mock("trigger")
			entryActive := mock("entry_seq_active")
			condA := mock("condA")
			condB := mock("condB")
			entryA := mock("entry_seq_stage_a")
			entryB := mock("entry_seq_stage_b")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryActive.ActivateOnNext()

			condA.MarkOnNext("check")
			condA.ParamTruthy["check"] = true
			condB.MarkOnNext("check")
			condB.ParamTruthy["check"] = true
			entryA.ActivateOnNext()
			entryB.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_active").
				Node("condA").
				Node("condB").
				Node("entry_seq_stage_a").
				Node("entry_seq_stage_b").
				Node("A").
				Node("B").
				OneShot("trigger", "activate", "entry_seq_active", "input").
				OneShot("condA", "check", "entry_seq_stage_a", "input").
				OneShot("condB", "check", "entry_seq_stage_b", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_active"}}).
				Sequence("seq", []testutil.StageSpec{
					// Both entries at the same stratum, stage_a first (source order)
					{Key: "active", Strata: [][]string{
						{"condA", "condB"},
						{"entry_seq_stage_a", "entry_seq_stage_b"},
					}},
					{Key: "stage_a", Strata: [][]string{{"A"}}},
					{Key: "stage_b", Strata: [][]string{{"B"}}},
				}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			// Source-order first entry (stage_a) wins
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})
	})

	Describe("Convergence Loop", func() {
		It("Should converge single transition", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should complete cascading transitions", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should stop when stable", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(2))
		})

		It("Should prevent infinite loop with max iterations", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(true).To(BeTrue())
		})

		It("Should detect transition during execution", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
		})
	})

	Describe("Error Handling", func() {
		It("Should pass errors to error handler", func(ctx SpecContext) {
			nodeA := mock("A")
			testErr := errors.New("test error")
			nodeA.ErrorOnNext(testErr)

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			handler := &MockErrorHandler{}
			s.SetErrorHandler(handler)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(handler.Errors).To(HaveLen(1))
			Expect(handler.Errors[0].NodeKey).To(Equal("A"))
			Expect(handler.Errors[0].Err).To(Equal(testErr))
		})

		It("Should continue execution after error", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")

			nodeA.OnNext = func(ctx node.Context) {
				ctx.ReportError(errors.New("error from A"))
				ctx.MarkChanged("output")
			}

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Edge("A", "output", "B", "input").
				Strata([][]string{{"A"}, {"B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should return normally after error", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeA.ErrorOnNext(errors.New("node error"))

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			// Should not panic
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
		})
	})

	Describe("Complex Graph Structures", func() {
		It("Should handle deep strata chain", func(ctx SpecContext) {
			for i := range 10 {
				m := mock(fmt.Sprintf("N%d", i))
				if i < 9 {
					m.MarkOnNext("output")
				}
			}

			builder := testutil.NewIRBuilder()
			for i := range 10 {
				builder.Node(fmt.Sprintf("N%d", i))
			}

			for i := range 9 {
				builder.Edge(fmt.Sprintf("N%d", i), "output", fmt.Sprintf("N%d", i+1), "input")
			}

			strata := make([][]string, 10)
			for i := range 10 {
				strata[i] = []string{fmt.Sprintf("N%d", i)}
			}

			prog := builder.Strata(strata).Build()
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			for i := range 10 {
				Expect(mocks[fmt.Sprintf("N%d", i)].NextCalled).To(Equal(1))
			}
		})

		It("Should handle mixed continuous and one-shot", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should handle global and staged mixed", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(globalNode.NextCalled).To(Equal(1))
			Expect(stagedNode.NextCalled).To(Equal(1))
		})

		It("Should handle multi-sequence with shared global", func(ctx SpecContext) {
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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(globalNode.NextCalled).To(Equal(1))
			Expect(staged1.NextCalled).To(Equal(1))
			Expect(staged2.NextCalled).To(Equal(1))
		})
	})

	Describe("Bang-Bang Authority Release Pattern", func() {
		It("Should execute yield stage authority nodes during start → stop → yield cascade", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryStart := mock("entry_bb_start")
			entryStop := mock("entry_bb_stop")
			entryYield := mock("entry_bb_yield")
			authHighCh1 := mock("auth_high_ch1")
			authHighCh2 := mock("auth_high_ch2")
			stopTrigger := mock("stop_trigger")
			writeCh1 := mock("write_ch1")
			writeCh2 := mock("write_ch2")
			yieldTrigger := mock("yield_trigger")
			authLowCh1 := mock("auth_low_ch1")
			authLowCh2 := mock("auth_low_ch2")
			reentryTrigger := mock("reentry_trigger")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryStart.ActivateOnNext()
			entryStop.ActivateOnNext()
			entryYield.ActivateOnNext()
			stopTrigger.MarkOnNext("to_stop")
			stopTrigger.ParamTruthy["to_stop"] = true
			yieldTrigger.MarkOnNext("to_yield")
			yieldTrigger.ParamTruthy["to_yield"] = true
			reentryTrigger.MarkOnNext("to_start")
			reentryTrigger.ParamTruthy["to_start"] = false

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_bb_start").
				Node("entry_bb_stop").
				Node("entry_bb_yield").
				Node("auth_high_ch1").
				Node("auth_high_ch2").
				Node("stop_trigger").
				Node("write_ch1").
				Node("write_ch2").
				Node("yield_trigger").
				Node("auth_low_ch1").
				Node("auth_low_ch2").
				Node("reentry_trigger").
				OneShot("trigger", "activate", "entry_bb_start", "input").
				OneShot("stop_trigger", "to_stop", "entry_bb_stop", "input").
				OneShot("yield_trigger", "to_yield", "entry_bb_yield", "input").
				OneShot("reentry_trigger", "to_start", "entry_bb_start", "input").
				Strata([][]string{
					{"trigger"},
					{"entry_bb_start", "entry_bb_stop", "entry_bb_yield"},
				}).
				Sequence("bb", []testutil.StageSpec{
					{Key: "start", Strata: [][]string{
						{"auth_high_ch1", "auth_high_ch2", "stop_trigger"},
						{"entry_bb_stop"},
					}},
					{Key: "stop", Strata: [][]string{
						{"write_ch1", "write_ch2", "yield_trigger"},
						{"entry_bb_yield"},
					}},
					{Key: "yield", Strata: [][]string{
						{"auth_low_ch1", "auth_low_ch2", "reentry_trigger"},
						{"entry_bb_start"},
					}},
				}).
				Build()

			s := build(prog)

			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(authHighCh1.NextCalled).To(Equal(1))
			Expect(authHighCh2.NextCalled).To(Equal(1))
			Expect(writeCh1.NextCalled).To(Equal(1))
			Expect(writeCh2.NextCalled).To(Equal(1))
			Expect(authLowCh1.NextCalled).To(Equal(1))
			Expect(authLowCh2.NextCalled).To(Equal(1))

			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(authLowCh1.NextCalled).To(Equal(2))
			Expect(authLowCh2.NextCalled).To(Equal(2))
			Expect(authHighCh1.NextCalled).To(Equal(1))
			Expect(authHighCh2.NextCalled).To(Equal(1))
			Expect(writeCh1.NextCalled).To(Equal(1))
			Expect(writeCh2.NextCalled).To(Equal(1))
		})

		It("Should re-execute start authority nodes when yield re-enters start via active trigger", func(ctx SpecContext) {
			// In the real scenario, stop → yield is timer-delayed (separate tick),
			// so re-entry only needs 2 transitions within one convergence loop
			// (stop → yield, yield → start), which fits within maxConvergenceIterations = 3.
			trigger := mock("trigger")
			entryStart := mock("entry_bb_start")
			entryStop := mock("entry_bb_stop")
			entryYield := mock("entry_bb_yield")
			authHighCh1 := mock("auth_high_ch1")
			authHighCh2 := mock("auth_high_ch2")
			stopTrigger := mock("stop_trigger")
			mock("write_ch1")
			mock("write_ch2")
			yieldTrigger := mock("yield_trigger")
			authLowCh1 := mock("auth_low_ch1")
			authLowCh2 := mock("auth_low_ch2")
			reentryTrigger := mock("reentry_trigger")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryStart.ActivateOnNext()
			entryStop.ActivateOnNext()
			entryYield.ActivateOnNext()
			reentryTrigger.MarkOnNext("to_start")
			reentryTrigger.ParamTruthy["to_start"] = true

			// stop_trigger fires on tick 1 (cascading start → stop).
			// yield_trigger fires on tick 2 (cascading stop → yield → start re-entry).
			// This mirrors the real behavior where wait{duration=250ms} delays
			// the stop → yield transition to a later tick.
			stopCallCount := 0
			stopTrigger.OnNext = func(nCtx node.Context) {
				stopCallCount++
				if stopCallCount == 1 {
					nCtx.MarkChanged("to_stop")
				}
			}
			stopTrigger.ParamTruthy["to_stop"] = true

			yieldCallCount := 0
			yieldTrigger.OnNext = func(nCtx node.Context) {
				yieldCallCount++
				if yieldCallCount == 1 {
					nCtx.MarkChanged("to_yield")
				}
			}
			yieldTrigger.ParamTruthy["to_yield"] = true

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_bb_start").
				Node("entry_bb_stop").
				Node("entry_bb_yield").
				Node("auth_high_ch1").
				Node("auth_high_ch2").
				Node("stop_trigger").
				Node("write_ch1").
				Node("write_ch2").
				Node("yield_trigger").
				Node("auth_low_ch1").
				Node("auth_low_ch2").
				Node("reentry_trigger").
				OneShot("trigger", "activate", "entry_bb_start", "input").
				OneShot("stop_trigger", "to_stop", "entry_bb_stop", "input").
				OneShot("yield_trigger", "to_yield", "entry_bb_yield", "input").
				OneShot("reentry_trigger", "to_start", "entry_bb_start", "input").
				Strata([][]string{
					{"trigger"},
					{"entry_bb_start", "entry_bb_stop", "entry_bb_yield"},
				}).
				Sequence("bb", []testutil.StageSpec{
					{Key: "start", Strata: [][]string{
						{"auth_high_ch1", "auth_high_ch2", "stop_trigger"},
						{"entry_bb_stop"},
					}},
					{Key: "stop", Strata: [][]string{
						{"write_ch1", "write_ch2", "yield_trigger"},
						{"entry_bb_yield"},
					}},
					{Key: "yield", Strata: [][]string{
						{"auth_low_ch1", "auth_low_ch2", "reentry_trigger"},
						{"entry_bb_start"},
					}},
				}).
				Build()

			s := build(prog)

			// Tick 1: trigger → start → stop (stop_trigger fires once).
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(authHighCh1.NextCalled).To(Equal(1))
			Expect(authHighCh2.NextCalled).To(Equal(1))

			// Tick 2: stop is active. yield_trigger fires → stop → yield.
			// reentry_trigger fires → yield → start (re-entry).
			// This is 2 transitions within maxConvergenceIterations = 3.
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(authLowCh1.NextCalled).To(Equal(1))
			Expect(authLowCh2.NextCalled).To(Equal(1))
			Expect(authHighCh1.NextCalled).To(Equal(2))
			Expect(authHighCh2.NextCalled).To(Equal(2))
		})
	})

	Describe("Self-Changed (Wait in Flow Chain)", func() {
		It("Should allow a self-changed node in a higher stratum to keep executing", func(ctx SpecContext) {
			// IR: comparison (stratum 0) => wait (stratum 1) -> entry_seq_next (stratum 2)
			// The comparison fires a one-shot to the wait. The wait needs multiple
			// timer ticks to complete. Without self-changed, the wait only gets one
			// tick (the one-shot fires once), then starves.
			comparison := mock("comparison")
			wait := mock("wait")
			entryNext := mock("entry_seq_next")

			comparison.MarkOnNext("output")
			comparison.ParamTruthy["output"] = true

			// Wait simulates a 1s timer: calls MarkSelfChanged while timing,
			// then calls MarkChanged("output") when done.
			waitStarted := false
			waitStartElapsed := telem.TimeSpan(0)
			wait.OnNext = func(nCtx node.Context) {
				if nCtx.Reason != node.ReasonTimerTick {
					return
				}
				if !waitStarted {
					waitStarted = true
					waitStartElapsed = nCtx.Elapsed
					nCtx.MarkSelfChanged()
					return
				}
				if nCtx.Elapsed-waitStartElapsed < telem.Second {
					nCtx.MarkSelfChanged()
					return
				}
				nCtx.MarkChanged("output")
			}
			wait.ParamTruthy["output"] = true

			entryNext.ActivateOnNext()

			trigger := mock("trigger")
			entry := mock("entry_seq_first")
			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("entry_seq_first").
				Node("comparison").
				Node("wait").
				Node("entry_seq_next").
				OneShot("trigger", "activate", "entry_seq_first", "input").
				OneShot("comparison", "output", "wait", "input").
				Edge("wait", "output", "entry_seq_next", "input").
				Strata([][]string{{"trigger"}, {"entry_seq_first"}}).
				Sequence("seq", []testutil.StageSpec{
					{Key: "first", Strata: [][]string{{"comparison"}, {"wait"}, {"entry_seq_next"}}},
					{Key: "next", Strata: [][]string{}},
				}).
				Build()

			s := scheduler.New(prog, nodes, 0)

			// Tick 0: trigger fires, stage activates, comparison fires one-shot to wait,
			// wait starts timing and calls MarkSelfChanged
			s.Next(ctx, 0, node.ReasonTimerTick)
			Expect(wait.NextCalled).To(Equal(1))
			Expect(entryNext.NextCalled).To(Equal(0))

			// Tick at 500ms: wait should execute (self-changed), but not fire yet
			s.Next(ctx, 500*telem.Millisecond, node.ReasonTimerTick)
			Expect(wait.NextCalled).To(Equal(2))
			Expect(entryNext.NextCalled).To(Equal(0))

			// Tick at 1s: wait fires, propagates to entry_seq_next
			s.Next(ctx, telem.Second, node.ReasonTimerTick)
			Expect(wait.NextCalled).To(Equal(3))
			Expect(entryNext.NextCalled).To(Equal(1))
		})

		It("Should not execute a self-changed node if it stops calling MarkSelfChanged", func(ctx SpecContext) {
			trigger := mock("trigger")
			nodeA := mock("A")

			trigger.MarkOnNext("output")
			trigger.ParamTruthy["output"] = true

			callCount := 0
			nodeA.OnNext = func(nCtx node.Context) {
				callCount++
				if callCount <= 2 {
					nCtx.MarkSelfChanged()
				}
			}

			prog := testutil.NewIRBuilder().
				Node("trigger").
				Node("A").
				OneShot("trigger", "output", "A", "input").
				Strata([][]string{{"trigger"}, {"A"}}).
				Build()

			s := build(prog)

			// Tick 0: trigger fires one-shot to A, A executes and self-changes
			s.Next(ctx, 0, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))

			// Tick 1: A executes via self-changed (one-shot already fired)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(2))

			// Tick 2: A executes via self-changed (callCount=2, still calls MarkSelfChanged)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(3))

			// Tick 3: A should NOT execute (callCount=3, stopped calling MarkSelfChanged,
			// one-shot already fired, not in changed set)
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(3))
		})

		It("Should clear self-changed when a node is reset via stage transition", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryA := mock("entry_seq_stage_a")
			entryB := mock("entry_seq_stage_b")
			nodeA := mock("A")
			nodeB := mock("B")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryA.ActivateOnNext()
			entryB.ActivateOnNext()

			// A calls MarkSelfChanged on first execution
			aCallCount := 0
			nodeA.OnNext = func(nCtx node.Context) {
				aCallCount++
				if aCallCount == 1 {
					nCtx.MarkSelfChanged()
					nCtx.MarkChanged("to_b")
				}
			}
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

			s := scheduler.New(prog, nodes, 0)

			// Tick 0: trigger → stage_a activates, A runs and self-changes + transitions to stage_b
			s.Next(ctx, 0, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))

			// Tick 1: stage_b is active, A's self-changed should have been cleared by reset
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(2))
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle zero elapsed time", func(ctx SpecContext) {
			nodeA := mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			s.Next(ctx, 0, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeA.ElapsedValues[0]).To(Equal(telem.TimeSpan(0)))
		})

		It("Should handle self-loop", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeA.MarkOnNext("output")

			prog := testutil.NewIRBuilder().
				Node("A").
				Edge("A", "output", "A", "input").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should handle empty sequence", func(ctx SpecContext) {
			mock("A")

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Sequence("empty_seq", []testutil.StageSpec{}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(mocks["A"].NextCalled).To(Equal(1))
		})
	})
	Describe("NextDeadline", func() {
		It("Should return TimeSpanMax when no node sets a deadline", func(ctx SpecContext) {
			nodeA := NewMockNode()
			nodeA.MarkOnNext("output")
			mocks["A"] = nodeA
			nodes["A"] = nodeA

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.TimeSpanMax))
		})
		It("Should return the minimum deadline across nodes", func(ctx SpecContext) {
			nodeA := NewMockNode()
			nodeA.OnNext = func(ctx node.Context) {
				if ctx.SetDeadline != nil {
					ctx.SetDeadline(3 * telem.Second)
				}
			}
			mocks["A"] = nodeA
			nodes["A"] = nodeA

			nodeB := NewMockNode()
			nodeB.OnNext = func(ctx node.Context) {
				if ctx.SetDeadline != nil {
					ctx.SetDeadline(1 * telem.Second)
				}
			}
			mocks["B"] = nodeB
			nodes["B"] = nodeB

			prog := testutil.NewIRBuilder().
				Node("A").
				Node("B").
				Strata([][]string{{"A", "B"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.Second))
		})
		It("Should reset to max between cycles", func(ctx SpecContext) {
			call := 0
			nodeA := NewMockNode()
			nodeA.OnNext = func(ctx node.Context) {
				call++
				if call == 1 && ctx.SetDeadline != nil {
					ctx.SetDeadline(telem.Second)
				}
			}
			mocks["A"] = nodeA
			nodes["A"] = nodeA

			prog := testutil.NewIRBuilder().
				Node("A").
				Strata([][]string{{"A"}}).
				Build()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.Second))

			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.TimeSpanMax))
		})
		It("Should track deadlines from stage nodes", func(ctx SpecContext) {
			nodeA := NewMockNode()
			nodeA.OnNext = func(ctx node.Context) {
				if ctx.SetDeadline != nil {
					ctx.SetDeadline(2 * telem.Second)
				}
			}
			mocks["A"] = nodeA
			nodes["A"] = nodeA

			trigger := NewMockNode()
			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			mocks["trigger"] = trigger
			nodes["trigger"] = trigger

			entry := NewMockNode()
			entry.ActivateOnNext()
			mocks["entry_seq_stage"] = entry
			nodes["entry_seq_stage"] = entry

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
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(2 * telem.Second))
		})
	})

	Describe("Flow Steps", func() {
		// flowSeqProg builds IR for a stageless sequence with flow steps.
		// Each step has one node. One-shot edges auto-wire each node's output
		// to the next step's entry node. Entry nodes for steps > 0 are placed
		// in a stratum after the data nodes.
		flowSeqProg := func(seqKey string, stepKeys []string) ir.IR {
			var (
				irNodes  ir.Nodes
				irEdges  ir.Edges
				steps    ir.Steps
				stratum0 []string
				stratum1 []string
			)
			for i, sk := range stepKeys {
				nodeKey := "node_" + sk
				entryKey := fmt.Sprintf("entry_%s_%s", seqKey, sk)
				irNodes = append(irNodes, ir.Node{Key: nodeKey})
				irNodes = append(irNodes, ir.Node{Key: entryKey})
				steps = append(steps, ir.Step{
					Key:  sk,
					Flow: &ir.Flow{Nodes: []string{nodeKey}},
				})
				stratum0 = append(stratum0, nodeKey)
				if i > 0 {
					stratum1 = append(stratum1, entryKey)
				}
				if i+1 < len(stepKeys) {
					nextEntry := fmt.Sprintf("entry_%s_%s", seqKey, stepKeys[i+1])
					irEdges = append(irEdges, ir.Edge{
						Source: ir.Handle{Node: nodeKey, Param: "output"},
						Target: ir.Handle{Node: nextEntry, Param: "activate"},
						Kind:   ir.EdgeKindOneShot,
					})
				}
			}
			// Trigger and first entry in global strata.
			triggerKey := "trigger_" + seqKey
			firstEntry := fmt.Sprintf("entry_%s_%s", seqKey, stepKeys[0])
			irNodes = append(irNodes, ir.Node{Key: triggerKey})
			irEdges = append(irEdges, ir.Edge{
				Source: ir.Handle{Node: triggerKey, Param: "activate"},
				Target: ir.Handle{Node: firstEntry, Param: "input"},
				Kind:   ir.EdgeKindOneShot,
			})

			seqStrata := ir.Strata{stratum0}
			if len(stratum1) > 0 {
				seqStrata = append(seqStrata, stratum1)
			}
			return ir.IR{
				Nodes: irNodes,
				Edges: irEdges,
				Root: ir.Stage{
					Strata: ir.Strata{{triggerKey}, {firstEntry}},
					Sequences: ir.Sequences{{
						Key:    seqKey,
						Steps:  steps,
						Strata: seqStrata,
					}},
				},
			}
		}

		It("Should execute a single flow step", func(ctx SpecContext) {
			prog := flowSeqProg("seq", []string{"s0"})

			trigger := mock("trigger_seq")
			entry := mock("entry_seq_s0")
			n := mock("node_s0")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entry.ActivateOnNext()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(n.NextCalled).To(Equal(1))
			Expect(n.ResetCalled).To(Equal(1))
		})

		It("Should cascade two consecutive truthy writes on the same tick", func(ctx SpecContext) {
			prog := flowSeqProg("seq", []string{"s0", "s1"})

			trigger := mock("trigger_seq")
			mock("entry_seq_s0")
			mock("entry_seq_s1")
			w0 := mock("node_s0")
			w1 := mock("node_s1")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			mocks["entry_seq_s0"].ActivateOnNext()
			w0.MarkOnNext("output")
			w0.ParamTruthy["output"] = true
			mocks["entry_seq_s1"].ActivateOnNext()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(w0.NextCalled).To(Equal(1))
			Expect(w1.NextCalled).To(Equal(1))
		})

		It("Should cascade three consecutive truthy writes on the same tick", func(ctx SpecContext) {
			prog := flowSeqProg("seq", []string{"s0", "s1", "s2"})

			trigger := mock("trigger_seq")
			mock("entry_seq_s0")
			mock("entry_seq_s1")
			mock("entry_seq_s2")
			w0 := mock("node_s0")
			w1 := mock("node_s1")
			w2 := mock("node_s2")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			mocks["entry_seq_s0"].ActivateOnNext()
			w0.MarkOnNext("output")
			w0.ParamTruthy["output"] = true
			mocks["entry_seq_s1"].ActivateOnNext()
			w1.MarkOnNext("output")
			w1.ParamTruthy["output"] = true
			mocks["entry_seq_s2"].ActivateOnNext()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(w0.NextCalled).To(Equal(1))
			Expect(w1.NextCalled).To(Equal(1))
			Expect(w2.NextCalled).To(Equal(1))
		})

		It("Should cascade five consecutive truthy writes on the same tick", func(ctx SpecContext) {
			keys := []string{"s0", "s1", "s2", "s3", "s4"}
			prog := flowSeqProg("seq", keys)

			trigger := mock("trigger_seq")
			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true

			ws := make([]*MockNode, len(keys))
			for i, k := range keys {
				entryKey := fmt.Sprintf("entry_seq_%s", k)
				mock(entryKey)
				mocks[entryKey].ActivateOnNext()
				ws[i] = mock("node_" + k)
				ws[i].MarkOnNext("output")
				ws[i].ParamTruthy["output"] = true
			}

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			for i, w := range ws {
				Expect(w.NextCalled).To(Equal(1), fmt.Sprintf("step %d should execute", i))
			}
		})

		It("Should block at a falsy gate and advance when it becomes truthy", func(ctx SpecContext) {
			prog := flowSeqProg("seq", []string{"s0", "s1"})

			trigger := mock("trigger_seq")
			mock("entry_seq_s0")
			mock("entry_seq_s1")
			gate := mock("node_s0")
			after := mock("node_s1")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			mocks["entry_seq_s0"].ActivateOnNext()
			gate.MarkOnNext("output")
			gate.ParamTruthy["output"] = false
			mocks["entry_seq_s1"].ActivateOnNext()

			s := build(prog)

			// Tick 1: gate is falsy, after should not execute.
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(gate.NextCalled).To(Equal(1))
			Expect(after.NextCalled).To(Equal(0))

			// Tick 2: gate still falsy. selfChanged keeps it executing.
			s.MarkNodeChanged("node_s0")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(gate.NextCalled).To(Equal(2))
			Expect(after.NextCalled).To(Equal(0))

			// Tick 3: gate becomes truthy. Advances to s1.
			gate.ParamTruthy["output"] = true
			s.MarkNodeChanged("node_s0")
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(after.NextCalled).To(Equal(1))
		})

		It("Should not execute nodes from inactive flow steps", func(ctx SpecContext) {
			prog := flowSeqProg("seq", []string{"s0", "s1", "s2"})

			trigger := mock("trigger_seq")
			mock("entry_seq_s0")
			mock("entry_seq_s1")
			mock("entry_seq_s2")
			w0 := mock("node_s0")
			w1 := mock("node_s1")
			w2 := mock("node_s2")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			mocks["entry_seq_s0"].ActivateOnNext()
			// s0 is falsy (gate). s1 and s2 should never execute.
			w0.MarkOnNext("output")
			w0.ParamTruthy["output"] = false
			mocks["entry_seq_s1"].ActivateOnNext()
			mocks["entry_seq_s2"].ActivateOnNext()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(w0.NextCalled).To(Equal(1))
			Expect(w1.NextCalled).To(Equal(0))
			Expect(w2.NextCalled).To(Equal(0))
		})

		It("Should reset flow nodes when entering the step", func(ctx SpecContext) {
			prog := flowSeqProg("seq", []string{"s0"})

			trigger := mock("trigger_seq")
			mock("entry_seq_s0")
			n := mock("node_s0")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			mocks["entry_seq_s0"].ActivateOnNext()

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(n.ResetCalled).To(Equal(1))
		})

		It("Should advance from a stage step to a flow step via => next", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryStage := mock("entry_main_stage_a")
			entryFlow := mock("entry_main_flow_b")
			stageNode := mock("stage_node")
			flowNode := mock("flow_node")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryStage.ActivateOnNext()
			stageNode.MarkOnNext("output")
			stageNode.ParamTruthy["output"] = true
			entryFlow.ActivateOnNext()

			prog := ir.IR{
				Nodes: ir.Nodes{
					{Key: "trigger"}, {Key: "entry_main_stage_a"},
					{Key: "entry_main_flow_b"}, {Key: "stage_node"}, {Key: "flow_node"},
				},
				Edges: ir.Edges{
					{Source: ir.Handle{Node: "trigger", Param: "activate"},
						Target: ir.Handle{Node: "entry_main_stage_a", Param: "input"},
						Kind:   ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "stage_node", Param: "output"},
						Target: ir.Handle{Node: "entry_main_flow_b", Param: "activate"},
						Kind:   ir.EdgeKindOneShot},
				},
				Root: ir.Stage{
					Strata: ir.Strata{{"trigger"}, {"entry_main_stage_a"}},
					Sequences: ir.Sequences{{
						Key: "main",
						Steps: ir.Steps{
							{Key: "stage_a", Stage: &ir.Stage{
								Key: "stage_a", Nodes: []string{"stage_node"},
								Strata: ir.Strata{{"stage_node"}, {"entry_main_flow_b"}},
							}},
							{Key: "flow_b", Flow: &ir.Flow{Nodes: []string{"flow_node"}}},
						},
						Strata: ir.Strata{{"flow_node"}},
					}},
				},
			}

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(stageNode.NextCalled).To(Equal(1))
			Expect(flowNode.NextCalled).To(Equal(1))
		})

		It("Should advance from a flow step to a stage step", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryFlow := mock("entry_main_flow_a")
			entryStage := mock("entry_main_stage_b")
			flowNode := mock("flow_node")
			stageNode := mock("stage_node")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryFlow.ActivateOnNext()
			flowNode.MarkOnNext("output")
			flowNode.ParamTruthy["output"] = true
			entryStage.ActivateOnNext()

			prog := ir.IR{
				Nodes: ir.Nodes{
					{Key: "trigger"}, {Key: "entry_main_flow_a"},
					{Key: "entry_main_stage_b"}, {Key: "flow_node"}, {Key: "stage_node"},
				},
				Edges: ir.Edges{
					{Source: ir.Handle{Node: "trigger", Param: "activate"},
						Target: ir.Handle{Node: "entry_main_flow_a", Param: "input"},
						Kind:   ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "flow_node", Param: "output"},
						Target: ir.Handle{Node: "entry_main_stage_b", Param: "activate"},
						Kind:   ir.EdgeKindOneShot},
				},
				Root: ir.Stage{
					Strata: ir.Strata{{"trigger"}, {"entry_main_flow_a"}},
					Sequences: ir.Sequences{{
						Key: "main",
						Steps: ir.Steps{
							{Key: "flow_a", Flow: &ir.Flow{Nodes: []string{"flow_node"}}},
							{Key: "stage_b", Stage: &ir.Stage{
								Key: "stage_b", Nodes: []string{"stage_node"},
								Strata: ir.Strata{{"stage_node"}},
							}},
						},
						Strata: ir.Strata{{"flow_node"}, {"entry_main_stage_b"}},
					}},
				},
			}

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(flowNode.NextCalled).To(Equal(1))
			Expect(stageNode.NextCalled).To(Equal(1))
		})

		It("Should support stage => name jumping to a named sibling", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryA := mock("entry_seq_a")
			mock("entry_seq_b")
			entryC := mock("entry_seq_c")
			nodeA := mock("node_a")
			nodeB := mock("node_b")
			nodeC := mock("node_c")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryA.ActivateOnNext()
			// Stage A: nodeA triggers entryC (skip B)
			nodeA.MarkOnNext("output")
			nodeA.ParamTruthy["output"] = true
			entryC.ActivateOnNext()

			prog := ir.IR{
				Nodes: ir.Nodes{
					{Key: "trigger"}, {Key: "entry_seq_a"}, {Key: "entry_seq_b"},
					{Key: "entry_seq_c"}, {Key: "node_a"}, {Key: "node_b"}, {Key: "node_c"},
				},
				Edges: ir.Edges{
					{Source: ir.Handle{Node: "trigger", Param: "activate"},
						Target: ir.Handle{Node: "entry_seq_a", Param: "input"},
						Kind:   ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "node_a", Param: "output"},
						Target: ir.Handle{Node: "entry_seq_c", Param: "activate"},
						Kind:   ir.EdgeKindOneShot},
				},
				Root: ir.Stage{
					Strata: ir.Strata{{"trigger"}, {"entry_seq_a"}},
					Sequences: ir.Sequences{{
						Key: "seq",
						Steps: ir.Steps{
							{Key: "a", Stage: &ir.Stage{
								Key: "a", Nodes: []string{"node_a"},
								Strata: ir.Strata{{"node_a"}, {"entry_seq_c"}},
							}},
							{Key: "b", Stage: &ir.Stage{
								Key: "b", Nodes: []string{"node_b"},
								Strata: ir.Strata{{"node_b"}},
							}},
							{Key: "c", Stage: &ir.Stage{
								Key: "c", Nodes: []string{"node_c"},
								Strata: ir.Strata{{"node_c"}},
							}},
						},
					}},
				},
			}

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should reset nodes when re-entering a step via backward jump", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryA := mock("entry_seq_a")
			entryB := mock("entry_seq_b")
			nodeA := mock("node_a")
			nodeB := mock("node_b")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryA.ActivateOnNext()
			// First tick: A transitions to B
			nodeA.MarkOnNext("output")
			nodeA.ParamTruthy["output"] = true
			entryB.ActivateOnNext()

			prog := ir.IR{
				Nodes: ir.Nodes{
					{Key: "trigger"}, {Key: "entry_seq_a"}, {Key: "entry_seq_b"},
					{Key: "node_a"}, {Key: "node_b"},
				},
				Edges: ir.Edges{
					{Source: ir.Handle{Node: "trigger", Param: "activate"},
						Target: ir.Handle{Node: "entry_seq_a", Param: "input"},
						Kind:   ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "node_a", Param: "output"},
						Target: ir.Handle{Node: "entry_seq_b", Param: "activate"},
						Kind:   ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "node_b", Param: "output"},
						Target: ir.Handle{Node: "entry_seq_a", Param: "activate"},
						Kind:   ir.EdgeKindOneShot},
				},
				Root: ir.Stage{
					Strata: ir.Strata{{"trigger"}, {"entry_seq_a"}},
					Sequences: ir.Sequences{{
						Key: "seq",
						Steps: ir.Steps{
							{Key: "a", Stage: &ir.Stage{
								Key: "a", Nodes: []string{"node_a"},
								Strata: ir.Strata{{"node_a"}, {"entry_seq_b"}},
							}},
							{Key: "b", Stage: &ir.Stage{
								Key: "b", Nodes: []string{"node_b"},
								Strata: ir.Strata{{"node_b"}, {"entry_seq_a"}},
							}},
						},
					}},
				},
			}

			s := build(prog)
			// Tick 1: trigger -> A -> B
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.ResetCalled).To(Equal(1))
			Expect(nodeB.ResetCalled).To(Equal(1))

			// Now make B jump back to A
			nodeB.MarkOnNext("output")
			nodeB.ParamTruthy["output"] = true
			s.MarkNodeChanged("node_b")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)

			// A should be reset again (re-entry)
			Expect(nodeA.ResetCalled).To(Equal(2))
		})

		It("Should support mixed stage, flow, stage pattern", func(ctx SpecContext) {
			trigger := mock("trigger")
			entryPress := mock("entry_main_press")
			entryWrite := mock("entry_main_write")
			entryVent := mock("entry_main_vent")
			pressNode := mock("press_node")
			writeNode := mock("write_node")
			ventNode := mock("vent_node")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			entryPress.ActivateOnNext()
			pressNode.MarkOnNext("output")
			pressNode.ParamTruthy["output"] = true
			entryWrite.ActivateOnNext()
			writeNode.MarkOnNext("output")
			writeNode.ParamTruthy["output"] = true
			entryVent.ActivateOnNext()

			prog := ir.IR{
				Nodes: ir.Nodes{
					{Key: "trigger"}, {Key: "entry_main_press"},
					{Key: "entry_main_write"}, {Key: "entry_main_vent"},
					{Key: "press_node"}, {Key: "write_node"}, {Key: "vent_node"},
				},
				Edges: ir.Edges{
					{Source: ir.Handle{Node: "trigger", Param: "activate"},
						Target: ir.Handle{Node: "entry_main_press", Param: "input"},
						Kind:   ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "press_node", Param: "output"},
						Target: ir.Handle{Node: "entry_main_write", Param: "activate"},
						Kind:   ir.EdgeKindOneShot},
					{Source: ir.Handle{Node: "write_node", Param: "output"},
						Target: ir.Handle{Node: "entry_main_vent", Param: "activate"},
						Kind:   ir.EdgeKindOneShot},
				},
				Root: ir.Stage{
					Strata: ir.Strata{{"trigger"}, {"entry_main_press"}},
					Sequences: ir.Sequences{{
						Key: "main",
						Steps: ir.Steps{
							{Key: "press", Stage: &ir.Stage{
								Key: "press", Nodes: []string{"press_node"},
								Strata: ir.Strata{{"press_node"}, {"entry_main_write"}},
							}},
							{Key: "write", Flow: &ir.Flow{Nodes: []string{"write_node"}}},
							{Key: "vent", Stage: &ir.Stage{
								Key: "vent", Nodes: []string{"vent_node"},
								Strata: ir.Strata{{"vent_node"}},
							}},
						},
						Strata: ir.Strata{{"write_node"}, {"entry_main_vent"}},
					}},
				},
			}

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(pressNode.NextCalled).To(Equal(1))
			Expect(writeNode.NextCalled).To(Equal(1))
			Expect(ventNode.NextCalled).To(Equal(1))
		})

		It("Should report deadline from a wait node in a flow step", func(ctx SpecContext) {
			prog := flowSeqProg("seq", []string{"s0"})

			trigger := mock("trigger_seq")
			mock("entry_seq_s0")
			waitNode := mock("node_s0")

			trigger.MarkOnNext("activate")
			trigger.ParamTruthy["activate"] = true
			mocks["entry_seq_s0"].ActivateOnNext()
			waitNode.OnNext = func(ctx node.Context) {
				ctx.SetDeadline(5 * telem.Second)
				ctx.MarkSelfChanged()
			}

			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			Expect(s.NextDeadline()).To(Equal(5 * telem.Second))
		})
	})
})
