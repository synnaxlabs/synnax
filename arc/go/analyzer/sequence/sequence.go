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

// CollectDeclarations registers all sequences and their children in the symbol table.
// This is called during the first pass of AnalyzeProgram to establish scopes before
// analyzing function bodies that may reference sequences or stages.
func CollectDeclarations(ctx context.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			collectSequenceDecl(context.Child(ctx, seqDecl), ctx.Scope)
		}
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			collectTopLevelStage(context.Child(ctx, stageDecl), ctx.Scope)
		}
	}
}

// collectSequenceDecl recursively registers a sequence and its children.
// Anonymous inline sequences (no IDENTIFIER) get a synthetic name from
// AutoName so the scope is still addressable via GetChildByParserRule.
func collectSequenceDecl(
	ctx context.Context[parser.ISequenceDeclarationContext],
	parentScope *symbol.Scope,
) {
	name := ""
	if id := ctx.AST.IDENTIFIER(); id != nil {
		name = id.GetText()
	}
	seqScope, err := parentScope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindSequence,
		Type: types.Sequence(),
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	if name == "" {
		seqScope.AutoName("seq_")
	}
	for _, item := range ctx.AST.AllSequenceItem() {
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			collectStageDecl(context.Child(ctx, stageDecl), seqScope)
		}
		if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
			collectSequenceDecl(context.Child(ctx, nestedSeq), seqScope)
		}
	}
}

// collectStageDecl registers a stage and any nested sequences within it.
// Anonymous stages get a synthetic name so they remain addressable.
func collectStageDecl(
	ctx context.Context[parser.IStageDeclarationContext],
	seqScope *symbol.Scope,
) {
	stageName := ""
	if id := ctx.AST.IDENTIFIER(); id != nil {
		stageName = id.GetText()
	}
	stageScope, err := seqScope.Add(ctx, symbol.Symbol{
		Name: stageName,
		Kind: symbol.KindStage,
		Type: types.Stage(),
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	if stageName == "" {
		stageScope.AutoName("stage_")
	}
	stageBody := ctx.AST.StageBody()
	if stageBody == nil {
		return
	}
	for _, item := range stageBody.AllStageItem() {
		if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
			collectSequenceDecl(context.Child(ctx, nestedSeq), stageScope)
		}
	}
}

// Analyze performs semantic analysis on a sequence declaration.
// This is called during the second pass after all declarations have been collected.
// For anonymous inline sequences, the scope is resolved by parser rule rather
// than by name.
func Analyze(ctx context.Context[parser.ISequenceDeclarationContext]) {
	var (
		seqScope *symbol.Scope
		err      error
	)
	if id := ctx.AST.IDENTIFIER(); id != nil {
		seqScope, err = ctx.Scope.Resolve(ctx, id.GetText())
	} else {
		seqScope, err = ctx.Scope.GetChildByParserRule(ctx.AST)
	}
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	for _, item := range ctx.AST.AllSequenceItem() {
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			analyzeStage(context.Child(ctx, stageDecl).WithScope(seqScope))
		}
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			flow.Analyze(context.Child(ctx, flowStmt).WithScope(seqScope))
		}
		if single := item.SingleInvocation(); single != nil {
			analyzeSingleInvocation(context.Child(ctx, single).WithScope(seqScope))
		}
		if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
			Analyze(context.Child(ctx, nestedSeq).WithScope(seqScope))
		}
	}
}

// collectTopLevelStage registers a top-level stage as a sequence in the symbol
// table (since the compiler wraps it in a single-step sequence for activation).
func collectTopLevelStage(
	ctx context.Context[parser.IStageDeclarationContext],
	parentScope *symbol.Scope,
) {
	id := ctx.AST.IDENTIFIER()
	if id == nil {
		return
	}
	name := id.GetText()
	if _, err := parentScope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindSequence,
		Type: types.Sequence(),
		AST:  ctx.AST,
	}); err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
	}
}

// AnalyzeTopLevelStage performs semantic analysis on a top-level stage declaration.
func AnalyzeTopLevelStage(ctx context.Context[parser.IStageDeclarationContext]) {
	analyzeStage(ctx)
}

func analyzeStage(ctx context.Context[parser.IStageDeclarationContext]) {
	stageBody := ctx.AST.StageBody()
	if stageBody == nil {
		return
	}
	// Resolve the stage's own scope so nested sequences see their registered
	// child scope. Top-level stages register at the root scope, so fall back
	// to ctx.Scope when no child match is found.
	stageScope := ctx.Scope
	if scope, err := ctx.Scope.GetChildByParserRule(ctx.AST); err == nil {
		stageScope = scope
	}
	for _, item := range stageBody.AllStageItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			flow.Analyze(context.Child(ctx, flowStmt))
		}
		if single := item.SingleInvocation(); single != nil {
			analyzeSingleInvocation(context.Child(ctx, single))
		}
		if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
			Analyze(context.Child(ctx, nestedSeq).WithScope(stageScope))
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
