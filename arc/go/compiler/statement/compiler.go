// Copyright 2026 Synnax Labs, Inc.
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

// Compile compiles a statement and returns whether execution diverged
// (return/break/etc)
func Compile(ctx context.Context[parser.IStatementContext]) (diverged bool, err error) {
	if varDecl := ctx.AST.VariableDeclaration(); varDecl != nil {
		return false, compileVariableDeclaration(ctx.Child(varDecl))
	}
	if assign := ctx.AST.Assignment(); assign != nil {
		return false, compileAssignment(ctx.Child(assign))
	}
	if ifStmt := ctx.AST.IfStatement(); ifStmt != nil {
		return compileIfStatement(ctx.Child(ifStmt))
	}
	if forStmt := ctx.AST.ForStatement(); forStmt != nil {
		return compileForStatement(ctx.Child(forStmt))
	}
	if ctx.AST.BreakStatement() != nil {
		return true, compileBreakStatement(ctx.Child(ctx.AST.BreakStatement()))
	}
	if ctx.AST.ContinueStatement() != nil {
		return true, compileContinueStatement(
			ctx.Child(ctx.AST.ContinueStatement()),
		)
	}
	if retStmt := ctx.AST.ReturnStatement(); retStmt != nil {
		return true, compileReturnStatement(ctx.Child(retStmt))
	}
	if expr := ctx.AST.Expression(); expr != nil {
		_, err = compileExpressionStatement(ctx.Child(expr))
		return false, err
	}
	return false, errors.New("unknown statement type")
}

// CompileBlock compiles a block and returns whether all paths diverged
func CompileBlock(
	ctx context.Context[parser.IBlockContext],
) (diverged bool, err error) {
	if ctx.AST == nil {
		return false, nil
	}
	blockScope, err := ctx.Scope.GetChildByParserRule(ctx.AST)
	if err != nil {
		panic(err)
	}
	blockCtx := ctx.WithScope(blockScope)
	for _, stmt := range ctx.AST.AllStatement() {
		var stmtDiverged bool
		stmtDiverged, err = Compile(blockCtx.Child(stmt))
		if err != nil {
			return false, err
		}
		// If we hit a diverging statement, all subsequent paths diverge
		if stmtDiverged {
			return true, nil
		}
	}
	return false, nil
}
