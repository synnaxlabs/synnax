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
	"strings"

	"github.com/synnaxlabs/arc/ir"
)

// InferFromExpression infers the type of an expression with access to the symbol table
func InferFromExpression(scope *ir.Scope, expr text.IExpressionContext, hint ir.Type) ir.Type {
	if expr == nil {
		return nil
	}
	if logicalOr := expr.LogicalOrExpression(); logicalOr != nil {
		return InferLogicalOr(scope, logicalOr, hint)
	}
	return nil
}

func InferLogicalOr(scope *ir.Scope, ctx text.ILogicalOrExpressionContext, hint ir.Type) ir.Type {
	ands := ctx.AllLogicalAndExpression()
	if len(ands) > 1 {
		return ir.U8{}
	}
	if len(ands) == 1 {
		return InferLogicalAnd(scope, ands[0], hint)
	}
	return nil
}

func InferLogicalAnd(scope *ir.Scope, ctx text.ILogicalAndExpressionContext, hint ir.Type) ir.Type {
	equalities := ctx.AllEqualityExpression()
	if len(equalities) > 1 {
		return ir.U8{}
	}
	if len(equalities) == 1 {
		return InferEquality(scope, equalities[0], hint)
	}
	return nil
}

func InferEquality(scope *ir.Scope, ctx text.IEqualityExpressionContext, hint ir.Type) ir.Type {
	rels := ctx.AllRelationalExpression()
	if len(rels) > 1 {
		return ir.U8{}
	}
	if len(rels) == 1 {
		return InferRelational(scope, rels[0], hint)
	}
	return nil
}

func InferRelational(scope *ir.Scope, ctx text.IRelationalExpressionContext, hint ir.Type) ir.Type {
	additives := ctx.AllAdditiveExpression()
	if len(additives) > 1 {
		return ir.U8{}
	}
	if len(additives) == 1 {
		return InferAdditive(scope, additives[0], hint)
	}
	return nil
}

func InferAdditive(scope *ir.Scope, ctx text.IAdditiveExpressionContext, hint ir.Type) ir.Type {
	multiplicatives := ctx.AllMultiplicativeExpression()
	if len(multiplicatives) == 0 {
		return nil
	}
	if len(multiplicatives) > 1 {
		firstType := InferMultiplicative(scope, multiplicatives[0], hint)
		for i := 1; i < len(multiplicatives); i++ {
			nextType := InferMultiplicative(scope, multiplicatives[i], hint)
			if firstType != nil && nextType != nil && !Compatible(firstType, nextType) {
				return firstType
			}
		}
		return firstType
	}
	return InferMultiplicative(scope, multiplicatives[0], hint)
}

func InferMultiplicative(
	scope *ir.Scope,
	ctx text.IMultiplicativeExpressionContext,
	hint ir.Type,
) ir.Type {
	powers := ctx.AllPowerExpression()
	if len(powers) == 0 {
		return nil
	}
	return InferPower(scope, powers[0], hint)
}

func InferPower(scope *ir.Scope, ctx text.IPowerExpressionContext, hint ir.Type) ir.Type {
	if unary := ctx.UnaryExpression(); unary != nil {
		return InferFromUnaryExpression(scope, unary, hint)
	}
	return nil
}

func InferFromUnaryExpression(scope *ir.Scope, ctx text.IUnaryExpressionContext, hint ir.Type) ir.Type {
	if ctx.UnaryExpression() != nil {
		return InferFromUnaryExpression(scope, ctx.UnaryExpression(), hint)
	}
	if blockingRead := ctx.BlockingReadExpr(); blockingRead != nil {
		if id := blockingRead.IDENTIFIER(); id != nil {
			if chanScope, err := scope.Resolve(id.GetText()); err == nil {
				if chanScope.Type != nil {
					if chanType, ok := chanScope.Type.(ir.Chan); ok {
						return chanType.ValueType
					}
				}
			}
		}
		return nil
	}
	if postfix := ctx.PostfixExpression(); postfix != nil {
		return inferPostfixType(scope, postfix, hint)
	}
	return nil
}

func inferPostfixType(scope *ir.Scope, ctx text.IPostfixExpressionContext, hint ir.Type) ir.Type {
	if primary := ctx.PrimaryExpression(); primary != nil {
		// TODO: Handle function calls and indexing which might change the type
		return inferPrimaryType(scope, primary, hint)
	}
	return nil
}

func inferPrimaryType(
	scope *ir.Scope,
	ctx text.IPrimaryExpressionContext,
	hint ir.Type,
) ir.Type {
	if id := ctx.IDENTIFIER(); id != nil {
		if varScope, err := scope.Resolve(id.GetText()); err == nil {
			if varScope.Type != nil {
				if t, ok := varScope.Type.(ir.Type); ok {
					return t
				}
			}
		}
		return nil
	}
	if literal := ctx.Literal(); literal != nil {
		return inferLiteralType(literal, hint)
	}
	if expr := ctx.Expression(); expr != nil {
		return InferFromExpression(scope, expr, hint)
	}
	if typeCast := ctx.TypeCast(); typeCast != nil {
		if typeCtx := typeCast.Type_(); typeCtx != nil {
			t, _ := InferFromTypeContext(typeCtx)
			return t
		}
	}
	return nil
}

func inferLiteralType(
	ctx text.ILiteralContext,
	hint ir.Type,
) ir.Type {
	text := ctx.GetText()
	if len(text) > 0 && (text[0] == '"' || text[0] == '\'') {
		return ir.String{}
	}
	if text == "true" || text == "false" {
		return ir.U8{}
	}
	if isDecimalLiteral(text) {
		if hint == nil || !ir.IsFloat(hint) {
			return ir.F64{}
		}
		return hint
	}
	if hint == nil || !ir.IsNumeric(hint) {
		return ir.I64{}
	}
	return hint
}

func isDecimalLiteral(text string) bool {
	if len(text) == 0 {
		return false
	}
	return strings.ContainsAny(text, ".eE")
}
