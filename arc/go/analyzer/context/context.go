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
//	funcCtx := rootCtx.Child(functionDecl)
//
//	// Create context with new scope for function body
//	bodyCtx := funcCtx.WithScope(functionScope)
//
//	// Add type hint for expression analysis
//	exprCtx := bodyCtx.WithTypeHint(types.I32())
package context

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/formatter"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// ChannelMapping records the actual channel passed for a chan-typed parameter at a
// call site.
type ChannelMapping struct {
	ChannelID   uint32
	ChannelName string
}

// CallEdge records a function call relationship between a caller and callee scope.
// These edges are collected during analysis and used in a post-pass to propagate
// channel accesses through function calls, handling forward references where the
// callee is declared after the caller in source order.
type CallEdge struct {
	Caller   *symbol.Symbol
	Callee   *symbol.Symbol
	CallSite antlr.ParserRuleContext
	// ArgChannels maps input parameter index (position in the callee's input list)
	// to the actual channel ID/name passed at this call site. Only populated for
	// chan-typed parameters. Keyed by index rather than symbol ID so the mapping
	// is valid even when the callee hasn't been fully analyzed yet (forward refs).
	ArgChannels map[int]ChannelMapping
}

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
	// Context is the standard Go context for cancellation and deadlines.
	context.Context
	// Scope is the current symbol scope for name resolution.
	Scope *symbol.Symbol
	// Diagnostics accumulates errors and warnings during analysis.
	Diagnostics *diagnostics.Diagnostics
	// Constraints accumulates type constraints for unification.
	Constraints *constraints.System
	// TypeMap caches resolved types for AST nodes.
	TypeMap map[antlr.ParserRuleContext]types.Type
	// FmtStrings holds the segments of each format string, so the compiler compiles
	// the same placeholder nodes the analyzer typed.
	FmtStrings parser.FmtStrings
	// CallEdges tracks function call relationships for post-analysis channel
	// propagation.
	CallEdges *[]CallEdge
	// AST is the current AST node being analyzed.
	AST AST
	// TypeHint is the expected type from surrounding context for type inference.
	TypeHint types.Type
	// InTypeInferenceMode indicates whether type inference is active.
	InTypeInferenceMode bool
	// Config carries per-parse language settings (e.g. dashed channel names).
	Config parser.Config
	// Module analyzes the program as a module, under the module rules.
	Module bool
}

// WithScope returns a new context with an updated scope. The original context is
// not mutated. All other fields (including shared state pointers) are preserved.
//
// WithScope is typically used when entering a new lexical scope, such as a function
// body or block statement.
func (c Context[AST]) WithScope(scope *symbol.Symbol) Context[AST] {
	c.Scope = scope
	return c
}

// WithModule returns a new context that analyzes the program as a module.
func (c Context[AST]) WithModule(module bool) Context[AST] {
	c.Module = module
	return c
}

// IdentifierAST is a declaration with an optional IDENTIFIER (nil when anonymous).
type IdentifierAST interface {
	antlr.ParserRuleContext
	IDENTIFIER() antlr.TerminalNode
}

// ResolveOwnScope returns the scope owned by the given declaration.
func ResolveOwnScope[T IdentifierAST](ctx Context[T]) (*symbol.Symbol, error) {
	if id := ctx.AST.IDENTIFIER(); id != nil {
		return ctx.Scope.Resolve(ctx, id.GetText())
	}
	return ctx.Scope.GetChildByParserRule(ctx.AST)
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

// WithConfig returns a new context carrying the given parse settings. The original
// context is not mutated.
func (c Context[AST]) WithConfig(cfg parser.Config) Context[AST] {
	c.Config = cfg
	return c
}

// Resolve resolves a single (unqualified) name in the current scope. If the
// resolved symbol is deprecated, a warning is added to diagnostics. Use this
// for bare identifier resolution. For qualified names (e.g., `math.abs`) use
// ResolveQualified so the head and tail are resolved as separate symbols.
func (c Context[AST]) Resolve(name string) (*symbol.Symbol, error) {
	result, err := c.Scope.Resolve(c, name)
	if err != nil {
		return result, err
	}
	c.warnIfDeprecated(name, result)
	return result, nil
}

// ResolveQualified resolves a possibly-qualified name. head is the first
// segment (always present); tail is the second segment, empty when the
// reference is bare. When tail is non-empty, ResolveQualified follows
// the head and then resolves tail through it; an alias head transparently
// dispatches to its Target inside Resolve. Deprecation warnings fire on
// whichever segment resolved to the deprecated symbol.
func (c Context[AST]) ResolveQualified(head, tail string) (*symbol.Symbol, error) {
	headSym, err := c.Scope.Resolve(c, head)
	if err != nil {
		return nil, err
	}
	if tail == "" {
		c.warnIfDeprecated(head, headSym)
		return headSym, nil
	}
	tailSym, err := headSym.Resolve(c, tail)
	if err != nil {
		return nil, err
	}
	c.warnIfDeprecated(head+"."+tail, tailSym)
	return tailSym, nil
}

// RejectTopLevel reports that name cannot be reassigned and returns true when sym is a
// variable declared at the top level of its program or module.
func (c Context[AST]) RejectTopLevel(sym *symbol.Symbol, name string) bool {
	if !sym.IsValueVariable() || sym.Parent != sym.Root() {
		return false
	}
	c.Diagnostics.Add(diagnostics.Errorf(c.AST,
		"cannot reassign top-level variable '%s'", name))
	return true
}

// FloatConstant returns the value of the float module constant that node reduces to,
// such as math.nan, with parentheses, f32 and f64 casts, and negation applied.
func (c Context[AST]) FloatConstant(node antlr.Tree) (float64, bool) {
	primary, negated := parser.SoleOperand(node)
	if primary == nil || primary.QualifiedIdentifier() == nil {
		return 0, false
	}
	head, tail := parser.QualifiedNameParts(primary.QualifiedIdentifier())
	headSym, err := c.Scope.Resolve(c, head)
	if err != nil {
		return 0, false
	}
	sym, err := headSym.Resolve(c, tail)
	if err != nil || sym.Kind != symbol.KindConstant {
		return 0, false
	}
	value, ok := sym.DefaultValue.(float64)
	if negated {
		value = -value
	}
	return value, ok
}

func (c Context[AST]) warnIfDeprecated(name string, sym *symbol.Symbol) {
	if sym == nil || sym.Deprecated == nil {
		return
	}
	d := codes.DeprecatedSymbol.At(c.AST, name, sym.Deprecated.QualifiedName())
	if token := bareIdentifier(c.AST, name); token != nil {
		d = c.withReplacement(d, token, sym.Deprecated)
	}
	c.Diagnostics.Add(d)
}

// withReplacement attaches the fix that writes repl in place of token.
func (c Context[AST]) withReplacement(
	d diagnostics.Diagnostic,
	token antlr.Token,
	repl *symbol.Symbol,
) diagnostics.Diagnostic {
	var (
		root  = c.Scope.Root()
		name  = repl.QualifiedName()
		edits []protocol.TextEdit
	)
	if module := repl.Parent; module != nil && module.Kind == symbol.KindModule {
		for _, child := range root.Children() {
			if child.Kind == symbol.KindModuleAlias && child.Target != nil &&
				child.Target.Name == module.Name {
				name = child.Name + "." + repl.Name
				break
			}
		}
		edits = formatter.AddImport(root, module.Name)
	}
	line, column := uint32(token.GetLine()-1), uint32(token.GetColumn())
	edits = append(edits, protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: column},
			End: protocol.Position{
				Line:      line,
				Character: column + uint32(len(token.GetText())),
			},
		},
		NewText: name,
	})
	return d.WithEdits(
		fmt.Sprintf("Replace '%s' with '%s'", token.GetText(), name),
		edits...,
	)
}

