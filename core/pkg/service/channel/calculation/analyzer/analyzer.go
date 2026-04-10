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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

type resolver struct {
	arc.SymbolResolver
	temp struct {
		names map[string]*symbol.Symbol
		keys  map[int]*symbol.Symbol
	}
	unresolved set.Set[string]
}

// Analyzer parses and type-checks calculated channel expressions. It caches
// previously analyzed channels so that later expressions can reference them by
// name without hitting the backing symbol resolver.
type Analyzer struct {
	resolver *resolver
}

// New returns an Analyzer that falls back to symbolResolver for symbols not yet
// in the internal cache.
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
	sym, resolveErr := r.SymbolResolver.Resolve(ctx, name)
	if errors.Is(resolveErr, query.ErrNotFound) {
		r.unresolved.Add(name)
	}
	return sym, resolveErr
}

// Result holds the output of an analysis. On error, Unresolved may be populated
// even though ChanDataType and Deps are zero-valued.
type Result struct {
	// ChanDataType is the inferred type of the expression when combined
	// with operations. For example, a derivative operation will convert the
	// channels return type to Float64.
	ChanDataType telem.DataType
	// ExpressionReturnType is the inferred type of the calculated expression itself.
	ExpressionReturnType types.Type
	// Deps lists the keys of channels read by the expression.
	Deps channel.Keys
	// Unresolved lists symbol names that could not be resolved during analysis.
	Unresolved []string
}

// Analyze parses the channel's expression, infers its return type, and extracts
// the set of channel dependencies. The analyzed channel is cached so that
// subsequent calls can reference it by name or key.
func (a *Analyzer) Analyze(ctx context.Context, ch channel.Channel) (Result, error) {
	a.resolver.unresolved = make(set.Set[string])
	t, err := parser.ParseBlock(fmt.Sprintf("{%s}", ch.Expression))
	if err != nil {
		return Result{}, err
	}
	aCtx := acontext.CreateRoot(ctx, t, a.resolver)
	dataType := statement.AnalyzeFunctionBody(aCtx)
	if !aCtx.Diagnostics.Ok() {
		return Result{Unresolved: a.resolver.unresolved.Slice()}, aCtx.Diagnostics
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
	var deps channel.Keys
	funcScope, scopeErr := aCtx.Scope.GetChildByParserRule(t)
	if scopeErr == nil {
		for k := range funcScope.Channels.Read {
			deps = append(deps, channel.Key(k))
		}
	}
	inferredDataType := types.ToTelem(dataType)
	if len(ch.Operations) > 0 &&
		ch.Operations[len(ch.Operations)-1].Type == channel.OperationTypeDerivative {
		inferredDataType = telem.Float64T
	}
	return Result{
		ChanDataType:         inferredDataType,
		Deps:                 deps,
		ExpressionReturnType: dataType,
	}, nil
}
