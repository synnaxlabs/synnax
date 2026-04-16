// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package scheduler

import (
	"github.com/synnaxlabs/arc/ir"
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/x/telem"
)

// builder assembles a Scheduler from a compiled IR. It owns every piece of
// state needed only during wiring — the node-key map, the per-node
// param-to-ordinal maps, and the running markedFlags handle counter — so
// the resulting Scheduler holds only what the hot path consults. Builder
// state is dropped once build returns; *node pointers stay reachable
// through the Scheduler's scope tree.
type builder struct {
	// nodes maps node key to its runtime *node wrapper. Used to resolve
	// every string-keyed reference in the IR (edge endpoints, activation
	// handles, transition sources, NodeRef members) into a pointer the
	// Scheduler can hold directly.
	nodes map[string]*node
	// nodeIndex maps node key to its dense flag-slice index. Used to
	// resolve edge targets; not retained on the Scheduler.
	nodeIndex map[string]int
	// outputByParam holds the per-node "output name → ordinal" map used
	// to find or allocate matching entries in n.outputs while wiring.
	// One entry per *node; not exposed on node itself so the long-lived
	// wrapper stays lean.
	outputByParam map[*node]map[string]int
	// nextHandleIdx is the next index to assign to a freshly-discovered
	// transition-source handle. After build, this is the final length of
	// the Scheduler's markedFlags slice.
	nextHandleIdx int
}

// New creates a scheduler from a compiled IR and a set of runtime node
// instances keyed by ir.Node.Key. tolerance controls how early timer-based
// nodes may fire relative to their deadline.
func New(prog ir.IR, nodes map[string]rnode.Node, tolerance telem.TimeSpan) *Scheduler {
	return newBuilder(prog, nodes).build(prog, tolerance)
}

func newBuilder(prog ir.IR, runtimeNodes map[string]rnode.Node) *builder {
	b := &builder{
		nodes:         make(map[string]*node, len(prog.Nodes)),
		nodeIndex:     make(map[string]int, len(prog.Nodes)),
		outputByParam: make(map[*node]map[string]int, len(prog.Nodes)),
	}
	for i, n := range prog.Nodes {
		b.nodeIndex[n.Key] = i
		rn := &node{
			key:     n.Key,
			idx:     i,
			Node:    runtimeNodes[n.Key],
			outputs: make([]outputResolved, 0, len(n.Outputs)),
		}
		// Pre-seed outputs in the IR's declared order. Node impls hardcode
		// integer ordinals that match positions in this list; the wiring
		// passes below use getOrCreateOutput, which finds these pre-seeded
		// entries by name via the side map.
		params := make(map[string]int, len(n.Outputs))
		for j, p := range n.Outputs {
			params[p.Name] = j
			rn.outputs = append(rn.outputs, outputResolved{markHandleIdx: -1})
		}
		b.outputByParam[rn] = params
		b.nodes[n.Key] = rn
	}
	return b
}

