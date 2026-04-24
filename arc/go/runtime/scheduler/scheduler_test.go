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
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// MockNode is a configurable runtime node used across scheduler tests.
// It deals exclusively in ordinals — output names live in ir.Node.Outputs
// and are declared at the IR construction layer (programOf + node).
// Tests construct mocks via the Describe-level mock helper, which takes
// a per-ordinal truthy slice that drives both IsOutputTruthy and the
// auto-mark loop in Next.
type MockNode struct {
	// OutputTruthy[i] reports whether output ordinal i is truthy. Drives
	// IsOutputTruthy and (unless SuppressAutoMark is set) the auto-mark
	// loop in Next. Length need not match the IR's declared output count
	// — out-of-range ordinals are treated as non-truthy.
	OutputTruthy []bool
	// SuppressAutoMark disables the default behavior of calling
	// MarkChanged for every currently-truthy output on each Next. Tests
	// that want to model a node whose output stays truthy across cycles
	// but only announces a change on specific cycles should set this and
	// drive MarkChanged manually from OnNext.
	SuppressAutoMark bool
	OnNext           func(node.Context)
	ElapsedValues    []telem.TimeSpan
	NextCalled       int
	ResetCalled      int
}

func NewMockNode() *MockNode { return &MockNode{} }

// SetTruthy marks the given ordinal as truthy, growing OutputTruthy as
// needed. Returns the receiver for chaining.
func (m *MockNode) SetTruthy(ordinal int) *MockNode {
	for ordinal >= len(m.OutputTruthy) {
		m.OutputTruthy = append(m.OutputTruthy, false)
	}
	m.OutputTruthy[ordinal] = true
	return m
}

func (m *MockNode) Next(ctx node.Context) {
	m.NextCalled++
	m.ElapsedValues = append(m.ElapsedValues, ctx.Elapsed)
	if !m.SuppressAutoMark {
		for i, truthy := range m.OutputTruthy {
			if truthy {
				ctx.MarkChanged(i)
			}
		}
	}
	if m.OnNext != nil {
		m.OnNext(ctx)
	}
}

func (m *MockNode) Reset() { m.ResetCalled++ }

func (m *MockNode) IsOutputTruthy(idx int) bool {
	if idx < 0 || idx >= len(m.OutputTruthy) {
		return false
	}
	return m.OutputTruthy[idx]
}

// markOnNext returns an OnNext callback that calls MarkChanged for the
// given ordinal each time Next runs. Replaces the symbolic
// MarkOnNext("name") form — the ordinal comes from the test's IR
// declaration.
func markOnNext(ordinal int) func(node.Context) {
	return func(ctx node.Context) { ctx.MarkChanged(ordinal) }
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

// stratum builds a Members slice from the given member list, representing
// one execution stratum within a parallel scope.
func stratum(members ...ir.Member) ir.Members { return members }

// parallelScope composes a parallel+gated Scope from one or more strata.
func parallelScope(key string, strata ...ir.Members) ir.Scope {
	return ir.Scope{
		Key:      key,
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessGated,
		Strata:   strata,
	}
}

// alwaysScope composes a parallel+always-live Scope from one or more strata.
// Use for nested scopes that run whenever their parent runs, and for
// anonymous top-level scopes that auto-start at boot.
func alwaysScope(key string, strata ...ir.Members) ir.Scope {
	return ir.Scope{
		Key:      key,
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
		Strata:   strata,
	}
}

// sequentialScope composes a sequential+gated Scope with the given
// ordered steps and transitions.
func sequentialScope(key string, steps []ir.Member, transitions ...ir.Transition) ir.Scope {
	return ir.Scope{
		Key:         key,
		Mode:        ir.ScopeModeSequential,
		Liveness:    ir.LivenessGated,
		Steps:       steps,
		Transitions: transitions,
	}
}

// rootScope wraps top-level members in a parallel+always-live root scope
// with a single catch-all stratum. Matches the shape the analyzer emits
// before stratification.
func rootScope(members ...ir.Member) ir.Scope {
	root := ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
	}
	if len(members) > 0 {
		root.Strata = []ir.Members{members}
	}
	return root
}

