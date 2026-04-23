// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package unused emits diagnostics for dead code: declarations that are never
// referenced (ARC51xx) and statements that can never execute (ARC52xx).
//
// Declaration checks walk the scope tree and rely on other analyzer passes
// having set Referenced on every scope that was resolved from a use-site.
// Statement-level unreachable-code checks walk function body ASTs directly.
package unused

import (
	"strings"

	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/function"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// Analyze walks the program and emits warnings for unreferenced declarations
// (ARC51xx) and unreachable statements and stages (ARC52xx). It should run
// after analyzeDeclarations so that every use-site has had a chance to mark
// its target scope Referenced.
func Analyze(ctx context.Context[parser.IProgramContext]) {
	walk(ctx.Scope, ctx.Diagnostics)
	analyzeUnreachableCode(ctx)
	analyzeSequenceReachability(ctx)
}

func walk(scope *symbol.Scope, diag *diagnostics.Diagnostics) {
	for _, child := range scope.Children {
		if d := unusedDiagnostic(child); d != nil {
			diag.Add(*d)
		}
		walk(child, diag)
	}
}

// unusedDiagnostic returns a pointer to a diagnostic for the scope if it is an
// unreferenced declaration subject to an ARC51xx rule, otherwise nil. Names
// beginning with an underscore are treated as intentionally unused, and
// declarations whose type failed to resolve are skipped to avoid piling a
// warning on top of the unrelated error the user is already seeing.
func unusedDiagnostic(scope *symbol.Scope) *diagnostics.Diagnostic {
	if scope.AST == nil {
		return nil
	}
	if scope.Referenced {
		return nil
	}
	if strings.HasPrefix(scope.Name, "_") {
		return nil
	}
	if !scope.Type.IsValid() {
		return nil
	}
	switch scope.Kind {
	case symbol.KindFunction:
		d := diagnostics.
			Warningf(scope.AST, "uncalled function '%s'", scope.Name).
			WithCode(codes.UncalledFunction).
			WithNote("prefix the name with an underscore to suppress this warning")
		return &d
	case symbol.KindGlobalConstant:
		d := diagnostics.
			Warningf(scope.AST, "unused global constant '%s'", scope.Name).
			WithCode(codes.UnusedGlobalConstant).
			WithNote("prefix the name with an underscore to suppress this warning")
		return &d
	case symbol.KindVariable, symbol.KindStatefulVariable:
		label, ok := variableLabel(scope)
		if !ok {
			return nil
		}
		d := diagnostics.
			Warningf(scope.AST, "unused %s '%s'", label, scope.Name).
			WithCode(codes.UnusedVariable).
			WithNote("prefix the name with an underscore to suppress this warning")
		return &d
	default:
		return nil
	}
}

// variableLabel returns the human-readable label for a variable-kind scope for
// use in the unused-variable diagnostic message. The second return value is
// false if this scope's kind does not have a variable-flavored label.
func variableLabel(scope *symbol.Scope) (string, bool) {
	switch scope.Kind {
	case symbol.KindStatefulVariable:
		return "stateful variable", true
	case symbol.KindVariable:
		if scope.SourceID != nil && scope.Type.Kind == basetypes.KindChan {
			return "channel alias", true
		}
		return "variable", true
	default:
		return "", false
	}
}

// analyzeUnreachableCode walks each top-level function body and emits an
// ARC5203 warning for the first statement in any block that is preceded by a
// statement that always returns. Only function bodies are analyzed: stage
// bodies are reactive parallel flows with no notion of statement ordering.
func analyzeUnreachableCode(ctx context.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if fnDecl := item.FunctionDeclaration(); fnDecl != nil {
			checkBlock(fnDecl.Block(), ctx.Diagnostics)
		}
	}
}

// checkBlock emits at most one ARC5203 warning per block and recurses into
// nested if / else-if / else blocks. It stops scanning a block after emitting
// a warning so callers are not overwhelmed by cascades in a long dead tail.
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
			next := statements[i+1]
			diag.Add(diagnostics.
				Warningf(next, "unreachable code: previous statement always returns").
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

// statementAlwaysExits reports whether execution cannot fall through past this
// statement in program order. A bare return always exits; an if/else exits
// only when every branch always returns (handled by IfStmtAlwaysReturns).
func statementAlwaysExits(stmt parser.IStatementContext) bool {
	if stmt.ReturnStatement() != nil {
		return true
	}
	if ifStmt := stmt.IfStatement(); ifStmt != nil {
		return function.IfStmtAlwaysReturns(ifStmt)
	}
	return false
}
