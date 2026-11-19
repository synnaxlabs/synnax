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
	"github.com/synnaxlabs/arc/analyzer/function"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// RejectReactiveAssignment reports that '=' cannot write to name in a reactive scope.
// A flow must be used instead.
func RejectReactiveAssignment(
	d *diagnostics.Diagnostics,
	assign parser.IAssignmentContext,
) {
	name := assign.IDENTIFIER().GetText()
	d.Add(
		codes.AssignmentTarget.Atf(
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
		ctx.Diagnostics.Add(codes.AssignmentTarget.Atf(assign,
			"compound and indexed assignment to a variable are not yet supported"))
		return
	}
	switch {
	case sym.IsSequenceInput() && sym.Type.Kind != types.KindChan:
		ctx.Diagnostics.Add(codes.AssignmentTarget.Atf(assign,
			"cannot reassign parameter '%s'", name))
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
		ctx.Diagnostics.Add(codes.TypeMismatch.Atf(ctx.AST,
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
			var decl declaration
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
		var decl antlr.ParserRuleContext
		switch n := node.(type) {
		case parser.ISequenceDeclarationContext:
			decl = n
		case parser.IStageDeclarationContext:
			decl = n
		}
		if decl != nil {
			if c, err := enclosing.GetChildByParserRule(decl); err == nil {
				next = c
			}
		}
		for i := range node.GetChildCount() {
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
	decl declaration,
	counter *int,
) *symbol.Symbol {
	collectParameters(ctx, decl, "", "", false)
	if decl.IDENTIFIER() != nil {
		ctx.Diagnostics.Add(codes.NamedInlineBody.At(decl))
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
	collectBodies(ctx, decl.Body(), synth)
	return synth
}

// collectBodies registers each stage and sequence that body declares under scope.
func collectBodies[T antlr.ParserRuleContext](
	ctx context.Context[T],
	body parser.IBodyContext,
	scope *symbol.Symbol,
) {
	for _, item := range parser.BodyItems(body) {
		if stage := item.StageDeclaration(); stage != nil {
			collectStageDecl(ctx.Child(stage), scope)
		}
		if nested := item.SequenceDeclaration(); nested != nil {
			collectSequenceDecl(ctx.Child(nested), scope)
		}
	}
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

// declaration is a sequence or a stage declaration.
type declaration interface {
	antlr.ParserRuleContext
	IDENTIFIER() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	InputList() parser.IInputListContext
	Body() parser.IBodyContext
}

// emptyParameters returns the diagnostic for the empty parentheses of decl.
func emptyParameters(decl declaration) diagnostics.Diagnostic {
	open, closed := decl.LPAREN().GetSymbol(), decl.RPAREN().GetSymbol()
	start := protocol.Position{
		Line:      uint32(open.GetLine() - 1),
		Character: uint32(open.GetColumn()),
	}
	end := protocol.Position{
		Line:      uint32(closed.GetLine() - 1),
		Character: uint32(closed.GetColumn() + 1),
	}
	// The fix also removes the blanks between the name and the parentheses.
	from := start
	for input, i := open.GetInputStream(), open.GetStart()-1; i >= 0; i-- {
		if c := input.GetText(i, i); c != " " && c != "\t" {
			break
		}
		from.Character--
	}
	return codes.BodyDeclaration.Atf(decl, "a parameter list cannot be empty").
		WithRange(start, end).
		WithNote("add a parameter or remove the parentheses").
		WithEdits(
			"Remove the empty parentheses",
			protocol.TextEdit{Range: protocol.Range{Start: from, End: end}},
		)
}

// collectParameters returns the parameters of a named top-level sequence or stage.
func collectParameters[T antlr.ParserRuleContext](
	ctx context.Context[T],
	decl declaration,
	kind, name string,
	topLevel bool,
) types.Params {
	if decl == nil {
		return nil
	}
	inputList := decl.InputList()
	if inputList == nil {
		if decl.LPAREN() != nil {
			ctx.Diagnostics.Add(emptyParameters(decl))
		}
		return nil
	}
	if !topLevel {
		ctx.Diagnostics.Add(codes.BodyDeclaration.Atf(inputList,
			"only a top-level stage or sequence can have parameters"))
		return nil
	}
	if name == "" {
		ctx.Diagnostics.Add(codes.BodyDeclaration.Atf(inputList,
			"a %s with parameters needs a name", kind))
		return nil
	}
	for _, input := range inputList.AllInput() {
		t, lit := input.Type_(), input.Literal()
		if lit == nil || t == nil || t.ChannelType() == nil {
			continue
		}
		ctx.Diagnostics.Add(codes.BodyDeclaration.Atf(
			lit,
			"channel parameter '%s' cannot have a default",
			input.IDENTIFIER().GetText(),
		))
	}
	var inputs types.Params
	function.CollectInputs(ctx, inputList, &inputs)
	return inputs
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
	t := types.Sequence()
	t.Inputs = collectParameters(
		ctx, ctx.AST, "sequence", name, parentScope == ctx.Scope,
	)
	seqScope, err := parentScope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindSequence,
		Type: t,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	if name == "" {
		seqScope.AutoName("seq_")
	}
	collectBodies(ctx, ctx.AST.Body(), seqScope)
}

// collectStageDecl registers a stage and the stages and sequences inside it.
// Anonymous stages get a synthetic name so they remain addressable.
func collectStageDecl(
	ctx context.Context[parser.IStageDeclarationContext],
	seqScope *symbol.Symbol,
) {
	stageName := ""
	if id := ctx.AST.IDENTIFIER(); id != nil {
		stageName = id.GetText()
	}
	collectParameters(ctx, ctx.AST, "stage", stageName, false)
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
	collectBodies(ctx, ctx.AST.Body(), stageScope)
}

// analyzeBody analyzes the items of a stage or sequence body in ctx.Scope, and flags
// each declaration that follows a step.
func analyzeBody[T antlr.ParserRuleContext](
	ctx context.Context[T],
	body parser.IBodyContext,
) {
	sawStep := false
	for _, item := range parser.BodyItems(body) {
		varDecl := item.VariableDeclaration()
		if varDecl == nil {
			sawStep = true
		} else {
			statement.AnalyzeVariableDeclaration(ctx.Child(varDecl))
			if sawStep {
				ctx.Diagnostics.Add(codes.DeclarationAfterStep.At(varDecl))
			}
		}
		if stage := item.StageDeclaration(); stage != nil {
			analyzeStage(ctx.Child(stage))
		}
		if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
			Analyze(ctx.Child(nestedSeq))
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
	function.AddInputsToScope(ctx, ctx.AST.InputList(), seqScope)
	analyzeBody(ctx, ctx.AST.Body())
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
	t := types.Sequence()
	t.Inputs = collectParameters(ctx, ctx.AST, "stage", name, true)
	stageScope, err := parentScope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindSequence,
		Type: t,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	if name == "" {
		stageScope.AutoName("stage_")
	}
	collectBodies(ctx, ctx.AST.Body(), stageScope)
}

// AnalyzeTopLevelStage performs semantic analysis on a top-level stage declaration.
func AnalyzeTopLevelStage(ctx context.Context[parser.IStageDeclarationContext]) {
	analyzeStage(ctx)
}

func analyzeStage(ctx context.Context[parser.IStageDeclarationContext]) {
	// Top-level stages have no owned child scope; keep ctx.Scope when absent.
	if stageScope, err := context.ResolveOwnScope(ctx); err == nil {
		ctx = ctx.WithScope(stageScope)
		function.AddInputsToScope(ctx, ctx.AST.InputList(), stageScope)
	}
	analyzeBody(ctx, ctx.AST.Body())
}

func analyzeSingleInvocation(ctx context.Context[parser.ISingleInvocationContext]) {
	if fn := ctx.AST.Function(); fn != nil {
		flow.AnalyzeSingleFunction(ctx.Child(fn))
		return
	}
	if expr := ctx.AST.Expression(); expr != nil {
		if flow.AnalyzeBodyName(ctx.Child(expr)) {
			return
		}
		flow.AnalyzeSingleExpression(ctx.Child(expr))
	}
}
