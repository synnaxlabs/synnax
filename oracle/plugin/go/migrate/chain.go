// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"context"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// chainFiles emits migrate.gen.go for every chain version: the auto-copy
// helpers transforming the preceding version's entries into vk, a pure function of the
// two adjacent version files. Regenerated on every sync and verified by check —
// hand edits do not survive.
func (p *Plugin) chainFiles(req *plugin.Request) ([]plugin.File, error) {
	ctx := context.Background()
	chainPaths, err := gotypes.ChainPaths(ctx, req)
	if err != nil {
		return nil, err
	}
	var files []plugin.File
	for _, origPath := range slices.Sorted(maps.Keys(chainPaths)) {
		cp := chainPaths[origPath]
		for i, k := range cp.Numbers {
			if i == 0 || k > cp.Current {
				continue
			}
			// A tombstone has no package: nothing migrates into an ended chain's final
			// version.
			if cp.Ended && k == cp.Current {
				continue
			}
			file, ok, err := chainFile(
				ctx, req, origPath, cp.LivePath, cp.Numbers[i-1], k,
			)
			if err != nil {
				return nil, err
			}
			if ok {
				files = append(files, file)
			}
		}
	}
	return files, nil
}

// chainFile emits one version's migrate.gen.go, when its entries changed.
func chainFile(
	ctx context.Context,
	req *plugin.Request,
	origPath, livePath string,
	prev, k int,
) (plugin.File, bool, error) {
	oldTable, oldNS, err := gotypes.ChainFrozenTable(
		ctx, req, origPath, livePath, prev,
	)
	if err != nil {
		return plugin.File{}, false, err
	}
	newTable, newNS, err := gotypes.ChainFrozenTable(
		ctx, req, origPath, livePath, k,
	)
	if err != nil {
		return plugin.File{}, false, err
	}
	newFile, err := req.Versions.File(ctx, livePath, k)
	if err != nil {
		return plugin.File{}, false, err
	}
	// A chain may span several Go outputs (task and task/config); this file
	// belongs to origPath, so members living in another output stay out of it.
	foreign := foreignOutputs(req, livePath, origPath)
	var oldEntries []resolution.Type
	for _, t := range oldTable.TypesInNamespace(oldNS) {
		if domain.HasExprFromType(t, "go", "migrate") && !foreign.Contains(t.Name) {
			oldEntries = append(oldEntries, t)
		}
	}
	roots := oldEntries
	if len(roots) == 0 {
		// Value-type paths have no migrate entries; their changed types still
		// need auto-copies because other resources' migrations consume them.
		for _, t := range newFile.Defined {
			if t.Synthetic || foreign.Contains(t.Name) {
				continue
			}
			if old, ok := oldTable.Get(oldNS + "." + t.Name); ok {
				roots = append(roots, old)
			}
		}
	}
	if len(roots) == 0 {
		return plugin.File{}, false, nil
	}
	slices.SortFunc(roots, func(a, b resolution.Type) int {
		return cmpStrings(a.Name, b.Name)
	})
	mapping := chainMapping{
		resolver: req.Versions, ctx: ctx, livePath: livePath, newFile: newFile,
	}
	counterpart := mapping.counterpart
	diff := make(map[string]schemadiff.TypeDiff)
	for _, oe := range roots {
		// Renamed or cross-resource-moved types have no same-name counterpart;
		// their transforms are hand-written in migrate.go.
		if ne, ok := newTable.Get(newNS + "." + oe.Name); ok {
			maps.Copy(
				diff, schemadiff.SchemaDiff(oe, ne, oldTable, newTable, counterpart),
			)
		}
	}
	if !needsAutoMigrate(roots, diff) {
		return plugin.File{}, false, nil
	}
	newPath := versioning.VersionedPath(origPath, k)
	content, err := generateAutoCopy(
		filepath.Base(newPath), newPath, req.RepoRoot,
		roots, diff, oldTable, newTable,
		mapping, wrapperNames(diff, oldTable, newTable, counterpart),
	)
	if err != nil {
		return plugin.File{}, false, errors.Wrapf(
			err, "auto-copy for %s", newPath,
		)
	}
	if content == nil {
		return plugin.File{}, false, nil
	}
	return plugin.File{Path: newPath + "/migrate.gen.go", Content: content},
		true, nil
}

// chainMapping relates the outgoing version's surface to the incoming one. Both
// surfaces materialize every name they carry in full, under their own version's
// namespace, so only the version files say whether the incoming version redeclared a
// name or aliased it back to an earlier one.
type chainMapping struct {
	resolver *versions.Resolver
	ctx      context.Context
	// livePath is the resource whose chain is being generated. Every other resource a
	// surface carries is a dependency pin.
	livePath string
	newFile  *versions.File
}

// counterpart returns the name the incoming version's surface files qualifiedName
// under. A resource or name the incoming version does not carry passes through
// unchanged.
func (m chainMapping) counterpart(qualifiedName string) string {
	lp, name, ok := m.split(qualifiedName)
	if !ok {
		return qualifiedName
	}
	v, ok := m.incomingVersion(lp)
	if !ok {
		return qualifiedName
	}
	if _, _, ok := m.resolver.Definer(m.ctx, lp, v, name); !ok {
		return qualifiedName
	}
	return versions.DepNS(lp, v) + "." + name
}

// sameGoType reports whether both versions denote qualifiedName with one Go type,
// which holds when they resolve the name to the same declaring version.
func (m chainMapping) sameGoType(qualifiedName string) (same, known bool) {
	lp, name, ok := m.split(qualifiedName)
	if !ok {
		return false, false
	}
	outgoing, ok := m.outgoingVersion(qualifiedName)
	if !ok {
		return false, false
	}
	incoming, ok := m.incomingVersion(lp)
	if !ok {
		return false, false
	}
	oldNS, _, oldOk := m.resolver.Definer(m.ctx, lp, outgoing, name)
	newNS, _, newOk := m.resolver.Definer(m.ctx, lp, incoming, name)
	if !oldOk || !newOk {
		return false, false
	}
	return oldNS == newNS, true
}

// split recovers the resource and name a versioned qualified name addresses.
func (m chainMapping) split(qualifiedName string) (livePath, name string, ok bool) {
	i := strings.LastIndexByte(qualifiedName, '.')
	if i < 0 {
		return "", "", false
	}
	livePath, _, ok = versions.ParseDepNS(qualifiedName[:i])
	return livePath, qualifiedName[i+1:], ok
}

func (m chainMapping) outgoingVersion(qualifiedName string) (int, bool) {
	i := strings.LastIndexByte(qualifiedName, '.')
	if i < 0 {
		return 0, false
	}
	_, v, ok := versions.ParseDepNS(qualifiedName[:i])
	return v, ok
}

func (m chainMapping) incomingVersion(livePath string) (int, bool) {
	if livePath == m.livePath {
		return m.newFile.N, true
	}
	v, ok := m.newFile.Pins[livePath]
	return v, ok
}

func cmpStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// foreignOutputs returns the names of livePath's chain members whose live @go
// output is a different package than origPath. A chain spanning several outputs
// (task and task/config) emits one migrate file per output; each covers only
// the types that live in it.
func foreignOutputs(
	req *plugin.Request, livePath, origPath string,
) set.Set[string] {
	foreign := make(set.Set[string])
	for _, t := range req.Resolutions.Types {
		if t.FilePath != livePath+".oracle" {
			continue
		}
		if p := output.GetPath(t, "go"); p != "" && p != origPath {
			foreign.Add(t.Name)
		}
	}
	return foreign
}
