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
//   - Init: Called once for stratum-0 (source) nodes during initialization
//   - Next: Called each cycle for stratum-0 nodes and any nodes marked as changed
//   - Change propagation: When a node's output changes, downstream nodes are marked
//     for execution in the next cycle
//
// The scheduler uses a "changed set" to track which nodes need execution, ensuring
// efficient incremental computation by only running nodes affected by changes.
package scheduler

import (
	"context"

	"github.com/samber/lo"
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
	// outgoing contains all edges where this node is the source.
	// Used for change propagation to downstream nodes.
	outgoing []ir.Edge
}

// Scheduler orchestrates the execution of nodes in topological order.
// It maintains the execution graph, tracks changed nodes, and propagates changes
// through the dependency graph. It also supports stage-based filtering for sequences.
type Scheduler struct {
	// strata defines the topological execution order.
	// Each stratum contains nodes at the same dependency level.
	strata ir.Strata
	// changed tracks which nodes need execution in the next cycle.
	changed set.Set[string]
	// nodes maps node keys to their runtime state.
	nodes map[string]*nodeState
	// currState points to the currently executing node.
	// Used for routing MarkChanged callbacks to the correct outgoing edges.
	currState *nodeState
	// errorHandler receives errors from node execution.
	errorHandler ErrorHandler
	// nodeCtx is a reusable context struct passed to nodes during execution.
	// This eliminates allocations by reusing the same struct across all executions.
	nodeCtx node.Context
	// startTime tracks when the scheduler was initialized for elapsed time calculation.
	startTime telem.TimeStamp

	// Stage management
	// sequences stores the IR sequence definitions for terminal detection.
	sequences ir.Sequences
	// activeStages maps sequence names to their currently active stage.
	// Multiple sequences can run concurrently.
	activeStages map[string]string
	// stageToNodes maps "sequence_stage" keys to lists of node keys in that stage.
	stageToNodes map[string][]string
	// stagedNodes contains all nodes that belong to any stage (for filtering).
	stagedNodes set.Set[string]
	// nodeToStage maps node keys to their (sequence, stage) pair for reverse lookup.
	nodeToStage map[string]stageRef
	// firedOneShots tracks which one-shot edges have fired, keyed by sequence.
	// Each sequence has its own set of fired edges, cleared when that sequence's stage changes.
	// Edge keys are strings like "nodeA.output=>nodeB.input".
	firedOneShots map[string]set.Set[string]
}

