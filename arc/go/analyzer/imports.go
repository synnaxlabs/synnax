// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
)

// checkImportOrder reports any import statement that appears after a
// non-import top-level declaration. Imports must form a contiguous prefix
// of the program.
func checkImportOrder(ctx acontext.Context[parser.IProgramContext]) {
	sawNonImport := false
	for _, item := range ctx.AST.AllTopLevelItem() {
		stmt := item.ImportStatement()
		if stmt == nil {
			sawNonImport = true
			continue
		}
		if sawNonImport {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				stmt,
				"import statements must appear before any other top-level declarations",
			))
		}
	}
}

// collectImports installs a KindModuleAlias child on the program root for
// each `import` statement. The alias's Target points at the underlying
// module symbol (looked up in the ambient prelude). Subsequent dotted
// lookups (`t.now`) resolve through the alias by ordinary scope walk and
// member descent — no separate import gate is needed.
//
// Diagnoses duplicate aliases and unknown module paths.
func collectImports(ctx acontext.Context[parser.IProgramContext]) {
	root := ctx.Scope.Root()
	ambient := ambientOf(root)
	for _, entry := range parser.Imports(ctx.AST) {
		module := lookupModule(ambient, entry.Path)
		if module == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry.AST,
				"unknown module %q",
				entry.Path,
			))
			continue
		}
		alias := symbol.Symbol{
			Name:   entry.Alias,
			Kind:   symbol.KindModuleAlias,
			AST:    entry.AST,
			Target: module,
		}
		if _, err := root.Add(ctx, alias); err != nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry.AST,
				"duplicate import: %q is already imported",
				entry.Alias,
			))
		}
	}
}

// reportUnusedImports flags every alias child of the program root that was
// never consulted by resolution.
func reportUnusedImports(ctx acontext.Context[parser.IProgramContext]) {
	root := ctx.Scope.Root()
	for _, child := range root.Children() {
		if child.Kind != symbol.KindModuleAlias {
			continue
		}
		if child.Used {
			continue
		}
		ctx.Diagnostics.Add(diagnostics.Errorf(
			child.AST,
			"imported module %q is unused",
			child.Name,
		).WithCode(codes.UnusedImport))
	}
}

// ambientOf returns the KindAmbient parent of the program root if any,
// otherwise nil. The ambient holds the STL modules available for import.
func ambientOf(root *symbol.Symbol) *symbol.Symbol {
	if root.Parent != nil && root.Parent.Kind == symbol.KindAmbient {
		return root.Parent
	}
	return nil
}

// lookupModule searches the ambient prelude for a KindModule child with the
// given canonical path. Internal modules are skipped so user `import`
// statements cannot reach compiler-only modules (channels, strings,
// series, stateful). Returns nil when ambient is nil, the module is not
// present, or the matched module is Internal.
func lookupModule(ambient *symbol.Symbol, path string) *symbol.Symbol {
	if ambient == nil {
		return nil
	}
	child := ambient.FindChild(path)
	if child == nil || child.Kind != symbol.KindModule || child.Internal {
		return nil
	}
	return child
}
