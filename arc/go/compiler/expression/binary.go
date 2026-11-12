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
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// firstOperandHint keeps an outer hint from retyping a literal-only first operand.
func firstOperandHint[N, T antlr.ParserRuleContext](
	ctx context.Context[N],
	operands []T,
) types.Type {
	if !literalOnly(ctx, operands[0]) {
		return ctx.Hint
	}
	if lo.EveryBy(operands[1:], func(op T) bool { return literalOnly(ctx, op) }) {
		return ctx.Hint
	}
	resolved, ok := ctx.TypeMap[ctx.AST]
	if !ok || resolved.Kind == types.KindVariable {
		return ctx.Hint
	}
	return resolved
}

// literalOnly reports whether every leaf of node is a numeric literal or a module
// constant.
func literalOnly[N antlr.ParserRuleContext](
	ctx context.Context[N],
	node antlr.Tree,
) bool {
	if prim, ok := node.(parser.IPrimaryExpressionContext); ok {
		if lit := prim.Literal(); lit != nil {
			return lit.NumericLiteral() != nil
		}
		if qid := prim.QualifiedIdentifier(); qid != nil {
			head, tail := parser.QualifiedNameParts(qid)
			scope, err := ctx.Scope.Resolve(ctx, head)
			if err != nil {
				return false
			}
			sym, err := scope.Resolve(ctx, tail)
			return err == nil && sym.Kind == symbol.KindConstant
		}
		if prim.LPAREN() == nil || prim.Expression() == nil {
			return false
		}
		return literalOnly(ctx, prim.Expression())
	}
	for _, child := range node.GetChildren() {
		if _, ok := child.(antlr.TerminalNode); ok {
			continue
		}
		if !literalOnly(ctx, child) {
			return false
		}
	}
	return true
}

// compileOperand compiles operand and makes it meet the value on the stack, of type
// left, in one unit. Two units of one dimension meet in the smaller one, and a value
// with no unit takes the unit of the other. It returns the type of the operand and the
// unit of both values.
func compileOperand[N, O antlr.ParserRuleContext](
	ctx context.Context[N],
	left types.Type,
	operand O,
	compile func(context.Context[O]) (types.Type, error),
) (types.Type, *types.Unit, error) {
	hint := left.Unwrap()
	unit := hint.Unit
	if next := ctx.TypeMap[operand].Unwrap().Unit; !types.UnitsIncompatible(
		unit,
		next,
	) {
		unit = types.CommonUnit(unit, next)
	}
	if types.UnitsConvertible(hint.Unit, unit) {
		if err := EmitUnitScale(ctx, left, unit); err != nil {
			return types.Type{}, nil, err
		}
	}
	hint.Unit = unit
	t, err := compile(ctx.Child(operand).WithHint(hint))
	if err != nil {
		return types.Type{}, nil, err
	}
	if from := t.Unwrap().Unit; types.UnitsConvertible(from, unit) {
		if from.Scale < unit.Scale {
			return types.Type{}, nil, errors.Newf(
				"operand in %s was not analyzed before a value in %s",
				from.Name, unit.Name,
			)
		}
		if err = EmitUnitScale(ctx, t, unit); err != nil {
			return types.Type{}, nil, err
		}
	}
	return t, unit, nil
}

func compileBinaryAdditive(
	ctx context.Context[parser.IAdditiveExpressionContext],
) (types.Type, error) {
	muls := ctx.AST.AllMultiplicativeExpression()
	resultType, err := compileMultiplicative(
		ctx.Child(muls[0]).WithHint(firstOperandHint(ctx, muls)),
	)
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
		left := hintType
		if firstIsSeries {
			left = resultType
		}
		operandType, unit, err := compileOperand(
			ctx, left, muls[i], compileMultiplicative,
		)
		if err != nil {
			return types.Type{}, err
		}
		hintType.Unit, elemType.Unit = unit, unit
		if firstIsSeries {
			resultType = types.Series(elemType)
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
			secondElemType.Unit = unit
			op := operators[i-1]
			if err := ctx.Resolver.EmitSeriesReverseArithmetic(
				ctx.Writer,
				ctx.WriterID,
				op,
				secondElemType,
			); err != nil {
				return types.Type{}, err
			}
			resultType = types.Series(secondElemType)
			elemType = secondElemType
			firstIsSeries = true
		} else if hintType.Kind == types.KindString && operators[i-1] == "+" {
			ctx.Resolver.EmitStringConcat(ctx.Writer, ctx.WriterID)
		} else {
			if err = ctx.Writer.WriteArithmetic(
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
	unaries := ctx.AST.AllUnaryExpression()

	resultType, err := compileUnary(
		ctx.Child(unaries[0]).WithHint(firstOperandHint(ctx, unaries)),
	)
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

	for i := 1; i < len(unaries); i++ {
		left := hintType
		if firstIsSeries {
			left = resultType
		}
		operandType, unit, err := compileOperand(ctx, left, unaries[i], compileUnary)
		if err != nil {
			return types.Type{}, err
		}
		if operators[i-1] != "%" {
			unit = types.ProductUnit(left.Unwrap().Unit, operandType.Unwrap().Unit)
		}
		hintType.Unit, elemType.Unit = unit, unit
		if firstIsSeries {
			resultType = types.Series(elemType)
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
			secondElemType.Unit = unit
			op := operators[i-1]
			if err := ctx.Resolver.EmitSeriesReverseArithmetic(
				ctx.Writer,
				ctx.WriterID,
				op,
				secondElemType,
			); err != nil {
				return types.Type{}, err
			}
			resultType = types.Series(secondElemType)
			elemType = secondElemType
			firstIsSeries = true
		} else {
			if err = ctx.Writer.WriteArithmetic(
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
	adds := ctx.AST.AllAdditiveExpression()
	// A comparison gives a bool, so an outer hint says nothing about its operands.
	leftType, err := compileAdditive(ctx.Child(adds[0]).WithHint(types.Type{}))
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

	left := hintType
	if isSeries {
		left = leftType
	}
	if _, _, err = compileOperand(ctx, left, adds[1], compileAdditive); err != nil {
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
	// An equality gives a bool, so an outer hint says nothing about its operands.
	leftType, err := compileRelational(ctx.Child(rels[0]).WithHint(types.Type{}))
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

	left := hintType
	if isSeries {
		left = leftType
	}
	if _, _, err = compileOperand(ctx, left, rels[1], compileRelational); err != nil {
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
