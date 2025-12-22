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
	"slices"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// keyGenerator creates unique, semantically meaningful node keys.
// Keys follow the pattern: {role}_{name}_{occurrence} where name provides semantic context.
type keyGenerator struct {
	occurrences   map[string]int
	seqName       string
	stageName     string
	nextStageName string
}

// newKeyGenerator creates a new keyGenerator instance.
func newKeyGenerator() *keyGenerator {
	return &keyGenerator{occurrences: make(map[string]int)}
}

func (kg *keyGenerator) setStageContext(seqName, stageName, nextStageName string) {
	kg.seqName = seqName
	kg.stageName = stageName
	kg.nextStageName = nextStageName
}

func (kg *keyGenerator) clearStageContext() {
	kg.seqName = ""
	kg.stageName = ""
	kg.nextStageName = ""
}

// generate creates a unique key for a node.
// For nodes with semantic names (channels, functions): role_name_N
// For anonymous nodes: role_N
func (kg *keyGenerator) generate(role, name string) string {
	base := role
	if name != "" {
		base = role + "_" + name
	}
	count := kg.occurrences[base]
	kg.occurrences[base]++
	return fmt.Sprintf("%s_%d", base, count)
}

// entry creates a deterministic key for stage entries (no counter needed).
func (kg *keyGenerator) entry(seqName, stageName string) string {
	return fmt.Sprintf("entry_%s_%s", seqName, stageName)
}

// nodeResult bundles the output of node analysis functions.
type nodeResult struct {
	node   ir.Node
	input  ir.Handle
	output ir.Handle
}

// newNodeResult creates a nodeResult with handles derived from the node's key and params.
func newNodeResult(node ir.Node, inputParam, outputParam string) nodeResult {
	return nodeResult{
		node:   node,
		input:  ir.Handle{Node: node.Key, Param: inputParam},
		output: ir.Handle{Node: node.Key, Param: outputParam},
	}
}

// emptyNodeResult creates a result with only an input handle (for stage/sequence refs).
func emptyNodeResult(inputHandle ir.Handle) nodeResult {
	return nodeResult{input: inputHandle}
}

// firstInputParam returns the first input param name or the default.
func firstInputParam(inputs types.Params) string {
	if len(inputs) > 0 {
		return inputs[0].Name
	}
	return ir.DefaultInputParam
}

// firstOutputParam returns the first output param name or the default.
func firstOutputParam(outputs types.Params) string {
	if len(outputs) > 0 {
		return outputs[0].Name
	}
	return ir.DefaultOutputParam
}

// analyzeFlowNode dispatches to the appropriate analyzer based on node type and position.
func analyzeFlowNode(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
	isSink bool,
) (nodeResult, bool) {
	if id := ctx.AST.Identifier(); id != nil {
		return analyzeIdentifierByRole(acontext.Child(ctx, id), kg, isSink)
	}
	if fn := ctx.AST.Function(); fn != nil {
		return analyzeFunctionNodeV2(acontext.Child(ctx, fn), kg)
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return analyzeExpressionV2(acontext.Child(ctx, expr), kg)
	}
	if ctx.AST.NEXT() != nil {
		return analyzeNextTokenV2(ctx, kg)
	}
	return nodeResult{}, true
}

// analyzeIdentifierByRole handles identifiers based on symbol kind and position.
func analyzeIdentifierByRole(
	ctx acontext.Context[parser.IIdentifierContext],
	kg *keyGenerator,
	isSink bool,
) (nodeResult, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return nodeResult{}, false
	}

	switch sym.Kind {
	case symbol.KindSequence:
		return analyzeSequenceRef(ctx, sym, kg)
	case symbol.KindStage:
		return analyzeStageRef(sym, kg)
	default:
		if isSink {
			return buildChannelWriteNode(name, sym, kg)
		}
		return buildChannelReadNode(name, sym, kg)
	}
}

// analyzeSequenceRef returns a handle to a sequence's first stage entry.
func analyzeSequenceRef(
	ctx acontext.Context[parser.IIdentifierContext],
	seqSym *symbol.Scope,
	kg *keyGenerator,
) (nodeResult, bool) {
	firstStage, err := seqSym.FirstChildOfKind(symbol.KindStage)
	if err != nil {
		ctx.Diagnostics.AddError(errors.Newf("sequence '%s' has no stages", seqSym.Name), ctx.AST)
		return nodeResult{}, false
	}
	entryKey := kg.entry(seqSym.Name, firstStage.Name)
	return emptyNodeResult(ir.Handle{Node: entryKey, Param: "activate"}), true
}

