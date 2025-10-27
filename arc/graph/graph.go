// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package graph provides visual graph compilation for Arc programs.
//
// This package transforms a visual graph representation (nodes, edges, viewport)
// into an intermediate representation (IR) suitable for stratification and runtime
// execution. It serves as the 5th stage in the Arc compiler pipeline:
//
//	Parser → Analyzer → Stratifier → Graph → Compiler → Runtime
//
// The package handles:
//   - Parsing function bodies from raw text into AST
//   - Type inference and constraint solving for polymorphic functions
//   - Edge validation between node inputs/outputs
//   - Configuration value type checking
//   - Stratification of execution order
//
// The core compilation process is performed by Analyze(), which implements a
// 10-step pipeline that produces executable IR from a visual graph.
package graph

import (
	"context"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
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
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/zyn"
)

// Type aliases for IR types to avoid circular dependencies while maintaining
// clean API boundaries.
type (
	Function = ir.Function
	Edge     = ir.Edge
	Edges    = ir.Edges
	Handle   = ir.Handle
)

// Node represents a visual node in an Arc graph. Unlike ir.Node, which contains
// compiled type information, Node represents the user's visual layout including
// position and raw configuration values.
type Node struct {
	// Key is the unique identifier for this node instance.
	Key string `json:"key"`
	// Type is the function type this node instantiates.
	Type string `json:"type"`
	// ConfigValues are the raw configuration parameter values.
	ConfigValues map[string]any `json:"config_values"`
	// Position is the visual position in the graph editor.
	Position spatial.XY `json:"position"`
}

// Nodes is a slice of Node with helper methods for lookup operations.
type Nodes []Node

// Get returns the node with the given key. Panics if the node is not found.
// Use Find for safe lookups with error handling.
func (n Nodes) Get(key string) Node {
	return lo.Must(lo.Find(n, func(n Node) bool { return n.Key == key }))
}

// Find returns the node with the given key and a boolean indicating whether
// the node was found. This is the safe variant of Get.
func (n Nodes) Find(key string) (Node, bool) {
	return lo.Find(n, func(n Node) bool { return n.Key == key })
}

// Viewport represents the visual viewport state of the graph editor.
type Viewport struct {
	// Position is the pan offset of the viewport.
	Position spatial.XY `json:"position"`
	// Zoom is the zoom level of the viewport.
	Zoom float32 `json:"zoom"`
}

// Graph represents a complete visual graph.
type Graph struct {
	// Viewport is the visual viewport state.
	Viewport Viewport `json:"viewport"`
	// Functions are the function definitions available in the graph.
	Functions []Function `json:"functions"`
	// Edges connect node outputs to node inputs.
	Edges Edges `json:"edges"`
	// Nodes are the visual node instances in the graph.
	Nodes Nodes `json:"nodes"`
}

// bindParams adds function parameters to the symbol scope with the specified kind.
// Used internally to bind Config, Input, and Output parameters during function
// registration.
func bindParams(
	ctx context.Context,
	s *symbol.Scope,
	p types.Params,
	kind symbol.Kind,
) error {
	for k, ty := range p.Iter() {
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

// validateEdge checks type compatibility between an edge's source and target.
// Returns true if validation succeeds, false if an error was added to diagnostics.
func validateEdge(
	ctx *acontext.Context[antlr.ParserRuleContext],
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

	sourceType, ok := freshFuncTypes[sourceNode.Key].Outputs.Get(edge.Source.Param)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Wrapf(
				query.NotFound,
				"output '%s' not found in node '%s'",
				edge.Source.Param,
				edge.Source.Node,
			), nil)
		return false
	}

	targetType, ok := freshFuncTypes[targetNode.Key].Inputs.Get(edge.Target.Param)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Wrapf(
				query.NotFound,
				"input '%s' not found in node '%s'",
				edge.Target.Param,
				edge.Target.Node,
			), nil)
		return false
	}

	if err := atypes.Check(
		ctx.Constraints,
		sourceType,
		targetType,
		nil,
		"",
	); err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return false
	}
	return true
}

// Parse parses the raw function bodies in the graph into AST representations.
// It skips functions with empty bodies and returns an error if parsing fails.
// This is typically the first step before calling Analyze.
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
		freshFuncTypes[n.Key] = ir.FreshType(fnSym.Type, n.Key)
		irNodes[i] = ir.Node{
			Key:          n.Key,
			Type:         n.Type,
			ConfigValues: n.ConfigValues,
			Channels:     fnSym.Channels.Copy(),
		}
		freshType := freshFuncTypes[n.Key]
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
				if err = zyn.Uint32().Coerce().Parse(configValue, &k); err != nil {
					return ir.IR{}, ctx.Diagnostics
				}
				channelSym, err := resolver.Resolve(ctx_, strconv.Itoa(int(k)))
				if err == nil && channelSym.Type.Kind == types.KindChan {
					if err = atypes.Check(
						ctx.Constraints,
						channelSym.Type,
						configType,
						nil,
						"",
					); err != nil {
						ctx.Diagnostics.AddError(err, nil)
						return ir.IR{}, ctx.Diagnostics
					}
					irNodes[i].Channels.Read.Add(k)
				}
			}
		}
	}

	// Step 5: Check Types Across Edges
	for _, edge := range g.Edges {
		if !validateEdge(ctx, edge, g.Nodes, freshFuncTypes) {
			return ir.IR{}, ctx.Diagnostics
		}
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
		irN.Outputs = *substituted.Outputs
		irN.Inputs = *substituted.Inputs
		irN.Config = *substituted.Config
		irNodes[i] = irN
	}

	// Step 8: Build Stratified Execution Plan
	strata, ok := stratifier.Stratify(ctx, irNodes, g.Edges, ctx.Diagnostics)
	if !ok {
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
