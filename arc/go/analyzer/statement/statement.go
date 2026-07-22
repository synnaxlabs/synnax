// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package statement implements semantic analysis for Arc statements including variable
// declarations, assignments, conditionals, and channel operations.
package statement

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/analyzer/flow"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// AnalyzeBlock validates a block of statements in a new scope.
func AnalyzeBlock(ctx context.Context[parser.IBlockContext]) {
	blockScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	for _, stmt := range ctx.AST.AllStatement() {
		Analyze(context.Child(ctx, stmt).WithScope(blockScope))
	}
}

// Analyze validates a statement and dispatches to specialized handlers based on statement type.
func Analyze(ctx context.Context[parser.IStatementContext]) {
	switch {
	case ctx.AST.VariableDeclaration() != nil:
		analyzeVariableDeclaration(context.Child(ctx, ctx.AST.VariableDeclaration()))
	case ctx.AST.IfStatement() != nil:
		analyzeIfStatement(context.Child(ctx, ctx.AST.IfStatement()))
	case ctx.AST.ForStatement() != nil:
		analyzeForStatement(context.Child(ctx, ctx.AST.ForStatement()))
	case ctx.AST.BreakStatement() != nil:
		analyzeBreakStatement(context.Child(ctx, ctx.AST.BreakStatement()))
	case ctx.AST.ContinueStatement() != nil:
		analyzeContinueStatement(context.Child(ctx, ctx.AST.ContinueStatement()))
	case ctx.AST.ReturnStatement() != nil:
		analyzeReturnStatement(context.Child(ctx, ctx.AST.ReturnStatement()))
	case ctx.AST.Assignment() != nil:
		analyzeAssignment(context.Child(ctx, ctx.AST.Assignment()))
	case ctx.AST.Expression() != nil:
		expression.Analyze(context.Child(ctx, ctx.AST.Expression()))
	}
}

// AnalyzeVariableDeclaration registers and type-checks a `:=`/`$=` declaration in
// ctx.Scope, letting sequences and stages declare scoped variables.
func AnalyzeVariableDeclaration(ctx context.Context[parser.IVariableDeclarationContext]) {
	analyzeVariableDeclaration(ctx)
	inferVarKind(ctx)
}

// inferVarKind classifies the just-declared value variable from its RHS and
// records the kind on its symbol, registering a channel-read `:=` initializer's flow.
func inferVarKind(ctx context.Context[parser.IVariableDeclarationContext]) {
	var (
		ident string
		expr  parser.IExpressionContext
	)
	stateful := false
	local := ctx.AST.LocalVariable()
	switch {
	case local != nil:
		ident = local.IDENTIFIER().GetText()
		expr = local.Expression()
	case ctx.AST.StatefulVariable() != nil:
		stateful = true
		ident = ctx.AST.StatefulVariable().IDENTIFIER().GetText()
		expr = ctx.AST.StatefulVariable().Expression()
	default:
		return
	}
	sym, err := ctx.Scope.Resolve(ctx, ident)
	if err != nil {
		return
	}
	if stateful {
		if rhsTracksChannel(ctx, expr) {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
				"channels and channel-read expressions cannot be assigned to stateful variables"))
			return
		}
		// SY-4474: Enable Const Expressions for Stateful Variables
		if expr != nil && !parser.IsLiteral(expr) {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
				"stateful variable initializer must be a literal value"))
		}
		return
	}
	// Channel read/write and literals already carry their type and SourceID.
	if sym.SourceID != nil || sym.Type.Kind == types.KindChan {
		return
	}
	if expr == nil || isLiteralExpression(context.Child(ctx, expr)) {
		return
	}
	// A bare identifier bound to a reactive variable is itself reactive.
	if primary := parser.GetPrimaryExpression(expr); primary != nil && primary.IDENTIFIER() != nil {
		if ref, rerr := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText()); rerr == nil {
			if ref.Kind == symbol.KindStatefulVariable {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"stateful variables cannot be assigned to ':=' variables"))
				return
			}
			if ref.IsReactive() {
				sym.Type = types.ReadChan(sym.Type)
			}
		}
		return
	}
	// A complex initializer that reads channels is reactive: retype it as a
	// read-only channel. AnalyzeSingleExpression also registers the reactive flow.
	if local != nil {
		flow.AnalyzeSingleExpression(context.Child(ctx, expr))
		if synth, serr := ctx.Scope.Root().GetChildByParserRule(expr); serr == nil &&
			len(synth.Channels.Read) > 0 {
			sym.Type = types.ReadChan(sym.Type)
		}
	}
}

// rhsTracksChannel reports whether expr binds a channel or reads one via a
// channel-read expression: the reactive-RHS test.
func rhsTracksChannel[T antlr.ParserRuleContext](
	ctx context.Context[T],
	expr parser.IExpressionContext,
) bool {
	if expr == nil {
		return false
	}
	childCtx := context.Child(ctx, expr)
	if isLiteralExpression(childCtx) {
		return false
	}
	// A bare reference that resolves to a channel, channel-read/write, or channel-read variable.
	if primary := parser.GetPrimaryExpression(expr); primary != nil && primary.IDENTIFIER() != nil {
		ref, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
		if err != nil {
			return false
		}
		return ref.Kind == symbol.KindChannel || ref.Type.Kind == types.KindChan
	}
	// A compound expression that reads one or more channels is channel-read.
	flow.AnalyzeSingleExpression(childCtx)
	synth, err := ctx.Scope.Root().GetChildByParserRule(expr)
	return err == nil && len(synth.Channels.Read) > 0
}

// rhsReadsVariable reports whether expr references a variable or input symbol.
func rhsReadsVariable[T antlr.ParserRuleContext](
	ctx context.Context[T],
	expr parser.IExpressionContext,
) bool {
	for _, name := range parser.CollectIdentifiers(expr) {
		sym, err := ctx.Scope.Resolve(ctx, name)
		if err != nil {
			continue
		}
		if sym.IsValueVariable() || sym.Kind == symbol.KindInput {
			return true
		}
	}
	return false
}

