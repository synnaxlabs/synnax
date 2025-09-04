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
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// compileIfStatement compiles if/else-if/else chains
func (c *Compiler) compileIfStatement(ifStmt parser.IIfStatementContext) error {
	// Compile the condition expression
	if _, err := c.expr.Compile(ifStmt.Expression()); err != nil {
		return errors.Wrap(err, "failed to compile if condition")
	}
	// Check if we have an else clause to determine block type
	hasElse := ifStmt.ElseClause() != nil || len(ifStmt.AllElseIfClause()) > 0
	if hasElse {
		// If-else structure
		c.enc.WriteIf(wasm.BlockTypeEmpty)
		// Compile the if block
		if err := c.CompileBlock(ifStmt.Block()); err != nil {
			return errors.Wrap(err, "failed to compile if block")
		}

		// Handle else-if clauses
		for i, elseIfClause := range ifStmt.AllElseIfClause() {
			c.enc.WriteElse()

			// Compile else-if condition
			_, err := c.expr.Compile(elseIfClause.Expression())
			if err != nil {
				return errors.Wrapf(err, "failed to compile else-if[%d] condition", i)
			}

			// Nested if for else-if
			c.enc.WriteIf(wasm.BlockTypeEmpty)

			// Compile else-if block
			if err := c.CompileBlock(elseIfClause.Block()); err != nil {
				return errors.Wrapf(err, "failed to compile else-if[%d] block", i)
			}
		}

		// Handle final else clause
		if elseClause := ifStmt.ElseClause(); elseClause != nil {
			c.enc.WriteElse()
			if err := c.CompileBlock(elseClause.Block()); err != nil {
				return errors.Wrap(err, "failed to compile else block")
			}
		} else if len(ifStmt.AllElseIfClause()) > 0 {
			// Close the last else-if without a final else
			c.enc.WriteElse()
			// Empty else block
		}

		// Close all nested ifs
		for range ifStmt.AllElseIfClause() {
			c.enc.WriteEnd()
		}
		c.enc.WriteEnd() // Close main if

	} else {
		// Simple if without else
		c.enc.WriteIf(wasm.BlockTypeEmpty)

		// Compile the if block
		if err := c.CompileBlock(ifStmt.Block()); err != nil {
			return errors.Wrap(err, "failed to compile if block")
		}

		c.enc.WriteEnd()
	}

	return nil
}

// compileReturnStatement compiles return statements
func (c *Compiler) compileReturnStatement(ret parser.IReturnStatementContext) error {
	// Check if we have a return expression
	if expr := ret.Expression(); expr != nil {
		// Compile the return expression
		exprType, err := c.expr.Compile(expr)
		if err != nil {
			return errors.Wrap(err, "failed to compile return expression")
		}

		// Add type conversion if needed (e.g., i64 literal to i32 return)
		if c.ctx.Current.ReturnType != nil && needsConversion(exprType, c.ctx.Current.ReturnType) {
			emitTypeConversion(c.enc, exprType, c.ctx.Current.ReturnType)
		}
	}
	// If no expression, it's a void return

	// Emit return instruction
	c.enc.WriteReturn()

	return nil
}

// compileChannelOperation handles channel writes and piping
func (c *Compiler) compileChannelOperation(chanOp parser.IChannelOperationContext) error {
	if chanWrite := chanOp.ChannelWrite(); chanWrite != nil {
		return c.compileChannelWrite(chanWrite)
	}

	if chanRead := chanOp.ChannelRead(); chanRead != nil {
		return c.compileChannelRead(chanRead)
	}

	// Channel piping not yet supported in parser
	// if chanPipe := chanOp.ChannelPipe(); chanPipe != nil {
	//	return c.compileChannelPipe(chanPipe)
	// }

	return errors.New("unknown channel operation")
}

// compileChannelWrite handles value -> channel or channel <- value
func (c *Compiler) compileChannelWrite(write parser.IChannelWriteContext) error {
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
	valueType, err := c.expr.Compile(valueExpr)
	if err != nil {
		return errors.Wrap(err, "failed to compile channel write value")
	}

	// Look up the channel to get its ID
	_, err = c.ctx.Symbols.Get(channelName)
	if err != nil {
		return errors.Wrapf(err, "channel '%s' not found", channelName)
	}

	// Get channel ID from local (channels are passed as parameters)
	chanIdx, ok := c.ctx.GetLocal(channelName)
	if !ok {
		return errors.Newf("channel '%s' not in local context", channelName)
	}

	// Push channel ID
	c.enc.WriteLocalGet(chanIdx)

	// Value is already on stack from expression compilation
	// Call channel write function
	importIdx := c.ctx.Imports.GetChannelWrite(valueType)
	c.enc.WriteCall(importIdx)

	return nil
}

// compileChannelRead handles blocking reads: x := <-channel
func (c *Compiler) compileChannelRead(read parser.IChannelReadContext) error {
	// This is handled as part of variable declaration
	// The parser should not give us standalone channel reads as statements
	return errors.New("standalone channel reads not implemented")
}

// compileFunctionCall handles function calls (may return a value)
func (c *Compiler) compileFunctionCall(call parser.IFunctionCallContext) (types.Type, error) {
	// TODO: Implement function calls
	return nil, errors.New("function calls not yet implemented")
}
