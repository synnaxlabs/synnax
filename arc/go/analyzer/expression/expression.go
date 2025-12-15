// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package expression implements type checking and semantic analysis for Arc expressions.
package expression

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/parser"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func isBool(t basetypes.Type) bool    { return t.IsBool() }
func isNumeric(t basetypes.Type) bool { return t.IsNumeric() }
func isAny(basetypes.Type) bool       { return true }

// Analyze validates type correctness of an expression and accumulates constraints.
func Analyze(ctx context.Context[parser.IExpressionContext]) bool {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		return analyzeLogicalOr(context.Child(ctx, logicalOr))
	}
	return true
}

func getLogicalOrOperator(antlr.ParserRuleContext) string { return "or" }

func getLogicalAndOperator(antlr.ParserRuleContext) string { return "and" }

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

func validateType[T antlr.ParserRuleContext, N antlr.ParserRuleContext](
	ctx context.Context[N],
	items []T,
	getOperator func(ctx antlr.ParserRuleContext) string,
	infer func(ctx context.Context[T]) basetypes.Type,
	check func(t basetypes.Type) bool,
) bool {
	if len(items) <= 1 {
		return true
	}
	firstType := infer(context.Child(ctx, items[0])).Unwrap()
	opName := getOperator(ctx.AST)

	// If first type is a type variable, we can't check it yet - will be validated during unification
	if firstType.Kind != basetypes.KindVariable && !check(firstType) {
		ctx.Diagnostics.AddError(
			errors.Newf("cannot use %s in %s operation", firstType, opName),
			ctx.AST,
		)
		return false
	}

	for i := 1; i < len(items); i++ {
		nextType := infer(context.Child(ctx, items[i]).WithTypeHint(firstType)).Unwrap()

		// If either type is a type variable, add a constraint instead of checking directly
		if firstType.Kind == basetypes.KindVariable || nextType.Kind == basetypes.KindVariable {
			ctx.Constraints.AddCompatible(firstType, nextType, items[i], opName+" operands must be compatible")
		} else if !types.Compatible(firstType, nextType) {
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
	return validateType(
		ctx,
		logicalAnds,
		getLogicalOrOperator,
		types.InferLogicalAnd,
		func(t basetypes.Type) bool { return t.IsBool() },
	)
}

func analyzeLogicalAnd(ctx context.Context[parser.ILogicalAndExpressionContext]) bool {
	equalities := ctx.AST.AllEqualityExpression()
	for _, equality := range equalities {
		if !analyzeEquality(context.Child(ctx, equality)) {
			return false
		}
	}
	return validateType(ctx, equalities, getLogicalAndOperator, types.InferEquality, isBool)
}

func analyzeEquality(ctx context.Context[parser.IEqualityExpressionContext]) bool {
	relExpressions := ctx.AST.AllRelationalExpression()
	for _, relational := range relExpressions {
		if !analyzeRelational(context.Child(ctx, relational)) {
			return false
		}
	}
	return validateType(
		ctx,
		relExpressions,
		getEqualityOperator,
		types.InferRelational,
		isAny,
	)
}

func analyzeRelational(ctx context.Context[parser.IRelationalExpressionContext]) bool {
	additives := ctx.AST.AllAdditiveExpression()
	for _, additive := range additives {
		if !analyzeAdditive(context.Child(ctx, additive)) {
			return false
		}
	}
	return validateType(
		ctx,
		additives,
		getRelationalOperator,
		types.InferAdditive,
		isNumeric,
	)
}

func analyzeAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) bool {
	mults := ctx.AST.AllMultiplicativeExpression()
	for _, multiplicative := range mults {
		if !analyzeMultiplicative(context.Child(ctx, multiplicative)) {
			return false
		}
	}
	return validateType[parser.IMultiplicativeExpressionContext](
		ctx,
		mults,
		getAdditiveOperator,
		types.InferMultiplicative,
		isNumeric,
	)
}

func analyzeMultiplicative(ctx context.Context[parser.IMultiplicativeExpressionContext]) bool {
	powers := ctx.AST.AllPowerExpression()
	for _, power := range powers {
		if !analyzePower(context.Child(ctx, power)) {
			return false
		}
	}
	return validateType[parser.IPowerExpressionContext](
		ctx,
		powers,
		getMultiplicativeOperator,
		types.InferPower,
		isNumeric,
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
	if innerUnary := ctx.AST.UnaryExpression(); innerUnary != nil {
		childCtx := context.Child(ctx, innerUnary)
		if !analyzeUnary(childCtx) {
			return false
		}
		operandType := types.InferFromUnaryExpression(childCtx)
		if ctx.AST.MINUS() != nil {
			if !operandType.IsNumeric() {
				ctx.Diagnostics.AddError(
					errors.Newf("operator - not supported for type %s", operandType),
					ctx.AST,
				)
				return false
			}
		} else if ctx.AST.NOT() != nil {
			if !operandType.IsBool() {
				ctx.Diagnostics.AddError(
					errors.Newf(
						"operator 'not' requires boolean operand, received %s",
						operandType,
					),
					ctx.AST,
				)
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
		if _, err := ctx.Scope.Resolve(ctx, id.GetText()); err != nil {
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
