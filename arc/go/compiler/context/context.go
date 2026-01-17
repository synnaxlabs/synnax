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
	"github.com/synnaxlabs/arc/compiler/bindings"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// Context maintains compilation state across all code generation
type Context[ASTNode antlr.ParserRuleContext] struct {
	context.Context
	AST     ASTNode
	Imports *bindings.ImportIndex
	Scope   *symbol.Scope
	Writer  *wasm.Writer
	Module  *wasm.Module
	TypeMap map[antlr.ParserRuleContext]types.Type
	// FunctionIndices maps function names to their WASM function indices for call resolution
	FunctionIndices map[string]uint32
	// Outputs and OutputMemoryBase are set for multi-output functions
	Outputs          types.Params
	Hint             types.Type
	OutputMemoryBase uint32
}

func Child[P, ASTNode antlr.ParserRuleContext](ctx Context[P], node ASTNode) Context[ASTNode] {
	return Context[ASTNode]{
		Context:          ctx.Context,
		Imports:          ctx.Imports,
		Scope:            ctx.Scope,
		Writer:           ctx.Writer,
		Module:           ctx.Module,
		TypeMap:          ctx.TypeMap,
		AST:              node,
		Hint:             ctx.Hint,
		Outputs:          ctx.Outputs,
		OutputMemoryBase: ctx.OutputMemoryBase,
		FunctionIndices:  ctx.FunctionIndices,
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

func (c Context[ASTNode]) WithNewWriter() Context[ASTNode] {
	c.Writer = wasm.NewWriter()
	return c
}

func CreateRoot(
	ctx context.Context,
	symbols *symbol.Scope,
	typeMap map[antlr.ParserRuleContext]types.Type,
	disableHostImports bool,
) Context[antlr.ParserRuleContext] {
	compCtx := Context[antlr.ParserRuleContext]{
		Context:         ctx,
		Module:          wasm.NewModule(),
		Scope:           symbols,
		TypeMap:         typeMap,
		Writer:          wasm.NewWriter(),
		FunctionIndices: make(map[string]uint32),
	}
	if !disableHostImports {
		compCtx.Imports = bindings.SetupImports(compCtx.Module)
	}
	return compCtx
}
