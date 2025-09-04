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
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/expression"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

// Compiler compiles statements to WASM bytecode
type Compiler struct {
	ctx  *compiler.Context
	expr *expression.Compiler
	enc  *wasm.Encoder
}

// NewCompiler creates a new statement compiler
func NewCompiler(ctx *compiler.Context) *Compiler {
	enc := wasm.NewEncoder()
	return &Compiler{
		ctx:  ctx,
		expr: expression.NewCompilerWithEncoder(ctx, enc),
		enc:  enc,
	}
}

// CompileStatement compiles a single statement
func (c *Compiler) CompileStatement(stmt parser.IStatementContext) error {
	if stmt == nil {
		return errors.New("cannot compile nil statement")
	}

	// Dispatch based on statement type
	if varDecl := stmt.VariableDeclaration(); varDecl != nil {
		return c.compileVariableDeclaration(varDecl)
	}
	
	if assign := stmt.Assignment(); assign != nil {
		return c.compileAssignment(assign)
	}
	
	if ifStmt := stmt.IfStatement(); ifStmt != nil {
		return c.compileIfStatement(ifStmt)
	}
	
	if retStmt := stmt.ReturnStatement(); retStmt != nil {
		return c.compileReturnStatement(retStmt)
	}
	
	if chanOp := stmt.ChannelOperation(); chanOp != nil {
		return c.compileChannelOperation(chanOp)
	}
	
	if fnCall := stmt.FunctionCall(); fnCall != nil {
		// Function calls as statements (for side effects)
		_, err := c.compileFunctionCall(fnCall)
		return err
	}
	
	return errors.New("unknown statement type")
}

// CompileBlock compiles a block of statements
func (c *Compiler) CompileBlock(block parser.IBlockContext) error {
	if block == nil {
		return nil
	}
	
	for _, stmt := range block.AllStatement() {
		if err := c.CompileStatement(stmt); err != nil {
			return err
		}
	}
	
	return nil
}

// Bytes returns the compiled bytecode
func (c *Compiler) Bytes() []byte {
	return c.enc.Bytes()
}

// Reset clears the bytecode buffer
func (c *Compiler) Reset() {
	c.enc.Reset()
	c.expr.Reset()
}