// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin/domain"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/set"
)

// VersionsGate verifies the explicitly managed version chains: the live file's
// version-owned content matches chain resolution (the merged projection), and every
// redeclaration differs structurally from its resolved predecessor.
type VersionsGate struct{}

// Name implements Checker.
func (VersionsGate) Name() string { return "versions" }

// Run implements Checker.
func (g VersionsGate) Run(
	ctx context.Context, p *pipeline.Result, env Env,
) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	defer func() { r.Elapsed = time.Since(start) }()
	if p.Resolutions == nil {
		return r
	}
	chains, resolver := p.Chains, p.Versions
	if len(chains) == 0 || resolver == nil {
		return r
	}
	for _, livePath := range slices.Sorted(maps.Keys(chains)) {
		g.checkChain(ctx, &r, p, env, resolver, chains[livePath])
	}
	return r
}

// checkChain runs the drift and minimality checks for one chain.
func (g VersionsGate) checkChain(
	ctx context.Context,
	r *GateReport,
	p *pipeline.Result,
	env Env,
	resolver *versions.Resolver,
	chain versions.Chain,
) {
	livePath := chain.LivePath()
	current := chain.Current()

	// Consistency: the live file's version-owned content must match chain resolution —
	// the merged projection is the canonical live file. The version files are the
	// authority; a live edit to version-owned content is overwritten by sync, and a
	// chain edit lands in the live file the same way.
	if merged, ok := p.MergedSources[livePath+".oracle"]; ok {
		filePath := livePath + ".oracle"
		onDisk, err := os.ReadFile(filepath.Join(env.RepoRoot, filePath))
		if err != nil && !os.IsNotExist(err) {
			r.fail(Finding{
				Path: filePath, Severity: SeverityError, Message: err.Error(),
			})
		} else if string(onDisk) != string(merged) {
			finding := Finding{
				Path:     filePath,
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"%s drifts from its version files", livePath,
				),
				FixHint: fmt.Sprintf(
					"the version files under %s are the authority for "+
						"version-owned content; run `oracle sync` to project them",
					chain.Dir(),
				),
			}
			if env.IncludeDiffs {
				finding.Diff = unifiedDiff(
					filePath, string(merged), string(onDisk), 200,
				)
			}
			r.fail(finding)
		}
	}

	// Import placement: stored references pin, resolved references float.
	for _, k := range chain.Numbers {
		fk, err := resolver.File(ctx, livePath, k)
		if err != nil {
			r.fail(Finding{
				Path:     chain.FilePath(k) + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
			return
		}
		g.checkImportPlacement(r, resolver, chain, fk)
	}

	// Pin currency: every pin the current surface's stored reference graph crosses must
	// target its dependency chain's current version.
	g.checkPinCurrency(ctx, r, resolver, chain)

	// Minimality: every redeclaration must differ structurally from its resolved
	// predecessor.
	for i, k := range chain.Numbers {
		if i == 0 || k > current {
			continue
		}
		fk, err := resolver.File(ctx, livePath, k)
		if err != nil {
			r.fail(Finding{
				Path:     chain.FilePath(k) + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
			return
		}
		surf, err := resolver.Surface(ctx, livePath, chain.Numbers[i-1])
		if err != nil {
			r.fail(Finding{
				Path:     chain.FilePath(k) + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
			return
		}
		fkTransient := fk.Transient()
		for _, t := range fk.Defined {
			if t.Synthetic {
				continue
			}
			// Omit-transient declarations track the live schema and may legitimately
			// match their predecessor.
			if fkTransient.Contains(t.Name) {
				continue
			}
			def, ok := surf[t.Name]
			if !ok {
				continue
			}
			definer, err := resolver.File(ctx, livePath, def.Version)
			if err != nil {
				r.fail(Finding{
					Path:     chain.FilePath(k) + ".oracle",
					Severity: SeverityError,
					Message:  err.Error(),
				})
				return
			}
			if gotypes.StructurallyEqual(def.Type, t, definer.Table, fk.Table) {
				r.fail(Finding{
					Path:     chain.FilePath(k) + ".oracle",
					Severity: SeverityError,
					Message: fmt.Sprintf(
						"%s v%d redeclares %s identically to v%d",
						livePath, k, t.Name, def.Version,
					),
					FixHint: fmt.Sprintf(
						"use an alias: %s = v%d.%s",
						t.Name, def.Version, t.Name,
					),
				})
			}
		}
	}
}

// fail records an error finding and flips the gate status.
func (r *GateReport) fail(f Finding) {
	r.Findings = append(r.Findings, f)
	r.Status = StatusFail
}

// checkImportPlacement verifies one version file's imports against the persistence
// boundary: a dependency reached by any stored reference must be pinned; a dependency
// reached only by resolved references must float on its live schema.
func (g VersionsGate) checkImportPlacement(
	r *GateReport,
	resolver *versions.Resolver,
	chain versions.Chain,
	f *versions.File,
) {
	stored, referenced := classifyDeps(f)
	path := chain.FilePath(f.N) + ".oracle"
	live := make(map[string]string, len(f.Live))
	for i, ns := range f.Live {
		live[ns] = f.LivePaths[i]
	}
	pinned := make(map[string]string, len(f.Pins))
	for depLive := range f.Pins {
		pinned[filepath.Base(depLive)] = depLive
	}
	for _, ns := range slices.Sorted(maps.Keys(stored)) {
		depLive, isLive := live[ns]
		if !isLive {
			continue
		}
		// A chainless dependency has no version file to pin: its live schema is the one
		// shape, and the stored reference rides it (arc v0's Program). The exposure is
		// the dependency's, not this file's.
		if _, versioned := resolver.Chains()[depLive]; !versioned {
			continue
		}
		r.fail(Finding{
			Path:     path,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"%s v%d stores %s but imports its live schema",
				chain.LivePath(), f.N, ns,
			),
			FixHint: fmt.Sprintf(
				"stored references resolve against the dependency's shape at "+
					"a fixed version; import a %s/%s version file",
				depLive, "versions",
			),
		})
	}
	for _, ns := range slices.Sorted(maps.Keys(referenced)) {
		if stored.Contains(ns) {
			continue
		}
		depLive, isPinned := pinned[ns]
		if !isPinned {
			continue
		}
		r.fail(Finding{
			Path:     path,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"%s v%d only resolves %s at read time but pins it",
				chain.LivePath(), f.N, ns,
			),
			FixHint: fmt.Sprintf(
				"resolved references always materialize the current shape; "+
					"import the live schema %q", depLive,
			),
		})
	}
}

// classifyDeps splits a version file's foreign dependency namespaces along the
// persistence boundary: stored holds namespaces reachable from a marshal root through
// persisted references; referenced holds every namespace the file's declarations name.
func classifyDeps(f *versions.File) (stored, referenced set.Set[string]) {
	stored, referenced = make(set.Set[string]), make(set.Set[string])
	local := make(map[string]resolution.Type, len(f.Defined))
	for _, t := range f.Defined {
		local[t.Name] = t
	}
	foreign := func(ref resolution.TypeRef, into set.Set[string]) []string {
		var locals []string
		var walk func(ref resolution.TypeRef)
		walk = func(ref resolution.TypeRef) {
			if ref.TypeParam == nil {
				if ns, name, found := strings.Cut(ref.Name, "."); found {
					if versionNSRe.MatchString(ns) || ns == f.Chain.Resource {
						locals = append(locals, name)
					} else {
						into.Add(ns)
					}
				} else if _, isLocal := local[ref.Name]; isLocal {
					locals = append(locals, ref.Name)
				}
			}
			for _, a := range ref.TypeArgs {
				walk(a)
			}
		}
		walk(ref)
		return locals
	}
	var walkValue func(v resolution.ExpressionValue, into set.Set[string])
	walkValue = func(v resolution.ExpressionValue, into set.Set[string]) {
		if v.Kind == resolution.ValueKindIdent {
			if ns, _, found := strings.Cut(v.IdentValue, "."); found &&
				!versionNSRe.MatchString(ns) && ns != f.Chain.Resource {
				into.Add(ns)
			}
		}
		for _, e := range v.Elements {
			walkValue(e, into)
		}
		for _, sf := range v.Fields {
			walkValue(sf.Value, into)
		}
	}
	// walkDecl visits one declaration's references. Omit fields are resolved contexts:
	// their targets never enter the stored set, and local types reached only through
	// them stay outside the stored closure.
	walkDecl := func(t resolution.Type, into set.Set[string], skipOmit bool) []string {
		var locals []string
		visit := func(ref resolution.TypeRef) {
			locals = append(locals, foreign(ref, into)...)
		}
		switch form := t.Form.(type) {
		case resolution.StructForm:
			for _, fd := range form.Fields {
				if skipOmit &&
					domain.GetStringFromField(fd, "go", "marshal") == "omit" {
					continue
				}
				visit(fd.Type)
				if fd.Default != nil {
					walkValue(*fd.Default, into)
				}
			}
			for _, a := range form.Actions {
				for _, fd := range a.Fields {
					visit(fd.Type)
				}
				for _, e := range a.Extends {
					visit(e)
				}
			}
			for _, e := range form.Extends {
				visit(e)
			}
			for _, p := range form.TypeParams {
				if p.Constraint != nil {
					visit(*p.Constraint)
				}
				if p.Default != nil {
					visit(*p.Default)
				}
			}
		case resolution.EnumForm:
			for _, e := range form.Extends {
				visit(e)
			}
		case resolution.UnionForm:
			for _, v := range form.Variants {
				visit(v.Type)
			}
			for _, e := range form.Extends {
				visit(e)
			}
		case resolution.DistinctForm:
			visit(form.Base)
		case resolution.AliasForm:
			visit(form.Target)
		}
		return locals
	}

	for _, t := range f.Defined {
		if t.Synthetic {
			continue
		}
		walkDecl(t, referenced, false)
	}
	visited := make(set.Set[string])
	var visitStored func(name string)
	visitStored = func(name string) {
		if visited.Contains(name) {
			return
		}
		visited.Add(name)
		t, ok := local[name]
		if !ok {
			return
		}
		for _, l := range walkDecl(t, stored, true) {
			visitStored(l)
		}
	}
	// Membership marks a type persisted, so every declaration is a stored root except
	// the omit-transient ones — a transient type tracks the live schema and resolves
	// its references there.
	transient := f.Transient()
	for _, t := range f.Defined {
		if t.Synthetic || transient.Contains(t.Name) {
			continue
		}
		visitStored(t.Name)
	}
	return stored, referenced
}

// versionNSRe matches a chain-sibling namespace qualifier ("v0", "v12").
var versionNSRe = regexp.MustCompile(`^v(0|[1-9][0-9]*)$`)

// checkPinCurrency walks the stored reference graph of the chain's current surface and
// fails on any pin that lags its dependency chain's current version: the current
// package embeds the dependency's current type, so a stale pin is a contradiction. A
// lagging pin survives when every member referenced through it resolves to the same
// defining version at both the pin and the dependency's current version — the two
// shapes are one declaration, so the lag is nominal.
func (g VersionsGate) checkPinCurrency(
	ctx context.Context,
	r *GateReport,
	resolver *versions.Resolver,
	chain versions.Chain,
) {
	chains := resolver.Chains()
	livePath := chain.LivePath()
	surf, err := resolver.Surface(ctx, livePath, chain.Current())
	if err != nil {
		r.fail(Finding{
			Path:     chain.FilePath(chain.Current()) + ".oracle",
			Severity: SeverityError,
			Message:  err.Error(),
		})
		return
	}
	visited := make(set.Set[string])
	var visit func(livePath string, n int, name string)
	visit = func(livePath string, n int, name string) {
		key := livePath + "@" + strconv.Itoa(n) + "." + name
		if visited.Contains(key) {
			return
		}
		visited.Add(key)
		f, err := resolver.File(ctx, livePath, n)
		if err != nil {
			return
		}
		t, ok := definedType(f, name)
		if !ok {
			return
		}
		stored := make(set.Set[string])
		locals := storedRefs(f, t, stored)
		for _, l := range locals {
			depNS, depName, defined := resolver.Definer(ctx, livePath, n, l)
			if !defined {
				continue
			}
			if depLive, dv, ok := versions.ParseDepNS(depNS); ok {
				visit(depLive, dv, depName)
			}
		}
		for _, ns := range slices.Sorted(maps.Keys(stored)) {
			for depLive, pv := range f.Pins {
				if filepath.Base(depLive) != ns {
					continue
				}
				depChain, ok := chains[depLive]
				if !ok {
					continue
				}
				if pv != depChain.Current() &&
					!pinAliasCurrent(
						ctx, resolver, depLive, pv, depChain.Current(),
						membersOf(f, ns),
					) {
					r.fail(Finding{
						Path:     chain.FilePath(chain.Current()) + ".oracle",
						Severity: SeverityError,
						Message: fmt.Sprintf(
							"%s's current surface resolves %s at v%d, but "+
								"%s's current version is v%d",
							chain.LivePath(), ns, pv, depLive, depChain.Current(),
						),
						FixHint: fmt.Sprintf(
							"redeclare the affected types with a fresh pin: "+
								"`oracle migrate %s` to bump, or amend v%d in "+
								"place if it has not shipped",
							chain.Resource, chain.Current(),
						),
					})
				}
				for _, member := range membersOf(f, ns) {
					visit(depLive, pv, member)
				}
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(surf)) {
		visit(livePath, surf[name].Version, name)
	}
}

// storedRefs collects one declaration's stored foreign namespaces into stored and
// returns the local names its persisted references reach.
func storedRefs(
	f *versions.File, t resolution.Type, stored set.Set[string],
) []string {
	sub := &versions.File{Chain: f.Chain, N: f.N, Defined: []resolution.Type{t}}
	s, _ := classifyDeps(sub)
	for ns := range s {
		stored.Add(ns)
	}
	// Local names come from a targeted walk: classifyDeps scopes its closure to the one
	// declaration, so its visited locals are exactly the decl's same-chain references.
	var locals []string
	switch form := t.Form.(type) {
	case resolution.StructForm:
		for _, fd := range form.Fields {
			if domain.GetStringFromField(fd, "go", "marshal") == "omit" {
				continue
			}
			locals = append(locals, localRefNames(f, fd.Type)...)
		}
		for _, e := range form.Extends {
			locals = append(locals, localRefNames(f, e)...)
		}
	case resolution.UnionForm:
		for _, v := range form.Variants {
			locals = append(locals, localRefNames(f, v.Type)...)
		}
	case resolution.DistinctForm:
		locals = localRefNames(f, form.Base)
	case resolution.AliasForm:
		locals = localRefNames(f, form.Target)
	}
	return locals
}

// definedType returns a file's full declaration for name.
func definedType(f *versions.File, name string) (resolution.Type, bool) {
	for _, t := range f.Defined {
		if t.Name == name {
			return t, true
		}
	}
	return resolution.Type{}, false
}

// localRefNames returns the same-chain type names a reference reaches.
func localRefNames(f *versions.File, ref resolution.TypeRef) []string {
	var names []string
	var walk func(ref resolution.TypeRef)
	walk = func(ref resolution.TypeRef) {
		if ref.TypeParam == nil {
			if ns, name, found := strings.Cut(ref.Name, "."); found {
				if versionNSRe.MatchString(ns) || ns == f.Chain.Resource {
					names = append(names, name)
				}
			}
		}
		for _, a := range ref.TypeArgs {
			walk(a)
		}
	}
	walk(ref)
	return names
}

// membersOf lists the member names a file's declarations reference in the given foreign
// namespace.
// pinAliasCurrent reports whether every referenced member of a lagging pin resolves to
// the same defining version at both the pinned and current versions of the dependency
// chain. When it does, the pinned and current shapes are one declaration and the lag
// is nominal.
func pinAliasCurrent(
	ctx context.Context,
	resolver *versions.Resolver,
	depLive string,
	pinned, current int,
	members []string,
) bool {
	for _, member := range members {
		curNS, _, ok := resolver.Definer(ctx, depLive, current, member)
		if !ok {
			return false
		}
		pinNS, _, ok := resolver.Definer(ctx, depLive, pinned, member)
		if !ok {
			return false
		}
		if curNS != pinNS {
			return false
		}
	}
	return true
}

func membersOf(f *versions.File, ns string) []string {
	found := make(set.Set[string])
	for _, t := range f.Defined {
		var walkRef func(ref resolution.TypeRef)
		walkRef = func(ref resolution.TypeRef) {
			if ref.TypeParam == nil {
				if refNS, name, ok := strings.Cut(ref.Name, "."); ok && refNS == ns {
					found.Add(name)
				}
			}
			for _, a := range ref.TypeArgs {
				walkRef(a)
			}
		}
		switch form := t.Form.(type) {
		case resolution.StructForm:
			for _, fd := range form.Fields {
				if domain.GetStringFromField(fd, "go", "marshal") == "omit" {
					continue
				}
				walkRef(fd.Type)
			}
			for _, e := range form.Extends {
				walkRef(e)
			}
		case resolution.UnionForm:
			for _, v := range form.Variants {
				walkRef(v.Type)
			}
		case resolution.DistinctForm:
			walkRef(form.Base)
		case resolution.AliasForm:
			walkRef(form.Target)
		}
	}
	names := found.Slice()
	slices.Sort(names)
	return names
}
