// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package scheduler executes an Arc program by walking the Layer-2 Scope
// tree attached to the IR's Root. Each cycle the scheduler walks the tree,
// executes the active members of each reachable scope, evaluates transitions
// on sequential scopes, and checks activation handles on gated scopes.
package scheduler

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// node pairs a runtime node instance with its outgoing edges, cached for
// fast propagation during markChanged.
type node struct {
	rnode.Node
	outgoing map[string][]ir.Edge
	key      string
}

// memberState holds runtime state for one member of a scope. Exactly one of
// nodeKey or scope is populated, mirroring the IR's Member tagged union.
type memberState struct {
	key     string
	nodeKey string
	scope   *scopeState
}

// isNodeRef reports whether this member refers to a dataflow node rather
// than a nested scope.
func (m *memberState) isNodeRef() bool { return m.scope == nil }

// scopeState holds runtime state for one scope in the program's Scope tree.
// It mirrors the static ir.Scope with mutable activation bookkeeping.
type scopeState struct {
	ir           *ir.Scope
	mode         ir.ScopeMode
	liveness     ir.Liveness
	active       bool
	activeMember int
	members      []memberState
	memberByKey  map[string]int
	// transitionOwner[i] is the index of the member whose node ownership
	// produces transition i's `on` handle, or -1 when the handle is
	// sourced from outside the scope (e.g., a module-scope channel read
	// that fires cross-scope activations). Only transitions rooted in
	// the currently-active member — or entirely outside the scope —
	// should be evaluated each cycle.
	transitionOwner []int
}

// Scheduler orchestrates the execution of nodes in a compiled Arc program.
// Construction is a one-time cost; Next is called once per scheduler tick.
type Scheduler struct {
	nodeCtx      rnode.Context
	errorHandler ErrorHandler
	changed      set.Set[string]
	selfChanged  set.Set[string]
	// markedThisCycle holds the set of (nodeKey, param) pairs for which
	// MarkChanged was called and the output is truthy during the current
	// cycle. Keyed by "nodeKey\x00param". Sequential transitions fire on
	// a fresh mark rather than on stale cached truthiness, mirroring the
	// conditional-edge firing semantic of the pre-Scope scheduler. Marks
	// are consumed when a transition fires and cleared at end of cycle.
	markedThisCycle set.Set[string]
	nodes           map[string]node
	currNodeKey     string
	// root mirrors prog.Root; always a parallel, always-live scope.
	root *scopeState
	// activationsBySource indexes scopes whose activation handle is sourced
	// by the keyed node. After each node's execution, the scheduler polls
	// the listed scopes to see whether their activation condition has become
	// truthy and if so activates them.
	activationsBySource map[string][]*scopeState
	// maxConvergenceIter bounds the per-scope transition loop. A sequential
	// scope may cascade through its own transitions within a single
	// scheduler cycle up to this many times before the cycle completes.
	maxConvergenceIter int
	tolerance          telem.TimeSpan
	nextDeadline       telem.TimeSpan
}

// ErrorHandler receives errors raised by node execution.
type ErrorHandler interface {
	HandleError(ctx context.Context, nodeKey string, err error)
}

// ErrorHandlerFunc adapts an ordinary function to the ErrorHandler interface.
type ErrorHandlerFunc func(ctx context.Context, nodeKey string, err error)

// HandleError implements ErrorHandler.
func (f ErrorHandlerFunc) HandleError(ctx context.Context, nodeKey string, err error) {
	f(ctx, nodeKey, err)
}