// analyzeStageRef returns a handle to a stage's entry node.
func analyzeStageRef(stageSym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	entryKey := kg.entry(stageSym.Parent.Name, stageSym.Name)
	return emptyNodeResult(ir.Handle{Node: entryKey, Param: "activate"}), true
}

// buildChannelReadNode creates an "on" node for channel sources.
func buildChannelReadNode(name string, sym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	nodeKey := kg.generate("on", name)
	chKey := uint32(sym.ID)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "on",
		Channels: symbol.NewChannels(),
		Config:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: sym.Type.Unwrap()}},
	}
	n.Channels.Read[chKey] = sym.Name
	return newNodeResult(n, "", ir.DefaultOutputParam), true
}

// buildChannelWriteNode creates a "write" node for channel sinks.
func buildChannelWriteNode(name string, sym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	nodeKey := kg.generate("write", name)
	chKey := uint32(sym.ID)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "write",
		Channels: symbol.NewChannels(),
		Config:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Inputs:   types.Params{{Name: ir.DefaultInputParam, Type: sym.Type.Unwrap()}},
	}
	n.Channels.Write[chKey] = sym.Name
	return newNodeResult(n, ir.DefaultInputParam, ""), true
}

// analyzeNextTokenV2 handles the 'next' keyword in sequences.
func analyzeNextTokenV2(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
) (nodeResult, bool) {
	if kg.seqName == "" {
		ctx.Diagnostics.AddError(errors.New("'next' used outside of a sequence"), ctx.AST)
		return nodeResult{}, false
	}
	if kg.nextStageName == "" {
		ctx.Diagnostics.AddError(
			errors.Newf("'next' in last stage '%s' has no next stage", kg.stageName),
			ctx.AST,
		)
		return nodeResult{}, false
	}
	entryKey := kg.entry(kg.seqName, kg.nextStageName)
	return emptyNodeResult(ir.Handle{Node: entryKey, Param: "activate"}), true
}

// analyzeFunctionNodeV2 creates a function instance node.
func analyzeFunctionNodeV2(
	ctx acontext.Context[parser.IFunctionContext],
	kg *keyGenerator,
) (nodeResult, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	key := kg.generate(name, "")
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return nodeResult{}, false
	}
	if sym.Type.Kind != types.KindFunction {
		ctx.Diagnostics.AddError(errors.Newf("expected function type, got %s", sym.Type), nil)
		return nodeResult{}, false
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
	n.Config, ok = extractConfigValues(acontext.Child(ctx, ctx.AST.ConfigValues()), n.Config, n)
	if !ok {
		return nodeResult{}, false
	}
	return newNodeResult(n, firstInputParam(n.Inputs), firstOutputParam(n.Outputs)), true
}

