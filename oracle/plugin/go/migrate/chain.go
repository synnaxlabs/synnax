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

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// chainFiles emits migrate.gen.go for every chain version: the auto-copy
// helpers transforming v(k-1) entries into vk, a pure function of the two
// adjacent version files. Regenerated on every sync and verified by check —
// hand edits do not survive.
func (p *Plugin) chainFiles(req *plugin.Request) ([]plugin.File, error) {
	chainPaths, err := gotypes.ChainPaths(req)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var files []plugin.File
	for _, origPath := range slices.Sorted(maps.Keys(chainPaths)) {
		cp := chainPaths[origPath]
		for k := cp.First + 1; k <= cp.Current; k++ {
			file, ok, err := chainFile(ctx, req, origPath, cp.LivePath, k)
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
	k int,
) (plugin.File, bool, error) {
	oldTable, oldNS, err := gotypes.ChainFrozenTable(
		ctx, req, origPath, livePath, k-1,
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
	var oldEntries []resolution.Type
	for _, t := range oldTable.TypesInNamespace(oldNS) {
		if domain.HasExprFromType(t, "go", "migrate") {
			oldEntries = append(oldEntries, t)
		}
	}
	if len(oldEntries) == 0 {
		return plugin.File{}, false, nil
	}
	slices.SortFunc(oldEntries, func(a, b resolution.Type) int {
		return cmpStrings(a.Name, b.Name)
	})
	diff := make(map[string]schemadiff.TypeDiff)
	for _, oe := range oldEntries {
		ne, ok := newTable.Get(newNS + "." + oe.Name)
		if !ok {
			continue
		}
		maps.Copy(diff, schemadiff.SchemaDiff(oe, ne, oldTable, newTable))
	}
	if !needsAutoMigrate(oldEntries, diff) {
		return plugin.File{}, false, nil
	}
	newPath := versioning.VersionedPath(origPath, k)
	content, err := generateAutoCopy(
		filepath.Base(newPath), newPath, req.RepoRoot,
		oldEntries, diff, oldTable, newTable,
		false, wrapperNames(diff, oldTable, newTable),
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
