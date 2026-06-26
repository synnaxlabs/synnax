// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package leak

import (
	"go/ast"
	"go/token"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/analysis"
)

const (
	// leakCheck is the per-scope goroutine-leak assertion that BeforeSuite and BeforeAll
	// nodes must call first.
	leakCheck = "ShouldNotLeakGoroutines"
	// perSpecCheck is the suite-wide per-spec goroutine-leak assertion that every suite
	// must register at package scope.
	perSpecCheck = "ShouldNotLeakGoroutinesPerSpec"
	// runSpecs is the Ginkgo entry point whose presence marks a package as a test suite.
	runSpecs = "RunSpecs"
)

// setupNodes are the Ginkgo lifecycle nodes whose fixtures persist across specs and so
// must verify their own teardown with leakCheck. Per-spec nodes (BeforeEach,
// JustBeforeEach) are intentionally excluded: they are covered by perSpecCheck.
var setupNodes = set.New("BeforeSuite", "BeforeAll")

var Analyzer = &analysis.Analyzer{
	Name: "leak",
	Doc: `enforces goroutine-leak checks in Ginkgo suites.

This analyzer enforces two conventions from github.com/synnaxlabs/x/testutil:

  1. Every package that calls RunSpecs (a Ginkgo suite) must register the per-spec
     leak check at package scope, e.g. ` + "`var _ = ShouldNotLeakGoroutinesPerSpec()`" + `.

  2. Every BeforeSuite and BeforeAll node must call ShouldNotLeakGoroutines() as its
     first statement, so the goroutine snapshot is taken before the node creates its
     fixtures and the matching AfterSuite/AfterAll teardown is verified to release them.

A node that legitimately needs to opt out (e.g. a suite that deliberately leaves a
process-global daemon running) can be exempted with a //nolint:leak comment.`,
	Run: run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if !setupNodes.Contains(calleeName(call)) {
				return true
			}
			body := setupBody(call)
			if body == nil {
				// The node was given a named function rather than a literal; its body is
				// not visible here, so there is nothing to check.
				return true
			}
			if !firstStmtIsLeakCheck(body) {
				pass.Report(analysis.Diagnostic{
					Pos: call.Pos(),
					End: call.End(),
					Message: calleeName(call) + " must call " + leakCheck +
						"() as its first statement so the goroutine snapshot is taken " +
						"before fixtures are created",
				})
			}
			return true
		})
	}
	checkSuiteHasPerSpec(pass)
	return nil, nil
}

// checkSuiteHasPerSpec reports every RunSpecs call in a package that does not also
// register perSpecCheck at package scope.
func checkSuiteHasPerSpec(pass *analysis.Pass) {
	if hasTopLevelPerSpec(pass) {
		return
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || calleeName(call) != runSpecs {
				return true
			}
			pass.Report(analysis.Diagnostic{
				Pos: call.Pos(),
				End: call.End(),
				Message: "test suite calls " + runSpecs + " but does not register " +
					perSpecCheck + "() at package scope; add `var _ = " +
					perSpecCheck + "()`",
			})
			return true
		})
	}
}

// hasTopLevelPerSpec reports whether any file in the package contains a package-scope
// call to perSpecCheck, e.g. `var _ = ShouldNotLeakGoroutinesPerSpec()`.
func hasTopLevelPerSpec(pass *analysis.Pass) bool {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			found := false
			ast.Inspect(gen, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok && calleeName(call) == perSpecCheck {
					found = true
					return false
				}
				return true
			})
			if found {
				return true
			}
		}
	}
	return false
}

// firstStmtIsLeakCheck reports whether the first statement of body is a bare call to
// leakCheck.
func firstStmtIsLeakCheck(body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}
	exprStmt, ok := body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	return calleeName(call) == leakCheck
}

// setupBody returns the body of the first function-literal argument of a
// BeforeSuite/BeforeAll call, or nil if the call has no literal argument (e.g. it was
// handed a named function, whose body cannot be inspected here).
func setupBody(call *ast.CallExpr) *ast.BlockStmt {
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.FuncLit); ok {
			return lit.Body
		}
	}
	return nil
}

// calleeName returns the called function's identifier for both unqualified F(...) (the
// dot-imported case) and qualified pkg.F(...) calls. It returns "" for any other callee
// expression.
func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}
	return ""
}
