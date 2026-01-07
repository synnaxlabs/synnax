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
	"github.com/synnaxlabs/arc/compiler/bindings"
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
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		return compileLogicalOr(context.Child(ctx, logicalOr))
	}
	return types.Type{}, errors.New("unknown expression type")
}

func compileLogicalOr(
	ctx context.Context[parser.ILogicalOrExpressionContext],
) (types.Type, error) {
	ands := validateNonZeroArray(ctx.AST.AllLogicalAndExpression(), "logical OR")
	if len(ands) == 1 {
		return compileLogicalAnd(context.Child(ctx, ands[0]))
	}
	return compileLogicalOrImpl(ctx)
}

func compileLogicalAnd(
	ctx context.Context[parser.ILogicalAndExpressionContext],
) (types.Type, error) {
	eqs := validateNonZeroArray(ctx.AST.AllEqualityExpression(), "logical AND")
	if len(eqs) == 1 {
		return compileEquality(context.Child(ctx, eqs[0]))
	}
	return compileLogicalAndImpl(ctx)
}

func compileEquality(
	ctx context.Context[parser.IEqualityExpressionContext],
) (types.Type, error) {
	rels := validateNonZeroArray(ctx.AST.AllRelationalExpression(), "equality")
	if len(rels) == 1 {
		return compileRelational(context.Child(ctx, rels[0]))
	}
	return compileBinaryEquality(ctx)
}

func compileRelational(
	ctx context.Context[parser.IRelationalExpressionContext],
) (types.Type, error) {
	adds := validateNonZeroArray(ctx.AST.AllAdditiveExpression(), "relational")
	if len(adds) == 1 {
		return compileAdditive(context.Child(ctx, adds[0]))
	}
	return compileBinaryRelational(ctx)
}

func compileAdditive(
	ctx context.Context[parser.IAdditiveExpressionContext],
) (types.Type, error) {
	muls := validateNonZeroArray(ctx.AST.AllMultiplicativeExpression(), "additive")
	if len(muls) == 1 {
		return compileMultiplicative(context.Child(ctx, muls[0]))
	}
	return compileBinaryAdditive(ctx)
}

func compileMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) (types.Type, error) {
	pows := validateNonZeroArray(ctx.AST.AllPowerExpression(), "multiplicative")
	if len(pows) == 1 {
		return compilePower(context.Child(ctx, pows[0]))
	}
	return compileBinaryMultiplicative(ctx)
}

// compilePower handles ^ operations (right-associative)
func compilePower(
	ctx context.Context[parser.IPowerExpressionContext],
) (types.Type, error) {
	unary := ctx.AST.UnaryExpression()
	validateNonZero(unary, "power")

	// Compile base (left operand)
	baseType, err := compileUnary(context.Child(ctx, unary))
	if err != nil {
		return types.Type{}, err
	}

	// If no caret operator, just return the base
	if ctx.AST.CARET() == nil || ctx.AST.PowerExpression() == nil {
		return baseType, nil
	}

	// Compile exponent (right operand, recursive for right-associativity)
	exponentType, err := compilePower(
		context.Child(ctx, ctx.AST.PowerExpression()).WithHint(baseType.Unwrap()),
	)
	if err != nil {
		return types.Type{}, err
	}

	// Determine result type and call appropriate power function
	var importIdx uint32
	if baseType.IsInteger() && exponentType.IsInteger() {
		// Both operands are integers, use IntPow
		importIdx, err = getIntPowImport(ctx.Imports, baseType)
		if err != nil {
			return types.Type{}, err
		}
	} else {
		// At least one float, use math.Pow
		if baseType.Is64Bit() || exponentType.Is64Bit() {
			importIdx = ctx.Imports.MathPowF64
		} else {
			importIdx = ctx.Imports.MathPowF32
		}
	}

	ctx.Writer.WriteCall(importIdx)
	return baseType, nil
}

// getIntPowImport returns the appropriate IntPow import index for the given type
func getIntPowImport(imports *bindings.ImportIndex, t types.Type) (uint32, error) {
	switch t.Kind {
	case types.KindI8:
		return imports.MathIntPowI8, nil
	case types.KindI16:
		return imports.MathIntPowI16, nil
	case types.KindI32:
		return imports.MathIntPowI32, nil
	case types.KindI64:
		return imports.MathIntPowI64, nil
	case types.KindU8:
		return imports.MathIntPowU8, nil
	case types.KindU16:
		return imports.MathIntPowU16, nil
	case types.KindU32:
		return imports.MathIntPowU32, nil
	case types.KindU64:
		return imports.MathIntPowU64, nil
	default:
		return 0, errors.Newf("no IntPow import for type %s", t)
	}
}

