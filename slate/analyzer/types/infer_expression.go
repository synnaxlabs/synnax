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
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/parser"
)

// InferFromExpression infers the type of an expression with access to the symbol table
func InferFromExpression(scope *symbol.Scope, expr parser.IExpressionContext) symbol.Type {
	if expr == nil {
		return nil
	}
	if logicalOr := expr.LogicalOrExpression(); logicalOr != nil {
		return InferLogicalOr(scope, logicalOr)
	}
	return nil
}

func InferLogicalOr(scope *symbol.Scope, ctx parser.ILogicalOrExpressionContext) symbol.Type {
	ands := ctx.AllLogicalAndExpression()
	if len(ands) > 1 {
		return U8{}
	}
	if len(ands) == 1 {
		return InferLogicalAnd(scope, ands[0])
	}
	return nil
}

func InferLogicalAnd(scope *symbol.Scope, ctx parser.ILogicalAndExpressionContext) symbol.Type {
	equalities := ctx.AllEqualityExpression()
	if len(equalities) > 1 {
		return U8{}
	}
	if len(equalities) == 1 {
		return InferEquality(scope, equalities[0])
	}
	return nil
}

func InferEquality(scope *symbol.Scope, ctx parser.IEqualityExpressionContext) symbol.Type {
	rels := ctx.AllRelationalExpression()
	if len(rels) > 1 {
		return U8{}
	}
	if len(rels) == 1 {
		return InferRelational(scope, rels[0])
	}
	return nil
}

func InferRelational(scope *symbol.Scope, ctx parser.IRelationalExpressionContext) symbol.Type {
	additives := ctx.AllAdditiveExpression()
	if len(additives) > 1 {
		return U8{}
	}
	if len(additives) == 1 {
		return InferAdditive(scope, additives[0])
	}
	return nil
}

func InferAdditive(scope *symbol.Scope, ctx parser.IAdditiveExpressionContext) symbol.Type {
	multiplicatives := ctx.AllMultiplicativeExpression()
	if len(multiplicatives) == 0 {
		return nil
	}
	if len(multiplicatives) > 1 {
		firstType := InferMultiplicative(scope, multiplicatives[0])
		for i := 1; i < len(multiplicatives); i++ {
			nextType := InferMultiplicative(scope, multiplicatives[i])
			if firstType != nil && nextType != nil && !Compatible(firstType, nextType) {
				return firstType
			}
		}
		return firstType
	}
	return InferMultiplicative(scope, multiplicatives[0])
}

func InferMultiplicative(
	scope *symbol.Scope,
	ctx parser.IMultiplicativeExpressionContext,
) symbol.Type {
	powers := ctx.AllPowerExpression()
	if len(powers) == 0 {
		return nil
	}
	return InferPower(scope, powers[0])
}

func InferPower(scope *symbol.Scope, ctx parser.IPowerExpressionContext) symbol.Type {
	if unary := ctx.UnaryExpression(); unary != nil {
		return InferFromUnaryExpression(scope, unary)
	}
	return nil
}

func InferFromUnaryExpression(scope *symbol.Scope, ctx parser.IUnaryExpressionContext) symbol.Type {
	if ctx.UnaryExpression() != nil {
		return InferFromUnaryExpression(scope, ctx.UnaryExpression())
	}
	if blockingRead := ctx.BlockingReadExpr(); blockingRead != nil {
		if id := blockingRead.IDENTIFIER(); id != nil {
			if chanScope, err := scope.Get(id.GetText()); err == nil {
				if chanScope.Symbol != nil && chanScope.Symbol.Type != nil {
					if chanType, ok := chanScope.Symbol.Type.(Chan); ok {
						return chanType.ValueType
					}
				}
			}
		}
		return nil
	}
	if postfix := ctx.PostfixExpression(); postfix != nil {
		return inferPostfixType(scope, postfix)
	}
	return nil
}

func inferPostfixType(scope *symbol.Scope, ctx parser.IPostfixExpressionContext) symbol.Type {
	if primary := ctx.PrimaryExpression(); primary != nil {
		// TODO: Handle function calls and indexing which might change the type
		return inferPrimaryType(scope, primary)
	}
	return nil
}

func inferPrimaryType(scope *symbol.Scope, ctx parser.IPrimaryExpressionContext) symbol.Type {
	if id := ctx.IDENTIFIER(); id != nil {
		if varScope, err := scope.Get(id.GetText()); err == nil {
			if varScope.Symbol != nil && varScope.Symbol.Type != nil {
				if t, ok := varScope.Symbol.Type.(symbol.Type); ok {
					return t
				}
			}
		}
		return nil
	}

	if literal := ctx.Literal(); literal != nil {
		return inferLiteralType(literal)
	}

	if expr := ctx.Expression(); expr != nil {
		return InferFromExpression(scope, expr)
	}

	if typeCast := ctx.TypeCast(); typeCast != nil {
		if typeCtx := typeCast.Type_(); typeCtx != nil {
			t, _ := InferFromTypeContext(typeCtx)
			return t
		}
	}

	return nil
}

func inferLiteralType(ctx parser.ILiteralContext) symbol.Type {
	text := ctx.GetText()
	if len(text) > 0 && (text[0] == '"' || text[0] == '\'') {
		return String{}
	}

	if text == "true" || text == "false" {
		return U8{}
	}

	// Check for numeric type suffixes
	if len(text) >= 3 {
		// Check for type suffixes (u8, u16, u32, u64, i8, i16, i32, i64, f32, f64)
		if text[len(text)-3:] == "u16" {
			return U16{}
		} else if text[len(text)-3:] == "u32" {
			return U32{}
		} else if text[len(text)-3:] == "u64" {
			return U64{}
		} else if text[len(text)-3:] == "i16" {
			return I16{}
		} else if text[len(text)-3:] == "i32" {
			return I32{}
		} else if text[len(text)-3:] == "i64" {
			return I64{}
		} else if text[len(text)-3:] == "f32" {
			return F32{}
		} else if text[len(text)-3:] == "f64" {
			return F64{}
		}
	}
	if len(text) >= 2 {
		if text[len(text)-2:] == "u8" {
			return U8{}
		} else if text[len(text)-2:] == "i8" {
			return I8{}
		}
	}

	// No suffix, use defaults
	for _, ch := range text {
		if ch == '.' || ch == 'e' || ch == 'E' {
			return F64{} // Default to f64 for float literals
		}
	}
	return I32{} // Default to i32 for integer literals
}
