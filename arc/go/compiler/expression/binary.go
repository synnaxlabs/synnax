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
	"github.com/synnaxlabs/x/errors"
)

func compileBitwiseChain[N antlr.ParserRuleContext](
	ctx context.Context[N],
	n int,
	opAt func(i int) string,
	compileChild func(i int, hint types.Type, hasHint bool) (types.Type, error),
) (types.Type, error) {
	resultType, err := compileChild(0, types.Type{}, false)
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
	for i := 1; i < n; i++ {
		op := opAt(i - 1)
		operandHint := hintType
		if firstIsSeries {
			operandHint = elemType
		}
		operandType, err := compileChild(i, operandHint, true)
		if err != nil {
			return types.Type{}, err
		}
		secondIsSeries := operandType.Kind == types.KindSeries
		if firstIsSeries || secondIsSeries {
			seriesElem := elemType
			if !firstIsSeries {
				seriesElem = *operandType.Elem
			}
			if !seriesElem.IsInteger() {
				return types.Type{}, errors.Newf(
					"bitwise operators require integer series elements, got %s",
					seriesElem,
				)
			}
		}
		if firstIsSeries {
			if err := ctx.Resolver.EmitSeriesArithmetic(
				ctx.Writer,
				ctx.WriterID,
				op,
				elemType,
				!secondIsSeries,
			); err != nil {
				return types.Type{}, err
			}
		} else if secondIsSeries {
			secondElemType := *operandType.Elem
			if err := ctx.Resolver.EmitSeriesReverseArithmetic(
				ctx.Writer,
				ctx.WriterID,
				op,
				secondElemType,
			); err != nil {
				return types.Type{}, err
			}
			resultType = operandType
			elemType = secondElemType
			firstIsSeries = true
		} else {
			if err := ctx.Writer.WriteBinaryOpInferred(op, hintType); err != nil {
				return types.Type{}, err
			}
		}
	}
	if firstIsSeries {
		return resultType, nil
	}
	return hintType, nil
}

func compileBitwiseOrImpl(
	ctx context.Context[parser.IBitwiseOrExpressionContext],
) (types.Type, error) {
	xors := ctx.AST.AllBitwiseXorExpression()
	return compileBitwiseChain(
		ctx,
		len(xors),
		func(int) string { return "|" },
		func(i int, hint types.Type, hasHint bool) (types.Type, error) {
			child := context.Child(ctx, xors[i])
			if hasHint {
				child = child.WithHint(hint)
			}
			return compileBitwiseXor(child)
		},
	)
}

func compileBitwiseXorImpl(
	ctx context.Context[parser.IBitwiseXorExpressionContext],
) (types.Type, error) {
	ands := ctx.AST.AllBitwiseAndExpression()
	return compileBitwiseChain(
		ctx,
		len(ands),
		func(int) string { return "^" },
		func(i int, hint types.Type, hasHint bool) (types.Type, error) {
			child := context.Child(ctx, ands[i])
			if hasHint {
				child = child.WithHint(hint)
			}
			return compileBitwiseAnd(child)
		},
	)
}

func compileBitwiseAndImpl(
	ctx context.Context[parser.IBitwiseAndExpressionContext],
) (types.Type, error) {
	eqs := ctx.AST.AllEqualityExpression()
	return compileBitwiseChain(
		ctx,
		len(eqs),
		func(int) string { return "&" },
		func(i int, hint types.Type, hasHint bool) (types.Type, error) {
			child := context.Child(ctx, eqs[i])
			if hasHint {
				child = child.WithHint(hint)
			}
			return compileEquality(child)
		},
	)
}

