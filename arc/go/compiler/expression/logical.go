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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
)

func compileLogicalOrImpl(
	ctx context.Context[parser.ILogicalOrExpressionContext],
) (types.Type, error) {
	ands := ctx.AST.AllLogicalAndExpression()
	if _, err := compileLogicalAnd(context.Child(ctx, ands[0])); err != nil {
		return types.Type{}, err
	}

	normalizeBoolean(ctx)

	// Process remaining operands with short-circuit evaluation
	for i := 1; i < len(ands); i++ {
		// The stack has the current boolean value (0 or 1)
		// If it's true (1), we skip evaluation of the right operand
		// Use if-else block for short-circuit evaluation
		ctx.Writer.WriteIf(wasm.BlockTypeI32)
		// True case: value is already 1, keep it
		ctx.Writer.WriteI32Const(1)
		ctx.Writer.WriteOpcode(wasm.OpElse)
		// False case: evaluate right operand
		if _, err := compileLogicalAnd(context.Child(ctx, ands[i])); err != nil {
			return types.Type{}, err
		}
		normalizeBoolean(ctx)
		ctx.Writer.WriteOpcode(wasm.OpEnd)
	}
	return types.U8(), nil
}

func compileLogicalAndImpl(ctx context.Context[parser.ILogicalAndExpressionContext]) (types.Type, error) {
	eqs := ctx.AST.AllEqualityExpression()

	// Compile first operand
	if _, err := compileEquality(context.Child(ctx, eqs[0])); err != nil {
		return types.Type{}, err
	}

	// Normalize the first operand
	normalizeBoolean(ctx)

	// Process remaining operands with short-circuit evaluation
	for i := 1; i < len(eqs); i++ {
		// The stack has the current boolean value (0 or 1)
		// If it's false (0), we skip evaluation of the right operand
		// Use if-else block for short-circuit evaluation
		ctx.Writer.WriteOpcode(wasm.OpI32Eqz) // Invert: 0 -> 1, 1 -> 0
		ctx.Writer.WriteIf(wasm.BlockTypeI32)
		// True case (was zero): result is 0
		ctx.Writer.WriteI32Const(0)
		ctx.Writer.WriteOpcode(wasm.OpElse)
		// False case (was non-zero): evaluate right operand
		if _, err := compileEquality(context.Child(ctx, eqs[i])); err != nil {
			return types.Type{}, err
		}
		// Normalize the result
		normalizeBoolean(ctx)
		ctx.Writer.WriteOpcode(wasm.OpEnd)
	}

	return types.U8(), nil
}

// normalizeBoolean converts any non-zero i32 value to 1
func normalizeBoolean[ASTNode antlr.ParserRuleContext](ctx context.Context[ASTNode]) {
	// Convert any non-zero value to 1
	// value != 0 ? 1 : 0
	// This is equivalent to: (value != 0)
	ctx.Writer.WriteI32Const(0)
	ctx.Writer.WriteOpcode(wasm.OpI32Ne)
}
