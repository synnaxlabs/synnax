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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// MockNode is a configurable runtime node used across scheduler tests.
type MockNode struct {
	ParamTruthy   set.Set[string]
	OnNext        func(node.Context)
	ElapsedValues []telem.TimeSpan
	NextCalled    int
	ResetCalled   int
}

func NewMockNode() *MockNode {
	return &MockNode{ParamTruthy: make(set.Set[string])}
}

func (m *MockNode) Next(ctx node.Context) {
	m.NextCalled++
	m.ElapsedValues = append(m.ElapsedValues, ctx.Elapsed)
	// Auto-propagate: any param currently marked as truthy is also
	// announced via MarkChanged. This mirrors how real nodes advertise
	// output updates and lets scheduler tests drive both conditional
	// edges and gated-scope activations from the ParamTruthy fixture.
	for param := range m.ParamTruthy {
		ctx.MarkChanged(param)
	}
	if m.OnNext != nil {
		m.OnNext(ctx)
	}
}

func (m *MockNode) Reset() { m.ResetCalled++ }

func (m *MockNode) IsOutputTruthy(param string) bool {
	return m.ParamTruthy.Contains(param)
}

// MarkOnNext configures the node to mark the named output param as changed
// each time Next runs.
func (m *MockNode) MarkOnNext(param string) {
	m.OnNext = func(ctx node.Context) { ctx.MarkChanged(param) }
}

// MockErrorHandler collects scheduler-reported errors for assertion.
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

// ----- IR construction helpers -----

// noderef builds a Member wrapping a NodeRef keyed by nodeKey.
func noderef(nodeKey string) ir.Member {
	return ir.Member{Key: nodeKey, NodeRef: &ir.NodeRef{Key: nodeKey}}
}

// scopeMember wraps a nested scope as a member whose key equals the
// scope's own key.
func scopeMember(s ir.Scope) ir.Member {
	return ir.Member{Key: s.Key, Scope: &s}
}

// phase builds a Phase from the given member list.
func phase(members ...ir.Member) ir.Phase { return ir.Phase{Members: members} }

// parallelScope composes a parallel+gated Scope from one or more phases.
func parallelScope(key string, phases ...ir.Phase) ir.Scope {
	return ir.Scope{
		Key:      key,
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessGated,
		Phases:   phases,
	}
}

// sequentialScope composes a sequential+gated Scope with the given
// ordered members and transitions.
func sequentialScope(key string, members []ir.Member, transitions ...ir.Transition) ir.Scope {
	return ir.Scope{
		Key:         key,
		Mode:        ir.ScopeModeSequential,
		Liveness:    ir.LivenessGated,
		Members:     members,
		Transitions: transitions,
	}
}

// rootScope wraps top-level members in a parallel+always-live root scope
// with a single catch-all phase. Matches the shape the analyzer emits
// before stratification.
func rootScope(members ...ir.Member) ir.Scope {
	root := ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
	}
	if len(members) > 0 {
		root.Phases = []ir.Phase{{Members: members}}
	}
	return root
}

// rootWithPhases builds a parallel+always-live root with explicit phases.
func rootWithPhases(phases ...ir.Phase) ir.Scope {
	return ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
		Phases:   phases,
	}
}

// continuousEdge is a helper for building a non-conditional dataflow edge.
func continuousEdge(src, srcParam, tgt, tgtParam string) ir.Edge {
	return ir.Edge{
		Source: ir.Handle{Node: src, Param: srcParam},
		Target: ir.Handle{Node: tgt, Param: tgtParam},
		Kind:   ir.EdgeKindContinuous,
	}
}

// conditionalEdge builds a conditional dataflow edge that only fires when
// the source output param is truthy.
func conditionalEdge(src, srcParam, tgt, tgtParam string) ir.Edge {
	return ir.Edge{
		Source: ir.Handle{Node: src, Param: srcParam},
		Target: ir.Handle{Node: tgt, Param: tgtParam},
		Kind:   ir.EdgeKindConditional,
	}
}

// memberKeyTarget builds a TransitionTarget that jumps to the named
// sibling member.
func memberKeyTarget(key string) ir.TransitionTarget {
	k := key
	return ir.TransitionTarget{MemberKey: &k}
}

// exitTarget builds a TransitionTarget that exits the sequence, yielding
// to the parent scope.
func exitTarget() ir.TransitionTarget {
	exit := true
	return ir.TransitionTarget{Exit: &exit}
}