func compilePostfix(ctx context.Context[parser.IPostfixExpressionContext]) (types.Type, error) {
	primary := ctx.AST.PrimaryExpression()
	currentType, err := compilePrimary(context.Child(ctx, primary))
	if err != nil {
		return types.Type{}, err
	}

	// Handle any indexOrSlice operations
	for _, indexOrSlice := range ctx.AST.AllIndexOrSlice() {
		currentType, err = compileIndexOrSlice(ctx, indexOrSlice, currentType)
		if err != nil {
			return types.Type{}, err
		}
	}

	// TODO: Handle functionCallSuffix when function calls are implemented

	return currentType, nil
}

func compileIndexOrSlice(
	ctx context.Context[parser.IPostfixExpressionContext],
	indexOrSlice parser.IIndexOrSliceContext,
	operandType types.Type,
) (types.Type, error) {
	expressions := indexOrSlice.AllExpression()
	isSliceOp := indexOrSlice.COLON() != nil

	if !isSliceOp {
		if operandType.Kind != types.KindSeries {
			return types.Type{}, errors.New("indexing is only supported on series types")
		}
		if _, err := Compile(context.Child(ctx, expressions[0]).WithHint(types.I32())); err != nil {
			return types.Type{}, err
		}
		t := operandType.Unwrap()
		funcIdx, err := ctx.Imports.GetSeriesIndex(t)
		if err != nil {
			return types.Type{}, err
		}
		ctx.Writer.WriteCall(funcIdx)
		return t, nil
	}

	// Slice operation: series[start:end]
	if operandType.Kind != types.KindSeries {
		return types.Type{}, errors.New("slicing is only supported on series types")
	}

	// Determine start and end expressions
	// Grammar: LBRACKET expression? COLON expression? RBRACKET
	var startExpr, endExpr parser.IExpressionContext
	if len(expressions) == 1 {
		if indexOrSlice.GetChild(1) == expressions[0] {
			startExpr = expressions[0] // before colon: s[start:]
		} else {
			endExpr = expressions[0] // after colon: s[:end]
		}
	} else if len(expressions) == 2 {
		startExpr = expressions[0]
		endExpr = expressions[1]
	} else {
		return types.Type{}, errors.Newf("expected 1 or 2 items in slice expression, received %v", len(expressions))
	}

	// Compile start index (or push 0 if not specified)
	if startExpr != nil {
		if _, err := Compile(context.Child(ctx, startExpr).WithHint(types.I32())); err != nil {
			return types.Type{}, err
		}
	} else {
		ctx.Writer.WriteI32Const(0)
	}

	// Compile end index (or push -1 to indicate "to end")
	if endExpr != nil {
		if _, err := Compile(context.Child(ctx, endExpr).WithHint(types.I32())); err != nil {
			return types.Type{}, err
		}
	} else {
		ctx.Writer.WriteI32Const(-1)
	}

	// Call SeriesSlice(handle, start, end) → new handle
	ctx.Writer.WriteCall(ctx.Imports.SeriesSlice)

	return operandType, nil
}

func compilePrimary(ctx context.Context[parser.IPrimaryExpressionContext]) (types.Type, error) {
	if lit := ctx.AST.Literal(); lit != nil {
		return compileLiteral(context.Child(ctx, lit))
	}
	if id := ctx.AST.IDENTIFIER(); id != nil {
		text := id.GetText()
		// Handle boolean literals (parsed as identifiers in the grammar)
		if text == "true" {
			ctx.Writer.WriteI32Const(1)
			return types.U8(), nil
		}
		if text == "false" {
			ctx.Writer.WriteI32Const(0)
			return types.U8(), nil
		}
		return compileIdentifier(ctx, text)
	}
	if ctx.AST.LPAREN() != nil && ctx.AST.Expression() != nil {
		return Compile(context.Child(ctx, ctx.AST.Expression()))
	}
	if cast := ctx.AST.TypeCast(); cast != nil {
		return compileTypeCast(context.Child(ctx, cast))
	}
	return types.Type{}, errors.New("unknown primary expression")
}