// bareIdentifier returns the first identifier in node that is name and not qualified.
func bareIdentifier(node antlr.Tree, name string) antlr.Token {
	if node == nil {
		return nil
	}
	if terminal, ok := node.(antlr.TerminalNode); ok {
		token := terminal.GetSymbol()
		_, qualified := terminal.GetParent().(parser.IQualifiedIdentifierContext)
		if token.GetTokenType() == parser.ArcParserIDENTIFIER &&
			token.GetText() == name && !qualified {
			return token
		}
		return nil
	}
	for _, child := range node.GetChildren() {
		if token := bareIdentifier(child, name); token != nil {
			return token
		}
	}
	return nil
}

// ScopedKey returns a name for node that is unique across a program. A graph parses
// each function body on its own, so a source position alone repeats between bodies.
func (c Context[AST]) ScopedKey(prefix string, node antlr.ParserRuleContext) string {
	var function string
	if fn, err := c.Scope.ClosestAncestorOfKind(symbol.KindFunction); err == nil {
		function = fn.Name
	}
	start := node.GetStart()
	return fmt.Sprintf(
		"%s_%s_%d_%d",
		prefix,
		function,
		start.GetLine(),
		start.GetColumn(),
	)
}

// NewRoot creates a new root context for program analysis. NewRoot
// initializes the shared state (Diagnostics, Constraints, TypeMap) and uses
// the supplied root symbol as the lexical scope.
//
// The root parameter is the pre-built program root. Callers in the consumer
// layer (LSP, compiler, graph mode, services) build it via symbol.NewRoot
// with any external (DynamicResolver / channel) resolution before passing
// it in. The analyzer does not know about STL or external services; it
// only walks the scope tree it is given.
//
// When root is nil, an empty root is created. This is suitable for tests
// that need to construct a hand-rolled scope without STL or external
// resolvers.
func NewRoot[ASTNode antlr.ParserRuleContext](
	ctx context.Context,
	ast ASTNode,
	root *symbol.Symbol,
) Context[ASTNode] {
	if root == nil {
		root = symbol.NewRoot(nil, nil)
	}
	return Context[ASTNode]{
		Context:     ctx,
		Scope:       root,
		Diagnostics: &diagnostics.Diagnostics{},
		Constraints: constraints.New(),
		TypeMap:     make(map[antlr.ParserRuleContext]types.Type),
		FmtStrings:  make(parser.FmtStrings),
		CallEdges:   &[]CallEdge{},
		AST:         ast,
	}
}

// Child creates a new child context for a different AST node. Child is the primary way
// to traverse the AST during analysis.
//
// The child context references the same shared state (Diagnostics, Constraints,
// TypeMap, Scope) as the parent, points to a different AST node, and preserves the
// parent's TypeHint and InTypeInferenceMode.
//
// The type parameter N is the child AST node type and can be different from the
// parent's.
func (ctx Context[P]) Child[N antlr.ParserRuleContext](next N) Context[N] {
	return Context[N]{
		Context:             ctx.Context,
		Scope:               ctx.Scope,
		Diagnostics:         ctx.Diagnostics,
		Constraints:         ctx.Constraints,
		TypeMap:             ctx.TypeMap,
		FmtStrings:          ctx.FmtStrings,
		CallEdges:           ctx.CallEdges,
		AST:                 next,
		TypeHint:            ctx.TypeHint,
		InTypeInferenceMode: ctx.InTypeInferenceMode,
		Config:              ctx.Config,
		Module:              ctx.Module,
	}
}
