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
	var files []plugin.File
	for _, origPath := range slices.Sorted(maps.Keys(chainPaths)) {
		cp := chainPaths[origPath]
		for _, k := range cp.Numbers {
			if k >= cp.Current {
				break
			}
			out, err := p.frozenCodecFiles(req, origPath, cp.LivePath, k)
			if err != nil {
				return nil, err
			}
			files = append(files, out...)
		}
	}
	return files, nil
}

// frozenEntries returns the struct and union types in namespace ns of a frozen table
// that get a codec, keyed by qualified name. isAlias reports whether the version file
// aliases a type to its predecessor, which then carries its codec.
func frozenEntries(
	table *resolution.Table,
	ns string,
	isAlias func(string) bool,
) map[string]resolution.Type {
	merged := make(map[string]resolution.Type)
	for _, entry := range append(table.StructTypes(), table.UnionTypes()...) {
		if entry.Namespace != ns ||
			!domain.HasExprFromType(entry, "go", "marshal") ||
			handCodec(entry) {
			continue
		}
		if isAlias(entry.Name) {
			continue
		}
		if entry.Synthetic {
			continue
		}
		merged[entry.QualifiedName] = entry
	}
	return merged
}

// frozenFlexCodecs returns the flex codecs of the distinct types in namespace ns of a
// frozen table.
func frozenFlexCodecs(
	table *resolution.Table,
	ns string,
	isAlias func(string) bool,
) []flexCodec {
	var flex []flexCodec
	for _, dt := range table.DistinctTypes() {
		if dt.Namespace != ns {
			continue
		}
		if isAlias(dt.Name) {
			continue
		}
		if domain.GetStringFromType(dt, "go", "marshal") != "flex" {
			continue
		}
		form := resolution.MustForm[resolution.DistinctForm](dt)
		goName := naming.GetGoName(dt)
		flex = append(flex, flexCodec{
			GoName:   goName,
			Receiver: receiverName(goName),
			BaseType: form.Base.Name,
		})
	}
	return flex
}

// frozenCodecFiles emits the codec (and test) files for one frozen version package,
// mirroring the current-package flow over the frozen table.
func (p *Plugin) frozenCodecFiles(
	req *plugin.Request,
	origPath, livePath string,
	k int,
) ([]plugin.File, error) {
	f, err := req.Versions.File(livePath, k)
	if err != nil {
		return nil, err
	}
	table, ns, err := gotypes.ChainFrozenTable(req, origPath, livePath, k)
	if err != nil {
		return nil, err
	}
	vkPath := versioning.VersionedPath(origPath, k)
	isAlias := func(name string) bool {
		_, ok := f.Aliases[name]
		return ok
	}
	merged := frozenEntries(table, ns, isAlias)
	flex := frozenFlexCodecs(table, ns, isAlias)
	entries := buildCodecEntries(merged)
	if len(entries) == 0 && len(flex) == 0 {
		return nil, nil
	}
	for _, entry := range entries {
		if err := validateEntryRefs(entry.Type, table); err != nil {
			return nil, errors.Wrapf(err, "frozen codec for %s", vkPath)
		}
	}
	packageName := naming.DerivePackageName(vkPath)
	content, err := generateEncoderCodecFile(
		packageName, vkPath, entries, flex, table, req.RepoRoot, versioning.Pins{},
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
			packageName, vkPath, entries, table, req.RepoRoot, versioning.Pins{},
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
