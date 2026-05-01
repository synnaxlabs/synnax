// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unused

import (
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/function"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/diagnostics"
)

// analyzeUnreachable emits ARC5203 for the first statement in any function
// body block that is preceded by a statement that always returns. Stage
// bodies are reactive parallel flows with no notion of statement ordering,
// so only function bodies are analyzed.
func analyzeUnreachable(ctx context.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if fnDecl := item.FunctionDeclaration(); fnDecl != nil {
			checkBlock(fnDecl.Block(), ctx.Diagnostics)
		}
	}
}

// checkBlock emits at most one ARC5203 warning per block and recurses into
// nested if / else-if / else blocks. It stops scanning a block after
// emitting so callers are not overwhelmed by cascades in a long dead tail.
func checkBlock(block parser.IBlockContext, diag *diagnostics.Diagnostics) {
	if block == nil {
		return
	}
	statements := block.AllStatement()
	for i, stmt := range statements {
		if ifStmt := stmt.IfStatement(); ifStmt != nil {
			checkIfStatement(ifStmt, diag)
		}
		if i < len(statements)-1 && statementAlwaysExits(stmt) {
			diag.Add(diagnostics.
				Warningf(statements[i+1], "unreachable code: previous statement always returns").
				WithCode(codes.UnreachableCode))
			return
		}
	}
}

func checkIfStatement(ifStmt parser.IIfStatementContext, diag *diagnostics.Diagnostics) {
	checkBlock(ifStmt.Block(), diag)
	for _, elseIf := range ifStmt.AllElseIfClause() {
		checkBlock(elseIf.Block(), diag)
	}
	if elseClause := ifStmt.ElseClause(); elseClause != nil {
		checkBlock(elseClause.Block(), diag)
	}
}

// statementAlwaysExits reports whether execution cannot fall through past
// this statement in program order. A bare return always exits; an if/else
// exits only when every branch always returns.
func statementAlwaysExits(stmt parser.IStatementContext) bool {
	if stmt.ReturnStatement() != nil {
		return true
	}
	if ifStmt := stmt.IfStatement(); ifStmt != nil {
		return function.IfStmtAlwaysReturns(ifStmt)
	}
	return false
}
