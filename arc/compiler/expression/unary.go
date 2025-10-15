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

// compileUnary handles unary -, !, and blocking read operations
func compileUnary(ctx context.Context[parser.IUnaryExpressionContext]) (types.Type, error) {
	if ctx.AST.MINUS() != nil {
		innerType, err := compileUnary(context.Child(ctx, ctx.AST.UnaryExpression()))
		if err != nil {
			return nil, err
		}
		switch innerType.(type) {
		case types.I8, types.I16, types.I32, types.U8, types.U16, types.U32:
			ctx.Writer.WriteI32Const(-1)
			ctx.Writer.WriteBinaryOp(wasm.OpI32Mul)
		case types.I64, types.U64:
			ctx.Writer.WriteI64Const(-1)
			ctx.Writer.WriteBinaryOp(wasm.OpI64Mul)
		case types.F32:
			ctx.Writer.WriteOpcode(wasm.OpF32Neg)
		case types.F64:
			ctx.Writer.WriteOpcode(wasm.OpF64Neg)
		default:
			return nil, errors.Newf("cannot negate type %s", innerType)
		}

		return innerType, nil
	}

	if ctx.AST.NOT() != nil {
		// Compile the inner expression
		_, err := compileUnary(context.Child(ctx, ctx.AST.UnaryExpression()))
		if err != nil {
			return nil, err
		}
		// Logical NOT expects a boolean (u8) and returns boolean
		// Use i32.eqz to check if value is 0 (false becomes true, true becomes false)
		ctx.Writer.WriteOpcode(wasm.OpI32Eqz)
		return types.U8{}, nil
	}

	if blockRead := ctx.AST.BlockingReadExpr(); blockRead != nil {
		// TODO: Implement blocking channel read
		return types.F64(), nil // Placeholder
	}
	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		return compilePostfix(context.Child(ctx, postfix))
	}
	return nil, errors.New("unknown unary expression")
}
