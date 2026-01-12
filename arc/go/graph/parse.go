// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import (
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/parser"
)

// Parse parses the raw function bodies in the graph into AST representations.
// It skips functions with empty bodies and returns an error if parsing fails.
// This is typically the first step before calling Analyze.
func Parse(g Graph) (Graph, *diagnostics.Diagnostics) {
	for i, function := range g.Functions {
		if function.Body.Raw == "" {
			continue
		}
		ast, err := parser.ParseBlock(function.Body.Raw)
		if err != nil {
			return Graph{}, err
		}
		function.Body.AST = ast
		g.Functions[i] = function
	}
	return g, nil
}
