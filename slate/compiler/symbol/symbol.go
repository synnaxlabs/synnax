// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import "github.com/synnaxlabs/slate/analyzer/symbol"

type Scope struct {
	GlobalResolver symbol.Resolver
	Symbol         *symbol.Symbol
	Parent         *Scope
	Children       []*Scope
	Index          uint32
}

func Build(analyzerScope *symbol.Scope) *Scope {}

func buildWithIdx(
	analyzerScope *symbol.Scope,
	idx uint32,
) *Scope {
	s := &Scope{
		GlobalResolver: analyzerScope.GlobalResolver,
		Symbol:         analyzerScope.Symbol,
	}
}
