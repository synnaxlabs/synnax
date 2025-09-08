// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/synnaxlabs/arc/compiler/core"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func validateNonZeroArray[T any](exprs []T, opName string) []T {
	if len(exprs) == 0 {
		panic(errors.Newf("cannot compile an empty %s expression", opName))
	}
	return exprs
}

func validateNonZero(expr text.IUnaryExpressionContext, opName string) {
	if expr == nil {
		panic(errors.Newf("cannot compile an empty %s expression", opName))
	}
}

// Compile compiles an expression and returns its type
func Compile(
	ctx *core.Context,
	expr text.IExpressionContext,
	hint types.Type,
) (types.Type, error) {
	// Main dispatch based on expression type. Grammar builds expressions in layres
	// in order of precedence.
	// Compilation order:
	// Expression ->
	// LogicalOrExpression ->
	// LogicalAndExpression ->
	// EqualityExpression ->
	// RelationalExpression ->
	// AdditiveExpression ->
	// MultiplicativeExpression ->
	// PowerExpression ->
	// UnaryExpression ->
	// PostfixExpression ->
	// PrimaryExpression
	if expr == nil {
		return nil, errors.New("cannot compile a nil expression")
	}
	// Since Expression just wraps LogicalOrExpression, unwrap it
	if logicalOr := expr.LogicalOrExpression(); logicalOr != nil {
		return compileLogicalOr(ctx, logicalOr, hint)
	}
	return nil, errors.New("unknown expression type")
}

// compileLogicalOr handles || operations
func compileLogicalOr(
	ctx *core.Context,
	expr text.ILogicalOrExpressionContext,
	hint types.Type,
) (types.Type, error) {
	ands := validateNonZeroArray(expr.AllLogicalAndExpression(), "logical OR")
	if len(ands) == 1 {
		return compileLogicalAnd(ctx, ands[0], hint)
	}
	return compileLogicalOrImpl(ctx, expr, hint)
}

// compileLogicalAnd handles && operations
func compileLogicalAnd(
	ctx *core.Context,
	expr text.ILogicalAndExpressionContext,
	hint types.Type,
) (types.Type, error) {
	eqs := validateNonZeroArray(expr.AllEqualityExpression(), "logical AND")
	if len(eqs) == 1 {
		return compileEquality(ctx, eqs[0], hint)
	}
	return compileLogicalAndImpl(ctx, expr, hint)
}

// compileEquality handles == and != operations
func compileEquality(
	ctx *core.Context,
	expr text.IEqualityExpressionContext,
	hint types.Type,
) (types.Type, error) {
	rels := validateNonZeroArray(expr.AllRelationalExpression(), "equality")
	if len(rels) == 1 {
		return compileRelational(ctx, rels[0], hint)
	}
	return compileBinaryEquality(ctx, expr, hint)
}

// compileRelational handles <, >, <=, >= operations
func compileRelational(ctx *core.Context, expr text.IRelationalExpressionContext, hint types.Type) (types.Type, error) {
	adds := validateNonZeroArray(expr.AllAdditiveExpression(), "relational")
	if len(adds) == 1 {
		return compileAdditive(ctx, adds[0], hint)
	}
	return compileBinaryRelational(ctx, expr, hint)
}

// compileAdditive handles + and - operations.
func compileAdditive(ctx *core.Context, expr text.IAdditiveExpressionContext, hint types.Type) (types.Type, error) {
	muls := validateNonZeroArray(expr.AllMultiplicativeExpression(), "additive")
	if len(muls) == 1 {
		return compileMultiplicative(ctx, muls[0], hint)
	}
	return compileBinaryAdditive(ctx, expr, hint)
}

// compileMultiplicative handles *, /, and % operations
func compileMultiplicative(
	ctx *core.Context,
	expr text.IMultiplicativeExpressionContext,
	hint types.Type,
) (types.Type, error) {
	pows := validateNonZeroArray(expr.AllPowerExpression(), "multiplicative")
	if len(pows) == 1 {
		return compilePower(ctx, pows[0], hint)
	}
	return compileBinaryMultiplicative(ctx, expr, hint)
}

// compilePower handles ^ operations
func compilePower(ctx *core.Context, expr text.IPowerExpressionContext, hint types.Type) (types.Type, error) {
	unary := expr.UnaryExpression()
	validateNonZero(unary, "power")
	if expr.CARET() != nil && expr.PowerExpression() != nil {
		// TODO: Implement exponentiation (needs host function)
		return compileUnary(ctx, unary, hint)
	}
	return compileUnary(ctx, unary, hint)
}

// compilePostfix handles array indexing, slicing, and function calls
func compilePostfix(ctx *core.Context, expr text.IPostfixExpressionContext, hint types.Type) (types.Type, error) {
	primary := expr.PrimaryExpression()
	if primary == nil {
		return nil, errors.New("empty postfix expression")
	}
	primaryType, err := compilePrimary(ctx, primary, hint)
	if err != nil {
		return nil, err
	}
	return primaryType, nil
}

// compilePrimary handles literals, identifiers, parenthesized expressions, etc.
func compilePrimary(ctx *core.Context, expr text.IPrimaryExpressionContext, hint types.Type) (types.Type, error) {
	if lit := expr.Literal(); lit != nil {
		return compileLiteral(ctx, lit, hint)
	}
	if expr.IDENTIFIER() != nil {
		return compileIdentifier(ctx, expr.IDENTIFIER().GetText())
	}
	if expr.LPAREN() != nil && expr.Expression() != nil {
		return Compile(ctx, expr.Expression(), hint)
	}
	if cast := expr.TypeCast(); cast != nil {
		return compileTypeCast(ctx, cast)
	}
	if builtin := expr.BuiltinFunction(); builtin != nil {
		return nil, errors.New("builtin functions not yet implemented")
	}
	return nil, errors.New("unknown primary expression")
}
