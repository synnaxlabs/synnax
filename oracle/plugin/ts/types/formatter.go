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

type TSFormatter struct{}

func (f *TSFormatter) FormatQualified(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return fmt.Sprintf("%s.%s", qualifier, typeName)
}

func (f *TSFormatter) FormatGeneric(baseName string, typeArgs []string) string {
	if len(typeArgs) == 0 {
		return baseName
	}
	return fmt.Sprintf("%s<%s>", baseName, strings.Join(typeArgs, ", "))
}

func (f *TSFormatter) FormatArray(elemType string) string {
	return elemType + "[]"
}

func (f *TSFormatter) FormatMap(keyType, valType string) string {
	return fmt.Sprintf("Record<%s, %s>", keyType, valType)
}

func (f *TSFormatter) FallbackType() string {
	return "unknown"
}

type TSImportResolver struct {
	CurrentOutputPath string
}

func (r *TSImportResolver) ResolveImport(outputPath string, ctx *resolver.Context) (importPath string, qualifier string, shouldImport bool) {
	importPath = calculateImportPath(r.CurrentOutputPath, outputPath)
	qualifier = ctx.Namespace
	return importPath, qualifier, true
}
