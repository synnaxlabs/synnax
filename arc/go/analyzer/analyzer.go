// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"github.com/antlr4-go/antlr/v4"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/function"
	"github.com/synnaxlabs/arc/analyzer/sequence"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
)

func AnalyzeProgram(ctx acontext.Context[parser.IProgramContext]) bool {
	if !collectDeclarations(ctx) {
		return false
	}
	if !analyzeDeclarations(ctx) {
		return false
	}
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		if !applyTypeSubstitutionsToSymbols(ctx, ctx.Scope) {
			return false
		}
		substituteTypeMap(ctx)
	}
	return true
}

func substituteTypeMap(ctx acontext.Context[parser.IProgramContext]) {
	for node, typ := range ctx.TypeMap {
		ctx.TypeMap[node] = ctx.Constraints.ApplySubstitutions(typ)
	}
}

func collectDeclarations(ctx acontext.Context[parser.IProgramContext]) bool {
	if !function.CollectDeclarations(ctx) {
		return false
	}
	if !sequence.CollectDeclarations(ctx) {
		return false
	}
	return true
}

func analyzeDeclarations(ctx acontext.Context[parser.IProgramContext]) bool {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			if !function.Analyze(acontext.Child(ctx, funcDecl)) {
				return false
			}
		} else if flowStmt := item.FlowStatement(); flowStmt != nil {
			if !flow.Analyze(acontext.Child(ctx, flowStmt)) {
				return false
			}
		} else if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			if !sequence.Analyze(acontext.Child(ctx, seqDecl)) {
				return false
			}
		}
	}
	return true
}

func AnalyzeStatement(ctx acontext.Context[parser.IStatementContext]) bool {
	if !statement.Analyze(ctx) {
		return false
	}
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
	}
	return true
}

func AnalyzeBlock(ctx acontext.Context[parser.IBlockContext]) bool {
	if !statement.AnalyzeBlock(ctx) {
		return false
	}
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
	}
	return true
}

func applyTypeSubstitutionsToSymbols[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	scope *symbol.Scope,
) bool {
	if scope.Type.IsValid() {
		scope.Type = ctx.Constraints.ApplySubstitutions(scope.Type)
	}
	for _, child := range scope.Children {
		if !applyTypeSubstitutionsToSymbols[T](ctx, child) {
			return false
		}
	}
	return true
}
