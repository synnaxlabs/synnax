// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package ir provides the intermediate representation (IR) for Arc programs.
//
// The IR represents a compiled Arc program as a dataflow graph consisting of:
//   - Functions: Template definitions for reusable computations (stages and user functions)
//   - Nodes: Instantiated functions with concrete configuration values
//   - Edges: Dataflow connections between node parameters (Handle-to-Handle mappings)
//   - Strata: Execution stratification for deterministic, glitch-free reactive execution
//
// The IR serves as the bridge between the semantic analyzer (which produces a symbol table
// and type information) and the WebAssembly compiler (which generates executable code).
// It captures all necessary information for code generation, optimization, and runtime
// execution of Arc programs.
//
// # Compilation Pipeline
//
// The IR fits into the Arc compilation pipeline as follows:
//
//	Parser → AST → Analyzer → Symbol Table + Types → IR Builder → IR → Compiler → WASM
//
// # Core Concepts
//
// Functions are templates that define reusable computations with typed inputs, outputs,
// and configuration parameters. They are analogous to function signatures in traditional
// languages but can represent both pure functions and stateful reactive stages.
//
// Nodes are concrete instantiations of Functions in the dataflow graph. Each node has
// a unique key, references its function type, stores configuration values, and declares
// its input/output parameter types. Nodes are the executable units of Arc programs.
//
// Edges represent dataflow connections. Each edge connects a source Handle (node + parameter)
// to a target Handle (node + parameter), forming the dependency graph that determines
// execution order and data routing.
//
// Root is a Scope (the unified Layer 2 execution primitive) whose strata
// organize module-scope reactive flow and whose nested Scope members capture
// top-level stages and sequences.
package ir

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/tree"
)

// InlinePrefix names the synthetic scopes lowered from inline stage/sequence flow
// targets, so the analyzer can detect and resolve them by key.
const InlinePrefix = "__inline_"

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

func (i *IR) IsZero() bool {
	return len(i.Functions) == 0 &&
		len(i.Nodes) == 0 &&
		len(i.Edges) == 0 &&
		i.Root.IsZero() &&
		i.Symbols == nil &&
		i.TypeMap == nil
}

// String returns the string representation of the IR.
func (i *IR) String() string { return i.stringWithPrefix("") }

// stringWithPrefix returns the string representation with tree formatting.
func (i *IR) stringWithPrefix(prefix string) string {
	var b strings.Builder

	hasFunctions := len(i.Functions) > 0
	hasNodes := len(i.Nodes) > 0
	hasEdges := len(i.Edges) > 0
	hasRoot := !i.Root.IsZero()

	if hasFunctions {
		isLast := !hasNodes && !hasEdges && !hasRoot
		i.writeFunctions(&b, prefix, isLast)
	}

	if hasNodes {
		isLast := !hasEdges && !hasRoot
		i.writeNodes(&b, prefix, isLast)
	}

	if hasEdges {
		isLast := !hasRoot
		i.writeEdges(&b, prefix, isLast)
	}

	if hasRoot {
		i.writeRoot(&b, prefix, true)
	}

	return b.String()
}

func (i *IR) writeFunctions(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(tree.Prefix(last))
	lo.Must(fmt.Fprintf(b, "Functions (%d)\n", len(i.Functions)))
	childPrefix := prefix + tree.Indent(last)
	for j, f := range i.Functions {
		isLast := j == len(i.Functions)-1
		b.WriteString(childPrefix)
		b.WriteString(tree.Prefix(isLast))
		b.WriteString(f.StringWithPrefix(childPrefix + tree.Indent(isLast)))
	}
}

func (i *IR) writeNodes(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(tree.Prefix(last))
	lo.Must(fmt.Fprintf(b, "Nodes (%d)\n", len(i.Nodes)))
	childPrefix := prefix + tree.Indent(last)
	for j, n := range i.Nodes {
		isLast := j == len(i.Nodes)-1
		b.WriteString(childPrefix)
		b.WriteString(tree.Prefix(isLast))
		b.WriteString(n.stringWithPrefix(childPrefix + tree.Indent(isLast)))
	}
}

func (i *IR) writeEdges(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(tree.Prefix(last))
	lo.Must(fmt.Fprintf(b, "Edges (%d)\n", len(i.Edges)))
	childPrefix := prefix + tree.Indent(last)
	for j, e := range i.Edges {
		isLast := j == len(i.Edges)-1
		b.WriteString(childPrefix)
		b.WriteString(tree.Prefix(isLast))
		b.WriteString(e.String())
		b.WriteString("\n")
	}
}

func (i *IR) writeRoot(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(tree.Prefix(last))
	b.WriteString("Root\n")
	childPrefix := prefix + tree.Indent(last)
	b.WriteString(i.Root.StringWithPrefix(childPrefix))
}