// rootWithStrata builds a parallel+always-live root with explicit strata.
func rootWithStrata(strata ...ir.Members) ir.Scope {
	return ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
		Strata:   strata,
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

// stepKeyTarget builds a transition target key that jumps to the named
// sibling step.
func stepKeyTarget(key string) *string {
	return new(key)
}

// exitTarget returns the nil target key that exits the sequence, yielding
// to the parent scope.
func exitTarget() *string {
	return nil
}

// irNode builds an ir.Node with the given key and ordered output names.
// The IR owns output names; ordinals used by the runtime mock are this
// list's positions. Pass no names for a node with no outputs.
func irNode(key string, outputs ...string) ir.Node {
	n := ir.Node{Key: key}
	if len(outputs) > 0 {
		n.Outputs = make(types.Params, len(outputs))
		for i, name := range outputs {
			n.Outputs[i] = types.Param{Name: name}
		}
	}
	return n
}

// programOf builds an IR program from the given nodes, edges, and root
// scope. Output names are declared per node via the node helper — the
// scheduler reads them exclusively from ir.Node.Outputs.
func programOf(nodes []ir.Node, edges []ir.Edge, root ir.Scope) ir.IR {
	return ir.IR{Nodes: nodes, Edges: edges, Root: root}
}

var _ = Describe("Scheduler", func() {
	var (
		nodes map[string]node.Node
		mocks map[string]*MockNode
	)

	// mock registers a MockNode under key with per-ordinal initial truthy
	// values. Pass no values for a silent mock; pass true/false per
	// declared output ordinal otherwise. The corresponding ir.Node and
	// its output names are declared separately at the IR layer (programOf
	// + node) — the mock is name-agnostic.
	mock := func(key string, truthy ...bool) *MockNode {
		m := NewMockNode()
		if len(truthy) > 0 {
			m.OutputTruthy = truthy
		}
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
				[]ir.Node{irNode("A"), irNode("B"), irNode("C")},
				nil,
				rootScope(ir.NodeMember("A"), ir.NodeMember("B"), ir.NodeMember("C")),
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
			prog := programOf([]ir.Node{irNode("A")}, nil, rootScope(ir.NodeMember("A")))
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
				[]ir.Node{irNode("A", "output"), irNode("B")},
				[]ir.Edge{continuousEdge("A", "output", "B", "input")},
				rootWithStrata(stratum(ir.NodeMember("A")), stratum(ir.NodeMember("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})

		It("Should propagate continuous edges to downstream members", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.OnNext = markOnNext(0)
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B")},
				[]ir.Edge{continuousEdge("A", "output", "B", "input")},
				rootWithStrata(stratum(ir.NodeMember("A")), stratum(ir.NodeMember("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should gate conditional edges on source output truthiness", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.OnNext = markOnNext(0)
			// Output is not truthy — B must not fire.
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B")},
				[]ir.Edge{conditionalEdge("A", "output", "B", "input")},
				rootWithStrata(stratum(ir.NodeMember("A")), stratum(ir.NodeMember("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))

			// Flip A's output truthy; now the conditional edge fires.
			nodeA.SetTruthy(0)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should replay a self-changed node on the next cycle", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeA.OnNext = func(ctx node.Context) { ctx.MarkSelfChanged() }
			// trigger sits in phase 0 with an edge into A, but only
			// announces a change on its second cycle. Once A has run
			// once it self-marks and should replay every subsequent
			// cycle without further upstream activity.
			trigger := mock("trigger")
			trigger.SuppressAutoMark = true
			triggerCalls := 0
			trigger.OnNext = func(c node.Context) {
				triggerCalls++
				if triggerCalls == 2 {
					c.MarkChanged(0)
				}
			}
			prog := programOf(
				[]ir.Node{irNode("trigger", "kick"), irNode("A")},
				[]ir.Edge{continuousEdge("trigger", "kick", "A", "in")},
				rootWithStrata(stratum(ir.NodeMember("trigger")), stratum(ir.NodeMember("A"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			// Trigger ran but didn't mark; A is in phase 1 with no
			// change pending, so it shouldn't have executed.
			Expect(nodeA.NextCalled).To(Equal(0))
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(2))
		})
	})

	Describe("Context pass-through", func() {
		It("Should pass elapsed time to node context", func(ctx SpecContext) {
			nodeA := mock("A")
			prog := programOf([]ir.Node{irNode("A")}, nil, rootScope(ir.NodeMember("A")))
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
				[]ir.Node{irNode("A"), irNode("B")},
				nil,
				rootScope(ir.NodeMember("A"), ir.NodeMember("B")),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(3 * telem.Microsecond))
		})

		It("Should route errors through the configured handler", func(ctx SpecContext) {
			nodeA := mock("A")
			targetErr := errors.New("boom")
			nodeA.OnNext = func(ctx node.Context) { ctx.ReportError(targetErr) }
			prog := programOf([]ir.Node{irNode("A")}, nil, rootScope(ir.NodeMember("A")))
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
			gated := parallelScope("stage", stratum(ir.NodeMember("stage_node")))
			gated.Activation = &act
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("stage_node")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(gated)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(trigger.NextCalled).To(Equal(1))
			Expect(stage.NextCalled).To(Equal(0))
		})

		It("Should activate a gated scope once its activation handle is truthy", func(ctx SpecContext) {
			mock("trigger", true)
			stage := mock("stage_node")
			act := ir.Handle{Node: "trigger", Param: "output"}
			gated := parallelScope("stage", stratum(ir.NodeMember("stage_node")))
			gated.Activation = &act
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("stage_node")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(gated)),
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
			first := parallelScope("first", stratum(ir.NodeMember("first_node")))
			second := parallelScope("second", stratum(ir.NodeMember("second_node")))
			main := sequentialScope("main", []ir.Member{
				{Scope: &first},
				{Scope: &second},
			}, ir.Transition{
				On:        ir.Handle{Node: onNode, Param: "output"},
				TargetKey: stepKeyTarget("second"),
			})
			// Module-scope trigger to activate the sequence.
			trigger := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &trigger
			main.Liveness = ir.LivenessGated
			return programOf(
				[]ir.Node{
					irNode("trigger", "output"),
					irNode("first_node", "output"),
					irNode("second_node"),
				},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(main)),
			)
		}

		It("Should advance the active member when a transition's handle fires", func(ctx SpecContext) {
			mock("trigger", true)
			firstNode := mock("first_node")
			secondNode := mock("second_node")
			prog := buildTwoStepSeq("first_node")
			s := build(prog)

			// Cycle 1: trigger fires, main activates at `first`; first_node
			// runs. first_node's output is not yet truthy, so no transition.
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(firstNode.NextCalled).To(Equal(1))
			Expect(secondNode.NextCalled).To(Equal(0))

			// Cycle 2: first_node becomes truthy, the transition fires; the
			// sequence advances to `second` in the same cycle.
			firstNode.SetTruthy(0)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(firstNode.NextCalled).To(Equal(2))
			Expect(secondNode.NextCalled).To(Equal(1))
			// The transition cascaded within the same cycle: second_node
			// reset once on activation.
			Expect(secondNode.ResetCalled).To(Equal(1))
		})

		It("Should exit the sequence when target is exit", func(ctx SpecContext) {
			trigger := mock("trigger", true)
			firstNode := mock("first_node")
			// Model a one-shot rising-edge trigger: fires on cycle 1
			// (so main activates), then releases on later cycles so the
			// activation handle isn't re-satisfying after exit.
			cycleCount := 0
			trigger.OnNext = func(ctx node.Context) {
				cycleCount++
				if cycleCount > 1 {
					trigger.OutputTruthy[0] = false
				}
			}
			first := parallelScope("first", stratum(ir.NodeMember("first_node")))
			main := sequentialScope("main",
				[]ir.Member{{Scope: &first}},
				ir.Transition{
					On:        ir.Handle{Node: "first_node", Param: "output"},
					TargetKey: exitTarget(),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("first_node", "output")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(main)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(firstNode.NextCalled).To(Equal(1))

			// Trip exit transition on cycle 2. The exit deactivates main;
			// trigger has already been released, so no re-activation.
			firstNode.SetTruthy(0)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			countAtExit := firstNode.NextCalled
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(firstNode.NextCalled).To(Equal(countAtExit))
		})

		It("Should honor source-order when multiple transitions become truthy simultaneously", func(ctx SpecContext) {
			mock("trigger", true)
			firstNode := mock("first_node")
			// Two transitions from first, in source order: first jumps to
			// `a`, the second would jump to `b`. Both are truthy at the
			// same cycle.
			first := parallelScope("first", stratum(ir.NodeMember("first_node")))
			aScope := parallelScope("a", stratum(ir.NodeMember("a_node")))
			bScope := parallelScope("b", stratum(ir.NodeMember("b_node")))
			main := sequentialScope("main",
				[]ir.Member{
					{Scope: &first},
					{Scope: &aScope},
					{Scope: &bScope},
				},
				ir.Transition{
					On:        ir.Handle{Node: "first_node", Param: "output"},
					TargetKey: stepKeyTarget("a"),
				},
				ir.Transition{
					On:        ir.Handle{Node: "first_node", Param: "output"},
					TargetKey: stepKeyTarget("b"),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			mock("a_node")
			mock("b_node")
			prog := programOf(
				[]ir.Node{
					irNode("trigger", "output"),
					irNode("first_node", "output"),
					irNode("a_node"),
					irNode("b_node"),
				},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(main)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			firstNode.SetTruthy(0)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			// First-match-wins: `a` activated, `b` did not.
			Expect(mocks["a_node"].NextCalled).To(Equal(1))
			Expect(mocks["b_node"].NextCalled).To(Equal(0))
		})

		It("Should cascade multiple transitions within a single cycle", func(ctx SpecContext) {
			// three-step sequence where step 1 immediately transitions to
			// step 2, step 2 immediately to step 3, all in one cycle.
			mock("trigger", true)
			s1 := mock("s1", true)
			s2 := mock("s2", true)
			s3 := mock("s3")
			mkStep := func(key, nodeKey string) ir.Scope {
				return parallelScope(key, stratum(ir.NodeMember(nodeKey)))
			}
			scope1 := mkStep("s1", "s1")
			scope2 := mkStep("s2", "s2")
			scope3 := mkStep("s3", "s3")
			main := sequentialScope("main",
				[]ir.Member{
					{Scope: &scope1},
					{Scope: &scope2},
					{Scope: &scope3},
				},
				ir.Transition{
					On:        ir.Handle{Node: "s1", Param: "output"},
					TargetKey: stepKeyTarget("s2"),
				},
				ir.Transition{
					On:        ir.Handle{Node: "s2", Param: "output"},
					TargetKey: stepKeyTarget("s3"),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]ir.Node{
					irNode("trigger", "output"),
					irNode("s1", "output"),
					irNode("s2", "output"),
					irNode("s3"),
				},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(main)),
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
			mock("trigger", true)
			stageNode := mock("n")
			act := ir.Handle{Node: "trigger", Param: "output"}
			stage := parallelScope("stage", stratum(ir.NodeMember("n")))
			stage.Activation = &act
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("n")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(stage)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(stageNode.ResetCalled).To(Equal(1))
		})

		It("Should cascade reset into nested always-live scopes on activation", func(ctx SpecContext) {
			mock("trigger", true)
			inner := mock("inner")
			nested := alwaysScope("nested", stratum(ir.NodeMember("inner")))
			outer := parallelScope("outer", stratum(ir.ScopeMember(nested)))
			act := ir.Handle{Node: "trigger", Param: "output"}
			outer.Activation = &act
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("inner")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(outer)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(inner.ResetCalled).To(Equal(1))
			Expect(inner.NextCalled).To(Equal(1))
		})

		It("Should auto-activate an anonymous top-level always-live scope", func(ctx SpecContext) {
			// Anonymous top-level scopes cannot be referenced by `=>` from
			// source, so the analyzer emits them as LivenessAlways to mark
			// them as program entrypoints. The parallel cascade activates
			// every always-live child of an active parent.
			inner := mock("n")
			stage := alwaysScope("anon", stratum(ir.NodeMember("n")))
			prog := programOf(
				[]ir.Node{irNode("n")},
				nil,
				rootScope(ir.ScopeMember(stage)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(inner.ResetCalled).To(Equal(1))
			Expect(inner.NextCalled).To(Equal(1))
		})

		It("Should leave a named top-level gated scope inert when it has no activation handle", func(ctx SpecContext) {
			// A named top-level scope with no activation is emitted by the
			// analyzer when the user declared `sequence main { ... }` but no
			// `=> main` trigger exists anywhere in source. It must stay
			// inert — the only way to activate it is an external trigger.
			inner := mock("n")
			stage := parallelScope("main", stratum(ir.NodeMember("n")))
			prog := programOf(
				[]ir.Node{irNode("n")},
				nil,
				rootScope(ir.ScopeMember(stage)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(inner.ResetCalled).To(Equal(0))
			Expect(inner.NextCalled).To(Equal(0))
		})

		It("Should cascade through nested always-live scopes at depth", func(ctx SpecContext) {
			// Exercises the uniform cascade rule at a non-root depth: root
			// (Always) → outer (Always) → middle (Always) → leaf node. Any
			// break in the rule would leave the leaf unactivated.
			leaf := mock("leaf")
			inner := alwaysScope("inner", stratum(ir.NodeMember("leaf")))
			middle := alwaysScope("middle", stratum(ir.ScopeMember(inner)))
			outer := alwaysScope("outer", stratum(ir.ScopeMember(middle)))
			prog := programOf(
				[]ir.Node{irNode("leaf")},
				nil,
				rootScope(ir.ScopeMember(outer)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(leaf.ResetCalled).To(Equal(1))
			Expect(leaf.NextCalled).To(Equal(1))
		})
	})

	Describe("Change propagation", func() {
		It("Should fire only the edge whose source param was marked", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeC := mock("C")
			// A declares two outputs ("x", "y"); only "x" (ordinal 0) fires.
			nodeA.OnNext = markOnNext(0)
			prog := programOf(
				[]ir.Node{irNode("A", "x", "y"), irNode("B"), irNode("C")},
				[]ir.Edge{
					continuousEdge("A", "x", "B", "in"),
					continuousEdge("A", "y", "C", "in"),
				},
				rootWithStrata(
					stratum(ir.NodeMember("A")),
					stratum(ir.NodeMember("B"), ir.NodeMember("C")),
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
			nodeA.OnNext = markOnNext(0)
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B"), irNode("C")},
				[]ir.Edge{
					continuousEdge("A", "output", "B", "in"),
					continuousEdge("A", "output", "C", "in"),
				},
				rootWithStrata(
					stratum(ir.NodeMember("A")),
					stratum(ir.NodeMember("B"), ir.NodeMember("C")),
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
			nodeA.OnNext = markOnNext(0)
			nodeB.OnNext = markOnNext(0)
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B", "output"), irNode("C")},
				[]ir.Edge{
					continuousEdge("A", "output", "C", "a"),
					continuousEdge("B", "output", "C", "b"),
				},
				rootWithStrata(
					stratum(ir.NodeMember("A"), ir.NodeMember("B")),
					stratum(ir.NodeMember("C")),
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
			nodeA.OnNext = markOnNext(0)
			nodeB.OnNext = markOnNext(0)
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B", "output"), irNode("C")},
				[]ir.Edge{
					continuousEdge("A", "output", "B", "in"),
					continuousEdge("B", "output", "C", "in"),
				},
				rootWithStrata(
					stratum(ir.NodeMember("A")),
					stratum(ir.NodeMember("B")),
					stratum(ir.NodeMember("C")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(1))
			Expect(nodeC.NextCalled).To(Equal(1))
		})

		It("Should execute a diamond graph's sink exactly once", func(ctx SpecContext) {
			mock("A").OnNext = markOnNext(0)
			mock("B").OnNext = markOnNext(0)
			mock("C").OnNext = markOnNext(0)
			nodeD := mock("D")
			prog := programOf(
				[]ir.Node{
					irNode("A", "output"),
					irNode("B", "output"),
					irNode("C", "output"),
					irNode("D"),
				},
				[]ir.Edge{
					continuousEdge("A", "output", "B", "in"),
					continuousEdge("A", "output", "C", "in"),
					continuousEdge("B", "output", "D", "a"),
					continuousEdge("C", "output", "D", "b"),
				},
				rootWithStrata(
					stratum(ir.NodeMember("A")),
					stratum(ir.NodeMember("B"), ir.NodeMember("C")),
					stratum(ir.NodeMember("D")),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeD.NextCalled).To(Equal(1))
		})

		It("Should not propagate when no edge targets the source's changed param", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.OnNext = markOnNext(0)
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B")},
				nil,
				rootWithStrata(stratum(ir.NodeMember("A")), stratum(ir.NodeMember("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeB.NextCalled).To(Equal(0))
		})
	})

	Describe("Conditional edge semantics", func() {
		It("Should fire every cycle while the source stays truthy", func(ctx SpecContext) {
			mock("A", true)
			nodeB := mock("B")
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B")},
				[]ir.Edge{conditionalEdge("A", "output", "B", "in")},
				rootWithStrata(stratum(ir.NodeMember("A")), stratum(ir.NodeMember("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(3))
		})

		It("Should stop firing when the source transitions from truthy to falsy", func(ctx SpecContext) {
			nodeA := mock("A", true)
			nodeB := mock("B")
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B")},
				[]ir.Edge{conditionalEdge("A", "output", "B", "in")},
				rootWithStrata(stratum(ir.NodeMember("A")), stratum(ir.NodeMember("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))

			nodeA.OutputTruthy[0] = false
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should keep continuous edges unaffected by source truthiness", func(ctx SpecContext) {
			// Source is not marked truthy; a continuous edge must still
			// propagate the change.
			nodeA := mock("A")
			nodeB := mock("B")
			nodeA.OnNext = markOnNext(0)
			prog := programOf(
				[]ir.Node{irNode("A", "output"), irNode("B")},
				[]ir.Edge{continuousEdge("A", "output", "B", "in")},
				rootWithStrata(stratum(ir.NodeMember("A")), stratum(ir.NodeMember("B"))),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(nodeB.NextCalled).To(Equal(1))
		})

		It("Should fire each conditional edge independently per param truthiness", func(ctx SpecContext) {
			// A declares two outputs ("x", "y"); only "x" (ordinal 0) is
			// truthy. Both fire MarkChanged but only the conditional edge
			// from "x" propagates.
			nodeA := mock("A", true, false)
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.OnNext = func(ctx node.Context) {
				ctx.MarkChanged(0)
				ctx.MarkChanged(1)
			}
			prog := programOf(
				[]ir.Node{irNode("A", "x", "y"), irNode("B"), irNode("C")},
				[]ir.Edge{
					conditionalEdge("A", "x", "B", "in"),
					conditionalEdge("A", "y", "C", "in"),
				},
				rootWithStrata(
					stratum(ir.NodeMember("A")),
					stratum(ir.NodeMember("B"), ir.NodeMember("C")),
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
			// trigger fires a single change into A on cycle 1, then
			// stays quiet. A's self-marking should drive the next two
			// replays on its own; once it stops marking, the scheduler
			// must not replay again.
			trigger := mock("trigger")
			trigger.SuppressAutoMark = true
			triggerFired := false
			trigger.OnNext = func(c node.Context) {
				if !triggerFired {
					c.MarkChanged(0)
					triggerFired = true
				}
			}
			prog := programOf(
				[]ir.Node{irNode("trigger", "kick"), irNode("A")},
				[]ir.Edge{continuousEdge("trigger", "kick", "A", "in")},
				rootWithStrata(stratum(ir.NodeMember("trigger")), stratum(ir.NodeMember("A"))),
			)
			s := build(prog)
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
			mock("trigger", true)
			stageNode := mock("stage_node", true)
			mock("second_node")
			triggerFired := false
			stageNode.OnNext = func(c node.Context) {
				c.MarkSelfChanged()
				if !triggerFired {
					triggerFired = true
					c.MarkChanged(0)
				}
			}

			first := parallelScope("first", stratum(ir.NodeMember("stage_node")))
			second := parallelScope("second", stratum(ir.NodeMember("second_node")))
			main := sequentialScope("main",
				[]ir.Member{{Scope: &first}, {Scope: &second}},
				ir.Transition{
					On:        ir.Handle{Node: "stage_node", Param: "done"},
					TargetKey: stepKeyTarget("second"),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH

			prog := programOf(
				[]ir.Node{
					irNode("trigger", "output"),
					irNode("stage_node", "done"),
					irNode("second_node"),
				},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(main)),
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
			prog := programOf([]ir.Node{irNode("A")}, nil, rootScope(ir.NodeMember("A")))
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
			prog := programOf([]ir.Node{irNode("A")}, nil, rootScope(ir.NodeMember("A")))
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.Second))
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(s.NextDeadline()).To(Equal(telem.TimeSpanMax))
		})

		It("Should track deadlines from nodes inside an active gated scope", func(ctx SpecContext) {
			mock("trigger", true)
			stageNode := mock("stage_node")
			stageNode.OnNext = func(c node.Context) { c.SetDeadline(2 * telem.Second) }
			act := ir.Handle{Node: "trigger", Param: "output"}
			gated := parallelScope("stage", stratum(ir.NodeMember("stage_node")))
			gated.Activation = &act
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("stage_node")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(gated)),
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
				[]ir.Node{irNode("A"), irNode("B"), irNode("C")},
				nil,
				rootWithStrata(stratum(ir.NodeMember("A"), ir.NodeMember("B"), ir.NodeMember("C"))),
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
				[]ir.Node{irNode("A"), irNode("B")},
				nil,
				rootScope(ir.NodeMember("A"), ir.NodeMember("B")),
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
			prog := programOf([]ir.Node{irNode("A")}, nil, rootScope(ir.NodeMember("A")))
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
			prog := programOf([]ir.Node{irNode("A")}, nil, rootScope(ir.NodeMember("A")))
			s := build(prog)
			s.Next(ctx, 0, node.ReasonTimerTick)
			Expect(nodeA.NextCalled).To(Equal(1))
			Expect(nodeA.ElapsedValues[0]).To(Equal(telem.TimeSpan(0)))
		})

		It("Should pass through ReasonChannelInput", func(ctx SpecContext) {
			var received node.RunReason
			nodeA := mock("A")
			nodeA.OnNext = func(c node.Context) { received = c.Reason }
			prog := programOf([]ir.Node{irNode("A")}, nil, rootScope(ir.NodeMember("A")))
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonChannelInput)
			Expect(received).To(Equal(node.ReasonChannelInput))
		})

		It("Should tolerate a self-loop edge in phase 0", func(ctx SpecContext) {
			nodeA := mock("A")
			nodeA.OnNext = markOnNext(0)
			prog := programOf(
				[]ir.Node{irNode("A", "output")},
				[]ir.Edge{continuousEdge("A", "output", "A", "in")},
				rootScope(ir.NodeMember("A")),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			// Phase 0 is unconditional, so A ran once. The self-loop adds
			// A to `changed`, but `changed` is cleared at cycle end, and
			// there is no higher phase to re-run into.
			Expect(nodeA.NextCalled).To(Equal(1))
		})

		It("Should tolerate an empty sequential scope", func(ctx SpecContext) {
			trigger := mock("trigger", true)
			main := ir.Scope{
				Key:      "main",
				Mode:     ir.ScopeModeSequential,
				Liveness: ir.LivenessGated,
			}
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]ir.Node{irNode("trigger", "output")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(main)),
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
			mock("trigger", true)
			stageNode := mock("stage_node")
			act := ir.Handle{Node: "trigger", Param: "output"}
			gated := parallelScope("stage", stratum(ir.NodeMember("stage_node")))
			gated.Activation = &act
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("stage_node")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(gated)),
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
			mock("trigger_a", true)
			mock("trigger_b")
			a := mock("A")
			b := mock("B")
			// trigger_b stays falsy — only `a` should activate.
			stageA := parallelScope("stage_a", stratum(ir.NodeMember("A")))
			actA := ir.Handle{Node: "trigger_a", Param: "output"}
			stageA.Activation = &actA
			stageB := parallelScope("stage_b", stratum(ir.NodeMember("B")))
			actB := ir.Handle{Node: "trigger_b", Param: "output"}
			stageB.Activation = &actB
			prog := programOf(
				[]ir.Node{
					irNode("trigger_a", "output"),
					irNode("trigger_b", "output"),
					irNode("A"),
					irNode("B"),
				},
				nil,
				rootScope(
					ir.NodeMember("trigger_a"),
					ir.NodeMember("trigger_b"),
					ir.ScopeMember(stageA),
					ir.ScopeMember(stageB),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(a.NextCalled).To(Equal(1))
			Expect(b.NextCalled).To(Equal(0))
		})

		It("Should mix continuous and conditional edges in a single graph", func(ctx SpecContext) {
			nodeA := mock("A", true, true)
			nodeB := mock("B")
			nodeC := mock("C")
			nodeA.OnNext = func(c node.Context) {
				c.MarkChanged(0)
				c.MarkChanged(1)
			}
			prog := programOf(
				[]ir.Node{irNode("A", "data", "trigger"), irNode("B"), irNode("C")},
				[]ir.Edge{
					continuousEdge("A", "data", "B", "in"),
					conditionalEdge("A", "trigger", "C", "in"),
				},
				rootWithStrata(
					stratum(ir.NodeMember("A")),
					stratum(ir.NodeMember("B"), ir.NodeMember("C")),
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
			trigger := mock("trigger", true)
			firstNode := mock("first_node", true)
			cycle := 0
			trigger.OnNext = func(c node.Context) {
				cycle++
				// Release after cycle 1, re-assert on cycle 3.
				if cycle == 2 {
					trigger.OutputTruthy[0] = false
				}
				if cycle == 3 {
					trigger.OutputTruthy[0] = true
				}
			}
			first := parallelScope("first", stratum(ir.NodeMember("first_node")))
			main := sequentialScope("main",
				[]ir.Member{{Scope: &first}},
				ir.Transition{
					On:        ir.Handle{Node: "first_node", Param: "output"},
					TargetKey: exitTarget(),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("first_node", "output")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(main)),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)   // activate + run + exit
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick) // trigger released, no action
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick) // trigger reasserted, main re-activates, runs, exits
			// Two activations ⇒ two Reset calls on first_node.
			Expect(firstNode.ResetCalled).To(Equal(2))
		})
	})

	Describe("Transitions gated on fresh output marks", func() {
		// Sequential transitions must fire only when the source node
		// called MarkChanged with a truthy output this cycle. Nodes whose
		// output cache stays truthy across cycles (e.g., wait, interval,
		// latched comparisons) must not drive repeated transitions after
		// their one-shot announcement. This mirrors the conditional-edge
		// firing semantic of the pre-Scope scheduler.
		It("Should not fire a transition when the source is truthy but never called MarkChanged", func(ctx SpecContext) {
			// Minimal repro. The sequence "main" has a single member
			// "body" and a single transition whose on-handle is a latch
			// node external to the sequence. The latch is constructed
			// truthy (ParamTruthy.Add("output")) but suppresses the
			// auto-mark behavior, so its IsOutputTruthy returns true
			// forever while MarkChanged is never called. A correctly
			// gated scheduler must not fire the transition.
			mock("trigger", true)
			latch := mock("latch", true)
			latch.SuppressAutoMark = true
			mock("worker")

			body := parallelScope("body", stratum(ir.NodeMember("worker")))
			main := sequentialScope("main", []ir.Member{
				{Scope: &body},
			}, ir.Transition{
				On:        ir.Handle{Node: "latch", Param: "output"},
				TargetKey: exitTarget(),
			})
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH

			prog := programOf(
				[]ir.Node{
					irNode("trigger", "output"),
					irNode("latch", "output"),
					irNode("worker"),
				},
				nil,
				rootScope(
					ir.NodeMember("trigger"),
					ir.NodeMember("latch"),
					ir.ScopeMember(main),
				),
			)
			s := build(prog)
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)

			// Main must remain active: the transition's source was
			// never MarkChanged, so the exit must not have fired.
			Expect(mocks["worker"].NextCalled).To(Equal(1))
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(mocks["worker"].NextCalled).To(Equal(2))
		})

		It("Should not re-fire a transition on a later cycle when the source stays truthy but only marked on the first cycle", func(ctx SpecContext) {
			// Covers the integration-level wait/interval regression:
			// a node calls MarkChanged on its first firing and stays
			// truthy across subsequent cycles without re-marking. The
			// scheduler must fire the transition exactly once (on the
			// cycle MarkChanged was called), not again on later cycles
			// where the on-handle is still truthy.
			mock("trigger", true)
			latch := mock("latch", true)
			latch.SuppressAutoMark = true
			marks := 0
			latch.OnNext = func(c node.Context) {
				marks++
				if marks == 1 {
					c.MarkChanged(0)
				}
			}
			mock("worker_a")
			mock("worker_b")

			a := parallelScope("a", stratum(ir.NodeMember("worker_a")))
			b := parallelScope("b", stratum(ir.NodeMember("worker_b")))
			main := sequentialScope("main", []ir.Member{
				{Scope: &a},
				{Scope: &b},
			}, ir.Transition{
				On:        ir.Handle{Node: "latch", Param: "output"},
				TargetKey: stepKeyTarget("b"),
			})
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH

			prog := programOf(
				[]ir.Node{
					irNode("trigger", "output"),
					irNode("latch", "output"),
					irNode("worker_a"),
					irNode("worker_b"),
				},
				nil,
				rootScope(
					ir.NodeMember("trigger"),
					ir.NodeMember("latch"),
					ir.ScopeMember(main),
				),
			)
			s := build(prog)

			// Cycle 1: main activates at "a"; latch calls MarkChanged
			// once so the transition fires → b becomes active. worker_b
			// runs this cycle; worker_a runs exactly once (before the
			// transition).
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(mocks["worker_a"].NextCalled).To(Equal(1))
			Expect(mocks["worker_b"].NextCalled).To(Equal(1))

			// Cycle 2: latch is still truthy but did NOT MarkChanged.
			// The transition's on-handle is external to main (owner=-1),
			// so it is still evaluated each cycle. It must NOT re-fire.
			// worker_b keeps running; worker_a must not be re-activated.
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(mocks["worker_a"].NextCalled).To(Equal(1))
			Expect(mocks["worker_b"].NextCalled).To(Equal(2))

			// Cycle 3: same invariant.
			s.Next(ctx, 3*telem.Microsecond, node.ReasonTimerTick)
			Expect(mocks["worker_a"].NextCalled).To(Equal(1))
			Expect(mocks["worker_b"].NextCalled).To(Equal(3))
		})

		It("Should fire a transition again when the source freshly marks changed on a later cycle", func(ctx SpecContext) {
			// Positive companion test: ensure the gating change does
			// not break the case where a source marks changed anew on a
			// later cycle. The transition must fire on that cycle.
			mock("trigger", true)
			latch := mock("latch")
			latch.SuppressAutoMark = true
			cycle := 0
			latch.OnNext = func(c node.Context) {
				cycle++
				if cycle == 2 {
					latch.SetTruthy(0)
					c.MarkChanged(0)
				}
			}
			mock("worker_a")
			mock("worker_b")

			a := parallelScope("a", stratum(ir.NodeMember("worker_a")))
			b := parallelScope("b", stratum(ir.NodeMember("worker_b")))
			main := sequentialScope("main", []ir.Member{
				{Scope: &a},
				{Scope: &b},
			}, ir.Transition{
				On:        ir.Handle{Node: "latch", Param: "output"},
				TargetKey: stepKeyTarget("b"),
			})
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH

			prog := programOf(
				[]ir.Node{
					irNode("trigger", "output"),
					irNode("latch", "output"),
					irNode("worker_a"),
					irNode("worker_b"),
				},
				nil,
				rootScope(
					ir.NodeMember("trigger"),
					ir.NodeMember("latch"),
					ir.ScopeMember(main),
				),
			)
			s := build(prog)
			// Cycle 1: latch does not mark; transition does not fire.
			s.Next(ctx, telem.Microsecond, node.ReasonTimerTick)
			Expect(mocks["worker_b"].NextCalled).To(Equal(0))
			// Cycle 2: latch marks; transition fires → worker_b runs.
			s.Next(ctx, 2*telem.Microsecond, node.ReasonTimerTick)
			Expect(mocks["worker_b"].NextCalled).To(Equal(1))
		})
	})

	Describe("Convergence bound", func() {
		It("Should not infinite-loop if a transition's on-handle stays truthy and owner stays active", func(ctx SpecContext) {
			// Construct a sequence where the transition targets the same
			// member that contains its on-handle. Under normal semantics
			// the transition would fire, reactivate the same member, and
			// re-fire. The convergence bound must keep this from looping
			// forever within a cycle.
			mock("trigger", true)
			loopNode := mock("loop_node", true)
			loop := parallelScope("loop", stratum(ir.NodeMember("loop_node")))
			main := sequentialScope("main",
				[]ir.Member{{Scope: &loop}},
				ir.Transition{
					On:        ir.Handle{Node: "loop_node", Param: "output"},
					TargetKey: stepKeyTarget("loop"),
				},
			)
			triggerH := ir.Handle{Node: "trigger", Param: "output"}
			main.Activation = &triggerH
			prog := programOf(
				[]ir.Node{irNode("trigger", "output"), irNode("loop_node", "output")},
				nil,
				rootScope(ir.NodeMember("trigger"), ir.ScopeMember(main)),
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
