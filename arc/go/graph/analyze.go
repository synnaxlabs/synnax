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
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/zyn"
)

// Analyze compiles a visual graph into executable IR with type inference,
// edge validation, and stratified execution planning. Errors are collected
// in the returned Diagnostics.
func Analyze(
	ctx_ context.Context,
	g Graph,
	resolver symbol.Resolver,
) (ir.IR, *diagnostics.Diagnostics) {
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
			return ir.IR{}, ctx.Diagnostics
		}
		if err = bindParams(ctx, funcScope, fn.Config, symbol.KindConfig); err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, ctx.Diagnostics
		}
		if err = bindParams(ctx, funcScope, fn.Inputs, symbol.KindInput); err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, ctx.Diagnostics
		}
		if err = bindParams(ctx, funcScope, fn.Outputs, symbol.KindOutput); err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, ctx.Diagnostics
		}
	}

	// Step 2: Analyze Function Bodies
	for i, fn := range g.Functions {
		funcScope, err := ctx.Scope.GetChildByParserRule(fn.Body.AST)
		if err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, ctx.Diagnostics
		}
		funcScope.AccumulateReadChannels()
		if fn.Body.Raw != "" {
			blockCtx, ok := fn.Body.AST.(parser.IBlockContext)
			if !ok {
				ctx.Diagnostics.AddError(errors.New("function body must be a block"), fn.Body.AST)
				return ir.IR{}, ctx.Diagnostics
			}
			if !analyzer.AnalyzeBlock(acontext.Child(ctx, blockCtx).WithScope(funcScope)) {
				return ir.IR{}, ctx.Diagnostics
			}
		}
		fn.Channels = funcScope.Channels
		g.Functions[i] = fn
	}

	// Step 3 & 4: Create Fresh Types and IR Nodes
	freshFuncTypes := make(map[string]types.Type)
	irNodes := make(ir.Nodes, len(g.Nodes))
	for i, n := range g.Nodes {
		fnSym, err := ctx.Scope.Resolve(ctx, n.Type)
		if err != nil {
			ctx.Diagnostics.AddError(err, nil)
			return ir.IR{}, ctx.Diagnostics
		}
		freshFuncTypes[n.Key] = types.Freshen(fnSym.Type, n.Key)
		freshType := freshFuncTypes[n.Key]
		node := ir.Node{
			Key:      n.Key,
			Type:     n.Type,
			Channels: fnSym.Channels.Copy(),
			Config:   freshType.Config,
			Inputs:   freshType.Inputs,
			Outputs:  freshType.Outputs,
		}
		// Process provided config values
		for j, configParam := range freshType.Config {
			configValue, ok := n.Config[configParam.Name]
			if !ok {
				continue
			}
			if configParam.Type.Kind == types.KindChan {
				var k uint32
				if err = zyn.Uint32().Coerce().Parse(configValue, &k); err != nil {
					return ir.IR{}, ctx.Diagnostics
				}
				channelSym, err := resolver.Resolve(ctx_, strconv.Itoa(int(k)))
				if err == nil && channelSym.Type.Kind == types.KindChan {
					if err = atypes.Check(
						ctx.Constraints,
						channelSym.Type,
						configParam.Type,
						nil,
						"",
					); err != nil {
						ctx.Diagnostics.AddError(err, nil)
						return ir.IR{}, ctx.Diagnostics
					}
					node.Channels.Read.Add(k)
				}
			}
			node.Config[j].Value = configValue
		}
		irNodes[i] = node

		// Validate all required config parameters are provided
		for _, configParam := range freshType.Config {
			if configParam.Value == nil {
				ctx.Diagnostics.AddError(
					errors.Wrapf(
						query.NotFound,
						"node '%s' (%s) missing required config parameter '%s'",
						n.Key,
						n.Type,
						configParam.Name,
					), nil)
			}
		}
	}

	// Step 5: Check Types Across Edges
	for _, edge := range g.Edges {
		if !validateEdge(ctx, edge, g.Nodes, freshFuncTypes) {
			return ir.IR{}, ctx.Diagnostics
		}
	}

	// Step 5A: Check for Duplicate Edge Targets and Build Connected Inputs Map
	connectedInputs := make(map[string]map[string]bool)
	for _, edge := range g.Edges {
		if connectedInputs[edge.Target.Node] == nil {
			connectedInputs[edge.Target.Node] = make(map[string]bool)
		}
		if connectedInputs[edge.Target.Node][edge.Target.Param] {
			ctx.Diagnostics.AddError(
				errors.Newf(
					"multiple edges target node '%s' parameter '%s'",
					edge.Target.Node,
					edge.Target.Param,
				), nil)
		}
		connectedInputs[edge.Target.Node][edge.Target.Param] = true
	}
	if !ctx.Diagnostics.Ok() {
		return ir.IR{}, ctx.Diagnostics
	}

	// Step 5B: Check Missing Required Inputs
	for _, n := range g.Nodes {
		freshType := freshFuncTypes[n.Key]
		if freshType.Inputs == nil {
			continue
		}
		connected := connectedInputs[n.Key]
		for _, inputParam := range freshType.Inputs {
			if !connected[inputParam.Name] {
				// Check if this parameter has a default value (is optional)
				if inputParam.Value == nil {
					// Required parameter is missing
					ctx.Diagnostics.AddError(
						errors.Wrapf(
							query.NotFound,
							"node '%s' (%s) missing required input '%s'",
							n.Key,
							n.Type,
							inputParam.Name,
						), nil)
				}
			}
		}
	}
	if !ctx.Diagnostics.Ok() {
		return ir.IR{}, ctx.Diagnostics
	}

	// Step 6: Unify Type Constraints
	if err := ctx.Constraints.Unify(); err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return ir.IR{}, ctx.Diagnostics
	}

	// Step 7: Build IR Nodes with Unified Type Constraints
	for i, n := range g.Nodes {
		substituted := ctx.Constraints.ApplySubstitutions(freshFuncTypes[n.Key])
		irN := irNodes[i]
		irN.Outputs = substituted.Outputs
		irN.Inputs = substituted.Inputs
		irN.Config = substituted.Config
		irNodes[i] = irN
	}

	// Step 8: Build Stratified Execution Plan
	// Graph-based compilation doesn't support sequences, so pass nil
	strata, err := stratifier.Stratify(ctx, irNodes, g.Edges, nil, ctx.Diagnostics)
	if err != nil {
		return ir.IR{}, ctx.Diagnostics
	}

	// Step 9: Substitute TypeMap after unification
	for node, typ := range ctx.TypeMap {
		ctx.TypeMap[node] = ctx.Constraints.ApplySubstitutions(typ)
	}

	// Step 10: Return IR
	return ir.IR{
		Functions: g.Functions,
		Edges:     g.Edges,
		Nodes:     irNodes,
		Symbols:   ctx.Scope,
		Strata:    strata,
		TypeMap:   ctx.TypeMap,
	}, ctx.Diagnostics
}

