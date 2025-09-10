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
	"fmt"

	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
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
	t Text,
	resolver ir.SymbolResolver,
) (ir.IR, analyzer.Diagnostics) {
	ctx := context.CreateRoot(t.AST, resolver)
	// Stage 1: Analyse the AST.
	if !analyzer.AnalyzeProgram(ctx) {
		return ir.IR{}, *ctx.Diagnostics
	}
	i := ir.IR{Symbols: ctx.Scope}

	// Stage 2: Iterate through the root scope children to assemble
	// functions and stages.
	for _, c := range i.Symbols.Children {
		if c.Kind == ir.KindStage {
			i.Stages = append(i.Stages, c.Type.(ir.Stage))
		} else if c.Kind == ir.KindFunction {
			i.Functions = append(i.Functions, c.Type.(ir.Function))
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
			nodes, edges, err := processFlow(context.Child(ctx, flow), generateKey)
			if err != nil {
				ctx.Diagnostics.AddError(err, nil)
				return ir.IR{}, *ctx.Diagnostics
			}
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		}
	}

	return i, *ctx.Diagnostics
}

func processFlow(
	ctx context.Context[parser.IFlowStatementContext],
	generateKey GenerateKey,
) ([]ir.Node, []ir.Edge, error) {
	var (
		prevHandle ir.Handle
		edges      []ir.Edge
		nodes      []ir.Node
	)
	for i, flowNode := range ctx.AST.AllFlowNode() {
		node, handle, err := processFlowNode(context.Child(ctx, flowNode), generateKey)
		if err != nil {
			return nil, nil, err
		}
		if i > 0 {
			edges = append(edges, ir.Edge{Source: prevHandle, Target: handle})
		}
		prevHandle = handle
		nodes = append(nodes, node)
	}
	return nodes, edges, nil
}

func processFlowNode(
	ctx context.Context[parser.IFlowNodeContext],
	generateKey GenerateKey,
) (ir.Node, ir.Handle, error) {
	if channel := ctx.AST.ChannelIdentifier(); channel != nil {
		return processChannel(channel, generateKey)
	}
	if stage := ctx.AST.StageInvocation(); stage != nil {
		return processStage(context.Child(ctx, stage), generateKey)
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return processExpression(context.Child(ctx, expr))
	}
	return ir.Node{}, ir.Handle{}, nil
}

func processChannel(channel parser.IChannelIdentifierContext, generateKey GenerateKey) (ir.Node, ir.Handle, error) {
	name := channel.IDENTIFIER().GetText()
	nodeKey := generateKey("on")
	return ir.Node{
		Key:    nodeKey,
		Type:   "on",
		Config: map[string]any{"channel": name},
	}, ir.Handle{Node: nodeKey, Param: "output"}, nil
}

func extractConfigValues(values parser.IConfigValuesContext, stageType ir.Stage) map[string]any {
	config := make(map[string]any)
	if values == nil {
		return config
	}
	if named := values.NamedConfigValues(); named != nil {
		for _, cv := range named.AllNamedConfigValue() {
			key := cv.IDENTIFIER().GetText()
			config[key] = getExpressionText(cv.Expression())
		}
	} else if anon := values.AnonymousConfigValues(); anon != nil {
		for i, expr := range anon.AllExpression() {
			key, _ := stageType.Params.At(i)
			config[key] = getExpressionText(expr)
		}
	}
	return config
}

func processStage(
	ctx context.Context[parser.IStageInvocationContext],
	generateKey GenerateKey,
) (ir.Node, ir.Handle, error) {
	var (
		name = ctx.AST.IDENTIFIER().GetText()
		key  = generateKey(name)
	)
	sym, err := ctx.Scope.Resolve(name)
	if err != nil {
		return ir.Node{}, ir.Handle{}, err
	}
	stageType, err := ir.Assert[ir.Stage](sym.Type)
	if err != nil {
		return ir.Node{}, ir.Handle{}, err
	}
	config := extractConfigValues(ctx.AST.ConfigValues(), stageType)
	if args := ctx.AST.Arguments(); args != nil {
		if argList := args.ArgumentList(); argList != nil {
			for i, expr := range argList.AllExpression() {
				config[fmt.Sprintf("_runtime%d", i)] = getExpressionText(expr)
			}
		}
	}
	return ir.NewNode(ir.Node{Key: key, Type: name, Config: config}), ir.Handle{Node: key, Param: "output"}, nil
}

func processExpression(ctx context.Context[parser.IExpressionContext]) (ir.Node, ir.Handle, error) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		return ir.Node{}, ir.Handle{}, err
	}
	return ir.NewNode(ir.Node{Key: sym.Name, Type: sym.Name}), ir.Handle{Node: sym.Name, Param: "output"}, nil
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
