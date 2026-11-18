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

	"github.com/synnaxlabs/oracle/paths"
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
	_ context.Context, res *pipeline.Result, env Env,
) GateReport {
	start := telem.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	defer func() { r.Elapsed = telem.Since(start) }()
	if res.Resolutions == nil {
		return r
	}
	chains, resolver := res.Chains, res.Versions
	if env.Shipped != nil {
		checkShipped(&r, env.RepoRoot, chains, resolver, *env.Shipped)
	}
	if len(chains) == 0 || resolver == nil {
		return r
	}
	for _, livePath := range slices.Sorted(maps.Keys(chains)) {
		checkChain(&r, res, env, resolver, chains[livePath])
	}
	return r
}

// checkChain runs the drift and minimality checks for one chain.
func checkChain(
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
		checkLiveDrift(r, env, chain, merged)
	}

	// Import placement: every versioned dependency is pinned.
	for _, k := range chain.Numbers {
		fk, err := resolver.File(livePath, k)
		if err == nil {
			err = checkImportPlacement(r, resolver, chain, fk)
		}
		if err != nil {
			r.failVersion(chain, k, err)
			return
		}
	}

	// Pin currency: every pin the current surface's stored reference graph crosses must
	// target its dependency chain's current version.
	checkPinCurrency(r, resolver, chain)

	// Rule parity: the current package's alias of a frozen type is that frozen type, so
	// the Go rules come from the definer's version file while the other languages read
	// the live file.
	checkAliasedRules(r, res, resolver, chain)

	// Minimality: every redeclaration must differ structurally from its resolved
	// predecessor.
	for i, k := range chain.Numbers {
		if i == 0 || k > current {
			continue
		}
		if err := checkMinimality(
			r,
			resolver,
			chain,
			k,
			chain.Numbers[i-1],
		); err != nil {
			r.failVersion(chain, k, err)
			return
		}
	}
}

func checkLiveDrift(r *GateReport, env Env, chain versions.Chain, merged []byte) {
	livePath := chain.LivePath()
	filePath := livePath + ".oracle"
	onDisk, err := paths.ReadFile(filePath, env.RepoRoot)
	if err != nil && !os.IsNotExist(err) {
		r.fail(Finding{
			Path: filePath, Severity: SeverityError, Message: err.Error(),
		})
		return
	}
	if string(onDisk) == string(merged) {
		return
	}
	finding := Finding{
		Path:     filePath,
		Severity: SeverityError,
		Message:  livePath + " drifts from its version files",
		FixHint: fmt.Sprintf(
			"the version files under %s are the authority for "+
				"version-owned content; run `oracle sync` to project them",
			chain.Dir(),
		),
	}
	if env.IncludeDiffs {
		finding.Diff = unifiedDiff(filePath, string(merged), string(onDisk), 200)
	}
	r.fail(finding)
}

func (r *GateReport) failVersion(chain versions.Chain, k int, err error) {
	r.fail(Finding{
		Path:     chain.FilePath(k) + ".oracle",
		Severity: SeverityError,
		Message:  err.Error(),
	})
}

func checkMinimality(
	r *GateReport,
	resolver *versions.Resolver,
	chain versions.Chain,
	k, prev int,
) error {
	livePath := chain.LivePath()
	fk, err := resolver.File(livePath, k)
	if err != nil {
		return err
	}
	surf, err := resolver.Surface(livePath, prev)
	if err != nil {
		return err
	}
	fkTransient := fk.Transient()
	// A declaration changes when its own shape differs from its predecessor, or when it
	// references a declaration that changes. An unchanged one must alias.
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
		definer, err := resolver.File(livePath, def.Version)
		if err != nil {
			return err
		}
		if !types.StructurallyEqual(def.Type, t, definer.Table, fk.Table) {
			changed.Add(t.QualifiedName)
			continue
		}
		unchanged = append(unchanged, t)
	}
	propagateChanges(unchanged, changed, fk.Table)
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
	return nil
}

