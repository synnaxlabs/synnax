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

	"github.com/synnaxlabs/oracle/plugin/resolver"
)

// CppFormatter is a TypeFormatter configured for C++ type formatting.
var CppFormatter = resolver.NewFormatter(resolver.CppFormatterConfig)

type CppImportResolver struct {
	FilePattern string
}

func (r *CppImportResolver) ResolveImport(outputPath string, ctx *resolver.Context) (importPath string, qualifier string, shouldImport bool) {
	includePath := fmt.Sprintf("%s/%s", outputPath, r.FilePattern)
	return includePath, "", true
}
