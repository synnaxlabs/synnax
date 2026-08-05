// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package suite

import (
	"go/ast"
	"path/filepath"
	"slices"
	"strings"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/analysis"
)

const (
	runSpecs        = "RunSpecs"
	registerFail    = "RegisterFailHandler"
	suiteFileSuffix = "_suite_test.go"
	// testutilPkg re-registers the fail handler inside specs to test the failure path
	// of its own matchers.
	testutilPkg = "github.com/synnaxlabs/x/testutil"
)

// containers are the Ginkgo calls that register specs when assigned at package scope.
var containers = set.New(
	"Describe", "FDescribe", "PDescribe", "XDescribe",
	"DescribeTable", "FDescribeTable", "PDescribeTable", "XDescribeTable",
	"Context", "It", "FIt", "PIt", "XIt", "Specify",
)

var Analyzer = &analysis.Analyzer{
	Name: "suite",
	Doc: `enforces the Ginkgo suite bootstrap layout.

This analyzer enforces three conventions:

  1. RunSpecs and RegisterFailHandler live only in the package's suite file, named
     <package>_suite_test.go (the external test package "foo_test" owns foo_suite_test.go).

  2. A package calls RunSpecs at most once.

  3. An external test package that registers specs at package scope (var _ = Describe)
     must also call RunSpecs, otherwise its specs silently never run.`,
	Run: run,
}

func run(pass *analysis.Pass) (any, error) {
	if isTestutil(pass.Pkg.Path()) {
		return nil, nil
	}
	base := strings.TrimSuffix(pass.Pkg.Name(), "_test")
	want := base + suiteFileSuffix
	var runSpecsCalls []*ast.CallExpr
	for _, file := range pass.Files {
		name := filepath.Base(pass.Fset.File(file.Pos()).Name())
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			callee := calleeName(call)
			if callee != runSpecs && callee != registerFail {
				return true
			}
			if callee == runSpecs {
				runSpecsCalls = append(runSpecsCalls, call)
			}
			if name != want {
				pass.Report(analysis.Diagnostic{
					Pos:     call.Pos(),
					End:     call.End(),
					Message: callee + " must live in the suite file " + want,
				})
			}
			return true
		})
	}
	reportExtraRunSpecs(pass, runSpecsCalls)
	if len(runSpecsCalls) == 0 {
		reportOrphanedSpecs(pass, want)
	}
	return nil, nil
}

// reportExtraRunSpecs flags every RunSpecs call after the first: a test binary runs
// one Ginkgo suite per package.
func reportExtraRunSpecs(pass *analysis.Pass, calls []*ast.CallExpr) {
	if len(calls) < 2 {
		return
	}
	slices.SortFunc(calls, func(a, b *ast.CallExpr) int {
		return int(a.Pos() - b.Pos())
	})
	for _, call := range calls[1:] {
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: "package already calls " + runSpecs + "; a package runs one suite",
		})
	}
}

// reportOrphanedSpecs flags the first package-scope spec container in an external test
// package that never calls RunSpecs. White-box spec files (package foo, not foo_test)
// are skipped: their suite bootstrap legitimately lives in the sibling external
// package.
func reportOrphanedSpecs(pass *analysis.Pass, want string) {
	if !strings.HasSuffix(pass.Pkg.Name(), "_test") {
		return
	}
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			var found *ast.CallExpr
			ast.Inspect(gen, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok && found == nil &&
					containers.Contains(calleeName(call)) {
					found = call
					return false
				}
				return found == nil
			})
			if found != nil {
				pass.Report(analysis.Diagnostic{
					Pos: found.Pos(),
					End: found.End(),
					Message: "package registers Ginkgo specs but never calls " +
						runSpecs + "; add " + want,
				})
				return
			}
		}
	}
}

// isTestutil reports whether path is the testutil package or one of its test variants.
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
