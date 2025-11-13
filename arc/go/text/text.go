// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package text provides parsing, analysis, and compilation of Arc source code.
//
// The package implements a three-stage pipeline:
//   - Parse: Converts raw text into an Abstract Syntax Tree (AST)
//   - Analyze: Performs semantic analysis and builds Intermediate Representation (IR)
//   - Compile: Generates WebAssembly bytecode from IR
//
// The analyzer uses a multi-pass approach:
//  1. Analyze function declarations and build the symbol table
//  2. Process flow statements to construct nodes and edges
//  3. Calculate execution stratification for deterministic reactive execution
package text

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// Text represents Arc source code with its parsed AST.
type Text struct {
	Raw string                 `json:"raw" msgpack:"raw"`
	AST parser.IProgramContext `json:"-"`
}

// generateKey generates unique node keys for flow graph construction.
type generateKey = func(name string) string

// Parse parses Arc source code into an AST.
//
// Returns the Text with both Raw source and parsed AST. Returns a diagnostic object
// that will be nil if no errors occurred during the parsing process.
func Parse(t Text) (Text, *diagnostics.Diagnostics) {
	ast, diag := parser.Parse(t.Raw)
	if diag != nil {
		return Text{}, diag
	}
	t.AST = ast
	return t, diag
}

// Analyze performs semantic analysis on parsed Arc code and builds the IR.
//
// The analysis uses a two-pass approach:
//  1. First pass: Analyzes function declarations and builds the symbol table
//  2. Second pass: Processes flow statements to construct nodes and edges
//
// The resolver parameter provides symbol resolution for external references such
// as channels. Pass nil if no external symbols are available.
//
// Returns a partially complete IR even if diagnostics contain errors, enabling
// tools like LSPs to provide the most complete understanding of the document.
func Analyze(
	ctx_ context.Context,
	t Text,
	resolver symbol.Resolver,
) (ir.IR, *diagnostics.Diagnostics) {
	var (
		ctx = acontext.CreateRoot(ctx_, t.AST, resolver)
		// We always return a partially complete IR to ensure that tools such as LSP's
		// have the most complete understanding of the document.
		i = ir.IR{Symbols: ctx.Scope, TypeMap: ctx.TypeMap}
	)
	// Step 1: Analyze the Program
	if !analyzer.AnalyzeProgram(ctx) {
		return i, ctx.Diagnostics
	}

	// Step 2: Iterate through the root scope children to assemble functions
	for _, c := range i.Symbols.Children {
		if c.Kind == symbol.KindFunction {
			fnDecl, ok := c.AST.(parser.IFunctionDeclarationContext)
			var bodyAst antlr.ParserRuleContext = fnDecl
			if ok {
				bodyAst = fnDecl.Block()
			}
			i.Functions = append(i.Functions, ir.Function{
				Key:      c.Name,
				Body:     ir.Body{Raw: "", AST: bodyAst},
				Config:   c.Type.Config,
				Inputs:   c.Type.Inputs,
				Outputs:  c.Type.Outputs,
				Channels: c.Channels,
			})
		}
	}

	var (
		counter     = 0
		generateKey = func(name string) string {
			return fmt.Sprintf("%s_%d", name, counter)
		}
	)

	// Step 3: Process Flow Nodes and Statements to Build Nodes/Edges
	for _, item := range t.AST.AllTopLevelItem() {
		if flow := item.FlowStatement(); flow != nil {
			nodes, edges, ok := analyzeFlow(acontext.Child(ctx, flow), generateKey)
			if !ok {
				return i, ctx.Diagnostics
			}
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		}
	}

	// Step 4: Calculate execution stratification for deterministic reactive execution
	if len(i.Nodes) > 0 {
		strata, diag := stratifier.Stratify(ctx_, i.Nodes, i.Edges, ctx.Diagnostics)
		if diag != nil && !diag.Ok() {
			ctx.Diagnostics = diag
			return i, ctx.Diagnostics
		}
		i.Strata = strata
	}

	return i, ctx.Diagnostics
}

