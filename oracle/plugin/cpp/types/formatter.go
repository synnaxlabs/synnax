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

type CppFormatter struct{}

func (f *CppFormatter) FormatQualified(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return fmt.Sprintf("%s::%s", qualifier, typeName)
}

func (f *CppFormatter) FormatGeneric(baseName string, typeArgs []string) string {
	if len(typeArgs) == 0 {
		return baseName
	}
	return fmt.Sprintf("%s<%s>", baseName, strings.Join(typeArgs, ", "))
}

func (f *CppFormatter) FormatArray(elemType string) string {
	return fmt.Sprintf("std::vector<%s>", elemType)
}

func (f *CppFormatter) FormatMap(keyType, valType string) string {
	return fmt.Sprintf("std::unordered_map<%s, %s>", keyType, valType)
}

func (f *CppFormatter) FallbackType() string {
	return "void"
}

type CppImportResolver struct {
	FilePattern string
}

func (r *CppImportResolver) ResolveImport(outputPath string, ctx *resolver.Context) (importPath string, qualifier string, shouldImport bool) {
	includePath := fmt.Sprintf("%s/%s", outputPath, r.FilePattern)
	return includePath, "", true
}
