// Copyright 2025 Synnax Labs, Inc.
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
// Strata partition nodes into execution layers where nodes in stratum N can execute
// in parallel, and nodes in stratum N depend only on nodes in strata 0 to N-1. This
// stratification enables single-pass reactive execution without glitches (temporary
// inconsistencies in computed values).
//
// # Example
//
// A simple Arc program computing "c = a + b" would have the following IR structure:
//
//	ir := &IR{
//	    Functions: Functions{
//	        {Key: "add", Inputs: Params{"a": I64, "b": I64}, Outputs: Params{"output": I64}},
//	    },
//	    Nodes: Nodes{
//	        {Key: "n1", Type: "add", Config: map[string]any{}},
//	    },
//	    Edges: Edges{
//	        {Source: Handle{Node: "input_a", Param: "value"}, Target: Handle{Node: "n1", Param: "a"}},
//	        {Source: Handle{Node: "input_b", Param: "value"}, Target: Handle{Node: "n1", Param: "b"}},
//	        {Source: Handle{Node: "n1", Param: "output"}, Target: Handle{Node: "output_c", Param: "value"}},
//	    },
//	    Strata: Strata{
//	        {"input_a", "input_b"},  // Stratum 0: inputs
//	        {"n1"},                   // Stratum 1: depends on inputs
//	        {"output_c"},             // Stratum 2: depends on n1
//	    },
//	}
package ir

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// IR is the intermediate representation of an Arc program. It contains function
// definitions, instantiated nodes, dataflow edges, execution stratification, and
// the symbol table from analysis.
type IR struct {
	// Functions contains all function and stage definitions in the program.
	Functions Functions `json:"functions"`
	// Sequences contains all sequence (state machine) definitions in the program.
	// Each sequence contains its own stages with embedded node references.
	Sequences Sequences `json:"sequences,omitempty"`
	// Nodes contains all instantiated function instances in the dataflow graph.
	Nodes Nodes `json:"nodes"`
	// Edges contains all dataflow connections between node parameters.
	// Edge.Kind distinguishes Continuous (-> reactive) from OneShot (=> transition) edges.
	Edges Edges `json:"edges"`
	// Strata contains the execution stratification for deterministic reactive execution.
	Strata Strata `json:"strata"`
	// Symbols is the symbol table from semantic analysis.
	Symbols *symbol.Scope `json:"-"`
	// TypeMap contains inferred types from the analyzer.
	TypeMap map[antlr.ParserRuleContext]types.Type `json:"-"`
}

func (i *IR) IsZero() bool {
	return len(i.Functions) == 0 &&
		len(i.Sequences) == 0 &&
		len(i.Nodes) == 0 &&
		len(i.Edges) == 0 &&
		len(i.Strata) == 0 &&
		i.Symbols == nil &&
		i.TypeMap == nil
}

// String returns the string representation of the IR.
func (i *IR) String() string {
	return i.stringWithPrefix("")
}

// stringWithPrefix returns the string representation with tree formatting.
func (i *IR) stringWithPrefix(prefix string) string {
	var b strings.Builder

	hasFunctions := len(i.Functions) > 0
	hasNodes := len(i.Nodes) > 0
	hasEdges := len(i.Edges) > 0
	hasStrata := len(i.Strata) > 0
	hasSequences := len(i.Sequences) > 0

	// Functions
	if hasFunctions {
		isLast := !hasNodes && !hasEdges && !hasStrata && !hasSequences
		i.writeFunctions(&b, prefix, isLast)
	}

	// Nodes
	if hasNodes {
		isLast := !hasEdges && !hasStrata && !hasSequences
		i.writeNodes(&b, prefix, isLast)
	}

	// Edges
	if hasEdges {
		isLast := !hasStrata && !hasSequences
		i.writeEdges(&b, prefix, isLast)
	}

	// Strata
	if hasStrata {
		isLast := !hasSequences
		i.writeStrata(&b, prefix, isLast)
	}

	// Sequences
	if hasSequences {
		i.writeSequences(&b, prefix, true)
	}

	return b.String()
}

func (i *IR) writeFunctions(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	fmt.Fprintf(b, "Functions (%d)\n", len(i.Functions))
	childPrefix := prefix + treeIndent(last)
	for j, f := range i.Functions {
		isLast := j == len(i.Functions)-1
		b.WriteString(childPrefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(f.stringWithPrefix(childPrefix + treeIndent(isLast)))
	}
}

func (i *IR) writeNodes(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	fmt.Fprintf(b, "Nodes (%d)\n", len(i.Nodes))
	childPrefix := prefix + treeIndent(last)
	for j, n := range i.Nodes {
		isLast := j == len(i.Nodes)-1
		b.WriteString(childPrefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(n.stringWithPrefix(childPrefix + treeIndent(isLast)))
	}
}

func (i *IR) writeEdges(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	fmt.Fprintf(b, "Edges (%d)\n", len(i.Edges))
	childPrefix := prefix + treeIndent(last)
	for j, e := range i.Edges {
		isLast := j == len(i.Edges)-1
		b.WriteString(childPrefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(e.String())
		b.WriteString("\n")
	}
}

func (i *IR) writeStrata(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	fmt.Fprintf(b, "Strata (%d layers)\n", len(i.Strata))
	childPrefix := prefix + treeIndent(last)
	b.WriteString(i.Strata.stringWithPrefix(childPrefix))
}

func (i *IR) writeSequences(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	fmt.Fprintf(b, "Sequences (%d)\n", len(i.Sequences))
	childPrefix := prefix + treeIndent(last)
	for j, s := range i.Sequences {
		isLast := j == len(i.Sequences)-1
		b.WriteString(childPrefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(s.stringWithPrefix(childPrefix + treeIndent(isLast)))
	}
}
