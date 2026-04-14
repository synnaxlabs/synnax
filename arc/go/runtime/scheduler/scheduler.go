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
	"context"

	"github.com/synnaxlabs/arc/ir"
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// node holds the runtime state for a single node in the scheduler.
type node struct {
	rnode.Node
	outgoing map[string][]ir.Edge
	key      string
}

// stepState holds runtime state for a single step in a sequence.
type stepState struct {
	ir      ir.Step
	subSeqs []sequenceState // sub-sequences within a stage step
	subSeq  *sequenceState  // nested sequence for a sequence step
}

// sequenceState holds runtime state for a sequence.
type sequenceState struct {
	ir            ir.Sequence
	steps         []stepState
	activeStepIdx int
	// flowNodeOwner maps node keys to the step index that owns them.
	// Only populated for sequences that contain flow steps.
	flowNodeOwner map[string]int
	// flowDataNodes is the subset of flowNodeOwner that are data nodes (not entry
	// nodes). Data nodes execute unconditionally when their step is active.
	// Entry nodes only execute when marked changed.
	flowDataNodes set.Set[string]
}

// transitionTarget identifies the target of an entry node activation.
type transitionTarget struct {
	seq     *sequenceState
	stepIdx int
}

// Scheduler orchestrates the execution of nodes in topological order.
type Scheduler struct {
	nodeCtx            rnode.Context
	errorHandler       ErrorHandler
	transitions        map[string]transitionTarget
	boundaries         map[string]*sequenceState
	changed            set.Set[string]
	selfChanged        set.Set[string]
	nodes              map[string]node
	currNodeKey        string
	globalStrata       ir.Strata
	sequences          []sequenceState
	maxConvergenceIter int
	currSeq            *sequenceState
	tolerance          telem.TimeSpan
	nextDeadline       telem.TimeSpan
	transitioned       bool
}

// ErrorHandler receives errors from node execution.
type ErrorHandler interface {
	HandleError(ctx context.Context, nodeKey string, err error)
}

// ErrorHandlerFunc is an adapter to allow ordinary functions to be used as ErrorHandlers.
type ErrorHandlerFunc func(ctx context.Context, nodeKey string, err error)

func (f ErrorHandlerFunc) HandleError(ctx context.Context, nodeKey string, err error) {
	f(ctx, nodeKey, err)
}

