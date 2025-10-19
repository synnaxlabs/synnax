// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	timewheel "github.com/synnaxlabs/arc/runtime/time"
	"github.com/synnaxlabs/x/set"
)

type nodeState struct {
	node     node.Node
	outgoing []ir.Edge
}

type Scheduler struct {
	strata    ir.Strata
	changed   set.Set[string]
	nodes     map[string]*nodeState
	currState *nodeState
	timeWheel *timewheel.Wheel
}

func NewScheduler(
	ctx context.Context,
	prog ir.IR,
	nodes map[string]node.Node,
	timeWheel *timewheel.Wheel,
) *Scheduler {
	s := &Scheduler{
		nodes:     make(map[string]*nodeState, len(prog.Nodes)),
		strata:    prog.Strata,
		changed:   make(set.Set[string], len(prog.Nodes)),
		timeWheel: timeWheel,
	}
	for _, n := range prog.Nodes {
		s.nodes[n.Key] = &nodeState{
			outgoing: lo.Filter(prog.Edges, func(item ir.Edge, _ int) bool {
				return item.Source.Node == n.Key
			}),
			node: nodes[n.Key],
		}
	}

	// Start time wheel if provided
	if s.timeWheel != nil {
		s.timeWheel.Start(ctx)
	}

	return s
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

func (s *Scheduler) Init(ctx context.Context) {
	for _, nodeKey := range s.strata[0] {
		s.currState = s.nodes[nodeKey]
		s.currState.node.Init(ctx, s.markChanged)
	}
}

func (s *Scheduler) Next(ctx context.Context) {
	for _, stratum := range s.strata {
		for _, nodeKey := range stratum {
			if s.changed.Contains(nodeKey) {
				s.currState = s.nodes[nodeKey]
				s.currState.node.Next(ctx, s.markChanged)
			}
		}
	}
	clear(s.changed)
}
