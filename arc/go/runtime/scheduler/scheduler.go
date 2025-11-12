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
// through the dependency graph.
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
		nodes:   make(map[string]*nodeState, len(prog.Nodes)),
		strata:  prog.Strata,
		changed: make(set.Set[string], len(prog.Nodes)),
	}
	s.nodeCtx = node.Context{
		MarkChanged: s.markChanged,
		ReportError: s.reportError,
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

func (s *Scheduler) markChanged(param string) {
	for _, edge := range s.currState.outgoing {
		if edge.Source.Param == param {
			s.changed.Add(edge.Target.Node)
		}
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
	s.nodeCtx.Context = ctx
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
//
// After execution, the changed set is cleared for the next cycle.
// Nodes can mark their outputs as changed during execution via MarkChanged callbacks,
// which will schedule downstream nodes for the next cycle.
func (s *Scheduler) Next(ctx context.Context) {
	s.nodeCtx.Context = ctx
	for i, stratum := range s.strata {
		for _, nodeKey := range stratum {
			if i == 0 || s.changed.Contains(nodeKey) {
				s.currState = s.nodes[nodeKey]
				s.currState.node.Next(s.nodeCtx)
			}
		}
	}
	clear(s.changed)
}
