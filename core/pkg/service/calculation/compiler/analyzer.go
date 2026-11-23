// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import (
	"context"
	"fmt"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/resolver"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

type cachingResolver struct {
	arc.SymbolResolver
	temp struct {
		names map[string]*symbol.Symbol
		keys  map[int]*symbol.Symbol
		// unstored holds analyzed channels that are not stored yet. A stored channel
		// with the same name resolves first.
		unstored map[string]*symbol.Symbol
	}
	unresolved set.Set[string]
	readErr    error
}

// Analyzer parses and type-checks calculated channel expressions. It caches
// previously analyzed channels so that later expressions can reference them by name
// without hitting the backing symbol resolver. A name that a stored channel holds
// resolves to that channel before an analyzed channel that is not stored yet.
// Analyzer is not safe for concurrent use.
type Analyzer struct {
	symbols  arc.SymbolResolver
	resolver *cachingResolver
	// stl holds the Arc standard library symbols, built on the first analysis.
	// symbol.NewRoot copies them into each analysis, so later analyses reuse them.
	stl []*symbol.Symbol
}

// NewAnalyzer returns an Analyzer that falls back to symbolResolver for
// symbols not yet in the internal cache.
func NewAnalyzer(symbolResolver arc.SymbolResolver) *Analyzer {
	r := &cachingResolver{}
	r.temp.keys = make(map[int]*symbol.Symbol)
	r.temp.names = make(map[string]*symbol.Symbol)
	r.temp.unstored = make(map[string]*symbol.Symbol)
	return &Analyzer{
		symbols:  symbolResolver,
		resolver: r,
	}
}

func (r *cachingResolver) Resolve(
	ctx context.Context,
	name string,
) (*symbol.Symbol, error) {
	if s, ok := r.temp.names[name]; ok {
		return s, nil
	}
	sym, resolveErr := r.SymbolResolver.Resolve(ctx, name)
	// The analysis reports an ambiguous name as a diagnostic, like an unknown one.
	_, ambiguous := errors.AsType[channel.AmbiguousNameError](resolveErr)
	switch {
	case errors.Is(resolveErr, query.ErrNotFound):
		if s, ok := r.temp.unstored[name]; ok {
			return s, nil
		}
		r.unresolved.Add(name)
	case resolveErr != nil && !ambiguous && r.readErr == nil:
		r.readErr = resolveErr
	}
	return sym, resolveErr
}

// ResolveKey resolves the channel with key, an analyzed channel first.
func (r *cachingResolver) ResolveKey(
	ctx context.Context,
	key uint32,
) (*symbol.Symbol, error) {
	if s, ok := r.temp.keys[int(key)]; ok {
		return s, nil
	}
	return r.SymbolResolver.ResolveKey(ctx, key)
}

// newCalculationRoot returns the root that a calculation's expression resolves in: the
// channels that resolver gives and the standard library with no module that a program
// can import, since a calculation imports none.
func newCalculationRoot(resolver arc.SymbolResolver) *symbol.Symbol {
	return symbol.NewRoot(resolver, calculationSymbols())
}

func calculationSymbols() []*symbol.Symbol {
	return lo.Reject(stl.NewSymbols(), func(s *symbol.Symbol, _ int) bool {
		return s.Kind == symbol.KindModule && !s.Internal
	})
}

// AnalysisResult holds the output of a calculation analysis. On error,
// Unresolved may be populated even though ChanDataType and Deps are zero-valued.
type AnalysisResult struct {
	// ChanDataType is the inferred type of the expression when combined with
	// operations. For example, a derivative operation will convert the channels return
	// type to Float64.
	ChanDataType telem.DataType
	// ExpressionReturnType is the inferred type of the calculated expression itself.
	ExpressionReturnType types.Type
	// Deps lists the keys of channels read by the expression.
	Deps channel.Keys
	// Reads names each channel that the expression reads, by its key.
	Reads map[uint32]string
	// Quoted lists each channel path that the expression names in quoted form, once.
	Quoted []string
	// Unresolved lists symbol names that could not be resolved during analysis.
	Unresolved []string
}

// AnalyzeExpression parses expression as the calculation that writes output, infers its
// return type, and extracts the set of channel dependencies. A derivative operation
// makes the output a float64 channel. The output is cached so that later calls can
// reference it by name or key. An expression that fails analysis returns
// *diagnostics.Diagnostics. A channel read that fails returns its error.
func (a *Analyzer) AnalyzeExpression(
	ctx context.Context,
	output channel.Channel,
	expression string,
	derivative bool,
) (AnalysisResult, error) {
	a.resolver.SymbolResolver = resolver.NewConsistent(a.symbols)
	a.resolver.unresolved = make(set.Set[string])
	a.resolver.readErr = nil
	t, err := parser.ParseBlock(fmt.Sprintf("{%s}", expression))
	if err != nil {
		return AnalysisResult{}, err
	}
	if a.stl == nil {
		a.stl = calculationSymbols()
	}
	root := symbol.NewRoot(a.resolver, a.stl)
	aCtx := acontext.NewRoot(ctx, t, root)
	dataType := statement.AnalyzeFunctionBody(aCtx)
	if a.resolver.readErr != nil {
		return AnalysisResult{}, a.resolver.readErr
	}
	if !aCtx.Diagnostics.Ok() {
		return AnalysisResult{
			Deps:       reads(aCtx, t),
			Unresolved: a.resolver.unresolved.Slice(),
		}, aCtx.Diagnostics
	}
	s := &symbol.Symbol{
		Name: output.Name,
		Kind: symbol.KindChannel,
		Type: types.Chan(dataType),
	}
	if output.LocalKey == 0 {
		a.resolver.temp.unstored[s.Name] = s
	} else {
		s.ID = int(output.Key())
		a.resolver.temp.keys[s.ID] = s
		a.resolver.temp.names[s.Name] = s
	}
	readNames := readsOf(aCtx, t)
	inferredDataType := dataType.ToTelem()
	if derivative {
		inferredDataType = telem.Float64T
	}
	return AnalysisResult{
		ChanDataType:         inferredDataType,
		Deps:                 keysOf(readNames),
		Reads:                readNames,
		Quoted:               quotedPaths(expression, t),
		ExpressionReturnType: dataType,
	}, nil
}

// reads returns the keys of the channels that the analysis of block t resolved, in
// order.
func reads(
	aCtx acontext.Context[parser.IBlockContext],
	t parser.IBlockContext,
) channel.Keys {
	return keysOf(readsOf(aCtx, t))
}

// readsOf returns the channels that the analysis of block t resolved, by key.
func readsOf(
	aCtx acontext.Context[parser.IBlockContext],
	t parser.IBlockContext,
) map[uint32]string {
	funcScope, err := aCtx.Scope.GetChildByParserRule(t)
	if err != nil {
		return nil
	}
	return funcScope.Channels.Read
}

// keysOf returns the keys of names, in order.
func keysOf(names map[uint32]string) channel.Keys {
	keys := make(channel.Keys, 0, len(names))
	for k := range names {
		keys = append(keys, channel.Key(k))
	}
	slices.Sort(keys)
	return keys
}
