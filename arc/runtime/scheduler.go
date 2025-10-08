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
}

func NewScheduler(
	prog ir.IR,
	nodes map[string]node.Node,
) *Scheduler {
	s := &Scheduler{
		nodes:   make(map[string]*nodeState, len(prog.Nodes)),
		strata:  prog.Strata,
		changed: make(set.Set[string], len(prog.Nodes)),
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

func (s *Scheduler) MarkChanged(key string) {
	s.changed.Add(key)
}

func (s *Scheduler) markChanged(param string) {
	for _, edge := range s.currState.outgoing {
		if edge.Source.Param == param {
			s.changed.Add(edge.Target.Node)
		}
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
