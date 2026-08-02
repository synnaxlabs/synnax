// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"context"
	"strings"

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// chainPredecessors resolves the alias split from explicitly managed version
// chains. A chain's current file enumerates the split directly: its alias
// lines alias, its full declarations define. Types whose live declarations
// carry omit fields always define — the frozen counterpart is persisted-only,
// so the two can never be the same Go type.
func chainPredecessors(
	ctx context.Context, req *plugin.Request,
) (map[string]predecessor, error) {
	preds := make(map[string]predecessor)
	if req.Versions == nil {
		return preds, nil
	}
	entries, err := versioning.EntryPaths(req.Resolutions)
	if err != nil {
		return nil, err
	}
	chains := req.Versions.Chains()
	for origPath, version := range entries {
		if version == 0 {
			continue
		}
		resource, livePath, ok := pathResource(req.Resolutions, origPath)
		if !ok {
			continue
		}
		chain, ok := chains[livePath]
		if !ok {
			continue
		}
		if chain.Current() != version {
			return nil, errors.Newf(
				"%s declares @go version %d but its chain's current file is v%d",
				livePath, version, chain.Current(),
			)
		}
		f, err := req.Versions.File(ctx, livePath, version)
		if err != nil {
			return nil, err
		}
		aliased := make(set.Set[string], len(f.Aliases))
		for name := range f.Aliases {
			live, ok := req.Resolutions.Get(resource + "." + name)
			if ok && hasOmitField(live) {
				continue
			}
			aliased.Add(resource + "." + name)
		}
		preds[versioning.VersionedPath(origPath, version)] = predecessor{
			path:    versioning.VersionedPath(origPath, version-1),
			aliased: aliased,
		}
	}
	return preds, nil
}

// pathResource maps a versioned output path to its schema resource and live
// import path via any version-declaring type's source file.
func pathResource(
	table *resolution.Table, origPath string,
) (resource, livePath string, ok bool) {
	for _, t := range versioning.TypesAtPath(table, origPath) {
		if _, versioned := versioning.Version(t); !versioned {
			continue
		}
		return t.Namespace, strings.TrimSuffix(t.FilePath, ".oracle"), true
	}
	return "", "", false
}

// hasOmitField reports whether the live declaration carries any non-persisted
// field.
func hasOmitField(t resolution.Type) bool {
	form, ok := t.Form.(resolution.StructForm)
	if !ok {
		return false
	}
	return len(schemadiff.PersistedFields(form.Fields)) != len(form.Fields)
}
