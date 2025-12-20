// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package scheduler orchestrates the execution of arc runtime nodes.
//
// The scheduler is responsible for executing nodes in topological order based on
// their dependency graph. It implements a reactive execution model where nodes are
// executed when they or their inputs change. The execution order is determined by
// strata (layers), where each stratum contains nodes at the same topological level.
//
// Execution flow:
//   - Init: Called once to initialize the scheduler with a start time
//   - Next: Called each cycle for stratum-0 nodes and any nodes marked as changed
//   - Change propagation: When a node's output changes, downstream nodes are marked
//     for execution in the current strata pass
//
// The scheduler uses a "changed set" to track which nodes need execution, clearing
// the set at the start of each strata execution to ensure independent propagation.
package scheduler

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// nodeState holds the runtime state for a single node in the scheduler.
type nodeState struct {
	key string
	// node is the executable node instance.
	node node.Node
	// outgoing contains edges where this node is the source, keyed by output param.
	// Used for efficient change propagation to downstream nodes.
	outgoing map[string][]ir.Edge
}

// stageState holds the runtime state for a single stage within a sequence.
type stageState struct {
	// strata defines the topological execution order for nodes in this stage.
	strata ir.Strata
	// firedOneShots tracks which one-shot edges have already fired in this stage
	// activation. Cleared when the stage is entered.
	firedOneShots map[ir.Edge]struct{}
}

// sequenceState holds the runtime state for a sequence.
type sequenceState struct {
	// stages contains the ordered list of stages in this sequence.
	stages []stageState
	// activeStageIdx is the index of the currently active stage, or -1 if none.
	activeStageIdx int
}

// Scheduler orchestrates the execution of nodes in topological order.
// It maintains the execution graph, tracks changed nodes, and propagates changes
// through the dependency graph. It supports stage-based filtering for sequences.
type Scheduler struct {
	// nodes maps node keys to their runtime state.
	nodes map[string]*nodeState
	// globalStrata defines the topological execution order for global nodes.
	globalStrata ir.Strata
	// sequences holds the runtime state for each sequence.
	sequences []sequenceState
	// transitions maps entry node keys to their target (seqIdx, stageIdx).
	transitions map[string]struct{ seqIdx, stageIdx int }
	// changed tracks which nodes need execution in the current strata pass.
	changed set.Set[string]
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
	// nodeCtx is a reusable context struct passed to nodes during execution.
	nodeCtx node.Context
	// startTime tracks when the scheduler was initialized for elapsed time calculation.
	startTime telem.TimeStamp
}

// ErrorHandler receives errors from node execution.
// Implementations can log, aggregate, or handle errors in custom ways.
type ErrorHandler interface {
	// HandleError is called when a node reports an error during execution.
	HandleError(nodeKey string, err error)
}