// propagateChanges adds to changed every type in unchanged that references a changed
// type, directly or through other types in unchanged.
func propagateChanges(
	unchanged []resolution.Type,
	changed set.Set[string],
	table *resolution.Table,
) {
	for grew := true; grew; {
		grew = false
		for _, t := range unchanged {
			if !changed.Contains(t.QualifiedName) &&
				referencesAny(t, changed, table) {
				changed.Add(t.QualifiedName)
				grew = true
			}
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
		ended, err := resolver.Ended(depLive)
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
	w := depWalker{local: make(map[string]resolution.Type, len(f.Defined))}
	for _, t := range f.Defined {
		w.local[t.Name] = t
	}
	for _, t := range f.Defined {
		w.decl(t, referenced, false)
	}
	visited := make(set.Set[string])
	var visitStored func(name string)
	visitStored = func(name string) {
		if visited.Contains(name) {
			return
		}
		visited.Add(name)
		t, ok := w.local[name]
		if !ok {
			return
		}
		for _, l := range w.decl(t, stored, true) {
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

type depWalker struct{ local map[string]resolution.Type }

// foreign adds the foreign namespaces ref names to into and returns the local type
// names it names.
func (w depWalker) foreign(ref resolution.TypeRef, into set.Set[string]) []string {
	var locals []string
	if ref.TypeParam == nil {
		if name, isLocal := w.localName(ref.Name, into); isLocal {
			locals = append(locals, name)
		}
	}
	for _, a := range ref.TypeArgs {
		locals = append(locals, w.foreign(a, into)...)
	}
	return locals
}

// localName returns the local type name that name refers to. A foreign name adds its
// namespace to into instead.
func (w depWalker) localName(name string, into set.Set[string]) (string, bool) {
	ns, member, qualified := strings.Cut(name, ".")
	if !qualified {
		_, isLocal := w.local[name]
		return name, isLocal
	}
	if versionNSRe.MatchString(ns) {
		return member, true
	}
	into.Add(ns)
	return "", false
}

func walkValue(v resolution.ExpressionValue, into set.Set[string]) {
	if v.Kind == resolution.ValueKindIdent {
		if ns, _, found := strings.Cut(v.IdentValue, "."); found &&
			!versionNSRe.MatchString(ns) {
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

// decl visits one declaration's references. Omit fields never reach the stored bytes:
// their targets stay out of the stored set, and so do local types reached only through
// them.
func (w depWalker) decl(
	t resolution.Type,
	into set.Set[string],
	skipOmit bool,
) []string {
	var locals []string
	visit := func(refs ...resolution.TypeRef) {
		for _, ref := range refs {
			locals = append(locals, w.foreign(ref, into)...)
		}
	}
	switch form := t.Form.(type) {
	case resolution.StructForm:
		for _, fd := range form.Fields {
			if skipOmit && domain.GetStringFromField(fd, "go", "marshal") == "omit" {
				continue
			}
			visit(fd.Type)
			if fd.Default != nil {
				walkValue(*fd.Default, into)
			}
		}
		visit(form.Extends...)
		for _, p := range form.TypeParams {
			if p.Constraint != nil {
				visit(*p.Constraint)
			}
			if p.Default != nil {
				visit(*p.Default)
			}
		}
	case resolution.EnumForm:
		visit(form.Extends...)
	case resolution.UnionForm:
		for _, v := range form.Variants {
			visit(v.Type)
		}
		visit(form.Extends...)
		visit(form.Included...)
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

// versionNSRe matches a chain-sibling namespace qualifier ("v0", "v12").
var versionNSRe = regexp.MustCompile(`^v(0|[1-9][0-9]*)$`)

// checkPinCurrency walks the stored reference graph of the chain's current surface and
// fails on any pin that lags its dependency chain's current version: the current
// package embeds the dependency's current type, so a stale pin is a contradiction. A
// lagging pin survives when every member referenced through it resolves to the same
// defining version at both the pin and the dependency's current version — the two
// shapes are one declaration, so the lag is nominal.
func checkPinCurrency(
	r *GateReport,
	resolver *versions.Resolver,
	chain versions.Chain,
) {
	surf, err := resolver.Surface(chain.LivePath(), chain.Current())
	if err != nil {
		r.failVersion(chain, chain.Current(), err)
		return
	}
	w := pinWalker{
		r:        r,
		resolver: resolver,
		chain:    chain,
		chains:   resolver.Chains(),
		visited:  make(set.Set[string]),
	}
	for _, name := range slices.Sorted(maps.Keys(surf)) {
		w.visit(chain.LivePath(), surf[name].Version, name)
	}
	checkOmittedPinCurrency(r, resolver, chain)
}

type pinWalker struct {
	r        *GateReport
	resolver *versions.Resolver
	chain    versions.Chain
	chains   map[string]versions.Chain
	visited  set.Set[string]
}

func (w pinWalker) visit(livePath string, n int, name string) {
	key := livePath + "@" + strconv.Itoa(n) + "." + name
	if w.visited.Contains(key) {
		return
	}
	w.visited.Add(key)
	f, err := w.resolver.File(livePath, n)
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
		depNS, depName, defined := w.resolver.Definer(livePath, n, l)
		if !defined {
			continue
		}
		if depLive, dv, ok := versions.ParseDepNS(depNS); ok {
			w.visit(depLive, dv, depName)
		}
	}
	for _, ns := range slices.Sorted(maps.Keys(stored)) {
		for depLive, pv := range f.Pins {
			if filepath.Base(depLive) != ns {
				continue
			}
			depChain, ok := w.chains[depLive]
			if !ok || !w.checkPin(f, ns, depLive, pv, depChain) {
				continue
			}
			for _, member := range membersOf(f, ns, false) {
				w.visit(depLive, pv, member)
			}
		}
	}
}

// checkPin reports a stored pin of f on depLive at pv that targets an ended chain or
// lags its current version. It returns false when the chain ended.
func (w pinWalker) checkPin(
	f *versions.File,
	ns, depLive string,
	pv int,
	depChain versions.Chain,
) bool {
	chain := w.chain
	if ended, err := w.resolver.Ended(depLive); err == nil && ended {
		w.r.fail(Finding{
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
		return false
	}
	if pv == depChain.Current() || pinAliasCurrent(
		w.resolver, depLive, pv, depChain.Current(), membersOf(f, ns, false),
	) {
		return true
	}
	w.r.fail(Finding{
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
	return true
}

// checkOmittedPinCurrency fails a pin that an omitted field of a current-surface member
// uses when it lags its dependency's current version. The member may be declared in an
// earlier version file that the current one aliases. The current package types those
// fields at the pin, and the Core assigns the dependency's root type to them.
func checkOmittedPinCurrency(
	r *GateReport,
	resolver *versions.Resolver,
	chain versions.Chain,
) {
	path := chain.FilePath(chain.Current()) + ".oracle"
	surf, err := resolver.Surface(chain.LivePath(), chain.Current())
	if err != nil {
		r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
		return
	}
	byVersion := make(map[int][]string)
	for name, def := range surf {
		byVersion[def.Version] = append(byVersion[def.Version], name)
	}
	for _, n := range slices.Sorted(maps.Keys(byVersion)) {
		f, err := resolver.File(chain.LivePath(), n)
		if err != nil {
			r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
			return
		}
		members := surfaceMembers(f, byVersion[n])
		for _, depLive := range slices.Sorted(maps.Keys(f.Pins)) {
			ns := filepath.Base(depLive)
			depChain, ok := resolver.Chains()[depLive]
			omitted := omittedOnlyMembers(members, ns)
			if !ok || len(omitted) == 0 {
				continue
			}
			pv := f.Pins[depLive]
			if pv == depChain.Current() || pinAliasCurrent(
				resolver, depLive, pv, depChain.Current(), omitted,
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

// surfaceMembers returns a file holding the declarations of f named by names.
func surfaceMembers(f *versions.File, names []string) *versions.File {
	members := &versions.File{Chain: f.Chain, N: f.N, Pins: f.Pins}
	for _, name := range names {
		t, ok := versions.DefinedType(f, name)
		if !ok {
			continue
		}
		members.Defined = append(members.Defined, t)
		// An inline variant's payload is a synthetic declaration the surface leaves
		// out, so its fields join through the union.
		u, isUnion := t.Form.(resolution.UnionForm)
		if !isUnion {
			continue
		}
		for _, v := range u.Variants {
			name := v.Type.Name[strings.LastIndex(v.Type.Name, ".")+1:]
			if p, ok := versions.DefinedType(f, name); ok && v.Inline {
				members.Defined = append(members.Defined, p)
			}
		}
	}
	return members
}

// omittedOnlyMembers lists the members of ns that only omitted fields reach. The stored
// check covers stored members.
func omittedOnlyMembers(members *versions.File, ns string) []string {
	stored := set.New(membersOf(members, ns, false)...)
	var omitted []string
	for _, m := range membersOf(members, ns, true) {
		if !stored.Contains(m) {
			omitted = append(omitted, m)
		}
	}
	return omitted
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
			locals = append(locals, localRefNames(fd.Type)...)
		}
		for _, e := range form.Extends {
			locals = append(locals, localRefNames(e)...)
		}
	case resolution.UnionForm:
		for _, v := range form.Variants {
			locals = append(locals, localRefNames(v.Type)...)
		}
		if form.BorrowsEnum() {
			locals = append(locals, localRefNames(form.Enum)...)
		}
	case resolution.DistinctForm:
		locals = localRefNames(form.Base)
	case resolution.AliasForm:
		locals = localRefNames(form.Target)
	}
	return locals
}

// localRefNames returns the same-chain type names a reference reaches.
func localRefNames(ref resolution.TypeRef) []string {
	var names []string
	var walk func(ref resolution.TypeRef)
	walk = func(ref resolution.TypeRef) {
		if ref.TypeParam == nil {
			if ns, name, found := strings.Cut(ref.Name, "."); found {
				if versionNSRe.MatchString(ns) {
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
// the lag is nominal. It skips a member that the pinned version does not carry, since
// another import of the namespace declares it.
func pinAliasCurrent(
	resolver *versions.Resolver,
	depLive string,
	pinned, current int,
	members []string,
) bool {
	for _, member := range members {
		pinNS, _, ok := resolver.Definer(depLive, pinned, member)
		if !ok {
			continue
		}
		curNS, _, ok := resolver.Definer(depLive, current, member)
		if !ok || curNS != pinNS {
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
		walkRef := func(refs ...resolution.TypeRef) {
			for _, ref := range refs {
				addMembers(ref, ns, found)
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
			walkRef(form.Extends...)
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

func addMembers(ref resolution.TypeRef, ns string, found set.Set[string]) {
	if ref.TypeParam == nil {
		if refNS, name, ok := strings.Cut(ref.Name, "."); ok && refNS == ns {
			found.Add(name)
		}
	}
	for _, a := range ref.TypeArgs {
		addMembers(a, ns, found)
	}
}

// aliasedGoRules names the type-level @go expressions that generate Go methods or
// fields.
var aliasedGoRules = set.New("to_any", "fields", "imports")

// checkAliasedRules fails when the current version aliases a type from a frozen definer
// and the live type's @validate, @default, or @go method rules differ from the
// definer's. Inline variant payloads of an aliased union count as aliased.
func checkAliasedRules(
	r *GateReport,
	res *pipeline.Result,
	resolver *versions.Resolver,
	chain versions.Chain,
) {
	livePath, current := chain.LivePath(), chain.Current()
	path := chain.FilePath(current) + ".oracle"
	ended, err := resolver.Ended(livePath)
	if err != nil {
		r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
		return
	}
	if ended {
		return
	}
	surf, err := resolver.Surface(livePath, current)
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
		definer, err := resolver.File(livePath, def.Version)
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
	checkReachedRules(r, res, resolver, chain, definers)
}

// checkReachedRules fails a live-only rule on a dependency that a frozen definer
// reaches at the dependency's current version. A frozen definer's package calls every
// dependency its pins reach, directly or through the dependencies' own pins, by the
// dependency's version file, so a live-only rule there never runs.
func checkReachedRules(
	r *GateReport,
	res *pipeline.Result,
	resolver *versions.Resolver,
	chain versions.Chain,
	definers map[int]*versions.File,
) {
	livePath := chain.LivePath()
	path := chain.FilePath(chain.Current()) + ".oracle"
	var queue []reachedPin
	for _, v := range slices.Sorted(maps.Keys(definers)) {
		for _, depLive := range slices.Sorted(maps.Keys(definers[v].Pins)) {
			queue = append(queue, reachedPin{v, definers[v].Pins[depLive], depLive})
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
		depFile, err := resolver.File(p.livePath, p.version)
		if err != nil {
			r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
			return
		}
		for _, next := range slices.Sorted(maps.Keys(depFile.Pins)) {
			queue = append(queue, reachedPin{p.origin, depFile.Pins[next], next})
		}
		ended, err := resolver.Ended(p.livePath)
		if err != nil {
			r.fail(Finding{Path: path, Severity: SeverityError, Message: err.Error()})
			return
		}
		if ended || p.version != depChain.Current() {
			continue
		}
		failReachedDrift(r, res, livePath, p, depChain, depFile)
	}
}

// reachedPin is a pin that frozen version origin reaches.
type reachedPin struct {
	origin, version int
	livePath        string
}

// failReachedDrift fails every live-only rule in depFile, which frozen livePath reaches
// through p.
func failReachedDrift(
	r *GateReport,
	res *pipeline.Result,
	livePath string,
	p reachedPin,
	depChain versions.Chain,
	depFile *versions.File,
) {
	origin, depLive, version := p.origin, p.livePath, p.version
	for _, t := range depFile.Defined {
		if t.Synthetic {
			continue
		}
		for _, rule := range rulesDrift(res, depLive, t, depFile.Table, false) {
			r.fail(Finding{
				Path:     depChain.FilePath(version) + ".oracle",
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"frozen %s v%d reaches %s v%d, but the live %s does not "+
						"match v%d's",
					livePath, origin, depLive, version, rule, version,
				),
				FixHint: fmt.Sprintf(
					"frozen %s v%d calls %s's v%d rules: restate the rule in "+
						"%s v%d",
					livePath, origin, depLive, version, depLive, version,
				),
			})
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
	for _, d := range []string{"validate", "default"} {
		if exprsKey(live.Domains[d].Expressions) !=
			exprsKey(frozen.Domains[d].Expressions) {
			drift = append(drift, "@"+d+" on "+live.Name)
		}
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
