// Copyright 2026 Synnax Labs, Inc.
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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/zyn"
)

// resolveQualified looks up name against scope. A literal-name lookup
// is tried first so a scope child whose Name contains a dot still
// resolves — graph tests register function symbols whose Key is the
// joined "module.member" string. When that misses, the name is split
// at the first dot and walked as head + tail. Used at the IR/string
// boundary where graph node Type is a joined "module.member" string.
func resolveQualified(
	ctx context.Context,
	scope *symbol.Symbol,
	name string,
) (*symbol.Symbol, error) {
	if sym, err := scope.Resolve(ctx, name, symbol.IncludeInternal); err == nil {
		return sym, nil
	}
	head, tail, ok := strings.Cut(name, ".")
	if !ok {
		return scope.Resolve(ctx, name)
	}
	headSym, err := scope.Resolve(ctx, head, symbol.IncludeInternal)
	if err != nil {
		return nil, err
	}
	return headSym.Resolve(ctx, tail)
}

// Analyze compiles a visual graph into executable IR with type inference,
// edge validation, and stratified execution planning. Errors are collected
// in the returned Diagnostics.
//
// The root parameter is the pre-built program root. Callers construct it
// (typically via stl.NewRoot) so the graph package never needs to know
// which built-ins are loaded or where external symbols come from.
func Analyze(
	ctx context.Context,
	g Graph,
	root *symbol.Symbol,
	cfgs ...parser.Config,
) (ir.IR, *diagnostics.Diagnostics) {
	// Step 1: Build Root Context and Register All Functions
	aCtx := acontext.NewRoot[antlr.ParserRuleContext](ctx, nil, root).WithConfig(parser.ConfigOf(cfgs...))
	for _, fn := range g.Functions {
		funcScope, err := aCtx.Scope.Add(aCtx, symbol.Symbol{
			Name: fn.Key,
			Kind: symbol.KindFunction,
			Type: fn.Type(),
			AST:  fn.Body.AST,
		})
		if err != nil {
			aCtx.Diagnostics.Add(diagnostics.Error(err, fn.Body.AST))
			return ir.IR{}, aCtx.Diagnostics
		}
		if err = bindParams(aCtx, funcScope, fn.Inputs, symbol.KindInput); err != nil {
			aCtx.Diagnostics.Add(diagnostics.Error(err, fn.Body.AST))
			return ir.IR{}, aCtx.Diagnostics
		}
		if err = bindParams(aCtx, funcScope, fn.Outputs, symbol.KindOutput); err != nil {
			aCtx.Diagnostics.Add(diagnostics.Error(err, fn.Body.AST))
			return ir.IR{}, aCtx.Diagnostics
		}
	}

	// Step 2: Analyze Function Bodies
	for i, fn := range g.Functions {
		funcScope, err := aCtx.Scope.GetChildByParserRule(fn.Body.AST)
		if err != nil {
			aCtx.Diagnostics.Add(diagnostics.Error(err, fn.Body.AST))
			return ir.IR{}, aCtx.Diagnostics
		}
		if fn.Body.Raw != "" {
			blockCtx, ok := fn.Body.AST.(parser.IBlockContext)
			if !ok {
				aCtx.Diagnostics.Add(diagnostics.Errorf(
					fn.Body.AST, "function body must be a block"))
				return ir.IR{}, aCtx.Diagnostics
			}
			analyzer.AnalyzeBlock(acontext.Child(aCtx, blockCtx).WithScope(funcScope))
			if !aCtx.Diagnostics.Ok() {
				return ir.IR{}, aCtx.Diagnostics
			}
		}
		fn.Channels = funcScope.Channels
		g.Functions[i] = fn
	}

	// Step 3 & 4: Create Fresh Types and IR Nodes
	freshFuncTypes := make(map[string]types.Type)
	irNodes := make(ir.Nodes, len(g.Nodes))
	for i, n := range g.Nodes {
		inputs := g.Inputs[n.Key]
		rawType, ok := inputs["type"]
		if !ok {
			aCtx.Diagnostics.Add(diagnostics.Errorf(
				nil,
				"node '%s' is missing its function type",
				n.Key,
			))
			return ir.IR{}, aCtx.Diagnostics
		}
		nodeType, ok := rawType.(string)
		if !ok {
			aCtx.Diagnostics.Add(diagnostics.Errorf(
				nil,
				"node '%s' function type must be a string, got %T",
				n.Key,
				rawType,
			))
			return ir.IR{}, aCtx.Diagnostics
		}
		fnSym, err := resolveQualified(aCtx, aCtx.Scope, nodeType)
		if err != nil {
			aCtx.Diagnostics.Add(diagnostics.Error(err, nil))
			return ir.IR{}, aCtx.Diagnostics
		}
		freshFuncTypes[n.Key] = types.Freshen(fnSym.Type, n.Key)
		freshType := freshFuncTypes[n.Key]
		node := ir.Node{
			Key:      n.Key,
			Type:     nodeType,
			Channels: fnSym.Channels.Copy(),
			Inputs:   freshType.Inputs,
			Outputs:  freshType.Outputs,
		}
		// Param values come from the node's entry in the graph inputs map.
		for j, param := range freshType.Inputs {
			paramValue, ok := inputs[param.Name]
			if !ok {
				continue
			}
			if param.Type.Kind == types.KindChan {
				var k uint32
				if err = zyn.Uint32().Coerce().Parse(paramValue, &k); err != nil {
					return ir.IR{}, aCtx.Diagnostics
				}
				channelSym, err := aCtx.Scope.Resolve(aCtx, strconv.Itoa(int(k)))
				if err == nil && channelSym.Type.Kind == types.KindChan {
					if err := param.Type.ChanDirection.CheckCompatibility(channelSym.Type.ChanDirection); err != nil {
						aCtx.Diagnostics.Add(diagnostics.Error(err, nil))
						return ir.IR{}, aCtx.Diagnostics
					}
					if err = atypes.Check(
						aCtx.Constraints,
						channelSym.Type,
						param.Type,
						nil,
						"",
					); err != nil {
						aCtx.Diagnostics.Add(diagnostics.Error(err, nil))
						return ir.IR{}, aCtx.Diagnostics
					}
					symbol.ResolveConfigChannel(
						&node.Channels,
						fnSym,
						param.Name,
						k,
						channelSym.Name,
					)
				}
			}
			node.Inputs[j].Value = paramValue
		}
		irNodes[i] = node
	}

	// Step 5: Check Types Across Edges, Unify, and Apply Substitutions
	irEdges := g.Edges.IR()
	if !analyzer.ResolveNodeTypes(irNodes, irEdges, aCtx.Constraints, aCtx.Diagnostics) {
		return ir.IR{}, aCtx.Diagnostics
	}

	// Step 5A: Check for Duplicate Edge Targets
	targets := set.New[ir.Handle]()
	for _, edge := range g.Edges {
		if targets.Contains(edge.Target) {
			aCtx.Diagnostics.Add(diagnostics.Errorf(nil,
				"multiple edges target node '%s' parameter '%s'",
				edge.Target.Node,
				edge.Target.Param,
			))
		}
		targets.Add(edge.Target)
	}
	if !aCtx.Diagnostics.Ok() {
		return ir.IR{}, aCtx.Diagnostics
	}

	// Step 6: Substitute TypeMap after unification
	for node, typ := range aCtx.TypeMap {
		aCtx.TypeMap[node] = aCtx.Constraints.ApplySubstitutions(typ)
	}

	// Step 7: Build the Layer-2 root scope. Graph-based compilation has no
	// sequence constructs, so every node becomes a member of the root
	// scope's catch-all phase; the stratifier rewrites the phase layout
	// based on the edge set.
	irRoot := ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
	}
	if len(irNodes) > 0 {
		members := make(ir.Members, 0, len(irNodes))
		for _, n := range irNodes {
			members = append(members, ir.Member{NodeKey: new(n.Key)})
		}
		irRoot.Strata = []ir.Members{members}
	}
	out := ir.IR{
		Functions: g.Functions,
		Edges:     irEdges,
		Nodes:     irNodes,
		Symbols:   aCtx.Scope,
		Root:      irRoot,
		TypeMap:   aCtx.TypeMap,
	}
	if len(irNodes) > 0 {
		if d := stratifier.Stratify(aCtx, &out, aCtx.Diagnostics); d != nil && !d.Ok() {
			return ir.IR{}, d
		}
	}
	return out, aCtx.Diagnostics
}

// bindParams adds function parameters to the symbol scope with the specified
// kind. Used internally to bind input and output parameters during function
// registration.
func bindParams(
	ctx context.Context,
	scope *symbol.Symbol,
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