// AnalyzeAssignment validates an `=` or compound assignment against the variable or
// channel it targets, letting sequences and stages reassign in-scope variables.
func AnalyzeAssignment(ctx context.Context[parser.IAssignmentContext]) {
	analyzeAssignment(ctx)
}

func analyzeVariableDeclaration(ctx context.Context[parser.IVariableDeclarationContext]) {
	if local := ctx.AST.LocalVariable(); local != nil {
		analyzeLocalVariable(context.Child(ctx, local))
		return
	}
	if stateful := ctx.AST.StatefulVariable(); stateful != nil {
		analyzeStatefulVariable(context.Child(ctx, stateful))
	}
}

func analyzeVariableDeclarationType[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	name string,
	expression parser.IExpressionContext,
	typeCtx parser.ITypeContext,
) types.Type {
	if typeCtx != nil {
		varType, err := atypes.InferFromTypeContext(typeCtx)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return types.Type{}
		}
		if expression != nil {
			exprType := atypes.InferFromExpression(context.Child(ctx, expression))
			if exprType.IsValid() && varType.IsValid() {
				// Check magnitude safety for unit conversions (warnings only)
				if varType.Unit != nil && exprType.Unit != nil {
					units.CheckAssignmentScaleSafety(ctx, exprType, varType, nil)
				}

				// If either type is a type variable, add a constraint instead of checking directly
				if exprType.Kind == types.KindVariable || varType.Kind == types.KindVariable {
					if err := atypes.Check(ctx.Constraints, varType, exprType, ctx.AST, "assignment type compatibility"); err != nil {
						ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
						return types.Type{}
					}
				} else {
					isLiteral := isLiteralExpression(context.Child(ctx, expression))
					if (isLiteral && !atypes.LiteralAssignmentCompatible(varType, exprType)) || (!isLiteral && !atypes.Compatible(varType, exprType)) {
						ctx.Diagnostics.Add(diagnostics.Errorf(
							ctx.AST, "type mismatch: cannot assign %s to '%s' (type %s)", exprType, name, varType,
						))
						return types.Type{}
					}
				}
			}
		}
		return varType
	}
	if expression != nil {
		return atypes.InferFromExpression(context.Child(ctx, expression))
	}
	ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "no type declaration found"))
	return types.Type{}
}

func isLiteralExpression(ctx context.Context[parser.IExpressionContext]) bool {
	primary := parser.GetPrimaryExpression(ctx.AST)
	return primary != nil && primary.Literal() != nil
}

// constDefaultValue folds a literal or cast-of-literal initializer into its value,
// returning nil when expr is absent or not a compile-time constant.
func constDefaultValue(expr parser.IExpressionContext, varType types.Type) any {
	if expr == nil {
		return nil
	}
	if parsed, err := literal.ParseConst(expr, varType); err == nil {
		return parsed.Value
	}
	if p := parser.GetPrimaryExpression(expr); p != nil {
		if cast := p.TypeCast(); cast != nil && cast.Expression() != nil {
			if parsed, err := literal.ParseConst(cast.Expression(), varType); err == nil {
				return parsed.Value
			}
		}
	}
	return nil
}

func analyzeLocalVariable(ctx context.Context[parser.ILocalVariableContext]) {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()

	if expr != nil && ctx.AST.Type_() == nil {
		childCtx := context.Child(ctx, expr)
		if chanSym := getChannelSymbol(childCtx); chanSym != nil {
			// Global channel - create a variable that holds the channel key
			// Use KindVariable so it gets a WASM local assigned
			sourceID := chanSym.ID
			chanType := chanSym.Type
			chanType.ChanDirection = types.ChanDirectionRead | types.ChanDirectionWrite
			_, err := childCtx.Scope.Add(ctx, symbol.Symbol{
				Name:     name,
				Kind:     symbol.KindVariable,
				Type:     chanType,
				AST:      ctx.AST,
				SourceID: &sourceID,
			})
			if err != nil {
				ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			}
			return
		}
	}

	if expr != nil {
		expression.Analyze(context.Child(ctx, expr))
	}
	varType := analyzeVariableDeclarationType(
		ctx,
		name,
		expr,
		ctx.AST.Type_(),
	)
	if !varType.IsValid() {
		_, _ = ctx.Scope.Add(ctx, symbol.Symbol{
			Name: name,
			Type: types.Type{},
			AST:  ctx.AST,
		})
		return
	}

	// If assigning from a symbol with channel type, propagate its SourceID
	var sourceID *int
	if varType.Kind == types.KindChan && expr != nil {
		sourceID = getChannelSourceFromExpr(ctx, expr)
		if sourceID != nil {
			varType.ChanDirection = types.ChanDirectionRead | types.ChanDirectionWrite
		}
	}

	defaultValue := constDefaultValue(expr, varType)

	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name:         name,
		Type:         varType,
		AST:          ctx.AST,
		SourceID:     sourceID,
		DefaultValue: defaultValue,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
	}
}

// getChannelSymbol checks if an expression is a simple identifier referencing
// a global channel symbol (KindChannel). Returns the symbol if so, nil otherwise.
func getChannelSymbol(ctx context.Context[parser.IExpressionContext]) *symbol.Symbol {
	primary := parser.GetPrimaryExpression(ctx.AST)
	if primary == nil || primary.IDENTIFIER() == nil {
		return nil
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return nil
	}
	// Must be an actual channel symbol (KindChannel), not just a symbol with channel type.
	// Input params with channel type (KindInput) should be read from, not aliased.
	if sym.Kind == symbol.KindChannel && sym.Type.Kind == types.KindChan {
		return sym
	}
	return nil
}

// getChannelSourceFromExpr extracts the source ID from an expression that references
// a symbol with channel type. Returns a pointer to the source ID, or nil if not found.
func getChannelSourceFromExpr[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	expr parser.IExpressionContext,
) *int {
	primary := parser.GetPrimaryExpression(expr)
	if primary == nil || primary.IDENTIFIER() == nil {
		return nil
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return nil
	}
	if sym.Type.Kind != types.KindChan {
		return nil
	}
	// If the symbol already has a SourceID, propagate it
	if sym.SourceID != nil {
		return sym.SourceID
	}
	// Otherwise, this symbol IS the source (e.g., an input param)
	id := sym.ID
	return &id
}

