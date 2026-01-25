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
	"github.com/synnaxlabs/arc/analyzer/constraints"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/function"
	"github.com/synnaxlabs/arc/analyzer/sequence"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
)

func AnalyzeProgram(ctx acontext.Context[parser.IProgramContext]) {
	collectDeclarations(ctx)
	analyzeDeclarations(ctx)
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			addUnificationError(ctx.Diagnostics, err, ctx.AST)
			return
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
		substituteTypeMap(ctx)
	}
}

func substituteTypeMap(ctx acontext.Context[parser.IProgramContext]) {
	for node, typ := range ctx.TypeMap {
		ctx.TypeMap[node] = ctx.Constraints.ApplySubstitutions(typ)
	}
}

func collectDeclarations(ctx acontext.Context[parser.IProgramContext]) {
	function.CollectDeclarations(ctx)
	sequence.CollectDeclarations(ctx)
}

func analyzeDeclarations(ctx acontext.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			function.Analyze(acontext.Child(ctx, funcDecl))
		} else if flowStmt := item.FlowStatement(); flowStmt != nil {
			flow.Analyze(acontext.Child(ctx, flowStmt))
		} else if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			sequence.Analyze(acontext.Child(ctx, seqDecl))
		}
	}
}

func AnalyzeStatement(ctx acontext.Context[parser.IStatementContext]) {
	statement.Analyze(ctx)
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			addUnificationError(ctx.Diagnostics, err, ctx.AST)
			return
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
	}
}

func AnalyzeBlock(ctx acontext.Context[parser.IBlockContext]) {
	statement.AnalyzeBlock(ctx)
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			addUnificationError(ctx.Diagnostics, err, ctx.AST)
			return
		}
		applyTypeSubstitutionsToSymbols(ctx, ctx.Scope)
	}
}

func applyTypeSubstitutionsToSymbols[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	scope *symbol.Scope,
) {
	if scope.Type.IsValid() {
		scope.Type = ctx.Constraints.ApplySubstitutions(scope.Type)
	}
	for _, child := range scope.Children {
		applyTypeSubstitutionsToSymbols[T](ctx, child)
	}
}

func addUnificationError(
	diag *diagnostics.Diagnostics,
	err error,
	fallbackCtx antlr.ParserRuleContext,
) {
	if ue, ok := err.(*constraints.UnificationError); ok && ue.Constraint != nil && ue.Constraint.Source != nil {
		diag.Add(diagnostics.Error(err, ue.Constraint.Source))
	} else {
		diag.Add(diagnostics.Error(err, fallbackCtx))
	}
}