// New creates a scheduler from a compiled IR and a set of runtime node
// instances keyed by ir.Node.Key. tolerance controls how early timer-based
// nodes may fire relative to their deadline.
func New(prog ir.IR, nodes map[string]rnode.Node, tolerance telem.TimeSpan) *Scheduler {
	s := &Scheduler{
		nodes:               make(map[string]node, len(prog.Nodes)),
		changed:             make(set.Set[string], len(prog.Nodes)),
		selfChanged:         make(set.Set[string]),
		markedThisCycle:     make(set.Set[string]),
		activationsBySource: make(map[string][]*scopeState),
		tolerance:           tolerance,
	}

	for _, n := range prog.Nodes {
		outgoing := make(map[string][]ir.Edge)
		for _, edge := range prog.Edges {
			if edge.Source.Node == n.Key {
				outgoing[edge.Source.Param] = append(outgoing[edge.Source.Param], edge)
			}
		}
		s.nodes[n.Key] = node{key: n.Key, outgoing: outgoing, Node: nodes[n.Key]}
	}

	// Build the scope state tree rooted at prog.Root. buildScopeState
	// captures activation sources and totals the transition-count budget.
	s.root = s.buildScopeState(&prog.Root)
	// The root scope is parallel+always-live; seed it active so its phases
	// execute every cycle. Activating it also resets direct members and
	// cascades into any gated children (there won't be any at the root
	// itself, but the recursive reset is harmless).
	s.activateScope(s.root)

	s.nodeCtx = rnode.Context{
		MarkChanged:     s.markChanged,
		MarkSelfChanged: s.markSelfChanged,
		SetDeadline:     s.setDeadline,
		ReportError:     s.reportError,
	}

	return s
}

// buildScopeState recursively constructs a scopeState mirror of the IR
// scope tree. The returned state is inert — any activations and member
// resets are performed by explicit activateScope calls.
func (s *Scheduler) buildScopeState(scope *ir.Scope) *scopeState {
	state := &scopeState{
		ir:           scope,
		mode:         scope.Mode,
		liveness:     scope.Liveness,
		activeMember: -1,
	}
	appendMember := func(m ir.Member) {
		ms := memberState{key: m.Key}
		switch {
		case m.NodeRef != nil:
			ms.nodeKey = m.NodeRef.Key
		case m.Scope != nil:
			ms.scope = s.buildScopeState(m.Scope)
		}
		state.members = append(state.members, ms)
	}
	switch scope.Mode {
	case ir.ScopeModeParallel:
		for _, phase := range scope.Phases {
			for _, m := range phase.Members {
				appendMember(m)
			}
		}
	case ir.ScopeModeSequential:
		for _, m := range scope.Members {
			appendMember(m)
		}
	}
	state.memberByKey = make(map[string]int, len(state.members))
	for i, m := range state.members {
		state.memberByKey[m.key] = i
	}
	// Register activation lookup for gated scopes with an explicit
	// activation handle (typically top-level scopes targeted by a
	// cross-scope reference in source).
	if scope.Liveness == ir.LivenessGated && scope.Activation != nil {
		s.activationsBySource[scope.Activation.Node] = append(
			s.activationsBySource[scope.Activation.Node], state,
		)
	}
	// Each sequential scope contributes one potential transition per cycle
	// to the convergence budget. Parallel scopes don't have transitions.
	if scope.Mode == ir.ScopeModeSequential {
		s.maxConvergenceIter += len(scope.Members) + 1
		state.transitionOwner = computeTransitionOwners(state, scope.Transitions)
	}
	return state
}

// computeTransitionOwners maps each transition to the index of the member
// whose node ownership sources the transition's `on` handle. A transition
// whose on-node sits outside the scope entirely gets owner -1 — those
// transitions fire regardless of which member is active.
func computeTransitionOwners(state *scopeState, transitions []ir.Transition) []int {
	nodeToMember := make(map[string]int)
	for i, m := range state.members {
		collectMemberNodes(&m, i, nodeToMember)
	}
	owners := make([]int, len(transitions))
	for i, t := range transitions {
		if idx, ok := nodeToMember[t.On.Node]; ok {
			owners[i] = idx
			continue
		}
		owners[i] = -1
	}
	return owners
}

// collectMemberNodes adds every node key owned (directly or transitively)
// by the given member to the provided map, tagged with the member index.
// A NodeRef member owns one node; a nested-scope member owns every node
// reachable through its scope tree.
func collectMemberNodes(m *memberState, idx int, out map[string]int) {
	if m.isNodeRef() {
		out[m.nodeKey] = idx
		return
	}
	if m.scope != nil {
		collectScopeNodes(m.scope, idx, out)
	}
}