func analyzeStatefulVariable(ctx context.Context[parser.IStatefulVariableContext]) {
	if ctx.Scope == ctx.Scope.Root() {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"stateful variables cannot be declared at the top level"))
		return
	}
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()
	varType := analyzeVariableDeclarationType(
		ctx,
		name,
		expr,
		ctx.AST.Type_(),
	)
	if !varType.IsValid() {
		_, _ = ctx.Scope.Add(ctx, symbol.Symbol{
			Name: name,
			Kind: symbol.KindStatefulVariable,
			Type: types.Type{},
			AST:  ctx.AST,
		})
		return
	}
	// Stateful variables store VALUES, not channel references.
	// If initialized from a channel, unwrap to get the value type.
	if varType.Kind == types.KindChan {
		varType = varType.Unwrap()
	}
	defaultValue := constDefaultValue(expr, varType)
	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name:         name,
		Kind:         symbol.KindStatefulVariable,
		Type:         varType,
		AST:          ctx.AST,
		DefaultValue: defaultValue,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	if expr != nil {
		expression.Analyze(context.Child(ctx, expr))
	}
}

func analyzeIfStatement(ctx context.Context[parser.IIfStatementContext]) {
	if expr := ctx.AST.Expression(); expr != nil {
		expression.Analyze(context.Child(ctx, expr))
	}

	if block := ctx.AST.Block(); block != nil {
		AnalyzeBlock(context.Child(ctx, block))
	}

	for _, elseIfClause := range ctx.AST.AllElseIfClause() {
		if expr := elseIfClause.Expression(); expr != nil {
			expression.Analyze(context.Child(ctx, expr))
		}
		if block := elseIfClause.Block(); block != nil {
			AnalyzeBlock(context.Child(ctx, block))
		}
	}

	if elseClause := ctx.AST.ElseClause(); elseClause != nil {
		if block := elseClause.Block(); block != nil {
			AnalyzeBlock(context.Child(ctx, block))
		}
	}
}

func analyzeForStatement(ctx context.Context[parser.IForStatementContext]) {
	clause := ctx.AST.ForClause()
	if clause == nil {
		return
	}

	loopScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindLoop,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	loopScope.Name = "__loop"
	loopCtx := ctx.WithScope(loopScope)

	idents := clause.AllIDENTIFIER()
	hasDeclare := clause.DECLARE() != nil
	hasComma := clause.COMMA() != nil
	expr := clause.Expression()

	switch {
	case hasComma && len(idents) == 2:
		analyzeForTwoIdent(loopCtx, clause, idents, expr)
	case hasDeclare && len(idents) == 1:
		analyzeForSingleIdent(loopCtx, clause, idents[0], expr)
	case expr != nil:
		expression.Analyze(context.Child(loopCtx, expr))
	}

	if block := ctx.AST.Block(); block != nil {
		AnalyzeBlock(context.Child(loopCtx, block))
	}
}

func isRangeCall(expr parser.IExpressionContext) (parser.IFunctionCallSuffixContext, bool) {
	primary := parser.GetPrimaryExpression(expr)
	if primary == nil || primary.IDENTIFIER() == nil {
		return nil, false
	}
	if primary.IDENTIFIER().GetText() != "range" {
		return nil, false
	}
	postfix := expr.LogicalOrExpression().
		AllLogicalAndExpression()[0].
		AllEqualityExpression()[0].
		AllRelationalExpression()[0].
		AllAdditiveExpression()[0].
		AllMultiplicativeExpression()[0].
		AllPowerExpression()[0].
		UnaryExpression().
		PostfixExpression()
	if postfix == nil {
		return nil, false
	}
	calls := postfix.AllFunctionCallSuffix()
	if len(calls) != 1 {
		return nil, false
	}
	return calls[0], true
}

func analyzeForSingleIdent(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	ident antlr.TerminalNode,
	expr parser.IExpressionContext,
) {
	name := ident.GetText()

	if funcCall, ok := isRangeCall(expr); ok {
		analyzeForRange(ctx, clause, name, funcCall)
		return
	}

	expression.Analyze(context.Child(ctx, expr))
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))

	if exprType.Kind == types.KindSeries && exprType.Elem != nil {
		elemType := *exprType.Elem
		_, err := ctx.Scope.Add(ctx, symbol.Symbol{
			Name: name,
			Kind: symbol.KindLoopVariable,
			Type: elemType,
			AST:  clause,
		})
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return
		}
		addHiddenLocals(ctx, clause)
		return
	}

	ctx.Diagnostics.Add(diagnostics.Errorf(
		ctx.AST,
		"cannot iterate over %s; did you mean range(%s)?",
		exprType,
		parser.GetExpressionText(expr),
	))
}

func analyzeForTwoIdent(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	idents []antlr.TerminalNode,
	expr parser.IExpressionContext,
) {
	indexName := idents[0].GetText()
	elemName := idents[1].GetText()

	expression.Analyze(context.Child(ctx, expr))
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))

	if exprType.Kind != types.KindSeries || exprType.Elem == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"two-variable for loop requires a series, got %s",
			exprType,
		))
		return
	}

	elemType := *exprType.Elem

	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: indexName,
		Kind: symbol.KindLoopVariable,
		Type: types.I32(),
		AST:  clause,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}

	_, err = ctx.Scope.Add(ctx, symbol.Symbol{
		Name: elemName,
		Kind: symbol.KindLoopVariable,
		Type: elemType,
		AST:  clause,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}

	addHiddenLocals(ctx, clause)
}

