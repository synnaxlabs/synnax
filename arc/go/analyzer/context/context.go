// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package context provides the analysis context for the Arc language analyzer.
//
// Context is a generic container that flows through the entire semantic analysis
// pipeline, carrying shared state including diagnostics, type constraints, scope
// information, and type mappings.
//
// The context package uses three key patterns:
//
// Generic Type Parameter: Context is parametrized over AST node types, allowing
// type-safe traversal of different parts of the parse tree.
//
// Immutable-Style API: Methods like WithScope and WithTypeHint return new context
// values rather than mutating the receiver, supporting functional programming patterns.
//
// Shared State: While contexts are copied, they share pointers to mutable state
// (Diagnostics, Constraints, TypeMap, Scope), allowing child contexts to accumulate
// state visible to parent contexts.
//
// Example usage:
//
//	// Create root context
//	rootCtx := context.CreateRoot(stdContext, program, resolver)
//
//	// Create child context with different AST node
//	funcCtx := context.Child(rootCtx, functionDecl)
//
//	// Create context with new scope for function body
//	bodyCtx := funcCtx.WithScope(functionScope)
//
//	// Add type hint for expression analysis
//	exprCtx := bodyCtx.WithTypeHint(types.I32())
package context

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// Context is a generic container for analysis state that flows through the semantic
// analysis pipeline. It is parametrized over AST node types to provide type-safe
// traversal of the parse tree.
//
// Context uses a hybrid approach to state management. The AST field is copied when
// creating child contexts (each context points to a different node). Shared state
// (Diagnostics, Constraints, TypeMap, Scope) is referenced via pointers. TypeHint
// and InTypeInferenceMode are copied by value.
//
// This design allows child contexts to accumulate diagnostics and constraints that
// are visible to parent contexts, while maintaining immutability for local state.
type Context[AST antlr.ParserRuleContext] struct {
	context.Context
	Scope               *symbol.Scope
	Diagnostics         *diagnostics.Diagnostics
	Constraints         *constraints.System
	TypeMap             map[antlr.ParserRuleContext]types.Type
	AST                 AST
	TypeHint            types.Type
	InTypeInferenceMode bool
}

// WithScope returns a new context with an updated scope. The original context is
// not mutated. All other fields (including shared state pointers) are preserved.
//
// WithScope is typically used when entering a new lexical scope, such as a function
// body or block statement.
func (c Context[AST]) WithScope(scope *symbol.Scope) Context[AST] {
	c.Scope = scope
	return c
}

// WithTypeHint returns a new context with an updated type hint. The original context
// is not mutated. All other fields (including shared state pointers) are preserved.
//
// Type hints are used for bottom-up type inference, where the expected type of
// expression is known from context (e.g., the declared type of variable being
// assigned).
func (c Context[AST]) WithTypeHint(hint types.Type) Context[AST] {
	c.TypeHint = hint
	return c
}

// CreateRoot creates a new root context for program analysis. CreateRoot initializes
// all shared state (Diagnostics, Constraints, TypeMap) and creates the root symbol
// scope.
//
// The ctx parameter is the standard Go context for cancellation and deadlines. The
// ast parameter is the root AST node (typically a program or top-level node). The
// resolver parameter is a symbol resolver for resolving built-in symbols and can
// be nil.
func CreateRoot[ASTNode antlr.ParserRuleContext](
	ctx context.Context,
	ast ASTNode,
	resolver symbol.Resolver,
) Context[ASTNode] {
	return Context[ASTNode]{
		Context:     ctx,
		Scope:       symbol.CreateRootScope(resolver),
		Diagnostics: &diagnostics.Diagnostics{},
		Constraints: constraints.New(),
		TypeMap:     make(map[antlr.ParserRuleContext]types.Type),
		AST:         ast,
	}

}

// Child creates a new child context for a different AST node. Child is the primary
// way to traverse the AST during analysis.
//
// The child context references the same shared state (Diagnostics, Constraints,
// TypeMap, Scope) as the parent, points to a different AST node, and preserves the
// parent's TypeHint and InTypeInferenceMode.
//
// The type parameter P is the parent AST node type. The type parameter N is the
// child AST node type and can be different from P.
func Child[P, N antlr.ParserRuleContext](ctx Context[P], next N) Context[N] {
	return Context[N]{
		Context:             ctx.Context,
		Scope:               ctx.Scope,
		Diagnostics:         ctx.Diagnostics,
		Constraints:         ctx.Constraints,
		TypeMap:             ctx.TypeMap,
		AST:                 next,
		TypeHint:            ctx.TypeHint,
		InTypeInferenceMode: ctx.InTypeInferenceMode,
	}
}
