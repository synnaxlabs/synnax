// Copyright 2026 Synnax Labs, Inc.
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
)

// Type aliases for IR types to avoid circular dependencies while maintaining
// clean API boundaries.
type (
	Function = ir.Function
	Edge     = ir.Edge
	Edges    = ir.Edges
	Handle   = ir.Handle
)

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
