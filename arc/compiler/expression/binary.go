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
)

// compileBinaryAdditive handles + and - operations
func compileBinaryAdditive(
	ctx *core.Context,
	expr text.IAdditiveExpressionContext,
	hint types.Type,
) (types.Type, error) {
	muls := expr.AllMultiplicativeExpression()
	resultType, err := compileMultiplicative(ctx, muls[0], hint)
	if err != nil {
		return nil, err
	}
	// Process remaining operands
	for i := 1; i < len(muls); i++ {
		// Compile next operand with the left operand's type as hint
		_, err := compileMultiplicative(ctx, muls[i], resultType)
		if err != nil {
			return nil, err
		}
		// Determine operator - check if it's + or -
		// This is simplified - in practice would check token positions
		op := "+"
		if i <= len(expr.AllPLUS()) {
			op = "+"
		} else {
			op = "-"
		}
		// Resolve and emit opcode (analyzer already validated types match)
		opcode, err := GetBinaryOpcode(op, resultType)
		if err != nil {
			return nil, err
		}
		ctx.Writer.WriteBinaryOp(opcode)
	}

	return resultType, nil
}

// compileBinaryMultiplicative handles *, /, % operations
func compileBinaryMultiplicative(
	ctx *core.Context,
	expr text.IMultiplicativeExpressionContext,
	hint types.Type,
) (types.Type, error) {
	pows := expr.AllPowerExpression()
	// Compile first operand
	resultType, err := compilePower(ctx, pows[0], hint)
	if err != nil {
		return nil, err
	}

	// Process remaining operands
	for i := 1; i < len(pows); i++ {
		// Compile next operand with the left operand's type as hint
		_, err := compilePower(ctx, pows[i], resultType)
		if err != nil {
			return nil, err
		}

		// Determine operator - simplified logic
		op := "*"
		if i <= len(expr.AllSTAR()) {
			op = "*"
		} else if i <= len(expr.AllSTAR())+len(expr.AllSLASH()) {
			op = "/"
		} else {
			op = "%"
		}

		// Resolve and emit opcode
		opcode, err := GetBinaryOpcode(op, resultType)
		if err != nil {
			return nil, err
		}
		ctx.Writer.WriteBinaryOp(opcode)
	}

	return resultType, nil
}

// compileBinaryRelational handles <, >, <=, >= operations
func compileBinaryRelational(ctx *core.Context, expr text.IRelationalExpressionContext, hint types.Type) (types.Type, error) {
	adds := expr.AllAdditiveExpression()

	// Compile left operand
	leftType, err := compileAdditive(ctx, adds[0], hint)
	if err != nil {
		return nil, err
	}

	// Compile right operand with the left operand's type as hint
	_, err = compileAdditive(ctx, adds[1], leftType)
	if err != nil {
		return nil, err
	}

	// Determine operator
	var op string
	if expr.LT(0) != nil {
		op = "<"
	} else if expr.GT(0) != nil {
		op = ">"
	} else if expr.LEQ(0) != nil {
		op = "<="
	} else if expr.GEQ(0) != nil {
		op = ">="
	}

	// Resolve and emit opcode
	opcode, err := GetBinaryOpcode(op, leftType)
	if err != nil {
		return nil, err
	}
	ctx.Writer.WriteBinaryOp(opcode)

	// Comparisons return u8 (boolean)
	return types.U8{}, nil
}

// compileBinaryEquality handles == and != operations
func compileBinaryEquality(ctx *core.Context, expr text.IEqualityExpressionContext, hint types.Type) (types.Type, error) {
	rels := expr.AllRelationalExpression()

	// Compile left operand
	leftType, err := compileRelational(ctx, rels[0], hint)
	if err != nil {
		return nil, err
	}

	// Compile right operand with the left operand's type as hint
	_, err = compileRelational(ctx, rels[1], leftType)
	if err != nil {
		return nil, err
	}

	// Determine operator
	var op string
	if expr.EQ(0) != nil {
		op = "=="
	} else if expr.NEQ(0) != nil {
		op = "!="
	}

	// Resolve and emit opcode
	opcode, err := GetBinaryOpcode(op, leftType)
	if err != nil {
		return nil, err
	}
	ctx.Writer.WriteBinaryOp(opcode)

	// Equality comparisons return u8 (boolean)
	return types.U8{}, nil
}
