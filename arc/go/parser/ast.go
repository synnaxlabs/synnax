// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser

import "github.com/antlr4-go/antlr/v4"

// IsLiteral checks if an expression is a single literal value with no operators.
func IsLiteral(expr IExpressionContext) bool {
	return isLiteral(expr.LogicalOrExpression())
}

func isLiteral(node antlr.ParserRuleContext) bool {
	if node == nil {
		return false
	}
	switch ctx := node.(type) {
	case ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		return len(ands) == 1 && isLiteral(ands[0])
	case ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		return len(eqs) == 1 && isLiteral(eqs[0])
	case IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		return len(rels) == 1 && isLiteral(rels[0])
	case IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		return len(adds) == 1 && isLiteral(adds[0])
	case IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		return len(muls) == 1 && isLiteral(muls[0])
	case IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		return len(pows) == 1 && isLiteral(pows[0])
	case IPowerExpressionContext:
		return ctx.CARET() == nil && isLiteral(ctx.UnaryExpression())
	case IUnaryExpressionContext:
		return ctx.UnaryExpression() == nil && isLiteral(ctx.PostfixExpression())
	case IPostfixExpressionContext:
		return len(ctx.AllIndexOrSlice()) == 0 &&
			len(ctx.AllFunctionCallSuffix()) == 0 &&
			isLiteral(ctx.PrimaryExpression())
	case IPrimaryExpressionContext:
		return ctx.Literal() != nil
	}
	return false
}

// GetLiteral extracts the literal node from a pure literal expression.
// Callers should first verify IsLiteral returns true.
func GetLiteral(expr IExpressionContext) ILiteralContext {
	return GetLiteralNode(expr.LogicalOrExpression())
}

// GetLiteralNode extracts the literal from any AST node type.
func GetLiteralNode(node antlr.ParserRuleContext) ILiteralContext {
	if node == nil {
		return nil
	}
	switch ctx := node.(type) {
	case ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		if len(ands) == 1 {
			return GetLiteralNode(ands[0])
		}
	case ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		if len(eqs) == 1 {
			return GetLiteralNode(eqs[0])
		}
	case IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		if len(rels) == 1 {
			return GetLiteralNode(rels[0])
		}
	case IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		if len(adds) == 1 {
			return GetLiteralNode(adds[0])
		}
	case IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		if len(muls) == 1 {
			return GetLiteralNode(muls[0])
		}
	case IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		if len(pows) == 1 {
			return GetLiteralNode(pows[0])
		}
	case IPowerExpressionContext:
		if ctx.CARET() == nil {
			return GetLiteralNode(ctx.UnaryExpression())
		}
	case IUnaryExpressionContext:
		if ctx.UnaryExpression() == nil {
			return GetLiteralNode(ctx.PostfixExpression())
		}
	case IPostfixExpressionContext:
		if len(ctx.AllIndexOrSlice()) == 0 && len(ctx.AllFunctionCallSuffix()) == 0 {
			return GetLiteralNode(ctx.PrimaryExpression())
		}
	case IPrimaryExpressionContext:
		return ctx.Literal()
	}
	return nil
}

// IsNumericLiteral checks if an expression is a numeric literal (int or float),
// possibly with a unary minus. This is more permissive than IsLiteral for cases
// like [-1, -2.0] where we want to treat negated numbers as literals.
func IsNumericLiteral(expr IExpressionContext) bool {
	return isNumericLiteral(expr.LogicalOrExpression())
}

func isNumericLiteral(node antlr.ParserRuleContext) bool {
	if node == nil {
		return false
	}
	switch ctx := node.(type) {
	case ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		return len(ands) == 1 && isNumericLiteral(ands[0])
	case ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		return len(eqs) == 1 && isNumericLiteral(eqs[0])
	case IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		return len(rels) == 1 && isNumericLiteral(rels[0])
	case IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		return len(adds) == 1 && isNumericLiteral(adds[0])
	case IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		return len(muls) == 1 && isNumericLiteral(muls[0])
	case IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		return len(pows) == 1 && isNumericLiteral(pows[0])
	case IPowerExpressionContext:
		return ctx.CARET() == nil && isNumericLiteral(ctx.UnaryExpression())
	case IUnaryExpressionContext:
		if ctx.MINUS() != nil {
			return isNumericLiteral(ctx.UnaryExpression())
		}
		return ctx.UnaryExpression() == nil && isNumericLiteral(ctx.PostfixExpression())
	case IPostfixExpressionContext:
		return len(ctx.AllIndexOrSlice()) == 0 &&
			len(ctx.AllFunctionCallSuffix()) == 0 &&
			isNumericLiteral(ctx.PrimaryExpression())
	case IPrimaryExpressionContext:
		if lit := ctx.Literal(); lit != nil {
			return lit.NumericLiteral() != nil
		}
		return false
	}
	return false
}

// GetPrimaryExpression extracts the primary expression from an expression that has no
// operators. Returns nil if the expression contains any binary or unary operators.
func GetPrimaryExpression(expr IExpressionContext) IPrimaryExpressionContext {
	if expr == nil {
		return nil
	}
	logicalOr := expr.LogicalOrExpression()
	if logicalOr == nil || len(logicalOr.AllLogicalAndExpression()) != 1 {
		return nil
	}
	ands := logicalOr.AllLogicalAndExpression()[0]
	if len(ands.AllEqualityExpression()) != 1 {
		return nil
	}
	eq := ands.AllEqualityExpression()[0]
	if len(eq.AllRelationalExpression()) != 1 {
		return nil
	}
	rel := eq.AllRelationalExpression()[0]
	if len(rel.AllAdditiveExpression()) != 1 {
		return nil
	}
	add := rel.AllAdditiveExpression()[0]
	if len(add.AllMultiplicativeExpression()) != 1 {
		return nil
	}
	mult := add.AllMultiplicativeExpression()[0]
	if len(mult.AllPowerExpression()) != 1 {
		return nil
	}
	pow := mult.AllPowerExpression()[0]
	if pow.CARET() != nil {
		return nil
	}
	unary := pow.UnaryExpression()
	if unary == nil {
		return nil
	}
	postfix := unary.PostfixExpression()
	if postfix == nil {
		return nil
	}
	return postfix.PrimaryExpression()
}

// GetExpressionText extracts the source text of an expression from the token stream.
func GetExpressionText(expr IExpressionContext) string {
	if expr == nil {
		return ""
	}
	start := expr.GetStart()
	stop := expr.GetStop()
	if start != nil && stop != nil {
		stream := start.GetTokenSource().GetInputStream()
		if stream != nil {
			return stream.GetText(start.GetStart(), stop.GetStop())
		}
	}
	return expr.GetText()
}
