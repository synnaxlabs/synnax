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
	"fmt"
	"strings"

	gointernal "github.com/synnaxlabs/oracle/plugin/go/internal"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/resolver"
)

type GoFormatter struct{}

func (f *GoFormatter) FormatQualified(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return fmt.Sprintf("%s.%s", qualifier, typeName)
}

func (f *GoFormatter) FormatGeneric(baseName string, typeArgs []string) string {
	if len(typeArgs) == 0 {
		return baseName
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(typeArgs, ", "))
}

func (f *GoFormatter) FormatArray(elemType string) string {
	return "[]" + elemType
}

func (f *GoFormatter) FormatFixedArray(elemType string, size int64) string {
	return fmt.Sprintf("[%d]%s", size, elemType)
}

func (f *GoFormatter) FormatMap(keyType, valType string) string {
	return fmt.Sprintf("map[%s]%s", keyType, valType)
}

func (f *GoFormatter) FallbackType() string {
	return "any"
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