// build wires edges, materializes the scope tree, and returns a fully
// initialized Scheduler with its root scope activated and its node-context
// callbacks bound. The receiver is no longer needed after this call.
func (b *builder) build(prog ir.IR, tolerance telem.TimeSpan) *Scheduler {
	for _, edge := range prog.Edges {
		src, ok := b.nodes[edge.Source.Node]
		if !ok {
			continue
		}
		target, ok := b.nodeIndex[edge.Target.Node]
		if !ok {
			continue
		}
		oIdx := b.getOrCreateOutput(src, edge.Source.Param)
		src.outputs[oIdx].edges = append(
			src.outputs[oIdx].edges,
			outEdge{targetIdx: target, conditional: edge.Kind == ir.EdgeKindConditional},
		)
	}
	s := &Scheduler{
		changedFlags:     make([]uint8, len(prog.Nodes)),
		selfChangedFlags: make([]uint8, len(prog.Nodes)),
		tolerance:        tolerance,
	}
	// Build the scope state tree rooted at prog.Root. buildScopeState
	// captures activation sources and assigns a dense index to every
	// (node, param) pair that sources a sequential-scope transition.
	// markedFlags is sized after the walk to match the final count.
	s.root = b.buildScopeState(&prog.Root)
	s.markedFlags = make([]uint8, b.nextHandleIdx)
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

// buildScopeState recursively constructs a scope mirror of the IR scope
// tree. The returned state is inert — activation runs only after build
// returns, via Scheduler.activateScope on the root.
func (b *builder) buildScopeState(sc *ir.Scope) *scope {
	state := &scope{
		ir:           sc,
		activeMember: -1,
	}
	appendMember := func(m ir.Member) {
		ms := member{key: m.Key}
		switch {
		case m.NodeRef != nil:
			ms.node = b.nodes[m.NodeRef.Key]
		case m.Scope != nil:
			ms.scope = b.buildScopeState(m.Scope)
		}
		state.members = append(state.members, ms)
	}
	switch sc.Mode {
	case ir.ScopeModeParallel:
		for _, phase := range sc.Phases {
			for _, m := range phase.Members {
				appendMember(m)
			}
		}
	case ir.ScopeModeSequential:
		for _, m := range sc.Members {
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
	if sc.Liveness == ir.LivenessGated && sc.Activation != nil {
		if src, ok := b.nodes[sc.Activation.Node]; ok {
			oidx := b.getOrCreateOutput(src, sc.Activation.Param)
			src.outputs[oidx].activates = append(src.outputs[oidx].activates, state)
		}
	}
	if sc.Mode == ir.ScopeModeSequential {
		b.resolveTransitions(state, sc.Transitions)
	}
	return state
}

// resolveTransitions populates the parallel transition slices on state from
// the IR's transition list. Each transition's on-handle is resolved to a
// (*node, outputIdx) pair so the hot-path truthy check is a single virtual
// call. The handle is also assigned a dense markedFlags index for the
// fresh-mark check, and the source node is mapped back to its owning
// member so transitions originating in inactive siblings can be skipped.
// Transitions whose on-node is unknown get nil/-1 throughout and are
// silently skipped at evaluation time.
func (b *builder) resolveTransitions(state *scope, transitions []ir.Transition) {
	nodeToMember := make(map[*node]int)
	for i := range state.members {
		collectMemberNodes(&state.members[i], i, nodeToMember)
	}
	state.transitionOwner = make([]int, len(transitions))
	state.transitionOnIdx = make([]int, len(transitions))
	state.transitionOnNode = make([]*node, len(transitions))
	state.transitionOnOutputIdx = make([]int, len(transitions))
	for i, t := range transitions {
		on, ok := b.nodes[t.On.Node]
		if !ok {
			state.transitionOwner[i] = -1
			state.transitionOnIdx[i] = -1
			state.transitionOnOutputIdx[i] = -1
			continue
		}
		oIdx := b.getOrCreateOutput(on, t.On.Param)
		if on.outputs[oIdx].markHandleIdx == -1 {
			on.outputs[oIdx].markHandleIdx = b.nextHandleIdx
			b.nextHandleIdx++
		}
		state.transitionOnNode[i] = on
		state.transitionOnOutputIdx[i] = oIdx
		state.transitionOnIdx[i] = on.outputs[oIdx].markHandleIdx
		if owner, ok := nodeToMember[on]; ok {
			state.transitionOwner[i] = owner
		} else {
			state.transitionOwner[i] = -1
		}
	}
	// Pre-filter transitions per member so evaluateTransitions can iterate
	// a short list instead of scanning all N transitions per cascade step.
	// Each member's list contains transitions owned by that member plus
	// every external transition (owner == -1), in source order.
	state.transitionsForMember = make([][]int, len(state.members))
	for m := range state.members {
		var list []int
		for i, owner := range state.transitionOwner {
			if owner == -1 || owner == m {
				list = append(list, i)
			}
		}
		state.transitionsForMember[m] = list
	}
}

// getOrCreateOutput returns the node-local output index for the given
// param, allocating a fresh entry in n.outputs if the param is not yet
// known. Used by every wiring pass — edges, transition sources, and
// activation sources — onto the owning node.
func (b *builder) getOrCreateOutput(n *node, param string) int {
	params := b.outputByParam[n]
	if idx, ok := params[param]; ok {
		return idx
	}
	idx := len(n.outputs)
	n.outputs = append(n.outputs, outputResolved{markHandleIdx: -1})
	params[param] = idx
	return idx
}

// collectMemberNodes adds every node owned (directly or transitively) by
// the given member to the provided map, tagged with the member index.
// Unresolved NodeRefs are skipped.
func collectMemberNodes(m *member, idx int, out map[*node]int) {
	if m.isNodeRef() {
		if m.node != nil {
			out[m.node] = idx
		}
		return
	}
	if m.scope != nil {
		for i := range m.scope.members {
			collectMemberNodes(&m.scope.members[i], idx, out)
		}
	}
}
