// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package imports provides import statement analysis for Arc programs.
package imports

import (
	"context"
	"strings"
	"sync"

	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/errors"
)

// TrackedResolver wraps a Resolver to track whether it was accessed.
type TrackedResolver struct {
	symbol.Resolver
	used bool
	mu   sync.Mutex
}

// Resolve delegates to the wrapped resolver and marks as used.
func (t *TrackedResolver) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	t.mu.Lock()
	t.used = true
	t.mu.Unlock()
	return t.Resolver.Resolve(ctx, name)
}

// Search delegates to the wrapped resolver and marks as used.
func (t *TrackedResolver) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	t.mu.Lock()
	t.used = true
	t.mu.Unlock()
	return t.Resolver.Search(ctx, term)
}

// Used returns whether this resolver was accessed.
func (t *TrackedResolver) Used() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.used
}

// Import tracks a single import for unused detection.
type Import struct {
	Path      string
	Qualifier string
	Alias     string // non-empty if the import uses "as alias" syntax
	Resolver  *TrackedResolver
	AST       parser.IImportItemContext
}

// Imports tracks all imports in a program for unused detection.
type Imports struct {
	imports []*Import
}

// Analyze processes the import block and adds module symbols to the scope.
// The modules parameter maps module paths to their symbol resolvers.
func Analyze(
	ctx acontext.Context[parser.IProgramContext],
	modules map[string]symbol.Resolver,
) (*Imports, bool) {
	imports := &Imports{}
	importBlock := ctx.AST.ImportBlock()
	if importBlock == nil {
		return imports, true
	}

	seen := make(map[string]string) // qualifier -> path (for duplicate detection)

	for _, item := range importBlock.AllImportItem() {
		modulePath := item.ModulePath()
		if modulePath == nil {
			continue
		}

		// Build full path from module path identifiers
		identifiers := modulePath.AllIDENTIFIER()
		var pathParts []string
		for _, id := range identifiers {
			pathParts = append(pathParts, id.GetText())
		}
		fullPath := strings.Join(pathParts, ".")

		// Get qualifier - use alias if provided, otherwise last segment of path
		var qualifier string
		var alias string
		if item.AS() != nil && item.IDENTIFIER() != nil {
			alias = item.IDENTIFIER().GetText()
			qualifier = alias
		} else {
			qualifier = pathParts[len(pathParts)-1]
		}

		// Check for duplicate qualifiers
		if existingPath, ok := seen[qualifier]; ok {
			ctx.Diagnostics.Add(diagnostics.Error(
				errors.Newf("duplicate import: %q conflicts with %q (both use qualifier %q)",
					fullPath, existingPath, qualifier),
				item,
			))
			return imports, false
		}
		seen[qualifier] = fullPath

		// Look up module in registry
		resolver, ok := modules[fullPath]
		if !ok {
			ctx.Diagnostics.Add(diagnostics.Error(
				errors.Newf("unknown module %q", fullPath),
				item,
			))
			return imports, false
		}

		// Wrap resolver to track usage
		tracked := &TrackedResolver{Resolver: resolver}

		// Add module symbol to scope
		if _, err := ctx.Scope.Add(ctx, symbol.Symbol{
			Name:     qualifier,
			Kind:     symbol.KindModule,
			Resolver: tracked,
			AST:      item,
		}); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, item))
			return imports, false
		}

		imports.imports = append(imports.imports, &Import{
			Path:      fullPath,
			Qualifier: qualifier,
			Alias:     alias,
			Resolver:  tracked,
			AST:       item,
		})
	}

	return imports, true
}

// CheckUnused reports warnings for any unused imports.
func (i *Imports) CheckUnused(ctx acontext.Context[parser.IProgramContext]) {
	for _, imp := range i.imports {
		if !imp.Resolver.Used() {
			ctx.Diagnostics.Add(diagnostics.Warningf(
				imp.AST,
				"imported %q but never used", imp.Path,
			))
		}
	}
}
