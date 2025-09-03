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
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// Compiler compiles expressions to WASM bytecode
type Compiler struct {
	ctx     *compiler.Context
	encoder *wasm.Encoder
}

// NewCompiler creates a new expression compiler
func NewCompiler(ctx *compiler.Context) *Compiler {
	return &Compiler{ctx: ctx, encoder: wasm.NewEncoder()}
}

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
func (e *Compiler) Compile(expr parser.IExpressionContext) (types.Type, error) {
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
		return e.compileLogicalOr(logicalOr)
	}
	return nil, errors.New("unknown expression type")
}

// compileLogicalOr handles || operations
func (e *Compiler) compileLogicalOr(expr parser.ILogicalOrExpressionContext) (types.Type, error) {
	ands := validateNonZeroArray(expr.AllLogicalAndExpression(), "logical OR")
	if len(ands) == 1 {
		return e.compileLogicalAnd(ands[0])
	}
	return e.compileLogicalAnd(ands[0])
}

// compileLogicalAnd handles && operations
func (e *Compiler) compileLogicalAnd(expr parser.ILogicalAndExpressionContext) (types.Type, error) {
	eqs := validateNonZeroArray(expr.AllEqualityExpression(), "logical AND")
	if len(eqs) == 1 {
		return e.compileEquality(eqs[0])
	}
	return e.compileEquality(eqs[0])
}

// compileEquality handles == and != operations
func (e *Compiler) compileEquality(expr parser.IEqualityExpressionContext) (types.Type, error) {
	rels := validateNonZeroArray(expr.AllRelationalExpression(), "equality")
	if len(rels) == 1 {
		return e.compileRelational(rels[0])
	}
	return e.compileBinaryEquality(expr)
}

// compileRelational handles <, >, <=, >= operations
func (e *Compiler) compileRelational(expr parser.IRelationalExpressionContext) (types.Type, error) {
	adds := validateNonZeroArray(expr.AllAdditiveExpression(), "relational")
	if len(adds) == 1 {
		return e.compileAdditive(adds[0])
	}
	return e.compileBinaryRelational(expr)
}

// compileAdditive handles + and - operations.
func (e *Compiler) compileAdditive(expr parser.IAdditiveExpressionContext) (types.Type, error) {
	muls := validateNonZeroArray(expr.AllMultiplicativeExpression(), "additive")
	if len(muls) == 1 {
		return e.compileMultiplicative(muls[0])
	}
	return e.compileBinaryAdditive(expr)
}

// compileMultiplicative handles *, /, and % operations
func (e *Compiler) compileMultiplicative(expr parser.IMultiplicativeExpressionContext) (types.Type, error) {
	pows := validateNonZeroArray(expr.AllPowerExpression(), "multiplicative")
	if len(pows) == 1 {
		return e.compilePower(pows[0])
	}
	return e.compileBinaryMultiplicative(expr)
}

// compilePower handles ^ operations
func (e *Compiler) compilePower(expr parser.IPowerExpressionContext) (types.Type, error) {
	unary := expr.UnaryExpression()
	validateNonZero(unary, "power")
	if expr.CARET() != nil && expr.PowerExpression() != nil {
		// TODO: Implement exponentiation (needs host function)
		return e.compileUnary(unary)
	}
	return e.compileUnary(unary)
}

// compilePostfix handles array indexing, slicing, and function calls
func (e *Compiler) compilePostfix(expr parser.IPostfixExpressionContext) (types.Type, error) {
	primary := expr.PrimaryExpression()
	if primary == nil {
		return nil, errors.New("empty postfix expression")
	}
	primaryType, err := e.compilePrimary(primary)
	if err != nil {
		return nil, err
	}
	return primaryType, nil
}

// compilePrimary handles literals, identifiers, parenthesized expressions, etc.
func (e *Compiler) compilePrimary(expr parser.IPrimaryExpressionContext) (types.Type, error) {
	if lit := expr.Literal(); lit != nil {
		return e.compileLiteral(lit)
	}
	if expr.IDENTIFIER() != nil {
		return e.compileIdentifier(expr.IDENTIFIER().GetText())
	}
	if expr.LPAREN() != nil && expr.Expression() != nil {
		return e.Compile(expr.Expression())
	}
	if cast := expr.TypeCast(); cast != nil {
		return nil, errors.New("type cast not yet implemented")
	}
	if builtin := expr.BuiltinFunction(); builtin != nil {
		return nil, errors.New("builtin functions not yet implemented")
	}
	return nil, errors.New("unknown primary expression")
}

// Bytes returns the compiled bytecode
func (e *Compiler) Bytes() []byte { return e.encoder.Bytes() }

// Reset clears the bytecode buffer
func (e *Compiler) Reset() { e.encoder.Reset() }
