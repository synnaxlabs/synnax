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
	"github.com/synnaxlabs/arc/compiler/context"
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

func validateNonZero(expr parser.IUnaryExpressionContext, opName string) {
	if expr == nil {
		panic(errors.Newf("cannot compile an empty %s expression", opName))
	}
}

// Compile compiles an expression and returns its type
func Compile(
	ctx context.Context[parser.IExpressionContext],
) (types.Type, error) {
	// Main dispatch based on expression type. Grammar builds expressions in layers
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
	if ctx.AST == nil {
		return nil, errors.New("cannot compile a nil expression")
	}
	// Since Expression just wraps LogicalOrExpression, unwrap it
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		return compileLogicalOr(context.Child(ctx, logicalOr))
	}
	return nil, errors.New("unknown expression type")
}

// compileLogicalOr handles || operations
func compileLogicalOr(
	ctx context.Context[parser.ILogicalOrExpressionContext],
) (types.Type, error) {
	ands := validateNonZeroArray(ctx.AST.AllLogicalAndExpression(), "logical OR")
	if len(ands) == 1 {
		return compileLogicalAnd(context.Child(ctx, ands[0]))
	}
	return compileLogicalOrImpl(ctx)
}

// compileLogicalAnd handles && operations
func compileLogicalAnd(
	ctx context.Context[parser.ILogicalAndExpressionContext],
) (types.Type, error) {
	eqs := validateNonZeroArray(ctx.AST.AllEqualityExpression(), "logical AND")
	if len(eqs) == 1 {
		return compileEquality(context.Child(ctx, eqs[0]))
	}
	return compileLogicalAndImpl(ctx)
}

// compileEquality handles == and != operations
func compileEquality(
	ctx context.Context[parser.IEqualityExpressionContext],
) (types.Type, error) {
	rels := validateNonZeroArray(ctx.AST.AllRelationalExpression(), "equality")
	if len(rels) == 1 {
		return compileRelational(context.Child(ctx, rels[0]))
	}
	return compileBinaryEquality(ctx)
}

// compileRelational handles <, >, <=, >= operations
func compileRelational(ctx context.Context[parser.IRelationalExpressionContext]) (types.Type, error) {
	adds := validateNonZeroArray(ctx.AST.AllAdditiveExpression(), "relational")
	if len(adds) == 1 {
		return compileAdditive(context.Child(ctx, adds[0]))
	}
	return compileBinaryRelational(ctx)
}

// compileAdditive handles + and - operations.
func compileAdditive(
	ctx context.Context[parser.IAdditiveExpressionContext],
) (types.Type, error) {
	muls := validateNonZeroArray(ctx.AST.AllMultiplicativeExpression(), "additive")
	if len(muls) == 1 {
		return compileMultiplicative(context.Child(ctx, muls[0]))
	}
	return compileBinaryAdditive(ctx)
}

// compileMultiplicative handles *, /, and % operations
func compileMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) (types.Type, error) {
	pows := validateNonZeroArray(ctx.AST.AllPowerExpression(), "multiplicative")
	if len(pows) == 1 {
		return compilePower(context.Child(ctx, pows[0]))
	}
	return compileBinaryMultiplicative(ctx)
}

// compilePower handles ^ operations
func compilePower(
	ctx context.Context[parser.IPowerExpressionContext],
) (types.Type, error) {
	unary := ctx.AST.UnaryExpression()
	validateNonZero(unary, "power")
	if ctx.AST.CARET() != nil && ctx.AST.PowerExpression() != nil {
		// TODO: Implement exponentiation (needs host function)
		return compileUnary(context.Child(ctx, unary))
	}
	return compileUnary(context.Child(ctx, unary))
}

// compilePostfix handles array indexing, slicing, and function calls
func compilePostfix(ctx context.Context[parser.IPostfixExpressionContext]) (types.Type, error) {
	primary := ctx.AST.PrimaryExpression()
	if primary == nil {
		return nil, errors.New("empty postfix expression")
	}
	primaryType, err := compilePrimary(context.Child(ctx, primary))
	if err != nil {
		return nil, err
	}
	return primaryType, nil
}

// compilePrimary handles literals, identifiers, parenthesized expressions, etc.
func compilePrimary(ctx context.Context[parser.IPrimaryExpressionContext]) (types.Type, error) {
	if lit := ctx.AST.Literal(); lit != nil {
		return compileLiteral(context.Child(ctx, lit))
	}
	if ctx.AST.IDENTIFIER() != nil {
		return compileIdentifier(ctx, ctx.AST.IDENTIFIER().GetText())
	}
	if ctx.AST.LPAREN() != nil && ctx.AST.Expression() != nil {
		return Compile(context.Child(ctx, ctx.AST.Expression()))
	}
	if cast := ctx.AST.TypeCast(); cast != nil {
		return compileTypeCast(context.Child(ctx, cast))
	}
	if builtin := ctx.AST.BuiltinFunction(); builtin != nil {
		return nil, errors.New("builtin functions not yet implemented")
	}
	return nil, errors.New("unknown primary expression")
}
