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

	"github.com/synnaxlabs/oracle/plugin/resolver"
)

type PyFormatter struct{}

func (f *PyFormatter) FormatQualified(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return fmt.Sprintf("%s.%s", qualifier, typeName)
}

func (f *PyFormatter) FormatGeneric(baseName string, typeArgs []string) string {
	if len(typeArgs) == 0 {
		return baseName
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(typeArgs, ", "))
}

func (f *PyFormatter) FormatArray(elemType string) string {
	return fmt.Sprintf("list[%s]", elemType)
}

func (f *PyFormatter) FormatMap(keyType, valType string) string {
	return fmt.Sprintf("dict[%s, %s]", keyType, valType)
}

func (f *PyFormatter) FallbackType() string {
	return "Any"
}

type PyImportResolver struct{}

func (r *PyImportResolver) ResolveImport(outputPath string, ctx *resolver.Context) (importPath string, qualifier string, shouldImport bool) {
	modulePath := toPythonModulePath(outputPath)
	parts := strings.Split(modulePath, ".")
	if len(parts) >= 2 {
		moduleName := parts[len(parts)-1]
		return modulePath, moduleName, true
	}
	return modulePath, modulePath, true
}
