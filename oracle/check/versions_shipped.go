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
	"path"
	"slices"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin/go/types"
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

// checkShipped fails on every version file that changes the stored shape a stable
// release shipped. A shipped version is frozen: a shape change needs a new version.
// Docs, validation rules, and omit-transient types do not reach the stored bytes, so
// they may change.
func checkShipped(
	ctx context.Context,
	r *GateReport,
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
	for _, livePath := range slices.Sorted(maps.Keys(shippedChains)) {
		chain := shippedChains[livePath]
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
			current, ok := resolver.Chains()[livePath]
			if !ok || !slices.Contains(current.Numbers, n) {
				fail("is gone")
				continue
			}
			old, err := shipped.File(ctx, livePath, n)
			if err != nil {
				r.fail(Finding{
					Path:     filePath,
					Severity: SeverityError,
					Message:  fmt.Sprintf("%s at %s: %s", filePath, release.Tag, err),
				})
				continue
			}
			f, err := resolver.File(ctx, livePath, n)
			if err != nil {
				r.fail(Finding{
					Path: filePath, Severity: SeverityError, Message: err.Error(),
				})
				continue
			}
			compareShipped(old, f, fail)
		}
	}
}

// compareShipped reports each stored-shape difference between the shipped and the
// current form of one version file.
func compareShipped(old, f *versions.File, fail func(format string, args ...any)) {
	persisted := old.Persisted()
	persisted.Add(f.Persisted().Slice()...)
	// A union compares its inline variant payloads itself.
	for _, t := range slices.Concat(old.Defined, f.Defined) {
		if t.Synthetic {
			persisted.Remove(t.Name)
		}
	}
	oldDefined, defined := declared(old), declared(f)
	for _, name := range slices.Sorted(maps.Keys(persisted)) {
		oldAlias, wasAlias := old.Aliases[name]
		oldType, wasDefined := oldDefined[name]
		alias, isAlias := f.Aliases[name]
		t, isDefined := defined[name]
		switch {
		case !wasAlias && !wasDefined:
			fail("adds %s", name)
		case !isAlias && !isDefined:
			fail("removes %s", name)
		case wasAlias && (!isAlias || alias != oldAlias):
			fail("changes the alias line of %s", name)
		case wasDefined && !isDefined:
			fail("replaces the declaration of %s with an alias line", name)
		case wasDefined &&
			!types.StructurallyEqual(oldType, t, old.Table, f.Table):
			fail("changes the shape of %s", name)
		}
	}
	oldStored, _ := classifyDeps(old)
	stored, _ := classifyDeps(f)
	for _, depLive := range slices.Sorted(maps.Keys(old.Pins)) {
		ns := path.Base(depLive)
		if !oldStored.Contains(ns) && !stored.Contains(ns) {
			continue
		}
		pv, ok := f.Pins[depLive]
		if !ok {
			fail("drops its pin of %s v%d", depLive, old.Pins[depLive])
		} else if pv != old.Pins[depLive] {
			fail("moves its pin of %s from v%d to v%d", depLive, old.Pins[depLive], pv)
		}
	}
}

// declared maps the non-synthetic declarations of f by name.
func declared(f *versions.File) map[string]resolution.Type {
	m := make(map[string]resolution.Type, len(f.Defined))
	for _, t := range f.Defined {
		if !t.Synthetic {
			m[t.Name] = t
		}
	}
	return m
}
