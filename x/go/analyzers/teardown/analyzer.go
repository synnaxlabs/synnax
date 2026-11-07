// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package teardown

import (
	"go/ast"
	"go/types"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports each Close call in a Ginkgo teardown node.
var Analyzer = &analysis.Analyzer{
	Name: "teardown",
	Doc: `forbids Close calls in AfterEach, JustAfterEach, AfterAll, and AfterSuite.

Cleanup belongs next to the open. MustOpen(v, err) or DeferClose(v) from
github.com/synnaxlabs/x/testutil registers DeferCleanup(v.Close) in the node that opens
v, and DeferCleanup runs in reverse order of registration.`,
	Run: run,
}

var (
	teardownNodes = set.New(
		"AfterEach",
		"JustAfterEach",
		"AfterAll",
		"AfterSuite",
		"SynchronizedAfterSuite",
	)
	errorType = types.Universe.Lookup("error").Type()
)

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			node := funcName(call)
			if !teardownNodes.Contains(node) {
				return true
			}
			for _, arg := range call.Args {
				if lit, ok := arg.(*ast.FuncLit); ok {
					reportCloses(pass, lit.Body, node)
				}
			}
			return false
		})
	}
	return nil, nil
}

func reportCloses(pass *analysis.Pass, body *ast.BlockStmt, node string) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if ok && isClose(pass, call) {
			pass.Reportf(
				call.Pos(),
				"close in the node that opens the value, with MustOpen or DeferClose, not in %s",
				node,
			)
		}
		return true
	})
}

// isClose reports whether call is a Close() error method call, the io.Closer method.
func isClose(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Close" || len(call.Args) != 0 {
		return false
	}
	sig, ok := pass.TypesInfo.TypeOf(sel).(*types.Signature)
	if !ok || sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return false
	}
	return types.Identical(sig.Results().At(0).Type(), errorType)
}

func funcName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}
	return ""
}
