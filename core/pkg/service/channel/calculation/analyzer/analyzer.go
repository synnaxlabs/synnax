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
	"strconv"

	"github.com/synnaxlabs/arc"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
)

type resolver struct {
	arc.SymbolResolver
	temp struct {
		names map[string]*symbol.Symbol
		keys  map[int]*symbol.Symbol
	}
}

type Analyzer struct {
	resolver *resolver
}

func New(symbolResolver arc.SymbolResolver) *Analyzer {
	r := &resolver{SymbolResolver: symbolResolver}
	r.temp.keys = make(map[int]*symbol.Symbol)
	r.temp.names = make(map[string]*symbol.Symbol)
	return &Analyzer{resolver: r}
}

func (r *resolver) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	i, err := strconv.Atoi(name)
	if err == nil {
		if s, ok := r.temp.keys[i]; ok {
			return *s, nil
		}
	} else {
		if s, ok := r.temp.names[name]; ok {
			return *s, nil
		}
	}
	return r.SymbolResolver.Resolve(ctx, name)
}

func (a *Analyzer) Analyze(ctx context.Context, ch channel.Channel) (telem.DataType, error) {
	t, err := parser.ParseBlock(fmt.Sprintf("{%s}", ch.Expression))
	if err != nil {
		return telem.UnknownT, err
	}
	aCtx := acontext.CreateRoot(ctx, t, a.resolver)
	dataType := statement.AnalyzeFunctionBody(aCtx)
	if !aCtx.Diagnostics.Ok() {
		return telem.UnknownT, aCtx.Diagnostics
	}
	s := &symbol.Symbol{
		Name: ch.Name,
		Kind: symbol.KindChannel,
		Type: types.Chan(dataType),
	}
	if ch.Key() != 0 {
		intKey := int(ch.Key())
		s.ID = intKey
		a.resolver.temp.keys[intKey] = s
	}
	a.resolver.temp.names[s.Name] = s
	return types.ToTelem(dataType), nil
}
