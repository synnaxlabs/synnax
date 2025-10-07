// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
)

func AnalyzeBlock(ctx context.Context[parser.IBlockContext]) bool {
	blockScope, err := ctx.Scope.Add(ctx, ir.Symbol{
		Kind:       ir.KindBlock,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	for _, stmt := range ctx.AST.AllStatement() {
		if !Analyze(context.Child(ctx, stmt).WithScope(blockScope)) {
			return false
		}
	}
	return true
}

// Analyze analyzes a statement
func Analyze(ctx context.Context[parser.IStatementContext]) bool {
	switch {
	case ctx.AST.VariableDeclaration() != nil:
		return analyzeVariableDeclaration(context.Child(ctx, ctx.AST.VariableDeclaration()))
	case ctx.AST.IfStatement() != nil:
		return analyzeIfStatement(context.Child(ctx, ctx.AST.IfStatement()))
	case ctx.AST.ReturnStatement() != nil:
		return analyzeReturnStatement(context.Child(ctx, ctx.AST.ReturnStatement()))
	case ctx.AST.ChannelOperation() != nil:
		return analyzeChannelOperation(context.Child(ctx, ctx.AST.ChannelOperation()))
	case ctx.AST.Assignment() != nil:
		return analyzeAssignment(context.Child(ctx, ctx.AST.Assignment()))
	case ctx.AST.Expression() != nil:
		return expression.Analyze(context.Child(ctx, ctx.AST.Expression()))
	}
	return true
}

func analyzeVariableDeclaration(ctx context.Context[parser.IVariableDeclarationContext]) bool {
	if local := ctx.AST.LocalVariable(); local != nil {
		return analyzeLocalVariable(context.Child(ctx, local))
	}
	if stateful := ctx.AST.StatefulVariable(); stateful != nil {
		return analyzeStatefulVariable(context.Child(ctx, stateful))
	}
	return true
}

func analyzeVariableDeclarationType[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	expression parser.IExpressionContext,
	typeCtx parser.ITypeContext,
) (ir.Type, bool) {
	if typeCtx != nil {
		var varType ir.Type
		var err error
		// Explicit type annotation
		varType, err = atypes.InferFromTypeContext(typeCtx)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return nil, false
		}
		// If there's an initializer, check type compatibility
		if expression != nil {
			// Check if the expression is a literal
			exprType := atypes.InferFromExpression(context.Child(ctx, expression))
			if exprType != nil && varType != nil {
				isLiteral := isLiteralExpression(context.Child(ctx, expression))
				// If it's a literal, we might allow some implicit conversions
				// For now, we still check compatibility the same way
				if (isLiteral && !atypes.LiteralAssignmentCompatible(varType, exprType)) || (!isLiteral && !atypes.Compatible(varType, exprType)) {
					ctx.Diagnostics.AddError(
						errors.Newf("type mismatch: cannot assign %s to %s", exprType, varType),
						ctx.AST,
					)
					return nil, false
				}
			}
		}
		return varType, true
	}
	if expression != nil {
		return atypes.InferFromExpression(context.Child(ctx, expression)), true
	}
	ctx.Diagnostics.AddError(
		errors.Newf("no type declaration found for %s", ctx.AST), ctx.AST,
	)
	return nil, false
}

func getPrimaryExpression(expr parser.IExpressionContext) parser.IPrimaryExpressionContext {
	if expr == nil {
		return nil
	}
	logicalOr := expr.LogicalOrExpression()
	if logicalOr == nil || len(logicalOr.AllLogicalAndExpression()) != 1 {
		return nil
	}
	ands := logicalOr.AllLogicalAndExpression()[0]
	if len(ands.AllEqualityExpression()) != 1 {
		return nil
	}
	eq := ands.AllEqualityExpression()[0]
	if len(eq.AllRelationalExpression()) != 1 {
		return nil
	}
	rel := eq.AllRelationalExpression()[0]
	if len(rel.AllAdditiveExpression()) != 1 {
		return nil
	}
	add := rel.AllAdditiveExpression()[0]
	if len(add.AllMultiplicativeExpression()) != 1 {
		return nil
	}
	mult := add.AllMultiplicativeExpression()[0]
	if len(mult.AllPowerExpression()) != 1 {
		return nil
	}
	pow := mult.AllPowerExpression()[0]
	unary := pow.UnaryExpression()
	if unary == nil {
		return nil
	}
	postfix := unary.PostfixExpression()
	if postfix == nil {
		return nil
	}
	return postfix.PrimaryExpression()
}

func isLiteralExpression(ctx context.Context[parser.IExpressionContext]) bool {
	primary := getPrimaryExpression(ctx.AST)
	return primary != nil && primary.Literal() != nil
}

// analyzeLocalVariable analyzes a local variable declaration
func analyzeLocalVariable(ctx context.Context[parser.ILocalVariableContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()

	// Check if this is actually a non-blocking channel read
	// (i.e., the expression is a simple identifier that refers to a channel)
	if expr != nil && ctx.AST.Type_() == nil {
		childCtx := context.Child(ctx, expr)
		if isChannelIdentifier(childCtx) {
			// This is a non-blocking channel read: varName := channelName
			chanType := getChannelType(childCtx)
			if chanType != nil {
				// Add variable with the channel's value type
				_, err := childCtx.Scope.Add(ctx, ir.Symbol{
					Name:       name,
					Kind:       ir.KindChannel,
					Type:       chanType.ValueType,
					ParserRule: ctx.AST,
				})
				if err != nil {
					ctx.Diagnostics.AddError(err, ctx.AST)
					return false
				}
				return true
			}
		}
	}

	// Also validate the expression for undefined variables
	if expr != nil {
		if !expression.Analyze(context.Child(ctx, expr)) {
			return false
		}
	}
	varType, ok := analyzeVariableDeclarationType(
		ctx,
		expr,
		ctx.AST.Type_(),
	)
	if !ok {
		return false
	}
	_, err := ctx.Scope.Add(ctx, ir.Symbol{
		Name:       name,
		Type:       varType,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return true
}

func isChannelIdentifier(ctx context.Context[parser.IExpressionContext]) bool {
	primary := getPrimaryExpression(ctx.AST)
	if primary == nil || primary.IDENTIFIER() == nil {
		return false
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return false
	}
	_, ok := sym.Type.(ir.Chan)
	return ok
}

func getChannelType(ctx context.Context[parser.IExpressionContext]) *ir.Chan {
	primary := getPrimaryExpression(ctx.AST)
	if primary == nil || primary.IDENTIFIER() == nil {
		return nil
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return nil
	}
	chanType, ok := sym.Type.(ir.Chan)
	if !ok {
		return nil
	}
	return &chanType
}

// analyzeStatefulVariable analyzes a stateful variable declaration
func analyzeStatefulVariable(ctx context.Context[parser.IStatefulVariableContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()
	varType, ok := analyzeVariableDeclarationType(
		ctx,
		expr,
		ctx.AST.Type_(),
	)
	if !ok {
		return false
	}
	_, err := ctx.Scope.Add(ctx, ir.Symbol{
		Name:       name,
		Kind:       ir.KindStatefulVariable,
		Type:       varType,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	if expr != nil {
		return expression.Analyze(context.Child(ctx, expr))
	}
	return true
}

func analyzeIfStatement(ctx context.Context[parser.IIfStatementContext]) bool {
	// First analyze the condition expression
	if expr := ctx.AST.Expression(); expr != nil {
		if !expression.Analyze(context.Child(ctx, expr)) {
			return false
		}
	}

	// Analyze the main if block
	if block := ctx.AST.Block(); block != nil {
		if !AnalyzeBlock(context.Child(ctx, block)) {
			return false
		}
	}

	// Analyze all else-if clauses
	for _, elseIfClause := range ctx.AST.AllElseIfClause() {
		// Analyze the else-if condition
		if expr := elseIfClause.Expression(); expr != nil {
			if !expression.Analyze(context.Child(ctx, expr)) {
				return false
			}
		}
		// Analyze the else-if block
		if block := elseIfClause.Block(); block != nil {
			if !AnalyzeBlock(context.Child(ctx, block)) {
				return false
			}
		}
	}

	// Analyze the else clause if present
	if elseClause := ctx.AST.ElseClause(); elseClause != nil {
		if block := elseClause.Block(); block != nil {
			if !AnalyzeBlock(context.Child(ctx, block)) {
				return false
			}
		}
	}

	return true
}

func analyzeReturnStatement(ctx context.Context[parser.IReturnStatementContext]) bool {
	enclosingScope, err := ctx.Scope.ClosestAncestorOfKind(ir.KindFunction)
	if err != nil {
		enclosingScope, err = ctx.Scope.ClosestAncestorOfKind(ir.KindStage)
		if err != nil {
			ctx.Diagnostics.AddError(
				errors.New("return statement not in function or stage"),
				ctx.AST,
			)
			return false
		}
	}
	var expectedReturnType ir.Type
	if enclosingScope.Kind == ir.KindFunction {
		fnType := enclosingScope.Type.(ir.Function)
		expectedReturnType, _ = fnType.Outputs.Get("output")
	} else if enclosingScope.Kind == ir.KindStage {
		stageType := enclosingScope.Type.(ir.Stage)
		expectedReturnType, _ = stageType.Outputs.Get("output")
	}
	returnExpr := ctx.AST.Expression()
	if returnExpr != nil {
		if !expression.Analyze(context.Child(ctx, returnExpr)) {
			return false
		}

		actualReturnType := atypes.InferFromExpression(context.Child(ctx, returnExpr).WithTypeHint(expectedReturnType))
		if expectedReturnType == nil {
			ctx.Diagnostics.AddError(
				errors.New("unexpected return value in function/stage with void return type"),
				ctx.AST,
			)
			return false
		}
		if actualReturnType != nil {
			if !atypes.Compatible(expectedReturnType, actualReturnType) {
				ctx.Diagnostics.AddError(
					errors.Newf(
						"type mismatch: cannot return %s, expected %s",
						actualReturnType,
						expectedReturnType,
					),
					ctx.AST,
				)
				return false
			}
		}
	} else {
		if expectedReturnType != nil {
			ctx.Diagnostics.AddError(
				errors.Newf(
					"return statement missing value of type %s",
					expectedReturnType,
				),
				ctx.AST,
			)
			return false
		}
	}
	return true
}

func analyzeChannelOperation(ctx context.Context[parser.IChannelOperationContext]) bool {
	if write := ctx.AST.ChannelWrite(); write != nil {
		return analyzeChannelWrite(context.Child(ctx, write))
	}
	if read := ctx.AST.ChannelRead(); read != nil {
		return analyzeChannelRead(context.Child(ctx, read))
	}
	return true
}

func analyzeChannelWrite(ctx context.Context[parser.IChannelWriteContext]) bool {
	// Get the channel name
	var channelName string
	if ctx.AST.IDENTIFIER() != nil {
		channelName = ctx.AST.IDENTIFIER().GetText()
	} else {
		return false
	}

	// Resolve the channel
	channelSym, err := ctx.Scope.Resolve(ctx, channelName)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	// Check it's a channel type
	chanType, ok := channelSym.Type.(ir.Chan)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Newf("%s is not a channel", channelName),
			ctx.AST,
		)
		return false
	}

	// Analyze the expression being written
	expr := ctx.AST.Expression()
	if expr == nil {
		return true
	}

	if !expression.Analyze(context.Child(ctx, expr)) {
		return false
	}

	// Check type compatibility
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if exprType != nil && chanType.ValueType != nil {
		if !atypes.Compatible(chanType.ValueType, exprType) {
			ctx.Diagnostics.AddError(
				errors.Newf("type mismatch: cannot write %s to channel of type %s",
					exprType, chanType.ValueType),
				ctx.AST,
			)
			return false
		}
	}

	return true
}

func analyzeChannelRead(ctx context.Context[parser.IChannelReadContext]) bool {
	if blocking := ctx.AST.BlockingRead(); blocking != nil {
		return analyzeBlockingRead(context.Child(ctx, blocking))
	}
	if nonBlocking := ctx.AST.NonBlockingRead(); nonBlocking != nil {
		return analyzeNonBlockingRead(context.Child(ctx, nonBlocking))
	}
	return true
}

func analyzeBlockingRead(ctx context.Context[parser.IBlockingReadContext]) bool {
	ids := ctx.AST.AllIDENTIFIER()
	if len(ids) != 2 {
		return false
	}
	return createChannelReadVariable(ctx, ids[0].GetText(), ids[1].GetText())
}

func analyzeNonBlockingRead(ctx context.Context[parser.INonBlockingReadContext]) bool {
	ids := ctx.AST.AllIDENTIFIER()
	if len(ids) != 2 {
		return false
	}
	return createChannelReadVariable(ctx, ids[0].GetText(), ids[1].GetText())
}

func createChannelReadVariable[T antlr.ParserRuleContext](
	ctx context.Context[T],
	varName, channelName string,
) bool {
	channelSym, err := ctx.Scope.Resolve(ctx, channelName)
	if err != nil {
		ctx.Diagnostics.AddError(errors.Wrapf(err, "undefined channel: %s", channelName), ctx.AST)
		return false
	}
	chanType, ok := channelSym.Type.(ir.Chan)
	if !ok {
		ctx.Diagnostics.AddError(errors.Newf("%s is not a channel", channelName), ctx.AST)
		return false
	}
	_, err = ctx.Scope.Add(ctx, ir.Symbol{Name: varName, Kind: ir.KindVariable, Type: chanType.ValueType, ParserRule: ctx.AST})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return true
}

func analyzeAssignment(ctx context.Context[parser.IAssignmentContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	varScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	expr := ctx.AST.Expression()
	if expr == nil {
		return true
	}
	if !expression.Analyze(context.Child(ctx, expr)) {
		return false
	}
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if exprType == nil {
		return true
	}
	if varScope.Type == nil {
		return true
	}
	varType := varScope.Type
	if atypes.Compatible(varType, exprType) {
		return true
	}
	ctx.Diagnostics.AddError(
		errors.Newf("type mismatch: cannot assign %s to variable of type %s", exprType, varType),
		ctx.AST,
	)
	return false
}
