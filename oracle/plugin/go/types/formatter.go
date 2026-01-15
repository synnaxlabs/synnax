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
	gointernal "github.com/synnaxlabs/oracle/plugin/go/internal"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/resolver"
)

// GoFormatter returns a TypeFormatter configured for Go type formatting.
func GoFormatter() resolver.TypeFormatter {
	return resolver.NewFormatter(resolver.GoFormatterConfig)
}

type GoImportResolver struct {
	RepoRoot       string
	CurrentPackage string
}

func (r *GoImportResolver) ResolveImport(outputPath string, ctx *resolver.Context) (importPath string, qualifier string, shouldImport bool) {
	alias := gointernal.DerivePackageAlias(outputPath, r.CurrentPackage)
	importPath = gomod.ResolveImportPath(outputPath, r.RepoRoot, gomod.DefaultModulePrefix)
	return importPath, alias, true
}
