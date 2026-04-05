// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package setlint

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "setlint",
	Doc: `detects map[T]struct{} and map[T]bool patterns that should use set.Set[T].

This analyzer finds type expressions where a map is used as a set (with struct{} or bool
values) and reports that they should be replaced with set.Set[T] from
github.com/synnaxlabs/x/set.`,
	Run: run,
}

func run(pass *analysis.Pass) (any, error) {
	if isSetPackage(pass.Pkg.Path()) {
		return nil, nil
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			mt, ok := n.(*ast.MapType)
			if !ok {
				return true
			}
			keyStr := nodeString(pass.Fset, mt.Key)
			if isEmptyStruct(mt.Value) {
				pass.Report(analysis.Diagnostic{
					Pos:     mt.Pos(),
					End:     mt.End(),
					Message: fmt.Sprintf("map[%s]struct{} can be replaced with set.Set[%s]", keyStr, keyStr),
				})
			} else if isBool(mt.Value) {
				pass.Report(analysis.Diagnostic{
					Pos:     mt.Pos(),
					End:     mt.End(),
					Message: fmt.Sprintf("map[%s]bool can be replaced with set.Set[%s]", keyStr, keyStr),
				})
			}
			return true
		})
	}
	return nil, nil
}

func isSetPackage(pkgPath string) bool {
	return pkgPath == "github.com/synnaxlabs/x/set"
}

func isEmptyStruct(expr ast.Expr) bool {
	st, ok := expr.(*ast.StructType)
	return ok && (st.Fields == nil || len(st.Fields.List) == 0)
}

func isBool(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "bool"
}

func nodeString(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}
