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
	"context"

	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/formatter"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
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
// module symbol (looked up in the ambient prelude, then the root's module
// resolver). Subsequent dotted lookups (`t.now`) resolve through the alias by
// ordinary scope walk and member descent — no separate import gate is needed.
//
// Diagnoses duplicate aliases and unknown module paths. A failed import still binds
// its alias, with no Target, so its uses report nothing more.
func collectImports(ctx acontext.Context[parser.IProgramContext]) {
	root := ctx.Scope.Root()
	ambient := ambientOf(root)
	for _, entry := range parser.Imports(ctx.AST) {
		module, err := lookupModule(ctx, root, ambient, entry.Path)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, entry.AST))
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
		if child.Used || child.Target == nil {
			continue
		}
		d := codes.UnusedImport.At(child.AST, child.Name)
		if item, ok := child.AST.(parser.IImportItemContext); ok {
			d = d.WithEdits("Remove unused import", formatter.RemoveImport(item))
		}
		ctx.Diagnostics.Add(d)
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

// lookupModule finds the module at path in the ambient prelude, then through the
// root's module resolver. Internal modules stay hidden from user imports.
func lookupModule(
	ctx context.Context,
	root, ambient *symbol.Symbol,
	path string,
) (*symbol.Symbol, error) {
	if ambient != nil {
		child := ambient.FindChild(path)
		if child != nil && child.Kind == symbol.KindModule && !child.Internal {
			return child, nil
		}
	}
	module, err := root.ResolveModule(ctx, path)
	if errors.Is(err, symbol.ErrModuleNotFound) {
		return nil, errors.Newf("unknown module %q", path)
	}
	return module, err
}
