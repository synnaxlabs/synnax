// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func isBool(t basetypes.Type) bool            { return t.IsBool() }
func isNumeric(t basetypes.Type) bool         { return t.IsNumeric() }
func isNumericOrString(t basetypes.Type) bool { return t.IsNumeric() || t.Kind == basetypes.KindString }
func isAny(basetypes.Type) bool               { return true }

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
	lit := parser.GetLiteralNode(current)
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
func Analyze(ctx context.Context[parser.IExpressionContext]) {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		analyzeLogicalOr(context.Child(ctx, logicalOr))
	}
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
) {
	if len(items) <= 1 {
		return
	}
	firstType := infer(context.Child(ctx, items[0])).Unwrap()
	opName := getOperator(ctx.AST)

	// If first operand is Invalid, skip validation - we can't check types we don't know
	if firstType.Kind == basetypes.KindInvalid {
		return
	}

	if firstType.Kind != basetypes.KindVariable && !check(firstType) {
		ctx.Diagnostics.AddError(
			errors.Newf("cannot use %s in %s operation", firstType, opName),
			ctx.AST,
		)
		return
	}

	for i := 1; i < len(items); i++ {
		nextType := infer(context.Child(ctx, items[i]).WithTypeHint(firstType)).Unwrap()

		// Skip if this operand is Invalid - we can't check types we don't know
		if nextType.Kind == basetypes.KindInvalid {
			continue
		}

		// Check dimensional compatibility first if either operand has units
		// This must be checked even for type variables since the unit is known at parse time
		// Note: Power operations (^) are handled separately in analyzePower via ValidatePowerOp.
		if firstType.Unit != nil || nextType.Unit != nil {
			if !units.ValidateBinaryOp(ctx, opName, firstType, nextType) {
				return
			}
		}

		if firstType.Kind == basetypes.KindVariable || nextType.Kind == basetypes.KindVariable {
			if err := ctx.Constraints.AddCompatible(firstType, nextType, items[i], opName+" operands must be compatible"); err != nil {
				ctx.Diagnostics.AddError(err, ctx.AST)
				return
			}
		} else {
			// Unit compatibility is already validated above by units.ValidateBinaryOp
			if !types.Compatible(firstType, nextType) {
				ctx.Diagnostics.AddError(
					errors.Newf("type mismatch: cannot use %s and %s in %s operation", firstType, nextType, opName),
					ctx.AST,
				)
				return
			}
		}
	}
}

func analyzeLogicalOr(ctx context.Context[parser.ILogicalOrExpressionContext]) {
	logicalAnds := ctx.AST.AllLogicalAndExpression()
	for _, logicalAnd := range logicalAnds {
		analyzeLogicalAnd(context.Child(ctx, logicalAnd))
	}
	validateType(
		ctx,
		logicalAnds,
		getLogicalOrOperator,
		types.InferLogicalAnd,
		func(t basetypes.Type) bool { return t.IsBool() },
	)
}

func analyzeLogicalAnd(ctx context.Context[parser.ILogicalAndExpressionContext]) {
	equalities := ctx.AST.AllEqualityExpression()
	for _, equality := range equalities {
		analyzeEquality(context.Child(ctx, equality))
	}
	validateType(ctx, equalities, getLogicalAndOperator, types.InferEquality, isBool)
}

func analyzeEquality(ctx context.Context[parser.IEqualityExpressionContext]) {
	relExpressions := ctx.AST.AllRelationalExpression()
	for _, relational := range relExpressions {
		analyzeRelational(context.Child(ctx, relational))
	}
	validateType(
		ctx,
		relExpressions,
		getEqualityOperator,
		types.InferRelational,
		isAny,
	)
}

func analyzeRelational(ctx context.Context[parser.IRelationalExpressionContext]) {
	additives := ctx.AST.AllAdditiveExpression()
	for _, additive := range additives {
		analyzeAdditive(context.Child(ctx, additive))
	}
	validateType(
		ctx,
		additives,
		getRelationalOperator,
		types.InferAdditive,
		isNumeric,
	)
}

func analyzeAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) {
	mults := ctx.AST.AllMultiplicativeExpression()
	for _, multiplicative := range mults {
		analyzeMultiplicative(context.Child(ctx, multiplicative))
	}
	// Determine the operator - strings are only allowed for + (concatenation)
	op := getAdditiveOperator(ctx.AST)
	var check func(basetypes.Type) bool
	if op == "+" {
		check = isNumericOrString
	} else {
		check = isNumeric
	}
	validateType[parser.IMultiplicativeExpressionContext](
		ctx,
		mults,
		getAdditiveOperator,
		types.InferMultiplicative,
		check,
	)
}

func analyzeMultiplicative(ctx context.Context[parser.IMultiplicativeExpressionContext]) {
	powers := ctx.AST.AllPowerExpression()
	for _, power := range powers {
		analyzePower(context.Child(ctx, power))
	}
	validateType[parser.IPowerExpressionContext](
		ctx,
		powers,
		getMultiplicativeOperator,
		types.InferPower,
		isNumeric,
	)
}

func analyzePower(ctx context.Context[parser.IPowerExpressionContext]) {
	if unary := ctx.AST.UnaryExpression(); unary != nil {
		analyzeUnary(context.Child(ctx, unary))
	}
	power := ctx.AST.PowerExpression()
	if power != nil {
		analyzePower(context.Child(ctx, power))
	}

	if ctx.AST.CARET() != nil && power != nil {
		baseType := types.InferFromUnaryExpression(context.Child(ctx, ctx.AST.UnaryExpression())).Unwrap()
		expType := types.InferPower(context.Child(ctx, power)).Unwrap()

		if baseType.Unit != nil || expType.Unit != nil {
			_, isLiteral := getSignedIntegerLiteral(power)
			if err := units.ValidatePowerOp(baseType, expType, isLiteral); err != nil {
				ctx.Diagnostics.AddError(err, ctx.AST)
				return
			}
		}
	}
}

func analyzeUnary(ctx context.Context[parser.IUnaryExpressionContext]) {
	if innerUnary := ctx.AST.UnaryExpression(); innerUnary != nil {
		childCtx := context.Child(ctx, innerUnary)
		analyzeUnary(childCtx)
		operandType := types.InferFromUnaryExpression(childCtx)
		if ctx.AST.MINUS() != nil {
			if !operandType.IsNumeric() {
				ctx.Diagnostics.AddError(
					errors.Newf("operator - not supported for type %s", operandType),
					ctx.AST,
				)
				return
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
				return
			}
		}
		return
	}
	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		analyzePostfix(context.Child(ctx, postfix))
	}
}

func analyzePostfix(ctx context.Context[parser.IPostfixExpressionContext]) {
	if primary := ctx.AST.PrimaryExpression(); primary != nil {
		analyzePrimary(context.Child(ctx, primary))
	}
	for _, indexOrSlice := range ctx.AST.AllIndexOrSlice() {
		for _, expr := range indexOrSlice.AllExpression() {
			Analyze(context.Child(ctx, expr))
		}
	}

	funcCalls := ctx.AST.AllFunctionCallSuffix()

	for _, funcCall := range funcCalls {
		if argList := funcCall.ArgumentList(); argList != nil {
			for _, expr := range argList.AllExpression() {
				Analyze(context.Child(ctx, expr))
			}
		}
	}

	if len(funcCalls) == 0 {
		return
	}
	primary := ctx.AST.PrimaryExpression()
	if id := primary.IDENTIFIER(); id != nil {
		funcName := id.GetText()
		scope, err := ctx.Scope.Resolve(ctx, funcName)
		if err != nil {
			ctx.Diagnostics.AddError(err, primary)
			return
		}
		if scope.Kind == symbol.KindFunction {
			validateFunctionCall(ctx, scope.Type, funcName, funcCalls[0])
		} else {
			ctx.Diagnostics.AddError(
				errors.Newf("cannot call non-function %s of type %s", funcName, scope.Type),
				funcCalls[0],
			)
		}
	}
}