// Compile generates WebAssembly bytecode from the provided IR.
//
// Returns a Module containing both the IR and the compiled WebAssembly output.
// Compiler options can be provided to customize the compilation process.
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
	generateKey generateKey,
) ([]ir.Node, []ir.Edge, bool) {
	var (
		prevOutputHandle ir.Handle
		prevNode         *ir.Node
		edges            []ir.Edge
		nodes            []ir.Node
	)

	// Check if this flow statement contains routing tables
	hasRoutingTables := len(ctx.AST.AllRoutingTable()) > 0

	if !hasRoutingTables {
		// Simple flow chain without routing tables
		for i, flowNode := range ctx.AST.AllFlowNode() {
			node, inputHandle, outputHandle, ok := analyzeExpressionNode(acontext.Child(ctx, flowNode), generateKey)
			if !ok {
				return nil, nil, false
			}
			if i > 0 {
				// Connect previous node's output to current node's input
				edges = append(edges, ir.Edge{Source: prevOutputHandle, Target: inputHandle})
			}
			// Store output handle for next iteration
			prevOutputHandle = outputHandle
			nodes = append(nodes, node)
		}
		return nodes, edges, true
	}

	// Flow chain with routing tables - iterate through children to maintain order
	children := ctx.AST.GetChildren()
	for _, child := range children {
		switch c := child.(type) {
		case parser.IFlowNodeContext:
			node, inputHandle, outputHandle, ok := analyzeExpressionNode(acontext.Child(ctx, c), generateKey)
			if !ok {
				return nil, nil, false
			}
			// Connect to previous node if exists
			if prevNode != nil {
				edges = append(edges, ir.Edge{Source: prevOutputHandle, Target: inputHandle})
			}
			// Store for next iteration
			prevOutputHandle = outputHandle
			prevNode = &node
			nodes = append(nodes, node)

		case parser.IRoutingTableContext:
			if prevNode == nil {
				// Input routing table: { param1: source1, param2: source2 } -> func
				newNodes, newEdges, ok := analyzeInputRoutingTable(
					acontext.Child(ctx, c),
					generateKey,
				)
				if !ok {
					return nil, nil, false
				}
				nodes = append(nodes, newNodes...)
				edges = append(edges, newEdges...)
				// Find the last node to connect to next flow node
				if len(newNodes) > 0 {
					lastNode := newNodes[len(newNodes)-1]
					prevNode = &lastNode
					// Output handle depends on what follows the routing table
					outputParam := ir.DefaultOutputParam
					if len(lastNode.Outputs) > 0 {
						outputParam = lastNode.Outputs[0].Name
					}
					prevOutputHandle = ir.Handle{Node: lastNode.Key, Param: outputParam}
				}
			} else {
				// Output routing table: func -> { output1: target1, output2: target2 }
				newNodes, newEdges, ok := analyzeOutputRoutingTable(
					acontext.Child(ctx, c),
					*prevNode,
					prevOutputHandle,
					generateKey,
				)
				if !ok {
					return nil, nil, false
				}
				nodes = append(nodes, newNodes...)
				edges = append(edges, newEdges...)
				// After output routing, we may have multiple branches
				// The prevNode becomes nil since we can't chain directly after routing
				prevNode = nil
			}
		}
	}

	return nodes, edges, true
}

func analyzeExpressionNode(
	ctx acontext.Context[parser.IFlowNodeContext],
	generateKey generateKey,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	if channel := ctx.AST.ChannelIdentifier(); channel != nil {
		return analyzeChannelNode(acontext.Child(ctx, channel), generateKey)
	}
	if fn := ctx.AST.Function(); fn != nil {
		return analyzeFunctionNode(acontext.Child(ctx, fn), generateKey)
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return analyzeExpression(acontext.Child(ctx, expr))
	}
	return ir.Node{}, ir.Handle{}, ir.Handle{}, true
}

func analyzeChannelNode(
	ctx acontext.Context[parser.IChannelIdentifierContext],
	generateKey generateKey,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	nodeKey := generateKey("on")
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	chKey := uint32(sym.ID)
	// Channel nodes produce output of the channel's inner type (unwrap chan<T> to get T)
	outputType := sym.Type.Unwrap()
	n := ir.Node{
		Key:      nodeKey,
		Type:     "on",
		Channels: symbol.NewChannels(),
		Config:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
	}
	n.Channels.Read.Add(chKey)
	// Channel nodes have no inputs (they're sources), only outputs
	inputHandle := ir.Handle{Node: nodeKey, Param: ""}
	outputHandle := ir.Handle{Node: nodeKey, Param: ir.DefaultOutputParam}
	return n, inputHandle, outputHandle, true
}

func extractConfigValues(
	ctx acontext.Context[parser.IConfigValuesContext],
	config types.Params,
	node ir.Node,
) (types.Params, bool) {
	if ctx.AST == nil {
		return config, true
	}
	if named := ctx.AST.NamedConfigValues(); named != nil {
		for _, cv := range named.AllNamedConfigValue() {
			key := cv.IDENTIFIER().GetText()
			idx := config.GetIndex(key)
			config[idx].Value = getExpressionText(cv.Expression())
		}
	} else if anon := ctx.AST.AnonymousConfigValues(); anon != nil {
		for i, expr := range anon.AllExpression() {
			config[i].Value = getExpressionText(expr)
		}
	}
	for i, p := range config {
		if p.Type.Kind == types.KindChan {
			sym, err := ctx.Scope.Resolve(ctx, p.Value.(string))
			if err != nil {
				ctx.Diagnostics.AddError(err, nil)
				return nil, false
			}
			channelKey := uint32(sym.ID)
			config[i].Value = channelKey
			node.Channels.Read.Add(channelKey)
		}
	}
	return config, true
}

