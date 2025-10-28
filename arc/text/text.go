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
// The analyzer uses a two-pass approach: first analyzing function declarations
// and building the symbol table, then processing flow statements to construct
// the execution graph of nodes and edges.
package text

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// Text represents Arc source code with its parsed AST.
type Text struct {
	Raw string                 `json:"raw" msgpack:"raw"`
	AST parser.IProgramContext `json:"-"`
}

// GenerateKey generates unique node keys for flow graph construction.
type GenerateKey = func(name string) string

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
				Config:   *c.Type.Config,
				Inputs:   *c.Type.Inputs,
				Outputs:  *c.Type.Outputs,
				Channels: c.Channels.Copy(),
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
	generateKey GenerateKey,
) ([]ir.Node, []ir.Edge, bool) {
	var (
		prevHandle ir.Handle
		edges      []ir.Edge
		nodes      []ir.Node
	)
	for i, flowNode := range ctx.AST.AllFlowNode() {
		node, handle, ok := analyzeExpressionNode(acontext.Child(ctx, flowNode), generateKey)
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

func analyzeExpressionNode(
	ctx acontext.Context[parser.IFlowNodeContext],
	generateKey GenerateKey,
) (ir.Node, ir.Handle, bool) {
	if channel := ctx.AST.ChannelIdentifier(); channel != nil {
		return analyzeChannelNode(acontext.Child(ctx, channel), generateKey)
	}
	if fn := ctx.AST.Function(); fn != nil {
		return analyzeFunctionNode(acontext.Child(ctx, fn), generateKey)
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return analyzeExpression(acontext.Child(ctx, expr))
	}
	return ir.Node{}, ir.Handle{}, true
}

func analyzeChannelNode(
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
		Channels:     symbol.NewChannels(),
	}
	n.Channels.Read.Add(chKey)
	h := ir.Handle{Node: nodeKey, Param: ir.DefaultOutputParam}
	return n, h, true
}

func extractConfigValues(
	ctx acontext.Context[parser.IConfigValuesContext],
	fnType types.Type,
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
			key, _ := fnType.Inputs.At(i)
			config[key] = getExpressionText(expr)
		}
	}
	for k, v := range config {
		t, ok := fnType.Config.Get(k)
		if !ok {
			panic("config key not found in function")
		}
		if t.Kind == types.KindChan {
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

func analyzeFunctionNode(
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
	fnType := sym.Type
	if fnType.Kind != types.KindFunction {
		ctx.Diagnostics.AddError(fmt.Errorf("expected function type, got %s", fnType), nil)
		return ir.Node{}, ir.Handle{}, false
	}
	n := ir.Node{
		Key:      key,
		Type:     name,
		Channels: sym.Channels.Copy(),
		Config:   *sym.Type.Config,
		Outputs:  *sym.Type.Outputs,
		Inputs:   *sym.Type.Inputs,
	}
	config, ok := extractConfigValues(
		acontext.Child(ctx, ctx.AST.ConfigValues()),
		fnType,
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
	n.ConfigValues = config
	h := ir.Handle{Node: key, Param: "input"}
	return n, h, true
}

func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) (ir.Node, ir.Handle, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return ir.Node{}, ir.Handle{}, false
	}
	n := ir.Node{
		Key:      sym.Name,
		Type:     sym.Name,
		Channels: symbol.NewChannels(),
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
