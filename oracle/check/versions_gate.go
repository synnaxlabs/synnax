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

	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// VersionsGate verifies the explicitly managed version chains: the live file's
// version-owned content matches chain resolution (the merged projection), every
// redeclaration differs structurally from its resolved predecessor, and no version file
// changes a stored shape that Env.Shipped shipped.
type versionsGate struct{}

// NewVersionsGate returns a Checker that asserts the version chain is well formed.
func NewVersionsGate() Checker { return versionsGate{} }

func (versionsGate) Name() string { return "versions" }

func (g versionsGate) Run(
	ctx context.Context, res *pipeline.Result, env Env,
) GateReport {
	start := telem.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	defer func() { r.Elapsed = telem.Since(start) }()
	if res.Resolutions == nil {
		return r
	}
	chains, resolver := res.Chains, res.Versions
	if env.Shipped != nil {
		checkShipped(ctx, &r, chains, resolver, *env.Shipped)
	}
	if len(chains) == 0 || resolver == nil {
		return r
	}
	for _, livePath := range slices.Sorted(maps.Keys(chains)) {
		checkChain(ctx, &r, res, env, resolver, chains[livePath])
	}
	return r
}

// checkChain runs the drift and minimality checks for one chain.
func checkChain(
	ctx context.Context,
	r *GateReport,
	res *pipeline.Result,
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
	if merged, ok := res.MergedSources[livePath+".oracle"]; ok {
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

	// Import placement: every versioned dependency is pinned.
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
		if err := checkImportPlacement(ctx, r, resolver, chain, fk); err != nil {
			r.fail(Finding{
				Path:     chain.FilePath(k) + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
			return
		}
	}

	// Pin currency: every pin the current surface's stored reference graph crosses must
	// target its dependency chain's current version.
	checkPinCurrency(ctx, r, resolver, chain)

	// Rule parity: the current package's alias of a frozen type is that frozen type, so
	// the Go rules come from the definer's version file while the other languages read
	// the live file.
	checkAliasedRules(ctx, r, res, resolver, chain)

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
		// A declaration changes when its own shape differs from its predecessor, or
		// when it references a declaration that changes. An unchanged one must alias.
		changed := make(set.Set[string], len(fk.Defined))
		var unchanged []resolution.Type
		for _, t := range fk.Defined {
			if t.Synthetic {
				continue
			}
			def, ok := surf[t.Name]
			if !ok || fkTransient.Contains(t.Name) {
				changed.Add(t.QualifiedName)
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
			if !types.StructurallyEqual(def.Type, t, definer.Table, fk.Table) {
				changed.Add(t.QualifiedName)
				continue
			}
			unchanged = append(unchanged, t)
		}
		for grew := true; grew; {
			grew = false
			for _, t := range unchanged {
				if !changed.Contains(t.QualifiedName) &&
					referencesAny(t, changed, fk.Table) {
					changed.Add(t.QualifiedName)
					grew = true
				}
			}
		}
		for _, t := range unchanged {
			if changed.Contains(t.QualifiedName) {
				continue
			}
			def := surf[t.Name]
			r.fail(Finding{
				Path:     chain.FilePath(k) + ".oracle",
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"%s v%d redeclares %s identically to v%d",
					livePath, k, t.Name, def.Version,
				),
				FixHint: fmt.Sprintf(
					"use an alias: %s = v%d.%s", t.Name, def.Version, t.Name,
				),
			})
		}
	}
}

// referencesAny reports whether t's fields, variants, or extended types name a type in
// names. An inline variant's payload is part of its union, so its references count as
// the union's.
func referencesAny(
	t resolution.Type,
	names set.Set[string],
	table *resolution.Table,
) bool {
	var refs []resolution.TypeRef
	var collect func(form resolution.TypeForm)
	collect = func(form resolution.TypeForm) {
		switch form := form.(type) {
		case resolution.StructForm:
			for _, f := range form.Fields {
				refs = append(refs, f.Type)
			}
			refs = append(refs, form.Extends...)
		case resolution.UnionForm:
			for _, v := range form.Variants {
				payload, ok := v.Type.Resolve(table)
				if v.Inline && ok {
					collect(payload.Form)
				} else {
					refs = append(refs, v.Type)
				}
			}
			refs = append(refs, form.Extends...)
		}
	}
	collect(t.Form)
	for len(refs) > 0 {
		ref := refs[len(refs)-1]
		refs = append(refs[:len(refs)-1], ref.TypeArgs...)
		if names.Contains(ref.Name) {
			return true
		}
	}
	return false
}

// fail records an error finding and flips the gate status.
func (r *GateReport) fail(f Finding) {
	r.Findings = append(r.Findings, f)
	r.Status = StatusFail
}

// checkImportPlacement verifies that one version file pins every versioned dependency
// it references. A version package renders each reference against the pinned version,
// so a live import would let the dependency's next version move the package.
func checkImportPlacement(
	ctx context.Context,
	r *GateReport,
	resolver *versions.Resolver,
	chain versions.Chain,
	f *versions.File,
) error {
	_, referenced := classifyDeps(f)
	path := chain.FilePath(f.N) + ".oracle"
	live := make(map[string]string, len(f.Live))
	for i, ns := range f.Live {
		live[ns] = f.LivePaths[i]
	}
	for _, ns := range slices.Sorted(maps.Keys(referenced)) {
		depLive, isLive := live[ns]
		if !isLive {
			continue
		}
		// A chainless dependency has no version file to pin, and an ended chain's live
		// schema is hand-owned: either way the live schema is the one shape (arc's
		// Program).
		if _, versioned := resolver.Chains()[depLive]; !versioned {
			continue
		}
		ended, err := resolver.Ended(ctx, depLive)
		if err != nil {
			return err
		}
		if ended {
			continue
		}
		r.fail(Finding{
			Path:     path,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"%s v%d references %s but imports its live schema",
				chain.LivePath(), f.N, ns,
			),
			FixHint: fmt.Sprintf(
				"a version package resolves every reference at a fixed version; "+
					"import a %s/%s version file",
				depLive, "versions",
			),
		})
	}
	return nil
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
	// walkDecl visits one declaration's references. Omit fields never reach the stored
	// bytes: their targets stay out of the stored set, and so do local types reached
	// only through them.
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
			for _, e := range form.Included {
				visit(e)
			}
			if form.BorrowsEnum() {
				visit(form.Enum)
			}
		case resolution.DistinctForm:
			visit(form.Base)
		case resolution.AliasForm:
			visit(form.Target)
		}
		return locals
	}

	for _, t := range f.Defined {
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
	// the omit-transient ones, which never reach the stored bytes.
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
func checkPinCurrency(
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
		t, ok := versions.DefinedType(f, name)
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
				if ended, err := resolver.Ended(ctx, depLive); err == nil && ended {
					r.fail(Finding{
						Path:     chain.FilePath(chain.Current()) + ".oracle",
						Severity: SeverityError,
						Message: fmt.Sprintf(
							"%s's current surface stores %s, but %s's chain "+
								"ended at v%d; nothing may persist an ended "+
								"resource",
							chain.LivePath(), ns, depLive, depChain.Current(),
						),
						FixHint: fmt.Sprintf(
							"drop the stored reference, or revive %s by "+
								"declaring members in a new version file",
							depLive,
						),
					})
					continue
				}
				if pv != depChain.Current() &&
					!pinAliasCurrent(
						ctx, resolver, depLive, pv, depChain.Current(),
						membersOf(f, ns, false),
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
				for _, member := range membersOf(f, ns, false) {
					visit(depLive, pv, member)
				}
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(surf)) {
		visit(livePath, surf[name].Version, name)
	}
	checkOmittedPinCurrency(ctx, r, resolver, chain)
}

// checkOmittedPinCurrency fails a pin that an omitted field of a current-surface member
// uses when it lags its dependency's current version. The member may be declared in an
// earlier version file that the current one aliases. The current package types those
// fields at the pin, and the Core assigns the dependency's root type to them.
func checkOmittedPinCurrency(
	ctx context.Context,
	r *GateReport,
	resolver *versions.Resolver,
	chain versions.Chain,
) {
	path := chain.FilePath(chain.Current()) + ".oracle"
	surf, err := resolver.Surface(ctx, chain.LivePath(), chain.Current())
	if err != nil {
		r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
		return
	}
	byVersion := make(map[int][]string)
	for name, def := range surf {
		byVersion[def.Version] = append(byVersion[def.Version], name)
	}
	for _, n := range slices.Sorted(maps.Keys(byVersion)) {
		f, err := resolver.File(ctx, chain.LivePath(), n)
		if err != nil {
			r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
			return
		}
		members := &versions.File{Chain: f.Chain, N: f.N, Pins: f.Pins}
		for _, name := range byVersion[n] {
			t, ok := versions.DefinedType(f, name)
			if !ok {
				continue
			}
			members.Defined = append(members.Defined, t)
			// An inline variant's payload is a synthetic declaration the surface leaves
			// out, so its fields join through the union.
			if u, isUnion := t.Form.(resolution.UnionForm); isUnion {
				for _, v := range u.Variants {
					name := v.Type.Name[strings.LastIndex(v.Type.Name, ".")+1:]
					if p, ok := versions.DefinedType(f, name); ok && v.Inline {
						members.Defined = append(members.Defined, p)
					}
				}
			}
		}
		for _, depLive := range slices.Sorted(maps.Keys(f.Pins)) {
			ns := filepath.Base(depLive)
			depChain, ok := resolver.Chains()[depLive]
			// The stored check covers stored members, so this one checks only the
			// members that omitted fields alone reach.
			stored := set.New(membersOf(members, ns, false)...)
			var omitted []string
			for _, m := range membersOf(members, ns, true) {
				if !stored.Contains(m) {
					omitted = append(omitted, m)
				}
			}
			if !ok || len(omitted) == 0 {
				continue
			}
			pv := f.Pins[depLive]
			if pv == depChain.Current() || pinAliasCurrent(
				ctx, resolver, depLive, pv, depChain.Current(), omitted,
			) {
				continue
			}
			r.fail(Finding{
				Path:     path,
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"%s's current surface resolves %s at v%d for omitted fields, "+
						"but %s's current version is v%d",
					chain.LivePath(), ns, pv, depLive, depChain.Current(),
				),
				FixHint: fmt.Sprintf(
					"omitted fields leave the stored bytes alone: bump the pin in "+
						"v%d in place, redeclaring any member it aliases",
					chain.Current(),
				),
			})
		}
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
		if form.BorrowsEnum() {
			locals = append(locals, localRefNames(f, form.Enum)...)
		}
	case resolution.DistinctForm:
		locals = localRefNames(f, form.Base)
	case resolution.AliasForm:
		locals = localRefNames(f, form.Target)
	}
	return locals
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
// namespace. pinAliasCurrent reports whether every referenced member of a lagging pin
// resolves to the same defining version at both the pinned and current versions of the
// dependency chain. When it does, the pinned and current shapes are one declaration and
// the lag is nominal.
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

// membersOf returns the names in ns that f's declarations reference. Omitted fields
// count only when omitted is set.
func membersOf(f *versions.File, ns string, omitted bool) []string {
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
				if !omitted &&
					domain.GetStringFromField(fd, "go", "marshal") == "omit" {
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
			if form.BorrowsEnum() {
				walkRef(form.Enum)
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

// aliasedGoRules names the type-level @go expressions that generate Go methods or
// fields.
var aliasedGoRules = set.New("to_any", "fields", "imports")

// checkAliasedRules fails when the current version aliases a type from a frozen definer
// and the live type's @validate, @default, or @go method rules differ from the
// definer's. Inline variant payloads of an aliased union count as aliased.
func checkAliasedRules(
	ctx context.Context,
	r *GateReport,
	res *pipeline.Result,
	resolver *versions.Resolver,
	chain versions.Chain,
) {
	livePath, current := chain.LivePath(), chain.Current()
	path := chain.FilePath(current) + ".oracle"
	ended, err := resolver.Ended(ctx, livePath)
	if err != nil {
		r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
		return
	}
	if ended {
		return
	}
	surf, err := resolver.Surface(ctx, livePath, current)
	if err != nil {
		r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
		return
	}
	definers := make(map[int]*versions.File)
	for _, name := range slices.Sorted(maps.Keys(surf)) {
		def := surf[name]
		if def.Version == current {
			continue
		}
		definer, err := resolver.File(ctx, livePath, def.Version)
		if err != nil {
			r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
			return
		}
		definers[def.Version] = definer
		for _, rule := range rulesDrift(
			res, livePath, def.Type, definer.Table, true,
		) {
			r.fail(Finding{
				Path:     path,
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"%s v%d aliases %s from v%d, but the live %s does not match v%d's",
					livePath, current, name, def.Version, rule, def.Version,
				),
				FixHint: fmt.Sprintf(
					"the Go alias uses v%d's rules: restate the rule in v%d",
					def.Version, def.Version,
				),
			})
		}
	}
	// A frozen definer's package calls every dependency its pins reach, directly or
	// through the dependencies' own pins, by the dependency's version file, so a
	// live-only rule on a dependency at its current version never runs.
	type pin struct {
		origin, version int
		livePath        string
	}
	var queue []pin
	for _, v := range slices.Sorted(maps.Keys(definers)) {
		for _, depLive := range slices.Sorted(maps.Keys(definers[v].Pins)) {
			queue = append(queue, pin{v, definers[v].Pins[depLive], depLive})
		}
	}
	visited := make(set.Set[string])
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		key := p.livePath + "@" + strconv.Itoa(p.version)
		depChain, versioned := resolver.Chains()[p.livePath]
		if !versioned || visited.Contains(key) {
			continue
		}
		visited.Add(key)
		depFile, err := resolver.File(ctx, p.livePath, p.version)
		if err != nil {
			r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
			return
		}
		for _, next := range slices.Sorted(maps.Keys(depFile.Pins)) {
			queue = append(queue, pin{p.origin, depFile.Pins[next], next})
		}
		ended, err := resolver.Ended(ctx, p.livePath)
		if err != nil {
			r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
			return
		}
		if ended || p.version != depChain.Current() {
			continue
		}
		for _, t := range depFile.Defined {
			if t.Synthetic {
				continue
			}
			for _, rule := range rulesDrift(
				res, p.livePath, t, depFile.Table, false,
			) {
				r.fail(Finding{
					Path:     depChain.FilePath(p.version) + ".oracle",
					Severity: SeverityError,
					Message: fmt.Sprintf(
						"frozen %s v%d reaches %s v%d, but the live %s does not "+
							"match v%d's",
						livePath, p.origin, p.livePath, p.version, rule, p.version,
					),
					FixHint: fmt.Sprintf(
						"frozen %s v%d calls %s's v%d rules: restate the rule in "+
							"%s v%d",
						livePath, p.origin, p.livePath, p.version, p.livePath,
						p.version,
					),
				})
			}
		}
	}
}

// rulesDrift describes every rule that differs between livePath's live declaration of
// def and def itself, including the inline variant payloads of a union. goRules
// includes the @go method rules, which only an alias of def carries.
func rulesDrift(
	res *pipeline.Result,
	livePath string,
	def resolution.Type,
	table *resolution.Table,
	goRules bool,
) []string {
	frozen := []resolution.Type{def}
	if u, isUnion := def.Form.(resolution.UnionForm); isUnion {
		for _, v := range u.Variants {
			if payload, ok := v.Type.Resolve(table); v.Inline && ok {
				frozen = append(frozen, payload)
			}
		}
	}
	var drift []string
	for _, ft := range frozen {
		for _, t := range res.Resolutions.Types {
			if t.FilePath == livePath+".oracle" && t.Name == ft.Name {
				drift = append(drift, ruleDrift(t, ft, goRules)...)
				break
			}
		}
	}
	return drift
}

// ruleDrift describes every rule that differs between the live and frozen declarations
// of one type.
func ruleDrift(live, frozen resolution.Type, goRules bool) []string {
	var drift []string
	if goRules && goRulesKey(live) != goRulesKey(frozen) {
		drift = append(drift, "@go method rules on "+live.Name)
	}
	lf, ok := live.Form.(resolution.StructForm)
	if !ok {
		return drift
	}
	ff, ok := frozen.Form.(resolution.StructForm)
	if !ok {
		return drift
	}
	frozenFields := make(map[string]resolution.Field, len(ff.Fields))
	for _, f := range ff.Fields {
		frozenFields[f.Name] = f
	}
	for _, f := range lf.Fields {
		fr, ok := frozenFields[f.Name]
		if !ok {
			continue
		}
		for _, d := range []string{"validate", "default"} {
			if exprsKey(
				f.Domains[d].Expressions,
			) != exprsKey(
				fr.Domains[d].Expressions,
			) {
				drift = append(drift, "@"+d+" on "+live.Name+"."+f.Name)
			}
		}
	}
	return drift
}

// goRulesKey returns a comparable key of t's Go method expressions.
func goRulesKey(t resolution.Type) string {
	var exprs resolution.Expressions
	for _, e := range t.Domains["go"].Expressions {
		if aliasedGoRules.Contains(e.Name) {
			exprs = append(exprs, e)
		}
	}
	return exprsKey(exprs)
}

// exprsKey returns an order-insensitive key of exprs' names and values.
func exprsKey(exprs resolution.Expressions) string {
	keys := make([]string, len(exprs))
	for i, e := range exprs {
		keys[i] = fmt.Sprintf("%s %+v", e.Name, e.Values)
	}
	slices.Sort(keys)
	return strings.Join(keys, "\n")
}
