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
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
)

func Compile(ctx context.Context[parser.IStatementContext]) error {
	if varDecl := ctx.AST.VariableDeclaration(); varDecl != nil {
		return compileVariableDeclaration(context.Child(ctx, varDecl))
	}
	if assign := ctx.AST.Assignment(); assign != nil {
		return compileAssignment(context.Child(ctx, assign))
	}
	if ifStmt := ctx.AST.IfStatement(); ifStmt != nil {
		return compileIfStatement(context.Child(ctx, ifStmt))
	}
	if retStmt := ctx.AST.ReturnStatement(); retStmt != nil {
		return compileReturnStatement(context.Child(ctx, retStmt))
	}
	if chanOp := ctx.AST.ChannelOperation(); chanOp != nil {
		return compileChannelOperation(context.Child(ctx, chanOp))
	}
	if fnCall := ctx.AST.FunctionCall(); fnCall != nil {
		_, err := compileFunctionCall(context.Child(ctx, fnCall))
		return err
	}
	return errors.New("unknown statement type")
}

func CompileBlock(ctx context.Context[parser.IBlockContext]) error {
	if ctx.AST == nil {
		return nil
	}
	blockScope, err := ctx.Scope.GetChildByParserRule(ctx.AST)
	if err != nil {
		panic(err)
	}
	blockCtx := ctx.WithScope(blockScope)
	for _, stmt := range ctx.AST.AllStatement() {
		if err = Compile(context.Child(blockCtx, stmt)); err != nil {
			return err
		}
	}
	return nil
}