// programOf builds an IR program with the given nodes, edges, and root
// scope. Nodes are created as minimal ir.Node records keyed only.
func programOf(nodeKeys []string, edges []ir.Edge, root ir.Scope) ir.IR {
	irNodes := make([]ir.Node, 0, len(nodeKeys))
	for _, k := range nodeKeys {
		irNodes = append(irNodes, ir.Node{Key: k})
	}
	return ir.IR{Nodes: irNodes, Edges: edges, Root: root}
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

	Describe("Construction", func() {
		It("Should run Next on an empty program without panicking", func(ctx SpecContext) {
			s := build(ir.IR{})
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
		})

		It("Should execute every member of a single-phase root scope", func(ctx SpecContext) {
			mock("A")
			mock("B")
			mock("C")
			prog := programOf(
				[]string{"A", "B", "C"},
				nil,
				rootScope(noderef("A"), noderef("B"), noderef("C")),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(mocks["A"].NextCalled).To(Equal(1))
			Expect(mocks["B"].NextCalled).To(Equal(1))
			Expect(mocks["C"].NextCalled).To(Equal(1))
		})
	})

	Describe("Phase-based execution", func() {
		It("Should execute phase-0 members unconditionally each cycle", func(ctx SpecContext) {
			nodeA := mock("A")
			prog := programOf([]string{"A"}, nil, rootScope(noderef("A")))
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(3))
		})

		It("Should skip phase-N members without an incoming change", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			prog := programOf(
				[]string{"A", "B"},
				[]ir.Edge{continuousEdge("A", "output", "B", "input")},
				rootWithPhases(phase(noderef("A")), phase(noderef("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should propagate continuous edges to downstream members", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.MarkOnNext("output")
			prog := programOf(
				[]string{"A", "B"},
				[]ir.Edge{continuousEdge("A", "output", "B", "input")},
				rootWithPhases(phase(noderef("A")), phase(noderef("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should gate conditional edges on source output truthiness", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.MarkOnNext("output")
			// Output is not truthy — B must not fire.
			prog := programOf(
				[]string{"A", "B"},
				[]ir.Edge{conditionalEdge("A", "output", "B", "input")},
				rootWithPhases(phase(noderef("A")), phase(noderef("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))

			// Flip A's output truthy; now the conditional edge fires.
			nodeA.ParamTruthy.Add("output")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should replay a self-changed node on the next cycle", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeA.OnNext = func(ctx node.Context) { ctx.MarkSelfChanged() }
			prog := programOf(
				[]string{"A", "B"},
				nil,
				rootWithPhases(
					phase(noderef("marker")),
					phase(noderef("A")),
				),
			)
			// "marker" is a phase-0 node to force the walk to run phase 1
			// on each cycle via the selfChanged replay; without it the
			// phase-0 loop would trivially re-execute A via the phase-0
			// fast path, masking the behavior we want to test.
			mock("marker")
			_ = prog
			// Re-build the program with A in phase-1 only.
			prog = programOf(
				[]string{"marker", "A"},
				nil,
				rootWithPhases(
					phase(noderef("marker")),
					phase(noderef("A")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			// A is not in phase 0 and nothing forwarded a change, so it
			// shouldn't have executed.
			Expect(nodeA.NextCalled).To(Equal(0))
			// Seed A as externally changed so it runs this cycle; then
			// it self-marks and should replay on the subsequent cycle.
			s.MarkNodeChanged("A")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(2))
		})
	})

	Describe("Context pass-through", func() {
		It("Should pass elapsed time to node context", func(ctx SpecContext) {
			nodeA := mock("A")
			prog := programOf([]string{"A"}, nil, rootScope(noderef("A")))
			s := build(prog)
			s.Next(ctx, 5*telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 10*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.ElapsedValues).To(Equal([]telem.TimeSpan{
				5 * telem.Microsecond, 10 * telem.Microsecond,
			}))
		})

		It("Should expose the minimum deadline set by any node", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.OnNext = func(ctx node.Context) { ctx.SetDeadline(10 * telem.Microsecond) }
			nodeB.OnNext = func(ctx node.Context) { ctx.SetDeadline(3 * telem.Microsecond) }
			prog := programOf(
				[]string{"A", "B"},
				nil,
				rootScope(noderef("A"), noderef("B")),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(3 * telem.Microsecond))
		})

		It("Should route errors through the configured handler", func(ctx SpecContext) {
			nodeA := mock("A")
			targetErr := errors.New("boom")
			nodeA.OnNext = func(ctx node.Context) { ctx.ReportError(targetErr) }
			prog := programOf([]string{"A"}, nil, rootScope(noderef("A")))
			s := build(prog)
			h := &MockErrorHandler{}
			s.SetErrorHandler(h)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(h.Errors).To(HaveLen(1))
			Expect(h.Errors[0].NodeKey).To(Equal("A"))
			Expect(h.Errors[0].Err).To(Equal(targetErr))
		})
	})

	Describe("Gated scope activation", func() {
		It("Should not execute a gated scope before its activation fires", func(ctx SpecContext) {
			trigger := mock("trigger")
			stage := mock("stage_node")
			act := ir.Handle{Node: "trigger", Param: "output"}
			gated := parallelScope("stage", phase(noderef("stage_node")))
			gated.Activation = &act
			prog := programOf(
				[]string{"trigger", "stage_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(gated)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(trigger.NextCalled).To(Equal(1))
			Expect(stage.NextCalled).To(Equal(0))
		})

		It("Should activate a gated scope once its activation handle is truthy", func(ctx SpecContext) {
			trigger := mock("trigger")
			stage := mock("stage_node")
			trigger.ParamTruthy.Add("output")
			act := ir.Handle{Node: "trigger", Param: "output"}
			gated := parallelScope("stage", phase(noderef("stage_node")))
			gated.Activation = &act
			prog := programOf(
				[]string{"trigger", "stage_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(gated)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(stage.NextCalled).To(Equal(1))
			// Reset called once on activation.
			Expect(stage.ResetCalled).To(Equal(1))
			// Stays active in subsequent cycles without re-activating.
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(stage.NextCalled).To(Equal(2))
			Expect(stage.ResetCalled).To(Equal(1))
		})
	})

	Describe("Sequential scope transitions", func() {
		buildTwoStepSeq := func(onNode string) ir.IR {
			// sequence main { stage first; stage second; } with a
			// transition first->second driven by `onNode`'s output.
			first := parallelScope("first", phase(noderef("first_node")))
			second := parallelScope("second", phase(noderef("second_node")))
			main := sequentialScope("main", []ir.Member{
				{Key: "first", Scope: &first},
				{Key: "second", Scope: &second},
			}, ir.Transition{
				On:     ir.Handle{Node: onNode, Param: "output"},
				Target: memberKeyTarget("second"),
			})
			// Module-scope trigger to activate the sequence.
			trigger := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &trigger
			main.Liveness = ir.LivenessGated
			return programOf(
				[]string{"trigger", "first_node", "second_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(main)),
			)
		}

		It("Should advance the active member when a transition's handle fires", func(ctx SpecContext) {
			trigger := mock("trigger")
			firstNode := mock("first_node")
			secondNode := mock("second_node")
			trigger.ParamTruthy.Add("output")
			prog := buildTwoStepSeq("first_node")
			s := build(prog)

			// Cycle 1: trigger fires, main activates at `first`; first_node
			// runs. first_node's output is not yet truthy, so no transition.
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(firstNode.NextCalled).To(Equal(1))
			Expect(secondNode.NextCalled).To(Equal(0))

			// Cycle 2: first_node becomes truthy, the transition fires; the
			// sequence advances to `second` in the same cycle.
			firstNode.ParamTruthy.Add("output")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(firstNode.NextCalled).To(Equal(2))
			Expect(secondNode.NextCalled).To(Equal(1))
			// The transition cascaded within the same cycle: second_node
			// reset once on activation.
			Expect(secondNode.ResetCalled).To(Equal(1))
		})

		It("Should exit the sequence when target is exit", func(ctx SpecContext) {
			trigger := mock("trigger")
			firstNode := mock("first_node")
			// Model a one-shot rising-edge trigger: fires on cycle 1
			// (so main activates), then releases on later cycles so the
			// activation handle isn't re-satisfying after exit.
			trigger.ParamTruthy.Add("output")
			cycleCount := 0
			trigger.OnNext = func(ctx node.Context) {
				cycleCount++
				if cycleCount > 1 {
					trigger.ParamTruthy.Remove("output")
				}
			}
			first := parallelScope("first", phase(noderef("first_node")))
			main := sequentialScope("main",
				[]ir.Member{{Key: "first", Scope: &first}},
				ir.Transition{
					On:     ir.Handle{Node: "first_node", Param: "output"},
					Target: exitTarget(),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]string{"trigger", "first_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(main)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(firstNode.NextCalled).To(Equal(1))

			// Trip exit transition on cycle 2. The exit deactivates main;
			// trigger has already been released, so no re-activation.
			firstNode.ParamTruthy.Add("output")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			countAtExit := firstNode.NextCalled
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(firstNode.NextCalled).To(Equal(countAtExit))
		})

		It("Should honor source-order when multiple transitions become truthy simultaneously", func(ctx SpecContext) {
			trigger := mock("trigger")
			firstNode := mock("first_node")
			trigger.ParamTruthy.Add("output")
			// Two transitions from first, in source order: first jumps to
			// `a`, the second would jump to `b`. Both are truthy at the
			// same cycle.
			first := parallelScope("first", phase(noderef("first_node")))
			aScope := parallelScope("a", phase(noderef("a_node")))
			bScope := parallelScope("b", phase(noderef("b_node")))
			main := sequentialScope("main",
				[]ir.Member{
					{Key: "first", Scope: &first},
					{Key: "a", Scope: &aScope},
					{Key: "b", Scope: &bScope},
				},
				ir.Transition{
					On:     ir.Handle{Node: "first_node", Param: "output"},
					Target: memberKeyTarget("a"),
				},
				ir.Transition{
					On:     ir.Handle{Node: "first_node", Param: "output"},
					Target: memberKeyTarget("b"),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			mock("a_node")
			mock("b_node")
			prog := programOf(
				[]string{"trigger", "first_node", "a_node", "b_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(main)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			firstNode.ParamTruthy.Add("output")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			// First-match-wins: `a` activated, `b` did not.
			Expect(mocks["a_node"].NextCalled).To(Equal(1))
			Expect(mocks["b_node"].NextCalled).To(Equal(0))
		})

		It("Should cascade multiple transitions within a single cycle", func(ctx SpecContext) {
			// three-step sequence where step 1 immediately transitions to
			// step 2, step 2 immediately to step 3, all in one cycle.
			trigger := mock("trigger")
			s1 := mock("s1")
			s2 := mock("s2")
			s3 := mock("s3")
			trigger.ParamTruthy.Add("output")
			s1.ParamTruthy.Add("output")
			s2.ParamTruthy.Add("output")
			mkStep := func(key, nodeKey string) ir.Scope {
				return parallelScope(key, phase(noderef(nodeKey)))
			}
			scope1 := mkStep("s1", "s1")
			scope2 := mkStep("s2", "s2")
			scope3 := mkStep("s3", "s3")
			main := sequentialScope("main",
				[]ir.Member{
					{Key: "s1", Scope: &scope1},
					{Key: "s2", Scope: &scope2},
					{Key: "s3", Scope: &scope3},
				},
				ir.Transition{
					On:     ir.Handle{Node: "s1", Param: "output"},
					Target: memberKeyTarget("s2"),
				},
				ir.Transition{
					On:     ir.Handle{Node: "s2", Param: "output"},
					Target: memberKeyTarget("s3"),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]string{"trigger", "s1", "s2", "s3"},
				nil,
				rootScope(noderef("trigger"), scopeMember(main)),
			)
			s := build(prog)
			// Cycle 1 activates s1 and runs it. In that same cycle the
			// transition fires (because s1's output is truthy already),
			// s2 runs and triggers, s3 runs.
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s1.NextCalled).To(Equal(1))
			Expect(s2.NextCalled).To(Equal(1))
			Expect(s3.NextCalled).To(Equal(1))
		})
	})

	Describe("Activation cascading & reset", func() {
		It("Should reset member nodes on activation", func(ctx SpecContext) {
			trigger := mock("trigger")
			stageNode := mock("n")
			trigger.ParamTruthy.Add("output")
			act := ir.Handle{Node: "trigger", Param: "output"}
			stage := parallelScope("stage", phase(noderef("n")))
			stage.Activation = &act
			prog := programOf(
				[]string{"trigger", "n"},
				nil,
				rootScope(noderef("trigger"), scopeMember(stage)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(stageNode.ResetCalled).To(Equal(1))
		})

		It("Should cascade reset into nested gated scopes on activation", func(ctx SpecContext) {
			trigger := mock("trigger")
			trigger.ParamTruthy.Add("output")
			inner := mock("inner")
			nested := parallelScope("nested", phase(noderef("inner")))
			outer := parallelScope("outer", phase(scopeMember(nested)))
			act := ir.Handle{Node: "trigger", Param: "output"}
			outer.Activation = &act
			prog := programOf(
				[]string{"trigger", "inner"},
				nil,
				rootScope(noderef("trigger"), scopeMember(outer)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(inner.ResetCalled).To(Equal(1))
			Expect(inner.NextCalled).To(Equal(1))
		})
	})

	Describe("External change injection", func() {
		It("Should execute a higher-phase node when marked from outside", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			prog := programOf(
				[]string{"A", "B"},
				nil,
				rootWithPhases(phase(noderef("A")), phase(noderef("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(0))

			s.MarkNodeChanged("B")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(2))
			Expect(nodeB.NextCalled).To(Equal(1))
		})
	})

	Describe("Change propagation", func() {
		It("Should fire only the edge whose source param was marked", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.OnNext = func(ctx node.Context) { ctx.MarkChanged("x") }
			prog := programOf(
				[]string{"A", "B", "C"},
				[]ir.Edge{
					continuousEdge("A", "x", "B", "in"),
					continuousEdge("A", "y", "C", "in"),
				},
				rootWithPhases(
					phase(noderef("A")),
					phase(noderef("B"), noderef("C")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(0))
		})

		It("Should fan out a single change to multiple downstream members", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.MarkOnNext("output")
			prog := programOf(
				[]string{"A", "B", "C"},
				[]ir.Edge{
					continuousEdge("A", "output", "B", "in"),
					continuousEdge("A", "output", "C", "in"),
				},
				rootWithPhases(
					phase(noderef("A")),
					phase(noderef("B"), noderef("C")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should run a join node once when several inputs fire in the same cycle", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.MarkOnNext("output")
			nodeB.MarkOnNext("output")
			prog := programOf(
				[]string{"A", "B", "C"},
				[]ir.Edge{
					continuousEdge("A", "output", "C", "a"),
					continuousEdge("B", "output", "C", "b"),
				},
				rootWithPhases(
					phase(noderef("A"), noderef("B")),
					phase(noderef("C")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should propagate a change through a chain in a single cycle", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.MarkOnNext("output")
			nodeB.MarkOnNext("output")
			prog := programOf(
				[]string{"A", "B", "C"},
				[]ir.Edge{
					continuousEdge("A", "output", "B", "in"),
					continuousEdge("B", "output", "C", "in"),
				},
				rootWithPhases(
					phase(noderef("A")),
					phase(noderef("B")),
					phase(noderef("C")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should execute a diamond graph's sink exactly once", func(ctx SpecContext) {
			mock("A").MarkOnNext("output")
			mock("B").MarkOnNext("output")
			mock("C").MarkOnNext("output")
			nodeD := mock("D")
			prog := programOf(
				[]string{"A", "B", "C", "D"},
				[]ir.Edge{
					continuousEdge("A", "output", "B", "in"),
					continuousEdge("A", "output", "C", "in"),
					continuousEdge("B", "output", "D", "a"),
					continuousEdge("C", "output", "D", "b"),
				},
				rootWithPhases(
					phase(noderef("A")),
					phase(noderef("B"), noderef("C")),
					phase(noderef("D")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeD.NextCalled).To(Equal(1))
		})

		It("Should not propagate when no edge targets the source's changed param", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.MarkOnNext("output")
			prog := programOf(
				[]string{"A", "B"},
				nil,
				rootWithPhases(phase(noderef("A")), phase(noderef("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})
	})

	Describe("Conditional edge semantics", func() {
		It("Should fire every cycle while the source stays truthy", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.ParamTruthy.Add("output")
			prog := programOf(
				[]string{"A", "B"},
				[]ir.Edge{conditionalEdge("A", "output", "B", "in")},
				rootWithPhases(phase(noderef("A")), phase(noderef("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(3))
		})

		It("Should stop firing when the source transitions from truthy to falsy", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.ParamTruthy.Add("output")
			prog := programOf(
				[]string{"A", "B"},
				[]ir.Edge{conditionalEdge("A", "output", "B", "in")},
				rootWithPhases(phase(noderef("A")), phase(noderef("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))

			nodeA.ParamTruthy.Remove("output")
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should keep continuous edges unaffected by source truthiness", func(ctx SpecContext) {
			// Source is not marked truthy; a continuous edge must still
			// propagate the change.
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.OnNext = func(ctx node.Context) { ctx.MarkChanged("output") }
			prog := programOf(
				[]string{"A", "B"},
				[]ir.Edge{continuousEdge("A", "output", "B", "in")},
				rootWithPhases(phase(noderef("A")), phase(noderef("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should fire each conditional edge independently per param truthiness", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.OnNext = func(ctx node.Context) {
				ctx.MarkChanged("x")
				ctx.MarkChanged("y")
			}
			nodeA.ParamTruthy.Add("x")
			// "y" is not truthy — its conditional edge must not fire.
			prog := programOf(
				[]string{"A", "B", "C"},
				[]ir.Edge{
					conditionalEdge("A", "x", "B", "in"),
					conditionalEdge("A", "y", "C", "in"),
				},
				rootWithPhases(
					phase(noderef("A")),
					phase(noderef("B"), noderef("C")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(0))
		})
	})

	Describe("Self-changed replay", func() {
		It("Should stop replaying once the node no longer marks itself", func(ctx SpecContext) {
			nodeA := mock("A")
			callCount := 0
			nodeA.OnNext = func(c node.Context) {
				callCount++
				if callCount <= 2 {
					c.MarkSelfChanged()
				}
			}
			prog := programOf(
				[]string{"marker", "A"},
				nil,
				rootWithPhases(phase(noderef("marker")), phase(noderef("A"))),
			)
			mock("marker")
			s := build(prog)
			s.MarkNodeChanged("A")
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)   // initial run, marks self
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick) // replay, marks self
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick) // replay, stops marking
			s.Next(ctx, 4*telem.Microsecond, node.ReasonTimerTick) // should not replay
			Expect(nodeA.NextCalled).To(Equal(3))
		})

		It("Should clear self-changed on deactivation", func(ctx SpecContext) {
			// Set up a sequential scope with two members. The first member
			// contains a node that self-marks; after a transition, the
			// old member is deactivated and the self-changed should be
			// cleared.
			trigger := mock("trigger")
			stageNode := mock("stage_node")
			mock("second_node")
			triggerFired := false
			stageNode.OnNext = func(c node.Context) {
				c.MarkSelfChanged()
				if !triggerFired {
					triggerFired = true
					c.MarkChanged("done")
				}
			}
			stageNode.ParamTruthy.Add("done")

			first := parallelScope("first", phase(noderef("stage_node")))
			second := parallelScope("second", phase(noderef("second_node")))
			main := sequentialScope("main",
				[]ir.Member{{Key: "first", Scope: &first}, {Key: "second", Scope: &second}},
				ir.Transition{
					On:     ir.Handle{Node: "stage_node", Param: "done"},
					Target: memberKeyTarget("second"),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			trigger.ParamTruthy.Add("output")

			prog := programOf(
				[]string{"trigger", "stage_node", "second_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(main)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			// stage_node ran once during activation, self-marked, then
			// transition fired. On the next cycle, with "first"
			// deactivated, stage_node's selfChanged should have been
			// cleared, so it should not re-run.
			prior := stageNode.NextCalled
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(stageNode.NextCalled).To(Equal(prior))
		})
	})

	Describe("NextDeadline", func() {
		It("Should return TimeSpanMax when no node sets a deadline", func(ctx SpecContext) {
			mock("A")
			prog := programOf([]string{"A"}, nil, rootScope(noderef("A")))
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.TimeSpanMax))
		})

		It("Should reset to TimeSpanMax between cycles when no node re-sets it", func(ctx SpecContext) {
			nodeA := mock("A")
			call := 0
			nodeA.OnNext = func(c node.Context) {
				call++
				if call == 1 {
					c.SetDeadline(telem.Second)
				}
			}
			prog := programOf([]string{"A"}, nil, rootScope(noderef("A")))
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.Second))
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.TimeSpanMax))
		})

		It("Should track deadlines from nodes inside an active gated scope", func(ctx SpecContext) {
			trigger := mock("trigger")
			stageNode := mock("stage_node")
			trigger.ParamTruthy.Add("output")
			stageNode.OnNext = func(c node.Context) { c.SetDeadline(2 * telem.Second) }
			act := ir.Handle{Node: "trigger", Param: "output"}
			gated := parallelScope("stage", phase(noderef("stage_node")))
			gated.Activation = &act
			prog := programOf(
				[]string{"trigger", "stage_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(gated)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(2 * telem.Second))
		})
	})

	Describe("Error handling", func() {
		It("Should continue executing remaining members after a node reports an error", func(ctx SpecContext) {
			errA := errors.New("boom-A")
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.OnNext = func(c node.Context) { c.ReportError(errA) }
			prog := programOf(
				[]string{"A", "B", "C"},
				nil,
				rootWithPhases(phase(noderef("A"), noderef("B"), noderef("C"))),
			)
			s := build(prog)
			h := &MockErrorHandler{}
			s.SetErrorHandler(h)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
			Expect(h.Errors).To(HaveLen(1))
			Expect(h.Errors[0].NodeKey).To(Equal("A"))
		})

		It("Should accumulate multiple errors across a cycle", func(ctx SpecContext) {
			errA := errors.New("boom-A")
			errB := errors.New("boom-B")
			mock("A").OnNext = func(c node.Context) { c.ReportError(errA) }
			mock("B").OnNext = func(c node.Context) { c.ReportError(errB) }
			prog := programOf(
				[]string{"A", "B"},
				nil,
				rootScope(noderef("A"), noderef("B")),
			)
			s := build(prog)
			h := &MockErrorHandler{}
			s.SetErrorHandler(h)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(h.Errors).To(HaveLen(2))
		})

		It("Should swallow errors silently when no handler is configured", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeA.OnNext = func(c node.Context) { c.ReportError(errors.New("dropped")) }
			prog := programOf([]string{"A"}, nil, rootScope(noderef("A")))
			s := build(prog)
			Expect(func() {
				s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			}).ToNot(Panic())
			Expect(nodeA.NextCalled).To(Equal(1))
		})
	})

	Describe("Edge cases", func() {
		It("Should accept zero elapsed time", func(ctx SpecContext) {
			nodeA := mock("A")
			prog := programOf([]string{"A"}, nil, rootScope(noderef("A")))
			s := build(prog)
			s.Next(ctx, 0, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeA.ElapsedValues[0]).To(Equal(telem.TimeSpan(0)))
		})

		It("Should pass through ReasonChannelInput", func(ctx SpecContext) {
			var received node.RunReason
			nodeA := mock("A")
			nodeA.OnNext = func(c node.Context) { received = c.Reason }
			prog := programOf([]string{"A"}, nil, rootScope(noderef("A")))
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonChannelInput)
			Expect(received).To(Equal(node.ReasonChannelInput))
		})

		It("Should tolerate a self-loop edge in phase 0", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeA.MarkOnNext("output")
			prog := programOf(
				[]string{"A"},
				[]ir.Edge{continuousEdge("A", "output", "A", "in")},
				rootScope(noderef("A")),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			// Phase 0 is unconditional, so A ran once. The self-loop adds
			// A to `changed`, but `changed` is cleared at cycle end, and
			// there is no higher phase to re-run into.
			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should tolerate an empty sequential scope", func(ctx SpecContext) {
			trigger := mock("trigger")
			trigger.ParamTruthy.Add("output")
			main := ir.Scope{
				Key:      "main",
				Mode:     ir.ScopeModeSequential,
				Liveness: ir.LivenessGated,
			}
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]string{"trigger"},
				nil,
				rootScope(noderef("trigger"), scopeMember(main)),
			)
			s := build(prog)
			Expect(func() {
				s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			}).ToNot(Panic())
			Expect(trigger.NextCalled).To(Equal(1))
		})
	})

	Describe("Complex graph and sequence interactions", func() {
		It("Should not re-activate an already-active gated scope on a subsequent cycle", func(ctx SpecContext) {
			trigger := mock("trigger")
			stageNode := mock("stage_node")
			trigger.ParamTruthy.Add("output")
			act := ir.Handle{Node: "trigger", Param: "output"}
			gated := parallelScope("stage", phase(noderef("stage_node")))
			gated.Activation = &act
			prog := programOf(
				[]string{"trigger", "stage_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(gated)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(stageNode.ResetCalled).To(Equal(1))
			// Trigger stays truthy but the scope is already active; no
			// additional Reset should be issued.
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(stageNode.ResetCalled).To(Equal(1))
		})

		It("Should keep two top-level sequences independent", func(ctx SpecContext) {
			triggerA := mock("trigger_a")
			_ = mock("trigger_b")
			a := mock("A")
			b := mock("B")
			triggerA.ParamTruthy.Add("output")
			// trigger_b stays falsy — only `a` should activate.
			stageA := parallelScope("stage_a", phase(noderef("A")))
			actA := ir.Handle{Node: "trigger_a", Param: "output"}
			stageA.Activation = &actA
			stageB := parallelScope("stage_b", phase(noderef("B")))
			actB := ir.Handle{Node: "trigger_b", Param: "output"}
			stageB.Activation = &actB
			prog := programOf(
				[]string{"trigger_a", "trigger_b", "A", "B"},
				nil,
				rootScope(
					noderef("trigger_a"),
					noderef("trigger_b"),
					scopeMember(stageA),
					scopeMember(stageB),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(a.NextCalled).To(Equal(1))
			Expect(b.NextCalled).To(Equal(0))
		})

		It("Should mix continuous and conditional edges in a single graph", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.MarkOnNext("data")
			nodeA.OnNext = func(c node.Context) {
				c.MarkChanged("data")
				c.MarkChanged("trigger")
			}
			nodeA.ParamTruthy.Add("trigger")
			prog := programOf(
				[]string{"A", "B", "C"},
				[]ir.Edge{
					continuousEdge("A", "data", "B", "in"),
					conditionalEdge("A", "trigger", "C", "in"),
				},
				rootWithPhases(
					phase(noderef("A")),
					phase(noderef("B"), noderef("C")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should reset sequence members when reactivated after an exit", func(ctx SpecContext) {
			// One-shot trigger: cycle 1 activates main, exit transition
			// fires on cycle 2, cycle 3 re-triggers, main re-activates.
			trigger := mock("trigger")
			firstNode := mock("first_node")
			cycle := 0
			trigger.ParamTruthy.Add("output")
			trigger.OnNext = func(c node.Context) {
				cycle++
				// Release after cycle 1, re-assert on cycle 3.
				if cycle == 2 {
					trigger.ParamTruthy.Remove("output")
				}
				if cycle == 3 {
					trigger.ParamTruthy.Add("output")
				}
			}
			firstNode.ParamTruthy.Add("output") // transition fires immediately
			first := parallelScope("first", phase(noderef("first_node")))
			main := sequentialScope("main",
				[]ir.Member{{Key: "first", Scope: &first}},
				ir.Transition{
					On:     ir.Handle{Node: "first_node", Param: "output"},
					Target: exitTarget(),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]string{"trigger", "first_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(main)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)   // activate + run + exit
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick) // trigger released, no action
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick) // trigger reasserted, main re-activates, runs, exits
			// Two activations ⇒ two Reset calls on first_node.
			Expect(firstNode.ResetCalled).To(Equal(2))
		})
	})

	Describe("Convergence bound", func() {
		It("Should not infinite-loop if a transition's on-handle stays truthy and owner stays active", func(ctx SpecContext) {
			// Construct a sequence where the transition targets the same
			// member that contains its on-handle. Under normal semantics
			// the transition would fire, reactivate the same member, and
			// re-fire. The convergence bound must keep this from looping
			// forever within a cycle.
			trigger := mock("trigger")
			loopNode := mock("loop_node")
			trigger.ParamTruthy.Add("output")
			loopNode.ParamTruthy.Add("output")
			loop := parallelScope("loop", phase(noderef("loop_node")))
			main := sequentialScope("main",
				[]ir.Member{{Key: "loop", Scope: &loop}},
				ir.Transition{
					On:     ir.Handle{Node: "loop_node", Param: "output"},
					Target: memberKeyTarget("loop"),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]string{"trigger", "loop_node"},
				nil,
				rootScope(noderef("trigger"), scopeMember(main)),
			)
			s := build(prog)
			done := make(chan struct{})
			go func() {
				s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
				close(done)
			}()
			Eventually(done).Should(BeClosed())
			// Bounded iterations: loop_node executed more than once
			// (cascade occurred), but not an unbounded number.
			Expect(loopNode.NextCalled).To(BeNumerically(">=", 1))
			Expect(loopNode.NextCalled).To(BeNumerically("<=", 10))
		})
	})
})
