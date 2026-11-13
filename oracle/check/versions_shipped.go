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
	"cmp"
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
)

// Release is the schema tree of a stable release.
type Release struct {
	// Tag names the release, such as core/v0.60.1.
	Tag string
	// Root is a directory that holds the release's schemas directory.
	Root string
}

// checkShipped fails on every version file whose stored types no longer decode the
// data a stable release wrote. It compares filled-in shapes, so a type may move to
// another file or chain, and an enum or union may gain members. A chain removed with
// its live file under repoRoot passes unless a current chain stores a type of the same
// name. resolver resolves the current chains, and is nil when there are none.
func checkShipped(
	ctx context.Context,
	r *GateReport,
	repoRoot string,
	chains map[string]versions.Chain,
	resolver *versions.Resolver,
	release Release,
) {
	shippedChains, err := versions.Discover(release.Root)
	if err != nil {
		r.fail(Finding{Severity: SeverityError, Message: err.Error()})
		return
	}
	shipped := versions.NewResolver(
		shippedChains, analyzer.NewStandardFileLoader(release.Root),
	)
	var owners map[string]string
	for _, livePath := range slices.Sorted(maps.Keys(shippedChains)) {
		chain := shippedChains[livePath]
		current, ok := chains[livePath]
		if !ok {
			live, err := liveExists(repoRoot, livePath)
			if err != nil {
				r.fail(Finding{
					Path:     livePath + ".oracle",
					Severity: SeverityError,
					Message:  err.Error(),
				})
				continue
			}
			if !live {
				if owners == nil {
					owners = storedOwners(ctx, r, chains, resolver)
				}
				checkRemoved(ctx, r, shipped, chain, owners, release)
				continue
			}
		}
		for _, n := range chain.Numbers {
			filePath := chain.FilePath(n) + ".oracle"
			fail := func(format string, args ...any) {
				r.fail(Finding{
					Path:     filePath,
					Severity: SeverityError,
					Message: fmt.Sprintf(
						"%s v%d %s, which %s shipped",
						livePath, n, fmt.Sprintf(format, args...), release.Tag,
					),
					FixHint: fmt.Sprintf(
						"restore %s from %s and change the shape in a new version",
						filePath, release.Tag,
					),
				})
			}
			if !ok || !slices.Contains(current.Numbers, n) {
				fail("is gone")
				continue
			}
			ns := versions.DepNS(livePath, n)
			old, now := resolution.NewTable(), resolution.NewTable()
			if err := shipped.SurfaceInto(ctx, old, livePath, n, ns); err != nil {
				r.fail(Finding{
					Path:     filePath,
					Severity: SeverityError,
					Message:  fmt.Sprintf("%s at %s: %s", filePath, release.Tag, err),
				})
				continue
			}
			if err := resolver.SurfaceInto(ctx, now, livePath, n, ns); err != nil {
				r.fail(Finding{
					Path: filePath, Severity: SeverityError, Message: err.Error(),
				})
				continue
			}
			compareStored(old, now, ns, fail)
		}
	}
}

// storedOwners returns the live path of the current chain that stores each type name.
func storedOwners(
	ctx context.Context,
	r *GateReport,
	chains map[string]versions.Chain,
	resolver *versions.Resolver,
) map[string]string {
	owners := make(map[string]string)
	for _, livePath := range slices.Sorted(maps.Keys(chains)) {
		names, err := storedNames(ctx, resolver, chains[livePath])
		if err != nil {
			r.fail(Finding{
				Path:     livePath + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
			continue
		}
		for _, name := range names {
			owners[name] = livePath
		}
	}
	return owners
}

// checkRemoved fails a removed chain whose stored types a current chain still stores
// under the same names, since the Core would read the shipped data with them.
func checkRemoved(
	ctx context.Context,
	r *GateReport,
	shipped *versions.Resolver,
	chain versions.Chain,
	owners map[string]string,
	release Release,
) {
	livePath := chain.LivePath()
	names, err := storedNames(ctx, shipped, chain)
	if err != nil {
		r.fail(Finding{
			Path:     livePath + ".oracle",
			Severity: SeverityError,
			Message:  fmt.Sprintf("%s at %s: %s", livePath, release.Tag, err),
		})
		return
	}
	for _, name := range names {
		owner, ok := owners[name]
		if !ok {
			continue
		}
		r.fail(Finding{
			Path:     livePath + ".oracle",
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"%s is gone, but %s still stores %s, which %s shipped",
				livePath, owner, name, release.Tag,
			),
			FixHint: fmt.Sprintf(
				"restore %s from %s, or give the stored type a new name",
				chain.Dir(), release.Tag,
			),
		})
	}
}

// storedNames returns the sorted names of the stored types in the current version of
// chain.
func storedNames(
	ctx context.Context, resolver *versions.Resolver, chain versions.Chain,
) ([]string, error) {
	livePath, n := chain.LivePath(), chain.Current()
	ns := versions.DepNS(livePath, n)
	t := resolution.NewTable()
	if err := resolver.SurfaceInto(ctx, t, livePath, n, ns); err != nil {
		return nil, err
	}
	stored := storedTypes(t, ns)
	names := make([]string, len(stored))
	for i, typ := range stored {
		names[i] = typ.Name
	}
	return names, nil
}

// storedTypes returns the types in namespace ns of t that have a codec or a migration,
// sorted by name.
func storedTypes(t *resolution.Table, ns string) []resolution.Type {
	var stored []resolution.Type
	for _, typ := range t.TypesInNamespace(ns) {
		if !typ.Synthetic && (domain.HasExprFromType(typ, "go", "marshal") ||
			domain.HasExprFromType(typ, "go", "migrate")) {
			stored = append(stored, typ)
		}
	}
	slices.SortFunc(
		stored,
		func(a, b resolution.Type) int { return cmp.Compare(a.Name, b.Name) },
	)
	return stored
}

// liveExists reports whether the live file of the chain at livePath exists under root.
func liveExists(root, livePath string) (bool, error) {
	_, err := os.Stat(filepath.Join(root, livePath+".oracle"))
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// compareStored checks each stored type of the surface in namespace ns: every type
// with a codec or a migration. A stored type the surface no longer carries is allowed
// only when the comparison of a stored type that remains reaches it.
func compareStored(
	old, now *resolution.Table, ns string, fail func(format string, args ...any),
) {
	s := newShapes(old, now)
	var removed []resolution.Type
	for _, t := range storedTypes(old, ns) {
		n, ok := now.Get(t.QualifiedName)
		if !ok {
			removed = append(removed, t)
			continue
		}
		if reason := s.root(t, n); reason != "" {
			fail("changes the stored shape of %s (%s)", t.Name, reason)
		}
	}
	for _, t := range removed {
		if !s.covered.Contains(t.QualifiedName) {
			fail("removes %s", t.Name)
		}
	}
}
