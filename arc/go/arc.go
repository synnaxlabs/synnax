// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package arc is the top-level Arc compiler entry point. CompileText and
// CompileGraph orchestrate parse → analyze → compile against a caller-
// supplied root scope. Callers are responsible for building the root
// (STL symbols, cluster channels, dynamic resolvers) via symbol.NewRoot.
package arc

import (
	"context"
	"slices"

	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
)

type (
	IR             = ir.IR
	Node           = ir.Node
	Edge           = ir.Edge
	Handle         = ir.Handle
	Function       = ir.Function
	Symbol         = symbol.Symbol
	SymbolResolver = symbol.Resolver
	Graph          = graph.Graph
	Text           = text.Text
	Program        = program.Program
)

// Option configures Arc compilation.
type Option func(*options)

type options struct {
	cfg parser.Config
}

func newOptions(opts []Option) options {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithAllowDashedNames permits '-' inside Arc channel-name identifiers (e.g.
// "sensor-1"). Off by default; enable only when Core runs with channel-name validation
// disabled.
func WithAllowDashedNames(allow bool) Option {
	return func(o *options) { o.cfg.AllowDashedNames = allow }
}

// NewRoot returns a program root with the standard library symbols and
// any extras attached to the ambient prelude, and resolver installed as
// the GlobalResolver. This is the canonical entry point for production
// compilation: callers pass any additional static symbols (e.g. status
// types) as extras alongside the resolver. STL symbols are freshly
// allocated per call so no analysis can corrupt another's view of the
// standard library.
func NewRoot(resolver SymbolResolver, extras ...*symbol.Symbol) *symbol.Symbol {
	return symbol.NewRoot(resolver, slices.Concat(stl.NewSymbols(), extras))
}

// CompileGraph parses, analyzes, and compiles a graph-mode program
// against root. root must have its ambient prelude populated by the
// caller. Graph mode auto-imports modules; callers do not need to call
// symbol.AutoImportModules themselves — CompileGraph does it.
func CompileGraph(
	ctx context.Context,
	g Graph,
	root *symbol.Symbol,
	opts ...Option,
) (Program, error) {
	o := newOptions(opts)
	graphWithAST, err := graph.Parse(g, o.cfg)
	if err != nil {
		return Program{}, err
	}
	symbol.AutoImportModules(root)
	inter, diagnostics := graph.Analyze(ctx, graphWithAST, root, o.cfg)
	if !diagnostics.Ok() {
		return Program{}, diagnostics
	}
	output, cErr := compiler.Compile(ctx, inter, compiler.WithConfig(o.cfg))
	if cErr != nil {
		return Program{}, cErr
	}
	return Program{IR: inter, Output: output}, nil
}

// CompileText parses, analyzes, and compiles a text-mode program against
// root. root must have its ambient prelude populated by the caller.
func CompileText(
	ctx context.Context,
	t Text,
	root *symbol.Symbol,
	opts ...Option,
) (Program, error) {
	o := newOptions(opts)
	textWithAST, err := text.Parse(t, o.cfg)
	if err != nil {
		return Program{}, err
	}
	inter, diagnostics := text.Analyze(ctx, textWithAST, root, o.cfg)
	if !diagnostics.Ok() {
		return Program{}, diagnostics
	}
	output, cErr := compiler.Compile(ctx, inter, compiler.WithConfig(o.cfg))
	if cErr != nil {
		return Program{}, cErr
	}
	return Program{IR: inter, Output: output}, nil
}
