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
	"github.com/synnaxlabs/x/telem"
)

// outEdge is the pre-resolved form of an ir.Edge cached on the source
// node. targetIdx is the index into the scheduler's dense node table;
// conditional records whether the edge only fires when the source output
// is truthy.
type outEdge struct {
	targetIdx   int
	conditional bool
}

// outputResolved is the per-output propagation table built once at
// construction and indexed by the node-local output ordinal. Avoids all
// string hashing on the hot path — MarkChanged does pure array access.
type outputResolved struct {
	// param is the output name, retained so the scheduler can call
	// node.IsOutputTruthy(param) — which still uses the string-keyed
	// interface.
	param string
	// edges is the pre-resolved set of outgoing dataflow edges.
	edges []outEdge
	// markHandleIdx is the index into the scheduler's markedFlags slice,
	// or -1 when no sequential-scope transition consumes this output.
	markHandleIdx int
	// activates is the set of scopes whose activation handle is this
	// output; polled after every truthy MarkChanged call.
	activates []*scopeState
}

// node pairs a runtime node instance with its pre-resolved per-output
// propagation tables, keyed by the node's local output ordinal. idx is
// the node's position in the scheduler's dense changed/self-changed
// flag slices.
type node struct {
	rnode.Node
	outputs []outputResolved
	// outputByParam maps output name to ordinal. Used by construction
	// paths that resolve edges / transitions / activations; not read on
	// the hot path.
	outputByParam map[string]int
	key           string
	idx           int
}

// memberState holds runtime state for one member of a scope. Exactly one of
// nodeKey or scope is populated, mirroring the IR's Member tagged union.
// nodeIdx is the dense flag-slice index for a NodeRef member; it is -1
// for scope members.
type memberState struct {
	key     string
	nodeKey string
	nodeIdx int
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
	// transitionOnIdx[i] is the scheduler-wide markedFlags index for
	// transition i's `on` handle, pre-resolved at construction so the
	// hot-path evaluation is a single array load.
	transitionOnIdx []int
}

// Scheduler orchestrates the execution of nodes in a compiled Arc program.
// Construction is a one-time cost; Next is called once per scheduler tick.
type Scheduler struct {
	nodeCtx      rnode.Context
	errorHandler ErrorHandler
	// changedFlags and selfChangedFlags are dense per-node uint8 slices
	// indexed by node.idx. A non-zero entry in changedFlags means the node
	// has a pending upstream change for the current cycle; a non-zero
	// entry in selfChangedFlags means the node requested re-execution on
	// the next cycle via MarkSelfChanged.
	changedFlags     []uint8
	selfChangedFlags []uint8
	// nodeIndex maps node key to the dense index used by changedFlags,
	// selfChangedFlags, and the outEdge.targetIdx field. Populated once
	// at construction; used only at API boundaries where a caller still
	// supplies a string key.
	nodeIndex map[string]int
	// markedFlags holds one flag per (node, param) pair that sources a
	// sequential-scope transition, indexed by the dense handle IDs
	// assigned at construction. A non-zero entry means MarkChanged was
	// called with a truthy output during the current cycle. Sequential
	// transitions fire on a fresh mark rather than on stale cached
	// truthiness, mirroring the conditional-edge firing semantic of the
	// pre-Scope scheduler. Marks are consumed when a transition fires
	// and cleared at end of cycle.
	markedFlags []uint8
	// nextHandleIdx is the next index to assign to a freshly-discovered
	// transition-source handle during construction. After construction
	// this is the length of markedFlags.
	nextHandleIdx int
	nodes         map[string]*node
	// currNode points at the node currently executing, cached so
	// MarkChanged / MarkSelfChanged callbacks skip the per-call map
	// lookup. nil between cycles.
	currNode *node
	// root mirrors prog.Root; always a parallel, always-live scope.
	root         *scopeState
	tolerance    telem.TimeSpan
	nextDeadline telem.TimeSpan
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
		nodes:            make(map[string]*node, len(prog.Nodes)),
		nodeIndex:        make(map[string]int, len(prog.Nodes)),
		changedFlags:     make([]uint8, len(prog.Nodes)),
		selfChangedFlags: make([]uint8, len(prog.Nodes)),
		tolerance:        tolerance,
	}

	for i, n := range prog.Nodes {
		s.nodeIndex[n.Key] = i
		impl := nodes[n.Key]
		rn := &node{key: n.Key, idx: i, Node: impl, outputByParam: map[string]int{}}
		// Pre-seed outputs in the node's declared order. Node impls
		// hardcode integer ordinals that match positions in this list;
		// the scheduler's edge / transition / activation wiring below
		// uses getOrCreateOutput, which finds these pre-seeded entries
		// by name.
		if impl != nil {
			for _, name := range impl.Outputs() {
				rn.outputByParam[name] = len(rn.outputs)
				rn.outputs = append(rn.outputs, outputResolved{
					param:         name,
					markHandleIdx: -1,
				})
			}
		}
		s.nodes[n.Key] = rn
	}
	for _, edge := range prog.Edges {
		src, ok := s.nodes[edge.Source.Node]
		if !ok {
			continue
		}
		tgt, ok := s.nodeIndex[edge.Target.Node]
		if !ok {
			continue
		}
		oidx := s.getOrCreateOutput(src, edge.Source.Param)
		src.outputs[oidx].edges = append(
			src.outputs[oidx].edges,
			outEdge{targetIdx: tgt, conditional: edge.Kind == ir.EdgeKindConditional},
		)
	}

	// Build the scope state tree rooted at prog.Root. buildScopeState
	// captures activation sources and assigns a dense index to every
	// (node, param) pair that sources a sequential-scope transition.
	// markedFlags is sized after the walk to match the final count.
	s.root = s.buildScopeState(&prog.Root)
	s.markedFlags = make([]uint8, s.nextHandleIdx)
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
		ms := memberState{key: m.Key, nodeIdx: -1}
		switch {
		case m.NodeRef != nil:
			ms.nodeKey = m.NodeRef.Key
			if idx, ok := s.nodeIndex[m.NodeRef.Key]; ok {
				ms.nodeIdx = idx
			}
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
	// cross-scope reference in source). Stored on the source node's
	// per-output table so markChanged needs zero scheduler-wide lookups.
	if scope.Liveness == ir.LivenessGated && scope.Activation != nil {
		if src, ok := s.nodes[scope.Activation.Node]; ok {
			oidx := s.getOrCreateOutput(src, scope.Activation.Param)
			src.outputs[oidx].activates = append(src.outputs[oidx].activates, state)
		}
	}
	if scope.Mode == ir.ScopeModeSequential {
		state.transitionOwner = computeTransitionOwners(state, scope.Transitions)
		state.transitionOnIdx = make([]int, len(scope.Transitions))
		for i, t := range scope.Transitions {
			state.transitionOnIdx[i] = s.registerTransitionHandle(t.On.Node, t.On.Param)
		}
	}
	return state
}

