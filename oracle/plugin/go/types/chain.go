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
	"fmt"
	"maps"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
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
		if e.ended || e.version == e.chain.First() {
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

// chainOrders maps every current version package to its version file's declaration
// order. A frozen package renders in that order, so a current package renders in it too
// and a freeze never moves a declaration.
func chainOrders(
	ctx context.Context, req *plugin.Request,
) (map[string]map[string]int, error) {
	orders := make(map[string]map[string]int)
	entries, err := chainEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	for origPath, e := range entries {
		if e.ended {
			continue
		}
		f, err := req.Versions.File(ctx, e.livePath, e.version)
		if err != nil {
			return nil, err
		}
		order := make(map[string]int, len(f.Order))
		for i, name := range f.Order {
			order[e.resource+"."+name] = i
		}
		orders[versioning.VersionedPath(origPath, e.version)] = order
	}
	return orders, nil
}

// chainEntry pairs a versioned Go output path with its resource and chain.
type chainEntry struct {
	resource string
	livePath string
	version  int
	chain    versions.Chain
	// ended marks a chain whose last version file is a tombstone: every real
	// version is frozen and no current package exists.
	ended bool
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
		ended, err := req.Versions.Ended(ctx, livePath)
		if err != nil {
			return nil, err
		}
		out[goPath] = chainEntry{
			resource: chain.Resource,
			livePath: livePath,
			version:  chain.Current(),
			chain:    chain,
			ended:    ended,
		}
	}
	return out, nil
}

// liveType returns livePath's live declaration of name.
func liveType(
	req *plugin.Request, livePath, name string,
) (resolution.Type, bool) {
	for _, t := range req.Resolutions.Types {
		if t.FilePath == livePath+".oracle" && t.Name == name {
			return t, true
		}
	}
	return resolution.Type{}, false
}

