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
	"github.com/synnaxlabs/arc/analyzer/result"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func Analyze(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IExpressionContext,
) bool {
	if logicalOr := ctx.LogicalOrExpression(); logicalOr != nil {
		return analyzeLogicalOr(parentScope, result, logicalOr)
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

func validateExpressionType[T any](
	ctx antlr.ParserRuleContext,
	scope *symbol.Scope,
	result *result.Result,
	items []T,
	getOperator func(ctx antlr.ParserRuleContext) string,
	infer func(scope *symbol.Scope, ctx T, hint types.Type) types.Type,
	check func(t types.Type) bool,
) bool {
	if len(items) <= 1 {
		return true
	}
	firstType := infer(scope, items[0], nil)
	opName := getOperator(ctx)
	if !check(firstType) {
		result.AddError(
			errors.Newf("cannot use %s in %s operation", firstType, opName),
			ctx,
		)
		return false
	}
	for i := 1; i < len(items); i++ {
		nextType := infer(scope, items[i], firstType)
		if firstType != nil && nextType != nil && !atypes.Compatible(firstType, nextType) {
			result.AddError(
				errors.Newf("type mismatch: cannot use %s and %s in %s operation", firstType, nextType, opName),
				ctx,
			)
			return false
		}
	}
	return true
}

func analyzeLogicalOr(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.ILogicalOrExpressionContext,
) bool {
	logicalAnds := ctx.AllLogicalAndExpression()
	for _, logicalAnd := range logicalAnds {
		if !analyzeLogicalAnd(parentScope, result, logicalAnd) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		parentScope,
		result,
		logicalAnds,
		getLogicalOrOperator,
		atypes.InferLogicalAnd,
		types.IsBool,
	)
}

func analyzeLogicalAnd(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.ILogicalAndExpressionContext,
) bool {
	equalities := ctx.AllEqualityExpression()
	for _, equality := range ctx.AllEqualityExpression() {
		if !analyzeEquality(parentScope, result, equality) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		parentScope,
		result,
		equalities,
		getLogicalAndOperator,
		atypes.InferEquality,
		types.IsBool,
	)
}

func analyzeEquality(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IEqualityExpressionContext,
) bool {
	relationals := ctx.AllRelationalExpression()
	for _, relational := range relationals {
		if !analyzeRelational(parentScope, result, relational) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		parentScope,
		result,
		relationals,
		getEqualityOperator,
		atypes.InferRelational,
		func(t types.Type) bool { return true },
	)
}

func analyzeRelational(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IRelationalExpressionContext,
) bool {
	additives := ctx.AllAdditiveExpression()
	for _, additive := range additives {
		if !analyzeAdditive(parentScope, result, additive) {
			return false
		}
	}
	return validateExpressionType(
		ctx,
		parentScope,
		result,
		additives,
		getRelationalOperator,
		atypes.InferAdditive,
		types.IsNumeric,
	)
}

func analyzeAdditive(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IAdditiveExpressionContext,
) bool {
	multiplicatives := ctx.AllMultiplicativeExpression()
	for _, multiplicative := range multiplicatives {
		if !analyzeMultiplicative(parentScope, result, multiplicative) {
			return false
		}
	}
	return validateExpressionType[parser.IMultiplicativeExpressionContext](
		ctx,
		parentScope,
		result,
		multiplicatives,
		getAdditiveOperator,
		atypes.InferMultiplicative,
		types.IsNumeric,
	)
}

func analyzeMultiplicative(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IMultiplicativeExpressionContext,
) bool {
	powers := ctx.AllPowerExpression()
	for _, power := range powers {
		if !analyzePower(parentScope, result, power) {
			return false
		}
	}
	return validateExpressionType[parser.IPowerExpressionContext](
		ctx,
		parentScope,
		result,
		powers,
		getMultiplicativeOperator,
		atypes.InferPower,
		types.IsNumeric,
	)
}

func analyzePower(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IPowerExpressionContext,
) bool {
	if unary := ctx.UnaryExpression(); unary != nil {
		if !analyzeUnary(parentScope, result, unary) {
			return false
		}
	}
	if power := ctx.PowerExpression(); power != nil {
		if !analyzePower(parentScope, result, power) {
			return false
		}
	}
	return true
}

func analyzeUnary(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IUnaryExpressionContext,
) bool {
	// Check if this is a unary operator expression
	if innerUnary := ctx.UnaryExpression(); innerUnary != nil {
		// First validate the nested expression
		if !analyzeUnary(parentScope, result, innerUnary) {
			return false
		}
		operandType := atypes.InferFromUnaryExpression(parentScope, innerUnary, nil)
		if ctx.MINUS() != nil {
			if operandType != nil && !types.IsNumeric(operandType) {
				result.AddError(
					errors.Newf("operator - not supported for type %s", operandType),
					ctx,
				)
				return false
			}
		} else if ctx.NOT() != nil {
			if operandType != nil && !types.IsBool(operandType) {
				result.AddError(
					errors.Newf("operator ! requires boolean operand, received %s", operandType),
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
			if _, err := parentScope.Resolve(name); err != nil {
				result.AddError(err, blockingRead)
				return false
			}
		}
		return true
	}
	if postfix := ctx.PostfixExpression(); postfix != nil {
		return analyzePostfix(parentScope, result, postfix)
	}
	return true
}

func analyzePostfix(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IPostfixExpressionContext,
) bool {
	if primary := ctx.PrimaryExpression(); primary != nil {
		if !analyzePrimary(parentScope, result, primary) {
			return false
		}
	}
	for _, indexOrSlice := range ctx.AllIndexOrSlice() {
		for _, expr := range indexOrSlice.AllExpression() {
			if !Analyze(parentScope, result, expr) {
				return false
			}
		}
	}
	for _, funcCall := range ctx.AllFunctionCallSuffix() {
		if argList := funcCall.ArgumentList(); argList != nil {
			for _, expr := range argList.AllExpression() {
				if !Analyze(parentScope, result, expr) {
					return false
				}
			}
		}
	}
	return true
}

func analyzePrimary(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IPrimaryExpressionContext,
) bool {
	if id := ctx.IDENTIFIER(); id != nil {
		name := id.GetText()
		sym, err := parentScope.Resolve(name)
		if err != nil {
			result.AddError(err, ctx)
			return false
		}
		if sym.Kind == symbol.KindChannel || sym.Kind == symbol.KindConfigParam || sym.Kind == symbol.KindParam {
			_, isChan := sym.Type.(types.Chan)
			if isChan {
				if taskScope, err := parentScope.ClosestAncestorOfKind(symbol.KindTask); err == nil {
					t := taskScope.Type.(types.Task)
					t.Channels.Read.Add(sym.ID)
				}
			}
		}
		return true
	}
	if ctx.Literal() != nil {
		return true
	}
	if expr := ctx.Expression(); expr != nil {
		return Analyze(parentScope, result, expr)
	}
	if typeCast := ctx.TypeCast(); typeCast != nil {
		if expr := typeCast.Expression(); expr != nil {
			return Analyze(parentScope, result, expr)
		}
	}
	if builtin := ctx.BuiltinFunction(); builtin != nil {
		if lenExpr := builtin.Expression(); lenExpr != nil {
			return Analyze(parentScope, result, lenExpr)
		}
	}
	return true
}