func analyzeFunctionNode(
	ctx acontext.Context[parser.IFunctionContext],
	generateKey generateKey,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	var (
		name = ctx.AST.IDENTIFIER().GetText()
		key  = generateKey(name)
	)
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	fnType := sym.Type
	if fnType.Kind != types.KindFunction {
		ctx.Diagnostics.AddError(fmt.Errorf("expected function type, got %s", fnType), nil)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	n := ir.Node{
		Key:      key,
		Type:     name,
		Channels: sym.Channels.Copy(),
		Config:   slices.Clone(sym.Type.Config),
		Outputs:  slices.Clone(sym.Type.Outputs),
		Inputs:   slices.Clone(sym.Type.Inputs),
	}
	var ok bool
	n.Config, ok = extractConfigValues(
		acontext.Child(ctx, ctx.AST.ConfigValues()),
		n.Config,
		n,
	)
	if !ok {
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	if args := ctx.AST.Arguments(); args != nil {
		if argList := args.ArgumentList(); argList != nil {
			for i, expr := range argList.AllExpression() {
				fnType.Inputs[i].Value = getExpressionText(expr)
			}
		}
	}

	// Determine input handle - use first input parameter name or default
	inputParam := ir.DefaultInputParam
	if len(n.Inputs) > 0 {
		inputParam = n.Inputs[0].Name
	}
	inputHandle := ir.Handle{Node: key, Param: inputParam}

	// Determine output handle - use first output parameter name or default
	outputParam := ir.DefaultOutputParam
	if len(n.Outputs) > 0 {
		outputParam = n.Outputs[0].Name
	}
	outputHandle := ir.Handle{Node: key, Param: outputParam}

	return n, inputHandle, outputHandle, true
}

func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) (ir.Node, ir.Handle, ir.Handle, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	n := ir.Node{
		Key:      sym.Name,
		Type:     sym.Name,
		Channels: symbol.NewChannels(),
	}
	// Expression nodes use default parameters
	inputHandle := ir.Handle{Node: sym.Name, Param: ir.DefaultInputParam}
	outputHandle := ir.Handle{Node: sym.Name, Param: ir.DefaultOutputParam}
	return n, inputHandle, outputHandle, true
}

func analyzeOutputRoutingTable(
	ctx acontext.Context[parser.IRoutingTableContext],
	sourceNode ir.Node,
	sourceHandle ir.Handle,
	generateKey generateKey,
) ([]ir.Node, []ir.Edge, bool) {
	var (
		nodes []ir.Node
		edges []ir.Edge
	)

	// Process each routing entry: outputName: targetNode(s)
	for _, entry := range ctx.AST.AllRoutingEntry() {
		outputName := entry.IDENTIFIER(0).GetText()

		// Validate that the source node has this output parameter
		if !sourceNode.Outputs.Has(outputName) {
			ctx.Diagnostics.AddError(
				fmt.Errorf("node '%s' does not have output '%s'", sourceNode.Key, outputName),
				entry,
			)
			return nil, nil, false
		}

		// Process flow nodes in this routing entry
		flowNodes := entry.AllFlowNode()
		if len(flowNodes) == 0 {
			continue
		}

		// Optional target parameter mapping (last identifier in entry)
		var targetParamName string
		if len(entry.AllIDENTIFIER()) > 1 {
			targetParamName = entry.IDENTIFIER(1).GetText()
		}

		// Analyze each flow node in the chain
		var prevOutputHandle ir.Handle
		for i, flowNode := range flowNodes {
			node, inputHandle, outputHandle, ok := analyzeExpressionNode(acontext.Child(ctx, flowNode), generateKey)
			if !ok {
				return nil, nil, false
			}

			// First node connects to source node's output
			if i == 0 {
				edges = append(edges, ir.Edge{
					Source: ir.Handle{Node: sourceNode.Key, Param: outputName},
					Target: inputHandle,
				})
			} else {
				// Chain subsequent nodes
				edges = append(edges, ir.Edge{
					Source: prevOutputHandle,
					Target: inputHandle,
				})
			}

			// If this is the last node and we have a target parameter mapping, override the input handle
			if i == len(flowNodes)-1 && targetParamName != "" {
				// Validate target parameter exists
				if !node.Inputs.Has(targetParamName) {
					ctx.Diagnostics.AddError(
						fmt.Errorf("node '%s' does not have input '%s'", node.Key, targetParamName),
						entry,
					)
					return nil, nil, false
				}
				// Update the last edge to use the mapped parameter
				edges[len(edges)-1].Target.Param = targetParamName
			}

			prevOutputHandle = outputHandle
			nodes = append(nodes, node)
		}
	}

	return nodes, edges, true
}

func analyzeInputRoutingTable(
	ctx acontext.Context[parser.IRoutingTableContext],
	generateKey generateKey,
) ([]ir.Node, []ir.Edge, bool) {
	// TODO: Implement input routing table
	// Input routing tables map sources to named input parameters
	// Example: { param1: source1, param2: source2 } -> func{}
	// This is more complex and less commonly used, so implementing as Phase 2.5

	ctx.Diagnostics.AddError(
		fmt.Errorf("input routing tables not yet implemented in text compiler"),
		ctx.AST,
	)
	return nil, nil, false
}

// getExpressionText extracts the text representation of an expression
// In a full implementation, this would properly serialize the expression AST
func getExpressionText(expr parser.IExpressionContext) string {
	if expr == nil {
		return ""
	}
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
