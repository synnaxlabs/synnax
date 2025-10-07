// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/text"
)

type (
	IR             = ir.IR
	Stage          = ir.Stage
	Node           = ir.Node
	Edge           = ir.Edge
	Handle         = ir.Handle
	Function       = ir.Function
	SymbolResolver = ir.SymbolResolver
	Symbol         = ir.Symbol
	Graph          = graph.Graph
	Text           = text.Text
	Module         = module.Module
)

type options struct {
	resolver SymbolResolver
}

type Option func(*options)

func WithResolver(resolver SymbolResolver) Option {
	return func(o *options) { o.resolver = resolver }
}

func newOptions(opts []Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func CompileGraph(ctx context.Context, g Graph, opts ...Option) (Module, error) {
	o := newOptions(opts)
	graphWithAST, err := graph.Parse(g)
	if err != nil {
		return Module{}, err
	}
	inter, diagnostics := graph.Analyze(ctx, graphWithAST, o.resolver)
	if !diagnostics.Ok() {
		return Module{}, diagnostics.Error()
	}
	return Module{IR: inter}, nil
}

func ConvertTextToGraph(text Text, opts ...Option) (Graph, error) {
	return Graph{}, nil
}

func ConvertGraphToText(graph Graph, opts ...Option) (Text, error) {
	return Text{}, nil
}
