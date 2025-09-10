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
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
)

type (
	IR             = ir.IR
	Stage          = ir.Stage
	Node           = ir.Node
	Edge           = ir.Edge
	Function       = ir.Function
	SymbolResolver = ir.SymbolResolver
	Symbol         = ir.Symbol
	Graph          = graph.Graph
	Text           = text.Text
)

type Module struct {
	ir.IR
	WASM []byte
}

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

func CompileText(t Text, opts ...Option) (Module, error) {
	o := newOptions(opts)
	textWithAST, err := text.Parse(t)
	if err != nil {
		return Module{}, err
	}
	inter, diagnostics := text.Analyze(textWithAST, o.resolver)
	if !diagnostics.Ok() {
		return Module{}, diagnostics.Error()
	}
	wasmBytes, err := compiler.Compile(inter)
	if err != nil {
		return Module{}, err
	}
	return Module{WASM: wasmBytes, IR: inter}, nil
}

func CompileGraph(g Graph, opts ...Option) (Module, error) {
	o := newOptions(opts)
	graphWithAST, err := graph.Parse(g)
	if err != nil {
		return Module{}, err
	}
	inter, diagnostics := graph.Analyze(graphWithAST, o.resolver)
	if !diagnostics.Ok() {
		return Module{}, diagnostics.Error()
	}
	wasmBytes, err := compiler.Compile(inter)
	if err != nil {
		return Module{}, err
	}
	return Module{WASM: wasmBytes, IR: inter}, nil
}

func ConvertTextToGraph(text Text, opts ...Option) (Graph, error) {
	return Graph{}, nil
}

func ConvertGraphToText(graph Graph, opts ...Option) (Text, error) {
	return Text{}, nil
}
