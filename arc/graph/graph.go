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
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/x/errors"
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
		if stage.Body.Raw == "" {
			continue
		}
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
		if stage.Body.Raw != "" {
			blockCtx, ok := stage.Body.AST.(parser.IBlockContext)
			if !ok {
				ctx.Diagnostics.AddError(errors.New("stage body must be a block"), stage.Body.AST)
				return ir.IR{}, *ctx.Diagnostics
			}
			if !analyzer.AnalyzeBlock(acontext.Child(ctx, blockCtx).WithScope(stageScope)) {
				return ir.IR{}, *ctx.Diagnostics
			}
		}
	}
	for _, fn := range g.Functions {
		funcScope, err := ctx.Scope.GetChildByParserRule(fn.Body.AST)
		if err != nil {
			ctx.Diagnostics.AddError(err, fn.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		blockCtx, ok := fn.Body.AST.(parser.IBlockContext)
		if !ok {
			ctx.Diagnostics.AddError(errors.New("function body must be a block"), fn.Body.AST)
			return ir.IR{}, *ctx.Diagnostics
		}
		if !analyzer.AnalyzeBlock(acontext.Child(ctx, blockCtx).WithScope(funcScope)) {
			return ir.IR{}, *ctx.Diagnostics
		}
	}
	// Step 3: Analyze node configurations
	for i, n := range g.Nodes {
		// Validate:
		// 1: Stage definition actually exists for node.
		// 2: Config parameters match their expected types.
		stageScope, err := ctx.Scope.Resolve(ctx, n.Type)
		if err != nil {
			ctx.Diagnostics.AddError(err, nil)
			return ir.IR{}, *ctx.Diagnostics
		}
		t := stageScope.Type.(ir.Stage)
		n.Channels = ir.OverrideChannels(t.Channels)
		for k, p := range n.Config {
			cfgT, ok := t.Config.Get(k)
			if !ok {
				ctx.Diagnostics.AddError(
					errors.Newf("node was provided config param %s not present in stage", k),
					nil,
				)
				return ir.IR{}, *ctx.Diagnostics
			}
			if _, ok = cfgT.(ir.Chan); ok {
				if f64, ok := p.(float64); ok {
					p = int(f64)
				}
				name := fmt.Sprintf("%v", p)
				sym, err := ctx.Scope.Resolve(ctx, name)
				if err != nil {
					ctx.Diagnostics.AddError(err, nil)
					return ir.IR{}, *ctx.Diagnostics
				}
				channelKey := uint32(sym.ID)
				n.Config[k] = channelKey
				n.Channels.Read.Add(channelKey)
			}
			g.Nodes[i] = n
		}
	}

	// Step 4: Analyze edge connections and create type constraints
	// Create fresh type variables for each node instance to avoid collisions
	nodeTypes := make(map[string]ir.Stage)
	for _, n := range g.Nodes {
		stageScope, _ := ctx.Scope.Resolve(ctx, n.Type)
		stage := stageScope.Type.(ir.Stage)
		// Instantiate the stage with fresh type variables unique to this node
		freshStage := ir.FreshStage(stage, n.Key)
		nodeTypes[n.Key] = freshStage

		// Create constraints for config channels
		// If a config parameter is a channel, we need to constrain its value type
		for configKey, configValue := range n.Config {
			cfgType, ok := freshStage.Config.Get(configKey)
			if !ok {
				continue
			}
			// Check if the config type is a channel type
			if chanType, isChan := cfgType.(ir.Chan); isChan {
				// Resolve the actual channel
				if channelKey, isUint := configValue.(uint32); isUint {
					channelName := fmt.Sprintf("%d", channelKey)
					channelSym, err := ctx.Scope.Resolve(ctx, channelName)
					if err == nil {
						// Create constraint: the fresh stage's channel config type must match the actual channel type
						if actualChanType, ok := channelSym.Type.(ir.Chan); ok {
							if err := atypes.CheckEqual(ctx.Constraints, chanType, actualChanType, nil,
								fmt.Sprintf("node %s config %s channel type", n.Key, configKey)); err != nil {
								ctx.Diagnostics.AddError(err, nil)
								return ir.IR{}, *ctx.Diagnostics
							}
						}
					}
				}
			}
		}
	}

	for _, edge := range g.Edges {
		// Validate source node exists
		sourceStage, ok := nodeTypes[edge.Source.Node]
		if !ok {
			ctx.Diagnostics.AddError(
				errors.Newf("edge source node '%s' not found", edge.Source.Node),
				nil,
			)
			return ir.IR{}, *ctx.Diagnostics
		}

		// Validate target node exists
		targetStage, ok := nodeTypes[edge.Target.Node]
		if !ok {
			ctx.Diagnostics.AddError(
				errors.Newf("edge target node '%s' not found", edge.Target.Node),
				nil,
			)
			return ir.IR{}, *ctx.Diagnostics
		}

		// Get source output type (from Outputs or Params)
		var sourceType ir.Type
		if edge.Source.Param == "output" {
			// Using the stage's output type - check Outputs first
			sourceType, ok = sourceStage.Outputs.Get("output")
			if !ok {
				ctx.Diagnostics.AddError(
					errors.Newf("node '%s' has no output", edge.Source.Node),
					nil,
				)
				return ir.IR{}, *ctx.Diagnostics
			}
		} else {
			// Using a named output parameter
			sourceType, ok = sourceStage.Outputs.Get(edge.Source.Param)
			if !ok {
				// Also check Params for backwards compatibility
				sourceType, ok = sourceStage.Params.Get(edge.Source.Param)
				if !ok {
					ctx.Diagnostics.AddError(
						errors.Newf("source param '%s' not found in node '%s'",
							edge.Source.Param, edge.Source.Node),
						nil,
					)
					return ir.IR{}, *ctx.Diagnostics
				}
			}
		}

		// Get target input type
		var targetType ir.Type
		if edge.Target.Param == "output" {
			// Connecting to first parameter (or no parameter if stage has none)
			if targetStage.Params.Count() > 0 {
				_, targetType = targetStage.Params.At(0)
			} else {
				// Stage has no parameters - it will ignore the input
				// This is allowed, similar to JavaScript functions ignoring extra arguments
				targetType = nil
			}
		} else {
			// Connecting to specific parameter
			targetType, ok = targetStage.Params.Get(edge.Target.Param)
			if !ok {
				ctx.Diagnostics.AddError(
					errors.Newf("target param '%s' not found in node '%s' (%s)",
						edge.Target.Param, edge.Target.Node, targetStage.Key),
					nil,
				)
				return ir.IR{}, *ctx.Diagnostics
			}
		}

		// Create type constraint for polymorphic types
		// Only if both source and target types exist (target might be nil for parameterless stages)
		if sourceType != nil && targetType != nil {
			if err := atypes.CheckEqual(ctx.Constraints, sourceType, targetType, nil,
				fmt.Sprintf("edge from %s.%s to %s.%s",
					edge.Source.Node, edge.Source.Param,
					edge.Target.Node, edge.Target.Param)); err != nil {
				// Only report error if neither type is a type variable
				if !atypes.HasTypeVariables(sourceType) && !atypes.HasTypeVariables(targetType) {
					ctx.Diagnostics.AddError(errors.Newf(
						"type mismatch: edge from %s (%s) to %s (%s)",
						edge.Source.Node, sourceType,
						edge.Target.Node, targetType,
					), nil)
					return ir.IR{}, *ctx.Diagnostics
				}
			}
		}
	}

	// Step 5: Unify type variables if any were found
	if ctx.Constraints.HasTypeVariables() {
		if err := ctx.Constraints.Unify(); err != nil {
			ctx.Diagnostics.AddError(err, nil)
			return ir.IR{}, *ctx.Diagnostics
		}
	}

	// Step 5.5: Resolve concrete types for each node instance
	// For polymorphic stages, each node gets its own resolved params/outputs
	nodes := lo.Map(g.Nodes, func(item Node, _ int) ir.Node {
		return item.Node
	})
	for i, n := range nodes {
		// Use the fresh stage that was created in Step 4
		freshStageInst, ok := nodeTypes[n.Key]
		if !ok {
			ctx.Diagnostics.AddError(errors.Newf("node %s not found in nodeTypes", n.Key), nil)
			return ir.IR{}, *ctx.Diagnostics
		}

		// Apply substitutions to get concrete types for THIS node instance
		// We apply to the fresh stage which has unique type variables for this node
		substituted := ctx.Constraints.ApplySubstitutions(freshStageInst)
		if substituted != nil {
			resolvedStage := substituted.(ir.Stage)
			nodes[i].Params = resolvedStage.Params
			nodes[i].Outputs = resolvedStage.Outputs
		} else {
			// No substitutions - use stage types directly
			// This is fine for non-polymorphic stages
			nodes[i].Params = freshStageInst.Params
			nodes[i].Outputs = freshStageInst.Outputs
		}
	}

	// Step 6: Compute stratification for reactive execution
	strata, ok := stratifier.Stratify(ctx, nodes, g.Edges, ctx.Diagnostics)
	if !ok {
		return ir.IR{}, *ctx.Diagnostics
	}

	// Step 7: Return the IR
	return ir.IR{
		Symbols:     ctx.Scope,
		Stages:      g.Stages,
		Edges:       g.Edges,
		Functions:   g.Functions,
		Nodes:       nodes,
		Constraints: ctx.Constraints,
		Strata:      strata,
	}, *ctx.Diagnostics
}
