// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

type Text struct {
	Raw string                 `json:"raw" msgpack:"raw"`
	AST parser.IProgramContext `json:"-"`
}

type GenerateKey = func(name string) string

func Parse(t Text) (Text, error) {
	ast, err := parser.Parse(t.Raw)
	if err != nil {
		return Text{}, err
	}
	t.AST = ast
	return t, err
}

func Analyze(
	ctx_ context.Context,
	t Text,
	resolver symbol.Resolver,
) (ir.IR, analyzer.Diagnostics) {
	ctx := acontext.CreateRoot(ctx_, t.AST, resolver)
	// func 1: Analyse the AST.
	if !analyzer.AnalyzeProgram(ctx) {
		return ir.IR{}, *ctx.Diagnostics
	}
	i := ir.IR{Symbols: ctx.Scope, Constraints: ctx.Constraints}

	// func 2: Iterate through the root scope children to assemble functions
	for _, c := range i.Symbols.Children {
		if c.Kind == symbol.KindFunction {
			// Convert types.Type + AST → ir.Function
			typeFunc, ok := c.Type.(types.Type)
			if !ok || typeFunc.Kind != types.KindFunction {
				continue
			}
			fn := ir.Function{
				Key:     c.Name,
				Body:    types.Body{Raw: "", AST: c.AST},
				Config:  *typeFunc.Config,
				Inputs:  *typeFunc.Inputs,
				Outputs: *typeFunc.Outputs,
			}
			i.Functions = append(i.Functions, fn)
		}
	}

	var (
		counter     = 0
		generateKey = func(name string) string {
			return fmt.Sprintf("%s_%d", name, counter)
		}
	)

	// Second pass: process flow statements to build nodes and edges
	for _, item := range t.AST.AllTopLevelItem() {
		if flow := item.FlowStatement(); flow != nil {
			nodes, edges, ok := analyzeFlow(acontext.Child(ctx, flow), generateKey)
			if !ok {
				return ir.IR{}, *ctx.Diagnostics
			}
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		}
	}

	return i, *ctx.Diagnostics
}

func Compile(
	ctx_ context.Context,
	ir ir.IR,
	opts ...compiler.Option,
) (module.Module, error) {
	o, err := compiler.Compile(ctx_, ir, opts...)
	if err != nil {
		return module.Module{}, err
	}
	return module.Module{IR: ir, Output: o}, nil
}

func analyzeFlow(
	ctx acontext.Context[parser.IFlowStatementContext],
	generateKey GenerateKey,
) ([]ir.Node, []ir.Edge, bool) {
	var (
		prevHandle ir.Handle
		edges      []ir.Edge
		nodes      []ir.Node
	)
	for i, flowNode := range ctx.AST.AllFlowNode() {
		node, handle, ok := analyzeNode(acontext.Child(ctx, flowNode), generateKey)
		if !ok {
			return nil, nil, false
		}
		if i > 0 {
			edges = append(edges, ir.Edge{Source: prevHandle, Target: handle})
		}
		prevHandle = handle
		nodes = append(nodes, node)
	}
	return nodes, edges, true
}

func analyzeNode(
	ctx acontext.Context[parser.IFlowNodeContext],
	generateKey GenerateKey,
) (ir.Node, ir.Handle, bool) {
	if channel := ctx.AST.ChannelIdentifier(); channel != nil {
		return analyzeChannel(acontext.Child(ctx, channel), generateKey)
	}
	if fn := ctx.AST.Function(); fn != nil {
		return analyzeFunction(acontext.Child(ctx, fn), generateKey)
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return analyzeExpression(acontext.Child(ctx, expr))
	}
	return ir.Node{}, ir.Handle{}, true
}

