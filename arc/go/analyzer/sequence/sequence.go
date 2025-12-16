// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package sequence implements semantic analysis for Arc sequences and stages.
//
// Sequences are state machines with ordered stages. Each stage can contain reactive
// flows (-> operator) that run continuously while the stage is active, and one-shot
// transitions (=> operator) that fire when conditions become true.
//
// The analyzer validates:
//   - Sequence and stage declarations are properly formed
//   - Stage names are unique within their sequence
//   - Stage names don't conflict with sequence names
//   - Transition targets resolve to valid stages or sequences
//   - The `next` keyword resolves to the next stage in definition order
//   - Reactive flows and transition conditions are type-correct
//
// Safety warnings are emitted when:
//   - Non-abort transitions appear before abort conditions (potential safety issue)
package sequence

import (
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// CollectDeclarations registers all sequences and their stages in the symbol table.
// This is called during the first pass of AnalyzeProgram to establish scopes before
// analyzing function bodies that may reference sequences or stages.
func CollectDeclarations(ctx context.Context[parser.IProgramContext]) bool {
	// First pass: collect all sequence names to establish their scopes
	for _, item := range ctx.AST.AllTopLevelItem() {
		if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			if !collectSequenceName(context.Child(ctx, seqDecl)) {
				return false
			}
		}
	}

	// Second pass: collect all stages (now we can check for name collisions with all sequences)
	for _, item := range ctx.AST.AllTopLevelItem() {
		if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			if !collectSequenceStages(context.Child(ctx, seqDecl)) {
				return false
			}
		}
	}
	return true
}

// collectSequenceName registers a sequence in the symbol table (first pass).
func collectSequenceName(ctx context.Context[parser.ISequenceDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	if existing, err := ctx.Scope.Resolve(ctx, name); err == nil && existing.AST != nil {
		tok := existing.AST.GetStart()
		ctx.Diagnostics.AddError(
			errors.Newf("sequence name '%s' conflicts with existing symbol at line %d, col %d",
				name, tok.GetLine(), tok.GetColumn()),
			ctx.AST,
		)
		return false
	}
	if _, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindSequence,
		Type: types.Sequence(),
		AST:  ctx.AST,
	}); err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return true
}

func collectSequenceStages(ctx context.Context[parser.ISequenceDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	seqScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	stages := ctx.AST.AllStageDeclaration()
	for _, stageDecl := range stages {
		if !collectStage(context.Child(ctx, stageDecl).WithScope(seqScope), seqScope) {
			return false
		}
	}
	return true
}

func collectStage(
	ctx context.Context[parser.IStageDeclarationContext],
	seqScope *symbol.Scope,
) bool {
	stageName := ctx.AST.IDENTIFIER().GetText()
	if existing, err := ctx.Scope.Root().Resolve(ctx, stageName); err == nil {
		if existing.Kind == symbol.KindSequence {
			ctx.Diagnostics.AddError(
				errors.Newf("stage name '%s' conflicts with sequence name", stageName),
				ctx.AST,
			)
			return false
		}
	}
	if _, err := seqScope.Add(ctx, symbol.Symbol{
		Name: stageName,
		Kind: symbol.KindStage,
		Type: types.Stage(),
		AST:  ctx.AST,
	}); err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return true
}

// Analyze performs semantic analysis on a sequence declaration.
// This is called during the second pass after all declarations have been collected.
func Analyze(ctx context.Context[parser.ISequenceDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	seqScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	stages := ctx.AST.AllStageDeclaration()
	for _, stageDecl := range stages {
		stageCtx := context.Child(ctx, stageDecl).WithScope(seqScope)
		if !analyzeStage(stageCtx, seqScope) {
			return false
		}
	}
	return true
}

// analyzeStage performs semantic analysis on a stage declaration.
// With unified flow statements, stages now just contain flows (no special transitions).
func analyzeStage(
	ctx context.Context[parser.IStageDeclarationContext],
	seqScope *symbol.Scope,
) bool {
	stageBody := ctx.AST.StageBody()
	if stageBody == nil {
		return true
	}
	for _, item := range stageBody.AllStageItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			if !flow.Analyze(context.Child(ctx, flowStmt)) {
				return false
			}
		}
	}
	return true
}
