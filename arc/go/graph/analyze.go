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
	"github.com/synnaxlabs/arc/analyzer/constant"
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
	aCtx := acontext.NewRoot[antlr.ParserRuleContext](
		ctx,
		nil,
		root,
	).WithConfig(parser.ConfigOf(cfgs...))
	if !registerFunctions(aCtx, g.Functions) ||
		!analyzeFunctionBodies(aCtx, g.Functions) {
		return ir.IR{}, aCtx.Diagnostics
	}
	irNodes := make(ir.Nodes, len(g.Nodes))
	for i, n := range g.Nodes {
		node, ok := buildNode(aCtx, n.Key, g.Inputs[n.Key])
		if !ok {
			return ir.IR{}, aCtx.Diagnostics
		}
		irNodes[i] = node
	}
	irEdges := g.Edges.IR()
	if !analyzer.ResolveNodeTypes(
		irNodes,
		irEdges,
		aCtx.Constraints,
		aCtx.Diagnostics,
	) {
		return ir.IR{}, aCtx.Diagnostics
	}
	checkEdgeTargets(aCtx, g.Edges)
	if !aCtx.Diagnostics.Ok() {
		return ir.IR{}, aCtx.Diagnostics
	}
	applySubstitutions(aCtx, g.Functions)
	if !aCtx.Diagnostics.Ok() {
		return ir.IR{}, aCtx.Diagnostics
	}
	out := ir.IR{
		Functions:  g.Functions,
		Edges:      irEdges,
		Nodes:      irNodes,
		Symbols:    aCtx.Scope,
		Root:       rootScope(irNodes),
		TypeMap:    aCtx.TypeMap,
		FmtStrings: aCtx.FmtStrings,
	}
	if len(irNodes) > 0 {
		if d := stratifier.Stratify(&out, aCtx.Diagnostics); d != nil && !d.Ok() {
			return ir.IR{}, d
		}
	}
	return out, aCtx.Diagnostics
}

type analysisContext = acontext.Context[antlr.ParserRuleContext]

func registerFunctions(aCtx analysisContext, fns ir.Functions) bool {
	for _, fn := range fns {
		funcScope, err := aCtx.Scope.Add(aCtx, symbol.Symbol{
			Name: fn.Key,
			Kind: symbol.KindFunction,
			Type: fn.Type(),
			AST:  fn.Body.AST,
		})
		if err == nil {
			err = bindParams(aCtx, funcScope, fn.Inputs, symbol.KindInput)
		}
		if err == nil {
			err = bindParams(aCtx, funcScope, fn.Outputs, symbol.KindOutput)
		}
		if err != nil {
			aCtx.Diagnostics.Add(diagnostics.Error(err, fn.Body.AST))
			return false
		}
	}
	return true
}

func analyzeFunctionBodies(aCtx analysisContext, fns ir.Functions) bool {
	for i, fn := range fns {
		funcScope, err := aCtx.Scope.GetChildByParserRule(fn.Body.AST)
		if err != nil {
			aCtx.Diagnostics.Add(diagnostics.Error(err, fn.Body.AST))
			return false
		}
		if fn.Body.Raw != "" {
			blockCtx, ok := fn.Body.AST.(parser.IBlockContext)
			if !ok {
				aCtx.Diagnostics.Add(diagnostics.Errorf(
					fn.Body.AST, "function body must be a block",
				))
				return false
			}
			analyzer.AnalyzeBlock(aCtx.Child(blockCtx).WithScope(funcScope))
			if !aCtx.Diagnostics.Ok() {
				return false
			}
		}
		fn.Channels = funcScope.Channels
		fn.Symbol = funcScope
		fns[i] = fn
	}
	return true
}

