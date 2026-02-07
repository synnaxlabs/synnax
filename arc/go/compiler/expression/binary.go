// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
)

func compileBinaryAdditive(
	ctx context.Context[parser.IAdditiveExpressionContext],
) (types.Type, error) {
	muls := ctx.AST.AllMultiplicativeExpression()
	resultType, err := compileMultiplicative(context.Child(ctx, muls[0]))
	if err != nil {
		return types.Type{}, err
	}

	firstIsSeries := resultType.Kind == types.KindSeries
	var elemType types.Type
	if firstIsSeries {
		elemType = *resultType.Elem
	}

	hintType := resultType
	if resultType.Kind == types.KindChan {
		hintType = resultType.Unwrap()
	}

	var operators []string
	for _, child := range ctx.AST.GetChildren() {
		if termNode, ok := child.(antlr.TerminalNode); ok {
			switch termNode.GetSymbol().GetTokenType() {
			case parser.ArcLexerPLUS:
				operators = append(operators, "+")
			case parser.ArcLexerMINUS:
				operators = append(operators, "-")
			}
		}
	}

	for i := 1; i < len(muls); i++ {
		operandHint := hintType
		if firstIsSeries {
			operandHint = elemType
		}
		operandType, err := compileMultiplicative(context.Child(ctx, muls[i]).WithHint(operandHint))
		if err != nil {
			return types.Type{}, err
		}

		secondIsSeries := operandType.Kind == types.KindSeries

		if firstIsSeries {
			isScalar := !secondIsSeries
			if err := ctx.Resolver.EmitSeriesArithmetic(ctx.Writer, ctx.WriterID, operators[i-1], elemType, isScalar); err != nil {
				return types.Type{}, err
			}
		} else if secondIsSeries {
			secondElemType := *operandType.Elem
			op := operators[i-1]
			if err := ctx.Resolver.EmitSeriesReverseArithmetic(ctx.Writer, ctx.WriterID, op, secondElemType); err != nil {
				return types.Type{}, err
			}
			resultType = operandType
			elemType = secondElemType
			firstIsSeries = true
		} else if hintType.Kind == types.KindString && operators[i-1] == "+" {
			if err := ctx.Resolver.EmitStringConcat(ctx.Writer, ctx.WriterID); err != nil {
				return types.Type{}, err
			}
		} else {
			if err = ctx.Writer.WriteBinaryOpInferred(operators[i-1], hintType); err != nil {
				return types.Type{}, err
			}
		}
	}

	if firstIsSeries {
		return resultType, nil
	}
	return hintType, nil
}

func compileBinaryMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) (types.Type, error) {
	pows := ctx.AST.AllPowerExpression()

	resultType, err := compilePower(context.Child(ctx, pows[0]))
	if err != nil {
		return types.Type{}, err
	}

	firstIsSeries := resultType.Kind == types.KindSeries
	var elemType types.Type
	if firstIsSeries {
		elemType = *resultType.Elem
	}

	hintType := resultType
	if resultType.Kind == types.KindChan {
		hintType = resultType.Unwrap()
	}

	var operators []string
	for _, child := range ctx.AST.GetChildren() {
		if termNode, ok := child.(antlr.TerminalNode); ok {
			switch termNode.GetSymbol().GetTokenType() {
			case parser.ArcLexerSTAR:
				operators = append(operators, "*")
			case parser.ArcLexerSLASH:
				operators = append(operators, "/")
			case parser.ArcLexerPERCENT:
				operators = append(operators, "%")
			}
		}
	}

	for i := 1; i < len(pows); i++ {
		operandHint := hintType
		if firstIsSeries {
			operandHint = elemType
		}
		operandType, err := compilePower(context.Child(ctx, pows[i]).WithHint(operandHint))
		if err != nil {
			return types.Type{}, err
		}

		secondIsSeries := operandType.Kind == types.KindSeries

		if firstIsSeries {
			isScalar := !secondIsSeries
			if err := ctx.Resolver.EmitSeriesArithmetic(ctx.Writer, ctx.WriterID, operators[i-1], elemType, isScalar); err != nil {
				return types.Type{}, err
			}
		} else if secondIsSeries {
			secondElemType := *operandType.Elem
			op := operators[i-1]
			if err := ctx.Resolver.EmitSeriesReverseArithmetic(ctx.Writer, ctx.WriterID, op, secondElemType); err != nil {
				return types.Type{}, err
			}
			resultType = operandType
			elemType = secondElemType
			firstIsSeries = true
		} else {
			if err = ctx.Writer.WriteBinaryOpInferred(operators[i-1], hintType); err != nil {
				return types.Type{}, err
			}
		}
	}

	if firstIsSeries {
		return resultType, nil
	}
	return hintType, nil
}

