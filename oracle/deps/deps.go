// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deps

import (
	"sort"

	"github.com/synnaxlabs/oracle/resolution"
)

// Graph holds forward and reverse dependency edges between types.
type Graph struct {
	forward map[string]map[string]bool
	reverse map[string]map[string]bool
	table   *resolution.Table
}

// Build constructs a dependency graph from all types in the resolution table.
func Build(table *resolution.Table) *Graph {
	g := &Graph{
		forward: make(map[string]map[string]bool),
		reverse: make(map[string]map[string]bool),
		table:   table,
	}
	for _, typ := range table.Types {
		deps := g.collectDependencies(typ)
		if len(deps) > 0 {
			if g.forward[typ.QualifiedName] == nil {
				g.forward[typ.QualifiedName] = make(map[string]bool)
			}
			for _, dep := range deps {
				g.forward[typ.QualifiedName][dep] = true
				if g.reverse[dep] == nil {
					g.reverse[dep] = make(map[string]bool)
				}
				g.reverse[dep][typ.QualifiedName] = true
			}
		}
	}
	return g
}

// AffectedEntries returns the qualified names of gorp entry types (HasKeyDomain)
// that transitively depend on any of the changedTypes. Results are sorted for
// deterministic output.
func (g *Graph) AffectedEntries(changedTypes []string) []string {
	visited := make(map[string]bool)
	queue := make([]string, len(changedTypes))
	copy(queue, changedTypes)
	for _, ct := range changedTypes {
		visited[ct] = true
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for dependent := range g.reverse[current] {
			if !visited[dependent] {
				visited[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}

	var entries []string
	for qname := range visited {
		typ, ok := g.table.Get(qname)
		if !ok {
			continue
		}
		sf, ok := typ.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		if sf.HasKeyDomain {
			entries = append(entries, qname)
		}
	}
	sort.Strings(entries)
	return entries
}

func (g *Graph) collectDependencies(typ resolution.Type) []string {
	var deps []string
	seen := make(map[string]bool)

	var addDep func(ref resolution.TypeRef)
	addDep = func(ref resolution.TypeRef) {
		if ref.Name == "" || ref.IsTypeParam() {
			return
		}
		if resolution.IsPrimitive(ref.Name) || ref.Name == "Array" || ref.Name == "Map" {
			for _, arg := range ref.TypeArgs {
				addDep(arg)
			}
			return
		}
		resolved, ok := g.table.Get(ref.Name)
		if !ok {
			resolved, ok = g.table.Lookup(typ.Namespace, ref.Name)
		}
		if ok {
			if !seen[resolved.QualifiedName] {
				seen[resolved.QualifiedName] = true
				deps = append(deps, resolved.QualifiedName)
			}
		}
		for _, arg := range ref.TypeArgs {
			addDep(arg)
		}
	}

	switch form := typ.Form.(type) {
	case resolution.StructForm:
		for _, extendsRef := range form.Extends {
			addDep(extendsRef)
		}
		for _, field := range form.Fields {
			addDep(field.Type)
		}
	case resolution.AliasForm:
		addDep(form.Target)
	case resolution.DistinctForm:
		addDep(form.Base)
	}
	return deps
}
