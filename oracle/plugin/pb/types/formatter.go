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
	"strings"

	"github.com/synnaxlabs/oracle/plugin/resolver"
)

// PbFormatter is a TypeFormatter configured for Protocol Buffers type formatting.
var PbFormatter = resolver.NewFormatter(resolver.PbFormatterConfig)

type PbImportResolver struct{}

func (r *PbImportResolver) ResolveImport(outputPath string, ctx *resolver.Context) (importPath string, qualifier string, shouldImport bool) {
	protoPath := outputPath + "/types.gen.proto"
	parts := strings.Split(outputPath, "/")
	qualifier = parts[len(parts)-1]
	return protoPath, qualifier, true
}