func buildNode(
	aCtx analysisContext,
	key string,
	inputs map[string]any,
) (ir.Node, bool) {
	rawType, ok := inputs["type"]
	if !ok {
		aCtx.Diagnostics.Add(diagnostics.Errorf(
			nil,
			"node '%s' is missing its function type",
			key,
		))
		return ir.Node{}, false
	}
	nodeType, ok := rawType.(string)
	if !ok {
		aCtx.Diagnostics.Add(diagnostics.Errorf(
			nil,
			"node '%s' function type must be a string, got %T",
			key,
			rawType,
		))
		return ir.Node{}, false
	}
	fnSym, err := resolveQualified(aCtx, aCtx.Scope, nodeType)
	if err != nil {
		aCtx.Diagnostics.Add(diagnostics.Error(err, nil))
		return ir.Node{}, false
	}
	if fnSym.Kind != symbol.KindFunction || fnSym.Exec == symbol.ExecWASM {
		aCtx.Diagnostics.Add(diagnostics.Errorf(
			nil,
			"node '%s' type '%s' is not a flow function",
			key,
			nodeType,
		))
		return ir.Node{}, false
	}
	freshType := types.Freshen(fnSym.Type, key)
	node := ir.Node{
		Key:      key,
		Type:     nodeType,
		Channels: fnSym.Channels.Copy(),
		Inputs:   freshType.Inputs,
		Outputs:  freshType.Outputs,
	}
	for j, param := range freshType.Inputs {
		paramValue, ok := inputs[param.Name]
		if !ok {
			continue
		}
		if param.Type.Kind == types.KindChan &&
			!bindChannelParam(aCtx, &node, fnSym, param, paramValue) {
			return ir.Node{}, false
		}
		node.Inputs[j].Value = paramValue
	}
	return node, true
}

// bindChannelParam binds the channel keyed by value into the node's channels. An
// unknown channel stays unbound.
func bindChannelParam(
	aCtx analysisContext,
	node *ir.Node,
	fnSym *symbol.Symbol,
	param types.Param,
	value any,
) bool {
	var k uint32
	if err := zyn.Uint32().Coerce().Parse(value, &k); err != nil {
		return false
	}
	channelSym, err := aCtx.Scope.Resolve(aCtx, strconv.Itoa(int(k)))
	if err != nil || channelSym.Type.Kind != types.KindChan {
		return true
	}
	if err := param.Type.ChanDirection.CheckCompatibility(
		channelSym.Type.ChanDirection,
	); err != nil {
		aCtx.Diagnostics.Add(diagnostics.Error(err, nil))
		return false
	}
	if err := atypes.Check(
		aCtx.Constraints,
		channelSym.Type,
		param.Type,
		nil,
		"",
	); err != nil {
		aCtx.Diagnostics.Add(diagnostics.Error(err, nil))
		return false
	}
	symbol.ResolveInputChannel(&node.Channels, fnSym, param.Name, k, channelSym.Name)
	return true
}

func checkEdgeTargets(aCtx analysisContext, edges Edges) {
	targets := set.New[ir.Handle]()
	for _, edge := range edges {
		if targets.Contains(edge.Target) {
			aCtx.Diagnostics.Add(diagnostics.Errorf(nil,
				"multiple edges target node '%s' parameter '%s'",
				edge.Target.Node,
				edge.Target.Param,
			))
		}
		targets.Add(edge.Target)
	}
}

// applySubstitutions applies the unified types to the type map, then checks the
// constants in each function body.
func applySubstitutions(aCtx analysisContext, fns ir.Functions) {
	for node, typ := range aCtx.TypeMap {
		aCtx.TypeMap[node] = aCtx.Constraints.ApplySubstitutions(typ)
	}
	for _, fn := range fns {
		if fn.Body.Raw != "" {
			constant.Check(aCtx.Child(fn.Body.AST))
		}
	}
}

// rootScope builds the root scope of a graph program. Graphs have no sequences, so
// every node is a member of one parallel stratum, which the stratifier then splits
// by the edges.
func rootScope(nodes ir.Nodes) ir.Scope {
	root := ir.Scope{Mode: ir.ScopeModeParallel, Liveness: ir.LivenessAlways}
	if len(nodes) == 0 {
		return root
	}
	members := make(ir.Members, 0, len(nodes))
	for _, n := range nodes {
		members = append(members, ir.Member{NodeKey: new(n.Key)})
	}
	root.Strata = []ir.Members{members}
	return root
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
