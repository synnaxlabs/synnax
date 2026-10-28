// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package exports

import (
	"go/ast"
	"go/types"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/packages"
)

// templateTypes returns the named types whose values a text or HTML template can
// reach from the data passed to Execute or ExecuteTemplate. A template calls their
// exported methods by name, which no Go reference shows.
func templateTypes(pkgs []*packages.Package) set.Set[*types.TypeName] {
	found := set.Set[*types.TypeName]{}
	seen := set.Set[types.Type]{}
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 || !isTemplateExecute(p, call) {
					return true
				}
				data := call.Args[len(call.Args)-1]
				reachTemplate(p.TypesInfo.TypeOf(data), seen, found)
				if lit, ok := ast.Unparen(data).(*ast.CompositeLit); ok {
					for _, elt := range lit.Elts {
						if kv, ok := elt.(*ast.KeyValueExpr); ok {
							elt = kv.Value
						}
						reachTemplate(p.TypesInfo.TypeOf(elt), seen, found)
					}
				}
				return true
			})
		}
	}
	return found
}

func isTemplateExecute(p *packages.Package, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	fn, ok := p.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}
	path := fn.Pkg().Path()
	return (path == "text/template" || path == "html/template") &&
		(fn.Name() == "Execute" || fn.Name() == "ExecuteTemplate")
}

// reachTemplate adds each named type that a template can reach from t through fields,
// elements, and the results of exported methods.
func reachTemplate(
	t types.Type,
	seen set.Set[types.Type],
	found set.Set[*types.TypeName],
) {
	reach(t, seen, func(tn *types.TypeName) {
		if found.Contains(tn) {
			return
		}
		found.Add(tn)
		ms := types.NewMethodSet(types.NewPointer(tn.Type()))
		for sel := range ms.Methods() {
			if sel.Obj().Exported() {
				reachTemplate(sel.Obj().Type(), seen, found)
			}
		}
	})
}
