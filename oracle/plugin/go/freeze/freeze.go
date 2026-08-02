// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package freeze derives canonical version-file content from the live
// schemas: the exact bytes `oracle migrate` writes at mint or amend time, and
// the bytes the drift gate requires the current version file to match.
package freeze

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// keptGoExpressions are the @go expressions a version file records: the
// persistence declarations. Everything else (outputs, language bindings,
// validation, indexing) is live-surface concern.
var keptGoExpressions = set.New("marshal", "hand", "migrate", "pinned")

// Input carries everything canonical emission derives from.
type Input struct {
	// Live is the live schema analysis.
	Live *resolution.Table
	// Resolver resolves the discovered version chains.
	Resolver *versions.Resolver
	// Chain is the resource being frozen.
	Chain versions.Chain
	// N is the version the emission declares: the current version for amend
	// and the drift gate, current+1 for a mint.
	N int
	// Pins fixes dependency pin versions by live path. Referenced
	// dependencies absent from Pins pin their chain's current version (the
	// mint behavior); the drift gate passes the existing file's pins so a
	// stale pin is never silently healed.
	Pins map[string]int
	// Docs carries the existing file's explicit @doc overrides by type name.
	// Names absent from Docs inherit their predecessor doc; names present
	// re-declare it.
	Docs map[string]string
	// Pinned carries the @go pinned type names the file enumerates despite
	// being unpersisted.
	Pinned set.Set[string]
}

// Canonical renders the canonical version file for in.Chain at in.N,
// formatted to canonical bytes.
func Canonical(ctx context.Context, in Input) (string, error) {
	livePath := in.Chain.LivePath()
	closure := gotypes.PersistedClosure(in.Live)
	var members []resolution.Type
	for _, t := range in.Live.TypesInNamespace(in.Chain.Resource) {
		if t.Synthetic || t.FilePath != livePath+".oracle" {
			continue
		}
		if closure.Contains(t.QualifiedName) || in.Pinned.Contains(t.Name) {
			members = append(members, t)
		}
	}
	if len(members) == 0 {
		return "", errors.Newf(
			"%s has no persisted types to freeze", livePath,
		)
	}
	var (
		surf map[string]versions.Definition
		err  error
	)
	if in.N > 0 {
		if surf, err = in.Resolver.Surface(ctx, livePath, in.N-1); err != nil {
			return "", err
		}
	}
	deps := make(set.Set[string])
	decls := make([]versions.Decl, 0, len(members))
	for _, t := range members {
		if def, ok := surf[t.Name]; ok {
			definer, err := in.Resolver.File(ctx, livePath, def.Version)
			if err != nil {
				return "", err
			}
			if schemadiff.SchemasEqual(def.Type, t, definer.Table, in.Live) {
				decls = append(decls, versions.Decl{
					Type:  resolution.Type{Name: t.Name},
					Alias: &versions.Alias{Version: def.Version, Name: t.Name},
				})
				continue
			}
		}
		ft := frozenDecl(t, in, surf)
		collectDepRefs(ft, in.Chain.Resource, deps)
		decls = append(decls, versions.Decl{Type: ft})
	}
	imports, err := depImports(in, deps)
	if err != nil {
		return "", err
	}
	rendered := versions.Render(decls, versions.RenderOptions{
		Imports:        imports,
		KeepExpression: keepExpression,
		Qualifier: func(ns string) string {
			if ns == in.Chain.Resource {
				return ""
			}
			return ns
		},
	})
	return formatter.Format(rendered)
}

// keepExpression filters domains down to what version files record.
func keepExpression(domain string, expr resolution.Expression) bool {
	switch domain {
	case "key", "doc":
		return true
	case "go":
		return keptGoExpressions.Contains(expr.Name)
	default:
		return false
	}
}

