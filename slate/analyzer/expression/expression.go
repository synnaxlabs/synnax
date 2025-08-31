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
	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

func Visit(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IExpressionContext,
) bool {
	if logicalOr := ctx.LogicalOrExpression(); logicalOr != nil {
		return visitLogicalOr(parentScope, result, logicalOr)
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

func validateExpressionType[T any](
	ctx antlr.ParserRuleContext,
	scope *symbol.Scope,
	result *result.Result,
	items []T,
	getOperator func(ctx antlr.ParserRuleContext) string,
	infer func(scope *symbol.Scope, ctx T) symbol.Type,
	check func(t symbol.Type) bool,
) bool {
	if len(items) <= 1 {
		return true
	}
	firstType := infer(scope, items[0])
	opName := getOperator(ctx)
	if !check(firstType) {
		result.AddError(
			errors.Newf("cannot use %s in %s operation", firstType, opName),
			ctx,
		)
		return false
	}
	for i := 1; i < len(items); i++ {
		nextType := infer(scope, items[i])
		if firstType != nil && nextType != nil && !types.Compatible(firstType, nextType) {
			result.AddError(
				errors.Newf("type mismatch: cannot use %s and %s in %s operation", firstType, nextType, opName),
				ctx,
			)
			return false
		}
	}
	return true
}

func visitLogicalOr(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.ILogicalOrExpressionContext,
) bool {
	logicalAnds := ctx.AllLogicalAndExpression()
	for _, logicalAnd := range logicalAnds {
		if !visitLogicalAnd(parentScope, result, logicalAnd) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		parentScope,
		result,
		logicalAnds,
		getLogicalOrOperator,
		types.InferLogicalAnd,
		types.IsBool,
	)
}

func visitLogicalAnd(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.ILogicalAndExpressionContext,
) bool {
	equalities := ctx.AllEqualityExpression()
	for _, equality := range ctx.AllEqualityExpression() {
		if !visitEquality(parentScope, result, equality) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		parentScope,
		result,
		equalities,
		getLogicalAndOperator,
		types.InferEquality,
		types.IsBool,
	)
}

func visitEquality(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IEqualityExpressionContext,
) bool {
	relationals := ctx.AllRelationalExpression()
	for _, relational := range relationals {
		if !visitRelational(parentScope, result, relational) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		parentScope,
		result,
		relationals,
		getEqualityOperator,
		types.InferRelational,
		func(t symbol.Type) bool { return true },
	)
}

func visitRelational(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IRelationalExpressionContext,
) bool {
	for _, additive := range ctx.AllAdditiveExpression() {
		if !visitAdditive(parentScope, result, additive) {
			return false
		}
	}
	return true
}

func visitAdditive(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IAdditiveExpressionContext,
) bool {
	multiplicatives := ctx.AllMultiplicativeExpression()
	for _, multiplicative := range multiplicatives {
		if !visitMultiplicative(parentScope, result, multiplicative) {
			return false
		}
	}
	return validateExpressionType[parser.IMultiplicativeExpressionContext](
		ctx,
		parentScope,
		result,
		multiplicatives,
		getAdditiveOperator,
		types.InferMultiplicative,
		types.IsNumeric,
	)
}

func visitMultiplicative(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IMultiplicativeExpressionContext,
) bool {
	powers := ctx.AllPowerExpression()
	for _, power := range powers {
		if !visitPower(parentScope, result, power) {
			return false
		}
	}
	return validateExpressionType[parser.IPowerExpressionContext](
		ctx,
		parentScope,
		result,
		powers,
		getMultiplicativeOperator,
		types.InferPower,
		types.IsNumeric,
	)
}

func visitPower(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IPowerExpressionContext,
) bool {
	if unary := ctx.UnaryExpression(); unary != nil {
		if !visitUnary(parentScope, result, unary) {
			return false
		}
	}
	if power := ctx.PowerExpression(); power != nil {
		if !visitPower(parentScope, result, power) {
			return false
		}
	}
	return true
}

func visitUnary(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IUnaryExpressionContext,
) bool {
	// Check if this is a unary operator expression
	if innerUnary := ctx.UnaryExpression(); innerUnary != nil {
		// First validate the nested expression
		if !visitUnary(parentScope, result, innerUnary) {
			return false
		}
		operandType := types.InferFromUnaryExpression(parentScope, innerUnary)
		if ctx.MINUS() != nil {
			// Unary negation requires numeric type
			if operandType != nil && !types.IsNumeric(operandType) {
				result.AddError(
					errors.Newf("operator - not supported for type %s", operandType),
					ctx,
				)
				return false
			}
		} else if ctx.NOT() != nil {
			// Logical NOT requires boolean type
			if operandType != nil && !types.IsBool(operandType) {
				result.AddError(
					errors.Newf("operator ! requires boolean operand, got %s", operandType),
					ctx,
				)
				return false
			}
		}

		return true
	}
	if blockingRead := ctx.BlockingReadExpr(); blockingRead != nil {
		if id := blockingRead.IDENTIFIER(); id != nil {
			name := id.GetText()
			if _, err := parentScope.Get(name); err != nil {
				result.AddError(err, blockingRead)
				return false
			}
		}
		return true
	}
	if postfix := ctx.PostfixExpression(); postfix != nil {
		return visitPostfix(parentScope, result, postfix)
	}
	return true
}

func visitPostfix(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IPostfixExpressionContext,
) bool {
	if primary := ctx.PrimaryExpression(); primary != nil {
		if !visitPrimary(parentScope, result, primary) {
			return false
		}
	}
	for _, indexOrSlice := range ctx.AllIndexOrSlice() {
		for _, expr := range indexOrSlice.AllExpression() {
			if !Visit(parentScope, result, expr) {
				return false
			}
		}
	}
	for _, funcCall := range ctx.AllFunctionCallSuffix() {
		if argList := funcCall.ArgumentList(); argList != nil {
			for _, expr := range argList.AllExpression() {
				if !Visit(parentScope, result, expr) {
					return false
				}
			}
		}
	}
	return true
}

func visitPrimary(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IPrimaryExpressionContext,
) bool {
	if id := ctx.IDENTIFIER(); id != nil {
		name := id.GetText()
		if _, err := parentScope.Get(name); err != nil {
			result.AddError(err, ctx)
			return false
		}
		return true
	}
	if ctx.Literal() != nil {
		return true
	}
	if expr := ctx.Expression(); expr != nil {
		return Visit(parentScope, result, expr)
	}
	if typeCast := ctx.TypeCast(); typeCast != nil {
		if expr := typeCast.Expression(); expr != nil {
			return Visit(parentScope, result, expr)
		}
	}
	if builtin := ctx.BuiltinFunction(); builtin != nil {
		if lenExpr := builtin.Expression(); lenExpr != nil {
			return Visit(parentScope, result, lenExpr)
		}
	}
	return true
}
