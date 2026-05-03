// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package imports tracks named TypeScript imports collected during code
// generation and emits them grouped by category (Synnax workspace, external,
// internal aliases) for use in templated import blocks.
package imports

import (
	"sort"
	"strings"

	"github.com/synnaxlabs/x/set"
)

// NamedImport groups a sorted list of names imported from a single path.
type NamedImport struct {
	// Path is the module specifier the names are imported from.
	Path string
	// Names is the sorted set of names imported from Path.
	Names []string
}

// Manager accumulates named TypeScript imports keyed by module path. The zero
// value is not usable; construct with NewManager.
type Manager struct {
	specs map[string]set.Set[string]
}

// NewManager returns a Manager ready to accept AddImport calls.
func NewManager() *Manager { return &Manager{specs: make(map[string]set.Set[string])} }

// AddImport records that name is imported from path. Duplicate calls are
// no-ops.
func (m *Manager) AddImport(path, name string) {
	s, ok := m.specs[path]
	if !ok {
		s = make(set.Set[string])
		m.specs[path] = s
	}
	s.Add(name)
}

// SynnaxImports returns Synnax workspace imports (paths starting with
// @synnaxlabs/), sorted by path with each NamedImport's names sorted
// alphabetically.
func (m *Manager) SynnaxImports() []NamedImport {
	return m.filter(func(p string) bool { return strings.HasPrefix(p, "@synnaxlabs/") })
}

// ExternalNamedImports returns third-party imports — paths that are neither
// @synnaxlabs/* nor @/* — sorted by path.
func (m *Manager) ExternalNamedImports() []NamedImport {
	return m.filter(func(p string) bool {
		return !strings.HasPrefix(p, "@/") && !strings.HasPrefix(p, "@synnaxlabs/")
	})
}

// InternalNamedImports returns alias-rooted imports (paths starting with @/),
// sorted by path.
func (m *Manager) InternalNamedImports() []NamedImport {
	return m.filter(func(p string) bool { return strings.HasPrefix(p, "@/") })
}

func (m *Manager) filter(keep func(string) bool) []NamedImport {
	var out []NamedImport
	for path, s := range m.specs {
		if len(s) == 0 || !keep(path) {
			continue
		}
		names := make([]string, 0, len(s))
		for n := range s {
			names = append(names, n)
		}
		sort.Strings(names)
		out = append(out, NamedImport{Path: path, Names: names})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
