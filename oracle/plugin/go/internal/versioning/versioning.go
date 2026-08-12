// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package versioning resolves per-resource schema versions from the version chains and
// rewrites
//
//	@go	output paths so that versioned packages emit into versions/vN/ sub-packages.
//
// Version-laid-out packages (those containing a gorp entry) emit their current version
// into versions/vN and re-export it from the package root; value-type packages (no gorp
// entry) keep their current code at the root and gain versions/vN packages only for
// frozen historical shapes.
package versioning

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/set"
)

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

// Survey reads the version chains: entries maps every chain-covered @go
// output path to its chain's current version, and members holds the qualified
// name of every current-surface member. Both are empty without a resolver.
func Survey(
	ctx context.Context, table *resolution.Table, res *versions.Resolver,
) (entries map[string]int, members set.Set[string], err error) {
	entries, members = make(map[string]int), make(set.Set[string])
	if res == nil {
		return entries, members, nil
	}
	for livePath, chain := range res.Chains() {
		surf, err := res.Surface(ctx, livePath, chain.Current())
		if err != nil {
			return nil, nil, err
		}
		for _, t := range table.Types {
			if t.FilePath != livePath+".oracle" {
				continue
			}
			if _, member := surf[t.Name]; !member {
				continue
			}
			members.Add(t.QualifiedName)
			if goPath := output.GetPath(t, "go"); goPath != "" {
				entries[goPath] = chain.Current()
			}
		}
	}
	return entries, members, nil
}

// EntryPaths maps every chain-covered versioned @go output path to its
// chain's current version. These packages emit their current version into
// versions/vN.
func EntryPaths(
	ctx context.Context, table *resolution.Table, res *versions.Resolver,
) (map[string]int, error) {
	entries, _, err := Survey(ctx, table, res)
	return entries, err
}

// RewriteCurrent returns a table with every version-laid-out package's @go output
// rewritten to its current versions/vN sub-path, plus the applied path map keyed
// by original path and the current-surface member set.
func RewriteCurrent(
	ctx context.Context, table *resolution.Table, res *versions.Resolver,
) (*resolution.Table, map[string]string, set.Set[string], error) {
	entries, members, err := Survey(ctx, table, res)
	if err != nil {
		return nil, nil, nil, err
	}
	pathMap := make(map[string]string, len(entries))
	for goPath, v := range entries {
		pathMap[goPath] = VersionedPath(goPath, v)
	}
	if len(pathMap) == 0 {
		return table, pathMap, members, nil
	}
	return RewriteOutputPaths(table, pathMap, members), pathMap, members, nil
}

// RewriteOutputPaths clones table, replacing the @go output value of every
// version-declaring type whose path appears in pathMap. Types at unmapped paths are
// unchanged. Non-member types also stay put: they are transient
// (never persisted), living at the package root rather than the version layout even
// when siblings at their path are versioned.
func RewriteOutputPaths(
	table *resolution.Table,
	pathMap map[string]string,
	members set.Set[string],
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
		if !members.Contains(typ.QualifiedName) && !omit.IsHand(typ, "go") {
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
