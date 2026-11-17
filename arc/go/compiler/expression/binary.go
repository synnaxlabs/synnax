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
		return primaryLiteralOnly(ctx, prim)
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

func primaryLiteralOnly[N antlr.ParserRuleContext](
	ctx context.Context[N],
	prim parser.IPrimaryExpressionContext,
) bool {
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

// arithmeticChain tracks the running type of a left-associative chain of arithmetic
// operators. Once a series operand appears, the rest of the chain is series math.
type arithmeticChain struct {
	result types.Type
	hint   types.Type
	elem   types.Type
	series bool
}

func newArithmeticChain(first types.Type) *arithmeticChain {
	c := &arithmeticChain{result: first, hint: first}
	if first.Kind == types.KindSeries {
		c.series = true
		c.elem = *first.Elem
	}
	if first.Kind == types.KindChan {
		c.hint = first.Unwrap()
	}
	return c
}

func (c *arithmeticChain) operandHint() types.Type {
	if c.series {
		return c.elem
	}
	return c.hint
}

func (c *arithmeticChain) resultType() types.Type {
	if c.series {
		return c.result
	}
	return c.hint
}

// emitArithmetic writes op applied to the chain so far and an operand of type
// operand, already on the stack.
func emitArithmetic[N antlr.ParserRuleContext](
	ctx context.Context[N],
	c *arithmeticChain,
	op string,
	operand types.Type,
) error {
	operandIsSeries := operand.Kind == types.KindSeries
	switch {
	case c.series:
		return ctx.Resolver.EmitSeriesArithmetic(
			ctx.Writer, ctx.WriterID, op, c.elem, !operandIsSeries,
		)
	case operandIsSeries:
		elem := *operand.Elem
		if err := ctx.Resolver.EmitSeriesReverseArithmetic(
			ctx.Writer, ctx.WriterID, op, elem,
		); err != nil {
			return err
		}
		c.result = operand
		c.elem = elem
		c.series = true
		return nil
	case c.hint.Kind == types.KindString && op == "+":
		ctx.Resolver.EmitStringConcat(ctx.Writer, ctx.WriterID)
		return nil
	default:
		return ctx.Writer.WriteArithmetic(op, c.hint)
	}
}

// operatorTokens returns the operators among node's terminal children, in order.
// symbolOf returns "" for a token that is not an operator.
func operatorTokens(node antlr.Tree, symbolOf func(tokenType int) string) []string {
	var operators []string
	for _, child := range node.GetChildren() {
		if termNode, ok := child.(antlr.TerminalNode); ok {
			if op := symbolOf(termNode.GetSymbol().GetTokenType()); op != "" {
				operators = append(operators, op)
			}
		}
	}
	return operators
}

func additiveSymbol(tokenType int) string {
	switch tokenType {
	case parser.ArcLexerPLUS:
		return "+"
	case parser.ArcLexerMINUS:
		return "-"
	default:
		return ""
	}
}

func multiplicativeSymbol(tokenType int) string {
	switch tokenType {
	case parser.ArcLexerSTAR:
		return "*"
	case parser.ArcLexerSLASH:
		return "/"
	case parser.ArcLexerPERCENT:
		return "%"
	default:
		return ""
	}
}

func compileBinaryAdditive(
	ctx context.Context[parser.IAdditiveExpressionContext],
) (types.Type, error) {
	muls := ctx.AST.AllMultiplicativeExpression()
	first, err := compileMultiplicative(
		ctx.Child(muls[0]).WithHint(firstOperandHint(ctx, muls)),
	)
	if err != nil {
		return types.Type{}, err
	}
	chain := newArithmeticChain(first)
	operators := operatorTokens(ctx.AST, additiveSymbol)
	for i := 1; i < len(muls); i++ {
		operand, err := compileMultiplicative(
			ctx.Child(muls[i]).WithHint(chain.operandHint()),
		)
		if err != nil {
			return types.Type{}, err
		}
		if err := emitArithmetic(ctx, chain, operators[i-1], operand); err != nil {
			return types.Type{}, err
		}
	}
	return chain.resultType(), nil
}

func compileBinaryMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) (types.Type, error) {
	unaries := ctx.AST.AllUnaryExpression()
	first, err := compileUnary(
		ctx.Child(unaries[0]).WithHint(firstOperandHint(ctx, unaries)),
	)
	if err != nil {
		return types.Type{}, err
	}
	chain := newArithmeticChain(first)
	operators := operatorTokens(ctx.AST, multiplicativeSymbol)
	for i := 1; i < len(unaries); i++ {
		operand, err := compileUnary(
			ctx.Child(unaries[i]).WithHint(chain.operandHint()),
		)
		if err != nil {
			return types.Type{}, err
		}
		if err := emitArithmetic(ctx, chain, operators[i-1], operand); err != nil {
			return types.Type{}, err
		}
	}
	return chain.resultType(), nil
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

	operandHint := hintType
	if isSeries {
		operandHint = elemType
	}

	_, err = compileAdditive(ctx.Child(adds[1]).WithHint(operandHint))
	if err != nil {
		return types.Type{}, err
	}
	var op string
	switch {
	case ctx.AST.LT(0) != nil:
		op = "<"
	case ctx.AST.GT(0) != nil:
		op = ">"
	case ctx.AST.LEQ(0) != nil:
		op = "<="
	case ctx.AST.GEQ(0) != nil:
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

	operandHint := hintType
	if isSeries {
		operandHint = elemType
	}

	_, err = compileRelational(ctx.Child(rels[1]).WithHint(operandHint))
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
