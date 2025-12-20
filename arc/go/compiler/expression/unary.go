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
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileUnary(ctx context.Context[parser.IUnaryExpressionContext]) (types.Type, error) {
	if ctx.AST.MINUS() != nil {
		innerType, err := compileUnary(context.Child(ctx, ctx.AST.UnaryExpression()))
		if err != nil {
			return types.Type{}, err
		}
		switch innerType.Kind {
		case types.KindI8, types.KindI16, types.KindI32, types.KindU8, types.KindU16, types.KindU32:
			ctx.Writer.WriteI32Const(-1)
			ctx.Writer.WriteBinaryOp(wasm.OpI32Mul)
		case types.KindI64, types.KindU64:
			ctx.Writer.WriteI64Const(-1)
			ctx.Writer.WriteBinaryOp(wasm.OpI64Mul)
		case types.KindF32:
			ctx.Writer.WriteOpcode(wasm.OpF32Neg)
		case types.KindF64:
			ctx.Writer.WriteOpcode(wasm.OpF64Neg)
		default:
			return types.Type{}, errors.Newf("cannot negate type %s", innerType)
		}
		return innerType, nil
	}

	if ctx.AST.NOT() != nil {
		// Compile the inner expression
		if _, err := compileUnary(context.Child(ctx, ctx.AST.UnaryExpression())); err != nil {
			return types.Type{}, err
		}
		// Logical NOT expects a boolean (u8) and returns boolean
		// Use i32.eqz to check if value is 0 (false becomes true, true becomes false)
		ctx.Writer.WriteOpcode(wasm.OpI32Eqz)
		return types.U8(), nil
	}

	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		return compilePostfix(context.Child(ctx, postfix))
	}
	return types.Type{}, errors.New("unknown unary expression")
}