func analyzeForRange(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	name string,
	funcCall parser.IFunctionCallSuffixContext,
) {
	args := funcCall.ArgumentList()
	if args == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST, "range() requires 1 to 3 arguments",
		))
		return
	}

	argExprs := args.AllExpression()
	if len(argExprs) < 1 || len(argExprs) > 3 {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST, "range() requires 1 to 3 arguments, got %d",
			len(argExprs),
		))
		return
	}

	var argTypes []types.Type
	for i, argExpr := range argExprs {
		expression.Analyze(context.Child(ctx, argExpr))
		argType := atypes.InferFromExpression(context.Child(ctx, argExpr))
		if !argType.IsValid() {
			return
		}
		if !argType.IsInteger() && !isIntegerLiteral(argType) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				ctx.AST,
				"range() argument %d must be an integer type, got %s",
				i+1, argType,
			))
			return
		}
		argTypes = append(argTypes, argType)
	}

	loopVarType := InferRangeType(argTypes)

	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindLoopVariable,
		Type: loopVarType,
		AST:  clause,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}

	addHiddenLocal(ctx, clause, "__for_limit", loopVarType)
	if len(argExprs) >= 3 {
		addHiddenLocal(ctx, clause, "__for_step", loopVarType)
	}
}

func isIntegerLiteral(t types.Type) bool {
	return t.Kind == types.KindVariable &&
		t.Constraint != nil &&
		(t.Constraint.Kind == types.KindIntegerConstant ||
			t.Constraint.Kind == types.KindNumericConstant)
}

// InferRangeType determines the loop variable type from range() argument types.
// It finds the widest concrete integer type among the arguments. If any concrete
// type is signed, the result is signed at that width. If all arguments are
// untyped literals (no concrete types), it defaults to i64.
func InferRangeType(argTypes []types.Type) types.Type {
	widestBits := 0
	anySigned := false
	for _, t := range argTypes {
		if !t.IsInteger() {
			continue
		}
		if t.IsSignedInteger() {
			anySigned = true
		}
		if bits := getTypeBits(t); bits > widestBits {
			widestBits = bits
		}
	}
	if widestBits == 0 {
		return types.I64()
	}
	if anySigned {
		switch {
		case widestBits <= 8:
			return types.I8()
		case widestBits <= 16:
			return types.I16()
		case widestBits <= 32:
			return types.I32()
		default:
			return types.I64()
		}
	}
	switch {
	case widestBits <= 8:
		return types.U8()
	case widestBits <= 16:
		return types.U16()
	case widestBits <= 32:
		return types.U32()
	default:
		return types.U64()
	}
}

func addHiddenLocals(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
) {
	addHiddenLocal(ctx, clause, "__for_handle", types.I32())
	addHiddenLocal(ctx, clause, "__for_len", types.I32())
	addHiddenLocal(ctx, clause, "__for_idx", types.I32())
}

func addHiddenLocal(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	name string,
	t types.Type,
) {
	// Add with an empty name so Scope.Add skips the parent-scope name conflict
	// check. Without this, nested loops fail because the inner loop's hidden local
	// (e.g. __for_limit) conflicts with the identically-named local in the outer
	// loop scope, causing Add to silently fail and the inner loop to share the
	// outer loop's limit variable.
	child, _ := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindVariable,
		Type: t,
		AST:  clause,
	})
	if child != nil {
		child.Name = name
	}
}

func analyzeBreakStatement(ctx context.Context[parser.IBreakStatementContext]) {
	_, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindLoop)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"break can only be used inside a for loop",
		))
	}
}

func analyzeContinueStatement(ctx context.Context[parser.IContinueStatementContext]) {
	_, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindLoop)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"continue can only be used inside a for loop",
		))
	}
}

func analyzeReturnStatement(ctx context.Context[parser.IReturnStatementContext]) {
	enclosingScope, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	if err != nil {
		enclosingScope, err = ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				ctx.AST,
				"return statement can only be used inside a function body",
			))
			return
		}
	}
	funcName := enclosingScope.Name
	var expectedReturnType types.Type
	if enclosingScope.Kind == symbol.KindFunction {
		if param, ok := enclosingScope.Type.Outputs.Get(ir.DefaultOutputParam); ok {
			expectedReturnType = param.Type
		}
	}
	returnExpr := ctx.AST.Expression()
	if returnExpr != nil {
		expression.Analyze(context.Child(ctx, returnExpr))
		actualReturnType := atypes.InferFromExpression(context.Child(ctx, returnExpr).WithTypeHint(expectedReturnType)).UnwrapChan()

		// Check for void function first - this error applies even in type inference mode
		if !expectedReturnType.IsValid() && !ctx.InTypeInferenceMode {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				ctx.AST,
				"cannot return a value from a function with no return type",
			))
			return
		}

		// Skip type compatibility validation in type inference mode - we're just collecting types
		if ctx.InTypeInferenceMode {
			return
		}
		if actualReturnType.IsValid() && expectedReturnType.IsValid() {
			// If either type is a type variable, add a constraint instead of checking directly
			if actualReturnType.Kind == types.KindVariable || expectedReturnType.Kind == types.KindVariable {
				if err = atypes.Check(
					ctx.Constraints,
					expectedReturnType,
					actualReturnType,
					ctx.AST,
					"return type compatibility",
				); err != nil {
					ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
					return
				}
			} else {
				isLiteral := isLiteralExpression(context.Child(ctx, returnExpr))
				if isLiteral {
					if !atypes.LiteralAssignmentCompatible(expectedReturnType, actualReturnType) {
						ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
							"cannot return %s from '%s': expected %s",
							actualReturnType,
							funcName,
							expectedReturnType,
						))
						return
					}
				} else {
					if !atypes.Compatible(expectedReturnType, actualReturnType) {
						ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
							"cannot return %s from '%s': expected %s",
							actualReturnType,
							funcName,
							expectedReturnType,
						))
						return
					}
				}
			}
		}
		return
	}
	if expectedReturnType.IsValid() {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"return statement in '%s' missing value of type %s",
			funcName,
			expectedReturnType,
		))
	}
}

// analyzeExprReadReassignment re-points an expression-read variable at a new
// derivation, requiring a channel- or variable-fed RHS of the variable's value type.
func analyzeExprReadReassignment(
	ctx context.Context[parser.IAssignmentContext],
	sym *symbol.Symbol,
) {
	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))
	if !rhsTracksChannel(ctx, expr) && !rhsReadsVariable(ctx, expr) {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"type mismatch: cannot reassign '%s' (%s) from a constant value",
			sym.Name, sym.Type.Unwrap()))
		return
	}
	exprType := atypes.InferFromExpression(context.Child(ctx, expr)).UnwrapChan()
	if exprType.IsValid() && !types.StructuralMatch(sym.Type.Unwrap(), exprType) {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"type mismatch: cannot reassign '%s' (%s) from a %s expression",
			sym.Name, sym.Type.Unwrap(), exprType))
		return
	}
	sym.Reassigned = true
}

