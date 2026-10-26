// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package suiteconfig

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

const (
	// runSpecs is the Ginkgo entry point of a test suite.
	runSpecs = "RunSpecs"
	// suiteConfig is the github.com/synnaxlabs/x/testutil function that returns the
	// shared suite configuration.
	suiteConfig = "SuiteConfig"
)

// Analyzer reports each RunSpecs call that does not pass SuiteConfig().
var Analyzer = &analysis.Analyzer{
	Name: "suiteconfig",
	Doc: `enforces the shared Ginkgo configuration in every test suite.

Every RunSpecs call must pass SuiteConfig() from github.com/synnaxlabs/x/testutil, e.g.
` + "`RunSpecs(t, \"Channel Suite\", SuiteConfig())`" + `. CI runs go test without
Ginkgo flags so that it can cache results, so a suite without SuiteConfig() skips the
checks it sets.`,
	Run: run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || calleeName(call) != runSpecs || passesSuiteConfig(call) {
				return true
			}
			pass.Report(analysis.Diagnostic{
				Pos: call.Pos(),
				End: call.End(),
				Message: runSpecs + " must pass " + suiteConfig +
					"() so the suite runs with the shared Ginkgo configuration",
			})
			return true
		})
	}
	return nil, nil
}

// passesSuiteConfig reports whether one of the arguments of call is a call to
// suiteConfig.
func passesSuiteConfig(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		argCall, ok := arg.(*ast.CallExpr)
		if ok && calleeName(argCall) == suiteConfig {
			return true
		}
	}
	return false
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
