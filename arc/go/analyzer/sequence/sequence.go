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
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// RejectReactiveAssignment reports that '=' cannot write to name in a reactive scope.
// A flow must be used instead.
func RejectReactiveAssignment(
	d *diagnostics.Diagnostics,
	assign parser.IAssignmentContext,
) {
	name := assign.IDENTIFIER().GetText()
	d.Add(
		diagnostics.Errorf(
			assign,
			"use '<value> -> %s' to write to '%s'",
			name,
			name,
		),
	)
}

// analyzeReactiveAssignment type-checks an '=' reassignment by the target's kind.
func analyzeReactiveAssignment[T antlr.ParserRuleContext](
	ctx context.Context[T],
	assign parser.IAssignmentContext,
) {
	name := assign.IDENTIFIER().GetText()
	sym, err := ctx.Resolve(name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, assign))
		return
	}
	if assign.CompoundOp() != nil || assign.IndexOrSlice() != nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(assign,
			"compound and indexed assignment to a variable are not yet supported"))
		return
	}
	switch {
	case sym.IsChannelReadWrite():
		target, ok := channelRebindTarget(ctx, assign.Expression())
		if !ok {
			RejectReactiveAssignment(ctx.Diagnostics, assign)
			return
		}
		analyzeRebind(ctx.Child(assign), sym, target)
	case sym.IsValueVariable():
		statement.AnalyzeAssignment(ctx.Child(assign))
		if expr := assign.Expression(); expr != nil &&
			statement.CastConstValue(ctx, expr, sym.Type) == nil {
			flow.AnalyzeSingleExpression(ctx.Child(expr))
			flow.BindOutput(ctx, expr, sym.Type.UnwrapChan())
		}
	default:
		RejectReactiveAssignment(ctx.Diagnostics, assign)
	}
}

// analyzeRebind re-points a channel alias at the channel rhs, recording rhs for read
// and write routing. It rejects a top-level alias and a channel of another type.
func analyzeRebind(
	ctx context.Context[parser.IAssignmentContext],
	alias, rhs *symbol.Symbol,
) {
	if ctx.RejectTopLevel(alias, alias.Name) {
		return
	}
	if !types.Equal(alias.Type.Unwrap(), rhs.Type.Unwrap()) {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"type mismatch: cannot rebind '%s' (%s) to '%s' (%s)",
			alias.Name, alias.Type, rhs.Name, rhs.Type))
		return
	}
	alias.Reassigned = true
	if alias.Channels.Write == nil {
		alias.Channels = types.NewChannels()
	}
	key := uint32(rhs.ID)
	if rhs.SourceID != nil {
		key = uint32(*rhs.SourceID)
	}
	alias.Channels.Read[key] = rhs.Name
	alias.Channels.Write[key] = rhs.Name
}

// channelRebindTarget resolves expr to the global channel a channel read/write rebind
// targets, reporting ok=false when expr is not a bare reference to a channel.
func channelRebindTarget[T antlr.ParserRuleContext](
	ctx context.Context[T],
	expr parser.IExpressionContext,
) (*symbol.Symbol, bool) {
	if expr == nil {
		return nil, false
	}
	head, tail := parser.ExpressionNameParts(expr)
	if head == "" {
		return nil, false
	}
	sym, err := ctx.ResolveQualified(head, tail)
	if err != nil {
		return nil, false
	}
	if sym.Kind != symbol.KindChannel || sym.Type.Kind != types.KindChan {
		return nil, false
	}
	return sym, true
}

