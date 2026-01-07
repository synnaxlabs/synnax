// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imports

import (
	"sort"
)

type Entry struct {
	Path  string
	Alias string
}

type Category struct {
	Name    string
	entries map[string]Entry
}

func newCategory(name string) *Category {
	return &Category{
		Name:    name,
		entries: make(map[string]Entry),
	}
}

func (c *Category) Add(path string, alias ...string) {
	e := Entry{Path: path}
	if len(alias) > 0 && alias[0] != "" {
		e.Alias = alias[0]
	}
	key := path
	if e.Alias != "" {
		key = path + ":" + e.Alias
	}
	c.entries[key] = e
}

func (c *Category) Has(path string) bool {
	for _, e := range c.entries {
		if e.Path == path {
			return true
		}
	}
	return false
}

func (c *Category) Entries() []Entry {
	result := make([]Entry, 0, len(c.entries))
	for _, e := range c.entries {
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Alias < result[j].Alias
	})
	return result
}

func (c *Category) Paths() []string {
	entries := c.Entries()
	paths := make([]string, 0, len(entries))
	seen := make(map[string]bool)
	for _, e := range entries {
		if !seen[e.Path] {
			paths = append(paths, e.Path)
			seen[e.Path] = true
		}
	}
	return paths
}

func (c *Category) Aliases() []string {
	entries := c.Entries()
	aliases := make([]string, 0)
	for _, e := range entries {
		if e.Alias != "" {
			aliases = append(aliases, e.Alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}

func (c *Category) Empty() bool {
	return len(c.entries) == 0
}

type Manager struct {
	categories map[string]*Category
	order      []string
}

func NewManager(categoryOrder ...string) *Manager {
	m := &Manager{
		categories: make(map[string]*Category),
		order:      categoryOrder,
	}
	for _, name := range categoryOrder {
		m.categories[name] = newCategory(name)
	}
	return m
}

func (m *Manager) Add(category, path string, alias ...string) {
	c := m.getOrCreateCategory(category)
	c.Add(path, alias...)
}

func (m *Manager) AddImport(category, path, alias string) {
	m.Add(category, path, alias)
}

func (m *Manager) Has(category, path string) bool {
	c, ok := m.categories[category]
	if !ok {
		return false
	}
	return c.Has(path)
}

func (m *Manager) Get(category string) *Category {
	return m.categories[category]
}

func (m *Manager) Categories() []*Category {
	result := make([]*Category, 0)
	for _, name := range m.order {
		if c := m.categories[name]; c != nil && !c.Empty() {
			result = append(result, c)
		}
	}
	for name, c := range m.categories {
		found := false
		for _, orderedName := range m.order {
			if name == orderedName {
				found = true
				break
			}
		}
		if !found && !c.Empty() {
			result = append(result, c)
		}
	}
	return result
}

func (m *Manager) Paths(category string) []string {
	c := m.categories[category]
	if c == nil {
		return nil
	}
	return c.Paths()
}

func (m *Manager) Aliases(category string) []string {
	c := m.categories[category]
	if c == nil {
		return nil
	}
	return c.Aliases()
}

func (m *Manager) Empty() bool {
	for _, c := range m.categories {
		if !c.Empty() {
			return false
		}
	}
	return true
}

func (m *Manager) getOrCreateCategory(name string) *Category {
	if c, ok := m.categories[name]; ok {
		return c
	}
	c := newCategory(name)
	m.categories[name] = c
	return c
}