// analyzeExpressionV2 creates constant or synthetic function nodes.
func analyzeExpressionV2(
	ctx acontext.Context[parser.IExpressionContext],
	kg *keyGenerator,
) (nodeResult, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return nodeResult{}, false
	}

	if sym.Kind == symbol.KindConstant {
		outputType := ctx.Constraints.ApplySubstitutions(sym.Type.Outputs[0].Type)
		literalCtx := expression.GetLiteral(ctx.AST)
		parsedValue, err := literal.Parse(literalCtx, outputType)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return nodeResult{}, false
		}
		key := kg.generate("const", "")
		n := ir.Node{
			Key:      key,
			Type:     "constant",
			Channels: symbol.NewChannels(),
			Config:   types.Params{{Name: "value", Type: outputType, Value: parsedValue.Value}},
			Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
		}
		return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
	}

	key := kg.generate(sym.Name, "")
	outputType := ctx.Constraints.ApplySubstitutions(sym.Type.Outputs[0].Type)
	n := ir.Node{
		Key:      key,
		Type:     sym.Name,
		Channels: symbol.NewChannels(),
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
	}
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
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
			exprDecl, ok := c.AST.(parser.IExpressionContext)
			if ok {
				bodyAst = exprDecl
			}
			i.Functions = append(i.Functions, ir.Function{
				Key:      c.Name,
				Body:     ir.Body{Raw: bodyAst.GetText(), AST: bodyAst},
				Config:   c.Type.Config,
				Inputs:   c.Type.Inputs,
				Outputs:  c.Type.Outputs,
				Channels: c.Channels,
			})
		}
	}

	kg := newKeyGenerator()

	// Step 3: Process Flow Nodes and Statements to Build Nodes/Edges
	for _, item := range t.AST.AllTopLevelItem() {
		if flow := item.FlowStatement(); flow != nil {
			nodes, edges, ok := analyzeFlow(acontext.Child(ctx, flow), kg)
			if !ok {
				return i, ctx.Diagnostics
			}
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		} else if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			seq, nodes, edges, ok := analyzeSequence(acontext.Child(ctx, seqDecl), kg)
			if !ok {
				return i, ctx.Diagnostics
			}
			i.Sequences = append(i.Sequences, seq)
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		}
	}

	// Step 4: Calculate execution stratification for deterministic reactive execution
	if len(i.Nodes) > 0 {
		strata, diag := stratifier.Stratify(ctx_, i.Nodes, i.Edges, i.Sequences, ctx.Diagnostics)
		if diag != nil && !diag.Ok() {
			ctx.Diagnostics = diag
			return i, ctx.Diagnostics
		}
		i.Strata = strata
	}

	return i, ctx.Diagnostics
}

// getFlowOperatorKind determines EdgeKind by examining the operator token at the specified index.
// Operators appear at odd indices when iterating through flow statement children:
// [node0, op0, node1, op1, node2, ...]
func getFlowOperatorKind(ctx acontext.Context[parser.IFlowStatementContext], operatorIndex int) ir.EdgeKind {
	children := ctx.AST.GetChildren()
	if operatorIndex < 0 || operatorIndex >= len(children) {
		return ir.Continuous // Safe default
	}

	if opCtx, ok := children[operatorIndex].(parser.IFlowOperatorContext); ok {
		if opCtx.TRANSITION() != nil {
			return ir.OneShot
		}
	}
	return ir.Continuous
}

func analyzeFlow(
	ctx acontext.Context[parser.IFlowStatementContext],
	kg *keyGenerator,
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
		children := ctx.AST.GetChildren()
		flowNodeIndex, operatorIndex := 0, 0

		var totalFlowNodes int
		for _, child := range children {
			if _, ok := child.(parser.IFlowNodeContext); ok {
				totalFlowNodes++
			}
		}

		for _, child := range children {
			if flowNode, ok := child.(parser.IFlowNodeContext); ok {
				flowNodeIndex++
				isLast := flowNodeIndex == totalFlowNodes
				isSink := isLast && flowNode.Identifier() != nil

				result, ok := analyzeFlowNode(acontext.Child(ctx, flowNode), kg, isSink)
				if !ok {
					return nil, nil, false
				}

				if flowNodeIndex > 1 && prevNode != nil {
					edgeKind := getFlowOperatorKind(ctx, operatorIndex-1)
					edges = append(edges, ir.Edge{Source: prevOutputHandle, Target: result.input, Kind: edgeKind})
				}

				prevOutputHandle = result.output
				prevNode = &result.node
				if result.node.Key != "" {
					nodes = append(nodes, result.node)
				}
				operatorIndex += 2
			}
		}

		if len(edges) < 1 {
			ctx.Diagnostics.AddError(errors.Newf("flow statement requires at least two nodes"), ctx.AST)
			return nil, nil, false
		}
		return nodes, edges, true
	}

	// Flow chain with routing tables
	children := ctx.AST.GetChildren()
	var lastOperatorIndex, totalFlowNodes, currentFlowNodeIndex int
	for _, child := range children {
		if _, ok := child.(parser.IFlowNodeContext); ok {
			totalFlowNodes++
		}
	}

	for i, child := range children {
		switch c := child.(type) {
		case parser.IFlowNodeContext:
			currentFlowNodeIndex++
			isLast := currentFlowNodeIndex == totalFlowNodes
			isSink := isLast && c.Identifier() != nil

			result, ok := analyzeFlowNode(acontext.Child(ctx, c), kg, isSink)
			if !ok {
				return nil, nil, false
			}
			if prevNode != nil {
				edgeKind := getFlowOperatorKind(ctx, lastOperatorIndex)
				edges = append(edges, ir.Edge{Source: prevOutputHandle, Target: result.input, Kind: edgeKind})
			}
			prevOutputHandle = result.output
			prevNode = &result.node
			if result.node.Key != "" {
				nodes = append(nodes, result.node)
			}

		case parser.IFlowOperatorContext:
			lastOperatorIndex = i

		case parser.IRoutingTableContext:
			if prevNode == nil {
				newNodes, newEdges, ok := analyzeInputRoutingTable(acontext.Child(ctx, c), kg)
				if !ok {
					return nil, nil, false
				}
				nodes = append(nodes, newNodes...)
				edges = append(edges, newEdges...)
				if len(newNodes) > 0 {
					lastNode := newNodes[len(newNodes)-1]
					prevNode = &lastNode
					prevOutputHandle = ir.Handle{Node: lastNode.Key, Param: firstOutputParam(lastNode.Outputs)}
				}
			} else {
				newNodes, newEdges, ok := analyzeOutputRoutingTable(acontext.Child(ctx, c), *prevNode, prevOutputHandle, kg)
				if !ok {
					return nil, nil, false
				}
				nodes = append(nodes, newNodes...)
				edges = append(edges, newEdges...)
				prevNode = nil
			}
		}
	}

	return nodes, edges, true
}

