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
	stdcontext "context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

func AnalyzeBlock(ctx context.Context[parser.IBlockContext]) bool {
	blockScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
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
) (types.Type, bool) {
	if typeCtx != nil {
		varType, err := atypes.InferFromTypeContext(typeCtx)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return types.Type{}, false
		}
		if expression != nil {
			exprType := atypes.InferFromExpression(context.Child(ctx, expression))
			if exprType.IsValid() && varType.IsValid() {
				isLiteral := isLiteralExpression(context.Child(ctx, expression))
				if (isLiteral && !atypes.LiteralAssignmentCompatible(varType, exprType)) || (!isLiteral && !atypes.Compatible(varType, exprType)) {
					ctx.Diagnostics.AddError(
						errors.Newf("type mismatch: cannot assign %s to %s", exprType, varType),
						ctx.AST,
					)
					return types.Type{}, false
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
	return types.Type{}, false
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

func analyzeLocalVariable(ctx context.Context[parser.ILocalVariableContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()

	if expr != nil && ctx.AST.Type_() == nil {
		childCtx := context.Child(ctx, expr)
		if isChannelIdentifier(childCtx) {
			chanType := getChannelType(childCtx)
			if chanType.IsValid() {
				_, err := childCtx.Scope.Add(ctx, symbol.Symbol{
					Name: name,
					Kind: symbol.KindChannel,
					Type: *chanType.ValueType,
					AST:  ctx.AST,
				})
				if err != nil {
					ctx.Diagnostics.AddError(err, ctx.AST)
					return false
				}
				return true
			}
		}
	}

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
	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Type: varType,
		AST:  ctx.AST,
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
	return sym.Type.Kind == types.KindChan
}

func getChannelType(ctx context.Context[parser.IExpressionContext]) types.Type {
	primary := getPrimaryExpression(ctx.AST)
	if primary == nil || primary.IDENTIFIER() == nil {
		return types.Type{}
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return types.Type{}
	}
	return sym.Type
}

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
	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindStatefulVariable,
		Type: varType,
		AST:  ctx.AST,
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
	if expr := ctx.AST.Expression(); expr != nil {
		if !expression.Analyze(context.Child(ctx, expr)) {
			return false
		}
	}

	if block := ctx.AST.Block(); block != nil {
		if !AnalyzeBlock(context.Child(ctx, block)) {
			return false
		}
	}

	for _, elseIfClause := range ctx.AST.AllElseIfClause() {
		if expr := elseIfClause.Expression(); expr != nil {
			if !expression.Analyze(context.Child(ctx, expr)) {
				return false
			}
		}
		if block := elseIfClause.Block(); block != nil {
			if !AnalyzeBlock(context.Child(ctx, block)) {
				return false
			}
		}
	}

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
	enclosingScope, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	if err != nil {
		enclosingScope, err = ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
		if err != nil {
			ctx.Diagnostics.AddError(
				errors.New("return statement not in function or fn"),
				ctx.AST,
			)
			return false
		}
	}
	var expectedReturnType types.Type
	if enclosingScope.Kind == symbol.KindFunction {
		expectedReturnType, _ = enclosingScope.Type.Outputs.Get(ir.DefaultOutputParam)
	}
	returnExpr := ctx.AST.Expression()
	if returnExpr != nil {
		if !expression.Analyze(context.Child(ctx, returnExpr)) {
			return false
		}
		actualReturnType := atypes.InferFromExpression(context.Child(ctx, returnExpr).WithTypeHint(expectedReturnType))
		if !expectedReturnType.IsValid() {
			ctx.Diagnostics.AddError(
				errors.New("unexpected return value in function/func with void return type"),
				ctx.AST,
			)
			return false
		}
		if actualReturnType.IsValid() && expectedReturnType.IsValid() {
			isLiteral := isLiteralExpression(context.Child(ctx, returnExpr))
			useLiteralRules := isLiteral || (actualReturnType.IsNumeric() && expectedReturnType.IsNumeric())
			if useLiteralRules {
				if !atypes.LiteralAssignmentCompatible(expectedReturnType, actualReturnType) {
					ctx.Diagnostics.AddError(
						errors.Newf(
							"cannot return %s, expected %s",
							actualReturnType,
							expectedReturnType,
						),
						ctx.AST,
					)
					return false
				}
			} else {
				if !atypes.Compatible(expectedReturnType, actualReturnType) {
					ctx.Diagnostics.AddError(
						errors.Newf(
							"cannot return %s, expected %s",
							actualReturnType,
							expectedReturnType,
						),
						ctx.AST,
					)
					return false
				}
			}
		}
		return true
	}
	if expectedReturnType.IsValid() {
		ctx.Diagnostics.AddError(
			errors.Newf(
				"return statement missing value of type %s",
				expectedReturnType,
			),
			ctx.AST,
		)
		return false
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
	var channelName string
	if ctx.AST.IDENTIFIER() != nil {
		channelName = ctx.AST.IDENTIFIER().GetText()
	} else {
		return false
	}

	fn, fnErr := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	var savedHook func(stdcontext.Context, *symbol.Scope) error
	if fnErr == nil && fn != nil {
		savedHook = fn.OnResolve
		fn.OnResolve = nil
	}

	channelSym, err := ctx.Scope.Resolve(ctx, channelName)

	if fnErr == nil && fn != nil {
		fn.OnResolve = savedHook
	}

	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	if channelSym.Type.Kind != types.KindChan {
		ctx.Diagnostics.AddError(
			errors.Newf("%s is not a channel", channelName),
			ctx.AST,
		)
		return false
	}

	if fnErr == nil && fn != nil {
		if fn.Channels.Write == nil {
			fn.Channels.Write = make(set.Set[uint32])
		}
		fn.Channels.Write.Add(uint32(channelSym.ID))
	}

	expr := ctx.AST.Expression()
	if expr == nil {
		return true
	}

	if !expression.Analyze(context.Child(ctx, expr)) {
		return false
	}

	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if exprType.IsValid() && channelSym.Type.ValueType != nil {
		if !atypes.Compatible(*channelSym.Type.ValueType, exprType) {
			ctx.Diagnostics.AddError(
				errors.Newf(
					"type mismatch: cannot write %s to channel of type %s",
					exprType,
					channelSym.Type.ValueType,
				),
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
	if channelSym.Kind != symbol.KindChannel && channelSym.Kind != symbol.KindConfig && channelSym.Kind != symbol.KindInput && channelSym.Kind != symbol.KindOutput {
		ctx.Diagnostics.AddError(errors.Newf("%s is not a channel", channelName), ctx.AST)
		return false
	}
	if channelSym.Type.Kind != types.KindChan {
		ctx.Diagnostics.AddError(errors.Newf("%s is not a channel", channelName), ctx.AST)
		return false
	}
	_, err = ctx.Scope.Add(ctx, symbol.Symbol{
		Name: varName,
		Kind: symbol.KindVariable,
		Type: channelSym.Type,
		AST:  ctx.AST,
	})
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
	if !exprType.IsValid() || !varScope.Type.IsValid() {
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
