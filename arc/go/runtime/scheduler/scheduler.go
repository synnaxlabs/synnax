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
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// node holds the runtime state for a single node in the scheduler.
type node struct {
	// node is the executable node instance.
	rnode.Node
	// outgoing contains edges where this node is the source, keyed by output param.
	// Used for efficient change propagation to downstream nodes.
	outgoing map[string][]ir.Edge
	key      string
}

// stage holds the runtime state for a single stage within a sequence.
type stage struct {
	// strata defines the topological execution order for nodes in this stage.
	strata ir.Strata
}

// sequenceState holds the runtime state for a sequence.
type sequenceState struct {
	// stages contains the ordered list of stages in this sequence.
	stages []stage
	// activeStageIdx is the index of the currently active stage, or -1 if none.
	activeStageIdx int
}

type transitionTarget struct {
	seqIdx, stageIdx int
}

// Scheduler orchestrates the execution of nodes in topological order.
// It maintains the execution graph, tracks changed nodes, and propagates changes
// through the dependency graph. It supports stage-based filtering for sequences.
type Scheduler struct {
	// nodeCtx is a reusable context struct passed to nodes during execution.
	nodeCtx rnode.Context
	// errorHandler receives errors from node execution.
	errorHandler ErrorHandler
	// transitions maps entry node keys to their target (seqIdx, stageIdx).
	transitions map[string]transitionTarget
	// changed tracks which nodes need execution in the current strata pass.
	changed set.Set[string]
	// selfChanged tracks nodes that requested re-execution on the next cycle.
	// Unlike changed, selfChanged persists across scheduler.Next() calls.
	selfChanged set.Set[string]
	// nodes maps node keys to their runtime state.
	nodes map[string]node
	// Current execution context
	// currNodeKey is the key of the currently executing node.
	currNodeKey string
	// globalStrata defines the topological execution order for global nodes.
	globalStrata ir.Strata
	// sequences holds the runtime state for each sequence.
	sequences []sequenceState
	// maxConvergenceIterations is the maximum iterations for stage convergence loop.
	maxConvergenceIterations int
	// currStageIdx is the index of the currently executing stage, or -1 if none.
	currStageIdx int
	// currSeqIdx is the index of the currently executing sequence, or -1 if global.
	currSeqIdx int
	// tolerance is the timing tolerance for interval/wait comparisons.
	tolerance telem.TimeSpan
	// nextDeadline is the minimum deadline (absolute elapsed time) reported by
	// nodes during the current Next() call. Reset to max at the start of each call.
	nextDeadline telem.TimeSpan
	// transitioned is set to true when transitionStage fires during strata execution.
	// Used to stop executing further nodes so the first transition wins.
	transitioned bool
}

// ErrorHandler receives errors from node execution.
// Implementations can log, aggregate, or handle errors in custom ways.
type ErrorHandler interface {
	// HandleError is called when a node reports an error during execution.
	HandleError(ctx context.Context, nodeKey string, err error)
}

// ErrorHandlerFunc is an adapter to allow ordinary functions to be used as ErrorHandlers.
type ErrorHandlerFunc func(ctx context.Context, nodeKey string, err error)

// HandleError implements ErrorHandler by calling the function.
func (f ErrorHandlerFunc) HandleError(ctx context.Context, nodeKey string, err error) {
	f(ctx, nodeKey, err)
}

