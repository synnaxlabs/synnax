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
	"github.com/synnaxlabs/slate/compiler/core"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
)

// compileBinaryAdditive handles + and - operations
func compileBinaryAdditive(
	ctx *core.Context,
	expr parser.IAdditiveExpressionContext,
) (types.Type, error) {
	muls := expr.AllMultiplicativeExpression()
	resultType, err := compileMultiplicative(ctx, muls[0])
	if err != nil {
		return nil, err
	}
	// Process remaining operands
	for i := 1; i < len(muls); i++ {
		// Compile next operand
		_, err := compileMultiplicative(ctx, muls[i])
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
		// Get and emit opcode (analyzer already validated types match)
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
	expr parser.IMultiplicativeExpressionContext,
) (types.Type, error) {
	pows := expr.AllPowerExpression()
	// Compile first operand
	resultType, err := compilePower(ctx, pows[0])
	if err != nil {
		return nil, err
	}

	// Process remaining operands
	for i := 1; i < len(pows); i++ {
		// Compile next operand
		_, err := compilePower(ctx, pows[i])
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

		// Get and emit opcode
		opcode, err := GetBinaryOpcode(op, resultType)
		if err != nil {
			return nil, err
		}
		ctx.Writer.WriteBinaryOp(opcode)
	}

	return resultType, nil
}

// compileBinaryRelational handles <, >, <=, >= operations
func compileBinaryRelational(ctx *core.Context, expr parser.IRelationalExpressionContext) (types.Type, error) {
	adds := expr.AllAdditiveExpression()

	// Compile left operand
	leftType, err := compileAdditive(ctx, adds[0])
	if err != nil {
		return nil, err
	}

	// Compile right operand
	_, err = compileAdditive(ctx, adds[1])
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

	// Get and emit opcode
	opcode, err := GetBinaryOpcode(op, leftType)
	if err != nil {
		return nil, err
	}
	ctx.Writer.WriteBinaryOp(opcode)

	// Comparisons return u8 (boolean)
	return types.U8{}, nil
}

// compileBinaryEquality handles == and != operations
func compileBinaryEquality(ctx *core.Context, expr parser.IEqualityExpressionContext) (types.Type, error) {
	rels := expr.AllRelationalExpression()

	// Compile left operand
	leftType, err := compileRelational(ctx, rels[0])
	if err != nil {
		return nil, err
	}

	// Compile right operand
	_, err = compileRelational(ctx, rels[1])
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

	// Get and emit opcode
	opcode, err := GetBinaryOpcode(op, leftType)
	if err != nil {
		return nil, err
	}
	ctx.Writer.WriteBinaryOp(opcode)

	// Equality comparisons return u8 (boolean)
	return types.U8{}, nil
}