// CollectDeclarations registers all sequences and their children in the symbol table.
// This is called during the first pass of AnalyzeProgram to establish scopes before
// analyzing function bodies that may reference sequences or stages.
func CollectDeclarations(ctx context.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			collectSequenceDecl(ctx.Child(seqDecl), ctx.Scope)
		}
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			collectTopLevelStage(ctx.Child(stageDecl), ctx.Scope)
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
				if synth := registerInlineBody(
					ctx,
					enclosing,
					decl,
					&counter,
				); synth != nil {
					walk(synth, decl)
				}
				return
			}
		}
		next := enclosing
		switch node.(type) {
		case parser.ISequenceDeclarationContext, parser.IStageDeclarationContext:
			if c, err := enclosing.GetChildByParserRule(
				node.(antlr.ParserRuleContext),
			); err == nil {
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
					collectSequenceDecl(ctx.Child(nested), synth)
				}
			}
		}
	case parser.ISequenceDeclarationContext:
		for _, item := range d.AllSequenceItem() {
			if stage := item.StageDeclaration(); stage != nil {
				collectStageDecl(ctx.Child(stage), synth)
			}
			if nested := item.SequenceDeclaration(); nested != nil {
				collectSequenceDecl(ctx.Child(nested), synth)
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
					AnalyzeTopLevelStage(ctx.Child(decl).WithScope(parent))
				case parser.ISequenceDeclarationContext:
					Analyze(ctx.Child(decl).WithScope(parent))
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
			collectStageDecl(ctx.Child(stageDecl), seqScope)
		}
		if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
			collectSequenceDecl(ctx.Child(nestedSeq), seqScope)
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
			collectSequenceDecl(ctx.Child(nestedSeq), stageScope)
		}
	}
}

// scopeBodyItem is a body line common to sequences and stages.
type scopeBodyItem interface {
	VariableDeclaration() parser.IVariableDeclarationContext
	Assignment() parser.IAssignmentContext
	FlowStatement() parser.IFlowStatementContext
	SingleInvocation() parser.ISingleInvocationContext
	SequenceDeclaration() parser.ISequenceDeclarationContext
}

// analyzeScopeBodyItem analyzes one body line in the construct's own ctx.Scope.
func analyzeScopeBodyItem[T antlr.ParserRuleContext](
	ctx context.Context[T],
	item scopeBodyItem,
) {
	if varDecl := item.VariableDeclaration(); varDecl != nil {
		statement.AnalyzeVariableDeclaration(ctx.Child(varDecl))
	}
	if assign := item.Assignment(); assign != nil {
		analyzeReactiveAssignment(ctx, assign)
	}
	if flowStmt := item.FlowStatement(); flowStmt != nil {
		flow.Analyze(ctx.Child(flowStmt))
	}
	if single := item.SingleInvocation(); single != nil {
		analyzeSingleInvocation(ctx.Child(single))
	}
	if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
		Analyze(ctx.Child(nestedSeq))
	}
}

// warnLateDeclarations flags each declaration that follows a step in the body.
func warnLateDeclarations[T antlr.ParserRuleContext, I interface {
	VariableDeclaration() parser.IVariableDeclarationContext
}](ctx context.Context[T], items []I) {
	sawStep := false
	for _, item := range items {
		decl := item.VariableDeclaration()
		if decl == nil {
			sawStep = true
			continue
		}
		if sawStep {
			ctx.Diagnostics.Add(codes.DeclarationAfterStep.At(decl))
		}
	}
}

// Analyze performs second-pass semantic analysis on a sequence declaration.
func Analyze(ctx context.Context[parser.ISequenceDeclarationContext]) {
	seqScope, err := context.ResolveOwnScope(ctx)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	ctx = ctx.WithScope(seqScope)
	for _, item := range ctx.AST.AllSequenceItem() {
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			analyzeStage(ctx.Child(stageDecl))
			continue
		}
		analyzeScopeBodyItem(ctx, item)
	}
	warnLateDeclarations(ctx, ctx.AST.AllSequenceItem())
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
			collectSequenceDecl(ctx.Child(nestedSeq), stageScope)
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
	// Top-level stages have no owned child scope; keep ctx.Scope when absent.
	if stageScope, err := context.ResolveOwnScope(ctx); err == nil {
		ctx = ctx.WithScope(stageScope)
	}
	for _, item := range stageBody.AllStageItem() {
		analyzeScopeBodyItem(ctx, item)
	}
	warnLateDeclarations(ctx, stageBody.AllStageItem())
}

func analyzeSingleInvocation(ctx context.Context[parser.ISingleInvocationContext]) {
	if fn := ctx.AST.Function(); fn != nil {
		flow.AnalyzeSingleFunction(ctx.Child(fn))
		return
	}
	if expr := ctx.AST.Expression(); expr != nil {
		flow.AnalyzeSingleExpression(ctx.Child(expr))
	}
}