func collectScopeNodes(ss *scopeState, idx int, out map[string]int) {
	for i := range ss.members {
		inner := &ss.members[i]
		if inner.isNodeRef() {
			out[inner.nodeKey] = idx
		} else if inner.scope != nil {
			collectScopeNodes(inner.scope, idx, out)
		}
	}
}

// SetErrorHandler configures the handler for node execution errors.
func (s *Scheduler) SetErrorHandler(handler ErrorHandler) {
	s.errorHandler = handler
}

// MarkNodeChanged marks a node as changed, scheduling it for execution on
// the next cycle. Typically invoked from outside the scheduler when an
// external input (e.g., a newly-received channel frame) becomes available.
func (s *Scheduler) MarkNodeChanged(nodeKey string) {
	s.changed.Add(nodeKey)
}

// NextDeadline returns the earliest deadline reported by any node during
// the previous Next call. Callers use this to sleep until the scheduler
// has work to do.
func (s *Scheduler) NextDeadline() telem.TimeSpan { return s.nextDeadline }

// Next executes one cycle of the reactive computation. Nodes with pending
// changes execute in phase order; sequential scopes advance via their
// transitions; gated scopes activate when their activation handle fires.
func (s *Scheduler) Next(ctx context.Context, elapsed telem.TimeSpan, reason rnode.RunReason) {
	s.nextDeadline = telem.TimeSpanMax
	s.nodeCtx.Context = ctx
	s.nodeCtx.Elapsed = elapsed
	s.nodeCtx.Tolerance = s.tolerance
	s.nodeCtx.Reason = reason

	s.walk(s.root)

	clear(s.changed)
	clear(s.markedThisCycle)
}

// markKey encodes a (node, param) pair for markedThisCycle lookups.
func markKey(nodeKey, param string) string {
	return nodeKey + "\x00" + param
}

// walk executes one pass over a scope. Sequential scopes internally loop to
// converge cascading transitions within a single cycle.
func (s *Scheduler) walk(ss *scopeState) {
	if !ss.active {
		return
	}
	switch ss.mode {
	case ir.ScopeModeParallel:
		s.walkParallel(ss)
	case ir.ScopeModeSequential:
		s.walkSequential(ss)
	}
}

// walkParallel runs every member of a parallel scope in phase order. Nodes
// execute when they're in phase 0, when they're in the changed set, or when
// they've been marked self-changed by a previous cycle.
func (s *Scheduler) walkParallel(ss *scopeState) {
	for phaseIdx, phase := range ss.ir.Phases {
		for _, m := range phase.Members {
			s.executeMember(phaseIdx, ss, m)
		}
	}
}

// walkSequential executes the active member, then loops checking this
// scope's transitions for same-cycle cascading. Bounded by the
// scheduler's convergence budget.
func (s *Scheduler) walkSequential(ss *scopeState) {
	if ss.activeMember < 0 {
		return
	}
	// +1 so a chain of N members can fire N consecutive transitions.
	budget := len(ss.members) + 1
	for i := 0; i < budget; i++ {
		if ss.activeMember < 0 {
			return
		}
		// Sequential members always execute while they're active — they
		// are not subject to the phase-0 / changed filtering of parallel
		// scopes. Pass a synthetic phase index of 0 so node execution is
		// unconditional for the currently-active sequential member.
		s.executeSequentialMember(ss.members[ss.activeMember])
		fired := s.evaluateTransitions(ss)
		if !fired {
			return
		}
	}
}

