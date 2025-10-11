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

	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
)

// InferFromExpression infers the type of an expression with access to the symbol table
func InferFromExpression(ctx context.Context[parser.IExpressionContext]) ir.Type {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		return InferLogicalOr(context.Child(ctx, logicalOr))
	}
	return nil
}

func InferLogicalOr(ctx context.Context[parser.ILogicalOrExpressionContext]) ir.Type {
	ands := ctx.AST.AllLogicalAndExpression()
	if len(ands) > 1 {
		return ir.U8{}
	}
	if len(ands) == 1 {
		return InferLogicalAnd(context.Child(ctx, ands[0]))
	}
	return nil
}

func InferLogicalAnd(ctx context.Context[parser.ILogicalAndExpressionContext]) ir.Type {
	equalities := ctx.AST.AllEqualityExpression()
	if len(equalities) > 1 {
		return ir.U8{}
	}
	if len(equalities) == 1 {
		return InferEquality(context.Child(ctx, equalities[0]))
	}
	return nil
}

func InferEquality(ctx context.Context[parser.IEqualityExpressionContext]) ir.Type {
	rels := ctx.AST.AllRelationalExpression()
	if len(rels) > 1 {
		return ir.U8{}
	}
	if len(rels) == 1 {
		return InferRelational(context.Child(ctx, rels[0]))
	}
	return nil
}

func InferRelational(ctx context.Context[parser.IRelationalExpressionContext]) ir.Type {
	additives := ctx.AST.AllAdditiveExpression()
	if len(additives) > 1 {
		return ir.U8{}
	}
	if len(additives) == 1 {
		return InferAdditive(context.Child(ctx, additives[0]))
	}
	return nil
}

func InferAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) ir.Type {
	multiplicatives := ctx.AST.AllMultiplicativeExpression()
	if len(multiplicatives) == 0 {
		return nil
	}
	if len(multiplicatives) > 1 {
		firstType := InferMultiplicative(context.Child(ctx, multiplicatives[0]))
		for i := 1; i < len(multiplicatives); i++ {
			nextType := InferMultiplicative(context.Child(ctx, multiplicatives[i]))
			if firstType != nil && nextType != nil && !Compatible(firstType, nextType) {
				return firstType
			}
		}
		return firstType
	}
	return InferMultiplicative(context.Child(ctx, multiplicatives[0]))
}

func InferMultiplicative(ctx context.Context[parser.IMultiplicativeExpressionContext]) ir.Type {
	powers := ctx.AST.AllPowerExpression()
	if len(powers) == 0 {
		return nil
	}
	return InferPower(context.Child(ctx, powers[0]))
}

func InferPower(ctx context.Context[parser.IPowerExpressionContext]) ir.Type {
	if unary := ctx.AST.UnaryExpression(); unary != nil {
		return InferFromUnaryExpression(context.Child(ctx, unary))
	}
	return nil
}

func InferFromUnaryExpression(ctx context.Context[parser.IUnaryExpressionContext]) ir.Type {
	if ctx.AST.UnaryExpression() != nil {
		return InferFromUnaryExpression(context.Child(ctx, ctx.AST.UnaryExpression()))
	}
	if blockingRead := ctx.AST.BlockingReadExpr(); blockingRead != nil {
		if id := blockingRead.IDENTIFIER(); id != nil {
			if chanScope, err := ctx.Scope.Resolve(ctx, id.GetText()); err == nil {
				if chanScope.Type != nil {
					if chanType, ok := chanScope.Type.(ir.Chan); ok {
						return chanType.ValueType
					}
				}
			}
		}
		return nil
	}
	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		return inferPostfixType(context.Child(ctx, postfix))
	}
	return nil
}

func inferPostfixType(ctx context.Context[parser.IPostfixExpressionContext]) ir.Type {
	if primary := ctx.AST.PrimaryExpression(); primary != nil {
		// TODO: Handle function calls and indexing which might change the type
		return inferPrimaryType(context.Child(ctx, primary))
	}
	return nil
}

func inferPrimaryType(ctx context.Context[parser.IPrimaryExpressionContext]) ir.Type {
	if id := ctx.AST.IDENTIFIER(); id != nil {
		if varScope, err := ctx.Scope.Resolve(ctx, id.GetText()); err == nil {
			if varScope.Type != nil {
				return varScope.Type
			}
		}
		return nil
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
	return nil
}

func inferLiteralType(ctx context.Context[parser.ILiteralContext]) ir.Type {
	text := ctx.AST.GetText()
	if len(text) > 0 && (text[0] == '"' || text[0] == '\'') {
		return ir.String{}
	}
	if text == "true" || text == "false" {
		return ir.U8{}
	}
	if isDecimalLiteral(text) {
		if ctx.TypeHint == nil || !ir.IsFloat(ctx.TypeHint) {
			return ir.F64{}
		}
		return ctx.TypeHint
	}
	if ctx.TypeHint == nil || !ir.IsNumeric(ctx.TypeHint) {
		return ir.I64{}
	}
	return ctx.TypeHint
}

func isDecimalLiteral(text string) bool {
	if len(text) == 0 {
		return false
	}
	return strings.ContainsAny(text, ".eE")
}