func analyzeChannel(
	ctx acontext.Context[parser.IChannelIdentifierContext],
	generateKey GenerateKey,
) (ir.Node, ir.Handle, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	nodeKey := generateKey("on")
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return ir.Node{}, ir.Handle{}, false
	}
	chKey := uint32(sym.ID)
	n := ir.Node{
		Key:          nodeKey,
		Type:         "on",
		ConfigValues: map[string]any{"channel": chKey},
		Channels:     ir.NewChannels(),
	}
	n.Channels.Read.Add(chKey)
	h := ir.Handle{Node: nodeKey, Param: ir.DefaultOutputParam}
	return n, h, true
}

func extractConfigValues(
	ctx acontext.Context[parser.IConfigValuesContext],
	stageType ir.Stage,
	node ir.Node,
) (map[string]any, bool) {
	config := make(map[string]any)
	if ctx.AST == nil {
		return config, true
	}
	if named := ctx.AST.NamedConfigValues(); named != nil {
		for _, cv := range named.AllNamedConfigValue() {
			key := cv.IDENTIFIER().GetText()
			config[key] = getExpressionText(cv.Expression())
		}
	} else if anon := ctx.AST.AnonymousConfigValues(); anon != nil {
		for i, expr := range anon.AllExpression() {
			key, _ := stageType.Inputs.At(i)
			config[key] = getExpressionText(expr)
		}
	}
	for k, v := range config {
		t, ok := stageType.Config.Get(k)
		if !ok {
			panic("config key not found in stage")
		}
		if _, ok = t.(types.Chan); ok {
			sym, err := ctx.Scope.Resolve(ctx, v.(string))
			if err != nil {
				ctx.Diagnostics.AddError(err, nil)
				return nil, false
			}
			channelKey := uint32(sym.ID)
			config[k] = channelKey
			node.Channels.Read.Add(channelKey)
		}
	}
	return config, true
}

func analyzeFunction(
	ctx acontext.Context[parser.IFunctionContext],
	generateKey GenerateKey,
) (ir.Node, ir.Handle, bool) {
	var (
		name = ctx.AST.IDENTIFIER().GetText()
		key  = generateKey(name)
	)
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return ir.Node{}, ir.Handle{}, false
	}
	stageType, err := types.Assert[ir.Stage](sym.Type)
	if err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return ir.Node{}, ir.Handle{}, false
	}
	n := ir.Node{
		Key:      key,
		Type:     name,
		Channels: ir.OverrideChannels(stageType.Channels),
	}
	config, ok := extractConfigValues(
		acontext.Child(ctx, ctx.AST.ConfigValues()),
		stageType,
		n,
	)
	if !ok {
		return ir.Node{}, ir.Handle{}, false
	}
	if args := ctx.AST.Arguments(); args != nil {
		if argList := args.ArgumentList(); argList != nil {
			for i, expr := range argList.AllExpression() {
				config[fmt.Sprintf("_runtime%d", i)] = getExpressionText(expr)
			}
		}
	}
	h := ir.Handle{Node: key, Param: "input"}
	return n, h, true
}

func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) (ir.Node, ir.Handle, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return ir.Node{}, ir.Handle{}, false
	}
	stageType, err := types.Assert[ir.Stage](sym.Type)
	if err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return ir.Node{}, ir.Handle{}, false
	}
	n := ir.Node{
		Key:      sym.Name,
		Type:     sym.Name,
		Channels: ir.OverrideChannels(stageType.Channels),
	}
	h := ir.Handle{Node: sym.Name, Param: ir.DefaultOutputParam}
	return n, h, true
}

// getExpressionText extracts the text representation of an expression
// In a full implementation, this would properly serialize the expression AST
func getExpressionText(expr parser.IExpressionContext) string {
	if expr == nil {
		return ""
	}
	// Resolve the original text from the token stream
	start := expr.GetStart()
	stop := expr.GetStop()
	if start != nil && stop != nil {
		stream := start.GetTokenSource().GetInputStream()
		if stream != nil {
			return stream.GetText(start.GetStart(), stop.GetStop())
		}
	}
	return expr.GetText()
}
