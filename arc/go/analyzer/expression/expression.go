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
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func isBool(t basetypes.Type) bool    { return t.IsBool() }
func isNumeric(t basetypes.Type) bool { return t.IsNumeric() }
func isAny(basetypes.Type) bool       { return true }

// IsLiteral checks if an expression is a single literal value with no operators.
func IsLiteral(expr parser.IExpressionContext) bool {
	return isLiteral(expr.LogicalOrExpression())
}

func isLiteral(node antlr.ParserRuleContext) bool {
	if node == nil {
		return false
	}
	switch ctx := node.(type) {
	case parser.ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		return len(ands) == 1 && isLiteral(ands[0])
	case parser.ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		return len(eqs) == 1 && isLiteral(eqs[0])
	case parser.IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		return len(rels) == 1 && isLiteral(rels[0])
	case parser.IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		return len(adds) == 1 && isLiteral(adds[0])
	case parser.IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		return len(muls) == 1 && isLiteral(muls[0])
	case parser.IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		return len(pows) == 1 && isLiteral(pows[0])
	case parser.IPowerExpressionContext:
		return ctx.CARET() == nil && isLiteral(ctx.UnaryExpression())
	case parser.IUnaryExpressionContext:
		return ctx.UnaryExpression() == nil && isLiteral(ctx.PostfixExpression())
	case parser.IPostfixExpressionContext:
		return len(ctx.AllIndexOrSlice()) == 0 &&
			len(ctx.AllFunctionCallSuffix()) == 0 &&
			isLiteral(ctx.PrimaryExpression())
	case parser.IPrimaryExpressionContext:
		return ctx.Literal() != nil
	}
	return false
}

// GetLiteral extracts the literal node from a pure literal expression.
// Callers should first verify IsLiteral returns true.
func GetLiteral(expr parser.IExpressionContext) parser.ILiteralContext {
	return getLiteralNode(expr.LogicalOrExpression())
}

// getLiteralNode extracts the literal from any AST node type.
// Returns nil if the node is not a pure literal. Mirrors isLiteral.
func getLiteralNode(node antlr.ParserRuleContext) parser.ILiteralContext {
	if node == nil {
		return nil
	}
	switch ctx := node.(type) {
	case parser.ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		if len(ands) == 1 {
			return getLiteralNode(ands[0])
		}
	case parser.ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		if len(eqs) == 1 {
			return getLiteralNode(eqs[0])
		}
	case parser.IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		if len(rels) == 1 {
			return getLiteralNode(rels[0])
		}
	case parser.IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		if len(adds) == 1 {
			return getLiteralNode(adds[0])
		}
	case parser.IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		if len(muls) == 1 {
			return getLiteralNode(muls[0])
		}
	case parser.IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		if len(pows) == 1 {
			return getLiteralNode(pows[0])
		}
	case parser.IPowerExpressionContext:
		if ctx.CARET() == nil {
			return getLiteralNode(ctx.UnaryExpression())
		}
	case parser.IUnaryExpressionContext:
		if ctx.UnaryExpression() == nil {
			return getLiteralNode(ctx.PostfixExpression())
		}
	case parser.IPostfixExpressionContext:
		if len(ctx.AllIndexOrSlice()) == 0 && len(ctx.AllFunctionCallSuffix()) == 0 {
			return getLiteralNode(ctx.PrimaryExpression())
		}
	case parser.IPrimaryExpressionContext:
		return ctx.Literal()
	}
	return nil
}

