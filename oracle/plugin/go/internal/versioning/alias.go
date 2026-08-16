// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versioning

import (
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// PathAliases records the define-vs-alias split for one version-laid-out path.
type PathAliases struct {
	// PredecessorVersion is the version the aliases point at (current minus one).
	PredecessorVersion int
	// Aliased holds qualified names of types whose shape is transitively
	// unchanged from the predecessor baseline.
	Aliased set.Set[string]
}

// AliasSplit computes, for every version-laid-out path at version M >= 1, the
// set of types that alias the predecessor version package instead of being
// re-defined. The baseline is the latest snapshot declaring M-1 for the path.
// Paths at v0, paths without a resolvable baseline (renamed paths, snapshots
// predating per-resource versioning), or runs without snapshots define every
// type.
func AliasSplit(
	table *resolution.Table,
	latestSnapshot int,
	loadSnapshot func(int) (*resolution.Table, error),
) (map[string]PathAliases, error) {
	if loadSnapshot == nil || latestSnapshot <= 0 {
		return nil, nil
	}
	entries, err := EntryPaths(table)
	if err != nil {
		return nil, err
	}
	result := make(map[string]PathAliases)
	type snap struct {
		table    *resolution.Table
		versions map[string]int
	}
	// Unparseable, pre-versioning, or version-inconsistent snapshots are
	// historical text, not baselines; they cache as empty and are skipped.
	cache := make(map[int]*snap)
	load := func(v int) *snap {
		if s, ok := cache[v]; ok {
			return s
		}
		s := &snap{}
		cache[v] = s
		t, err := loadSnapshot(v)
		if err != nil || t == nil || PreVersioning(t) {
			return s
		}
		pv, err := PathVersions(t)
		if err != nil {
			return s
		}
		s.table, s.versions = t, pv
		return s
	}
	for path, version := range entries {
		if version == 0 {
			continue
		}
		var baseline *resolution.Table
		for v := latestSnapshot; v >= 1; v-- {
			s := load(v)
			if s.table == nil {
				continue
			}
			declared, ok := s.versions[path]
			if !ok {
				continue
			}
			if declared == version-1 {
				baseline = s.table
				break
			}
			// Versions never decrease across snapshots; older snapshots only
			// declare lower versions, so the baseline cannot exist.
			if declared < version-1 {
				break
			}
		}
		if baseline == nil {
			continue
		}
		oldByName := TypesAtPath(baseline, path)
		aliased := make(set.Set[string])
		for _, t := range TypesAtPath(table, path) {
			old, ok := oldByName[t.Name]
			if !ok {
				continue
			}
			if schemadiff.SchemasEqual(old, t, baseline, table) {
				aliased.Add(t.QualifiedName)
			}
		}
		if len(aliased) > 0 {
			result[path] = PathAliases{
				PredecessorVersion: version - 1,
				Aliased:            aliased,
			}
		}
	}
	return result, nil
}

// TypesAtPath returns the declarable types whose @go output is path, keyed by
// bare type name so tables compare across namespace reorganizations (a
// snapshot may predate a schema-directory move).
func TypesAtPath(table *resolution.Table, path string) map[string]resolution.Type {
	result := make(map[string]resolution.Type)
	for _, t := range table.TypesWithDomain("go") {
		if output.GetPath(t, "go") != path {
			continue
		}
		switch t.Form.(type) {
		case resolution.StructForm, resolution.AliasForm, resolution.DistinctForm,
			resolution.EnumForm, resolution.UnionForm:
			result[t.Name] = t
		}
	}
	return result
}