// stageRef identifies a stage within a sequence.
type stageRef struct {
	sequence string
	stage    string
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
//
// Parameters:
//   - ctx: Context for initialization (currently unused but available for future use)
//   - prog: IR program containing nodes, edges, and computed strata
//   - nodes: Map of node keys to executable node instances
//
// Returns a new Scheduler ready for Init and Next execution.
func New(
	ctx context.Context,
	prog ir.IR,
	nodes map[string]node.Node,
) *Scheduler {
	s := &Scheduler{
		nodes:         make(map[string]*nodeState, len(prog.Nodes)),
		strata:        prog.Strata,
		changed:       make(set.Set[string], len(prog.Nodes)),
		sequences:     prog.Sequences,
		activeStages:  make(map[string]string),
		stageToNodes:  make(map[string][]string),
		stagedNodes:   make(set.Set[string]),
		nodeToStage:   make(map[string]stageRef),
		firedOneShots: make(map[string]set.Set[string]),
	}
	s.nodeCtx = node.Context{
		MarkChanged:   s.markChanged,
		ReportError:   s.reportError,
		ActivateStage: s.activateStageByNode,
	}

	for _, n := range prog.Nodes {
		s.nodes[n.Key] = &nodeState{
			key: n.Key,
			outgoing: lo.Filter(prog.Edges, func(item ir.Edge, _ int) bool {
				return item.Source.Node == n.Key
			}),
			node: nodes[n.Key],
		}
	}

	// Load sequence/stage information for stage filtering
	s.loadSequences(prog.Sequences)

	return s
}

// stageKey creates a unique key for a stage within a sequence.
func stageKey(seqName, stageName string) string {
	return seqName + "_" + stageName
}

// loadSequences builds the stage-to-nodes and node-to-stage mappings from the IR sequences.
func (s *Scheduler) loadSequences(sequences ir.Sequences) {
	for _, seq := range sequences {
		for _, stage := range seq.Stages {
			key := stageKey(seq.Key, stage.Key)
			s.stageToNodes[key] = stage.Nodes

			// Track all nodes that belong to any stage and build reverse map
			for _, nodeKey := range stage.Nodes {
				s.stagedNodes.Add(nodeKey)
				s.nodeToStage[nodeKey] = stageRef{sequence: seq.Key, stage: stage.Key}
			}
		}
	}
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

func (s *Scheduler) markChanged(param string) {
	for _, edge := range s.currState.outgoing {
		if edge.Source.Param != param {
			continue
		}
		// For one-shot edges, only propagate if output is truthy and not already fired
		if edge.Kind == ir.OneShot {
			// Check truthiness before marking as fired - falsy values don't fire
			if !s.currState.node.IsOutputTruthy(edge.Source.Param) {
				continue
			}

			edgeKey := edge.Source.String() + "=>" + edge.Target.String()
			// Determine which sequence this edge belongs to (use source node's sequence)
			seqName := ""
			if ref, ok := s.nodeToStage[s.currState.key]; ok {
				seqName = ref.sequence
			}
			// Get or create the sequence's fired set
			firedSet, ok := s.firedOneShots[seqName]
			if !ok {
				firedSet = make(set.Set[string])
				s.firedOneShots[seqName] = firedSet
			}
			if firedSet.Contains(edgeKey) {
				continue // already fired this stage activation
			}
			firedSet.Add(edgeKey)
		}
		s.changed.Add(edge.Target.Node)
	}
}

// reportError reports an error from the currently executing node.
// This method uses s.currNodeKey to identify the node without requiring closure allocation.
func (s *Scheduler) reportError(err error) {
	if s.errorHandler != nil {
		s.errorHandler.HandleError(s.currState.key, err)
	}
}

// Init performs one-time initialization for stratum-0 (source) nodes.
// This is called once before any Next executions to allow source nodes to set up
// their initial state, establish connections, or perform other startup tasks.
//
// Only nodes in stratum-0 have their Init method called. Downstream nodes are
// initialized implicitly through their first Next execution when marked as changed.
func (s *Scheduler) Init(ctx context.Context) {
	s.startTime = telem.Now()
	s.nodeCtx.Context = ctx
	s.nodeCtx.Elapsed = 0
	for _, stratum := range s.strata {
		for _, nodeKey := range stratum {
			s.currState = s.nodes[nodeKey]
			s.currState.node.Init(s.nodeCtx)
		}
	}
}

// Next executes one cycle of the reactive computation.
// Nodes are executed in topological order (stratum by stratum). Within each cycle:
//   - Stratum-0 nodes always execute (they are source nodes)
//   - Other nodes only execute if marked as changed
//   - Nodes in stages only execute if their stage is active
//
// After execution, the changed set is cleared for the next cycle.
// Nodes can mark their outputs as changed during execution via MarkChanged callbacks,
// which will schedule downstream nodes for the next cycle.
func (s *Scheduler) Next(ctx context.Context) {
	s.nodeCtx.Context = ctx
	s.nodeCtx.Elapsed = telem.TimeSpan(telem.Now() - s.startTime)
	for i, stratum := range s.strata {
		for _, nodeKey := range stratum {
			// Apply stage filtering
			if !s.shouldExecuteNode(nodeKey) {
				continue
			}
			if i == 0 || s.changed.Contains(nodeKey) {
				s.currState = s.nodes[nodeKey]
				s.currState.node.Next(s.nodeCtx)
			}
		}
	}
	clear(s.changed)

	// Auto-deactivate sequences in terminal stages that have completed
	s.checkTerminalStages()
}

// checkTerminalStages deactivates sequences that have reached terminal stages
// and have no unfired one-shot edges remaining.
func (s *Scheduler) checkTerminalStages() {
	// Iterate over a copy of keys since we may modify activeStages
	for seqName, stageName := range s.activeStages {
		// Check if this stage is terminal (no next stage)
		seq, ok := s.sequences.Find(seqName)
		if !ok {
			continue
		}
		_, hasNext := seq.NextStage(stageName)
		if hasNext {
			continue // Not terminal, skip
		}

		// Stage is terminal - check if all one-shot edges have fired
		if s.stageHasUnfiredOneShots(seqName, stageName) {
			continue // Still has unfired one-shots, don't deactivate yet
		}

		// Deactivate the sequence
		s.DeactivateSequence(seqName)
	}
}

// stageHasUnfiredOneShots returns true if any node in the stage has unfired one-shot edges.
func (s *Scheduler) stageHasUnfiredOneShots(seqName, stageName string) bool {
	key := stageKey(seqName, stageName)
	nodes, ok := s.stageToNodes[key]
	if !ok {
		return false
	}

	firedSet := s.firedOneShots[seqName]

	for _, nodeKey := range nodes {
		state, ok := s.nodes[nodeKey]
		if !ok {
			continue
		}
		// Check if this node has any one-shot outgoing edges that haven't fired
		for _, edge := range state.outgoing {
			if edge.Kind != ir.OneShot {
				continue
			}
			edgeKey := edge.Source.String() + "=>" + edge.Target.String()
			if firedSet == nil || !firedSet.Contains(edgeKey) {
				return true // Found an unfired one-shot
			}
		}
	}
	return false
}

// shouldExecuteNode determines if a node should execute based on stage filtering.
// A node should execute if:
//  1. No sequences are defined (no stage filtering active), OR
//  2. The node is NOT part of any stage (always runs), OR
//  3. The node's sequence is active and its stage matches the active stage for that sequence
func (s *Scheduler) shouldExecuteNode(nodeKey string) bool {
	// If no stage filtering is active (no sequences defined), run all nodes
	if len(s.stageToNodes) == 0 {
		return true
	}

	// If the node is not part of any stage, always run it
	if !s.stagedNodes.Contains(nodeKey) {
		return true
	}

	// Check if node's sequence is active and in the correct stage
	ref, ok := s.nodeToStage[nodeKey]
	if !ok {
		return false
	}
	activeStage, seqActive := s.activeStages[ref.sequence]
	return seqActive && activeStage == ref.stage
}

// activateStageByNode looks up the stage that a node belongs to and activates it.
// This is the callback provided to nodes via the Context.
func (s *Scheduler) activateStageByNode(nodeKey string) {
	ref, ok := s.nodeToStage[nodeKey]
	if !ok {
		return
	}
	s.ActivateStage(ref.sequence, ref.stage)
}

// ActivateStage transitions to a new stage within a sequence.
// Multiple sequences can be active concurrently. This updates the stage for the
// given sequence and resets nodes in the new stage.
func (s *Scheduler) ActivateStage(seqName, stageName string) {
	// Update the active stage for this sequence
	s.activeStages[seqName] = stageName

	// Reset nodes in this stage and clear one-shot tracking for this sequence
	s.resetStageNodes(seqName, stageName)
}

// DeactivateSequence stops a sequence, removing it from the active sequences.
// This is called when a sequence reaches a terminal stage or is explicitly stopped.
func (s *Scheduler) DeactivateSequence(seqName string) {
	delete(s.activeStages, seqName)
	delete(s.firedOneShots, seqName)
}

// resetStageNodes resets all nodes in the given stage and clears one-shot tracking
// for the sequence. This is called when a stage is entered to reset timers,
// other stateful nodes, and allow one-shot edges to fire again.
func (s *Scheduler) resetStageNodes(seqName, stageName string) {
	// Clear one-shot tracking for this sequence so edges can fire again
	delete(s.firedOneShots, seqName)

	key := stageKey(seqName, stageName)
	nodes, ok := s.stageToNodes[key]
	if !ok {
		return
	}
	for _, nodeKey := range nodes {
		if state, ok := s.nodes[nodeKey]; ok {
			state.node.Reset()
		}
	}
}

// ActiveSequences returns all currently active sequence names.
func (s *Scheduler) ActiveSequences() []string {
	seqs := make([]string, 0, len(s.activeStages))
	for seq := range s.activeStages {
		seqs = append(seqs, seq)
	}
	return seqs
}

// ActiveStageFor returns the currently active stage for a given sequence.
// Returns empty string if the sequence is not active.
func (s *Scheduler) ActiveStageFor(seqName string) string {
	return s.activeStages[seqName]
}

// IsSequenceActive returns whether the given sequence is currently active.
func (s *Scheduler) IsSequenceActive(seqName string) bool {
	_, ok := s.activeStages[seqName]
	return ok
}