func analyzeExpressionNode(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	if id := ctx.AST.Identifier(); id != nil {
		return analyzeChannelNode(acontext.Child(ctx, id), kg)
	}
	if fn := ctx.AST.Function(); fn != nil {
		return analyzeFunctionNode(acontext.Child(ctx, fn), kg)
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return analyzeExpression(acontext.Child(ctx, expr), kg)
	}
	if ctx.AST.NEXT() != nil {
		return analyzeNextToken(ctx, kg)
	}
	return ir.Node{}, ir.Handle{}, ir.Handle{}, true
}

func analyzeNextToken(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	if kg.seqName == "" {
		ctx.Diagnostics.AddError(errors.New("'next' used outside of a sequence"), ctx.AST)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	if kg.nextStageName == "" {
		ctx.Diagnostics.AddError(
			errors.Newf("'next' in last stage '%s' has no next stage", kg.stageName),
			ctx.AST,
		)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	entryKey := kg.entry(kg.seqName, kg.nextStageName)
	inputHandle := ir.Handle{Node: entryKey, Param: "activate"}
	return ir.Node{}, inputHandle, ir.Handle{}, true
}

func analyzeExpressionNodeAsSink(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	if id := ctx.AST.Identifier(); id != nil {
		return analyzeIdentifierAsSink(acontext.Child(ctx, id), kg)
	}
	// For non-channel nodes, use normal analysis (functions can be sinks too)
	return analyzeExpressionNode(ctx, kg)
}

func analyzeChannelNode(
	ctx acontext.Context[parser.IIdentifierContext],
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	nodeKey := kg.generate("on", name)
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
	n.Channels.Read[chKey] = sym.Name
	// Channel nodes have no inputs (they're sources), only outputs
	inputHandle := ir.Handle{Node: nodeKey, Param: ""}
	outputHandle := ir.Handle{Node: nodeKey, Param: ir.DefaultOutputParam}
	return n, inputHandle, outputHandle, true
}

// analyzeIdentifierAsSink handles identifiers in sink position (end of flow chain).
// The identifier may resolve to a channel (creates a write node) or a sequence
// (connects to the sequence's first stage entry node).
func analyzeIdentifierAsSink(
	ctx acontext.Context[parser.IIdentifierContext],
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}

	// If the target is a sequence, connect to its first stage's entry node
	if sym.Kind == symbol.KindSequence {
		return analyzeSequenceNodeAsSink(ctx, sym, kg)
	}

	// If the target is a stage, connect to that stage's entry node
	if sym.Kind == symbol.KindStage {
		return analyzeStageNodeAsSink(ctx, sym, kg)
	}

	// Otherwise, treat as a channel write node
	nodeKey := kg.generate("write", name)
	chKey := uint32(sym.ID)
	// Channel sink nodes consume input of the channel's inner type (unwrap chan<T> to get T)
	inputType := sym.Type.Unwrap()
	n := ir.Node{
		Key:      nodeKey,
		Type:     "write",
		Channels: symbol.NewChannels(),
		Config:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Inputs:   types.Params{{Name: ir.DefaultInputParam, Type: inputType}},
	}
	n.Channels.Write[chKey] = sym.Name
	// Write nodes have inputs (sink), and no outputs
	inputHandle := ir.Handle{Node: nodeKey, Param: ir.DefaultInputParam}
	outputHandle := ir.Handle{Node: nodeKey, Param: ""}
	return n, inputHandle, outputHandle, true
}

// analyzeStageNodeAsSink handles stage identifiers in sink position.
// Instead of creating a write node, it returns a handle to the stage's
// entry node's "activate" input.
func analyzeStageNodeAsSink(
	ctx acontext.Context[parser.IIdentifierContext],
	stageSym *symbol.Scope,
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	stageName := stageSym.Name
	// Get the sequence name from the stage's parent (the sequence scope)
	seqName := stageSym.Parent.Name

	// Build the entry node key using the same pattern as analyzeStage
	entryKey := kg.entry(seqName, stageName)

	// Return a reference to the existing entry node (no new node created)
	// The input handle points to the stage entry's "activate" input
	inputHandle := ir.Handle{Node: entryKey, Param: "activate"}
	// Stage sinks have no meaningful output
	outputHandle := ir.Handle{Node: entryKey, Param: ""}

	// Return an empty node - we don't create a new node, just return the handle
	// to the existing stage entry node that was already created by analyzeStage
	return ir.Node{}, inputHandle, outputHandle, true
}

// analyzeSequenceNodeAsSink handles sequence identifiers in sink position.
// Instead of creating a write node, it returns a handle to the sequence's
// first stage entry node's "activate" input.
func analyzeSequenceNodeAsSink(
	ctx acontext.Context[parser.IIdentifierContext],
	seqSym *symbol.Scope,
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	seqName := seqSym.Name

	// Get the first stage from the sequence's children
	firstStage, err := seqSym.FirstChildOfKind(symbol.KindStage)
	if err != nil {
		ctx.Diagnostics.AddError(
			errors.Newf("sequence '%s' has no stages", seqName),
			ctx.AST,
		)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	stageName := firstStage.Name

	// Build the entry node key using the same pattern as analyzeStage
	entryKey := kg.entry(seqName, stageName)

	// Return a reference to the existing entry node (no new node created)
	// The input handle points to the stage entry's "activate" input
	inputHandle := ir.Handle{Node: entryKey, Param: "activate"}
	// Sequence sinks have no meaningful output
	outputHandle := ir.Handle{Node: entryKey, Param: ""}

	// Return an empty node - we don't create a new node, just return the handle
	// to the existing stage entry node that was already created by analyzeStage
	return ir.Node{}, inputHandle, outputHandle, true
}

func extractConfigValues(
	ctx acontext.Context[parser.IConfigValuesContext],
	config types.Params,
	node ir.Node,
) (types.Params, bool) {
	if ctx.AST == nil {
		return config, true
	}

	// parseConfigExpr parses a configuration value expression.
	// For channels, it resolves to a uint32 channel ID.
	// For other types, it parses the literal value.
	parseConfigExpr := func(expr parser.IExpressionContext, paramType types.Type, paramName string) (any, bool) {
		// Channel types: resolve to uint32 ID
		if paramType.Kind == types.KindChan {
			channelName := getExpressionText(expr)
			sym, err := ctx.Scope.Resolve(ctx, channelName)
			if err != nil {
				ctx.Diagnostics.AddError(err, expr)
				return nil, false
			}
			channelKey := uint32(sym.ID)
			node.Channels.Read[channelKey] = sym.Name
			return channelKey, true
		}

		// Must be a pure literal (no operators, function calls, etc.)
		if !expression.IsLiteral(expr) {
			ctx.Diagnostics.AddError(
				fmt.Errorf("config value for '%s' must be a literal", paramName),
				expr,
			)
			return nil, false
		}

		// Parse the literal with the expected parameter type
		literalCtx := expression.GetLiteral(expr)
		parsedValue, err := literal.Parse(literalCtx, paramType)
		if err != nil {
			ctx.Diagnostics.AddError(err, expr)
			return nil, false
		}
		return parsedValue.Value, true
	}

	if named := ctx.AST.NamedConfigValues(); named != nil {
		for _, cv := range named.AllNamedConfigValue() {
			key := cv.IDENTIFIER().GetText()
			idx := config.GetIndex(key)
			if expr := cv.Expression(); expr != nil {
				value, ok := parseConfigExpr(expr, config[idx].Type, key)
				if !ok {
					return nil, false
				}
				config[idx].Value = value
			}
		}
	} else if anon := ctx.AST.AnonymousConfigValues(); anon != nil {
		for i, expr := range anon.AllExpression() {
			value, ok := parseConfigExpr(expr, config[i].Type, fmt.Sprintf("position %d", i))
			if !ok {
				return nil, false
			}
			config[i].Value = value
		}
	}

	return config, true
}

func analyzeFunctionNode(
	ctx acontext.Context[parser.IFunctionContext],
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	var (
		name = ctx.AST.IDENTIFIER().GetText()
		key  = kg.generate(name, "")
	)
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, nil)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}
	fnType := sym.Type
	if fnType.Kind != types.KindFunction {
		ctx.Diagnostics.AddError(errors.Newf("expected function type, got %s", fnType), nil)
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

func analyzeExpression(
	ctx acontext.Context[parser.IExpressionContext],
	kg *keyGenerator,
) (ir.Node, ir.Handle, ir.Handle, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return ir.Node{}, ir.Handle{}, ir.Handle{}, false
	}

	// Constants (pure literals) become constant IR nodes
	if sym.Kind == symbol.KindConstant {
		// Get the resolved output type from the symbol (after type unification)
		outputType := sym.Type.Outputs[0].Type
		resolvedType := ctx.Constraints.ApplySubstitutions(outputType)

		// Parse the literal value with the resolved type
		literalCtx := expression.GetLiteral(ctx.AST)
		parsedValue, err := literal.Parse(literalCtx, resolvedType)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return ir.Node{}, ir.Handle{}, ir.Handle{}, false
		}

		key := kg.generate("const", "")
		n := ir.Node{
			Key:      key,
			Type:     "constant",
			Channels: symbol.NewChannels(),
			Config:   types.Params{{Name: "value", Type: resolvedType, Value: parsedValue.Value}},
			Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: resolvedType}},
		}
		inputHandle := ir.Handle{Node: key, Param: ir.DefaultInputParam}
		outputHandle := ir.Handle{Node: key, Param: ir.DefaultOutputParam}
		return n, inputHandle, outputHandle, true
	}

	// Non-constant expressions (e.g., inline functions)
	key := kg.generate(sym.Name, "")
	outputType := ctx.Constraints.ApplySubstitutions(sym.Type.Outputs[0].Type)
	n := ir.Node{
		Key:      key,
		Type:     sym.Name,
		Channels: symbol.NewChannels(),
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
	}
	inputHandle := ir.Handle{Node: key, Param: ir.DefaultInputParam}
	outputHandle := ir.Handle{Node: key, Param: ir.DefaultOutputParam}
	return n, inputHandle, outputHandle, true
}

