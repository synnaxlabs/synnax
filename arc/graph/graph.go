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
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/zyn"
)

type (
	Function = ir.Function
	Edge     = ir.Edge
	Edges    = ir.Edges
	Handle   = ir.Handle
)

type Node struct {
	Key          string         `json:"key"`
	Type         string         `json:"type"`
	ConfigValues map[string]any `json:"config_values"`
	Position     spatial.XY     `json:"position"`
}

type Nodes []Node

func (n Nodes) Get(key string) Node {
	return lo.Must(lo.Find(n, func(n Node) bool { return n.Key == key }))
}

func (n Nodes) Find(key string) (Node, bool) {
	return lo.Find(n, func(n Node) bool { return n.Key == key })
}

type Viewport struct {
	Position spatial.XY `json:"position"`
	Zoom     float32    `json:"zoom"`
}

type Graph struct {
	Viewport  Viewport   `json:"viewport"`
	Functions []Function `json:"functions"`
	Edges     Edges      `json:"edges"`
	Nodes     Nodes      `json:"nodes"`
}

func bindNamedTypes(ctx context.Context, s *symbol.Scope, t types.Params, kind symbol.Kind) error {
	for k, ty := range t.Iter() {
		if _, err := s.Add(ctx, symbol.Symbol{
			Name: k,
			Kind: kind,
			Type: ty,
		}); err != nil {
			return err
		}
	}
	return nil
}

func Parse(g Graph) (Graph, error) {
	for i, function := range g.Functions {
		if function.Body.Raw == "" {
			continue
		}
		ast, err := parser.ParseBlock(function.Body.Raw)
		if err != nil {
			return Graph{}, err
		}
		function.Body.AST = ast
		g.Functions[i] = function
	}
	return g, nil
}

