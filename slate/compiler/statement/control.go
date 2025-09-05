// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement

import (
	"github.com/synnaxlabs/slate/compiler/core"
	"github.com/synnaxlabs/slate/compiler/expression"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// compileIfStatement compiles if/else-if/else chains
func compileIfStatement(
	ctx *core.Context,
	ifStmt parser.IIfStatementContext,
) error {
	// Compile the condition expression
	if _, err := expression.Compile(ctx, ifStmt.Expression()); err != nil {
		return errors.Wrap(err, "failed to compile if condition")
	}
	// Check if we have an else clause to determine block type
	hasElse := ifStmt.ElseClause() != nil || len(ifStmt.AllElseIfClause()) > 0
	if hasElse {
		// If-else structure
		ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
		// Compile the if block
		if err := CompileBlock(ctx, ifStmt.Block()); err != nil {
			return errors.Wrap(err, "failed to compile if block")
		}

		// Handle else-if clauses
		for i, elseIfClause := range ifStmt.AllElseIfClause() {
			ctx.Writer.WriteElse()
			_, err := expression.Compile(ctx, elseIfClause.Expression())
			if err != nil {
				return errors.Wrapf(err, "failed to compile else-if[%d] condition", i)
			}
			ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
			if err := CompileBlock(ctx, elseIfClause.Block()); err != nil {
				return errors.Wrapf(err, "failed to compile else-if[%d] block", i)
			}
		}
		if elseClause := ifStmt.ElseClause(); elseClause != nil {
			ctx.Writer.WriteElse()
			if err := CompileBlock(ctx, elseClause.Block()); err != nil {
				return errors.Wrap(err, "failed to compile else block")
			}
		} else if len(ifStmt.AllElseIfClause()) > 0 {
			ctx.Writer.WriteElse()
		}

		for range ifStmt.AllElseIfClause() {
			ctx.Writer.WriteEnd()
		}
		ctx.Writer.WriteEnd() // Close main if

	} else {
		// Simple if without else
		ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
		// Compile the if block
		if err := CompileBlock(ctx, ifStmt.Block()); err != nil {
			return errors.Wrap(err, "failed to compile if block")
		}
		ctx.Writer.WriteEnd()
	}

	return nil
}

// compileReturnStatement compiles return statements
func compileReturnStatement(ctx *core.Context, ret parser.IReturnStatementContext) error {
	// Check if we have a return expression
	if expr := ret.Expression(); expr != nil {
		// Compile the return expression
		exprType, err := expression.Compile(ctx, expr)
		if err != nil {
			return errors.Wrap(err, "failed to compile return expression")
		}
		functionScope, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
		if err != nil {
			return errors.Wrap(err, "failed to compile function declaration")
		}
		fType := functionScope.Type.(types.Function)
		if fType.Return != exprType {
			expression.EmitCast(ctx, exprType, fType.Return)
		}
	}
	// If no expression, it's a void return
	// Emit return instruction
	ctx.Writer.WriteReturn()
	return nil
}

// compileChannelOperation handles channel writes and piping
func compileChannelOperation(ctx *core.Context, chanOp parser.IChannelOperationContext) error {
	if chanWrite := chanOp.ChannelWrite(); chanWrite != nil {
		return compileChannelWrite(ctx, chanWrite)
	}

	if chanRead := chanOp.ChannelRead(); chanRead != nil {
		return compileChannelRead(ctx, chanRead)
	}

	// Channel piping not yet supported in parser
	// if chanPipe := chanOp.ChannelPipe(); chanPipe != nil {
	//	return c.compileChannelPipe(chanPipe)
	// }

	return errors.New("unknown channel operation")
}

// compileChannelWrite handles value -> channel or channel <- value
func compileChannelWrite(ctx *core.Context, write parser.IChannelWriteContext) error {
	// Grammar: Expression '->' Identifier | Identifier '<-' Expression

	var channelName string
	var valueExpr parser.IExpressionContext

	// Determine which form we have
	if write.Expression() != nil && write.IDENTIFIER() != nil {
		// Could be either form, check arrow position
		// For now, assume first form: expr -> channel
		valueExpr = write.Expression()
		channelName = write.IDENTIFIER().GetText()
	}

	// Compile the value expression
	valueType, err := expression.Compile(ctx, valueExpr)
	if err != nil {
		return errors.Wrap(err, "failed to compile channel write value")
	}

	// Look up the channel to get its ID
	_, err = ctx.Scope.Resolve(channelName)
	if err != nil {
		return errors.Wrapf(err, "channel '%s' not found", channelName)
	}

	// Resolve channel ID from local (channels are passed as parameters)
	sym, err := ctx.Scope.Resolve(channelName)
	if err != nil {
		return errors.Newf("channel '%s' not in local context", channelName)
	}
	// Push channel ID
	ctx.Writer.WriteLocalGet(sym.ID)
	// Value is already on stack from expression compilation
	// Call channel write function
	importIdx, err := ctx.Imports.GetChannelWrite(valueType)
	if err != nil {
		return errors.Wrap(err, "failed to compile channel write import")
	}
	ctx.Writer.WriteCall(importIdx)
	return nil
}

// compileChannelRead handles blocking reads: x := <-channel
func compileChannelRead(ctx *core.Context, read parser.IChannelReadContext) error {
	// This is handled as part of variable declaration
	// The parser should not give us standalone channel reads as statements
	return errors.New("standalone channel reads not implemented")
}

// compileFunctionCall handles function calls (may return a value)
func compileFunctionCall(ctx *core.Context, call parser.IFunctionCallContext) (types.Type, error) {
	// TODO: Implement function calls
	return nil, errors.New("function calls not yet implemented")
}
