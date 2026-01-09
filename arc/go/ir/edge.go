// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"fmt"

	"github.com/samber/lo"
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

// String returns the string representation of the handle as "node.param".
func (h Handle) String() string {
	return h.Node + "." + h.Param
}

// String returns the string representation of the edge kind.
func (k EdgeKind) String() string {
	if k == EdgeKindOneShot {
		return "OneShot"
	}
	return "Continuous"
}

// String returns the string representation of the edge.
// Format: "source.param -> target.param (continuous)" or "source.param => target.param (EdgeKindOneShot)"
func (e Edge) String() string {
	arrow := "->"
	if e.Kind == EdgeKindOneShot {
		arrow = "=>"
	}
	return fmt.Sprintf("%s %s %s (%s)", e.Source, arrow, e.Target, e.Kind)
}
