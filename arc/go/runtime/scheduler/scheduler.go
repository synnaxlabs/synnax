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

// outEdge is the pre-resolved form of an ir.Edge cached on the source node.
type outEdge struct {
	// targetIdx is the destination node's flag-slice index.
	targetIdx int
	// conditional means the edge only fires when the source output is truthy.
	conditional bool
}

// outputResolved is the per-output propagation table built at construction
// and indexed by the node-local output ordinal. Hot-path access is pure
// array indexing.
type outputResolved struct {
	// edges are the outgoing dataflow edges from this output.
	edges []outEdge
	// markHandleIdx is the markedFlags index for sequential-scope
	// transitions sourced from this output, or -1 if none.
	markHandleIdx int
	// activates are the gated scopes whose activation handle is this output.
	activates []*scope
}

// node pairs a runtime node with its pre-resolved per-output propagation
// tables.
type node struct {
	rnode.Node
	// outputs is the propagation table indexed by output ordinal.
	outputs []outputResolved
	// key is the IR node key, retained for error reporting.
	key string
	// idx is the node's position in changedFlags / selfChangedFlags.
	idx int
}

// member mirrors an ir.Member: exactly one of node or scope is set.
// An unresolved node key leaves both nil and is skipped at execution.
type member struct {
	// key is the IR member key, used for `=> name` transition lookups.
	key string
	// node is the resolved leaf-node target, nil for scope members and
	// unresolved node keys.
	node *node
	// scope is the nested scope state, nil for leaf-node members.
	scope *scope
}

// isNode reports whether this member refers to a dataflow node rather
// than a nested scope. Returns true for unresolved node keys as well; callers
// must nil-check m.node before dereferencing.
func (m *member) isNode() bool { return m.scope == nil }

// scope is the runtime mirror of an ir.Scope, holding activation
// bookkeeping for one scope.
type scope struct {
	// ir is the static IR scope this state mirrors.
	ir *ir.Scope
	// active gates whether walk descends into this scope.
	active bool
	// activeStep is the running sequential step's index, or -1 when
	// inactive or in a parallel scope.
	activeStep int
	// members are flattened in execution order: stratum-major for parallel
	// scopes, sequence order for sequential.
	members []member
	// memberByKey resolves `=> name` transition targets to member indices.
	memberByKey map[string]int
	// transitionOwner[i] is the step index that owns transition i's
	// `on`-handle source, or -1 if the source is outside this scope.
	// Transitions owned by inactive steps are skipped each cycle.
	transitionOwner []int
	// transitionOnIdx[i] is the markedFlags index for transition i's
	// `on`-handle, or -1 if unresolved.
	transitionOnIdx []int
	// transitionOnNode[i] is the source node of transition i's `on`-handle,
	// or nil if unresolved.
	transitionOnNode []*node
	// transitionOnOutputIdx[i] is the output ordinal on transitionOnNode[i]
	// for transition i's `on`-handle, or -1 if unresolved. Paired with
	// transitionOnNode[i] so the truthy check is a direct virtual call.
	transitionOnOutputIdx []int
	// transitionsForStep[s] holds transition indices to evaluate when
	// step s is active, in source order. Each entry is the union of:
	//   - transitions whose owner is s (sourced inside that step)
	//   - transitions whose owner is -1 (sourced outside the scope, always
	//     evaluated)
	// Pre-computed once at construction so evaluateTransitions iterates a
	// short per-step list instead of the full transitions slice. Reduces
	// sequential-cascade work from O(N²) to O(N·K) where K is the typical
	// per-step transition count (almost always 1).
	transitionsForStep [][]int
}

