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
	resolved, ok := ctx.TypeMap[ctx.AST]
	if !ok {
		return ctx.Hint
	}
	if resolved.Kind == types.KindVariable ||
		lo.EveryBy(operands[1:], func(op T) bool { return literalOnly(ctx, op) }) {
		hint := ctx.Hint
		if hint.IsNumeric() {
			hint.Unit = resolved.Unit
		}
		return hint
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

// compileOperand compiles operand in the unit that it and the value on the stack, of
// type left, meet in. It returns the type of the operand and that unit.
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

// arithmeticOperators maps the token of each arithmetic operator to its text.
var arithmeticOperators = map[int]string{
	parser.ArcLexerPLUS:    "+",
	parser.ArcLexerMINUS:   "-",
	parser.ArcLexerSTAR:    "*",
	parser.ArcLexerSLASH:   "/",
	parser.ArcLexerPERCENT: "%",
}

// compileArithmetic compiles operands, which the operators of ctx.AST join, from left
// to right.
func compileArithmetic[N, O antlr.ParserRuleContext](
	ctx context.Context[N],
	operands []O,
	compile func(context.Context[O]) (types.Type, error),
) (types.Type, error) {
	resultType, err := compile(
		ctx.Child(operands[0]).WithHint(firstOperandHint(ctx, operands)),
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
		if term, ok := child.(antlr.TerminalNode); ok {
			if op, ok := arithmeticOperators[term.GetSymbol().GetTokenType()]; ok {
				operators = append(operators, op)
			}
		}
	}
	for i, operand := range operands[1:] {
		op, left := operators[i], hintType
		if firstIsSeries {
			left = resultType
		}
		operandType, unit, err := compileOperand(ctx, left, operand, compile)
		if err != nil {
			return types.Type{}, err
		}
		if op == "*" || op == "/" {
			unit = types.ProductUnit(left.Unwrap().Unit, operandType.Unwrap().Unit)
		}
		hintType.Unit, elemType.Unit = unit, unit
		if firstIsSeries {
			resultType = types.Series(elemType)
		}
		secondIsSeries := operandType.Kind == types.KindSeries
		switch {
		case firstIsSeries:
			err = ctx.Resolver.EmitSeriesArithmetic(
				ctx.Writer, ctx.WriterID, op, elemType, !secondIsSeries,
			)
		case secondIsSeries:
			elemType = *operandType.Elem
			elemType.Unit = unit
			err = ctx.Resolver.EmitSeriesReverseArithmetic(
				ctx.Writer, ctx.WriterID, op, elemType,
			)
			resultType, firstIsSeries = types.Series(elemType), true
		case hintType.Kind == types.KindString && op == "+":
			ctx.Resolver.EmitStringConcat(ctx.Writer, ctx.WriterID)
		default:
			err = ctx.Writer.WriteArithmetic(op, hintType)
		}
		if err != nil {
			return types.Type{}, err
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
	return compileComparison(ctx, ctx.AST.AllAdditiveExpression(), op, compileAdditive)
}

func compileBinaryEquality(
	ctx context.Context[parser.IEqualityExpressionContext],
) (types.Type, error) {
	var op string
	if ctx.AST.EQ(0) != nil {
		op = "=="
	} else if ctx.AST.NEQ(0) != nil {
		op = "!="
	}
	return compileComparison(
		ctx, ctx.AST.AllRelationalExpression(), op, compileRelational,
	)
}

// compileComparison compiles the first two of operands and compares them with op.
func compileComparison[N, O antlr.ParserRuleContext](
	ctx context.Context[N],
	operands []O,
	op string,
	compile func(context.Context[O]) (types.Type, error),
) (types.Type, error) {
	// A comparison gives a bool, so an outer hint says nothing about its operands.
	leftType, err := compile(ctx.Child(operands[0]).WithHint(types.Type{}))
	if err != nil {
		return types.Type{}, err
	}
	hintType := leftType
	if leftType.Kind == types.KindChan {
		hintType = leftType.Unwrap()
	}
	isSeries := leftType.Kind == types.KindSeries
	left := hintType
	if isSeries {
		left = leftType
	}
	if _, _, err = compileOperand(ctx, left, operands[1], compile); err != nil {
		return types.Type{}, err
	}
	if isSeries {
		if err = ctx.Resolver.EmitSeriesComparison(
			ctx.Writer, ctx.WriterID, op, *leftType.Elem,
		); err != nil {
			return types.Type{}, err
		}
		return types.Series(types.Bool()), nil
	}
	if hintType.Kind == types.KindString && (op == "==" || op == "!=") {
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
