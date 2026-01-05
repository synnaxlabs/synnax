// Copyright 2025 Synnax Labs, Inc.
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

type PbFormatter struct{}

func (f *PbFormatter) FormatQualified(qualifier, typeName string) string {
	if qualifier == "" {
		return typeName
	}
	return fmt.Sprintf("%s.%s", qualifier, typeName)
}

func (f *PbFormatter) FormatGeneric(baseName string, typeArgs []string) string {
	return baseName
}

func (f *PbFormatter) FormatArray(elemType string) string {
	return elemType
}

func (f *PbFormatter) FormatMap(keyType, valType string) string {
	return fmt.Sprintf("map<%s, %s>", keyType, valType)
}

func (f *PbFormatter) FallbackType() string {
	return "bytes"
}

type PbImportResolver struct{}

func (r *PbImportResolver) ResolveImport(outputPath string, ctx *resolver.Context) (importPath string, qualifier string, shouldImport bool) {
	protoPath := outputPath + "/types.gen.proto"
	parts := strings.Split(outputPath, "/")
	qualifier = parts[len(parts)-1]
	return protoPath, qualifier, true
}
