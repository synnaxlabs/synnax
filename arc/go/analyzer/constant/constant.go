// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package constant reports integer overflow and integer division by zero in
// expressions made only of integer literals.
package constant

import (
	"math/big"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// Check adds a diagnostic for each constant integer expression in ctx.AST that
// overflows its type or divides by zero. It reads types from ctx.TypeMap, so run it
// after type unification. It only reports: the runtime computes every value.
func Check[T antlr.ParserRuleContext](ctx context.Context[T]) {
	e := evaluator{typeMap: ctx.TypeMap}
	walk(ctx.Diagnostics, e, ctx.AST)
}

func walk(diags *diagnostics.Diagnostics, e evaluator, node antlr.Tree) {
	r := e.eval(node)
	if r.diag != nil {
		diags.Add(*r.diag)
		return
	}
	if r.constant() {
		return
	}
	for _, child := range node.GetChildren() {
		walk(diags, e, child)
	}
}

// result is a constant value, a diagnostic, or neither when the expression is not a
// constant integer expression.
type result struct {
	value *big.Int
	typ   types.Type
	diag  *diagnostics.Diagnostic
}

func (r result) constant() bool { return r.value != nil }

type evaluator struct {
	typeMap map[antlr.ParserRuleContext]types.Type
}

func (e evaluator) eval(node antlr.Tree) result {
	switch n := node.(type) {
	case parser.IExpressionContext:
		return e.eval(n.LogicalOrExpression())
	case parser.ILogicalOrExpressionContext:
		return single(e, n.AllLogicalAndExpression())
	case parser.ILogicalAndExpressionContext:
		return single(e, n.AllEqualityExpression())
	case parser.IEqualityExpressionContext:
		return single(e, n.AllRelationalExpression())
	case parser.IRelationalExpressionContext:
		return single(e, n.AllAdditiveExpression())
	case parser.IAdditiveExpressionContext:
		return e.binary(n)
	case parser.IMultiplicativeExpressionContext:
		return e.binary(n)
	case parser.IUnaryExpressionContext:
		return e.unary(n)
	case parser.IPowerExpressionContext:
		return e.power(n)
	case parser.IPostfixExpressionContext:
		if n.GetChildCount() != 1 {
			return result{}
		}
		return e.eval(n.PrimaryExpression())
	case parser.IPrimaryExpressionContext:
		return e.primary(n)
	}
	return result{}
}

// single evaluates the only operand of an operator level, or reports nothing for a
// comparison or logical operation.
func single[N antlr.Tree](e evaluator, operands []N) result {
	if len(operands) != 1 {
		return result{}
	}
	return e.eval(operands[0])
}

func (e evaluator) binary(node antlr.ParserRuleContext) result {
	var (
		operands []result
		ops      []int
	)
	for _, child := range node.GetChildren() {
		if term, ok := child.(antlr.TerminalNode); ok {
			ops = append(ops, term.GetSymbol().GetTokenType())
			continue
		}
		r := e.eval(child)
		if r.diag != nil {
			return r
		}
		operands = append(operands, r)
	}
	if len(ops) != len(operands)-1 {
		return result{}
	}
	for _, r := range operands {
		if !r.constant() {
			return result{}
		}
	}
	if len(operands) == 1 {
		return operands[0]
	}
	t, ok := e.typeMap[node]
	if !ok || !isInteger(t) {
		return result{}
	}
	acc := new(big.Int).Set(operands[0].value)
	for i, op := range ops {
		rhs := operands[i+1].value
		switch op {
		case parser.ArcParserPLUS:
			acc.Add(acc, rhs)
		case parser.ArcParserMINUS:
			acc.Sub(acc, rhs)
		case parser.ArcParserSTAR:
			acc.Mul(acc, rhs)
		case parser.ArcParserSLASH, parser.ArcParserPERCENT:
			if rhs.Sign() == 0 {
				return divisionByZero(node, op)
			}
			if op == parser.ArcParserSLASH {
				acc.Quo(acc, rhs)
			} else {
				acc.Rem(acc, rhs)
			}
		}
		if !fits(acc, t) {
			return overflow(node, acc.String(), t)
		}
	}
	return result{value: acc, typ: t}
}

func (e evaluator) unary(node parser.IUnaryExpressionContext) result {
	if power := node.PowerExpression(); power != nil {
		return e.eval(power)
	}
	operand := node.UnaryExpression()
	// A negated literal is one signed literal, so the minimum of each type fits.
	if node.MINUS() != nil && operand.UnaryExpression() == nil {
		if lit := parser.GetLiteralNode(operand); lit != nil {
			return e.literal(node, lit, true)
		}
	}
	inner := e.eval(operand)
	if inner.diag != nil || !inner.constant() || node.MINUS() == nil {
		return result{diag: inner.diag}
	}
	v := new(big.Int).Neg(inner.value)
	if !fits(v, inner.typ) {
		return overflow(node, v.String(), inner.typ)
	}
	return result{value: v, typ: inner.typ}
}

// maxExponent bounds the exponents evaluated exactly. A base of magnitude two or more
// raised past it overflows every integer type.
const maxExponent = 64

func (e evaluator) power(node parser.IPowerExpressionContext) result {
	base := e.eval(node.PostfixExpression())
	if node.CARET() == nil || base.diag != nil {
		return base
	}
	exp := e.eval(node.UnaryExpression())
	if exp.diag != nil {
		return exp
	}
	if !base.constant() || !exp.constant() || exp.value.Sign() < 0 {
		return result{}
	}
	if new(big.Int).Abs(base.value).Cmp(big.NewInt(1)) > 0 &&
		exp.value.Cmp(big.NewInt(maxExponent)) > 0 {
		return overflow(node, base.value.String()+"^"+exp.value.String(), base.typ)
	}
	v := new(big.Int).Exp(base.value, exp.value, nil)
	if !fits(v, base.typ) {
		return overflow(node, v.String(), base.typ)
	}
	return result{value: v, typ: base.typ}
}

func (e evaluator) primary(node parser.IPrimaryExpressionContext) result {
	if expr := node.Expression(); expr != nil {
		return e.eval(expr)
	}
	if cast := node.TypeCast(); cast != nil {
		return e.cast(cast)
	}
	if lit := node.Literal(); lit != nil {
		return e.literal(lit, lit, false)
	}
	return result{}
}

// cast passes an inner constant through a cast only when the target type holds it
// unchanged. Other casts convert at runtime and end the constant expression.
func (e evaluator) cast(node parser.ITypeCastContext) result {
	inner := e.eval(node.Expression())
	if inner.diag != nil || !inner.constant() {
		return result{diag: inner.diag}
	}
	t, err := atypes.InferFromTypeContext(node.Type_())
	if err != nil || !isInteger(t) || !fits(inner.value, t) {
		return result{}
	}
	return result{value: inner.value, typ: t}
}

// literal evaluates lit, negated when negate is true, and reports an overflow at node.
func (e evaluator) literal(
	node antlr.ParserRuleContext,
	lit parser.ILiteralContext,
	negate bool,
) result {
	num := lit.NumericLiteral()
	if num == nil || num.INTEGER_LITERAL() == nil || num.IDENTIFIER() != nil {
		return result{}
	}
	t, ok := e.typeMap[lit]
	if !ok || !isInteger(t) {
		return result{}
	}
	v, ok := new(big.Int).SetString(num.INTEGER_LITERAL().GetText(), 10)
	if !ok {
		return result{}
	}
	if (num.MINUS() != nil) != negate {
		v.Neg(v)
	}
	if !fits(v, t) {
		return overflow(node, v.String(), t)
	}
	return result{value: v, typ: t}
}

func isInteger(t types.Type) bool {
	return t.IsSignedInteger() || t.IsUnsignedInteger()
}

// fits reports whether v is in the range of the integer type t.
func fits(v *big.Int, t types.Type) bool {
	bits := uint(t.Density() * 8)
	lo, hi := new(big.Int), new(big.Int).Lsh(big.NewInt(1), bits)
	if t.IsSignedInteger() {
		hi.Rsh(hi, 1)
		lo.Neg(hi)
	}
	hi.Sub(hi, big.NewInt(1))
	return v.Cmp(lo) >= 0 && v.Cmp(hi) <= 0
}

func overflow(node antlr.ParserRuleContext, value string, t types.Type) result {
	d := diagnostics.Errorf(node, "constant %s overflows %s", value, t).
		WithCode(codes.ConstantOverflow)
	return result{diag: &d}
}

func divisionByZero(node antlr.ParserRuleContext, op int) result {
	name := "division"
	if op == parser.ArcParserPERCENT {
		name = "modulo"
	}
	d := diagnostics.Errorf(node, "integer %s by zero", name).
		WithCode(codes.DivisionByZero)
	return result{diag: &d}
}
