// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package goroutineleak

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/analysis"
)

const (
	// leakCheck is the per-scope goroutine-leak assertion that BeforeSuite and
	// BeforeAll nodes must call first.
	leakCheck = "ShouldNotLeakGoroutines"
	// perSpecCheck is the suite-wide per-spec goroutine-leak assertion that every suite
	// must register at package scope.
	perSpecCheck = "ShouldNotLeakGoroutinesPerSpec"
	// runSpecs is the Ginkgo entry point whose presence marks a package as a test
	// suite.
	runSpecs = "RunSpecs"
	// testutilPkg is the package that implements both leak checks; its own tests
	// exercise them outside the enforced positions.
	testutilPkg = "github.com/synnaxlabs/x/testutil"
)

// setupNodes are the Ginkgo lifecycle nodes whose fixtures persist across specs and so
// must verify their own teardown with leakCheck. Per-spec nodes (BeforeEach,
// JustBeforeEach) are intentionally excluded: they are covered by perSpecCheck.
var setupNodes = set.New("BeforeSuite", "BeforeAll")

var Analyzer = &analysis.Analyzer{
	Name: "goroutineleak",
	Doc: `enforces goroutine-leak checks in Ginkgo suites.

This analyzer enforces four conventions from github.com/synnaxlabs/x/testutil:

  1. Every package that calls RunSpecs (a Ginkgo suite) must register the per-spec
     leak check at package scope, e.g. ` + "`var _ = ShouldNotLeakGoroutinesPerSpec()`" + `.

  2. Every BeforeSuite and BeforeAll node must call ShouldNotLeakGoroutines() as its
     first statement, so the goroutine snapshot is taken before the node creates its
     fixtures and the matching AfterSuite/AfterAll teardown is verified to release them.

  3. ShouldNotLeakGoroutinesPerSpec is registered exactly once per package, and only
     at package scope.

  4. ShouldNotLeakGoroutines appears nowhere except the position convention 2 puts it
     in.

A node that legitimately needs to opt out (e.g. a suite that deliberately leaves a
process-global daemon running) can be exempted with a //nolint:goroutineleak comment.`,
	Run: run,
}

func run(pass *analysis.Pass) (any, error) {
	if isTestutil(pass.Pkg.Path()) {
		return nil, nil
	}
	approved := set.Set[token.Pos]{}
	setupFuncs := set.Set[string]{}
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
				// The node was given a named function rather than a literal; its body
				// is checked when the walk reaches the function declaration.
				if name, ok := setupArgName(call); ok {
					setupFuncs.Add(name)
				}
				return true
			}
			if check := firstStmtLeakCheck(body); check != nil {
				approved.Add(check.Pos())
			} else {
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
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || !setupFuncs.Contains(fd.Name.Name) {
				continue
			}
			if check := firstStmtLeakCheck(fd.Body); check != nil {
				approved.Add(check.Pos())
			}
		}
	}
	topLevel := topLevelPerSpecCalls(pass)
	for _, call := range topLevel {
		approved.Add(call.Pos())
	}
	reportDuplicatePerSpec(pass, topLevel)
	reportStrays(pass, approved)
	checkSuiteHasPerSpec(pass, len(topLevel) > 0)
	return nil, nil
}

// reportStrays flags every leakCheck and perSpecCheck call outside the positions the
// conventions allow.
func reportStrays(pass *analysis.Pass, approved set.Set[token.Pos]) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || approved.Contains(call.Pos()) {
				return true
			}
			switch calleeName(call) {
			case leakCheck:
				pass.Report(analysis.Diagnostic{
					Pos: call.Pos(),
					End: call.End(),
					Message: leakCheck + "() may only be called as the first " +
						"statement of a BeforeSuite or BeforeAll node",
				})
			case perSpecCheck:
				pass.Report(analysis.Diagnostic{
					Pos: call.Pos(),
					End: call.End(),
					Message: perSpecCheck + "() may only be registered at package " +
						"scope via `var _ = " + perSpecCheck + "()`",
				})
			}
			return true
		})
	}
}

// reportDuplicatePerSpec flags every package-scope perSpecCheck registration after the
// first.
func reportDuplicatePerSpec(pass *analysis.Pass, calls []*ast.CallExpr) {
	if len(calls) < 2 {
		return
	}
	slices.SortFunc(calls, func(a, b *ast.CallExpr) int {
		return int(a.Pos() - b.Pos())
	})
	for _, call := range calls[1:] {
		pass.Report(analysis.Diagnostic{
			Pos: call.Pos(),
			End: call.End(),
			Message: perSpecCheck + "() is already registered in this package; " +
				"remove the duplicate registration",
		})
	}
}

// checkSuiteHasPerSpec reports every RunSpecs call in a package that does not also
// register perSpecCheck at package scope.
func checkSuiteHasPerSpec(pass *analysis.Pass, hasPerSpec bool) {
	if hasPerSpec {
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

// topLevelPerSpecCalls returns every package-scope call to perSpecCheck, e.g.
// `var _ = ShouldNotLeakGoroutinesPerSpec()`.
func topLevelPerSpecCalls(pass *analysis.Pass) []*ast.CallExpr {
	var calls []*ast.CallExpr
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				// Only a direct `var _ = ShouldNotLeakGoroutinesPerSpec()` value
				// counts: a call nested inside a Describe closure is not a
				// package-scope registration.
				for _, value := range vs.Values {
					if call, ok := value.(*ast.CallExpr); ok &&
						calleeName(call) == perSpecCheck {
						calls = append(calls, call)
					}
				}
			}
		}
	}
	return calls
}

// firstStmtLeakCheck returns the leakCheck call when it is the first statement of
// body, and nil otherwise.
func firstStmtLeakCheck(body *ast.BlockStmt) *ast.CallExpr {
	if body == nil || len(body.List) == 0 {
		return nil
	}
	exprStmt, ok := body.List[0].(*ast.ExprStmt)
	if !ok {
		return nil
	}
	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok || calleeName(call) != leakCheck {
		return nil
	}
	return call
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

// setupArgName returns the identifier of the first named-function argument of a
// BeforeSuite/BeforeAll call.
func setupArgName(call *ast.CallExpr) (string, bool) {
	for _, arg := range call.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			return ident.Name, true
		}
	}
	return "", false
}

// isTestutil reports whether path is the testutil package (or one of its test
// variants), which implements the leak checks and exercises them in arbitrary
// positions.
func isTestutil(path string) bool {
	path = strings.TrimSuffix(strings.TrimSuffix(path, "_test"), ".test")
	return path == testutilPkg
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