// analyzeAliasRebind re-points a channel alias at rhs, recording the candidate
// for read and write routing.
func analyzeAliasRebind(
	ctx context.Context[parser.IAssignmentContext],
	alias, rhs *symbol.Symbol,
) {
	if alias.Parent == alias.Root() {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"cannot rebind top-level variable '%s'", alias.Name))
		return
	}
	if !types.StructuralMatch(alias.Type.Unwrap(), rhs.Type.Unwrap()) {
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

func analyzeChannelAssignment(ctx context.Context[parser.IAssignmentContext], channelSym *symbol.Symbol) {
	// A bare-channel RHS on an alias variable rebinds it rather than writing a value.
	if channelSym.Kind == symbol.KindVariable {
		if expr := ctx.AST.Expression(); expr != nil {
			if p := parser.GetPrimaryExpression(expr); p != nil && p.IDENTIFIER() != nil {
				if rhs, rerr := ctx.Resolve(p.IDENTIFIER().GetText()); rerr == nil &&
					rhs.Kind == symbol.KindChannel {
					analyzeAliasRebind(ctx, channelSym, rhs)
					return
				}
			}
		}
	}
	// Validate we're in a function context (channel writes only allowed in imperative context)
	fn, fnErr := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	if errors.Skip(fnErr, query.ErrNotFound) != nil {
		ctx.Diagnostics.Add(diagnostics.Error(fnErr, ctx.AST))
		return
	}
	if fn != nil {
		// Use SourceID if available (for variables assigned from input params),
		// otherwise use the symbol's own ID
		writeID := uint32(channelSym.ID)
		if channelSym.SourceID != nil {
			writeID = uint32(*channelSym.SourceID)
		}
		fn.Channels.Write[writeID] = channelSym.Name
	}

	// Track this as a channel write in the function

	// Analyze and type-check the expression
	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))

	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	chanValueType := channelSym.Type.Unwrap()

	if !exprType.IsValid() || !chanValueType.IsValid() {
		return
	}

	// Check magnitude safety for unit conversions (warnings only)
	if chanValueType.Unit != nil && exprType.Unit != nil {
		units.CheckAssignmentScaleSafety(ctx, exprType, chanValueType, nil)
	}

	// If either type is a type variable, add a constraint instead of checking directly
	if exprType.Kind == types.KindVariable || chanValueType.Kind == types.KindVariable {
		if err := atypes.Check(
			ctx.Constraints,
			chanValueType,
			exprType,
			ctx.AST,
			"channel write type compatibility",
		); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return
		}
	} else {
		isLiteral := isLiteralExpression(context.Child(ctx, expr))
		if (isLiteral && !atypes.LiteralAssignmentCompatible(chanValueType, exprType)) || (!isLiteral && !atypes.Compatible(chanValueType, exprType)) {
			channelName := ctx.AST.IDENTIFIER().GetText()
			ctx.Diagnostics.Add(diagnostics.Errorf(
				ctx.AST, "type mismatch: cannot write %s to channel '%s' (type %s)",
				exprType,
				channelName,
				chanValueType,
			))
		}
	}
}

// analyzeIndexedAssignment validates indexed assignment statements (series[i] = value)
func analyzeIndexedAssignment(
	ctx context.Context[parser.IAssignmentContext],
	varScope *symbol.Symbol,
	indexOrSlice parser.IIndexOrSliceContext,
) {
	// 1. Verify base is a series type
	if varScope.Type.Kind != types.KindSeries {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"indexed assignment only supported on series types",
		))
		return
	}

	// 2. Only support single index (not slices) for now
	if indexOrSlice.COLON() != nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "slice assignment not supported"))
		return
	}

	// 3. Analyze index expression
	indexExprs := indexOrSlice.AllExpression()
	if len(indexExprs) != 1 {
		return
	}
	expression.Analyze(context.Child(ctx, indexExprs[0]))

	// 4. Analyze value expression and check type compatibility
	valueExpr := ctx.AST.Expression()
	expression.Analyze(context.Child(ctx, valueExpr))

	elemType := *varScope.Type.Elem
	exprType := atypes.InferFromExpression(context.Child(ctx, valueExpr))

	if !exprType.IsValid() || !elemType.IsValid() {
		return
	}

	// If either type is a type variable, add a constraint instead of checking directly
	if exprType.Kind == types.KindVariable || elemType.Kind == types.KindVariable {
		if err := atypes.Check(
			ctx.Constraints,
			elemType,
			exprType,
			ctx.AST,
			"indexed assignment type compatibility",
		); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		}
		return
	}

	isLiteral := isLiteralExpression(context.Child(ctx, valueExpr))
	if (isLiteral && !atypes.LiteralAssignmentCompatible(elemType, exprType)) ||
		(!isLiteral && !atypes.Compatible(elemType, exprType)) {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"type mismatch: cannot assign %s to series element of type %s",
			exprType,
			elemType,
		))
	}
}

