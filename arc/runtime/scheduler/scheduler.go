// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/interval"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/x/set"
)

type nodeState struct {
	node     node.Node
	outgoing []ir.Edge
}

type Scheduler struct {
	strata       ir.Strata
	changed      set.Set[string]
	nodes        map[string]*nodeState
	currState    *nodeState
	timeWheel    *interval.Wheel
	errorHandler ErrorHandler
}

// ErrorHandler receives errors from node execution.
type ErrorHandler interface {
	HandleError(nodeKey string, err error)
}

func New(
	ctx context.Context,
	prog ir.IR,
	nodes map[string]node.Node,
	timeWheel *interval.Wheel,
) *Scheduler {
	s := &Scheduler{
		nodes:     make(map[string]*nodeState, len(prog.Nodes)),
		strata:    prog.Strata,
		changed:   make(set.Set[string], len(prog.Nodes)),
		timeWheel: timeWheel,
	}

	// Set callback on time wheel
	if s.timeWheel != nil {
		s.timeWheel.SetCallback(s.MarkNodesChange)
		s.timeWheel.Start(ctx)
	}

	for _, n := range prog.Nodes {
		s.nodes[n.Key] = &nodeState{
			outgoing: lo.Filter(prog.Edges, func(item ir.Edge, _ int) bool {
				return item.Source.Node == n.Key
			}),
			node: nodes[n.Key],
		}
	}

	return s
}

func (s *Scheduler) SetErrorHandler(handler ErrorHandler) {
	s.errorHandler = handler
}

func (s *Scheduler) MarkNodesChange(nodeKey string) {
	s.changed.Add(nodeKey)
}

func (s *Scheduler) markChanged(param string) {
	for _, edge := range s.currState.outgoing {
		if edge.Source.Param == param {
			s.changed.Add(edge.Target.Node)
		}
	}
}

func (s *Scheduler) makeReportError(nodeKey string) func(error) {
	return func(err error) {
		if s.errorHandler != nil {
			s.errorHandler.HandleError(nodeKey, err)
		}
	}
}

func (s *Scheduler) makeContext(ctx context.Context, nodeKey string) node.Context {
	return node.Context{
		Context:     ctx,
		MarkChanged: s.markChanged,
		ReportError: s.makeReportError(nodeKey),
	}
}

func (s *Scheduler) Init(ctx context.Context) {
	for _, nodeKey := range s.strata[0] {
		s.currState = s.nodes[nodeKey]
		sctx := s.makeContext(ctx, nodeKey)
		s.currState.node.Init(sctx)
	}
}

func (s *Scheduler) Next(ctx context.Context) {
	for i, stratum := range s.strata {
		for _, nodeKey := range stratum {
			if i == 0 || s.changed.Contains(nodeKey) {
				s.currState = s.nodes[nodeKey]
				sctx := s.makeContext(ctx, nodeKey)
				s.currState.node.Next(sctx)
			}
		}
	}
	clear(s.changed)
}

func (s *Scheduler) TimeWheel() *interval.Wheel {
	return s.timeWheel
}
