// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/compiler/runtime"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
)

// Context maintains compilation state across all code generation
type Context[ASTNode antlr.ParserRuleContext] struct {
	context.Context
	Imports *runtime.ImportIndex
	Scope   *ir.Scope
	Writer  *wasm.Writer
	Module  *wasm.Module
	AST     ASTNode
	Hint    ir.Type
}

func Child[P, ASTNode antlr.ParserRuleContext](ctx Context[P], node ASTNode) Context[ASTNode] {
	return Context[ASTNode]{
		Context: ctx.Context,
		Imports: ctx.Imports,
		Scope:   ctx.Scope,
		Writer:  ctx.Writer,
		Module:  ctx.Module,
		AST:     node,
		Hint:    ctx.Hint,
	}
}
func (c Context[AstNode]) WithHint(hint ir.Type) Context[AstNode] {
	c.Hint = hint
	return c
}

func (c Context[AstNode]) WithScope(scope *ir.Scope) Context[AstNode] {
	c.Scope = scope
	return c
}

func (c Context[ASTNode]) WithNewWriter() Context[ASTNode] {
	c.Writer = wasm.NewWriter()
	return c
}

func CreateRoot(
	ctx_ context.Context,
	symbols *ir.Scope,
	disableHostImports bool,
) Context[antlr.ParserRuleContext] {
	ctx := Context[antlr.ParserRuleContext]{
		Context: ctx_,
		Module:  wasm.NewModule(),
		Scope:   symbols,
		Writer:  wasm.NewWriter(),
	}
	if !disableHostImports {
		ctx.Imports = runtime.SetupImports(ctx.Module)
	}
	return ctx
}