// analyzeIndexedCompoundAssignment validates indexed compound assignment statements (series[i] += value)
func analyzeIndexedCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	varScope *symbol.Symbol,
	indexOrSlice parser.IIndexOrSliceContext,
	compoundOp parser.ICompoundOpContext,
) {
	if varScope.Type.Kind != types.KindSeries {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"indexed compound assignment only supported on series types",
		))
		return
	}

	if indexOrSlice.COLON() != nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "slice compound assignment not supported"))
		return
	}

	elemType := *varScope.Type.Elem
	if elemType.Kind == types.KindString {
		if compoundOp.PLUS_ASSIGN() == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "string series elements only support += operator"))
			return
		}
	} else if !elemType.IsNumeric() {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"compound assignment requires numeric element type, got %s",
			elemType,
		))
		return
	}

	indexExpressions := indexOrSlice.AllExpression()
	if len(indexExpressions) != 1 {
		return
	}
	expression.Analyze(context.Child(ctx, indexExpressions[0]))

	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))

	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if !exprType.IsValid() || !elemType.IsValid() {
		return
	}

	if exprType.Kind == types.KindVariable || elemType.Kind == types.KindVariable {
		if err := atypes.Check(ctx.Constraints, elemType, exprType, ctx.AST,
			"indexed compound assignment type compatibility"); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		}
		return
	}

	isLiteral := isLiteralExpression(context.Child(ctx, expr))
	if (isLiteral && !atypes.LiteralAssignmentCompatible(elemType, exprType)) ||
		(!isLiteral && !atypes.Compatible(elemType, exprType)) {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"type mismatch: cannot use %s in compound assignment to series element of type %s",
			exprType,
			elemType,
		))
	}
}

// analyzeSeriesCompoundAssignment validates whole-series compound assignment (series += value)
// Supports both series += scalar (broadcast) and series += series (element-wise)
func analyzeSeriesCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	varScope *symbol.Symbol,
	compoundOp parser.ICompoundOpContext,
) {
	elemType := *varScope.Type.Elem

	if elemType.Kind == types.KindString {
		if compoundOp.PLUS_ASSIGN() == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "string series only support += operator"))
			return
		}
	} else if !elemType.IsNumeric() {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "compound assignment requires numeric element type, got %s", elemType))
		return
	}

	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))

	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if !exprType.IsValid() || !elemType.IsValid() {
		return
	}

	if exprType.Kind == types.KindVariable || elemType.Kind == types.KindVariable {
		targetType := elemType
		if exprType.Kind == types.KindSeries {
			targetType = *exprType.Elem
		}
		if err := atypes.Check(ctx.Constraints, elemType, targetType, ctx.AST,
			"series compound assignment type compatibility"); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		}
		return
	}

	// Type compatibility: RHS must be scalar or series with matching element type
	if exprType.Kind == types.KindSeries {
		rhsElemType := *exprType.Elem
		if !atypes.Compatible(elemType, rhsElemType) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				ctx.AST,
				"type mismatch: cannot use %s in compound assignment to %s",
				exprType,
				varScope.Type,
			))
		}
	} else {
		isLiteral := isLiteralExpression(context.Child(ctx, expr))
		if (isLiteral && !atypes.LiteralAssignmentCompatible(elemType, exprType)) ||
			(!isLiteral && !atypes.Compatible(elemType, exprType)) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				ctx.AST,
				"type mismatch: cannot use %s in compound assignment to series of %s",
				exprType,
				elemType,
			))
		}
	}
}

func analyzeCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	varScope *symbol.Symbol,
	compoundOp parser.ICompoundOpContext,
) {
	if indexOrSlice := ctx.AST.IndexOrSlice(); indexOrSlice != nil {
		analyzeIndexedCompoundAssignment(ctx, varScope, indexOrSlice, compoundOp)
		return
	}

	varType := varScope.Type

	if varType.Kind == types.KindChan {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "compound assignment not supported on channels"))
		return
	}

	if varType.Kind == types.KindSeries {
		analyzeSeriesCompoundAssignment(ctx, varScope, compoundOp)
		return
	}

	if varType.Kind == types.KindString {
		if compoundOp.PLUS_ASSIGN() == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "strings only support += operator"))
			return
		}
	} else if !varType.IsNumeric() {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"compound assignment requires numeric type, got %s",
			varType,
		))
		return
	}

	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if !exprType.IsValid() || !varType.IsValid() {
		return
	}
	if exprType.Kind == types.KindVariable || varType.Kind == types.KindVariable {
		if err := atypes.Check(
			ctx.Constraints,
			varType,
			exprType,
			ctx.AST,
			"compound assignment type compatibility",
		); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		}
		return
	}
	if atypes.Compatible(varType, exprType) {
		return
	}
	ctx.Diagnostics.Add(diagnostics.Errorf(
		ctx.AST,
		"type mismatch: cannot use %s in compound assignment to %s",
		exprType,
		varType,
	))
}

func analyzeAssignment(ctx context.Context[parser.IAssignmentContext]) {
	name := ctx.AST.IDENTIFIER().GetText()
	varScope, err := ctx.Resolve(name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}

	if varScope.Kind == symbol.KindLoopVariable {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST, "cannot assign to loop variable '%s'", name,
		))
		return
	}

	// Top-level variables are immutable;
	isChanAlias := varScope.Type.Kind == types.KindChan && varScope.SourceID != nil
	if !isChanAlias && varScope.Parent == varScope.Root() &&
		(varScope.Kind == symbol.KindVariable ||
			varScope.Kind == symbol.KindStatefulVariable) {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"cannot reassign top-level variable '%s'", name))
		return
	}

	if compoundOp := ctx.AST.CompoundOp(); compoundOp != nil {
		analyzeCompoundAssignment(ctx, varScope, compoundOp)
		return
	}

	if indexOrSlice := ctx.AST.IndexOrSlice(); indexOrSlice != nil {
		analyzeIndexedAssignment(ctx, varScope, indexOrSlice)
		return
	}

	if varScope.Type.Kind == types.KindChan {
		if varScope.Kind == symbol.KindVariable && varScope.SourceID == nil {
			analyzeExprReadReassignment(ctx, varScope)
			return
		}
		analyzeChannelAssignment(ctx, varScope)
		return
	}

	if varScope.IsValueVariable() && !varScope.IsReactive() {
		if _, fnErr := varScope.ClosestAncestorOfKind(symbol.KindFunction); errors.Is(fnErr, query.ErrNotFound) {
			varScope.Reassigned = true
		}
	}

	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))
	exprType := atypes.InferFromExpression(context.Child(ctx, expr)).UnwrapChan()
	if !exprType.IsValid() || !varScope.Type.IsValid() {
		return
	}
	varType := varScope.Type

	// Check magnitude safety for unit conversions (warnings only)
	if varType.Unit != nil && exprType.Unit != nil {
		units.CheckAssignmentScaleSafety(ctx, exprType, varType, nil)
	}

	// Check structural compatibility (series/channel structure must match)
	if !types.StructuralMatch(varType, exprType) {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST, "type mismatch: cannot assign %s to '%s' (type %s)", exprType, name, varType,
		))
		return
	}

	// If either type is a type variable, add a constraint instead of checking directly
	if exprType.Kind == types.KindVariable || varType.Kind == types.KindVariable {
		if err := atypes.Check(ctx.Constraints, varType, exprType, ctx.AST, "assignment type compatibility"); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		}
		return
	}
	if atypes.AssignmentCompatible(varType, exprType) {
		return
	}
	ctx.Diagnostics.Add(diagnostics.Errorf(
		ctx.AST, "type mismatch: cannot assign %s to '%s' (type %s)", exprType, name, varType,
	))
}

