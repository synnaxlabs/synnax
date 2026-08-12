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

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// chainPredecessors resolves the alias split from explicitly managed version chains. A
// chain's current file enumerates the split directly: its alias lines alias, its full
// declarations define. Types whose live declarations carry omit fields always define —
// the frozen counterpart is persisted-only, so the two can never be the same Go type.
func chainPredecessors(
	ctx context.Context, req *plugin.Request,
) (map[string]predecessor, error) {
	preds := make(map[string]predecessor)
	entries, err := chainEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	for origPath, e := range entries {
		if e.version == e.chain.First() {
			continue
		}
		f, err := req.Versions.File(ctx, e.livePath, e.version)
		if err != nil {
			return nil, err
		}
		aliased := make(set.Set[string], len(f.Aliases))
		for name := range f.Aliases {
			aliased.Add(e.resource + "." + name)
		}
		prev, _ := e.chain.Predecessor(e.version)
		preds[versioning.VersionedPath(origPath, e.version)] = predecessor{
			path:    versioning.VersionedPath(origPath, prev),
			aliased: aliased,
		}
	}
	return preds, nil
}

// chainEntry pairs a versioned Go output path with its resource and chain.
type chainEntry struct {
	resource string
	livePath string
	version  int
	chain    versions.Chain
}

// chainEntries maps every chain-covered versioned Go output path to its
// resource, current version, and chain.
func chainEntries(
	ctx context.Context, req *plugin.Request,
) (map[string]chainEntry, error) {
	out := make(map[string]chainEntry)
	if req.Versions == nil {
		return out, nil
	}
	for livePath, chain := range req.Versions.Chains() {
		goPath, ok, err := memberGoPath(ctx, req, livePath, chain)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		out[goPath] = chainEntry{
			resource: chain.Resource,
			livePath: livePath,
			version:  chain.Current(),
			chain:    chain,
		}
	}
	return out, nil
}

// memberGoPath returns the @go output path a chain's current-surface members
// occupy in the live table.
func memberGoPath(
	ctx context.Context,
	req *plugin.Request,
	livePath string,
	chain versions.Chain,
) (string, bool, error) {
	surf, err := req.Versions.Surface(ctx, livePath, chain.Current())
	if err != nil {
		return "", false, err
	}
	for _, t := range req.Resolutions.Types {
		if t.FilePath != livePath+".oracle" {
			continue
		}
		if _, member := surf[t.Name]; !member {
			continue
		}
		if p := output.GetPath(t, "go"); p != "" {
			return p, true, nil
		}
	}
	return "", false, nil
}

// chainFrozenFiles regenerates every frozen version package of every chain: for each
// version k below current, the package at <path>/versions/vk is emitted from the
// chain's surface at k — full declarations for types defined at k, predecessor aliases
// for the rest. Frozen packages are persisted-only: they render exactly what the
// version files record.
func chainFrozenFiles(
	ctx context.Context, req *plugin.Request, fileName string,
) ([]plugin.File, error) {
	entries, err := chainEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	var files []plugin.File
	for _, origPath := range slices.Sorted(maps.Keys(entries)) {
		e := entries[origPath]
		for _, k := range e.chain.Numbers {
			if k >= e.version {
				break
			}
			file, err := frozenFile(ctx, req, origPath, e.livePath, k, fileName)
			if err != nil {
				return nil, err
			}
			if file.Path != "" {
				files = append(files, file)
			}
		}
	}
	return files, nil
}

// ChainPath locates one chain-covered versioned output path.
type ChainPath struct {
	// LivePath is the resource's live import path ("schemas/x/telem").
	LivePath string
	// Current is the chain's current version.
	Current int
	// Numbers holds the chain's declared versions, ascending.
	Numbers []int
}

// ChainPaths maps every chain-covered versioned Go output path to its chain.
func ChainPaths(
	ctx context.Context, req *plugin.Request,
) (map[string]ChainPath, error) {
	entries, err := chainEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(map[string]ChainPath, len(entries))
	for origPath, e := range entries {
		out[origPath] = ChainPath{
			LivePath: e.livePath,
			Current:  e.version,
			Numbers:  e.chain.Numbers,
		}
	}
	return out, nil
}

// ChainFrozenTable builds the resolution table for one frozen version
// package: livePath's surface at k under its DepNS namespace, transitive
// pinned surfaces alongside, and @go output annotations resolving every type
// to its frozen versions/vN directory (or its live root, for hand-written
// types). The returned namespace qualifies the package's own types.
func ChainFrozenTable(
	ctx context.Context,
	req *plugin.Request,
	origPath, livePath string,
	k int,
) (*resolution.Table, string, error) {
	ns := versions.DepNS(livePath, k)
	table := resolution.NewTable()
	if err := req.Versions.SurfaceInto(ctx, table, livePath, k, ns); err != nil {
		return nil, "", err
	}
	if err := annotateOutputs(ctx, req, table, origPath, livePath); err != nil {
		return nil, "", err
	}
	return table, ns, nil
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
	table, ns, err := ChainFrozenTable(ctx, req, origPath, livePath, k)
	if err != nil {
		return plugin.File{}, err
	}
	pred := predecessor{}
	if prev, has := f.Chain.Predecessor(k); has {
		pred = predecessor{
			path:    versioning.VersionedPath(origPath, prev),
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
	// A fully hand-written frozen package (every declaration @go hand) keeps
	// its historical files; there is nothing to generate.
	if len(structs)+len(enums)+len(typedefs)+len(unions) == 0 {
		return plugin.File{}, nil
	}
	content, err := generateGoFile(
		vkPath, structs, enums, typedefs, unions,
		table, req.RepoRoot, pred, nil, PersistedClosure(table), false,
	)
	if err != nil {
		return plugin.File{}, errors.Wrapf(err, "frozen package %s", vkPath)
	}
	return plugin.File{Path: vkPath + "/" + fileName, Content: content}, nil
}

// annotateOutputs assigns a @go output to every namespaced type in a frozen emission
// table: chain surfaces resolve to their versions/vN directory, and hand-written types
// resolve to their resource's live root package.
func annotateOutputs(
	ctx context.Context,
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
			var err error
			if goRoot, ok, err = versionedGoRoot(ctx, req, lp); err != nil {
				return err
			} else if !ok {
				return errors.Newf(
					"%s pins %s, which has no versioned Go output in the live schemas",
					ownLive, lp,
				)
			}
		}
		path := versioning.VersionedPath(goRoot, v)
		// A dependency's hand-written type has one Go home at its live root; the
		// chain's own hand-written types live in the frozen directory itself.
		if omit.IsHand(*t, "go") && lp != ownLive {
			path = goRoot
		}
		withGoOutput(t, path)
	}
	return nil
}

// versionedGoRoot finds the @go output path a live schema file's versions
// directories append to: the path the chain's current-surface members name,
// or — for a chainless dependency — the file's plain output.
func versionedGoRoot(
	ctx context.Context, req *plugin.Request, livePath string,
) (string, bool, error) {
	if chain, ok := req.Versions.Chains()[livePath]; ok {
		return memberGoPath(ctx, req, livePath, chain)
	}
	for _, t := range req.Resolutions.Types {
		if t.FilePath != livePath+".oracle" {
			continue
		}
		if p := output.GetPath(t, "go"); p != "" {
			return p, true, nil
		}
	}
	return "", false, nil
}

// withGoOutput replaces the type's @go output expression, cloning the shared domain
// maps first.
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
