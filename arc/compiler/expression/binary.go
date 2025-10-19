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

func compileBinaryAdditive(
	ctx context.Context[parser.IAdditiveExpressionContext],
) (types.Type, error) {
	muls := ctx.AST.AllMultiplicativeExpression()
	resultType, err := compileMultiplicative(context.Child(ctx, muls[0]))
	if err != nil {
		return types.Type{}, err
	}
	for i := 1; i < len(muls); i++ {
		_, err = compileMultiplicative(context.Child(ctx, muls[i]).WithHint(resultType))
		if err != nil {
			return types.Type{}, err
		}
		op := "+"
		if i <= len(ctx.AST.AllPLUS()) {
			op = "+"
		} else {
			op = "-"
		}
		if err = ctx.Writer.WriteBinaryOpInferred(op, resultType); err != nil {
			return types.Type{}, err
		}
	}
	return resultType, nil
}

func compileBinaryMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) (types.Type, error) {
	pows := ctx.AST.AllPowerExpression()
	resultType, err := compilePower(context.Child(ctx, pows[0]))
	if err != nil {
		return types.Type{}, err
	}
	for i := 1; i < len(pows); i++ {
		_, err := compilePower(context.Child(ctx, pows[i]).WithHint(resultType))
		if err != nil {
			return types.Type{}, err
		}
		var op string
		if i <= len(ctx.AST.AllSTAR()) {
			op = "*"
		} else if i <= len(ctx.AST.AllSTAR())+len(ctx.AST.AllSLASH()) {
			op = "/"
		} else {
			op = "%"
		}
		if err = ctx.Writer.WriteBinaryOpInferred(op, resultType); err != nil {
			return types.Type{}, err
		}
	}
	return resultType, nil
}

func compileBinaryRelational(ctx context.Context[parser.IRelationalExpressionContext]) (types.Type, error) {
	adds := ctx.AST.AllAdditiveExpression()
	leftType, err := compileAdditive(context.Child(ctx, adds[0]))
	if err != nil {
		return types.Type{}, err
	}
	_, err = compileAdditive(context.Child(ctx, adds[1]).WithHint(leftType))
	if err != nil {
		return types.Type{}, err
	}
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
	if err = ctx.Writer.WriteBinaryOpInferred(op, leftType); err != nil {
		return types.Type{}, err
	}
	return types.U8(), nil
}

func compileBinaryEquality(ctx context.Context[parser.IEqualityExpressionContext]) (types.Type, error) {
	rels := ctx.AST.AllRelationalExpression()
	leftType, err := compileRelational(context.Child(ctx, rels[0]))
	if err != nil {
		return types.Type{}, err
	}
	_, err = compileRelational(context.Child(ctx, rels[1]).WithHint(leftType))
	if err != nil {
		return types.Type{}, err
	}
	var op string
	if ctx.AST.EQ(0) != nil {
		op = "=="
	} else if ctx.AST.NEQ(0) != nil {
		op = "!="
	}
	if err = ctx.Writer.WriteBinaryOpInferred(op, leftType); err != nil {
		return types.Type{}, err
	}
	return types.U8(), nil
}
