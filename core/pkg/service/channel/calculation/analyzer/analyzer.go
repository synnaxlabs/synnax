// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/arc"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

type resolver struct {
	arc.SymbolResolver
	temp map[string]symbol.Symbol
}

type Analyzer struct {
	resolver *resolver
}

func New(symbolResolver arc.SymbolResolver) *Analyzer {
	return &Analyzer{
		resolver: &resolver{
			SymbolResolver: symbolResolver,
			temp: make(map[string]symbol.Symbol),
		},
	}
}

func (r *resolver) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	if s, ok := r.temp[name]; ok {
		return s, nil
	}
	return r.SymbolResolver.Resolve(ctx, name)
}

func (a *Analyzer) Analyze(ctx context.Context, name string, expr string) (telem.DataType, error) {
	t, err := parser.ParseBlock(fmt.Sprintf("{%s}", expr))
	if err != nil {
		return telem.UnknownT, err
	}
	aCtx := acontext.CreateRoot(ctx, t, a.resolver)
	dataType := statement.AnalyzeFunctionBody(aCtx)
	if !aCtx.Diagnostics.Ok() {
		return telem.UnknownT, aCtx.Diagnostics
	}
	a.resolver.temp[name] = symbol.Symbol{
		Name: name,
		Kind: symbol.KindChannel,
		Type: types.Chan(dataType),
	}
	return types.ToTelem(dataType), nil
}
