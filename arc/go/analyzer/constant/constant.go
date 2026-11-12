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

// Check adds a diagnostic for each constant integer operation in ctx.AST, format
// string placeholders included, that overflows its type or divides by zero. It reads
// types from ctx.TypeMap, so run it after type unification. It only reports: the
// runtime computes every value.
func Check[T antlr.ParserRuleContext](ctx context.Context[T]) {
	w := walker{
		diags:      ctx.Diagnostics,
		fmtStrings: ctx.FmtStrings,
		e: evaluator{
			typeMap: ctx.TypeMap,
			results: make(map[antlr.Tree]result),
		},
	}
	w.walk(ctx.AST)
}

type walker struct {
	e          evaluator
	diags      *diagnostics.Diagnostics
	fmtStrings parser.FmtStrings
}

// walk reports each operation that fails on its own operands, once, and descends
// into every subexpression that is not a valid constant.
func (w walker) walk(node antlr.Tree) {
	if term, ok := node.(antlr.TerminalNode); ok {
		for _, seg := range w.fmtStrings[term] {
			if seg.Expr != nil {
				w.walk(seg.Expr)
			}
		}
		return
	}
	r := w.e.eval(node)
	if r.diag != nil {
		w.diags.Add(*r.diag)
		return
	}
	if r.constant() {
		return
	}
	for _, child := range node.GetChildren() {
		w.walk(child)
	}
}

// result is a constant value, a failure, or neither when the expression is not a
// constant integer expression. diag is set only on the node whose operation failed;
// its ancestors carry failed alone.
type result struct {
	value  *big.Int
	typ    types.Type
	diag   *diagnostics.Diagnostic
	failed bool
}

func (r result) constant() bool { return r.value != nil }

var propagated = result{failed: true}

type evaluator struct {
	typeMap map[antlr.ParserRuleContext]types.Type
	results map[antlr.Tree]result
}

func (e evaluator) eval(node antlr.Tree) result {
	if r, ok := e.results[node]; ok {
		return r
	}
	r := e.evalNode(node)
	e.results[node] = r
	return r
}

func (e evaluator) evalNode(node antlr.Tree) result {
	switch n := node.(type) {
	case parser.IExpressionContext:
		return e.forward(n.LogicalOrExpression())
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
		return e.forward(n.PrimaryExpression())
	case parser.IPrimaryExpressionContext:
		return e.primary(n)
	}
	return result{}
}

// forward evaluates node as the value of its parent, so a failure in node does not
// fail the parent a second time.
func (e evaluator) forward(node antlr.Tree) result {
	r := e.eval(node)
	if r.failed {
		return propagated
	}
	return r
}

// single evaluates the only operand of an operator level. A comparison or logical
// operation is not a constant integer expression.
func single[N antlr.Tree](e evaluator, operands []N) result {
	if len(operands) != 1 {
		return result{}
	}
	return e.forward(operands[0])
}

func (e evaluator) binary(node antlr.ParserRuleContext) result {
	var (
		operands []result
		ops      []int
		failed   bool
	)
	for _, child := range node.GetChildren() {
		if term, ok := child.(antlr.TerminalNode); ok {
			ops = append(ops, term.GetSymbol().GetTokenType())
			continue
		}
		r := e.eval(child)
		failed = failed || r.failed
		operands = append(operands, r)
	}
	if failed {
		return propagated
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
		return e.forward(power)
	}
	operand := node.UnaryExpression()
	// A negated literal is one signed literal, so the minimum of each type fits.
	if node.MINUS() != nil && operand.UnaryExpression() == nil {
		if lit := parser.GetLiteralNode(operand); lit != nil {
			return e.literal(node, lit, true)
		}
	}
	inner := e.eval(operand)
	if inner.failed {
		return propagated
	}
	if !inner.constant() || node.MINUS() == nil {
		return result{}
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
	if node.CARET() == nil {
		return e.forward(node.PostfixExpression())
	}
	base, exp := e.eval(node.PostfixExpression()), e.eval(node.UnaryExpression())
	if base.failed || exp.failed {
		return propagated
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
		return e.forward(expr)
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
// unchanged. Other casts convert at runtime and end the constant expression. A bare
// literal that fits the target type is read as that type, as the runtime does.
func (e evaluator) cast(node parser.ITypeCastContext) result {
	expr := node.Expression()
	t, err := atypes.InferFromTypeContext(node.Type_())
	integer := err == nil && isInteger(t)
	if lit, negate, ok := signedLiteral(expr); integer && ok {
		if v, ok := literalValue(lit, negate); ok && fits(v, t) {
			return result{value: v, typ: t}
		}
	}
	inner := e.eval(expr)
	if !integer || !inner.constant() || !fits(inner.value, t) {
		return result{failed: inner.failed}
	}
	return result{value: inner.value, typ: t}
}

// signedLiteral returns the literal of expr when expr is a literal or a literal with
// one minus sign, and whether the sign is present.
func signedLiteral(
	expr parser.IExpressionContext,
) (lit parser.ILiteralContext, negate, ok bool) {
	lit = parser.GetLiteral(expr)
	if lit == nil {
		return nil, false, false
	}
	switch expr.GetText() {
	case lit.GetText():
		return lit, false, true
	case "-" + lit.GetText():
		return lit, true, true
	}
	return nil, false, false
}

// literal evaluates lit, negated when negate is true, and reports an overflow at node.
func (e evaluator) literal(
	node antlr.ParserRuleContext,
	lit parser.ILiteralContext,
	negate bool,
) result {
	t, ok := e.typeMap[lit]
	if !ok || !isInteger(t) {
		return result{}
	}
	v, ok := literalValue(lit, negate)
	if !ok {
		return result{}
	}
	if !fits(v, t) {
		return overflow(node, v.String(), t)
	}
	return result{value: v, typ: t}
}

// literalValue returns the value of an integer literal without a unit.
func literalValue(lit parser.ILiteralContext, negate bool) (*big.Int, bool) {
	num := lit.NumericLiteral()
	if num == nil || num.INTEGER_LITERAL() == nil || num.IDENTIFIER() != nil {
		return nil, false
	}
	v, ok := new(big.Int).SetString(num.INTEGER_LITERAL().GetText(), 10)
	if !ok {
		return nil, false
	}
	if (num.MINUS() != nil) != negate {
		v.Neg(v)
	}
	return v, true
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
	d := codes.ConstantOverflow.At(node, value, t)
	return result{diag: &d, failed: true}
}

func divisionByZero(node antlr.ParserRuleContext, op int) result {
	name := "division"
	if op == parser.ArcParserPERCENT {
		name = "modulo"
	}
	d := codes.DivisionByZero.At(node, name)
	return result{diag: &d, failed: true}
}
