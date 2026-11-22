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
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
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
	return scope.ResolveMember(ctx, head, tail, symbol.IncludeInternal)
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
	if !registerFunctions(aCtx, g.Functions) {
		return ir.IR{}, aCtx.Diagnostics
	}
	if !analyzeFunctionBodies(aCtx, g.Functions) {
		return ir.IR{}, aCtx.Diagnostics
	}
	irNodes, ok := buildNodes(aCtx, g)
	if !ok {
		return ir.IR{}, aCtx.Diagnostics
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
	if !checkEdgeTargets(aCtx, g.Edges) {
		return ir.IR{}, aCtx.Diagnostics
	}
	for node, typ := range aCtx.TypeMap {
		aCtx.TypeMap[node] = aCtx.Constraints.ApplySubstitutions(typ)
	}
	for _, fn := range g.Functions {
		if fn.Body.Raw != "" {
			constant.Check(aCtx.Child(fn.Body.AST))
		}
	}
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

// registerFunctions adds a scope for each function in fns, with its inputs and
// outputs bound. It reports false after it adds an error to the diagnostics.
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

// analyzeFunctionBodies analyzes the body of each function in fns and binds the
// resolved channels and scope back into fns. It reports false when analysis fails.
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

// buildNodes returns an IR node with freshened types for each node in g. It reports
// false after it adds an error to the diagnostics.
func buildNodes(aCtx analysisContext, g Graph) (ir.Nodes, bool) {
	irNodes := make(ir.Nodes, len(g.Nodes))
	for i, n := range g.Nodes {
		node, ok := buildNode(aCtx, n.Key, g.Inputs[n.Key])
		if !ok {
			return nil, false
		}
		irNodes[i] = node
	}
	return irNodes, true
}

func buildNode(
	aCtx analysisContext,
	key string,
	inputs msgpack.EncodedJSON,
) (ir.Node, bool) {
	nodeType, fnSym, ok := resolveNodeFunction(aCtx, key, inputs)
	if !ok {
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
	// Param values come from the node's entry in the graph inputs map.
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

// resolveNodeFunction resolves the flow function that the "type" entry of a node's
// inputs names. It reports false after it adds an error to the diagnostics.
func resolveNodeFunction(
	aCtx analysisContext,
	key string,
	inputs msgpack.EncodedJSON,
) (string, *symbol.Symbol, bool) {
	rawType, ok := inputs["type"]
	if !ok {
		aCtx.Diagnostics.Add(diagnostics.Errorf(
			nil,
			"node '%s' is missing its function type",
			key,
		))
		return "", nil, false
	}
	nodeType, ok := rawType.(string)
	if !ok {
		aCtx.Diagnostics.Add(diagnostics.Errorf(
			nil,
			"node '%s' function type must be a string, got %T",
			key,
			rawType,
		))
		return "", nil, false
	}
	fnSym, err := resolveQualified(aCtx, aCtx.Scope, nodeType)
	if err != nil {
		aCtx.Diagnostics.Add(diagnostics.Error(err, nil))
		return "", nil, false
	}
	if fnSym.Kind != symbol.KindFunction || fnSym.Exec == symbol.ExecWASM {
		aCtx.Diagnostics.Add(diagnostics.Errorf(
			nil,
			"node '%s' type '%s' is not a flow function",
			key,
			nodeType,
		))
		return "", nil, false
	}
	return nodeType, fnSym, true
}

// bindChannelParam checks the channel that value names against param and binds it
// into the channels of node. A key that resolves to no channel binds nothing. It
// reports false after it adds an error to the diagnostics.
func bindChannelParam(
	aCtx analysisContext,
	node *ir.Node,
	fnSym *symbol.Symbol,
	param types.Param,
	value any,
) bool {
	var k uint32
	if err := zyn.Uint32().Coerce().Parse(value, &k); err != nil {
		aCtx.Diagnostics.Add(diagnostics.Error(errors.Wrapf(
			err,
			"node '%s' parameter '%s' is not a channel key",
			node.Key,
			param.Name,
		), nil))
		return false
	}
	channelSym, err := aCtx.Scope.ResolveChannelKey(aCtx, k)
	if err != nil || channelSym.Type.Kind != types.KindChan {
		return true
	}
	if err = param.Type.ChanDirection.CheckCompatibility(
		channelSym.Type.ChanDirection,
	); err != nil {
		aCtx.Diagnostics.Add(diagnostics.Error(err, nil))
		return false
	}
	if err = atypes.Check(
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

// checkEdgeTargets adds an error for each edge that targets a parameter an earlier
// edge already targets. It reports false when the diagnostics hold any error.
func checkEdgeTargets(aCtx analysisContext, edges Edges) bool {
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
	return aCtx.Diagnostics.Ok()
}

// rootScope returns the root scope of a graph program. A graph has no sequence
// constructs, so every node joins the catch-all phase of the root scope. The
// stratifier rewrites the phase layout from the edge set.
func rootScope(nodes ir.Nodes) ir.Scope {
	root := ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
	}
	if len(nodes) > 0 {
		members := make(ir.Members, 0, len(nodes))
		for _, n := range nodes {
			members = append(members, ir.Member{NodeKey: new(n.Key)})
		}
		root.Strata = []ir.Members{members}
	}
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
