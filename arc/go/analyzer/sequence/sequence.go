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
	switch sym.VarKind {
	case symbol.VarKindLiteral, symbol.VarKindChannelRead:
		statement.AnalyzeAssignment(context.Child(ctx, assign))
		if expr := assign.Expression(); expr != nil {
			flow.AnalyzeSingleExpression(context.Child(ctx, expr))
		}
	case symbol.VarKindChannelReadWrite:
		analyzeChannelReadWriteRebind(ctx, assign, sym)
	default:
		rejectReactiveAssignment(ctx.Diagnostics, assign)
	}
}

// analyzeChannelReadWriteRebind checks that the channel read/write variable can be rebound to the channel named by assign's
// right-hand side: the RHS must reference a channel whose value type matches the channel read/write variable.
func analyzeChannelReadWriteRebind[T antlr.ParserRuleContext](
	ctx context.Context[T],
	assign parser.IAssignmentContext,
	channelReadWrite *symbol.Symbol,
) {
	name := assign.IDENTIFIER().GetText()
	target, ok := channelRebindTarget(ctx, assign.Expression())
	if !ok {
		ctx.Diagnostics.Add(diagnostics.Errorf(assign,
			"cannot rebind channel read/write variable %s; the right-hand side must be a channel", name))
		return
	}
	if !types.Equal(channelReadWrite.Type.Unwrap(), target.Type.Unwrap()) {
		ctx.Diagnostics.Add(diagnostics.Errorf(assign,
			"cannot rebind channel read/write variable %s of type %s to a channel of type %s",
			name, channelReadWrite.Type.Unwrap(), target.Type.Unwrap()))
	}
}

// channelRebindTarget resolves expr to the global channel a channel read/write rebind targets,
// reporting ok=false when expr is not a bare reference to a channel.
func channelRebindTarget[T antlr.ParserRuleContext](
	ctx context.Context[T],
	expr parser.IExpressionContext,
) (*symbol.Symbol, bool) {
	if expr == nil {
		return nil, false
	}
	primary := parser.GetPrimaryExpression(expr)
	if primary == nil || primary.IDENTIFIER() == nil {
		return nil, false
	}
	sym, err := ctx.Resolve(primary.IDENTIFIER().GetText())
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

// scopeBodyItem is a body line common to sequences and stages.
type scopeBodyItem interface {
	VariableDeclaration() parser.IVariableDeclarationContext
	Assignment() parser.IAssignmentContext
	FlowStatement() parser.IFlowStatementContext
	SingleInvocation() parser.ISingleInvocationContext
	SequenceDeclaration() parser.ISequenceDeclarationContext
}

// analyzeScopeBodyItem analyzes one body line in the construct's own ctx.Scope.
func analyzeScopeBodyItem[T antlr.ParserRuleContext](ctx context.Context[T], item scopeBodyItem) {
	if varDecl := item.VariableDeclaration(); varDecl != nil {
		statement.AnalyzeVariableDeclaration(context.Child(ctx, varDecl))
	}
	if assign := item.Assignment(); assign != nil {
		analyzeReactiveAssignment(ctx, assign)
	}
	if flowStmt := item.FlowStatement(); flowStmt != nil {
		flow.Analyze(context.Child(ctx, flowStmt))
	}
	if single := item.SingleInvocation(); single != nil {
		analyzeSingleInvocation(context.Child(ctx, single))
	}
	if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
		Analyze(context.Child(ctx, nestedSeq))
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
			analyzeStage(context.Child(ctx, stageDecl))
			continue
		}
		analyzeScopeBodyItem(ctx, item)
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
	// Top-level stages have no owned child scope; keep ctx.Scope when absent.
	if stageScope, err := context.ResolveOwnScope(ctx); err == nil {
		ctx = ctx.WithScope(stageScope)
	}
	for _, item := range stageBody.AllStageItem() {
		analyzeScopeBodyItem(ctx, item)
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