// AnalyzeFunctionBody analyzes a block and infers its return type by examining
// all return statements across control flow paths.
// Returns the inferred return type (invalid if error occurred).
func AnalyzeFunctionBody(ctx context.Context[parser.IBlockContext]) types.Type {
	ctx.InTypeInferenceMode = true
	funcScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{}),
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return types.Type{}
	}
	blockScope, err := funcScope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return types.Type{}
	}
	var collectedReturnTypes []types.Type
	for _, stmt := range ctx.AST.AllStatement() {
		Analyze(context.Child(ctx, stmt).WithScope(blockScope))
		returnTypes := collectStatementReturnTypes(
			context.Child(ctx, stmt).WithScope(blockScope),
		)
		for _, rt := range returnTypes {
			if rt.IsValid() {
				collectedReturnTypes = append(collectedReturnTypes, rt)
			}
		}
	}
	inferredType, err := unifyReturnTypes(collectedReturnTypes)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return types.Type{}
	}
	return inferredType.Unwrap()
}

// collectStatementReturnTypes extracts all return types from a statement.
// Returns a slice of types (empty if no returns).
func collectStatementReturnTypes(
	ctx context.Context[parser.IStatementContext],
) []types.Type {
	switch {
	case ctx.AST.ReturnStatement() != nil:
		returnStmt := ctx.AST.ReturnStatement()
		returnExpr := returnStmt.Expression()
		if returnExpr == nil {
			// Return statement with no expression (void return)
			return []types.Type{{}}
		}
		returnType := atypes.InferFromExpression(context.Child(ctx, returnExpr))
		if returnType.IsValid() {
			return []types.Type{returnType}
		}
		return []types.Type{}

	case ctx.AST.IfStatement() != nil:
		_, returnTypes := getIfStatementReturnTypes(
			context.Child(ctx, ctx.AST.IfStatement()),
		)
		return returnTypes

	default:
		return []types.Type{}
	}
}

// getIfStatementReturnTypes recursively extracts return types from if/else branches.
// Returns (allPathsReturn bool, returnTypes []types.Type)
func getIfStatementReturnTypes(
	ctx context.Context[parser.IIfStatementContext],
) (bool, []types.Type) {
	var returnTypes []types.Type
	allPathsReturn := true

	// Check main if block
	if block := ctx.AST.Block(); block != nil {
		hasReturn, blockTypes := getBlockReturnTypes(
			context.Child(ctx, block),
		)
		if hasReturn {
			returnTypes = append(returnTypes, blockTypes...)
		} else {
			allPathsReturn = false
		}
	}

	// Check else-if clauses
	for _, elseIfClause := range ctx.AST.AllElseIfClause() {
		if block := elseIfClause.Block(); block != nil {
			hasReturn, blockTypes := getBlockReturnTypes(
				context.Child(ctx, block),
			)
			if hasReturn {
				returnTypes = append(returnTypes, blockTypes...)
			} else {
				allPathsReturn = false
			}
		}
	}

	// Check else clause
	if elseClause := ctx.AST.ElseClause(); elseClause != nil {
		if block := elseClause.Block(); block != nil {
			hasReturn, blockTypes := getBlockReturnTypes(
				context.Child(ctx, block),
			)
			if hasReturn {
				returnTypes = append(returnTypes, blockTypes...)
			} else {
				allPathsReturn = false
			}
		}
	} else {
		// No else clause means not all paths return
		allPathsReturn = false
	}

	return allPathsReturn, returnTypes
}

// getBlockReturnTypes extracts all return types from a block's statements.
// Returns (hasReturn bool, returnTypes []types.Type)
func getBlockReturnTypes(
	ctx context.Context[parser.IBlockContext],
) (bool, []types.Type) {
	var returnTypes []types.Type
	for _, stmt := range ctx.AST.AllStatement() {
		stmtTypes := collectStatementReturnTypes(context.Child(ctx, stmt))
		for _, rt := range stmtTypes {
			if rt.IsValid() {
				returnTypes = append(returnTypes, rt)
			}
		}
	}
	return len(returnTypes) > 0, returnTypes
}

