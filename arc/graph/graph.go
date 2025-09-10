// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/spatial"
)

type (
	Stage    = ir.Stage
	Edge     = ir.Edge
	Function = ir.Function
)

type Node struct {
	ir.Node
	Position spatial.XY `json:"position"`
}

type Viewport struct {
	Position spatial.XY `json:"position"`
	Zoom     float32    `json:"zoom"`
}

type Graph struct {
	Viewport  Viewport   `json:"viewport"`
	Stages    []Stage    `json:"stages"`
	Functions []Function `json:"functions"`
	Edges     []Edge     `json:"edges"`
	Nodes     []Node     `json:"nodes"`
}

func Parse(g Graph) (Graph, error) {
	for i, stage := range g.Stages {
		ast, err := parser.ParseBlock(stage.Body.Raw)
		if err != nil {
			return Graph{}, err
		}
		stage.Body.AST = ast
		g.Stages[i] = stage
	}
	for i, function := range g.Functions {
		ast, err := parser.ParseBlock(function.Body.Raw)
		if err != nil {
			return Graph{}, err
		}
		function.Body.AST = ast
		g.Functions[i] = function
	}
	return g, nil
}

func bindNamedTypes(ctx context.Context, s *ir.Scope, t ir.NamedTypes, kind ir.Kind) error {
	for k, ty := range t.Iter() {
		if _, err := s.Add(ctx, ir.Symbol{
			Name: k,
			Kind: kind,
			Type: ty,
		}); err != nil {
			return err
		}
	}
	return nil
}

func Analyze(
	ctx_ context.Context,
	g Graph,
	resolver ir.SymbolResolver,
) (ir.IR, analyzer.Diagnostics) {
	ctx := acontext.CreateRoot[antlr.ParserRuleContext](ctx_, nil, resolver)
	// Step 1: Build the root context.
	for _, stage := range g.Stages {
		stageScope, err := ctx.Scope.Add(ctx, ir.Symbol{
			Name:       stage.Key,
			Kind:       ir.KindStage,
			Type:       stage,
			ParserRule: stage.Body.AST,
		})
		if err != nil {
			ctx.Diagnostics.AddError(err, stage.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if err = bindNamedTypes(ctx, stageScope, stage.Config, ir.KindConfigParam); err != nil {
			ctx.Diagnostics.AddError(err, stage.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if err = bindNamedTypes(ctx, stageScope, stage.Params, ir.KindParam); err != nil {
			ctx.Diagnostics.AddError(err, stage.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
	}
	for _, stage := range g.Functions {
		funcScope, err := ctx.Scope.Add(ctx, ir.Symbol{
			Name:       stage.Key,
			Kind:       ir.KindFunction,
			Type:       stage,
			ParserRule: stage.Body.AST,
		})
		if err != nil {
			ctx.Diagnostics.AddError(err, stage.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if err = bindNamedTypes(ctx, funcScope, stage.Params, ir.KindParam); err != nil {
			ctx.Diagnostics.AddError(err, stage.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
	}

	// Step 2: Analyze stage & function bodies
	for _, stage := range g.Stages {
		stageScope, err := ctx.Scope.GetChildByParserRule(stage.Body.AST)
		if err != nil {
			ctx.Diagnostics.AddError(err, stage.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if !analyzer.AnalyzeBlock(acontext.Child(ctx, stage.Body.AST).WithScope(stageScope)) {
			return ir.IR{}, *ctx.Diagnostics
		}
	}
	for _, fn := range g.Functions {
		funcScope, err := ctx.Scope.GetChildByParserRule(fn.Body.AST)
		if err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if !analyzer.AnalyzeBlock(acontext.Child(ctx, fn.Body.AST).WithScope(funcScope)) {
			return ir.IR{}, *ctx.Diagnostics
		}
	}
	// Step 3: Analyze flow statement relationships

	// Step 4: Return the IR
	return ir.IR{
		Symbols:   ctx.Scope,
		Stages:    g.Stages,
		Edges:     g.Edges,
		Functions: g.Functions,
		Nodes: lo.Map(g.Nodes, func(item Node, _ int) ir.Node {
			return item.Node
		}),
	}, *ctx.Diagnostics
}
