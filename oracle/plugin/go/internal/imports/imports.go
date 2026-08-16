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
	"regexp"
	"sort"
	"strings"

	"github.com/synnaxlabs/x/set"
)

// Manager tracks Go imports needed for generated files.
type Manager struct {
	external set.Set[string]
	internal map[string]*internalImport
}

type internalImport struct {
	Path  string
	Alias string
}

// NewManager creates a new import manager.
func NewManager() *Manager {
	return &Manager{
		external: make(set.Set[string]),
		internal: make(map[string]*internalImport),
	}
}

// AddExternal adds an external package import.
func (m *Manager) AddExternal(path string) { m.external.Add(path) }

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

// stdlib reports whether an import path belongs to the standard library: its
// first segment carries no dot.
func stdlib(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

// StdImports returns sorted standard-library import paths.
func (m *Manager) StdImports() []string {
	result := make([]string, 0, len(m.external))
	for imp := range m.external {
		if stdlib(imp) {
			result = append(result, imp)
		}
	}
	sort.Strings(result)
	return result
}

// NonStdImports returns every non-standard-library import — plain external
// paths and aliased internal ones alike — sorted by path. An external import
// whose package name is claimed by an internal alias is dropped: the internal
// package shadows it, and selectors resolve through the alias instead.
func (m *Manager) NonStdImports() []InternalImportData {
	result := make([]InternalImportData, 0, len(m.external)+len(m.internal))
	for imp := range m.external {
		if stdlib(imp) {
			continue
		}
		if shadow := m.internal[filepath.Base(imp)]; shadow == nil ||
			shadow.Path == imp {
			result = append(result, InternalImportData{Path: imp})
		}
	}
	for _, imp := range m.internal {
		if m.external.Contains(imp.Path) {
			continue
		}
		result = append(result, InternalImportData{Path: imp.Path, Alias: imp.Alias})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

// InternalImportData holds data for an internal import in templates.
type InternalImportData struct {
	Path  string
	Alias string
}

// versionDir matches version sub-directory names ("v0", "v12").
var versionDir = regexp.MustCompile(`^v\d+$`)

// NeedsAlias returns true if the import renders with an explicit alias.
// Version directories always do (v6 "…/types/v6"), even when the alias
// matches the package name, so the qualifier's origin stays visible.
func (i InternalImportData) NeedsAlias() bool {
	if i.Alias == "" {
		return false
	}
	base := filepath.Base(i.Path)
	return i.Alias != base || versionDir.MatchString(base)
}

// InternalImports returns sorted internal imports, excluding any that are already
// in the external imports list to avoid duplicates.
func (m *Manager) InternalImports() []InternalImportData {
	result := make([]InternalImportData, 0, len(m.internal))
	for _, imp := range m.internal {
		if m.external.Contains(imp.Path) {
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
