// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"

	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/x/errors"
)

// Scaffold renders the next version file for chain: every current-surface member as an
// alias line to its definer, omit-transient declarations redeclared in full, and the
// file's imports carried forward. The developer converts the changed types to full
// declarations and bumps stale pins; an all-alias file fails the minimality gate by
// design.
func Scaffold(ctx context.Context, r *Resolver, chain Chain) (string, error) {
	current := chain.Current()
	cur, err := r.File(ctx, chain.LivePath(), current)
	if err != nil {
		return "", err
	}
	surf, err := r.Surface(ctx, chain.LivePath(), current)
	if err != nil {
		return "", err
	}
	opts := RenderOptions{
		Imports: fileImports(cur),
		Qualifier: func(ns string) string {
			if versionNS.MatchString(ns) || ns == chain.Resource {
				return ""
			}
			if live, _, ok := ParseDepNS(ns); ok {
				return resourceOf(live)
			}
			return ns
		},
		Resolve: cur.Table.Get,
	}
	transient := cur.Transient()
	var decls []Decl
	for _, name := range cur.Order {
		def, member := surf[name]
		if !member {
			continue
		}
		if transient.Contains(name) {
			decls = append(decls, Decl{Type: def.Type})
			continue
		}
		decls = append(decls, Decl{
			Type:  def.Type,
			Alias: &Alias{Version: def.Version, Name: name},
		})
	}
	formatted, err := formatter.Format(Render(decls, opts))
	if err != nil {
		return "", errors.Wrapf(
			err, "failed to format scaffold for %s", chain.LivePath(),
		)
	}
	return license(ctx, r.RepoRoot(), formatted)
}

// fileImports reconstructs a file's import paths: pins as version-file paths, live
// imports verbatim.
func fileImports(f *File) []string {
	imports := make([]string, 0, len(f.Pins)+len(f.LivePaths))
	for depLive, pv := range f.Pins {
		depChain := Chain{Resource: resourceOf(depLive)}
		depChain.Domain = depLive[len("schemas/") : len(depLive)-
			len(depChain.Resource)-1]
		imports = append(imports, depChain.FilePath(pv))
	}
	imports = append(imports, f.LivePaths...)
	return imports
}

// license prepends the repository's license header, the same one every other checked-in
// file carries.
func license(ctx context.Context, repoRoot, content string) (string, error) {
	l, err := format.NewLicense(repoRoot)
	if err != nil {
		return "", err
	}
	out, err := l.Format(ctx, []byte(content), "v.oracle")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
