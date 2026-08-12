// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package versioning resolves per-resource schema versions (@go version N) and rewrites
//
//	@go	output paths so that versioned packages emit into versions/vN/ sub-packages.
//
// Version-laid-out packages (those containing a gorp entry) emit their current version
// into versions/vN and re-export it from the package root; value-type packages (no gorp
// entry) keep their current code at the root and gain versions/vN packages only for
// frozen historical shapes.
package versioning

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Version returns the declared @go version for a type, or false when the type's file
// declares none.
func Version(t resolution.Type) (int, bool) {
	dom, ok := t.Domains["go"]
	if !ok {
		return 0, false
	}
	expr, ok := dom.Expressions.Find("version")
	if !ok || len(expr.Values) == 0 {
		return 0, false
	}
	return int(expr.Values[0].IntValue), true
}

// Dir returns the version sub-directory name for version n ("v3").
func Dir(n int) string { return fmt.Sprintf("v%d", n) }

// VersionedPath returns the versions/vN sub-path of goPath for version n.
func VersionedPath(goPath string, n int) string {
	return goPath + "/versions/" + Dir(n)
}

// VersionDirs returns the numeric version sub-directories present under goPath's
// versions/ tree on disk, ascending.
func VersionDirs(repoRoot, goPath string) ([]int, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, goPath, "versions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var versions []int
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "v") {
			continue
		}
		k, err := strconv.Atoi(e.Name()[1:])
		if err != nil {
			continue
		}
		versions = append(versions, k)
	}
	slices.Sort(versions)
	return versions, nil
}

// EntryPaths maps every version-laid-out @go output path in the table to its declared
// version. These packages emit their current version into versions/vN. It errors when
// two types at the same path disagree on the version, when a declared version is
// negative, or when a @go migrate entry lacks a version.
func EntryPaths(table *resolution.Table) (map[string]int, error) {
	versions := make(map[string]int)
	declared := make(map[string]string)
	for _, t := range table.TypesWithDomain("go") {
		goPath := output.GetPath(t, "go")
		if goPath == "" {
			continue
		}
		v, ok := Version(t)
		if !ok {
			continue
		}
		if v < 0 {
			return nil, errors.Newf(
				"%s: @go version must be a non-negative integer, got %d",
				t.QualifiedName, v,
			)
		}
		if prev, ok := versions[goPath]; ok && prev != v {
			return nil, errors.Newf(
				"conflicting @go version declarations for %s: %s declares %d, %s declares %d",
				goPath,
				declared[goPath],
				prev,
				t.QualifiedName,
				v,
			)
		}
		versions[goPath] = v
		declared[goPath] = t.QualifiedName
	}
	return versions, nil
}

// currentPathMap maps each version-laid-out path to its current versions/vN sub-path.
func currentPathMap(table *resolution.Table) (map[string]string, error) {
	entries, err := EntryPaths(table)
	if err != nil {
		return nil, err
	}
	pathMap := make(map[string]string, len(entries))
	for goPath, v := range entries {
		pathMap[goPath] = VersionedPath(goPath, v)
	}
	return pathMap, nil
}

// RewriteCurrent returns a table with every version-laid-out package's @go output
// rewritten to its current versions/vN sub-path, plus the applied path map keyed by
// original path.
func RewriteCurrent(
	table *resolution.Table,
) (*resolution.Table, map[string]string, error) {
	pathMap, err := currentPathMap(table)
	if err != nil {
		return nil, nil, err
	}
	if len(pathMap) == 0 {
		return table, pathMap, nil
	}
	return RewriteOutputPaths(table, pathMap), pathMap, nil
}

// RewriteOutputPaths clones table, replacing the @go output value of every
// version-declaring type whose path appears in pathMap. Types at unmapped paths are
// unchanged. Types that do not declare @go version also stay put: they are transient
// (never persisted), living at the package root rather than the version layout even
// when siblings at their path are versioned.
func RewriteOutputPaths(
	table *resolution.Table,
	pathMap map[string]string,
) *resolution.Table {
	clone := &resolution.Table{
		Imports:    table.Imports,
		Namespaces: table.Namespaces,
		Types:      make([]resolution.Type, 0, len(table.Types)),
	}
	for _, typ := range table.Types {
		goPath := output.GetPath(typ, "go")
		mirroredPath, needsRewrite := pathMap[goPath]
		// Omitted types carry no generated declaration; their hand-written homes follow
		// the version layout (per-version alias files), so their references keep
		// tracking the version directory.
		if _, versioned := Version(typ); !versioned && !omit.IsHand(typ, "go") {
			needsRewrite = false
		}
		if !needsRewrite {
			clone.Types = append(clone.Types, typ)
			continue
		}
		newDomains := make(map[string]resolution.Domain, len(typ.Domains))
		for k, v := range typ.Domains {
			if k == "go" {
				newExprs := make(resolution.Expressions, len(v.Expressions))
				for i, expr := range v.Expressions {
					if expr.Name == "output" && len(expr.Values) > 0 {
						newVals := make([]resolution.ExpressionValue, len(expr.Values))
						copy(newVals, expr.Values)
						newVals[0] = resolution.ExpressionValue{
							StringValue: mirroredPath,
						}
						newExprs[i] = resolution.Expression{
							AST:    expr.AST,
							Name:   expr.Name,
							Values: newVals,
						}
					} else {
						newExprs[i] = expr
					}
				}
				newDomains[k] = resolution.Domain{
					AST:         v.AST,
					Name:        v.Name,
					Expressions: newExprs,
				}
			} else {
				newDomains[k] = v
			}
		}
		rewritten := typ
		rewritten.Domains = newDomains
		clone.Types = append(clone.Types, rewritten)
	}
	return clone
}

// TypesAtPath returns the declarable types whose @go output is path, keyed by bare type
// name so tables compare across namespace reorganizations (a snapshot may predate a
// schema-directory move).
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