// unifyReturnTypes unifies multiple return types to find the smallest reasonable common type.
func unifyReturnTypes(
	returnTypes []types.Type,
) (types.Type, error) {
	if len(returnTypes) == 0 {
		return types.Type{}, nil
	}

	// Unwrap all types first (Chan(T) -> T, Series(T) -> T) for consistent handling
	unwrappedTypes := make([]types.Type, len(returnTypes))
	for i, t := range returnTypes {
		unwrappedTypes[i] = t.Unwrap()
	}

	if len(unwrappedTypes) == 1 {
		t := unwrappedTypes[0]
		// If it's a type variable (literal), resolve it to a concrete default type
		if t.Kind == types.KindVariable {
			if t.Constraint != nil && t.Constraint.Kind == types.KindIntegerConstant {
				return types.I64(), nil
			}
			if t.Constraint != nil && t.Constraint.Kind == types.KindFloatConstant {
				return types.F64(), nil
			}
			if t.Constraint != nil && t.Constraint.Kind == types.KindNumericConstant {
				return types.F64(), nil
			}
			if t.Constraint != nil && t.Constraint.Kind == types.KindExactIntegerFloatConstant {
				return types.F64(), nil
			}
		}
		return t, nil
	}

	// Separate type variables from concrete types (now all unwrapped)
	var concreteTypes []types.Type
	var typeVariables []types.Type
	for _, t := range unwrappedTypes {
		if t.Kind == types.KindVariable {
			typeVariables = append(typeVariables, t)
		} else {
			concreteTypes = append(concreteTypes, t)
		}
	}

	// If all are type variables (all literals), unify them to a concrete default type
	if len(concreteTypes) == 0 {
		// All literals should unify to a default concrete type
		// For integers, default to i64; for floats, default to f64
		firstVar := typeVariables[0]
		if firstVar.Constraint != nil && firstVar.Constraint.Kind == types.KindIntegerConstant {
			return types.I64(), nil
		}
		if firstVar.Constraint != nil && firstVar.Constraint.Kind == types.KindFloatConstant {
			return types.F64(), nil
		}
		if firstVar.Constraint != nil && firstVar.Constraint.Kind == types.KindNumericConstant {
			return types.F64(), nil
		}
		if firstVar.Constraint != nil && firstVar.Constraint.Kind == types.KindExactIntegerFloatConstant {
			return types.F64(), nil
		}
		return typeVariables[0], nil
	}

	// If we have concrete types, use them to guide the unification
	// Replace type variables with types compatible with the concrete types
	resolvedTypes := make([]types.Type, 0, len(unwrappedTypes))
	for _, t := range unwrappedTypes {
		if t.Kind == types.KindVariable {
			// Infer appropriate type based on concrete types present
			resolved := resolveTypeVariableWithContext(t, concreteTypes)
			resolvedTypes = append(resolvedTypes, resolved)
		} else {
			resolvedTypes = append(resolvedTypes, t)
		}
	}

	firstType := resolvedTypes[0]
	allEqual := true
	for _, t := range resolvedTypes[1:] {
		if !types.Equal(firstType, t) {
			allEqual = false
			break
		}
	}
	if allEqual {
		return firstType, nil
	}

	allNumeric := true
	hasFloat := false
	hasSigned := false
	hasUnsigned := false
	maxBits := 0

	for _, t := range resolvedTypes {
		if !t.IsNumeric() {
			allNumeric = false
			break
		}

		// Unwrap channel/series types to get the actual value type for classification
		unwrapped := t.Unwrap()

		if unwrapped.IsFloat() {
			hasFloat = true
			if unwrapped.Kind == types.KindF32 {
				if maxBits < 32 {
					maxBits = 32
				}
			} else {
				if maxBits < 64 {
					maxBits = 64
				}
			}
		} else if unwrapped.IsInteger() {
			if unwrapped.IsSignedInteger() {
				hasSigned = true
			} else if unwrapped.IsUnsignedInteger() {
				hasUnsigned = true
			}

			bits := getTypeBits(unwrapped)
			if bits > maxBits {
				maxBits = bits
			}
		}
	}

	if !allNumeric {
		return types.Type{}, errors.Newf(
			"incompatible return types: cannot unify %s and %s",
			returnTypes[0],
			returnTypes[1],
		)
	}

	if hasFloat {
		hasInteger := false
		for _, t := range returnTypes {
			if t.IsInteger() {
				hasInteger = true
				break
			}
		}
		if hasInteger {
			return types.Type{}, errors.New(
				"mixed integer and floating-point returns are not allowed",
			)
		}
		if maxBits > 32 {
			return types.F64(), nil
		}
		return types.F32(), nil
	}

	if hasSigned && hasUnsigned {
		if maxBits <= 8 {
			return types.I16(), nil
		} else if maxBits <= 16 {
			return types.I32(), nil
		} else if maxBits <= 32 {
			return types.I64(), nil
		}
		return types.I64(), nil
	}

	if hasSigned {
		if maxBits <= 8 {
			return types.I8(), nil
		} else if maxBits <= 16 {
			return types.I16(), nil
		} else if maxBits <= 32 {
			return types.I32(), nil
		}
		return types.I64(), nil
	}
	if maxBits <= 8 {
		return types.U8(), nil
	} else if maxBits <= 16 {
		return types.U16(), nil
	} else if maxBits <= 32 {
		return types.U32(), nil
	}
	return types.U64(), nil
}

func getTypeBits(t types.Type) int {
	switch t.Kind {
	case types.KindI8, types.KindU8:
		return 8
	case types.KindI16, types.KindU16:
		return 16
	case types.KindI32, types.KindU32:
		return 32
	case types.KindI64, types.KindU64:
		return 64
	case types.KindF32:
		return 32
	case types.KindF64:
		return 64
	default:
		return 0
	}
}

func resolveTypeVariableWithContext(tv types.Type, concreteTypes []types.Type) types.Type {
	if tv.Kind != types.KindVariable {
		return tv
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindIntegerConstant {
		// Check if any concrete type is a float - integer literals can be coerced to floats
		for _, t := range concreteTypes {
			if t.IsFloat() {
				// Integer constant can be coerced to match the float type
				return t
			}
		}

		// All concrete types are integers, determine the best matching integer type
		allUnsigned := true
		for _, t := range concreteTypes {
			if !t.IsInteger() {
				// Not a numeric type we can work with, default to I32
				return types.I32()
			}
			if t.IsSignedInteger() {
				allUnsigned = false
			}
		}
		if allUnsigned {
			maxBits := 0
			for _, t := range concreteTypes {
				bits := getTypeBits(t)
				if bits > maxBits {
					maxBits = bits
				}
			}
			if maxBits <= 8 {
				return types.U8()
			} else if maxBits <= 16 {
				return types.U16()
			} else if maxBits <= 32 {
				return types.U32()
			}
			return types.U64()
		}
		return types.I32()
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindFloatConstant {
		for _, t := range concreteTypes {
			if t.IsFloat() {
				return t
			}
		}
		return types.F64()
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindNumericConstant {
		return types.F64()
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindExactIntegerFloatConstant {
		for _, t := range concreteTypes {
			if t.IsNumeric() {
				return t
			}
		}
		return types.F64()
	}
	return tv
}