// New creates a scheduler from an IR program and node instances.
func New(prog ir.IR, nodes map[string]rnode.Node, tolerance telem.TimeSpan) *Scheduler {
	s := &Scheduler{
		nodes:        make(map[string]node, len(prog.Nodes)),
		globalStrata: prog.Root.Strata,
		transitions:  make(map[string]transitionTarget),
		boundaries:   make(map[string]*sequenceState),
		changed:      make(set.Set[string], len(prog.Nodes)),
		selfChanged:  make(set.Set[string]),
		tolerance:    tolerance,
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

	s.sequences = make([]sequenceState, len(prog.Root.Sequences))
	for i, seq := range prog.Root.Sequences {
		s.sequences[i] = buildSequenceState(seq)
		s.registerTransitions(&s.sequences[i])
		s.registerBoundaries(&s.sequences[i])
		s.maxConvergenceIter += countSteps(seq)
	}

	s.nodeCtx = rnode.Context{
		MarkChanged:     s.markChanged,
		MarkSelfChanged: s.markSelfChanged,
		SetDeadline:     s.setDeadline,
		ReportError:     s.reportError,
		ActivateStage:   s.transitionStep,
	}

	return s
}

// buildSequenceState recursively builds runtime state from an IR sequence.
func buildSequenceState(seq ir.Sequence) sequenceState {
	state := sequenceState{
		ir:            seq,
		steps:         make([]stepState, len(seq.Steps)),
		activeStepIdx: -1,
	}

	hasFlowSteps := false
	for i, step := range seq.Steps {
		ss := stepState{ir: step}
		if step.Stage != nil {
			for _, subSeq := range step.Stage.Sequences {
				ss.subSeqs = append(ss.subSeqs, buildSequenceState(subSeq))
			}
		}
		if step.Sequence != nil {
			nested := buildSequenceState(*step.Sequence)
			ss.subSeq = &nested
		}
		if step.Flow != nil {
			hasFlowSteps = true
		}
		state.steps[i] = ss
	}

	if hasFlowSteps {
		state.flowNodeOwner = make(map[string]int)
		state.flowDataNodes = make(set.Set[string])
		for i, step := range seq.Steps {
			if step.Flow != nil {
				for _, nodeKey := range step.Flow.Nodes {
					state.flowNodeOwner[nodeKey] = i
					state.flowDataNodes.Add(nodeKey)
				}
			}
		}
	}

	return state
}

// registerTransitions maps entry node keys to their transition targets.
func (s *Scheduler) registerTransitions(seq *sequenceState) {
	for i, step := range seq.steps {
		ek := "entry_" + seq.ir.Key + "_" + step.ir.Key
		s.transitions[ek] = transitionTarget{seq: seq, stepIdx: i}
		if step.ir.Stage != nil {
			for j := range step.subSeqs {
				s.registerTransitions(&step.subSeqs[j])
			}
		}
		if step.subSeq != nil {
			s.registerTransitions(step.subSeq)
		}
	}
}

// registerBoundaries maps boundary keys to their child sequence states.
func (s *Scheduler) registerBoundaries(seq *sequenceState) {
	for i, step := range seq.steps {
		if step.ir.Stage != nil {
			for j := range step.subSeqs {
				subBk := stratifier.BoundaryKey(step.subSeqs[j].ir.Key)
				s.boundaries[subBk] = &seq.steps[i].subSeqs[j]
				s.registerBoundaries(&step.subSeqs[j])
			}
		}
		if step.ir.Sequence != nil {
			bk := stratifier.BoundaryKey(step.ir.Key)
			s.boundaries[bk] = step.subSeq
			s.registerBoundaries(step.subSeq)
		}
	}
}

// countSteps returns the total number of steps across a sequence and its children.
func countSteps(seq ir.Sequence) int {
	count := len(seq.Steps)
	for _, step := range seq.Steps {
		if step.Stage != nil {
			for _, subSeq := range step.Stage.Sequences {
				count += countSteps(subSeq)
			}
		}
		if step.Sequence != nil {
			count += countSteps(*step.Sequence)
		}
	}
	return count
}

// SetErrorHandler configures the handler for node execution errors.
func (s *Scheduler) SetErrorHandler(handler ErrorHandler) {
	s.errorHandler = handler
}

// MarkNodeChanged marks a node as changed, scheduling it for execution.
func (s *Scheduler) MarkNodeChanged(nodeKey string) {
	s.changed.Add(nodeKey)
}

// markChanged propagates changes from the current node's output to downstream nodes.
// For continuous edges, always propagates. For conditional edges, only propagates
// when the source output is truthy.
func (s *Scheduler) markChanged(param string) {
	n := s.nodes[s.currNodeKey]
	for _, edge := range n.outgoing[param] {
		if edge.Kind == ir.EdgeKindConditional {
			if n.IsOutputTruthy(param) {
				s.changed.Add(edge.Target.Node)
			}
			continue
		}
		s.changed.Add(edge.Target.Node)
	}
}

func (s *Scheduler) markSelfChanged() {
	s.selfChanged.Add(s.currNodeKey)
}

func (s *Scheduler) reportError(err error) {
	if s.errorHandler != nil {
		s.errorHandler.HandleError(s.nodeCtx.Context, s.currNodeKey, err)
	}
}

// Next executes one cycle of the reactive computation.
func (s *Scheduler) Next(ctx context.Context, elapsed telem.TimeSpan, reason rnode.RunReason) {
	s.nextDeadline = telem.TimeSpanMax
	s.nodeCtx.Context = ctx
	s.nodeCtx.Elapsed = elapsed
	s.nodeCtx.Tolerance = s.tolerance
	s.nodeCtx.Reason = reason
	s.currSeq = nil
	s.execStrata(s.globalStrata, nil)
	s.execSequences()
	clear(s.changed)
}

// NextDeadline returns the minimum deadline reported by nodes during the last Next() call.
func (s *Scheduler) NextDeadline() telem.TimeSpan { return s.nextDeadline }

func (s *Scheduler) setDeadline(d telem.TimeSpan) {
	if d < s.nextDeadline {
		s.nextDeadline = d
	}
}

// execStrata executes nodes in a strata, propagating changes between layers.
// activeSeq is the sequence context for filtering flow nodes (nil for global/stage).
func (s *Scheduler) execStrata(strata ir.Strata, activeSeq *sequenceState) {
	clear(s.changed)
	s.transitioned = false
	inContext := s.currSeq != nil
	for i, stratum := range strata {
		for _, key := range stratum {
			if subSeq, ok := s.boundaries[key]; ok {
				if subSeq.activeStepIdx != -1 {
					// Preserve the parent's changed/transitioned state across
					// the recursive sub-context execution. execStrata clears
					// changed on entry; without saving, propagations from
					// earlier nodes in this stratum would be lost.
					savedChanged := s.changed
					savedTransitioned := s.transitioned
					s.changed = make(set.Set[string])
					s.execSequenceStep(subSeq)
					s.changed = savedChanged
					s.transitioned = savedTransitioned
				}
				continue
			}

			// Stage step boundaries in a sequence's strata are markers used for
			// ordering and dependency tracking only. Their step is activated and
			// executed via execSequenceStep, not through the parent's strata
			// walk. Skip them here.
			if _, isNode := s.nodes[key]; !isNode {
				continue
			}

			if activeSeq != nil && activeSeq.flowDataNodes.Contains(key) {
				if activeSeq.flowNodeOwner[key] != activeSeq.activeStepIdx {
					continue
				}
			}

			isActiveFlowNode := false
			if activeSeq != nil && activeSeq.flowDataNodes.Contains(key) {
				if activeSeq.flowNodeOwner[key] == activeSeq.activeStepIdx {
					isActiveFlowNode = true
				}
			}

			wasSelfChanged := s.selfChanged.Contains(key)
			if wasSelfChanged {
				delete(s.selfChanged, key)
			}
			if i == 0 || s.changed.Contains(key) || wasSelfChanged || isActiveFlowNode {
				s.currNodeKey = key
				s.nodes[key].Next(s.nodeCtx)
				if inContext && s.transitioned {
					return
				}
			}
		}
	}
}

// execSequences runs the convergence loop across all sequences.
func (s *Scheduler) execSequences() {
	for iter := 0; iter < s.maxConvergenceIter; iter++ {
		stable := true
		for i := range s.sequences {
			seq := &s.sequences[i]
			if seq.activeStepIdx == -1 {
				continue
			}
			prevStepIdx := seq.activeStepIdx
			s.execSequenceStep(seq)
			if seq.activeStepIdx != prevStepIdx {
				stable = false
			}
		}
		if stable {
			return
		}
	}
}

// execSequenceStep executes the active step of a sequence.
func (s *Scheduler) execSequenceStep(seq *sequenceState) {
	if seq.activeStepIdx < 0 || seq.activeStepIdx >= len(seq.steps) {
		return
	}
	step := &seq.steps[seq.activeStepIdx]
	prevSeq := s.currSeq
	s.currSeq = seq

	switch {
	case step.ir.Stage != nil:
		s.execStrata(step.ir.Stage.Strata, nil)
	case step.ir.Flow != nil:
		s.execStrata(seq.ir.Strata, seq)
	case step.ir.Sequence != nil && step.subSeq != nil:
		if step.subSeq.activeStepIdx != -1 {
			s.execSequenceStep(step.subSeq)
		}
	}

	s.currSeq = prevSeq
}

// transitionStep transitions to the step associated with the currently executing
// entry node. This deactivates the source step and activates the target step.
// If entering from global strata and the target's sequence already has an active
// step, this is a no-op to prevent re-entering a sequence that has already started.
func (s *Scheduler) transitionStep() {
	target, ok := s.transitions[s.currNodeKey]
	if !ok {
		return
	}
	if s.currSeq == nil && target.seq.activeStepIdx != -1 {
		return
	}

	// Deactivate the current step in the target's sequence.
	if target.seq.activeStepIdx != -1 {
		s.deactivateStep(target.seq)
	}

	// Activate the target step.
	s.activateStep(target.seq, target.stepIdx)
	s.transitioned = true
}

// activateStep activates a step, resetting its nodes on entry.
func (s *Scheduler) activateStep(seq *sequenceState, stepIdx int) {
	seq.activeStepIdx = stepIdx
	step := &seq.steps[stepIdx]

	switch {
	case step.ir.Stage != nil:
		s.resetStrata(step.ir.Stage.Strata)
		for i := range step.subSeqs {
			s.enterSequence(&step.subSeqs[i])
		}
	case step.ir.Flow != nil:
		for _, nodeKey := range step.ir.Flow.Nodes {
			delete(s.selfChanged, nodeKey)
			if n, ok := s.nodes[nodeKey]; ok {
				n.Reset()
			}
		}
	case step.ir.Sequence != nil && step.subSeq != nil:
		s.enterSequence(step.subSeq)
	}
}

// deactivateStep clears self-changed state for the current step.
func (s *Scheduler) deactivateStep(seq *sequenceState) {
	step := &seq.steps[seq.activeStepIdx]
	switch {
	case step.ir.Stage != nil:
		s.clearSelfChanged(step.ir.Stage.Strata)
	case step.ir.Flow != nil:
		for _, nodeKey := range step.ir.Flow.Nodes {
			delete(s.selfChanged, nodeKey)
		}
	}
	seq.activeStepIdx = -1
}

// enterSequence activates a sequence at step 0.
func (s *Scheduler) enterSequence(seq *sequenceState) {
	if len(seq.steps) == 0 {
		return
	}
	s.activateStep(seq, 0)
}

// clearSelfChanged removes all nodes in a strata from the selfChanged set.
func (s *Scheduler) clearSelfChanged(strata ir.Strata) {
	for _, stratum := range strata {
		for _, key := range stratum {
			delete(s.selfChanged, key)
		}
	}
}

// resetStrata resets all nodes in a strata to their initial state.
func (s *Scheduler) resetStrata(strata ir.Strata) {
	for _, stratum := range strata {
		for _, key := range stratum {
			delete(s.selfChanged, key)
			if n, ok := s.nodes[key]; ok {
				n.Reset()
			}
		}
	}
}
