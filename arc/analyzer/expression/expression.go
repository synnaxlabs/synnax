// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
)

func Analyze(ctx context.Context[parser.IExpressionContext]) bool {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		return analyzeLogicalOr(context.Child(ctx, logicalOr))
	}
	return true
}

func getLogicalOrOperator(antlr.ParserRuleContext) string {
	return "||"
}

func getLogicalAndOperator(antlr.ParserRuleContext) string {
	return "&&"
}

func getEqualityOperator(ctx antlr.ParserRuleContext) string {
	if eqCtx, ok := ctx.(parser.IEqualityExpressionContext); ok {
		if len(eqCtx.AllEQ()) > 0 {
			return "=="
		}
		if len(eqCtx.AllNEQ()) > 0 {
			return "!="
		}
	}
	return "equality"
}

func getAdditiveOperator(ctx antlr.ParserRuleContext) string {
	if addCtx, ok := ctx.(parser.IAdditiveExpressionContext); ok {
		if len(addCtx.AllPLUS()) > 0 {
			return "+"
		}
		if len(addCtx.AllMINUS()) > 0 {
			return "-"
		}
	}
	return "additive"
}

func getMultiplicativeOperator(ctx antlr.ParserRuleContext) string {
	if mulCtx, ok := ctx.(parser.IMultiplicativeExpressionContext); ok {
		if len(mulCtx.AllSTAR()) > 0 {
			return "*"
		}
		if len(mulCtx.AllSLASH()) > 0 {
			return "/"
		}
		if len(mulCtx.AllPERCENT()) > 0 {
			return "%"
		}
	}
	return "multiplicative"
}

func getRelationalOperator(ctx antlr.ParserRuleContext) string {
	if relCtx, ok := ctx.(parser.IRelationalExpressionContext); ok {
		if len(relCtx.AllLT()) > 0 {
			return "<"
		}
		if len(relCtx.AllGT()) > 0 {
			return ">"
		}
		if len(relCtx.AllLEQ()) > 0 {
			return "<="
		}
		if len(relCtx.AllGEQ()) > 0 {
			return ">="
		}
	}
	return "comparison"
}

func validateExpressionType[T antlr.ParserRuleContext, N antlr.ParserRuleContext](
	ctx context.Context[N],
	items []T,
	getOperator func(ctx antlr.ParserRuleContext) string,
	infer func(ctx context.Context[T]) ir.Type,
	check func(t ir.Type) bool,
) bool {
	if len(items) <= 1 {
		return true
	}
	firstType := infer(context.Child(ctx, items[0]))
	opName := getOperator(ctx.AST)
	if !check(firstType) {
		ctx.Diagnostics.AddError(
			errors.Newf("cannot use %s in %s operation", firstType, opName),
			ctx.AST,
		)
		return false
	}
	for i := 1; i < len(items); i++ {
		nextType := infer(context.Child(ctx, items[i]).WithTypeHint(firstType))
		if firstType != nil && nextType != nil && !atypes.Compatible(firstType, nextType) {
			ctx.Diagnostics.AddError(
				errors.Newf("type mismatch: cannot use %s and %s in %s operation", firstType, nextType, opName),
				ctx.AST,
			)
			return false
		}
	}
	return true
}

func analyzeLogicalOr(ctx context.Context[parser.ILogicalOrExpressionContext]) bool {
	logicalAnds := ctx.AST.AllLogicalAndExpression()
	for _, logicalAnd := range logicalAnds {
		if !analyzeLogicalAnd(context.Child(ctx, logicalAnd)) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		logicalAnds,
		getLogicalOrOperator,
		atypes.InferLogicalAnd,
		ir.IsBool,
	)
}

func analyzeLogicalAnd(ctx context.Context[parser.ILogicalAndExpressionContext]) bool {
	equalities := ctx.AST.AllEqualityExpression()
	for _, equality := range equalities {
		if !analyzeEquality(context.Child(ctx, equality)) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		equalities,
		getLogicalAndOperator,
		atypes.InferEquality,
		ir.IsBool,
	)
}

func analyzeEquality(ctx context.Context[parser.IEqualityExpressionContext]) bool {
	rels := ctx.AST.AllRelationalExpression()
	for _, relational := range rels {
		if !analyzeRelational(context.Child(ctx, relational)) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		rels,
		getEqualityOperator,
		atypes.InferRelational,
		func(t ir.Type) bool { return true },
	)
}