func compileBinaryRelational(ctx context.Context[parser.IRelationalExpressionContext]) (types.Type, error) {
	adds := ctx.AST.AllAdditiveExpression()
	leftType, err := compileAdditive(context.Child(ctx, adds[0]))
	if err != nil {
		return types.Type{}, err
	}

	isSeries := leftType.Kind == types.KindSeries
	var elemType types.Type
	if isSeries {
		elemType = *leftType.Elem
	}

	hintType := leftType
	if leftType.Kind == types.KindChan {
		hintType = leftType.Unwrap()
	}

	operandHint := hintType
	if isSeries {
		operandHint = elemType
	}

	_, err = compileAdditive(context.Child(ctx, adds[1]).WithHint(operandHint))
	if err != nil {
		return types.Type{}, err
	}
	var op string
	if ctx.AST.LT(0) != nil {
		op = "<"
	} else if ctx.AST.GT(0) != nil {
		op = ">"
	} else if ctx.AST.LEQ(0) != nil {
		op = "<="
	} else if ctx.AST.GEQ(0) != nil {
		op = ">="
	}

	if isSeries {
		if err := ctx.Resolver.EmitSeriesComparison(ctx.Writer, ctx.WriterID, op, elemType); err != nil {
			return types.Type{}, err
		}
		return types.Series(types.U8()), nil
	}

	if err = ctx.Writer.WriteBinaryOpInferred(op, hintType); err != nil {
		return types.Type{}, err
	}
	return types.U8(), nil
}

func compileBinaryEquality(ctx context.Context[parser.IEqualityExpressionContext]) (types.Type, error) {
	rels := ctx.AST.AllRelationalExpression()
	leftType, err := compileRelational(context.Child(ctx, rels[0]))
	if err != nil {
		return types.Type{}, err
	}

	isSeries := leftType.Kind == types.KindSeries
	var elemType types.Type
	if isSeries {
		elemType = *leftType.Elem
	}

	hintType := leftType
	if leftType.Kind == types.KindChan {
		hintType = leftType.Unwrap()
	}

	operandHint := hintType
	if isSeries {
		operandHint = elemType
	}

	_, err = compileRelational(context.Child(ctx, rels[1]).WithHint(operandHint))
	if err != nil {
		return types.Type{}, err
	}
	var op string
	if ctx.AST.EQ(0) != nil {
		op = "=="
	} else if ctx.AST.NEQ(0) != nil {
		op = "!="
	}

	if isSeries {
		if err := ctx.Resolver.EmitSeriesComparison(ctx.Writer, ctx.WriterID, op, elemType); err != nil {
			return types.Type{}, err
		}
		return types.Series(types.U8()), nil
	}

	if hintType.Kind == types.KindString {
		if err := ctx.Resolver.EmitStringEqual(ctx.Writer, ctx.WriterID); err != nil {
			return types.Type{}, err
		}
		if op == "!=" {
			ctx.Writer.WriteI32Eqz()
		}
		return types.U8(), nil
	}

	if err = ctx.Writer.WriteBinaryOpInferred(op, hintType); err != nil {
		return types.Type{}, err
	}
	return types.U8(), nil
}