// frozenDecl prepares one live declaration for version-file rendering:
// non-persisted fields stripped, docs resolved per the inheritance rules, and
// the pinned marker injected where the file enumerates it.
func frozenDecl(
	t resolution.Type,
	in Input,
	surf map[string]versions.Definition,
) resolution.Type {
	if sf, ok := t.Form.(resolution.StructForm); ok {
		sf.Fields = schemadiff.PersistedFields(sf.Fields)
		t.Form = sf
	}
	domains := maps.Clone(t.Domains)
	if domains == nil {
		domains = make(map[string]resolution.Domain)
	}
	_, existed := surf[t.Name]
	if doc, override := in.Docs[t.Name]; override {
		domains["doc"] = docDomain(doc)
	} else if existed {
		// A redeclared name inherits its predecessor doc; the live doc
		// remains the current surface's and is not retroactively frozen.
		delete(domains, "doc")
	}
	if in.Pinned.Contains(t.Name) {
		dom := domains["go"]
		dom.Name = "go"
		if _, has := dom.Expressions.Find("pinned"); !has {
			dom.Expressions = append(
				slices.Clone(dom.Expressions),
				resolution.Expression{Name: "pinned"},
			)
		}
		domains["go"] = dom
	}
	t.Domains = domains
	return t
}

// docDomain synthesizes a @doc value domain.
func docDomain(doc string) resolution.Domain {
	return resolution.Domain{
		Name: "doc",
		Expressions: resolution.Expressions{{
			Name: "value",
			Values: []resolution.ExpressionValue{{
				Kind:        resolution.ValueKindString,
				StringValue: doc,
			}},
		}},
	}
}

// collectDepRefs records the foreign resource namespaces a declaration's
// persisted references name.
func collectDepRefs(t resolution.Type, resource string, deps set.Set[string]) {
	var walkRef func(ref resolution.TypeRef)
	walkRef = func(ref resolution.TypeRef) {
		for _, arg := range ref.TypeArgs {
			walkRef(arg)
		}
		ns, _, found := strings.Cut(ref.Name, ".")
		if found && ns != resource {
			deps.Add(ns)
		}
	}
	switch form := t.Form.(type) {
	case resolution.StructForm:
		for _, f := range form.Fields {
			walkRef(f.Type)
		}
		for _, a := range form.Actions {
			for _, f := range a.Fields {
				walkRef(f.Type)
			}
			for _, ext := range a.Extends {
				walkRef(ext)
			}
		}
		for _, ext := range form.Extends {
			walkRef(ext)
		}
		for _, tp := range form.TypeParams {
			if tp.Constraint != nil {
				walkRef(*tp.Constraint)
			}
			if tp.Default != nil {
				walkRef(*tp.Default)
			}
		}
	case resolution.EnumForm:
		for _, ext := range form.Extends {
			walkRef(ext)
		}
	case resolution.UnionForm:
		for _, ext := range form.Extends {
			walkRef(ext)
		}
		for _, v := range form.Variants {
			walkRef(v.Type)
		}
	case resolution.DistinctForm:
		walkRef(form.Base)
	case resolution.AliasForm:
		walkRef(form.Target)
	}
}

// depImports maps referenced dependency resources to version-file import
// paths, pinning per Input.Pins with the dependency chain's current version
// as the mint default.
func depImports(in Input, deps set.Set[string]) ([]string, error) {
	chains := in.Resolver.Chains()
	imports := make([]string, 0, len(deps))
	for _, ns := range deps.Slice() {
		livePath, ok := depLivePath(in.Live, ns)
		if !ok {
			return nil, errors.Newf(
				"cannot resolve the live schema for dependency %q", ns,
			)
		}
		chain, ok := chains[livePath]
		if !ok {
			return nil, errors.Newf(
				"%s references %s, which has no version chain; freeze it first",
				in.Chain.LivePath(), livePath,
			)
		}
		pin, ok := in.Pins[livePath]
		if !ok {
			pin = chain.Current()
		}
		imports = append(imports, chain.FilePath(pin))
	}
	slices.Sort(imports)
	return imports, nil
}

// depLivePath resolves a referenced namespace to its live schema import path.
func depLivePath(table *resolution.Table, ns string) (string, bool) {
	for _, t := range table.TypesInNamespace(ns) {
		if t.FilePath != "" {
			return strings.TrimSuffix(t.FilePath, ".oracle"), true
		}
	}
	return "", false
}