func analyzeRelational(ctx context.Context[parser.IRelationalExpressionContext]) bool {
	additives := ctx.AST.AllAdditiveExpression()
	for _, additive := range additives {
		if !analyzeAdditive(context.Child(ctx, additive)) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		additives,
		getRelationalOperator,
		atypes.InferAdditive,
		ir.IsNumeric,
	)
}

func analyzeAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) bool {
	mults := ctx.AST.AllMultiplicativeExpression()
	for _, multiplicative := range mults {
		if !analyzeMultiplicative(context.Child(ctx, multiplicative)) {
			return false
		}
	}
	return validateExpressionType[parser.IMultiplicativeExpressionContext](
		ctx,
		mults,
		getAdditiveOperator,
		atypes.InferMultiplicative,
		ir.IsNumeric,
	)
}

func analyzeMultiplicative(ctx context.Context[parser.IMultiplicativeExpressionContext]) bool {
	powers := ctx.AST.AllPowerExpression()
	for _, power := range powers {
		if !analyzePower(context.Child(ctx, power)) {
			return false
		}
	}
	return validateExpressionType[parser.IPowerExpressionContext](
		ctx,
		powers,
		getMultiplicativeOperator,
		atypes.InferPower,
		ir.IsNumeric,
	)
}

func analyzePower(ctx context.Context[parser.IPowerExpressionContext]) bool {
	if unary := ctx.AST.UnaryExpression(); unary != nil {
		if !analyzeUnary(context.Child(ctx, unary)) {
			return false
		}
	}
	if power := ctx.AST.PowerExpression(); power != nil {
		if !analyzePower(context.Child(ctx, power)) {
			return false
		}
	}
	return true
}

func analyzeUnary(ctx context.Context[parser.IUnaryExpressionContext]) bool {
	// Check if this is a unary operator expression
	if innerUnary := ctx.AST.UnaryExpression(); innerUnary != nil {
		// First validate the nested expression
		childCtx := context.Child(ctx, innerUnary)
		if !analyzeUnary(childCtx) {
			return false
		}
		operandType := atypes.InferFromUnaryExpression(childCtx)
		if ctx.AST.MINUS() != nil {
			if operandType != nil && !ir.IsNumeric(operandType) {
				ctx.Diagnostics.AddError(
					errors.Newf("operator - not supported for type %s", operandType),
					ctx.AST,
				)
				return false
			}
		} else if ctx.AST.NOT() != nil {
			if operandType != nil && !ir.IsBool(operandType) {
				ctx.Diagnostics.AddError(
					errors.Newf("operator ! requires boolean operand, received %s", operandType),
					ctx.AST,
				)
				return false
			}
		}
		return true
	}
	if blockingRead := ctx.AST.BlockingReadExpr(); blockingRead != nil {
		if id := blockingRead.IDENTIFIER(); id != nil {
			name := id.GetText()
			if _, err := ctx.Scope.Resolve(ctx, name); err != nil {
				ctx.Diagnostics.AddError(err, blockingRead)
				return false
			}
		}
		return true
	}
	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		return analyzePostfix(context.Child(ctx, postfix))
	}
	return true
}

func analyzePostfix(ctx context.Context[parser.IPostfixExpressionContext]) bool {
	if primary := ctx.AST.PrimaryExpression(); primary != nil {
		if !analyzePrimary(context.Child(ctx, primary)) {
			return false
		}
	}
	for _, indexOrSlice := range ctx.AST.AllIndexOrSlice() {
		for _, expr := range indexOrSlice.AllExpression() {
			if !Analyze(context.Child(ctx, expr)) {
				return false
			}
		}
	}
	for _, funcCall := range ctx.AST.AllFunctionCallSuffix() {
		if argList := funcCall.ArgumentList(); argList != nil {
			for _, expr := range argList.AllExpression() {
				if !Analyze(context.Child(ctx, expr)) {
					return false
				}
			}
		}
	}
	return true
}

func analyzePrimary(ctx context.Context[parser.IPrimaryExpressionContext]) bool {
	if id := ctx.AST.IDENTIFIER(); id != nil {
		name := id.GetText()
		if _, err := ctx.Scope.Resolve(ctx, name); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		return true
	}
	if ctx.AST.Literal() != nil {
		return true
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return Analyze(context.Child(ctx, expr))
	}
	if typeCast := ctx.AST.TypeCast(); typeCast != nil {
		if expr := typeCast.Expression(); expr != nil {
			return Analyze(context.Child(ctx, expr))
		}
	}
	if builtin := ctx.AST.BuiltinFunction(); builtin != nil {
		if lenExpr := builtin.Expression(); lenExpr != nil {
			return Analyze(context.Child(ctx, lenExpr))
		}
	}
	return true
}