func analyzeOutputRoutingTable(
	ctx acontext.Context[parser.IRoutingTableContext],
	sourceNode ir.Node,
	sourceHandle ir.Handle,
	kg *keyGenerator,
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
				errors.Newf("node '%s' does not have output '%s'", sourceNode.Key, outputName),
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

		var prevOutputHandle ir.Handle
		for i, flowNode := range flowNodes {
			isLast := i == len(flowNodes)-1
			isSink := isLast && flowNode.Identifier() != nil

			result, ok := analyzeFlowNode(acontext.Child(ctx, flowNode), kg, isSink)
			if !ok {
				return nil, nil, false
			}

			if i == 0 {
				edges = append(edges, ir.Edge{
					Source: ir.Handle{Node: sourceNode.Key, Param: outputName},
					Target: result.input,
					Kind:   ir.Continuous,
				})
			} else {
				edges = append(edges, ir.Edge{Source: prevOutputHandle, Target: result.input, Kind: ir.Continuous})
			}

			if isLast && targetParamName != "" {
				if !result.node.Inputs.Has(targetParamName) {
					ctx.Diagnostics.AddError(
						errors.Newf("node '%s' does not have input '%s'", result.node.Key, targetParamName),
						entry,
					)
					return nil, nil, false
				}
				edges[len(edges)-1].Target.Param = targetParamName
			}

			prevOutputHandle = result.output
			if result.node.Key != "" {
				nodes = append(nodes, result.node)
			}
		}
	}

	return nodes, edges, true
}

