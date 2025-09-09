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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/diagnostics"
	"github.com/synnaxlabs/arc/ir"
)

type Context[ASTNode antlr.ParserRuleContext] struct {
	Scope       *ir.Scope
	Diagnostics *diagnostics.Diagnostics
	AST         ASTNode
}

func CreateRoot[ASTNode antlr.ParserRuleContext](
	ast ASTNode,
	resolver ir.SymbolResolver,
) Context[ASTNode] {
	return Context[ASTNode]{
		Scope:       ir.CreateRootScope(resolver),
		Diagnostics: &diagnostics.Diagnostics{},
		AST:         ast,
	}

}

func Child[P, N antlr.ParserRuleContext](ctx Context[P], next N) Context[N] {
	return ChildWithScope(ctx, next, ctx.Scope)
}

func ChildWithScope[P, N antlr.ParserRuleContext](ctx Context[P], nextAST N, nextScope *ir.Scope) Context[N] {
	return Context[N]{Scope: nextScope, Diagnostics: ctx.Diagnostics, AST: nextAST}
}