// executeMember executes a single member of a parallel scope. If the
// member is a nested scope, the scope is walked (assuming it was
// previously activated — activation itself fires reactively when the
// activation handle's source calls MarkChanged). If the member is a
// NodeRef, the node runs subject to the usual phase-0 / changed /
// self-changed filtering.
func (s *Scheduler) executeMember(phaseIdx int, parent *scopeState, m ir.Member) {
	if m.Scope != nil {
		idx, ok := parent.memberByKey[m.Key]
		if !ok {
			return
		}
		child := parent.members[idx].scope
		if child == nil {
			return
		}
		s.walk(child)
		return
	}
	if m.NodeRef == nil {
		return
	}
	key := m.NodeRef.Key
	wasSelfChanged := s.selfChanged.Contains(key)
	if wasSelfChanged {
		delete(s.selfChanged, key)
	}
	if phaseIdx == 0 || s.changed.Contains(key) || wasSelfChanged {
		s.runNode(key)
	}
}

// executeSequentialMember unconditionally executes a sequential scope's
// active member. Sequential members are "always on" while active — they do
// not wait for an upstream change signal.
func (s *Scheduler) executeSequentialMember(m memberState) {
	if !m.isNodeRef() {
		s.walk(m.scope)
		return
	}
	delete(s.selfChanged, m.nodeKey)
	s.runNode(m.nodeKey)
}

// runNode dispatches a node's Next method. Activation polling runs inside
// markChanged when the current node calls MarkChanged on an output that
// sources a gated scope's activation handle.
func (s *Scheduler) runNode(key string) {
	n, ok := s.nodes[key]
	if !ok {
		return
	}
	s.currNodeKey = key
	n.Next(s.nodeCtx)
}

// evaluateTransitions walks the sequential scope's transitions in source
// order and applies the first one whose `on` handle was freshly marked
// changed with a truthy value during the current cycle. Transitions
// whose on-handle is owned by an inactive sibling member are skipped —
// their source node is frozen (and possibly still truthy) from a prior
// activation, but the step that defined the transition is no longer the
// one running. A stale-truthy source that did not re-mark this cycle does
// not fire the transition; this mirrors the conditional-edge firing
// semantic of the pre-Scope scheduler and prevents wait/interval/latched
// comparisons (which mark once per event) from driving spurious repeated
// transitions on later cycles. Firing consumes the mark, so a single
// MarkChanged call produces at most one transition firing per cycle.
// Reports whether a transition fired so the caller can drive the
// convergence loop.
func (s *Scheduler) evaluateTransitions(ss *scopeState) bool {
	for i, t := range ss.ir.Transitions {
		if owner := ss.transitionOwner[i]; owner >= 0 && owner != ss.activeMember {
			continue
		}
		key := markKey(t.On.Node, t.On.Param)
		if !s.markedThisCycle.Contains(key) {
			continue
		}
		if !s.isHandleTruthy(t.On) {
			continue
		}
		delete(s.markedThisCycle, key)
		if ss.activeMember >= 0 {
			s.deactivateMember(ss, ss.activeMember)
		}
		switch {
		case t.Target.Exit != nil && *t.Target.Exit:
			s.deactivateScope(ss)
		case t.Target.MemberKey != nil:
			idx, ok := ss.memberByKey[*t.Target.MemberKey]
			if !ok {
				return false
			}
			s.activateSequentialMember(ss, idx)
		}
		return true
	}
	return false
}

// activateScope marks a scope active and primes its member(s) for
// execution. Behavior differs by mode and liveness:
//
//   - Sequential scope: becomes active at member 0 and the member's node
//     is reset (or, if the member is a nested scope, activated).
//   - Parallel + gated scope: every direct NodeRef member is reset, and
//     every nested gated scope member is cascade-activated.
//   - Parallel + always-live scope (the root): every direct NodeRef
//     member is reset. Nested scope members are NOT cascade-activated;
//     gated children of the root become active only when their
//     activation handle fires.
func (s *Scheduler) activateScope(ss *scopeState) {
	ss.active = true
	if ss.mode == ir.ScopeModeSequential {
		if len(ss.members) > 0 {
			s.activateSequentialMember(ss, 0)
		}
		return
	}
	for i := range ss.members {
		m := &ss.members[i]
		switch {
		case m.isNodeRef():
			delete(s.selfChanged, m.nodeKey)
			if n, ok := s.nodes[m.nodeKey]; ok {
				n.Reset()
			}
		case m.scope != nil && ss.liveness == ir.LivenessGated:
			// Cascade into nested gated scopes only when this scope is
			// itself gated — the root (always-live) does not cascade.
			s.activateScope(m.scope)
		}
	}
}

