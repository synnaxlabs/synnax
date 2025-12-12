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

// EdgeKind distinguishes between continuous reactive flows and one-shot transitions.
type EdgeKind int

const (
	// Continuous edges fire every tick while the source is active (-> operator).
	// Data flows continuously from source to target.
	Continuous EdgeKind = iota
	// OneShot edges fire once when the condition becomes true (=> operator).
	// Used for state transitions and one-time actions.
	OneShot
)

// Parameter naming conventions for IR nodes and functions.
const (
	// DefaultOutputParam is the parameter name for single-output functions and stages.
	// Use this for unary operations like neg, sqrt, etc.
	DefaultOutputParam = "output"
	// DefaultInputParam is the parameter name for single-input functions and stages.
	// Use this for unary operations that take one input.
	DefaultInputParam = "input"
	// LHSInputParam is the left-hand side parameter name for binary operators.
	// Use this as the first operand name in operations like add, multiply, etc.
	LHSInputParam = "a"
	// RHSInputParam is the right-hand side parameter name for binary operators.
	// Use this as the second operand name in operations like add, multiply, etc.
	RHSInputParam = "b"
)

// Handle uniquely identifies a parameter port on a node in the dataflow graph.
// The combination of Node key and Param name uniquely identifies an input or
// output port for dataflow edge connections.
type Handle struct {
	// Node is the key of the node.
	Node string `json:"node"`
	// Param is the name of the parameter on the node.
	Param string `json:"param"`
}

// Edge represents a dataflow connection between two node parameters.
// Data flows from the Source handle's output parameter to the Target handle's
// input parameter. The Kind determines whether the edge is continuous (reactive)
// or one-shot (transition).
type Edge struct {
	// Source is the output parameter that provides data.
	Source Handle `json:"source"`
	// Target is the input parameter that receives data.
	Target Handle `json:"target"`
	// Kind specifies whether this is a continuous reactive flow or a one-shot transition.
	Kind EdgeKind `json:"kind"`
}

// Edges is a collection of dataflow edges.
type Edges []Edge

// Predicate constructors for edge queries.
func sourceEquals(handle Handle) func(Edge) bool {
	return func(e Edge) bool { return e.Source == handle }
}

func targetEquals(handle Handle) func(Edge) bool {
	return func(e Edge) bool { return e.Target == handle }
}

func targetNodeEquals(nodeKey string) func(Edge, int) bool {
	return func(e Edge, _ int) bool { return e.Target.Node == nodeKey }
}

func sourceNodeEquals(nodeKey string) func(Edge, int) bool {
	return func(e Edge, _ int) bool { return e.Source.Node == nodeKey }
}

// GetBySource returns the edge with the given source handle. Panics if not found.
func (e Edges) GetBySource(handle Handle) Edge {
	return e.get(sourceEquals(handle))
}

// GetByTarget returns the edge with the given target handle. Panics if not found.
func (e Edges) GetByTarget(handle Handle) Edge {
	return e.get(targetEquals(handle))
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
	return e.find(sourceEquals(handle))
}

// FindByTarget searches for an edge with the given target handle.
func (e Edges) FindByTarget(handle Handle) (Edge, bool) {
	return e.find(targetEquals(handle))
}

// GetInputs returns all edges targeting the given node.
func (e Edges) GetInputs(nodeKey string) []Edge {
	return e.filter(targetNodeEquals(nodeKey))
}

// GetOutputs returns all edges sourced from the given node.
func (e Edges) GetOutputs(nodeKey string) []Edge {
	return e.filter(sourceNodeEquals(nodeKey))
}

// GetByKind returns all edges with the specified kind.
func (e Edges) GetByKind(kind EdgeKind) Edges {
	return lo.Filter(e, func(edge Edge, _ int) bool { return edge.Kind == kind })
}
