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
	key string
	// node is the executable node instance.
	rnode.Node
	// outgoing contains edges where this node is the source, keyed by output param.
	// Used for efficient change propagation to downstream nodes.
	outgoing map[string][]ir.Edge
}

// stage holds the runtime state for a single stage within a sequence.
type stage struct {
	// strata defines the topological execution order for nodes in this stage.
	strata ir.Strata
	// firedOneShots tracks which one-shot edges have already fired in this stage
	// activation. Cleared when the stage is entered.
	firedOneShots set.Set[ir.Edge]
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
	// nodes maps node keys to their runtime state.
	nodes map[string]node
	// globalStrata defines the topological execution order for global nodes.
	globalStrata ir.Strata
	// sequences holds the runtime state for each sequence.
	sequences []sequenceState
	// transitions maps entry node keys to their target (seqIdx, stageIdx).
	transitions map[string]transitionTarget
	// changed tracks which nodes need execution in the current strata pass.
	changed set.Set[string]
	// globalFiredOneShots tracks which one-shot edges in global strata have fired.
	// Unlike per-stage one-shots, global one-shots fire once ever and never reset.
	globalFiredOneShots set.Set[ir.Edge]
	// maxConvergenceIterations is the maximum iterations for stage convergence loop.
	maxConvergenceIterations int

	// Current execution context
	// currNodeKey is the key of the currently executing node.
	currNodeKey string
	// currSeqIdx is the index of the currently executing sequence, or -1 if global.
	currSeqIdx int
	// currStageIdx is the index of the currently executing stage, or -1 if none.
	currStageIdx int

	// errorHandler receives errors from node execution.
	errorHandler ErrorHandler
	// cycleCallback is called at the end of each Next() cycle for cleanup.
	cycleCallback CycleCallback
	// nodeCtx is a reusable context struct passed to nodes during execution.
	nodeCtx rnode.Context
}

// ErrorHandler receives errors from node execution.
// Implementations can log, aggregate, or handle errors in custom ways.
type ErrorHandler interface {
	// HandleError is called when a node reports an error during execution.
	HandleError(nodeKey string, err error)
}

// CycleCallback is called at the end of each scheduler cycle.
// Used for cleanup operations like clearing temporary series handles.
type CycleCallback interface {
	// OnCycleEnd is called after all nodes have executed in a cycle.
	OnCycleEnd()
}

// New creates a scheduler from an IR program and node instances.
// The scheduler organizes nodes into strata for topological execution and
// builds the change propagation graph from the IR edges.
func New(prog ir.IR, nodes map[string]rnode.Node) *Scheduler {
	s := &Scheduler{
		nodes:               make(map[string]node, len(prog.Nodes)),
		globalStrata:        prog.Strata,
		sequences:           make([]sequenceState, len(prog.Sequences)),
		transitions:         make(map[string]transitionTarget),
		changed:             make(set.Set[string], len(prog.Nodes)),
		globalFiredOneShots: make(set.Set[ir.Edge]),
		currSeqIdx:          -1,
		currStageIdx:        -1,
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
				strata:        irStage.Strata,
				firedOneShots: make(set.Set[ir.Edge]),
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
		MarkChanged:   s.markChanged,
		ReportError:   s.reportError,
		ActivateStage: s.transitionStage,
	}

	return s
}

// SetErrorHandler configures the handler for node execution errors.
// If no handler is set, errors are silently ignored.
func (s *Scheduler) SetErrorHandler(handler ErrorHandler) {
	s.errorHandler = handler
}

// SetCycleCallback sets a callback to be invoked at the end of each scheduler cycle.
// Used for cleanup operations like clearing temporary series handles.
func (s *Scheduler) SetCycleCallback(cb CycleCallback) {
	s.cycleCallback = cb
}

// MarkNodeChanged marks a node as changed, scheduling it for execution in the next cycle.
// This is used externally to trigger execution based on external events or inputs.
func (s *Scheduler) MarkNodeChanged(nodeKey string) {
	s.changed.Add(nodeKey)
}

// markChanged propagates changes from the current node's output to downstream nodes.
// For continuous edges (and unspecified kind), always propagates.
// For one-shot edges, propagates only if:
// - The output is truthy, AND
// - First time firing (global one-shots fire once ever, stage one-shots fire once per activation)
func (s *Scheduler) markChanged(param string) {
	n := s.nodes[s.currNodeKey]
	for _, edge := range n.outgoing[param] {
		if edge.Kind == ir.OneShot {
			if !n.IsOutputTruthy(param) {
				continue
			}
			if s.currStageIdx == -1 {
				// Global one-shot: fire once ever (tracked in globalFiredOneShots)
				if _, fired := s.globalFiredOneShots[edge]; !fired {
					s.globalFiredOneShots.Add(edge)
					s.changed.Add(edge.Target.Node)
				}
				continue
			}
			// Stage one-shot: fire once per activation (tracked in stage's firedOneShots)
			currStage := s.sequences[s.currSeqIdx].stages[s.currStageIdx]
			if _, fired := currStage.firedOneShots[edge]; !fired {
				currStage.firedOneShots.Add(edge)
				s.changed.Add(edge.Target.Node)
			}
			continue
		}
		s.changed.Add(edge.Target.Node)
	}
}

// reportError reports an error from the currently executing node.
func (s *Scheduler) reportError(err error) {
	if s.errorHandler != nil {
		s.errorHandler.HandleError(s.currNodeKey, err)
	}
}

// Next executes one cycle of the reactive computation.
// Execution proceeds in two phases:
//  1. Global strata: Execute nodes not in any stage
//  2. Stage strata: Execute active stages until convergence
//
// The changed set is cleared at the start of each stage strata execution to ensure
// independent change propagation between stages.
func (s *Scheduler) Next(ctx context.Context, elapsed telem.TimeSpan) {
	s.nodeCtx.Context = ctx
	s.nodeCtx.Elapsed = elapsed
	s.currSeqIdx = -1
	s.currStageIdx = -1
	s.execStrata(s.globalStrata)
	s.execStages()
	clear(s.changed)
	// Call cleanup callback if set (e.g., to clear temporary series handles)
	if s.cycleCallback != nil {
		s.cycleCallback.OnCycleEnd()
	}
}

// execStrata executes nodes in a stage strata, propagating changes between layers.
// The changed set is cleared at the start to ensure independent propagation from
// other stages.
func (s *Scheduler) execStrata(strata ir.Strata) {
	clear(s.changed)
	for i, stratum := range strata {
		for _, key := range stratum {
			if i == 0 || s.changed.Contains(key) {
				s.currNodeKey = key
				s.nodes[key].Next(s.nodeCtx)
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
func (s *Scheduler) transitionStage() {
	if s.currSeqIdx != -1 {
		s.sequences[s.currSeqIdx].activeStageIdx = -1
	}
	target, ok := s.transitions[s.currNodeKey]
	if !ok {
		return
	}
	targetStage := &s.sequences[target.seqIdx].stages[target.stageIdx]
	clear(targetStage.firedOneShots)
	s.resetStrata(targetStage.strata)
	s.sequences[target.seqIdx].activeStageIdx = target.stageIdx
}

// resetStrata resets all nodes in a strata to their initial state.
// Called when a stage is activated to reset timers and other stateful nodes.
func (s *Scheduler) resetStrata(strata ir.Strata) {
	for _, stratum := range strata {
		for _, key := range stratum {
			s.nodes[key].Reset()
		}
	}
}