// Scheduler executes one tick of a compiled Arc program per Next call.
// Construction is one-time and happens through New (see builder.go).
type Scheduler struct {
	// nodeCtx is the rnode.Context passed to every node's Next; rebound
	// per cycle with the latest elapsed time and run reason.
	nodeCtx rnode.Context
	// errorHandler receives errors raised by nodes via ctx.ReportError;
	// nil drops them.
	errorHandler ErrorHandler
	// changedFlags[i] is set when node i has a pending upstream change
	// for the current cycle. Cleared at end of cycle.
	changedFlags []uint8
	// selfChangedFlags[i] is set by node i via MarkSelfChanged to request
	// replay on the next cycle. Cleared when the replay runs or when the
	// owning member is deactivated.
	selfChangedFlags []uint8
	// markedFlags[i] is set when the (node, output) pair behind transition
	// handle i fired truthy this cycle. Cleared at end of cycle so
	// transitions fire on fresh marks, not stale truthiness.
	markedFlags []uint8
	// currNode is the node whose Next is currently executing, cached so
	// MarkChanged / MarkSelfChanged callbacks know whom they came from.
	currNode *node
	// root is the program's parallel + always-live root scope.
	root *scope
	// tolerance is how early a timer-based node may fire relative to its
	// deadline.
	tolerance telem.TimeSpan
	// nextDeadline is the earliest deadline reported by any node during
	// the previous Next; reset to TimeSpanMax at the start of each cycle.
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

// SetErrorHandler configures the handler for node execution errors.
func (s *Scheduler) SetErrorHandler(handler ErrorHandler) {
	s.errorHandler = handler
}

// NextDeadline returns the earliest deadline reported by any node during
// the previous Next call. Callers use this to sleep until the scheduler
// has work to do.
func (s *Scheduler) NextDeadline() telem.TimeSpan { return s.nextDeadline }

// Next executes one cycle of the reactive computation. Nodes with pending
// changes execute in stratum order; sequential scopes advance via their
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

// walk executes one pass over a scope; no-op if inactive.
func (s *Scheduler) walk(ss *scope) {
	if !ss.active {
		return
	}
	if ss.ir.Mode == ir.ScopeModeSequential {
		s.walkSequential(ss)
		return
	}
	s.walkParallel(ss)
}

// walkParallel runs every member of a parallel scope in stratum order.
// Members are stored stratum-flattened in ss.members; ss.ir.Strata is read
// only for the stratum-index argument passed to executeMember.
func (s *Scheduler) walkParallel(ss *scope) {
	flat := 0
	for stratumIdx, stratum := range ss.ir.Strata {
		for range stratum {
			s.executeMember(stratumIdx, &ss.members[flat])
			flat++
		}
	}
}

// walkSequential executes the active step and loops on its transitions
// for same-cycle cascading. Bounded by len(members)+1 so a chain of N
// steps can fire N transitions. stratumIdx=0 forces the active step to
// run unconditionally, matching stratum-0 parallel semantics.
func (s *Scheduler) walkSequential(ss *scope) {
	budget := len(ss.members) + 1
	for range budget {
		if ss.activeStep < 0 {
			return
		}
		s.executeMember(0, &ss.members[ss.activeStep])
		if !s.evaluateTransitions(ss) {
			return
		}
	}
}

// executeMember walks a nested-scope member or runs a leaf-node member.
// A leaf runs when stratumIdx==0, when changedFlags is set, or when the
// node was self-changed on a prior cycle.
func (s *Scheduler) executeMember(stratumIdx int, m *member) {
	if m.scope != nil {
		s.walk(m.scope)
		return
	}
	if m.node == nil {
		return
	}
	idx := m.node.idx
	wasSelfChanged := s.selfChangedFlags[idx] != 0
	if wasSelfChanged {
		s.selfChangedFlags[idx] = 0
	}
	if stratumIdx == 0 || s.changedFlags[idx] != 0 || wasSelfChanged {
		s.currNode = m.node
		s.currNode.Next(s.nodeCtx)
	}
}

// evaluateTransitions fires the first transition whose `on` handle was
// freshly marked truthy by the active step this cycle. Inactive-owner
// transitions and stale truthiness without a fresh mark are both ignored
// — the latter prevents latched comparisons from driving repeat
// transitions. Iterates the pre-filtered transitionsForStep list for
// the active step, which interleaves external transitions and the
// active step's own transitions in source order. Returns true if a
// transition fired.
func (s *Scheduler) evaluateTransitions(ss *scope) bool {
	if ss.activeStep < 0 || ss.activeStep >= len(ss.transitionsForStep) {
		return false
	}
	for _, i := range ss.transitionsForStep[ss.activeStep] {
		handleIdx := ss.transitionOnIdx[i]
		if handleIdx < 0 || s.markedFlags[handleIdx] == 0 {
			continue
		}
		if !ss.transitionOnNode[i].IsOutputTruthy(ss.transitionOnOutputIdx[i]) {
			continue
		}
		s.markedFlags[handleIdx] = 0
		if ss.activeStep >= 0 {
			s.deactivateStep(&ss.members[ss.activeStep])
		}
		t := ss.ir.Transitions[i]
		if t.TargetKey == nil {
			s.deactivateScope(ss)
		} else {
			idx, ok := ss.memberByKey[*t.TargetKey]
			if !ok {
				return false
			}
			s.activateSequentialStep(ss, idx)
		}
		return true
	}
	return false
}

// resetLeafNode clears selfChanged and calls Reset on m's node.
func (s *Scheduler) resetLeafNode(m *member) {
	if n := m.node; m.isNode() && n != nil {
		s.selfChangedFlags[n.idx] = 0
		n.Reset()
	}
}

// clearLeafNodeSelfChanged clears selfChanged on m's node.
func (s *Scheduler) clearLeafNodeSelfChanged(m *member) {
	if n := m.node; m.isNode() && n != nil {
		s.selfChangedFlags[n.idx] = 0
	}
}

// activateScope marks a scope active and primes its members. Sequential
// scopes activate step 0; parallel scopes reset every leaf-node member and
// cascade-activate always-live nested scopes. Gated children wait for their
// Activation handle to fire via markChanged; gated children with no handle
// stay inert (used for named top-level scopes awaiting an external trigger).
func (s *Scheduler) activateScope(ss *scope) {
	ss.active = true
	if ss.ir.Mode == ir.ScopeModeSequential {
		if len(ss.members) > 0 {
			s.activateSequentialStep(ss, 0)
		}
		return
	}
	for i := range ss.members {
		m := &ss.members[i]
		if m.isNode() {
			s.resetLeafNode(m)
			continue
		}
		if m.scope != nil && m.scope.ir.Liveness == ir.LivenessAlways {
			s.activateScope(m.scope)
		}
	}
}

// activateSequentialStep points the active pointer at idx and resets
// (or cascade-activates) that step.
func (s *Scheduler) activateSequentialStep(ss *scope, idx int) {
	ss.activeStep = idx
	m := &ss.members[idx]
	if m.isNode() {
		s.resetLeafNode(m)
		return
	}
	if m.scope != nil {
		s.activateScope(m.scope)
	}
}

// deactivateStep clears selfChanged for the step's node, or marks a
// nested scope inactive. Nested-scope state freezes and is overwritten
// on the next parent activation.
func (s *Scheduler) deactivateStep(m *member) {
	if m.scope != nil {
		s.deactivateScope(m.scope)
		return
	}
	s.clearLeafNodeSelfChanged(m)
}

// deactivateScope marks a scope inactive and clears selfChanged on its
// direct leaf-node members. Does not recurse — nested scope state freezes
// until the next activation overwrites it.
func (s *Scheduler) deactivateScope(ss *scope) {
	if ss.ir.Mode == ir.ScopeModeSequential {
		ss.activeStep = -1
	}
	for i := range ss.members {
		s.clearLeafNodeSelfChanged(&ss.members[i])
	}
	ss.active = false
}

// markChanged propagates the current node's output to downstream nodes,
// records a fresh transition mark when truthy, and fires any gated scope
// activations attached to this output. Conditional edges propagate only
// when the source is truthy.
func (s *Scheduler) markChanged(outputIdx int) {
	if outputIdx < 0 || outputIdx >= len(s.currNode.outputs) {
		return
	}
	out := &s.currNode.outputs[outputIdx]
	truthy := s.currNode.IsOutputTruthy(outputIdx)
	if truthy && out.markHandleIdx >= 0 {
		s.markedFlags[out.markHandleIdx] = 1
	}
	for _, edge := range out.edges {
		if !edge.conditional || truthy {
			s.changedFlags[edge.targetIdx] = 1
		}
	}
	for _, sc := range out.activates {
		if !sc.active {
			s.activateScope(sc)
		}
	}
}

// markSelfChanged schedules the current node to replay on the next cycle
// without an upstream change.
func (s *Scheduler) markSelfChanged() {
	s.selfChangedFlags[s.currNode.idx] = 1
}

// reportError forwards err to the error handler, tagged with the
// currently-executing node's key. No-op if no handler is set.
func (s *Scheduler) reportError(err error) {
	if s.errorHandler != nil {
		s.errorHandler.HandleError(s.nodeCtx.Context, s.currNode.key, err)
	}
}

// setDeadline lowers nextDeadline to d if d is earlier.
func (s *Scheduler) setDeadline(d telem.TimeSpan) {
	if d < s.nextDeadline {
		s.nextDeadline = d
	}
}
