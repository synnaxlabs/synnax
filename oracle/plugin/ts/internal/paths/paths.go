// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package paths resolves TypeScript module specifiers between generator output
// directories. Two output paths in the same npm workspace produce an internal
// alias import (e.g. @/foo/bar); paths in different workspaces produce a
// package-name import (e.g. @synnaxlabs/x); paths outside any known workspace
// fall back to a relative import.
package paths

import (
	"path/filepath"
	"strings"
)

// PackageMapping describes one workspace package known to the generator.
type PackageMapping struct {
	// PathPrefix is the source-tree prefix for files in this package
	// (e.g. "client/ts/src").
	PathPrefix string
	// PackageName is the npm package name (e.g. "@synnaxlabs/client").
	PackageName string
	// InternalPrefix is the in-package import alias prefix (e.g. "@/").
	InternalPrefix string
}

// KnownPackages enumerates the workspace packages the generator can target.
// It is intentionally exported so callers can extend or override the list in
// tests; production code should treat it as constant.
var KnownPackages = []PackageMapping{
	{PathPrefix: "client/ts/src", PackageName: "@synnaxlabs/client", InternalPrefix: "@/"},
	{PathPrefix: "x/ts/src", PackageName: "@synnaxlabs/x", InternalPrefix: "@/"},
	{PathPrefix: "pluto/src", PackageName: "@synnaxlabs/pluto", InternalPrefix: "@/"},
	{PathPrefix: "freighter/ts/src", PackageName: "@synnaxlabs/freighter", InternalPrefix: "@/"},
	{PathPrefix: "alamos/ts/src", PackageName: "@synnaxlabs/alamos", InternalPrefix: "@/"},
	{PathPrefix: "drift/src", PackageName: "@synnaxlabs/drift", InternalPrefix: "@/"},
}

// FindPackage returns the workspace mapping that owns outputPath, or nil if
// outputPath is not inside any known workspace.
func FindPackage(outputPath string) *PackageMapping {
	for i := range KnownPackages {
		if strings.HasPrefix(outputPath, KnownPackages[i].PathPrefix) {
			return &KnownPackages[i]
		}
	}
	return nil
}

// CalculateImport returns the TypeScript module specifier to import toPath from
// fromPath. Same workspace yields an internal alias; different workspaces yield
// the destination workspace's npm package name; unknown paths yield a relative
// path.
func CalculateImport(fromPath, toPath string) string {
	fromPkg, toPkg := FindPackage(fromPath), FindPackage(toPath)
	if fromPkg == nil || toPkg == nil {
		return calculateRelative(fromPath, toPath)
	}
	if fromPkg.PackageName == toPkg.PackageName {
		relativePath := strings.TrimPrefix(strings.TrimPrefix(toPath, toPkg.PathPrefix), "/")
		return toPkg.InternalPrefix + relativePath
	}
	return toPkg.PackageName
}

func calculateRelative(from, to string) string {
	rel, err := filepath.Rel(from, to)
	if err != nil {
		return "./" + to
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}
