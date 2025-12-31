// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package internal provides shared utilities for Go code generation plugins.
package internal

import (
	"path/filepath"
	"sort"

	"github.com/samber/lo"
)

// ToPascalCase converts snake_case to PascalCase.
func ToPascalCase(s string) string { return lo.PascalCase(s) }

// DerivePackageName extracts the package name from an output path.
// Example: "core/pkg/service/user" -> "user"
func DerivePackageName(outputPath string) string { return filepath.Base(outputPath) }

// DerivePackageAlias creates a unique alias for an imported package to avoid conflicts.
// If the base name conflicts with the current package, it prepends the parent directory.
func DerivePackageAlias(outputPath, currentPackage string) string {
	base := filepath.Base(outputPath)
	if base == currentPackage {
		parent := filepath.Base(filepath.Dir(outputPath))
		return parent + base
	}
	return base
}

// ImportManager tracks Go imports needed for generated files.
type ImportManager struct {
	external map[string]bool
	internal map[string]*InternalImport
}

// InternalImport represents an internal package import with optional alias.
type InternalImport struct {
	Path  string
	Alias string
}

// NewImportManager creates a new import manager.
func NewImportManager() *ImportManager {
	return &ImportManager{
		external: make(map[string]bool),
		internal: make(map[string]*InternalImport),
	}
}

// AddExternal adds an external package import.
func (m *ImportManager) AddExternal(path string) { m.external[path] = true }

// AddInternal adds an internal package import with an alias.
func (m *ImportManager) AddInternal(alias, path string) {
	m.internal[alias] = &InternalImport{Path: path, Alias: alias}
}

// HasImports returns true if any imports have been added.
func (m *ImportManager) HasImports() bool {
	return len(m.external) > 0 || len(m.internal) > 0
}

// ExternalImports returns sorted external import paths.
func (m *ImportManager) ExternalImports() []string {
	imports := make([]string, 0, len(m.external))
	for imp := range m.external {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports
}

// InternalImportData holds data for an internal import in templates.
type InternalImportData struct {
	Path  string
	Alias string
}

// NeedsAlias returns true if the import needs an alias in the generated code.
func (i InternalImportData) NeedsAlias() bool {
	return i.Alias != "" && i.Alias != filepath.Base(i.Path)
}

// InternalImports returns sorted internal imports.
func (m *ImportManager) InternalImports() []InternalImportData {
	imports := make([]InternalImportData, 0, len(m.internal))
	for _, imp := range m.internal {
		imports = append(imports, InternalImportData{
			Path:  imp.Path,
			Alias: imp.Alias,
		})
	}
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
	return imports
}
