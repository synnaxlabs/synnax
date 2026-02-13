// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package imports tracks and manages Go import statements for generated code files.
package imports

import (
	"path/filepath"
	"sort"
)

// Manager tracks Go imports needed for generated files.
type Manager struct {
	external map[string]bool
	internal map[string]*internalImport
}

type internalImport struct {
	Path  string
	Alias string
}

// NewManager creates a new import manager.
func NewManager() *Manager {
	return &Manager{
		external: make(map[string]bool),
		internal: make(map[string]*internalImport),
	}
}

// AddExternal adds an external package import.
func (m *Manager) AddExternal(path string) { m.external[path] = true }

// AddInternal adds an internal package import with an alias.
func (m *Manager) AddInternal(alias, path string) {
	m.internal[alias] = &internalImport{Path: path, Alias: alias}
}

// AddImport routes imports to AddExternal or AddInternal based on category.
func (m *Manager) AddImport(category, path, alias string) {
	if category == "external" || alias == "" {
		m.AddExternal(path)
	} else {
		m.AddInternal(alias, path)
	}
}

// HasImports returns true if any imports have been added.
func (m *Manager) HasImports() bool {
	return len(m.external) > 0 || len(m.internal) > 0
}

// ExternalImports returns sorted external import paths.
func (m *Manager) ExternalImports() []string {
	result := make([]string, 0, len(m.external))
	for imp := range m.external {
		result = append(result, imp)
	}
	sort.Strings(result)
	return result
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

// InternalImports returns sorted internal imports, excluding any that are already
// in the external imports list to avoid duplicates.
func (m *Manager) InternalImports() []InternalImportData {
	result := make([]InternalImportData, 0, len(m.internal))
	for _, imp := range m.internal {
		if m.external[imp.Path] {
			continue
		}
		result = append(result, InternalImportData{
			Path:  imp.Path,
			Alias: imp.Alias,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}