// activateSequentialMember moves a sequential scope's active pointer to
// idx and resets (or cascade-activates) the newly-active member.
func (s *Scheduler) activateSequentialMember(ss *scopeState, idx int) {
	ss.activeMember = idx
	m := &ss.members[idx]
	if m.isNodeRef() {
		delete(s.selfChanged, m.nodeKey)
		if n, ok := s.nodes[m.nodeKey]; ok {
			n.Reset()
		}
		return
	}
	if m.scope != nil {
		s.activateScope(m.scope)
	}
}

// deactivateMember clears selfChanged for the member's owned node and, if
// the member wraps a nested scope, marks that scope inactive (without
// recursing into its own members per RFC 0035 §3.3).
func (s *Scheduler) deactivateMember(ss *scopeState, idx int) {
	m := &ss.members[idx]
	if m.isNodeRef() {
		delete(s.selfChanged, m.nodeKey)
		return
	}
	if m.scope != nil {
		s.deactivateScope(m.scope)
	}
}

// deactivateScope marks a scope inactive and clears the selfChanged set
// for its direct NodeRef members. Per RFC 0035 §3.3, deactivation does not
// cascade — nested scope members retain their frozen state.
func (s *Scheduler) deactivateScope(ss *scopeState) {
	if ss.mode == ir.ScopeModeSequential {
		ss.activeMember = -1
	}
	for i := range ss.members {
		m := &ss.members[i]
		if m.isNodeRef() {
			delete(s.selfChanged, m.nodeKey)
		}
	}
	ss.active = false
}

// isHandleTruthy reports whether the node referenced by h has a truthy
// output on the handle's parameter. Missing nodes produce false.
func (s *Scheduler) isHandleTruthy(h ir.Handle) bool {
	n, ok := s.nodes[h.Node]
	if !ok {
		return false
	}
	return n.IsOutputTruthy(h.Param)
}

// markChanged propagates changes from the current node's output to
// downstream nodes and fires any gated scope activations sourced from
// this output. Continuous edges always propagate; conditional edges only
// propagate when the source output is truthy. Activations fire on any
// MarkChanged notification for the matching handle — equivalent to the
// continuous-edge semantics of the pre-Scope IR.
func (s *Scheduler) markChanged(param string) {
	n := s.nodes[s.currNodeKey]
	if n.IsOutputTruthy(param) {
		s.markedThisCycle.Add(markKey(s.currNodeKey, param))
	}
	for _, edge := range n.outgoing[param] {
		if edge.Kind == ir.EdgeKindConditional {
			if n.IsOutputTruthy(param) {
				s.changed.Add(edge.Target.Node)
			}
			continue
		}
		s.changed.Add(edge.Target.Node)
	}
	for _, scope := range s.activationsBySource[s.currNodeKey] {
		if scope.active {
			continue
		}
		if scope.ir.Activation == nil || scope.ir.Activation.Param != param {
			continue
		}
		s.activateScope(scope)
	}
}

// markSelfChanged requests that the current node execute again on the
// next scheduler cycle without requiring an upstream change.
func (s *Scheduler) markSelfChanged() {
	s.selfChanged.Add(s.currNodeKey)
}

// reportError forwards a node-reported error to the configured handler
// (if any), annotating it with the node currently executing.
func (s *Scheduler) reportError(err error) {
	if s.errorHandler != nil {
		s.errorHandler.HandleError(s.nodeCtx.Context, s.currNodeKey, err)
	}
}

// setDeadline lowers the scheduler's nextDeadline if d is earlier.
func (s *Scheduler) setDeadline(d telem.TimeSpan) {
	if d < s.nextDeadline {
		s.nextDeadline = d
	}
}
