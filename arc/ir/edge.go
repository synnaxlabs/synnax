// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"github.com/samber/lo"
)

// Handle uniquely identifies a parameter on a node in the dataflow graph.
type Handle struct {
	Node  string `json:"node"`
	Param string `json:"param"`
}

// Edge represents a dataflow connection between two node parameters.
type Edge struct {
	Source Handle `json:"source"`
	Target Handle `json:"target"`
}

// Edges is a collection of dataflow edges.
type Edges []Edge

// GetBySource returns the edge with the given source handle. Panics if not found.
func (e Edges) GetBySource(handle Handle) Edge {
	return e.get(func(e Edge) bool { return e.Source == handle })
}

// GetByTarget returns the edge with the given target handle. Panics if not found.
func (e Edges) GetByTarget(handle Handle) Edge {
	return e.get(func(e Edge) bool { return e.Target == handle })
}

func (e Edges) find(f func(e Edge) bool) (Edge, bool) {
	return lo.Find(e, f)
}

func (e Edges) get(f func(e Edge) bool) Edge {
	return lo.Must(e.find(f))
}

func (e Edges) filter(f func(e Edge, _ int) bool) []Edge {
	return lo.Filter(e, f)
}

// FindBySource searches for an edge with the given source handle.
func (e Edges) FindBySource(handle Handle) (Edge, bool) {
	return e.find(func(e Edge) bool { return e.Source == handle })
}

// FindByTarget searches for an edge with the given target handle.
func (e Edges) FindByTarget(handle Handle) (Edge, bool) {
	return e.find(func(e Edge) bool { return e.Target == handle })
}

// GetInputs returns all edges targeting the given node.
func (e Edges) GetInputs(nodeKey string) []Edge {
	return e.filter(func(e Edge, _ int) bool { return e.Target.Node == nodeKey })
}

// GetOutputs returns all edges sourced from the given node.
func (e Edges) GetOutputs(nodeKey string) []Edge {
	return e.filter(func(e Edge, _ int) bool { return e.Source.Node == nodeKey })
}