// New creates a scheduler from an IR program and node instances.
// The scheduler organizes nodes into strata for topological execution and
// builds the change propagation graph from the IR edges.
func New(
	prog ir.IR,
	nodes map[string]node.Node,
) *Scheduler {
	s := &Scheduler{
		nodes:        make(map[string]*nodeState, len(prog.Nodes)),
		globalStrata: prog.Strata,
		sequences:    make([]sequenceState, len(prog.Sequences)),
		transitions:  make(map[string]struct{ seqIdx, stageIdx int }),
		changed:      make(set.Set[string], len(prog.Nodes)),
		currSeqIdx:   -1,
		currStageIdx: -1,
	}

	// Build node states with edges grouped by output param
	for _, n := range prog.Nodes {
		outgoing := make(map[string][]ir.Edge)
		for _, edge := range prog.Edges {
			if edge.Source.Node == n.Key {
				outgoing[edge.Source.Param] = append(outgoing[edge.Source.Param], edge)
			}
		}
		s.nodes[n.Key] = &nodeState{
			key:      n.Key,
			outgoing: outgoing,
			node:     nodes[n.Key],
		}
	}

	// Build sequences and transitions
	for seqIdx, seq := range prog.Sequences {
		seqState := sequenceState{
			stages:         make([]stageState, len(seq.Stages)),
			activeStageIdx: -1,
		}
		s.maxConvergenceIterations += len(seq.Stages)

		for stageIdx, stage := range seq.Stages {
			seqState.stages[stageIdx] = stageState{
				strata:        stage.Strata,
				firedOneShots: make(map[ir.Edge]struct{}),
			}
			entryKey := "entry_" + seq.Key + "_" + stage.Key
			s.transitions[entryKey] = struct{ seqIdx, stageIdx int }{seqIdx, stageIdx}
		}
		s.sequences[seqIdx] = seqState
	}

	s.nodeCtx = node.Context{
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

// MarkNodeChanged marks a node as changed, scheduling it for execution in the next cycle.
// This is used externally to trigger execution based on external events or inputs.
func (s *Scheduler) MarkNodeChanged(nodeKey string) {
	s.changed.Add(nodeKey)
}

// markChanged propagates changes from the current node's output to downstream nodes.
// For continuous edges (and unspecified kind), always propagates.
// For one-shot edges, propagates only if:
// - The output is truthy, AND
// - Either not in a stage (always fire) OR first time firing in this stage activation.
func (s *Scheduler) markChanged(param string) {
	state := s.nodes[s.currNodeKey]
	edges := state.outgoing[param]

	for _, edge := range edges {
		// OneShot edges have special tracking behavior
		if edge.Kind == ir.OneShot {
			// Check truthiness first
			if !state.node.IsOutputTruthy(param) {
				continue
			}

			// If not in a stage, always fire (no tracking)
			if s.currStageIdx == -1 {
				s.changed.Add(edge.Target.Node)
				continue
			}

			// In a stage - track per-stage
			currStage := &s.sequences[s.currSeqIdx].stages[s.currStageIdx]
			if _, fired := currStage.firedOneShots[edge]; !fired {
				currStage.firedOneShots[edge] = struct{}{}
				s.changed.Add(edge.Target.Node)
			}
			continue
		}

		// Continuous edge (or unspecified kind) - always propagate
		s.changed.Add(edge.Target.Node)
	}
}

// reportError reports an error from the currently executing node.
func (s *Scheduler) reportError(err error) {
	if s.errorHandler != nil {
		s.errorHandler.HandleError(s.currNodeKey, err)
	}
}

// Init performs one-time initialization for the scheduler.
// This is called once before any Next executions to set up the start time.
func (s *Scheduler) Init(ctx context.Context) {
	s.startTime = telem.Now()
	s.nodeCtx.Context = ctx
	s.nodeCtx.Elapsed = 0
}

// Next executes one cycle of the reactive computation.
// Execution proceeds in two phases:
//  1. Global strata: Execute nodes not in any stage
//  2. Stage strata: Execute active stages until convergence
//
// The changed set is cleared at the start of each stage strata execution to ensure
// independent change propagation between stages.
func (s *Scheduler) Next(ctx context.Context) {
	s.nodeCtx.Context = ctx
	s.nodeCtx.Elapsed = telem.TimeSpan(telem.Now() - s.startTime)

	// Execute global strata (preserves external marks from MarkNodeChanged)
	s.executeGlobalStrata()

	// Execute active stages until convergence
	s.execStages()

	// Clear changed set at end (for next cycle)
	clear(s.changed)
}

// executeGlobalStrata executes nodes in the global strata.
// Stratum 0 always executes, higher strata execute if their nodes are in the changed set.
func (s *Scheduler) executeGlobalStrata() {
	firstStratum := true
	for _, stratum := range s.globalStrata {
		for _, key := range stratum {
			if firstStratum || s.changed.Contains(key) {
				s.currNodeKey = key
				if state, ok := s.nodes[key]; ok {
					state.node.Next(s.nodeCtx)
				}
			}
		}
		firstStratum = false
	}
}

// executeStrata executes nodes in a stage strata, propagating changes between layers.
// The changed set is cleared at the start to ensure independent propagation from
// other stages.
func (s *Scheduler) executeStrata(strata ir.Strata) {
	clear(s.changed)
	firstStratum := true
	for _, stratum := range strata {
		for _, key := range stratum {
			if firstStratum || s.changed.Contains(key) {
				s.currNodeKey = key
				if state, ok := s.nodes[key]; ok {
					state.node.Next(s.nodeCtx)
				}
			}
		}
		firstStratum = false
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
			s.executeStrata(seq.stages[s.currStageIdx].strata)
			// Detect if a transition occurred during execution
			if seq.activeStageIdx != s.currStageIdx {
				stable = false
			}
		}
		if stable {
			break
		}
	}
	// Reset sequence/stage indices after loop
	s.currSeqIdx = -1
	s.currStageIdx = -1
}

// transitionStage transitions to the stage associated with the currently executing node.
// This deactivates the current sequence's stage first, then activates the target stage.
func (s *Scheduler) transitionStage() {
	// Deactivate current sequence's stage first
	if s.currSeqIdx != -1 {
		s.sequences[s.currSeqIdx].activeStageIdx = -1
	}

	// Look up target
	target, ok := s.transitions[s.currNodeKey]
	if !ok {
		return
	}

	// Clear one-shots for target stage
	targetStage := &s.sequences[target.seqIdx].stages[target.stageIdx]
	clear(targetStage.firedOneShots)

	// Reset nodes in target strata
	s.resetStrata(targetStage.strata)

	// Activate target stage
	s.sequences[target.seqIdx].activeStageIdx = target.stageIdx
}

// resetStrata resets all nodes in a strata to their initial state.
// Called when a stage is activated to reset timers and other stateful nodes.
func (s *Scheduler) resetStrata(strata ir.Strata) {
	for _, stratum := range strata {
		for _, key := range stratum {
			if state, ok := s.nodes[key]; ok {
				state.node.Reset()
			}
		}
	}
}
