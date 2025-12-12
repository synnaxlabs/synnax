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
	goctx "context"
	"strings"

	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/diagnostics"
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

	// Check for name collision with existing symbols
	if existing, err := ctx.Scope.Resolve(ctx, name); err == nil && existing.AST != nil {
		tok := existing.AST.GetStart()
		ctx.Diagnostics.AddError(
			errors.Newf("sequence name '%s' conflicts with existing symbol at line %d, col %d",
				name, tok.GetLine(), tok.GetColumn()),
			ctx.AST,
		)
		return false
	}

	// Add sequence to root scope
	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindSequence,
		Type: types.Type{Kind: types.KindInvalid}, // Sequences don't have a traditional type
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	return true
}

// collectSequenceStages registers all stages for a sequence (second pass).
func collectSequenceStages(ctx context.Context[parser.ISequenceDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()

	// Resolve the sequence scope we created in the first pass
	seqScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	// Collect all stages in definition order
	stages := ctx.AST.AllStageDeclaration()
	for _, stageDecl := range stages {
		if !collectStage(context.Child(ctx, stageDecl).WithScope(seqScope), seqScope, name) {
			return false
		}
	}

	return true
}

// collectStage registers a stage in the sequence scope.
func collectStage(
	ctx context.Context[parser.IStageDeclarationContext],
	seqScope *symbol.Scope,
	seqName string,
) bool {
	stageName := ctx.AST.IDENTIFIER().GetText()

	// Check that stage name doesn't conflict with any sequence name
	if existing, err := ctx.Scope.Root().Resolve(ctx, stageName); err == nil {
		if existing.Kind == symbol.KindSequence {
			ctx.Diagnostics.AddError(
				errors.Newf("stage name '%s' conflicts with sequence name", stageName),
				ctx.AST,
			)
			return false
		}
	}

	// Add stage to sequence scope
	_, err := seqScope.Add(ctx, symbol.Symbol{
		Name: stageName,
		Kind: symbol.KindStage,
		Type: types.Type{Kind: types.KindInvalid},
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	return true
}

// Analyze performs semantic analysis on a sequence declaration.
// This is called during the second pass after all declarations have been collected.
func Analyze(ctx context.Context[parser.ISequenceDeclarationContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()

	// Resolve the sequence scope
	seqScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	// Analyze each stage
	stages := ctx.AST.AllStageDeclaration()
	for i, stageDecl := range stages {
		stageCtx := context.Child(ctx, stageDecl).WithScope(seqScope)
		if !analyzeStage(stageCtx, seqScope, i, len(stages)) {
			return false
		}
	}

	return true
}

// analyzeStage performs semantic analysis on a stage declaration.
func analyzeStage(
	ctx context.Context[parser.IStageDeclarationContext],
	seqScope *symbol.Scope,
	stageIndex int,
	totalStages int,
) bool {
	stageBody := ctx.AST.StageBody()
	if stageBody == nil {
		return true // Empty stage body is valid
	}

	// Track whether we've seen a transition to an abort target
	// to warn about potentially unsafe ordering
	seenAbort := false
	seenNonAbortTransition := false

	for _, item := range stageBody.AllStageItem() {
		if imperative := item.ImperativeTransition(); imperative != nil {
			if !analyzeImperativeTransition(context.Child(ctx, imperative), seqScope, stageIndex, totalStages) {
				return false
			}
		} else if transition := item.TransitionStatement(); transition != nil {
			target := transition.TransitionTarget()
			isAbort := isAbortTarget(target)

			// Warn if we see a non-abort transition before an abort transition
			if seenNonAbortTransition && isAbort && !seenAbort {
				ctx.Diagnostics.AddWarning(
					errors.Newf("abort transition appears after non-abort transition; "+
						"consider reordering to ensure abort conditions take priority"),
					transition,
				)
			}

			if isAbort {
				seenAbort = true
			} else if !seenAbort {
				seenNonAbortTransition = true
			}

			if !analyzeTransition(context.Child(ctx, transition), seqScope, stageIndex, totalStages) {
				return false
			}
		} else if stageFlow := item.StageFlow(); stageFlow != nil {
			if !analyzeStageFlow(context.Child(ctx, stageFlow)) {
				return false
			}
		}
	}

	return true
}

// isAbortTarget checks if a transition target is likely an abort target.
func isAbortTarget(target parser.ITransitionTargetContext) bool {
	if target == nil {
		return false
	}
	if id := target.IDENTIFIER(); id != nil {
		return strings.Contains(strings.ToLower(id.GetText()), "abort")
	}
	return false
}

// analyzeImperativeTransition analyzes an imperative block with match routing.
func analyzeImperativeTransition(
	ctx context.Context[parser.IImperativeTransitionContext],
	seqScope *symbol.Scope,
	stageIndex int,
	totalStages int,
) bool {
	// The imperative block is analyzed like a function body
	// For now, we'll defer full block analysis to the statement analyzer
	// TODO: Analyze the imperative block body

	// Analyze the match block targets
	matchBlock := ctx.AST.MatchBlock()
	if matchBlock == nil {
		ctx.Diagnostics.AddError(
			errors.New("imperative transition requires a match block"),
			ctx.AST,
		)
		return false
	}

	for _, entry := range matchBlock.AllMatchEntry() {
		target := entry.TransitionTarget()
		if target == nil {
			continue
		}
		if !validateTransitionTargetFromImperative(ctx, target, seqScope, stageIndex, totalStages) {
			return false
		}
	}

	return true
}

// analyzeTransition analyzes a transition statement.
func analyzeTransition(
	ctx context.Context[parser.ITransitionStatementContext],
	seqScope *symbol.Scope,
	stageIndex int,
	totalStages int,
) bool {
	// Analyze the condition (left side of =>)
	// The condition can be: timerBuiltin, logBuiltin, function, or expression
	if timer := ctx.AST.TimerBuiltin(); timer != nil {
		if !analyzeTimerBuiltin(context.Child(ctx, timer)) {
			return false
		}
	} else if log := ctx.AST.LogBuiltin(); log != nil {
		// log{} is always valid - no type checking needed
	} else if fn := ctx.AST.Function(); fn != nil {
		// Function call in transition condition - should return bool
		// Defer to flow analyzer for function validation
		// TODO: Validate function returns bool
	} else if expr := ctx.AST.Expression(); expr != nil {
		// Expression in transition condition - should be boolean
		// TODO: Type check expression is boolean
	}

	// Analyze the target (right side of =>)
	target := ctx.AST.TransitionTarget()
	if target == nil {
		ctx.Diagnostics.AddError(
			errors.New("transition requires a target"),
			ctx.AST,
		)
		return false
	}

	return validateTransitionTargetFromTransition(ctx, target, seqScope, stageIndex, totalStages)
}

// validateTransitionTargetFromTransition validates a transition target from a transition statement.
func validateTransitionTargetFromTransition(
	ctx context.Context[parser.ITransitionStatementContext],
	target parser.ITransitionTargetContext,
	seqScope *symbol.Scope,
	stageIndex int,
	totalStages int,
) bool {
	return validateTransitionTargetImpl(ctx.Diagnostics, ctx.Scope, target, seqScope, stageIndex, totalStages)
}

// validateTransitionTargetFromImperative validates a transition target from an imperative transition.
func validateTransitionTargetFromImperative(
	ctx context.Context[parser.IImperativeTransitionContext],
	target parser.ITransitionTargetContext,
	seqScope *symbol.Scope,
	stageIndex int,
	totalStages int,
) bool {
	return validateTransitionTargetImpl(ctx.Diagnostics, ctx.Scope, target, seqScope, stageIndex, totalStages)
}

// validateTransitionTargetImpl is the implementation of transition target validation.
func validateTransitionTargetImpl(
	diag *diagnostics.Diagnostics,
	scope *symbol.Scope,
	target parser.ITransitionTargetContext,
	seqScope *symbol.Scope,
	stageIndex int,
	totalStages int,
) bool {
	// Case 1: `next` keyword
	if target.NEXT() != nil {
		if stageIndex >= totalStages-1 {
			diag.AddError(
				errors.New("`next` cannot be used in the last stage of a sequence"),
				target,
			)
			return false
		}
		return true
	}

	// Case 2: Match block (for pattern matching)
	if matchBlock := target.MatchBlock(); matchBlock != nil {
		for _, entry := range matchBlock.AllMatchEntry() {
			entryTarget := entry.TransitionTarget()
			if entryTarget == nil {
				continue
			}
			if !validateTransitionTargetImpl(diag, scope, entryTarget, seqScope, stageIndex, totalStages) {
				return false
			}
		}
		return true
	}

	// Case 3: Stage flow (one-shot action, stay in stage)
	if target.StageFlow() != nil {
		// Stage flows are analyzed elsewhere; just validate syntax here
		return true
	}

	// Case 4: Identifier (stage name or sequence name)
	if id := target.IDENTIFIER(); id != nil {
		targetName := id.GetText()

		// First, check if it's a stage in the current sequence
		if stageScope := seqScope.FindChildByName(targetName); stageScope != nil {
			if stageScope.Kind == symbol.KindStage {
				return true
			}
		}

		// Then, check if it's a sequence in the file
		rootScope := scope.Root()
		if targetSeq, err := rootScope.Resolve(goctx.Background(), targetName); err == nil {
			if targetSeq.Kind == symbol.KindSequence {
				return true
			}
		}

		// Unknown target
		diag.AddError(
			errors.Newf("unknown transition target '%s': not a stage or sequence", targetName),
			target,
		)
		return false
	}

	return true
}

// analyzeStageFlow analyzes a stage flow (reactive flow chain).
func analyzeStageFlow(ctx context.Context[parser.IStageFlowContext]) bool {
	// Stage flows are like regular flow statements but within a stage
	// They follow the pattern: source -> func{} -> ... -> sink
	// For now, we'll defer to the flow analyzer
	// TODO: Implement full stage flow analysis
	return true
}

// analyzeTimerBuiltin analyzes a timer builtin (wait{} or interval{}).
func analyzeTimerBuiltin(ctx context.Context[parser.ITimerBuiltinContext]) bool {
	// Timer builtins take a duration config value
	configVals := ctx.AST.ConfigValues()
	if configVals == nil {
		ctx.Diagnostics.AddError(
			errors.New("timer builtin requires a duration parameter"),
			ctx.AST,
		)
		return false
	}

	// Validate the duration parameter
	// Expected: either a literal timespan (100ms, 2s) or a config reference
	// TODO: Type check that the value is a timespan

	return true
}

// AnalyzeTopLevelTransition analyzes a top-level transition (entry point).
func AnalyzeTopLevelTransition(ctx context.Context[parser.ITopLevelTransitionContext]) bool {
	identifiers := ctx.AST.AllIDENTIFIER()
	if len(identifiers) < 2 {
		ctx.Diagnostics.AddError(
			errors.New("top-level transition requires source and target identifiers"),
			ctx.AST,
		)
		return false
	}

	// Source is typically a channel name
	sourceName := identifiers[0].GetText()
	targetName := identifiers[1].GetText()

	// Validate source is a channel (or expression - will be validated elsewhere)
	// For now, just check it resolves to something
	if _, err := ctx.Scope.Resolve(ctx, sourceName); err != nil {
		// May be a channel not yet defined - that's okay for now
		// The channel might be defined externally
	}

	// Validate target is a sequence
	targetScope, err := ctx.Scope.Resolve(ctx, targetName)
	if err != nil {
		ctx.Diagnostics.AddError(
			errors.Newf("unknown sequence '%s' in top-level transition", targetName),
			ctx.AST,
		)
		return false
	}

	if targetScope.Kind != symbol.KindSequence {
		ctx.Diagnostics.AddError(
			errors.Newf("'%s' is not a sequence (got %s)", targetName, targetScope.Kind),
			ctx.AST,
		)
		return false
	}

	return true
}
