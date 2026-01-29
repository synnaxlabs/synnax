// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sequence

import (
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// CollectDeclarations registers all sequences and their stages in the symbol table.
// This is called during the first pass of AnalyzeProgram to establish scopes before
// analyzing function bodies that may reference sequences or stages.
func CollectDeclarations(ctx context.Context[parser.IProgramContext]) {
	// First pass: collect all sequence names to establish their scopes
	for _, item := range ctx.AST.AllTopLevelItem() {
		if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			collectSequenceName(context.Child(ctx, seqDecl))
		}
	}

	// Second pass: collect all stages (now we can check for name collisions with all sequences)
	for _, item := range ctx.AST.AllTopLevelItem() {
		if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			collectSequenceStages(context.Child(ctx, seqDecl))
		}
	}
}

// collectSequenceName registers a sequence in the symbol table (first pass).
func collectSequenceName(ctx context.Context[parser.ISequenceDeclarationContext]) {
	name := ctx.AST.IDENTIFIER().GetText()
	if _, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindSequence,
		Type: types.Sequence(),
		AST:  ctx.AST,
	}); err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
	}
}

func collectSequenceStages(ctx context.Context[parser.ISequenceDeclarationContext]) {
	name := ctx.AST.IDENTIFIER().GetText()
	seqScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	stages := ctx.AST.AllStageDeclaration()
	for _, stageDecl := range stages {
		collectStage(context.Child(ctx, stageDecl).WithScope(seqScope), seqScope)
	}
}

func collectStage(
	ctx context.Context[parser.IStageDeclarationContext],
	seqScope *symbol.Scope,
) {
	stageName := ctx.AST.IDENTIFIER().GetText()
	if _, err := seqScope.Add(ctx, symbol.Symbol{
		Name: stageName,
		Kind: symbol.KindStage,
		Type: types.Stage(),
		AST:  ctx.AST,
	}); err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
	}
}

// Analyze performs semantic analysis on a sequence declaration.
// This is called during the second pass after all declarations have been collected.
func Analyze(ctx context.Context[parser.ISequenceDeclarationContext]) {
	name := ctx.AST.IDENTIFIER().GetText()
	seqScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	for _, stageDecl := range ctx.AST.AllStageDeclaration() {
		analyzeStage(context.Child(ctx, stageDecl).WithScope(seqScope))
	}
}

// analyzeStage performs semantic analysis on a stage declaration.
// With unified flow statements, stages now just contain flows (no special transitions).
func analyzeStage(
	ctx context.Context[parser.IStageDeclarationContext],
) {
	stageBody := ctx.AST.StageBody()
	if stageBody == nil {
		return
	}
	for _, item := range stageBody.AllStageItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			flow.Analyze(context.Child(ctx, flowStmt))
		}
		if single := item.SingleInvocation(); single != nil {
			analyzeSingleInvocation(context.Child(ctx, single))
		}
	}
}

func analyzeSingleInvocation(ctx context.Context[parser.ISingleInvocationContext]) {
	if fn := ctx.AST.Function(); fn != nil {
		flow.AnalyzeSingleFunction(context.Child(ctx, fn))
		return
	}
	if expr := ctx.AST.Expression(); expr != nil {
		flow.AnalyzeSingleExpression(context.Child(ctx, expr))
	}
}