func validateFunctionCall(
	ctx context.Context[parser.IPostfixExpressionContext],
	funcType basetypes.Type,
	funcName string,
	funcCall parser.IFunctionCallSuffixContext,
) {
	_, hasDefaultOutput := funcType.Outputs.Get(ir.DefaultOutputParam)
	hasMultipleOutputs := len(funcType.Outputs) > 1 || (len(funcType.Outputs) == 1 && !hasDefaultOutput)
	if hasMultipleOutputs {
		ctx.Diagnostics.AddError(
			errors.Newf("cannot call function %s: functions with multiple named outputs are not callable", funcName),
			funcCall,
		)
		return
	}

	var args []parser.IExpressionContext
	if argList := funcCall.ArgumentList(); argList != nil {
		args = argList.AllExpression()
	}

	totalCount := len(funcType.Inputs)
	requiredCount := funcType.Inputs.RequiredCount()
	actualCount := len(args)
	signature := ir.FormatFunctionSignature(funcName, funcType)

	if actualCount < requiredCount || actualCount > totalCount {
		var msg string
		if requiredCount == totalCount {
			msg = fmt.Sprintf("function %s expects %d argument(s), got %d",
				funcName, totalCount, actualCount)
		} else {
			msg = fmt.Sprintf("function %s expects %d to %d argument(s), got %d",
				funcName, requiredCount, totalCount, actualCount)
		}
		ctx.Diagnostics.AddErrorWithCodeAndNote(
			diagnostics.ErrorCodeFuncArgCount,
			msg,
			funcCall,
			"signature: "+signature,
		)
		return
	}

	for i, arg := range args {
		paramType := funcType.Inputs[i].Type
		argType := types.InferFromExpression(context.Child(ctx, arg)).UnwrapChan()
		if paramType.Kind == basetypes.KindVariable || argType.Kind == basetypes.KindVariable {
			if err := ctx.Constraints.AddCompatible(argType, paramType, arg,
				fmt.Sprintf("argument %d of %s", i+1, funcName)); err != nil {
				ctx.Diagnostics.AddErrorWithNote(err, arg, "signature: "+signature)
				return
			}
			continue
		}
		if !types.Compatible(argType, paramType) {
			diag := diagnostics.Diagnostic{
				Severity: diagnostics.SeverityError,
				Code:     diagnostics.ErrorCodeFuncArgType,
				Message: fmt.Sprintf("argument %d of %s: expected %s, got %s",
					i+1, funcName, paramType, argType),
				Notes: []diagnostics.Note{{Message: "signature: " + signature}},
			}
			if paramType.IsNumeric() && argType.IsNumeric() {
				diag.Notes = append(
					[]diagnostics.Note{{Message: fmt.Sprintf("hint: use %s(value) to convert", paramType)}},
					diag.Notes...,
				)
			}
			diag.SetRange(arg)
			ctx.Diagnostics.Add(diag)
			return
		}
	}
}

func analyzePrimary(ctx context.Context[parser.IPrimaryExpressionContext]) {
	if id := ctx.AST.IDENTIFIER(); id != nil {
		if _, err := ctx.Scope.Resolve(ctx, id.GetText()); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
		}
		return
	}
	if ctx.AST.Literal() != nil {
		return
	}
	if expr := ctx.AST.Expression(); expr != nil {
		Analyze(context.Child(ctx, expr))
		return
	}
	if typeCast := ctx.AST.TypeCast(); typeCast != nil {
		if expr := typeCast.Expression(); expr != nil {
			Analyze(context.Child(ctx, expr))
			// Validate that the cast is allowed
			sourceType := types.InferFromExpression(context.Child(ctx, expr)).Unwrap()
			if typeCtx := typeCast.Type_(); typeCtx != nil {
				targetType, _ := types.InferFromTypeContext(typeCtx)
				if !isValidCast(sourceType, targetType) {
					ctx.Diagnostics.AddError(
						errors.Newf("cannot cast %s to %s", sourceType, targetType),
						ctx.AST,
					)
				}
			}
		}
	}
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
