// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package graph provides visual graph compilation for Arc programs.
//
// This package transforms a visual graph representation (nodes, edges, viewport)
// into an intermediate representation (IR) suitable for stratification and runtime
// execution. It serves as the 5th stage in the Arc compiler pipeline:
//
//	Parser → Analyzer → Stratifier → Graph → Compiler → Runtime
//
// The package handles:
//   - Parsing function bodies from raw text into AST
//   - Type inference and constraint solving for polymorphic functions
//   - Edge validation between node inputs/outputs
//   - Configuration value type checking
//   - Stratification of execution order
//
// The core compilation process is performed by Analyze(), which implements a
// 10-step pipeline that produces executable IR from a visual graph.
package graph

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/spatial"
)

// Type aliases for IR types to avoid circular dependencies while maintaining
// clean API boundaries.
type (
	Function = ir.Function
	Edge     = ir.Edge
	Edges    = ir.Edges
	Handle   = ir.Handle
)

// Node represents a visual node in an Arc graph. Unlike ir.Node, which contains
// compiled type information, Node represents the user's visual layout including
// position and raw configuration values.
type Node struct {
	// Key is the unique identifier for this node instance.
	Key string `json:"key"`
	// Type is the function type this node instantiates.
	Type string `json:"type"`
	// Config are the raw configuration parameter values.
	Config map[string]any `json:"config"`
	// Position is the visual position in the graph editor.
	Position spatial.XY `json:"position"`
}

// Nodes is a slice of Node with helper methods for lookup operations.
type Nodes []Node

// Get returns the node with the given key. Panics if the node is not found.
// Use Find for safe lookups with error handling.
func (n Nodes) Get(key string) Node {
	return lo.Must(lo.Find(n, func(n Node) bool { return n.Key == key }))
}

// Find returns the node with the given key and a boolean indicating whether
// the node was found. This is the safe variant of Get.
func (n Nodes) Find(key string) (Node, bool) {
	return lo.Find(n, func(n Node) bool { return n.Key == key })
}

// Viewport represents the visual viewport state of the graph editor.
type Viewport struct {
	// Position is the pan offset of the viewport.
	Position spatial.XY `json:"position"`
	// Zoom is the zoom level of the viewport.
	Zoom float32 `json:"zoom"`
}

// Graph represents a complete visual graph.
type Graph struct {
	// Viewport is the visual viewport state.
	Viewport Viewport `json:"viewport"`
	// Functions are the function definitions available in the graph.
	Functions []Function `json:"functions"`
	// Edges connect node outputs to node inputs.
	Edges Edges `json:"edges"`
	// Nodes are the visual node instances in the graph.
	Nodes Nodes `json:"nodes"`
}