// New creates a scheduler from an IR program and node instances.
// The scheduler organizes nodes into strata for topological execution and
// builds the change propagation graph from the IR edges.
func New(prog ir.IR, nodes map[string]rnode.Node, tolerance telem.TimeSpan) *Scheduler {
	s := &Scheduler{
		nodes:        make(map[string]node, len(prog.Nodes)),
		globalStrata: prog.Strata,
		sequences:    make([]sequenceState, len(prog.Sequences)),
		transitions:  make(map[string]transitionTarget),
		changed:      make(set.Set[string], len(prog.Nodes)),
		selfChanged:  make(set.Set[string]),
		currSeqIdx:   -1,
		currStageIdx: -1,
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

	for seqIdx, seq := range prog.Sequences {
		seqState := sequenceState{
			stages:         make([]stage, len(seq.Stages)),
			activeStageIdx: -1,
		}
		s.maxConvergenceIterations += len(seq.Stages)

		for stageIdx, irStage := range seq.Stages {
			seqState.stages[stageIdx] = stage{
				strata: irStage.Strata,
			}
			entryKey := "entry_" + seq.Key + "_" + irStage.Key
			s.transitions[entryKey] = transitionTarget{
				seqIdx:   seqIdx,
				stageIdx: stageIdx,
			}
		}
		s.sequences[seqIdx] = seqState
	}

	s.nodeCtx = rnode.Context{
		MarkChanged:     s.markChanged,
		MarkSelfChanged: s.markSelfChanged,
		SetDeadline:     s.setDeadline,
		ReportError:     s.reportError,
		ActivateStage:   s.transitionStage,
	}

	return s
}

// SetErrorHandler configures the handler for node execution errors.
// If no handler is set, errors are silently ignored.
func (s *Scheduler) SetErrorHandler(handler ErrorHandler) {
	s.errorHandler = handler
}

// MarkNodeChanged marks a node as changed, scheduling it for execution in the next cycle.
// This is used externally to trigger execution based on external events or inputs.
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

// markSelfChanged adds the currently executing node to the selfChanged set,
// requesting re-execution on the next scheduler cycle.
func (s *Scheduler) markSelfChanged() {
	s.selfChanged.Add(s.currNodeKey)
}

// reportError reports an error from the currently executing node.
func (s *Scheduler) reportError(err error) {
	if s.errorHandler != nil {
		s.errorHandler.HandleError(s.nodeCtx.Context, s.currNodeKey, err)
	}
}

// Next executes one cycle of the reactive computation.
// Execution proceeds in two phases:
//  1. Global strata: Execute nodes not in any stage
//  2. Stage strata: Execute active stages until convergence
//
// The changed set is cleared at the start of each stage strata execution to ensure
// independent change propagation between stages.
// The reason parameter indicates what triggered this scheduler run (timer tick or
// channel input). Time-based nodes use this to only fire on timer ticks.
func (s *Scheduler) Next(ctx context.Context, elapsed telem.TimeSpan, reason rnode.RunReason) {
	s.nextDeadline = telem.TimeSpanMax
	s.nodeCtx.Context = ctx
	s.nodeCtx.Elapsed = elapsed
	s.nodeCtx.Tolerance = s.tolerance
	s.nodeCtx.Reason = reason
	s.currSeqIdx = -1
	s.currStageIdx = -1
	s.execStrata(s.globalStrata)
	s.execStages()
	clear(s.changed)
}

// NextDeadline returns the minimum deadline (absolute elapsed time) reported by nodes
// during the last Next() call. Returns telem.TimeSpanMax if no node reported a deadline.
func (s *Scheduler) NextDeadline() telem.TimeSpan { return s.nextDeadline }

func (s *Scheduler) setDeadline(d telem.TimeSpan) {
	if d < s.nextDeadline {
		s.nextDeadline = d
	}
}

// execStrata executes nodes in a stage strata, propagating changes between layers.
// The changed set is cleared at the start to ensure independent propagation from
// other stages.
func (s *Scheduler) execStrata(strata ir.Strata) {
	clear(s.changed)
	s.transitioned = false
	inStage := s.currStageIdx != -1
	for i, stratum := range strata {
		for _, key := range stratum {
			wasSelfChanged := s.selfChanged.Contains(key)
			if wasSelfChanged {
				delete(s.selfChanged, key)
			}
			if i == 0 || s.changed.Contains(key) || wasSelfChanged {
				s.currNodeKey = key
				s.nodes[key].Next(s.nodeCtx)
				if inStage && s.transitioned {
					return
				}
			}
		}
	}
}

// execStages executes active stages across all sequences until convergence.
// A stage transition during execution triggers re-evaluation until stable.
func (s *Scheduler) execStages() {
	for iter := 0; iter < s.maxConvergenceIterations; iter++ {
		stable := true
		for s.currSeqIdx = 0; s.currSeqIdx < len(s.sequences); s.currSeqIdx++ {
			seq := &s.sequences[s.currSeqIdx]
			if seq.activeStageIdx == -1 {
				continue
			}
			s.currStageIdx = seq.activeStageIdx
			s.execStrata(seq.stages[s.currStageIdx].strata)
			if seq.activeStageIdx != s.currStageIdx {
				stable = false
			}
		}
		if stable {
			return
		}
	}
}

// transitionStage transitions to the stage associated with the currently executing node.
// This deactivates the current sequence's stage first, then activates the target stage.
// If entering from global strata and the sequence already has an active stage, this is
// a no-op to prevent re-entering a sequence that has already been started. Within-sequence
// transitions always proceed.
func (s *Scheduler) transitionStage() {
	target, ok := s.transitions[s.currNodeKey]
	if !ok {
		return
	}
	if s.currSeqIdx == -1 && s.sequences[target.seqIdx].activeStageIdx != -1 {
		return
	}
	if s.currSeqIdx != -1 {
		sourceStage := s.sequences[s.currSeqIdx].stages[s.currStageIdx]
		s.clearSelfChanged(sourceStage.strata)
		s.sequences[s.currSeqIdx].activeStageIdx = -1
	}
	s.resetStrata(s.sequences[target.seqIdx].stages[target.stageIdx].strata)
	s.sequences[target.seqIdx].activeStageIdx = target.stageIdx
	s.transitioned = true
}

// clearSelfChanged removes all nodes in a strata from the selfChanged set
// without resetting the nodes themselves.
func (s *Scheduler) clearSelfChanged(strata ir.Strata) {
	for _, stratum := range strata {
		for _, key := range stratum {
			delete(s.selfChanged, key)
		}
	}
}

// resetStrata resets all nodes in a strata to their initial state.
// Called when a stage is activated to reset timers and other stateful nodes.
func (s *Scheduler) resetStrata(strata ir.Strata) {
	for _, stratum := range strata {
		for _, key := range stratum {
			delete(s.selfChanged, key)
			s.nodes[key].Reset()
		}
	}
}
