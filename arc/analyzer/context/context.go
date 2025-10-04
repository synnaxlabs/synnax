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
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/analyzer/diagnostics"
	"github.com/synnaxlabs/arc/ir"
)

type Context[AST antlr.ParserRuleContext] struct {
	context.Context
	Scope       *ir.Scope
	Diagnostics *diagnostics.Diagnostics
	Constraints *constraints.System
	AST         AST
	TypeHint    ir.Type
}

func (c Context[AST]) WithScope(scope *ir.Scope) Context[AST] {
	c.Scope = scope
	return c
}

func (c Context[AST]) WithTypeHint(hint ir.Type) Context[AST] {
	c.TypeHint = hint
	return c
}

func CreateRoot[ASTNode antlr.ParserRuleContext](
	ctx context.Context,
	ast ASTNode,
	resolver ir.SymbolResolver,
) Context[ASTNode] {
	return Context[ASTNode]{
		Context:     ctx,
		Scope:       ir.CreateRootScope(resolver),
		Diagnostics: &diagnostics.Diagnostics{},
		Constraints: constraints.New(),
		AST:         ast,
	}

}

func Child[P, N antlr.ParserRuleContext](ctx Context[P], next N) Context[N] {
	return Context[N]{
		Context:     ctx.Context,
		Scope:       ctx.Scope,
		Diagnostics: ctx.Diagnostics,
		Constraints: ctx.Constraints,
		AST:         next,
		TypeHint:    ctx.TypeHint,
	}
}
