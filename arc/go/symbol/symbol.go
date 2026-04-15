// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package symbol implements symbol table management for the Arc programming language.
//
// This package provides a hierarchical scoping system that tracks all named entities
// (variables, functions, channels, etc.) throughout an Arc program. It supports lexical
// scoping with parent-child relationships, cascading symbol resolution, automatic ID
// assignment for variables, and pluggable global symbol resolvers.
//
// # Core Concepts
//
// Scope Hierarchy: Scopes form a tree structure where each scope can have a parent and
// multiple children. The root scope represents the entire program, and child scopes
// represent functions, blocks, and other lexical scopes.
//
// ID Assignment: Variables, inputs, outputs, config, and stateful variables receive
// unique integer IDs within their containing function scope. Functions create new ID
// counter scopes, ensuring that variable IDs are independent per function.
//
// Symbol Resolution: Name lookup searches in order: children → global resolver → parent
// scope. This enables lexical scoping with proper shadowing semantics.
//
// # Basic Usage
//
//	// Create root scope with global symbols
//	resolver := symbol.MapResolver{
//	    "pi": symbol.Symbol{Name: "pi", Kind: symbol.KindConfig, Type: types.F64()},
//	}
//	root := symbol.CreateRootScope(resolver)
//
//	// Add function scope
//	funcScope, _ := root.Add(ctx, symbol.Symbol{
//	    Name: "main",
//	    Kind: symbol.KindFunction,
//	})
//
//	// Add variable to function
//	varScope, _ := funcScope.Add(ctx, symbol.Symbol{
//	    Name: "x",
//	    Kind: symbol.KindVariable,
//	    Type: types.I32(),
//	})
//
//	// Resolve symbols
//	resolved, _ := varScope.Resolve(ctx, "pi")  // Finds global symbol
package symbol

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/types"
)

// ExecContext indicates which execution context a symbol is valid in.
type ExecContext int

const (
	// ExecWASM marks a symbol as only usable in func blocks (WASM compilation).
	ExecWASM ExecContext = iota + 1
	// ExecFlow marks a symbol as only usable in flow statements (graph nodes).
	ExecFlow
	// ExecBoth marks a symbol as usable in both contexts.
	ExecBoth
)

// Compatible reports whether a symbol with exec context e is visible in a
// context that filters for target. An untagged symbol (e == 0) is never
// visible. An unset filter (target == 0) shows all tagged symbols.
func (e ExecContext) Compatible(target ExecContext) bool {
	if e == 0 {
		return false // untagged symbol is never visible
	}
	if target == 0 {
		return true // no filter active = show everything tagged
	}
	if e == ExecBoth || target == ExecBoth {
		return true
	}
	return e == target
}

// Kind categorizes symbols by their role in the Arc program.
type Kind int

//go:generate stringer -type=Kind
const (
	// KindVariable represents a regular variable within a function or block scope.
	KindVariable Kind = iota
	// KindStatefulVariable represents a variable that persists across reactive stage executions.
	KindStatefulVariable
	// KindChannel represents a channel type for inter-task communication via unbounded FIFO queues.
	KindChannel
	// KindFunction represents a function declaration.
	KindFunction
	// KindBlock represents a block scope such as a task or stage.
	KindBlock
	// KindConfig represents a configuration parameter (constant).
	KindConfig
	// KindInput represents an input parameter to a function or task.
	KindInput
	// KindOutput represents an output parameter from a function or task.
	KindOutput
	// KindSequence represents a sequence (state machine) declaration.
	KindSequence
	// KindStage represents a stage within a sequence.
	KindStage
	// KindConstant represents a pure literal value in a flow statement.
	KindConstant
	// KindGlobalConstant represents a compile-time constant declared at global scope.
	// Values are inlined at each reference site with no runtime overhead.
	KindGlobalConstant
	// KindLoop represents a loop scope (for break/continue validation).
	KindLoop
	// KindLoopVariable represents an immutable loop iteration variable.
	KindLoopVariable
)

// Symbol represents a named entity in an Arc program.
//
// Each symbol has an associated type from Arc's type system, a kind that categorizes
// its role, and an optional AST node for source location information used in error
// reporting. Symbols that receive unique IDs (variables, inputs, outputs, config, and
// stateful variables) are assigned sequential IDs within their containing function scope.
type Symbol struct {
	// Type is the symbol's type from Arc's type system.
	Type types.Type
	// AST is the parser node for source location information. Global symbols from
	// resolvers have AST == nil, while locally-defined symbols have non-nil AST.
	AST antlr.ParserRuleContext
	// DefaultValue stores the default value literal for optional parameters.
	// Only used for KindInput and KindConfig symbols. Nil means no default (required parameter).
	DefaultValue any
	// Name is the symbol's identifier.
	Name string
	// Kind categorizes the symbol (variable, function, channel, etc.).
	Kind Kind
	// ID is a unique identifier within the containing function scope. Only assigned
	// to KindVariable, KindStatefulVariable, KindInput, KindOutput, KindChannel, and
	// KindConfig.
	ID int
	// SourceID tracks the ID of the source symbol for channel type propagation.
	// When a variable or config param holds a channel reference, this field stores
	// the ID of the original source (config param or global channel) so that
	// Channels.Read/Write can be correctly resolved at instantiation time.
	// A nil value means this symbol is the original source (e.g., a config param).
	SourceID *int
	// Internal marks symbols that are only accessible to the compiler (e.g., WASM
	// host function signatures used for type suffix derivation). Internal symbols
	// are skipped during scope resolution so user code cannot reference them, but
	// remain visible to the compiler's resolve.Resolver which queries the
	// symbol.Resolver interface directly.
	Internal bool
	// Exec indicates which execution context this symbol is valid in (WASM, Flow,
	// or Both). A zero value is invalid and will cause resolution to fail, forcing
	// every symbol to be explicitly tagged.
	Exec ExecContext
}
