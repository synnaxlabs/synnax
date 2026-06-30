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
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// rejectReactiveAssignment reports that '=' cannot mutate a variable in a reactive scope; a flow must be used instead.
func rejectReactiveAssignment(d *diagnostics.Diagnostics, assign parser.IAssignmentContext) {
	name := assign.IDENTIFIER().GetText()
	d.Add(diagnostics.Errorf(assign, "cannot use '=' here; write to '%s' with a flow: <value> -> %s", name, name))
}

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
	desugarInlineDecls(ctx)
}

// desugarInlineDecls registers a synth scope under the lexically enclosing
// scope for each anonymous inline stage/sequence body used as a flow target.
func desugarInlineDecls(ctx context.Context[parser.IProgramContext]) {
	counter := 0
	var walk func(enclosing *symbol.Symbol, node antlr.Tree)
	walk = func(enclosing *symbol.Symbol, node antlr.Tree) {
		if fn, ok := node.(parser.IFlowNodeContext); ok {
			var decl antlr.ParserRuleContext
			if s := fn.StageDeclaration(); s != nil {
				decl = s
			} else if s := fn.SequenceDeclaration(); s != nil {
				decl = s
			}
			if decl != nil {
				if synth := registerInlineBody(ctx, enclosing, decl, &counter); synth != nil {
					walk(synth, decl)
				}
				return
			}
		}
		next := enclosing
		switch node.(type) {
		case parser.ISequenceDeclarationContext, parser.IStageDeclarationContext:
			if c, err := enclosing.GetChildByParserRule(node.(antlr.ParserRuleContext)); err == nil {
				next = c
			}
		}
		for i := 0; i < node.GetChildCount(); i++ {
			walk(next, node.GetChild(i))
		}
	}
	walk(ctx.Scope, ctx.AST)
}

// registerInlineBody registers a synth scope under parentScope for an inline
// routing case body; named bodies emit a diagnostic and return nil.
func registerInlineBody(
	ctx context.Context[parser.IProgramContext],
	parentScope *symbol.Symbol,
	decl antlr.ParserRuleContext,
	counter *int,
) *symbol.Symbol {
	var (
		id   antlr.TerminalNode
		kind string
	)
	switch d := decl.(type) {
	case parser.IStageDeclarationContext:
		id, kind = d.IDENTIFIER(), "stages"
	case parser.ISequenceDeclarationContext:
		id, kind = d.IDENTIFIER(), "sequences"
	}
	if id != nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(decl,
			"inline routing case body %s must be anonymous; remove name %q",
			kind, id.GetText()))
		return nil
	}
	synth, err := parentScope.Add(ctx, symbol.Symbol{
		Name: ir.InlinePrefix + strconv.Itoa(*counter),
		Kind: symbol.KindSequence,
		Type: types.Sequence(),
		AST:  decl,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, decl))
		return nil
	}
	*counter++
	switch d := decl.(type) {
	case parser.IStageDeclarationContext:
		if body := d.StageBody(); body != nil {
			for _, item := range body.AllStageItem() {
				if nested := item.SequenceDeclaration(); nested != nil {
					collectSequenceDecl(context.Child(ctx, nested), synth)
				}
			}
		}
	case parser.ISequenceDeclarationContext:
		for _, item := range d.AllSequenceItem() {
			if stage := item.StageDeclaration(); stage != nil {
				collectStageDecl(context.Child(ctx, stage), synth)
			}
			if nested := item.SequenceDeclaration(); nested != nil {
				collectSequenceDecl(context.Child(ctx, nested), synth)
			}
		}
	}
	return synth
}

// AnalyzeSynthInlines analyzes every synth inline scope in the tree; must run
// after the top-level analyze loop so bodies can resolve forward references.
func AnalyzeSynthInlines(ctx context.Context[parser.IProgramContext]) {
	var walk func(parent *symbol.Symbol)
	walk = func(parent *symbol.Symbol) {
		for _, child := range parent.Children() {
			if strings.HasPrefix(child.Name, ir.InlinePrefix) {
				switch decl := child.AST.(type) {
				case parser.IStageDeclarationContext:
					AnalyzeTopLevelStage(context.Child(ctx, decl).WithScope(parent))
				case parser.ISequenceDeclarationContext:
					Analyze(context.Child(ctx, decl).WithScope(parent))
				}
			}
			walk(child)
		}
	}
	walk(ctx.Scope)
}

// collectSequenceDecl recursively registers a sequence and its children.
// Anonymous inline sequences (no IDENTIFIER) get a synthetic name from
// AutoName so the scope is still addressable via GetChildByParserRule.
func collectSequenceDecl(
	ctx context.Context[parser.ISequenceDeclarationContext],
	parentScope *symbol.Symbol,
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
	seqScope *symbol.Symbol,
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
		seqScope *symbol.Symbol
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
		if varDecl := item.VariableDeclaration(); varDecl != nil {
			statement.AnalyzeVariableDeclaration(context.Child(ctx, varDecl).WithScope(seqScope))
		}
		if assign := item.Assignment(); assign != nil {
			rejectReactiveAssignment(ctx.Diagnostics, assign)
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
// Anonymous top-level stages get a synthetic name so the scope remains
// addressable via GetChildByParserRule.
func collectTopLevelStage(
	ctx context.Context[parser.IStageDeclarationContext],
	parentScope *symbol.Symbol,
) {
	name := ""
	if id := ctx.AST.IDENTIFIER(); id != nil {
		name = id.GetText()
	}
	stageScope, err := parentScope.Add(ctx, symbol.Symbol{
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
		if varDecl := item.VariableDeclaration(); varDecl != nil {
			statement.AnalyzeVariableDeclaration(context.Child(ctx, varDecl).WithScope(stageScope))
		}
		if assign := item.Assignment(); assign != nil {
			rejectReactiveAssignment(ctx.Diagnostics, assign)
		}
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			flow.Analyze(context.Child(ctx, flowStmt).WithScope(stageScope))
		}
		if single := item.SingleInvocation(); single != nil {
			analyzeSingleInvocation(context.Child(ctx, single).WithScope(stageScope))
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
