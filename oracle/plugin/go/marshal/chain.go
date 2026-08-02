// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package marshal

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// chainFrozenCodecs regenerates codec files for every frozen version package
// of every chain. Types aliased at a version carry the definer's codec through
// the alias, so only that version's defined declarations get codecs.
func (p *Plugin) chainFrozenCodecs(req *plugin.Request) ([]plugin.File, error) {
	chainPaths, err := gotypes.ChainPaths(req)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	var files []plugin.File
	for _, origPath := range slices.Sorted(maps.Keys(chainPaths)) {
		cp := chainPaths[origPath]
		for k := range cp.Current {
			out, err := p.frozenCodecFiles(ctx, req, origPath, cp.LivePath, k)
			if err != nil {
				return nil, err
			}
			files = append(files, out...)
		}
	}
	return files, nil
}

// frozenCodecFiles emits the codec (and test) files for one frozen version
// package, mirroring the current-package flow over the frozen table.
func (p *Plugin) frozenCodecFiles(
	ctx context.Context,
	req *plugin.Request,
	origPath, livePath string,
	k int,
) ([]plugin.File, error) {
	f, err := req.Versions.File(ctx, livePath, k)
	if err != nil {
		return nil, err
	}
	table, ns, err := gotypes.ChainFrozenTable(ctx, req, origPath, livePath, k)
	if err != nil {
		return nil, err
	}
	vkPath := versioning.VersionedPath(origPath, k)
	merged := make(map[string]resolution.Type)
	for _, entry := range table.StructTypes() {
		if entry.Namespace != ns ||
			!domain.HasExprFromType(entry, "go", "marshal") {
			continue
		}
		byPkg, _ := collectSerializableTypes(entry, table)
		for _, t := range byPkg[vkPath] {
			if _, isAlias := f.Aliases[t.Name]; isAlias {
				continue
			}
			merged[t.QualifiedName] = t
		}
	}
	var flex []FlexCodec
	for _, dt := range table.DistinctTypes() {
		if dt.Namespace != ns {
			continue
		}
		if _, isAlias := f.Aliases[dt.Name]; isAlias {
			continue
		}
		if domain.GetStringFromType(dt, "go", "marshal") != "flex" {
			continue
		}
		form := dt.Form.(resolution.DistinctForm)
		goName := naming.GetGoName(dt)
		flex = append(flex, FlexCodec{
			GoName:   goName,
			Receiver: ReceiverName(goName),
			BaseType: form.Base.Name,
		})
	}
	entries := buildCodecEntries(merged)
	if len(entries) == 0 && len(flex) == 0 {
		return nil, nil
	}
	packageName := naming.DerivePackageName(vkPath)
	content, err := generateEncoderCodecFile(
		packageName, vkPath, entries, flex, table, req.RepoRoot,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "frozen codec for %s", vkPath)
	}
	files := []plugin.File{{
		Path:    fmt.Sprintf("%s/%s", vkPath, p.Options.FileNamePattern),
		Content: content,
	}}
	if p.Options.GenerateTests && len(entries) > 0 {
		testContent, err := generateTestCodecFile(
			packageName, vkPath, entries, table, req.RepoRoot,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "frozen codec tests for %s", vkPath)
		}
		if testContent != nil {
			files = append(files, plugin.File{
				Path: fmt.Sprintf(
					"%s/%s", vkPath, p.Options.TestFileNamePattern,
				),
				Content: testContent,
			})
		}
	}
	return files, nil
}