// bindParams adds function parameters to the symbol scope with the specified kind.
// Used internally to bind Config, Input, and Output parameters during function
// registration.
func bindParams(
	ctx context.Context,
	scope *symbol.Scope,
	params types.Params,
	kind symbol.Kind,
) error {
	for _, p := range params {
		if _, err := scope.Add(ctx, symbol.Symbol{
			Name:         p.Name,
			Kind:         kind,
			Type:         p.Type,
			DefaultValue: p.Value,
		}); err != nil {
			return err
		}
	}
	return nil
}

// validateEdge checks type compatibility between an edge's source and target.
// Returns true if validation succeeds, false if an error was added to diagnostics.
func validateEdge(
	ctx acontext.Context[antlr.ParserRuleContext],
	edge Edge,
	nodes Nodes,
	freshFuncTypes map[string]types.Type,
) bool {
	sourceNode, ok := nodes.Find(edge.Source.Node)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Wrapf(query.NotFound, "edge source node '%s' not found", edge.Source.Node),
			nil,
		)
		return false
	}
	targetNode, ok := nodes.Find(edge.Target.Node)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Wrapf(query.NotFound, "edge target node '%s' not found", edge.Target.Node),
			nil,
		)
		return false
	}

	sourceFunc := freshFuncTypes[sourceNode.Key]
	sourceParam, ok := sourceFunc.Outputs.Get(edge.Source.Param)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Wrapf(
				query.NotFound,
				"output '%s' not found in node '%s' (%s)",
				edge.Source.Param,
				edge.Source.Node,
				sourceNode.Type,
			), nil)
		return false
	}

	targetFunc := freshFuncTypes[targetNode.Key]
	targetParam, ok := targetFunc.Inputs.Get(edge.Target.Param)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Wrapf(
				query.NotFound,
				"input '%s' not found in node '%s' (%s)",
				edge.Target.Param,
				edge.Target.Node,
				targetNode.Type,
			), nil)
		return false
	}

	if err := atypes.Check(
		ctx.Constraints,
		sourceParam.Type,
		targetParam.Type,
		nil,
		"",
	); err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return false
	}
	return true
}