func compileShiftImpl(
	ctx context.Context[parser.IShiftExpressionContext],
) (types.Type, error) {
	adds := ctx.AST.AllAdditiveExpression()
	var operators []string
	for _, child := range ctx.AST.GetChildren() {
		if termNode, ok := child.(antlr.TerminalNode); ok {
			switch termNode.GetSymbol().GetTokenType() {
			case parser.ArcLexerLSHIFT:
				operators = append(operators, "<<")
			case parser.ArcLexerRSHIFT:
				operators = append(operators, ">>")
			}
		}
	}
	return compileBitwiseChain(
		ctx,
		len(adds),
		func(i int) string { return operators[i] },
		func(i int, hint types.Type, hasHint bool) (types.Type, error) {
			child := context.Child(ctx, adds[i])
			if hasHint {
				child = child.WithHint(hint)
			}
			return compileAdditive(child)
		},
	)
}

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
		operandType, err := compileMultiplicative(
			context.Child(ctx, muls[i]).WithHint(operandHint),
		)
		if err != nil {
			return types.Type{}, err
		}

		secondIsSeries := operandType.Kind == types.KindSeries

		if firstIsSeries {
			isScalar := !secondIsSeries
			if err := ctx.Resolver.EmitSeriesArithmetic(
				ctx.Writer,
				ctx.WriterID,
				operators[i-1],
				elemType,
				isScalar,
			); err != nil {
				return types.Type{}, err
			}
		} else if secondIsSeries {
			secondElemType := *operandType.Elem
			op := operators[i-1]
			if err := ctx.Resolver.EmitSeriesReverseArithmetic(
				ctx.Writer,
				ctx.WriterID,
				op,
				secondElemType,
			); err != nil {
				return types.Type{}, err
			}
			resultType = operandType
			elemType = secondElemType
			firstIsSeries = true
		} else if hintType.Kind == types.KindString && operators[i-1] == "+" {
			ctx.Resolver.EmitStringConcat(ctx.Writer, ctx.WriterID)
		} else {
			if err = ctx.Writer.WriteBinaryOpInferred(
				operators[i-1],
				hintType,
			); err != nil {
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
		operandType, err := compilePower(
			context.Child(ctx, pows[i]).WithHint(operandHint),
		)
		if err != nil {
			return types.Type{}, err
		}

		secondIsSeries := operandType.Kind == types.KindSeries

		if firstIsSeries {
			isScalar := !secondIsSeries
			if err := ctx.Resolver.EmitSeriesArithmetic(
				ctx.Writer,
				ctx.WriterID,
				operators[i-1],
				elemType,
				isScalar,
			); err != nil {
				return types.Type{}, err
			}
		} else if secondIsSeries {
			secondElemType := *operandType.Elem
			op := operators[i-1]
			if err := ctx.Resolver.EmitSeriesReverseArithmetic(
				ctx.Writer,
				ctx.WriterID,
				op,
				secondElemType,
			); err != nil {
				return types.Type{}, err
			}
			resultType = operandType
			elemType = secondElemType
			firstIsSeries = true
		} else {
			if err = ctx.Writer.WriteBinaryOpInferred(
				operators[i-1],
				hintType,
			); err != nil {
				return types.Type{}, err
			}
		}
	}

	if firstIsSeries {
		return resultType, nil
	}
	return hintType, nil
}

func compileBinaryRelational(
	ctx context.Context[parser.IRelationalExpressionContext],
) (types.Type, error) {
	shifts := ctx.AST.AllShiftExpression()
	leftType, err := compileShift(context.Child(ctx, shifts[0]))
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

	_, err = compileShift(context.Child(ctx, shifts[1]).WithHint(operandHint))
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
		if err := ctx.Resolver.EmitSeriesComparison(
			ctx.Writer,
			ctx.WriterID,
			op,
			elemType,
		); err != nil {
			return types.Type{}, err
		}
		return types.Series(types.Bool()), nil
	}

	if err = ctx.Writer.WriteBinaryOpInferred(op, hintType); err != nil {
		return types.Type{}, err
	}
	return types.Bool(), nil
}

func compileBinaryEquality(
	ctx context.Context[parser.IEqualityExpressionContext],
) (types.Type, error) {
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
		if err := ctx.Resolver.EmitSeriesComparison(
			ctx.Writer,
			ctx.WriterID,
			op,
			elemType,
		); err != nil {
			return types.Type{}, err
		}
		return types.Series(types.Bool()), nil
	}

	if hintType.Kind == types.KindString {
		ctx.Resolver.EmitStringEqual(ctx.Writer, ctx.WriterID)
		if op == "!=" {
			ctx.Writer.WriteI32Eqz()
		}
		return types.Bool(), nil
	}

	if err = ctx.Writer.WriteBinaryOpInferred(op, hintType); err != nil {
		return types.Type{}, err
	}
	return types.Bool(), nil
}
