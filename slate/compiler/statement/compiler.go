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
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

// Compile compiles a single statement
func Compile(
	ctx *core.Context,
	stmt parser.IStatementContext,
) error {
	if stmt == nil {
		return errors.New("cannot compile nil statement")
	}
	if varDecl := stmt.VariableDeclaration(); varDecl != nil {
		return compileVariableDeclaration(ctx, varDecl)
	}
	if assign := stmt.Assignment(); assign != nil {
		return compileAssignment(ctx, assign)
	}
	if ifStmt := stmt.IfStatement(); ifStmt != nil {
		return compileIfStatement(ctx, ifStmt)
	}
	if retStmt := stmt.ReturnStatement(); retStmt != nil {
		return compileReturnStatement(ctx, retStmt)
	}
	if chanOp := stmt.ChannelOperation(); chanOp != nil {
		return compileChannelOperation(ctx, chanOp)
	}
	if fnCall := stmt.FunctionCall(); fnCall != nil {
		// Function calls as statements (for side effects)
		_, err := compileFunctionCall(ctx, fnCall)
		return err
	}
	return errors.New("unknown statement type")
}

// CompileBlock compiles a block of statements
func CompileBlock(ctx *core.Context, block parser.IBlockContext) error {
	if block == nil {
		return nil
	}
	blockScope, err := ctx.Scope.GetChildByParserRule(block)
	if err != nil {
		panic(err)
	}
	blockCtx := ctx.WithScope(blockScope)
	for _, stmt := range block.AllStatement() {
		if err := Compile(blockCtx, stmt); err != nil {
			return err
		}
	}
	return nil
}
