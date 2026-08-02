// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// chainPredecessors resolves the alias split from explicitly managed version
// chains. A chain's current file enumerates the split directly: its alias
// lines alias, its full declarations define. Types whose live declarations
// carry omit fields always define — the frozen counterpart is persisted-only,
// so the two can never be the same Go type.
func chainPredecessors(
	ctx context.Context, req *plugin.Request,
) (map[string]predecessor, error) {
	preds := make(map[string]predecessor)
	if req.Versions == nil {
		return preds, nil
	}
	entries, err := versioning.EntryPaths(req.Resolutions)
	if err != nil {
		return nil, err
	}
	chains := req.Versions.Chains()
	for origPath, version := range entries {
		if version == 0 {
			continue
		}
		resource, livePath, ok := pathResource(req.Resolutions, origPath)
		if !ok {
			continue
		}
		chain, ok := chains[livePath]
		if !ok {
			continue
		}
		if chain.Current() != version {
			return nil, errors.Newf(
				"%s declares @go version %d but its chain's current file is v%d",
				livePath, version, chain.Current(),
			)
		}
		f, err := req.Versions.File(ctx, livePath, version)
		if err != nil {
			return nil, err
		}
		aliased := make(set.Set[string], len(f.Aliases))
		for name := range f.Aliases {
			live, ok := req.Resolutions.Get(resource + "." + name)
			if ok && hasOmitField(live) {
				continue
			}
			aliased.Add(resource + "." + name)
		}
		preds[versioning.VersionedPath(origPath, version)] = predecessor{
			path:    versioning.VersionedPath(origPath, version-1),
			aliased: aliased,
		}
	}
	return preds, nil
}

// chainFrozenFiles regenerates every frozen version package of every chain:
// for each version k below current, the package at <path>/versions/vk is
// emitted from the chain's surface at k — full declarations for types defined
// at k, predecessor aliases for the rest. Frozen packages are persisted-only:
// they render exactly what the version files record.
func chainFrozenFiles(
	ctx context.Context, req *plugin.Request, fileName string,
) ([]plugin.File, error) {
	if req.Versions == nil {
		return nil, nil
	}
	entries, err := versioning.EntryPaths(req.Resolutions)
	if err != nil {
		return nil, err
	}
	chains := req.Versions.Chains()
	var files []plugin.File
	for _, origPath := range slices.Sorted(maps.Keys(entries)) {
		version := entries[origPath]
		_, livePath, ok := pathResource(req.Resolutions, origPath)
		if !ok {
			continue
		}
		if _, ok := chains[livePath]; !ok {
			continue
		}
		for k := range version {
			file, err := frozenFile(ctx, req, origPath, livePath, k, fileName)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
		}
	}
	return files, nil
}

// frozenFile emits one frozen version package's type file.
func frozenFile(
	ctx context.Context,
	req *plugin.Request,
	origPath, livePath string,
	k int,
	fileName string,
) (plugin.File, error) {
	f, err := req.Versions.File(ctx, livePath, k)
	if err != nil {
		return plugin.File{}, err
	}
	ns := versions.DepNS(livePath, k)
	table := resolution.NewTable()
	if err := req.Versions.SurfaceInto(ctx, table, livePath, k, ns); err != nil {
		return plugin.File{}, err
	}
	if err := annotateOutputs(req, table, origPath, livePath); err != nil {
		return plugin.File{}, err
	}
	pred := predecessor{}
	if k > 0 {
		pred = predecessor{
			path:    versioning.VersionedPath(origPath, k-1),
			aliased: make(set.Set[string], len(f.Aliases)),
		}
		for name := range f.Aliases {
			pred.aliased.Add(ns + "." + name)
		}
	}
	var structs, enums, typedefs, unions []resolution.Type
	for _, name := range f.Order {
		t, ok := table.Get(ns + "." + name)
		if !ok || t.Synthetic || omit.IsSkipped(t, "go") {
			continue
		}
		switch t.Form.(type) {
		case resolution.StructForm:
			structs = append(structs, t)
		case resolution.EnumForm:
			enums = append(enums, t)
		case resolution.UnionForm:
			unions = append(unions, t)
		case resolution.DistinctForm, resolution.AliasForm:
			typedefs = append(typedefs, t)
		}
	}
	vkPath := versioning.VersionedPath(origPath, k)
	content, err := generateGoFile(
		vkPath, structs, enums, typedefs, unions,
		table, req.RepoRoot, pred, nil, PersistedClosure(table), false,
	)
	if err != nil {
		return plugin.File{}, errors.Wrapf(err, "frozen package %s", vkPath)
	}
	return plugin.File{Path: vkPath + "/" + fileName, Content: content}, nil
}

// annotateOutputs assigns a @go output to every namespaced type in a frozen
// emission table: chain surfaces resolve to their versions/vN directory, and
// hand-written types resolve to their resource's live root package.
func annotateOutputs(
	req *plugin.Request,
	table *resolution.Table,
	origPath, ownLive string,
) error {
	for i := range table.Types {
		t := &table.Types[i]
		lp, v, ok := versions.ParseDepNS(t.Namespace)
		if !ok {
			continue
		}
		goRoot := origPath
		if lp != ownLive {
			if goRoot, ok = versionedGoRoot(req.Resolutions, lp); !ok {
				return errors.Newf(
					"%s pins %s, which has no versioned Go output in the live schemas",
					ownLive, lp,
				)
			}
		}
		path := versioning.VersionedPath(goRoot, v)
		if omit.IsHand(*t, "go") {
			path = goRoot
		}
		withGoOutput(t, path)
	}
	return nil
}

// versionedGoRoot finds the versioned @go output path declared by a live
// schema file.
func versionedGoRoot(table *resolution.Table, livePath string) (string, bool) {
	for _, t := range table.Types {
		if t.FilePath != livePath+".oracle" {
			continue
		}
		if _, ok := versioning.Version(t); !ok {
			continue
		}
		if p := output.GetPath(t, "go"); p != "" {
			return p, true
		}
	}
	return "", false
}

// withGoOutput replaces the type's @go output expression, cloning the shared
// domain maps first.
func withGoOutput(t *resolution.Type, path string) {
	domains := maps.Clone(t.Domains)
	if domains == nil {
		domains = make(map[string]resolution.Domain)
	}
	dom := domains["go"]
	dom.Name = "go"
	exprs := make(resolution.Expressions, 0, len(dom.Expressions)+1)
	for _, e := range dom.Expressions {
		if e.Name != "output" {
			exprs = append(exprs, e)
		}
	}
	exprs = append(exprs, resolution.Expression{
		Name: "output",
		Values: []resolution.ExpressionValue{
			{Kind: resolution.ValueKindString, StringValue: path},
		},
	})
	dom.Expressions = exprs
	domains["go"] = dom
	t.Domains = domains
}

// pathResource maps a versioned output path to its schema resource and live
// import path via any version-declaring type's source file.
func pathResource(
	table *resolution.Table, origPath string,
) (resource, livePath string, ok bool) {
	for _, t := range versioning.TypesAtPath(table, origPath) {
		if _, versioned := versioning.Version(t); !versioned {
			continue
		}
		return t.Namespace, strings.TrimSuffix(t.FilePath, ".oracle"), true
	}
	return "", "", false
}

// hasOmitField reports whether the live declaration carries any non-persisted
// field.
func hasOmitField(t resolution.Type) bool {
	form, ok := t.Form.(resolution.StructForm)
	if !ok {
		return false
	}
	return len(schemadiff.PersistedFields(form.Fields)) != len(form.Fields)
}
