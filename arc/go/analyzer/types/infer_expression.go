// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"fmt"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
)

// InferFromExpression determines the type of an Arc expression through recursive descent.
func InferFromExpression(ctx context.Context[parser.IExpressionContext]) types.Type {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		return InferLogicalOr(context.Child(ctx, logicalOr))
	}
	return types.Type{}
}

func InferLogicalOr(ctx context.Context[parser.ILogicalOrExpressionContext]) types.Type {
	ands := ctx.AST.AllLogicalAndExpression()
	if len(ands) > 1 {
		return types.U8()
	}
	if len(ands) == 1 {
		return InferLogicalAnd(context.Child(ctx, ands[0]))
	}
	return types.Type{}
}

func InferLogicalAnd(ctx context.Context[parser.ILogicalAndExpressionContext]) types.Type {
	equalities := ctx.AST.AllEqualityExpression()
	if len(equalities) > 1 {
		return types.U8()
	}
	if len(equalities) == 1 {
		return InferEquality(context.Child(ctx, equalities[0]))
	}
	return types.Type{}
}

func InferEquality(ctx context.Context[parser.IEqualityExpressionContext]) types.Type {
	relExpressions := ctx.AST.AllRelationalExpression()
	if len(relExpressions) > 1 {
		return types.U8()
	}
	if len(relExpressions) == 1 {
		return InferRelational(context.Child(ctx, relExpressions[0]))
	}
	return types.Type{}
}

func InferRelational(ctx context.Context[parser.IRelationalExpressionContext]) types.Type {
	additives := ctx.AST.AllAdditiveExpression()
	if len(additives) > 1 {
		return types.U8()
	}
	if len(additives) == 1 {
		return InferAdditive(context.Child(ctx, additives[0]))
	}
	return types.Type{}
}

func InferAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) types.Type {
	multiplicatives := ctx.AST.AllMultiplicativeExpression()
	if len(multiplicatives) == 0 {
		return types.Type{}
	}
	if len(multiplicatives) > 1 {
		firstType := InferMultiplicative(context.Child(ctx, multiplicatives[0])).Unwrap()
		for i := 1; i < len(multiplicatives); i++ {
			nextType := InferMultiplicative(context.Child(ctx, multiplicatives[i])).Unwrap()
			if !Compatible(firstType, nextType) {
				return firstType
			}
		}
		return firstType
	}
	return InferMultiplicative(context.Child(ctx, multiplicatives[0]))
}

func InferMultiplicative(ctx context.Context[parser.IMultiplicativeExpressionContext]) types.Type {
	powers := ctx.AST.AllPowerExpression()
	if len(powers) == 0 {
		return types.Type{}
	}
	if len(powers) > 1 {
		firstType := InferPower(context.Child(ctx, powers[0])).Unwrap()
		for i := 1; i < len(powers); i++ {
			nextType := InferPower(context.Child(ctx, powers[i])).Unwrap()
			if !Compatible(firstType, nextType) {
				ctx.TypeMap[ctx.AST] = firstType
				return firstType
			}
		}
		ctx.TypeMap[ctx.AST] = firstType
		return firstType
	}
	resultType := InferPower(context.Child(ctx, powers[0]))
	ctx.TypeMap[ctx.AST] = resultType
	return resultType
}

func InferPower(ctx context.Context[parser.IPowerExpressionContext]) types.Type {
	if unary := ctx.AST.UnaryExpression(); unary != nil {
		baseType := InferFromUnaryExpression(context.Child(ctx, unary))

		// If no caret operator, return base type
		if ctx.AST.CARET() == nil || ctx.AST.PowerExpression() == nil {
			return baseType
		}

		// Recursively infer exponent type (right-associative)
		_ = InferPower(context.Child(ctx, ctx.AST.PowerExpression()))

		// Power operation returns the base type
		// (e.g., i32 ^ i32 = i32, f64 ^ f64 = f64)
		return baseType
	}
	return types.Type{}
}

func InferFromUnaryExpression(ctx context.Context[parser.IUnaryExpressionContext]) types.Type {
	if ctx.AST.UnaryExpression() != nil {
		return InferFromUnaryExpression(context.Child(ctx, ctx.AST.UnaryExpression()))
	}
	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		return inferPostfixType(context.Child(ctx, postfix))
	}
	return types.Type{}
}

func inferPostfixType(ctx context.Context[parser.IPostfixExpressionContext]) types.Type {
	if primary := ctx.AST.PrimaryExpression(); primary != nil {
		// TODO: Handle function calls and indexing which might change the type
		// See https://linear.app/synnax/issue/SY-3177/handle-function-calls-in-arc
		return inferPrimaryType(context.Child(ctx, primary))
	}
	return types.Type{}
}

func inferPrimaryType(ctx context.Context[parser.IPrimaryExpressionContext]) types.Type {
	if id := ctx.AST.IDENTIFIER(); id != nil {
		if varScope, err := ctx.Scope.Resolve(ctx, id.GetText()); err == nil {
			if varScope.Type.Kind != types.KindInvalid {
				return varScope.Type
			}
		}
		return types.Type{}
	}
	if literal := ctx.AST.Literal(); literal != nil {
		return inferLiteralType(context.Child(ctx, literal))
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return InferFromExpression(context.Child(ctx, expr))
	}
	if typeCast := ctx.AST.TypeCast(); typeCast != nil {
		if typeCtx := typeCast.Type_(); typeCtx != nil {
			t, _ := InferFromTypeContext(typeCtx)
			return t
		}
	}
	return types.Type{}
}

func inferLiteralType(ctx context.Context[parser.ILiteralContext]) types.Type {
	if numLit := ctx.AST.NumericLiteral(); numLit != nil {
		return inferNumericLiteralType(ctx, numLit)
	}
	text := ctx.AST.GetText()
	if len(text) > 0 && (text[0] == '"' || text[0] == '\'') {
		t := types.String()
		ctx.TypeMap[ctx.AST] = t
		return t
	}
	if text == "true" || text == "false" {
		t := types.U8()
		ctx.TypeMap[ctx.AST] = t
		return t
	}
	// Fallback for unknown literals
	t := types.I64()
	ctx.TypeMap[ctx.AST] = t
	return t
}

func inferNumericLiteralType(
	ctx context.Context[parser.ILiteralContext],
	numLit parser.INumericLiteralContext,
) types.Type {
	// Determine constraint based on literal form (integer vs float)
	// This applies to both plain numeric literals AND unit literals
	var (
		isFloat    = numLit.FLOAT_LITERAL() != nil
		line       = ctx.AST.GetStart().GetLine()
		col        = ctx.AST.GetStart().GetColumn()
		tvName     = fmt.Sprintf("lit_%d_%d", line, col)
		constraint = lo.Ternary(isFloat, types.FloatConstraint(), types.IntegerConstraint())
		tv         = types.Variable(tvName, &constraint)
	)

	// Check for unit suffix (e.g., 5psi, 3s, 100Hz)
	if unitID := numLit.IDENTIFIER(); unitID != nil {
		unitName := unitID.GetText()
		if unit, ok := units.Resolve(unitName); ok {
			tv.Unit = unit
		}
	}

	ctx.Constraints.AddEquality(tv, tv, ctx.AST, "literal type variable")
	ctx.TypeMap[ctx.AST] = tv
	return tv
}