func analyzeInputRoutingTable(
	ctx acontext.Context[parser.IRoutingTableContext],
	kg *keyGenerator,
) ([]ir.Node, []ir.Edge, bool) {
	// TODO: Implement input routing table
	// Input routing tables map sources to named input parameters
	// Example: { param1: source1, param2: source2 } -> func{}
	// This is more complex and less commonly used, so implementing as Phase 2.5

	ctx.Diagnostics.AddError(
		errors.Newf("input routing tables not yet implemented in text compiler"),
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

// analyzeSequence processes a sequence declaration and builds the IR representation.
// It creates a Sequence with embedded Stages, stage entry nodes, and all edges.
func analyzeSequence(
	ctx acontext.Context[parser.ISequenceDeclarationContext],
	kg *keyGenerator,
) (ir.Sequence, []ir.Node, []ir.Edge, bool) {
	seqName := ctx.AST.IDENTIFIER().GetText()
	seq := ir.Sequence{Key: seqName}

	// Resolve the sequence scope to access stage symbols
	seqScope, err := ctx.Scope.Resolve(ctx, seqName)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return ir.Sequence{}, nil, nil, false
	}

	var allNodes []ir.Node
	var allEdges []ir.Edge

	stageDecls := ctx.AST.AllStageDeclaration()
	for i, stageDecl := range stageDecls {
		stageName := stageDecl.IDENTIFIER().GetText()
		nextStageName := ""
		if i+1 < len(stageDecls) {
			nextStageName = stageDecls[i+1].IDENTIFIER().GetText()
		}
		kg.setStageContext(seqName, stageName, nextStageName)
		// Pass the sequence scope so stage transitions can resolve stage names
		stage, nodes, edges, ok := analyzeStage(
			acontext.Child(ctx, stageDecl).WithScope(seqScope),
			seqName,
			kg,
		)
		kg.clearStageContext()
		if !ok {
			return ir.Sequence{}, nil, nil, false
		}
		seq.Stages = append(seq.Stages, stage)
		allNodes = append(allNodes, nodes...)
		allEdges = append(allEdges, edges...)
	}

	return seq, allNodes, allEdges, true
}

// analyzeStage processes a stage declaration and builds nodes/edges for it.
// Creates a stage entry node and processes all stage items (flows).
func analyzeStage(
	ctx acontext.Context[parser.IStageDeclarationContext],
	seqName string,
	kg *keyGenerator,
) (ir.Stage, []ir.Node, []ir.Edge, bool) {
	stageName := ctx.AST.IDENTIFIER().GetText()
	stage := ir.Stage{Key: stageName}

	var nodes []ir.Node
	var edges []ir.Edge

	// Create stage entry node (deterministic key, no counter)
	// Note: entry node is NOT added to stage.Nodes because it must be able to run
	// before the stage is active - it's the mechanism by which the stage activates.
	entryKey := kg.entry(seqName, stageName)
	entryNode := ir.Node{
		Key:      entryKey,
		Type:     "stage_entry",
		Channels: symbol.NewChannels(),
		Inputs: types.Params{
			{Name: "activate", Type: types.U8()},
		},
	}
	nodes = append(nodes, entryNode)

	// Process stage body
	stageBody := ctx.AST.StageBody()
	if stageBody == nil {
		return stage, nodes, edges, true
	}

	for _, item := range stageBody.AllStageItem() {
		flowStmt := item.FlowStatement()
		if flowStmt == nil {
			continue
		}
		itemNodes, itemEdges, ok := analyzeFlow(acontext.Child(ctx, flowStmt), kg)
		if !ok {
			return ir.Stage{}, nil, nil, false
		}
		nodes = append(nodes, itemNodes...)
		edges = append(edges, itemEdges...)

		// Track node keys in stage
		for _, n := range itemNodes {
			stage.Nodes = append(stage.Nodes, n.Key)
		}
	}

	return stage, nodes, edges, true
}
