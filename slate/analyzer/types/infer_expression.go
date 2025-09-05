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

	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
)

// InferFromExpression infers the type of an expression with access to the symbol table
func InferFromExpression(scope *symbol.Scope, expr parser.IExpressionContext, hint types.Type) types.Type {
	if expr == nil {
		return nil
	}
	if logicalOr := expr.LogicalOrExpression(); logicalOr != nil {
		return InferLogicalOr(scope, logicalOr, hint)
	}
	return nil
}

func InferLogicalOr(scope *symbol.Scope, ctx parser.ILogicalOrExpressionContext, hint types.Type) types.Type {
	ands := ctx.AllLogicalAndExpression()
	if len(ands) > 1 {
		return types.U8{}
	}
	if len(ands) == 1 {
		return InferLogicalAnd(scope, ands[0], hint)
	}
	return nil
}

func InferLogicalAnd(scope *symbol.Scope, ctx parser.ILogicalAndExpressionContext, hint types.Type) types.Type {
	equalities := ctx.AllEqualityExpression()
	if len(equalities) > 1 {
		return types.U8{}
	}
	if len(equalities) == 1 {
		return InferEquality(scope, equalities[0], hint)
	}
	return nil
}

func InferEquality(scope *symbol.Scope, ctx parser.IEqualityExpressionContext, hint types.Type) types.Type {
	rels := ctx.AllRelationalExpression()
	if len(rels) > 1 {
		return types.U8{}
	}
	if len(rels) == 1 {
		return InferRelational(scope, rels[0], hint)
	}
	return nil
}

func InferRelational(scope *symbol.Scope, ctx parser.IRelationalExpressionContext, hint types.Type) types.Type {
	additives := ctx.AllAdditiveExpression()
	if len(additives) > 1 {
		return types.U8{}
	}
	if len(additives) == 1 {
		return InferAdditive(scope, additives[0], hint)
	}
	return nil
}

func InferAdditive(scope *symbol.Scope, ctx parser.IAdditiveExpressionContext, hint types.Type) types.Type {
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
	scope *symbol.Scope,
	ctx parser.IMultiplicativeExpressionContext,
	hint types.Type,
) types.Type {
	powers := ctx.AllPowerExpression()
	if len(powers) == 0 {
		return nil
	}
	return InferPower(scope, powers[0], hint)
}

func InferPower(scope *symbol.Scope, ctx parser.IPowerExpressionContext, hint types.Type) types.Type {
	if unary := ctx.UnaryExpression(); unary != nil {
		return InferFromUnaryExpression(scope, unary, hint)
	}
	return nil
}

func InferFromUnaryExpression(scope *symbol.Scope, ctx parser.IUnaryExpressionContext, hint types.Type) types.Type {
	if ctx.UnaryExpression() != nil {
		return InferFromUnaryExpression(scope, ctx.UnaryExpression(), hint)
	}
	if blockingRead := ctx.BlockingReadExpr(); blockingRead != nil {
		if id := blockingRead.IDENTIFIER(); id != nil {
			if chanScope, err := scope.Resolve(id.GetText()); err == nil {
				if chanScope.Type != nil {
					if chanType, ok := chanScope.Type.(types.Chan); ok {
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

func inferPostfixType(scope *symbol.Scope, ctx parser.IPostfixExpressionContext, hint types.Type) types.Type {
	if primary := ctx.PrimaryExpression(); primary != nil {
		// TODO: Handle function calls and indexing which might change the type
		return inferPrimaryType(scope, primary, hint)
	}
	return nil
}

func inferPrimaryType(
	scope *symbol.Scope,
	ctx parser.IPrimaryExpressionContext,
	hint types.Type,
) types.Type {
	if id := ctx.IDENTIFIER(); id != nil {
		if varScope, err := scope.Resolve(id.GetText()); err == nil {
			if varScope.Type != nil {
				if t, ok := varScope.Type.(types.Type); ok {
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
	ctx parser.ILiteralContext,
	hint types.Type,
) types.Type {
	text := ctx.GetText()
	if len(text) > 0 && (text[0] == '"' || text[0] == '\'') {
		return types.String{}
	}
	if text == "true" || text == "false" {
		return types.U8{}
	}
	if isDecimalLiteral(text) {
		if hint == nil || !types.IsFloat(hint) {
			return types.F64{}
		}
		return hint
	}
	if hint == nil || !types.IsNumeric(hint) {
		return types.I64{}
	}
	return hint
}

func isDecimalLiteral(text string) bool {
	if len(text) == 0 {
		return false
	}
	return strings.ContainsAny(text, ".eE")
}