// getSignedIntegerLiteral extracts a signed integer value from a node.
// Supports both plain integer literals (2) and negated ones (-2).
// Returns (value, true) if successful, (0, false) otherwise.
func getSignedIntegerLiteral(node antlr.ParserRuleContext) (int, bool) {
	if node == nil {
		return 0, false
	}
	var (
		sign    = 1
		current = node
	)
	if power, ok := current.(parser.IPowerExpressionContext); ok {
		if power.CARET() != nil {
			return 0, false
		}
		current = power.UnaryExpression()
	}
	if unary, ok := current.(parser.IUnaryExpressionContext); ok {
		if unary.MINUS() != nil {
			sign = -1
			current = unary.UnaryExpression()
		} else if unary.NOT() != nil {
			// NOT doesn't make sense for integer exponent
			return 0, false
		}
	}
	lit := getLiteralNode(current)
	if lit == nil {
		return 0, false
	}
	numLit := lit.NumericLiteral()
	if numLit == nil {
		return 0, false
	}
	intLit := numLit.INTEGER_LITERAL()
	if intLit == nil {
		return 0, false
	}
	if numLit.IDENTIFIER() != nil {
		return 0, false
	}
	parsed, err := literal.ParseNumeric(numLit, basetypes.I64())
	if err != nil {
		return 0, false
	}
	if parsed.Type.Unit != nil {
		return 0, false
	}
	intVal, ok := parsed.Value.(int64)
	if !ok {
		return 0, false
	}
	return sign * int(intVal), true
}

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

	if firstType.Kind != basetypes.KindVariable && !check(firstType) {
		ctx.Diagnostics.AddError(
			errors.Newf("cannot use %s in %s operation", firstType, opName),
			ctx.AST,
		)
		return false
	}

	for i := 1; i < len(items); i++ {
		nextType := infer(context.Child(ctx, items[i]).WithTypeHint(firstType)).Unwrap()

		// Check dimensional compatibility first if either operand has units
		// This must be checked even for type variables since the unit is known at parse time
		// Note: Power operations (^) are handled separately in analyzePower via ValidatePowerOp.
		if firstType.Unit != nil || nextType.Unit != nil {
			if !units.ValidateBinaryOp(ctx, opName, firstType, nextType) {
				return false
			}
		}

		if firstType.Kind == basetypes.KindVariable || nextType.Kind == basetypes.KindVariable {
			ctx.Constraints.AddCompatible(firstType, nextType, items[i], opName+" operands must be compatible")
		} else {
			// Unit compatibility is already validated above by units.ValidateBinaryOp
			if !types.Compatible(firstType, nextType) {
				ctx.Diagnostics.AddError(
					errors.Newf("type mismatch: cannot use %s and %s in %s operation", firstType, nextType, opName),
					ctx.AST,
				)
				return false
			}
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
	power := ctx.AST.PowerExpression()
	if power != nil {
		if !analyzePower(context.Child(ctx, power)) {
			return false
		}
	}

	if ctx.AST.CARET() != nil && power != nil {
		baseType := types.InferFromUnaryExpression(context.Child(ctx, ctx.AST.UnaryExpression())).Unwrap()
		expType := types.InferPower(context.Child(ctx, power)).Unwrap()

		if baseType.Unit != nil || expType.Unit != nil {
			_, isLiteral := getSignedIntegerLiteral(power)
			if err := units.ValidatePowerOp(baseType, expType, isLiteral); err != nil {
				ctx.Diagnostics.AddError(err, ctx.AST)
				return false
			}
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

	funcCalls := ctx.AST.AllFunctionCallSuffix()

	for _, funcCall := range funcCalls {
		if argList := funcCall.ArgumentList(); argList != nil {
			for _, expr := range argList.AllExpression() {
				if !Analyze(context.Child(ctx, expr)) {
					return false
				}
			}
		}
	}

	if len(funcCalls) > 0 {
		primary := ctx.AST.PrimaryExpression()
		if id := primary.IDENTIFIER(); id != nil {
			funcName := id.GetText()
			if funcName != "true" && funcName != "false" {
				if scope, err := ctx.Scope.Resolve(ctx, funcName); err == nil {
					if scope.Kind == symbol.KindFunction {
						if !validateFunctionCall(ctx, scope.Type, funcName, funcCalls[0]) {
							return false
						}
					}
				}
			}
		}
	}

	return true
}

func validateFunctionCall(
	ctx context.Context[parser.IPostfixExpressionContext],
	funcType basetypes.Type,
	funcName string,
	funcCall parser.IFunctionCallSuffixContext,
) bool {
	_, hasDefaultOutput := funcType.Outputs.Get(ir.DefaultOutputParam)
	hasMultipleOutputs := len(funcType.Outputs) > 1 || (len(funcType.Outputs) == 1 && !hasDefaultOutput)
	if hasMultipleOutputs {
		ctx.Diagnostics.AddError(
			errors.Newf("cannot call function %s: functions with multiple named outputs are not callable", funcName),
			funcCall,
		)
		return false
	}

	var args []parser.IExpressionContext
	if argList := funcCall.ArgumentList(); argList != nil {
		args = argList.AllExpression()
	}

	totalCount := len(funcType.Inputs)
	requiredCount := funcType.Inputs.RequiredCount()
	actualCount := len(args)

	if actualCount < requiredCount || actualCount > totalCount {
		if requiredCount == totalCount {
			// No optional params - use existing message format
			ctx.Diagnostics.AddError(
				errors.Newf("function %s expects %d argument(s), got %d",
					funcName, totalCount, actualCount),
				funcCall,
			)
		} else {
			ctx.Diagnostics.AddError(
				errors.Newf("function %s expects %d to %d argument(s), got %d",
					funcName, requiredCount, totalCount, actualCount),
				funcCall,
			)
		}
		return false
	}

	for i, arg := range args {
		paramType := funcType.Inputs[i].Type
		argType := types.InferFromExpression(context.Child(ctx, arg)).Unwrap()
		if paramType.Kind == basetypes.KindVariable || argType.Kind == basetypes.KindVariable {
			ctx.Constraints.AddCompatible(argType, paramType, arg,
				fmt.Sprintf("argument %d of %s", i+1, funcName))
			continue
		}
		if !types.Compatible(argType, paramType) {
			ctx.Diagnostics.AddError(
				errors.Newf("argument %d of %s: expected %s, got %s",
					i+1, funcName, paramType, argType),
				arg,
			)
			return false
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
			if !Analyze(context.Child(ctx, expr)) {
				return false
			}
			// Validate that the cast is allowed
			sourceType := types.InferFromExpression(context.Child(ctx, expr)).Unwrap()
			if typeCtx := typeCast.Type_(); typeCtx != nil {
				targetType, _ := types.InferFromTypeContext(typeCtx)
				if !isValidCast(sourceType, targetType) {
					ctx.Diagnostics.AddError(
						errors.Newf("cannot cast %s to %s", sourceType, targetType),
						ctx.AST,
					)
					return false
				}
			}
		}
	}
	return true
}

func isValidCast(source, target basetypes.Type) bool {
	// Constraint unification will handle type variables
	if source.Kind == basetypes.KindVariable || target.Kind == basetypes.KindVariable {
		return true
	}
	if source.Kind == target.Kind {
		return true
	}
	if source.Kind == basetypes.KindString || target.Kind == basetypes.KindString {
		return false
	}
	return source.IsNumeric() && target.IsNumeric()
}