// getOrCreateOutput returns the node-local output index for the given
// param, allocating a fresh entry in n.outputs if the param is not yet
// known. Used by construction paths that wire edges, transition sources,
// and activation sources onto the owning node.
func (s *Scheduler) getOrCreateOutput(n *node, param string) int {
	if idx, ok := n.outputByParam[param]; ok {
		return idx
	}
	idx := len(n.outputs)
	n.outputs = append(n.outputs, outputResolved{param: param, markHandleIdx: -1})
	n.outputByParam[param] = idx
	return idx
}

// registerTransitionHandle assigns a dense markedFlags index to the
// node-local output entry and returns it. Idempotent — reused by multiple
// transitions returns the same index. Returns -1 if the node is unknown.
func (s *Scheduler) registerTransitionHandle(nodeKey, param string) int {
	n, ok := s.nodes[nodeKey]
	if !ok {
		return -1
	}
	oidx := s.getOrCreateOutput(n, param)
	if n.outputs[oidx].markHandleIdx == -1 {
		n.outputs[oidx].markHandleIdx = s.nextHandleIdx
		s.nextHandleIdx++
	}
	return n.outputs[oidx].markHandleIdx
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
	if idx, ok := s.nodeIndex[nodeKey]; ok {
		s.changedFlags[idx] = 1
	}
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

	clear(s.changedFlags)
	clear(s.markedFlags)
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
// they've been marked self-changed by a previous cycle. Members are stored
// in phase-flattened order in ss.members; we iterate that slice directly
// (using the IR phases only for phase-index lookup) so every member access
// uses its pre-resolved nodeIdx without a map lookup.
func (s *Scheduler) walkParallel(ss *scopeState) {
	flat := 0
	for phaseIdx, phase := range ss.ir.Phases {
		for range phase.Members {
			s.executeMember(phaseIdx, &ss.members[flat])
			flat++
		}
	}
}

// walkSequential executes the active member, then loops checking this
// scope's transitions for same-cycle cascading. Bounded by a budget of
// members+1 so a chain of N members can fire N consecutive transitions.
// Sequential members are "always on" while active — passing phaseIdx=0
// to executeMember makes the node run unconditionally, matching how
// phase-0 parallel members execute every cycle.
func (s *Scheduler) walkSequential(ss *scopeState) {
	budget := len(ss.members) + 1
	for range budget {
		if ss.activeMember < 0 {
			return
		}
		s.executeMember(0, &ss.members[ss.activeMember])
		if !s.evaluateTransitions(ss) {
			return
		}
	}
}

// executeMember executes a single member of a scope. If the member is a
// nested scope, the scope is walked (assuming it was previously
// activated — activation itself fires reactively when the activation
// handle's source calls MarkChanged). If the member is a NodeRef, the
// node runs when phaseIdx==0, when it has a pending upstream change, or
// when it was marked self-changed by a previous cycle. Sequential
// scopes pass phaseIdx=0 to force unconditional execution.
func (s *Scheduler) executeMember(phaseIdx int, m *memberState) {
	if m.scope != nil {
		s.walk(m.scope)
		return
	}
	if m.nodeIdx < 0 {
		return
	}
	wasSelfChanged := s.selfChangedFlags[m.nodeIdx] != 0
	if wasSelfChanged {
		s.selfChangedFlags[m.nodeIdx] = 0
	}
	if phaseIdx == 0 || s.changedFlags[m.nodeIdx] != 0 || wasSelfChanged {
		s.runNode(m.nodeKey)
	}
}

// runNode dispatches a node's Next method. Activation polling runs inside
// markChanged when the current node calls MarkChanged on an output that
// sources a gated scope's activation handle.
func (s *Scheduler) runNode(key string) {
	n, ok := s.nodes[key]
	if !ok {
		return
	}
	s.currNode = n
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
		handleIdx := ss.transitionOnIdx[i]
		if handleIdx < 0 || s.markedFlags[handleIdx] == 0 {
			continue
		}
		if !s.isHandleTruthy(t.On) {
			continue
		}
		s.markedFlags[handleIdx] = 0
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
// execution. Behavior differs by mode:
//
//   - Sequential scope: becomes active at member 0 and the member's node
//     is reset (or, if the member is a nested scope, activated).
//   - Parallel scope: every direct NodeRef member is reset, and every
//     nested gated scope member with no Activation handle is
//     cascade-activated. Gated children that have an Activation handle
//     wait for that handle to fire via markChanged — they are not
//     cascade-activated by their parent. This rule applies uniformly at
//     the root and at every nested parallel scope.
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
			if m.nodeIdx >= 0 {
				s.selfChangedFlags[m.nodeIdx] = 0
			}
			if n, ok := s.nodes[m.nodeKey]; ok {
				n.Reset()
			}
		case m.scope != nil &&
			m.scope.ir.Liveness == ir.LivenessGated &&
			m.scope.ir.Activation == nil:
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
		if m.nodeIdx >= 0 {
			s.selfChangedFlags[m.nodeIdx] = 0
		}
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
		if m.nodeIdx >= 0 {
			s.selfChangedFlags[m.nodeIdx] = 0
		}
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
		if m.isNodeRef() && m.nodeIdx >= 0 {
			s.selfChangedFlags[m.nodeIdx] = 0
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
// this output. Pure array access — zero hash lookups on the hot path.
// Continuous edges always propagate; conditional edges only propagate
// when the source output is truthy.
func (s *Scheduler) markChanged(outputIdx int) {
	n := s.currNode
	if outputIdx < 0 || outputIdx >= len(n.outputs) {
		return
	}
	out := &n.outputs[outputIdx]
	truthy := n.IsOutputTruthy(out.param)
	if truthy && out.markHandleIdx >= 0 {
		s.markedFlags[out.markHandleIdx] = 1
	}
	for _, edge := range out.edges {
		if edge.conditional && !truthy {
			continue
		}
		s.changedFlags[edge.targetIdx] = 1
	}
	for _, scope := range out.activates {
		if !scope.active {
			s.activateScope(scope)
		}
	}
}

// markSelfChanged requests that the current node execute again on the
// next scheduler cycle without requiring an upstream change.
func (s *Scheduler) markSelfChanged() {
	s.selfChangedFlags[s.currNode.idx] = 1
}

// reportError forwards a node-reported error to the configured handler
// (if any), annotating it with the node currently executing.
func (s *Scheduler) reportError(err error) {
	if s.errorHandler != nil {
		s.errorHandler.HandleError(s.nodeCtx.Context, s.currNode.key, err)
	}
}

// setDeadline lowers the scheduler's nextDeadline if d is earlier.
func (s *Scheduler) setDeadline(d telem.TimeSpan) {
	if d < s.nextDeadline {
		s.nextDeadline = d
	}
}
