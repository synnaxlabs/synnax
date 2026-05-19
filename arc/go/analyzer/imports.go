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
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// collectImports populates the root scope's ImportSet and diagnoses
// duplicates and unknown modules. Must run before any resolution pass.
func collectImports(ctx acontext.Context[parser.IProgramContext]) {
	imports := symbol.NewImportSet()
	root := ctx.Scope.Root()
	root.Imports = imports

	known := knownModuleSet(root.GlobalResolver)

	for _, entry := range parser.Imports(ctx.AST) {
		rec := symbol.ImportRecord{
			Path:  entry.Path,
			Alias: entry.Alias,
			AST:   entry.AST,
		}
		if duplicate := imports.Add(rec); duplicate {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry.AST,
				"duplicate import: %q is already imported",
				entry.Alias,
			))
			continue
		}
		if !known.Contains(entry.Path) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry.AST,
				"unknown module %q",
				entry.Path,
			))
			imports.MarkUsed(entry.Alias)
		}
	}
}

// knownModuleSet returns every module name the resolver exposes, used to
// diagnose imports of unknown modules.
func knownModuleSet(r symbol.Resolver) set.Set[string] {
	if r == nil {
		return set.New[string]()
	}
	return set.New(symbol.ListModules(r)...)
}

// reportUnusedImports flags every import that was never consulted.
func reportUnusedImports(ctx acontext.Context[parser.IProgramContext]) {
	root := ctx.Scope.Root()
	if root.Imports == nil {
		return
	}
	for _, rec := range root.Imports.Unused() {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			rec.AST,
			"imported module %q is unused",
			rec.Alias,
		))
	}
}
