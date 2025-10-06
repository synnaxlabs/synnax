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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
)

// compileLogicalOrImpl handles || operations with short-circuit evaluation
func compileLogicalOrImpl(
	ctx context.Context[parser.ILogicalOrExpressionContext],
) (ir.Type, error) {
	ands := ctx.AST.AllLogicalAndExpression()

	// Compile first operand
	_, err := compileLogicalAnd(context.Child(ctx, ands[0]))
	if err != nil {
		return nil, err
	}

	// Normalize the first operand
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
		_, err = compileLogicalAnd(context.Child(ctx, ands[i]))
		if err != nil {
			return nil, err
		}

		// Normalize the result
		normalizeBoolean(ctx)

		ctx.Writer.WriteOpcode(wasm.OpEnd)
	}

	return ir.U8{}, nil
}

// compileLogicalAndImpl handles && operations with short-circuit evaluation
func compileLogicalAndImpl(ctx context.Context[parser.ILogicalAndExpressionContext]) (ir.Type, error) {
	eqs := ctx.AST.AllEqualityExpression()

	// Compile first operand
	_, err := compileEquality(context.Child(ctx, eqs[0]))
	if err != nil {
		return nil, err
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
		_, err := compileEquality(context.Child(ctx, eqs[i]))
		if err != nil {
			return nil, err
		}

		// Normalize the result
		normalizeBoolean(ctx)

		ctx.Writer.WriteOpcode(wasm.OpEnd)
	}

	return ir.U8{}, nil
}

// normalizeBoolean converts any non-zero i32 value to 1
func normalizeBoolean[ASTNode antlr.ParserRuleContext](ctx context.Context[ASTNode]) {
	// Convert any non-zero value to 1
	// value != 0 ? 1 : 0
	// This is equivalent to: (value != 0)
	ctx.Writer.WriteI32Const(0)
	ctx.Writer.WriteOpcode(wasm.OpI32Ne)
}
