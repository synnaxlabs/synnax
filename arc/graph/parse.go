// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import "github.com/synnaxlabs/arc/parser"

func Parse(g Graph) (Graph, error) {
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