func Analyze(
	ctx_ context.Context,
	g Graph,
	resolver symbol.Resolver,
) (ir.IR, analyzer.Diagnostics) {
	// Step 1: Build Root Context and Register All Functions
	ctx := acontext.CreateRoot[antlr.ParserRuleContext](ctx_, nil, resolver)
	for _, fn := range g.Functions {
		funcScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
			Name: fn.Key,
			Kind: symbol.KindFunction,
			Type: fn.Type(),
			AST:  fn.Body.AST,
		})
		if err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if err = bindNamedTypes(ctx, funcScope, fn.Config, symbol.KindConfig); err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if err = bindNamedTypes(ctx, funcScope, fn.Inputs, symbol.KindInput); err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if err = bindNamedTypes(ctx, funcScope, fn.Outputs, symbol.KindOutput); err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
	}

	// Step 2: Analyze Function Bodies
	for i, fn := range g.Functions {
		funcScope, err := ctx.Scope.GetChildByParserRule(fn.Body.AST)
		if err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		funcScope.Channels = symbol.NewChannels()
		funcScope.OnResolve = func(ctx context.Context, s *symbol.Scope) error {
			if s.Kind == symbol.KindChannel || s.Type.Kind == types.KindChan {
				funcScope.Channels.Read[uint32(s.ID)] = s.Name
			}
			return nil
		}
		if fn.Body.Raw != "" {
			blockCtx, ok := fn.Body.AST.(parser.IBlockContext)
			if !ok {
				ctx.Diagnostics.AddError(errors.New("function body must be a block"), fn.Body.AST)
				return ir.IR{}, *ctx.Diagnostics
			}
			if !analyzer.AnalyzeBlock(acontext.Child(ctx, blockCtx).WithScope(funcScope)) {
				return ir.IR{}, *ctx.Diagnostics
			}
		}
		fn.Channels = funcScope.Channels
		g.Functions[i] = fn
	}

	// Step 3: Create Fresh Types for Each Node
	freshFnTypes := make(map[string]types.Type)
	for _, n := range g.Nodes {
		fnSym, err := ctx.Scope.Resolve(ctx, n.Type)
		if err != nil {
			ctx.Diagnostics.AddError(err, nil)
			return ir.IR{}, *ctx.Diagnostics
		}
		freshFnTypes[n.Key] = ir.FreshType(fnSym.Type, n.Key)
	}

	// Step 4: Check Config Values Against Function Config Types
	irNodes := make(ir.Nodes, len(g.Nodes))
	for i, n := range g.Nodes {
		fnSym, err := ctx.Scope.Resolve(ctx, n.Type)
		if err != nil {
			ctx.Diagnostics.AddError(err, nil)
			return ir.IR{}, *ctx.Diagnostics
		}
		irNodes[i] = ir.Node{
			Key:          n.Key,
			Type:         n.Type,
			ConfigValues: n.ConfigValues,
			Channels:     fnSym.Channels.Copy(),
		}
		freshType := freshFnTypes[n.Key]
		if freshType.Config == nil {
			continue
		}
		for key, configType := range freshType.Config.Iter() {
			configValue, ok := n.ConfigValues[key]
			if !ok {
				continue
			}
			if configType.Kind == types.KindChan {
				var k uint32
				if err := zyn.Uint32().Coerce().Parse(configValue, &k); err != nil {
					return ir.IR{}, *ctx.Diagnostics
				}
				channelSym, err := resolver.Resolve(ctx_, strconv.Itoa(int(k)))
				if err == nil && channelSym.Type.Kind == types.KindChan {
					if err := atypes.Check(
						ctx.Constraints,
						channelSym.Type,
						configType,
						nil,
						"",
					); err != nil {
						ctx.Diagnostics.AddError(err, nil)
						return ir.IR{}, *ctx.Diagnostics
					}
					irNodes[i].Channels.Read.Add(k)
				}
			}
		}
	}

	// Step 5: Check Types Across Edges
	for _, edge := range g.Edges {
		sourceNode, ok := g.Nodes.Find(edge.Source.Node)
		if !ok {
			ctx.Diagnostics.AddError(
				errors.Wrapf(query.NotFound, "edge source node '%s' not found", edge.Source.Node),
				nil,
			)
			return ir.IR{}, *ctx.Diagnostics
		}
		targetNode, ok := g.Nodes.Find(edge.Target.Node)
		if !ok {
			ctx.Diagnostics.AddError(
				errors.Wrapf(query.NotFound, "edge target node '%s' not found", edge.Target.Node),
				nil,
			)
			return ir.IR{}, *ctx.Diagnostics
		}

		sourceType, ok := freshFnTypes[sourceNode.Key].Outputs.Get(edge.Source.Param)
		if !ok {
			ctx.Diagnostics.AddError(
				errors.Wrapf(
					query.NotFound,
					"output '%s' not found in node '%s'",
					edge.Source.Param,
					edge.Source.Node,
				), nil)
			return ir.IR{}, *ctx.Diagnostics
		}

		targetType, ok := freshFnTypes[targetNode.Key].Inputs.Get(edge.Target.Param)
		if !ok {
			ctx.Diagnostics.AddError(
				errors.Wrapf(
					query.NotFound,
					"input '%s' not found in node '%s'",
					edge.Target.Param,
					edge.Target.Node,
				), nil)
			return ir.IR{}, *ctx.Diagnostics
		}
		if err := atypes.Check(
			ctx.Constraints,
			sourceType,
			targetType,
			nil,
			"",
		); err != nil {
			ctx.Diagnostics.AddError(err, nil)
			return ir.IR{}, *ctx.Diagnostics
		}
	}

	if err := ctx.Constraints.Unify(); err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return ir.IR{}, *ctx.Diagnostics
	}

	// Step 6: Build IR Nodes with Unified Type Constraints
	for i, n := range g.Nodes {
		substituted := ctx.Constraints.ApplySubstitutions(freshFnTypes[n.Key])
		irN := irNodes[i]
		irN.Outputs = *substituted.Outputs
		irN.Inputs = *substituted.Inputs
		irN.Config = *substituted.Config
		irNodes[i] = irN
	}

	// Step 7: Build Stratified Execution Plan
	strata, ok := stratifier.Stratify(ctx, irNodes, g.Edges, ctx.Diagnostics)
	if !ok {
		return ir.IR{}, *ctx.Diagnostics
	}

	// Step 8: Substitute TypeMap after unification
	for node, typ := range ctx.TypeMap {
		ctx.TypeMap[node] = ctx.Constraints.ApplySubstitutions(typ)
	}

	// Step 9: Return IR
	return ir.IR{
		Functions: g.Functions,
		Edges:     g.Edges,
		Nodes:     irNodes,
		Symbols:   ctx.Scope,
		Strata:    strata,
		TypeMap:   ctx.TypeMap,
	}, *ctx.Diagnostics
}
