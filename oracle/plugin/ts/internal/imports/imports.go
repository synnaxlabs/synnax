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
	"path"
	"sort"
	"strings"

	"github.com/synnaxlabs/x/set"
)

// NamedImport groups a sorted list of names imported from a single path, or binds the
// whole module at that path to one namespace identifier.
type NamedImport struct {
	// Path is the module specifier the names are imported from.
	Path string
	// Namespace, when set, renders the import as `import * as Namespace from Path`, and
	// Names is empty.
	Namespace string
	// Names is the sorted set of names imported from Path. An aliased name reads
	// "name as alias".
	Names []string
}

// Key identifies a name that the module at Path exports.
type Key struct {
	// Path is the module specifier.
	Path string
	// Name is the exported name.
	Name string
}

// Manager accumulates named TypeScript imports keyed by module path. The zero value is
// not usable; construct with NewManager.
type Manager struct {
	specs      map[string]set.Set[string]
	namespaces map[string]string
	// bound holds the paths that declare each identifier of the file.
	bound map[string]set.Set[string]
}

// NewManager returns a Manager ready to accept AddImport calls.
func NewManager() *Manager {
	return &Manager{
		specs:      make(map[string]set.Set[string]),
		namespaces: make(map[string]string),
		bound:      make(map[string]set.Set[string]),
	}
}

// AddImport records that name is imported from path. Duplicate calls are no-ops.
func (m *Manager) AddImport(path, name string) { m.add(path, name, name) }

// AddAliasedImport records that name is imported from path as alias.
func (m *Manager) AddAliasedImport(path, name, alias string) {
	m.add(path, name+" as "+alias, alias)
}

func (m *Manager) add(path, spec, ident string) {
	s, ok := m.specs[path]
	if !ok {
		s = make(set.Set[string])
		m.specs[path] = s
	}
	s.Add(spec)
	m.bind(ident, path)
}

// AddNamespaceImport records that the whole module at path is imported as ident.
func (m *Manager) AddNamespaceImport(path, ident string) {
	m.namespaces[path] = ident
	m.bind(ident, path)
}

func (m *Manager) bind(ident, path string) {
	s, ok := m.bound[ident]
	if !ok {
		s = make(set.Set[string])
		m.bound[ident] = s
	}
	s.Add(path)
}

// Conflicts returns, for each identifier that two or more paths declare, the sorted
// paths that declare it. It returns nil when every identifier has one path.
func (m *Manager) Conflicts() map[string][]string {
	var out map[string][]string
	for ident, paths := range m.bound {
		if len(paths) < 2 {
			continue
		}
		if out == nil {
			out = make(map[string][]string)
		}
		s := paths.Slice()
		sort.Strings(s)
		out[ident] = s
	}
	return out
}

// Aliases returns the aliases that resolve conflicts, as Conflicts returns them. A
// path whose last segment is the conflicting name keeps the name. Each other path
// prefixes the name with its last segment, so "color" from "@synnaxlabs/x" becomes
// "xcolor".
func Aliases(conflicts map[string][]string) map[Key]string {
	aliases := make(map[Key]string)
	for name, paths := range conflicts {
		for _, p := range paths {
			segment := path.Base(strings.TrimSuffix(p, "/types.gen"))
			if segment == name {
				continue
			}
			aliases[Key{Path: p, Name: name}] = segment + name
		}
	}
	return aliases
}

// SynnaxImports returns Synnax workspace imports (paths starting with @synnaxlabs/),
// sorted by path with each NamedImport's names sorted alphabetically.
func (m *Manager) SynnaxImports() []NamedImport {
	return m.filter(func(p string) bool { return strings.HasPrefix(p, "@synnaxlabs/") })
}

// ExternalNamedImports returns third-party imports, whose paths match neither
// "@synnaxlabs/*" nor "@/*", sorted by path.
func (m *Manager) ExternalNamedImports() []NamedImport {
	return m.filter(func(p string) bool {
		return !strings.HasPrefix(p, "@/") && !strings.HasPrefix(p, "@synnaxlabs/")
	})
}

// InternalNamedImports returns alias-rooted imports (paths starting with @/), sorted by
// path.
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
	for path, ident := range m.namespaces {
		if keep(path) {
			out = append(out, NamedImport{Path: path, Namespace: ident})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
