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
)

func (i *IR) IsZero() bool {
	return len(i.Functions) == 0 &&
		len(i.Nodes) == 0 &&
		len(i.Edges) == 0 &&
		i.Root.IsZero() &&
		i.Symbols == nil &&
		i.TypeMap == nil
}

// IsZero reports whether the scope carries no execution content.
func (s Scope) IsZero() bool {
	return s.Key == "" &&
		s.Mode == ScopeModeUnspecified &&
		s.Liveness == LivenessUnspecified &&
		s.Activation == nil &&
		len(s.Strata) == 0 &&
		len(s.Steps) == 0 &&
		len(s.Transitions) == 0
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
	b.WriteString(treePrefix(last))
	lo.Must(fmt.Fprintf(b, "Functions (%d)\n", len(i.Functions)))
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
	lo.Must(fmt.Fprintf(b, "Nodes (%d)\n", len(i.Nodes)))
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
	lo.Must(fmt.Fprintf(b, "Edges (%d)\n", len(i.Edges)))
	childPrefix := prefix + treeIndent(last)
	for j, e := range i.Edges {
		isLast := j == len(i.Edges)-1
		b.WriteString(childPrefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(e.String())
		b.WriteString("\n")
	}
}

func (i *IR) writeRoot(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	b.WriteString("Root\n")
	childPrefix := prefix + treeIndent(last)
	b.WriteString(i.Root.stringWithPrefix(childPrefix))
}

// String returns the tree representation of a Scope.
func (s Scope) String() string { return s.stringWithPrefix("") }

func (s Scope) stringWithPrefix(prefix string) string {
	var b strings.Builder
	lo.Must(fmt.Fprintf(&b, "%s [%s, %s]\n", scopeLabel(s), s.Mode, s.Liveness))
	if s.Mode == ScopeModeParallel {
		for i, stratum := range s.Strata {
			isLast := i == len(s.Strata)-1 && len(s.Transitions) == 0
			b.WriteString(prefix)
			b.WriteString(treePrefix(isLast))
			lo.Must(fmt.Fprintf(&b, "stratum %d\n", i))
			childPrefix := prefix + treeIndent(isLast)
			for j, m := range stratum {
				isLastMember := j == len(stratum)-1
				b.WriteString(childPrefix)
				b.WriteString(treePrefix(isLastMember))
				b.WriteString(m.stringWithPrefix(childPrefix + treeIndent(isLastMember)))
			}
		}
	} else {
		for i, m := range s.Steps {
			isLast := i == len(s.Steps)-1 && len(s.Transitions) == 0
			b.WriteString(prefix)
			b.WriteString(treePrefix(isLast))
			b.WriteString(m.stringWithPrefix(prefix + treeIndent(isLast)))
		}
	}
	for i, t := range s.Transitions {
		isLast := i == len(s.Transitions)-1
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(t.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func scopeLabel(s Scope) string {
	if s.Key == "" {
		return "(scope)"
	}
	return s.Key
}

// NodeMember builds a leaf Member referencing the node with the given key.
func NodeMember(key string) Member { return Member{NodeKey: new(key)} }

// ScopeMember builds a Member wrapping the given nested Scope.
func ScopeMember(s Scope) Member { return Member{Scope: &s} }

// Key returns the member's lookup key — the string transitions target via
// `=> name`. Derived from the set variant: the referenced node's key for
// leaf members, the nested scope's key for scope members. Returns the empty
// string for an unset member.
func (m Member) Key() string {
	switch {
	case m.NodeKey != nil:
		return *m.NodeKey
	case m.Scope != nil:
		return m.Scope.Key
	default:
		return ""
	}
}

// String returns the tree representation of a Member.
func (m Member) String() string { return m.stringWithPrefix("") }

func (m Member) stringWithPrefix(prefix string) string {
	switch {
	case m.NodeKey != nil:
		return fmt.Sprintf("%s\n", *m.NodeKey)
	case m.Scope != nil:
		return m.Scope.stringWithPrefix(prefix)
	default:
		return "(empty member)\n"
	}
}

// String returns a concise description of the transition.
func (t Transition) String() string {
	target := "=> exit"
	if t.TargetKey != nil {
		target = "=> " + *t.TargetKey
	}
	return fmt.Sprintf("on %s/%s %s", t.On.Node, t.On.Param, target)
}
