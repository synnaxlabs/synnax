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
	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/spatial"
)

type (
	Stage = ir.Stage
	Edge  = ir.Edge
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
	Viewport Viewport `json:"viewport"`
	Stages   []Stage  `json:"stages"`
	Edges    []Edge   `json:"edges"`
	Nodes    []Node   `json:"nodes"`
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
	return g, nil
}

func Analyze(
	g Graph,
	resolver ir.SymbolResolver,
) (ir.IR, analyzer.Diagnostics) {
	ctx := context.CreateRoot[antlr.ParserRuleContext](nil, resolver)
	// Step 1: Build the root context.
	for _, stage := range g.Stages {
		if _, err := ctx.Scope.Add(ir.Symbol{
			Name:       stage.Key,
			Kind:       ir.KindStage,
			Type:       stage,
			ParserRule: stage.Body.AST,
		}); err != nil {
			ctx.Diagnostics.AddError(err, stage.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
	}

	// Step 2: Analyze stage bodies
	for _, stage := range g.Stages {
		stageScope, err := ctx.Scope.GetChildByParserRule(stage.Body.AST)
		if err != nil {
			ctx.Diagnostics.AddError(err, stage.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if !analyzer.AnalyzeBlock(context.ChildWithScope(ctx, stage.Body.AST, stageScope)) {
			return ir.IR{}, *ctx.Diagnostics
		}
	}
	// Step 3: Analyze flow statement relationships

	// Step 4: Return the IR
	return ir.IR{
		Symbols: ctx.Scope,
		Stages:  g.Stages,
		Edges:   g.Edges,
		Nodes: lo.Map(g.Nodes, func(item Node, _ int) ir.Node {
			return item.Node
		}),
	}, *ctx.Diagnostics
}
