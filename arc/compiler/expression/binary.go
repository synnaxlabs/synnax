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
)

// compileBinaryAdditive handles + and - operations
func compileBinaryAdditive(
	ctx context.Context[parser.IAdditiveExpressionContext],
) (types.Type, error) {
	muls := ctx.AST.AllMultiplicativeExpression()
	resultType, err := compileMultiplicative(context.Child(ctx, muls[0]))
	if err != nil {
		return nil, err
	}
	// Process remaining operands
	for i := 1; i < len(muls); i++ {
		// Compile next operand with the left operand's type as hint
		_, err = compileMultiplicative(context.Child(ctx, muls[i]).WithHint(resultType))
		if err != nil {
			return nil, err
		}
		// Determine operator - check if it's + or -
		// This is simplified - in practice would check token positions
		op := "+"
		if i <= len(ctx.AST.AllPLUS()) {
			op = "+"
		} else {
			op = "-"
		}
		if err = ctx.Writer.WriteBinaryOpInferred(op, resultType); err != nil {
			return nil, err
		}

	}

	return resultType, nil
}

// compileBinaryMultiplicative handles *, /, % operations
func compileBinaryMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) (types.Type, error) {
	pows := ctx.AST.AllPowerExpression()
	// Compile first operand
	resultType, err := compilePower(context.Child(ctx, pows[0]))
	if err != nil {
		return nil, err
	}

	// Process remaining operands
	for i := 1; i < len(pows); i++ {
		// Compile next operand with the left operand's type as hint
		_, err := compilePower(context.Child(ctx, pows[i]).WithHint(resultType))
		if err != nil {
			return nil, err
		}

		// Determine operator - simplified logic
		op := "*"
		if i <= len(ctx.AST.AllSTAR()) {
			op = "*"
		} else if i <= len(ctx.AST.AllSTAR())+len(ctx.AST.AllSLASH()) {
			op = "/"
		} else {
			op = "%"
		}

		// Resolve and emit opcode
		if err = ctx.Writer.WriteBinaryOpInferred(op, resultType); err != nil {
			return nil, err
		}
	}

	return resultType, nil
}

// compileBinaryRelational handles <, >, <=, >= operations
func compileBinaryRelational(ctx context.Context[parser.IRelationalExpressionContext]) (types.Type, error) {
	adds := ctx.AST.AllAdditiveExpression()

	// Compile left operand
	leftType, err := compileAdditive(context.Child(ctx, adds[0]))
	if err != nil {
		return nil, err
	}

	// Compile right operand with the left operand's type as hint
	_, err = compileAdditive(context.Child(ctx, adds[1]).WithHint(leftType))
	if err != nil {
		return nil, err
	}

	// Determine operator
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

	// Resolve and emit opcode
	if err = ctx.Writer.WriteBinaryOpInferred(op, leftType); err != nil {
		return nil, err
	}

	// Comparisons return u8 (boolean)
	return types.U8{}, nil
}

// compileBinaryEquality handles == and != operations
func compileBinaryEquality(ctx context.Context[parser.IEqualityExpressionContext]) (types.Type, error) {
	rels := ctx.AST.AllRelationalExpression()

	// Compile left operand
	leftType, err := compileRelational(context.Child(ctx, rels[0]))
	if err != nil {
		return nil, err
	}

	// Compile right operand with the left operand's type as hint
	_, err = compileRelational(context.Child(ctx, rels[1]).WithHint(leftType))
	if err != nil {
		return nil, err
	}

	// Determine operator
	var op string
	if ctx.AST.EQ(0) != nil {
		op = "=="
	} else if ctx.AST.NEQ(0) != nil {
		op = "!="
	}

	// Resolve and emit opcode
	if err = ctx.Writer.WriteBinaryOpInferred(op, leftType); err != nil {
		return nil, err
	}

	// Equality comparisons return u8 (boolean)
	return types.U8{}, nil
}