// memberGoPath returns the @go output path a chain's surface members occupy
// in the live table. An ended chain's tombstone surface is empty, so the walk
// falls back through earlier versions: the frozen packages still anchor to
// the live schema's output path.
func memberGoPath(
	ctx context.Context,
	req *plugin.Request,
	livePath string,
	chain versions.Chain,
) (string, bool, error) {
	for _, v := range slices.Backward(chain.Numbers) {
		surf, err := req.Versions.Surface(ctx, livePath, v)
		if err != nil {
			return "", false, err
		}
		if len(surf) == 0 {
			continue
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
	return "", false, nil
}

// chainFrozenFiles regenerates every frozen version package of every chain: for each
// version k below current, the package at <path>/versions/vk is emitted from the
// chain's surface at k — full declarations for types defined at k, predecessor aliases
// for the rest. A frozen package renders what the same version rendered as current, so
// a mint moves nothing in it.
func chainFrozenFiles(
	ctx context.Context,
	req *plugin.Request,
	fileName string,
	closure set.Set[string],
) ([]plugin.File, error) {
	entries, err := chainEntries(ctx, req)
	if err != nil {
		return nil, err
	}
	drops, err := liveDrops(ctx, req, entries)
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
			file, err := frozenFile(
				ctx,
				req,
				origPath,
				e.livePath,
				k,
				fileName,
				closure,
				drops,
			)
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

// liveDrops returns the qualified names of the frozen declarations that a frozen
// package leaves out, as the package did while it was current: members that the live
// file marks @go omit and the current version shares, which no kept frozen declaration
// of any chain uses.
func liveDrops(
	ctx context.Context, req *plugin.Request, entries map[string]chainEntry,
) (set.Set[string], error) {
	type decl struct {
		table *resolution.Table
		typ   resolution.Type
	}
	var decls []decl
	candidates := make(set.Set[string])
	// A member that later versions alias keeps one definition, so keeping it in one
	// package keeps it in every package of the chain that resolves it the same way.
	definition := make(map[string]string)
	sameDefinition := make(map[string][]string)
	for origPath, e := range entries {
		for _, k := range e.chain.Numbers {
			if k >= e.version {
				break
			}
			table, ns, err := ChainFrozenTable(ctx, req, origPath, e.livePath, k)
			if err != nil {
				return nil, err
			}
			shared, err := sharedMembers(ctx, req, e.livePath, e.chain, k)
			if err != nil {
				return nil, err
			}
			surface, err := req.Versions.Surface(ctx, e.livePath, k)
			if err != nil {
				return nil, err
			}
			for _, t := range table.TypesInNamespace(ns) {
				decls = append(decls, decl{table: table, typ: t})
				live, has := liveType(req, e.livePath, t.Name)
				if !has || !omit.IsType(live, "go") || !shared.Contains(t.Name) {
					continue
				}
				candidates.Add(t.QualifiedName)
				version := surface[t.Name].Version
				def := fmt.Sprintf("%s@%d.%s", e.livePath, version, t.Name)
				definition[t.QualifiedName] = def
				sameDefinition[def] = append(sameDefinition[def], t.QualifiedName)
			}
		}
	}
	// A candidate is kept once a kept declaration uses it, until nothing changes.
	kept := make(set.Set[string])
	for changed := true; changed; {
		changed = false
		for _, d := range decls {
			name := d.typ.QualifiedName
			if candidates.Contains(name) && !kept.Contains(name) {
				continue
			}
			for used := range referencedBy(d.table, []resolution.Type{d.typ}) {
				if !candidates.Contains(used) || kept.Contains(used) {
					continue
				}
				changed = true
				kept.Add(used)
				for _, same := range sameDefinition[definition[used]] {
					kept.Add(same)
				}
			}
		}
	}
	drops := make(set.Set[string], len(candidates))
	for name := range candidates {
		if !kept.Contains(name) {
			drops.Add(name)
		}
	}
	return drops, nil
}

// ChainPath locates one chain-covered versioned output path.
type ChainPath struct {
	// LivePath is the resource's live import path ("schemas/x/telem").
	LivePath string
	// Current is the chain's current version. For an ended chain it is the
	// tombstone: every version below it is frozen, and no package generates
	// for it.
	Current int
	// Numbers holds the chain's declared versions, ascending.
	Numbers []int
	// Ended marks a chain whose last version file is a tombstone.
	Ended bool
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
			Ended:    e.ended,
		}
	}
	return out, nil
}

// ChainFrozenTable builds the resolution table for one frozen version package:
// livePath's surface at k under its DepNS namespace, transitive pinned surfaces
// alongside, and @go output annotations resolving every type to its frozen versions/vN
// directory. The returned namespace qualifies the package's own types.
func ChainFrozenTable(
	ctx context.Context,
	req *plugin.Request,
	origPath, livePath string,
	k int,
) (*resolution.Table, string, error) {
	return chainFrozenTable(ctx, req, origPath, livePath, k, true)
}

// chainFrozenTable builds a frozen package's table. With pinned false, dependency types
// resolve to their root packages instead of their versions/vN directories.
func chainFrozenTable(
	ctx context.Context,
	req *plugin.Request,
	origPath, livePath string,
	k int,
	pinned bool,
) (*resolution.Table, string, error) {
	ns := versions.DepNS(livePath, k)
	table := resolution.NewTable()
	if err := req.Versions.SurfaceInto(ctx, table, livePath, k, ns); err != nil {
		return nil, "", err
	}
	err := annotateOutputs(ctx, req, table, origPath, livePath, pinned)
	if err != nil {
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
	liveClosure set.Set[string],
	drops set.Set[string],
) (plugin.File, error) {
	f, err := req.Versions.File(ctx, livePath, k)
	if err != nil {
		return plugin.File{}, err
	}
	table, ns, err := ChainFrozenTable(ctx, req, origPath, livePath, k)
	if err != nil {
		return plugin.File{}, err
	}
	if err = withLiveMethods(ctx, req, table, ns, livePath, f.Chain, k); err != nil {
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
	// The frozen surface table registers sorted, so the version file's declaration
	// order passes to the generator explicitly.
	order := make(map[string]int, len(f.Order))
	for i, name := range f.Order {
		order[ns+"."+name] = i
	}
	for _, name := range f.Order {
		t, ok := table.Get(ns + "." + name)
		if !ok || t.Synthetic || omit.IsSkipped(t, "go") {
			continue
		}
		if drops.Contains(t.QualifiedName) {
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
	// A fully hand-written frozen package (every declaration @go hand) keeps its
	// historical files; there is nothing to generate.
	if len(structs)+len(enums)+len(typedefs)+len(unions) == 0 {
		return plugin.File{}, nil
	}
	latest, _, err := chainFrozenTable(ctx, req, origPath, livePath, k, false)
	if err != nil {
		return plugin.File{}, err
	}
	if err = withLiveMethods(ctx, req, latest, ns, livePath, f.Chain, k); err != nil {
		return plugin.File{}, err
	}
	// Another resource's codec can reach a declaration its own chain does not, so a
	// frozen package splits its pinned references on the closure a current one uses.
	closure := PersistedClosure(table)
	for _, name := range f.Order {
		if liveClosure.Contains(f.Chain.Resource + "." + name) {
			closure.Add(ns + "." + name)
		}
	}
	content, err := generateGoFile(
		vkPath, structs, enums, typedefs, unions,
		table, req.RepoRoot, pred, latest, closure, false, order, nil,
	)
	if err != nil {
		return plugin.File{}, errors.Wrapf(err, "frozen package %s", vkPath)
	}
	return plugin.File{Path: vkPath + "/" + fileName, Content: content}, nil
}

// sharedMembers returns the names that version k and the chain's current version
// resolve to the same definition.
func sharedMembers(
	ctx context.Context,
	req *plugin.Request,
	livePath string,
	chain versions.Chain,
	k int,
) (set.Set[string], error) {
	current, err := req.Versions.Surface(ctx, livePath, chain.Current())
	if err != nil {
		return nil, err
	}
	frozen, err := req.Versions.Surface(ctx, livePath, k)
	if err != nil {
		return nil, err
	}
	shared := make(set.Set[string])
	for name, def := range current {
		if at, member := frozen[name]; member && at.Version == def.Version {
			shared.Add(name)
		}
	}
	return shared, nil
}

// referencedBy returns the qualified names that the given declarations reference. An
// enum's extends copies its members, so the extended enum does not count.
func referencedBy(table *resolution.Table, types []resolution.Type) set.Set[string] {
	used := make(set.Set[string])
	var walk func(ref resolution.TypeRef)
	walk = func(ref resolution.TypeRef) {
		for _, arg := range ref.TypeArgs {
			walk(arg)
		}
		if t, ok := ref.Resolve(table); ok {
			used.Add(t.QualifiedName)
		}
	}
	for _, t := range types {
		switch form := t.Form.(type) {
		case resolution.StructForm:
			for _, ext := range form.Extends {
				walk(ext)
			}
			for _, field := range form.Fields {
				walk(field.Type)
			}
		case resolution.UnionForm:
			if form.BorrowsEnum() {
				walk(form.Enum)
			}
			for _, ext := range form.Extends {
				walk(ext)
			}
			for _, v := range form.Variants {
				walk(v.Type)
			}
		case resolution.AliasForm:
			walk(form.Target)
		case resolution.DistinctForm:
			walk(form.Base)
		}
	}
	return used
}

// withLiveMethods copies the live file's @validate rules, @default groups, and @go
// to_any, fields, and imports onto the version k surface members that the chain's
// current version aliases, and onto the inline variant payloads of those unions. The
// current package's alias is the frozen type, so the live surface uses its methods and
// fields.
func withLiveMethods(
	ctx context.Context,
	req *plugin.Request,
	table *resolution.Table,
	ns, livePath string,
	chain versions.Chain,
	k int,
) error {
	shared, err := sharedMembers(ctx, req, livePath, chain, k)
	if err != nil {
		return err
	}
	aliased := make(set.Set[string], len(shared))
	for name := range shared {
		aliased.Add(ns + "." + name)
	}
	for _, t := range table.Types {
		u, isUnion := t.Form.(resolution.UnionForm)
		if !isUnion || !aliased.Contains(t.QualifiedName) {
			continue
		}
		for _, v := range u.Variants {
			if !v.Inline {
				continue
			}
			if payload, ok := v.Type.Resolve(table); ok {
				aliased.Add(payload.QualifiedName)
			}
		}
	}
	for i := range table.Types {
		t := &table.Types[i]
		if !aliased.Contains(t.QualifiedName) {
			continue
		}
		form, isStruct := t.Form.(resolution.StructForm)
		live, hasLive := liveType(req, livePath, t.Name)
		if !isStruct || !hasLive {
			continue
		}
		liveForm, ok := live.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		fields := slices.Clone(form.Fields)
		for j := range fields {
			lf, _ := lo.Find(liveForm.Fields, func(f resolution.Field) bool {
				return f.Name == fields[j].Name
			})
			for _, name := range []string{"validate", "default"} {
				d, has := lf.Domains[name]
				if !has {
					continue
				}
				domains := make(map[string]resolution.Domain, len(fields[j].Domains)+1)
				maps.Copy(domains, fields[j].Domains)
				domains[name] = d
				fields[j].Domains = domains
			}
		}
		form.Fields = fields
		t.Form = form
		for _, name := range []string{"to_any", "fields", "imports"} {
			if expr, has := live.Domains["go"].Expressions.Find(name); has {
				withGoExpr(t, expr)
			}
		}
	}
	return nil
}

// annotateOutputs assigns a @go output to every namespaced type in a frozen emission
// table, resolving each to its versions/vN directory. A hand-written type lives there
// too, since hand-ness is version-local.
func annotateOutputs(
	ctx context.Context,
	req *plugin.Request,
	table *resolution.Table,
	origPath, ownLive string,
	pinned bool,
) error {
	for i := range table.Types {
		t := &table.Types[i]
		lp, v, ok := versions.ParseDepNS(t.Namespace)
		if !ok {
			continue
		}
		goRoot, dependency := origPath, lp != ownLive
		if dependency {
			var err error
			if goRoot, ok, err = versionedGoRoot(ctx, req, lp); err != nil {
				return err
			} else if !ok {
				return errors.Newf(
					"%s pins %s, which has no versioned Go output in the live schemas",
					ownLive, lp,
				)
			}
			// A dependency chain may span several Go outputs (task and
			// task/config); a member with its own live output resolves there,
			// not at the chain's first output. That package is generated from
			// the live declaration, so its @go name comes along.
			if lt, has := liveType(req, lp, t.Name); has {
				if p := output.GetPath(lt, "go"); p != "" {
					goRoot = p
				}
				if n := domain.GetStringFromType(lt, "go", "name"); n != "" {
					withGoName(t, n)
				}
			}
		}
		if dependency && !pinned {
			withGoOutput(t, goRoot)
			continue
		}
		withGoOutput(t, versioning.VersionedPath(goRoot, v))
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
	withGoExpr(t, goStringExpr("output", path))
}

// withGoName overrides the type's generated Go identifier.
func withGoName(t *resolution.Type, name string) {
	withGoExpr(t, goStringExpr("name", name))
}

func goStringExpr(name, value string) resolution.Expression {
	return resolution.Expression{
		Name: name,
		Values: []resolution.ExpressionValue{
			{Kind: resolution.ValueKindString, StringValue: value},
		},
	}
}

func withGoExpr(t *resolution.Type, expr resolution.Expression) {
	domains := maps.Clone(t.Domains)
	if domains == nil {
		domains = make(map[string]resolution.Domain)
	}
	dom := domains["go"]
	dom.Name = "go"
	exprs := make(resolution.Expressions, 0, len(dom.Expressions)+1)
	for _, e := range dom.Expressions {
		if e.Name != expr.Name {
			exprs = append(exprs, e)
		}
	}
	exprs = append(exprs, expr)
	dom.Expressions = exprs
	domains["go"] = dom
	t.Domains = domains
}
