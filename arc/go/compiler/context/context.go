// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package context

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// LoopEntry tracks the block nesting depth at the point where a loop's block
// and loop instructions were emitted, enabling correct label computation for
// break (targets the outer block) and continue (targets the loop).
type LoopEntry struct {
	BreakDepth    int
	ContinueDepth int
}

// Context maintains compilation state across all code generation
type Context[ASTNode antlr.ParserRuleContext] struct {
	context.Context
	AST      ASTNode
	Resolver *resolve.Resolver
	Scope    *symbol.Scope
	Writer   *wasm.Writer
	Module   *wasm.Module
	TypeMap  map[antlr.ParserRuleContext]types.Type
	WriterID int
	// Outputs and OutputMemoryBase are set for multi-output functions
	Outputs          types.Params
	Hint             types.Type
	OutputMemoryBase uint32
	// blockDepth tracks the current WASM block nesting depth for label computation.
	blockDepth int
	// loopStack tracks active loops for break/continue label resolution.
	loopStack []LoopEntry
}

func Child[P, ASTNode antlr.ParserRuleContext](ctx Context[P], node ASTNode) Context[ASTNode] {
	return Context[ASTNode]{
		Context:          ctx.Context,
		Resolver:         ctx.Resolver,
		Scope:            ctx.Scope,
		Writer:           ctx.Writer,
		Module:           ctx.Module,
		TypeMap:          ctx.TypeMap,
		AST:              node,
		Hint:             ctx.Hint,
		Outputs:          ctx.Outputs,
		OutputMemoryBase: ctx.OutputMemoryBase,
		WriterID:   ctx.WriterID,
		blockDepth: ctx.blockDepth,
		loopStack:  ctx.loopStack,
	}
}

func (c Context[AstNode]) WithHint(hint types.Type) Context[AstNode] {
	c.Hint = hint
	return c
}

func (c Context[AstNode]) WithScope(scope *symbol.Scope) Context[AstNode] {
	c.Scope = scope
	return c
}

// EnterBlock returns a new context with an incremented block depth. This should
// be called whenever a WASM block, loop, or if instruction is emitted.
func (c Context[ASTNode]) EnterBlock() Context[ASTNode] {
	c.blockDepth++
	return c
}

// BlockDepth returns the current WASM block nesting depth.
func (c Context[ASTNode]) BlockDepth() int {
	return c.blockDepth
}

// EnterLoop returns a new context with the given loop entry pushed onto the
// loop stack. The stack is copied so the caller's context is not affected.
func (c Context[ASTNode]) EnterLoop(entry LoopEntry) Context[ASTNode] {
	copied := make([]LoopEntry, len(c.loopStack)+1)
	copy(copied, c.loopStack)
	copied[len(c.loopStack)] = entry
	c.loopStack = copied
	return c
}

// CurrentLoop returns the innermost loop entry and true, or a zero value and
// false if there is no enclosing loop.
func (c Context[ASTNode]) CurrentLoop() (LoopEntry, bool) {
	if len(c.loopStack) == 0 {
		return LoopEntry{}, false
	}
	return c.loopStack[len(c.loopStack)-1], true
}

func (c Context[ASTNode]) WithNewWriter() Context[ASTNode] {
	c.Writer = wasm.NewWriter()
	if c.Resolver != nil {
		c.WriterID = c.Resolver.TrackWriter(c.Writer)
	}
	return c
}

func CreateRoot(
	ctx context.Context,
	symbols *symbol.Scope,
	typeMap map[antlr.ParserRuleContext]types.Type,
	resolver *resolve.Resolver,
) Context[antlr.ParserRuleContext] {
	w := wasm.NewWriter()
	var writerID int
	if resolver != nil {
		writerID = resolver.TrackWriter(w)
	}
	return Context[antlr.ParserRuleContext]{
		Context:  ctx,
		Module:   wasm.NewModule(),
		Scope:    symbols,
		TypeMap:  typeMap,
		Writer:   w,
		Resolver: resolver,
		WriterID: writerID,
	}
}
