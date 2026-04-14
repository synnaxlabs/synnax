// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/analyzer/authority"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl/stage"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

type contextFrame struct {
	seqName     string
	stepKey     string
	nextStepKey string
}

type keyGenerator struct {
	occurrences map[string]int
	stack       []contextFrame
}

func newKeyGenerator() *keyGenerator {
	return &keyGenerator{occurrences: make(map[string]int)}
}

func (kg *keyGenerator) push(seqName, stepKey, nextStepKey string) {
	kg.stack = append(kg.stack, contextFrame{
		seqName:     seqName,
		stepKey:     stepKey,
		nextStepKey: nextStepKey,
	})
}

func (kg *keyGenerator) pop() {
	if len(kg.stack) > 0 {
		kg.stack = kg.stack[:len(kg.stack)-1]
	}
}

func (kg *keyGenerator) top() contextFrame {
	if len(kg.stack) == 0 {
		return contextFrame{}
	}
	return kg.stack[len(kg.stack)-1]
}

func (kg *keyGenerator) seqName() string     { return kg.top().seqName }
func (kg *keyGenerator) stepKey() string     { return kg.top().stepKey }
func (kg *keyGenerator) nextStepKey() string { return kg.top().nextStepKey }
func (kg *keyGenerator) inSequence() bool    { return len(kg.stack) > 0 }

func (kg *keyGenerator) generate(role, name string) string {
	base := role
	if name != "" {
		base = role + "_" + name
	}
	count := kg.occurrences[base]
	kg.occurrences[base]++
	return fmt.Sprintf("%s_%d", base, count)
}

func (kg *keyGenerator) entry(seqName, stepKey string) string {
	return fmt.Sprintf("entry_%s_%s", seqName, stepKey)
}

type nodeResult struct {
	node   ir.Node
	input  ir.Handle
	output ir.Handle
}

func newNodeResult(node ir.Node, inputParam, outputParam string) nodeResult {
	return nodeResult{
		node:   node,
		input:  ir.Handle{Node: node.Key, Param: inputParam},
		output: ir.Handle{Node: node.Key, Param: outputParam},
	}
}

func emptyNodeResult(inputHandle ir.Handle) nodeResult {
	return nodeResult{input: inputHandle}
}

func firstInputParam(inputs types.Params) string {
	if len(inputs) > 0 {
		return inputs[0].Name
	}
	return ir.DefaultInputParam
}

func firstOutputParam(outputs types.Params) string {
	if len(outputs) > 0 {
		return outputs[0].Name
	}
	return ir.DefaultOutputParam
}

func analyzeFlowNode(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
	isSink bool,
) (nodeResult, bool) {
	if id := ctx.AST.Identifier(); id != nil {
		return analyzeIdentifierByRole(acontext.Child(ctx, id), kg, isSink)
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
	return nodeResult{}, true
}

func analyzeIdentifierByRole(
	ctx acontext.Context[parser.IIdentifierContext],
	kg *keyGenerator,
	isSink bool,
) (nodeResult, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, false
	}

	switch sym.Kind {
	case symbol.KindSequence:
		return analyzeSequenceRef(ctx, sym, kg)
	case symbol.KindStage:
		return analyzeStageRef(sym, kg)
	case symbol.KindGlobalConstant:
		return buildGlobalConstantNode(name, sym, kg)
	default:
		if isSink {
			return buildChannelWriteNode(name, sym, kg)
		}
		return buildChannelReadNode(name, sym, kg)
	}
}

func newStageTransition(seqName, stageName string, kg *keyGenerator) nodeResult {
	entryKey := kg.entry(seqName, stageName)
	return emptyNodeResult(ir.Handle{
		Node:  entryKey,
		Param: stage.EntryActivationParam,
	})
}

func analyzeSequenceRef(
	ctx acontext.Context[parser.IIdentifierContext],
	seqSym *symbol.Scope,
	kg *keyGenerator,
) (nodeResult, bool) {
	// For top-level stages (AST is IStageDeclarationContext), the step key
	// matches the sequence/stage name.
	if _, ok := seqSym.AST.(parser.IStageDeclarationContext); ok {
		return newStageTransition(seqSym.Name, seqSym.Name, kg), true
	}
	// For sequences, target the entry node of the first step. The step key
	// matches the named stage/sequence identifier when present, otherwise
	// the synthetic "step_N" key used by the text builder's prescan.
	seqDecl, ok := seqSym.AST.(parser.ISequenceDeclarationContext)
	if !ok || len(seqDecl.AllSequenceItem()) == 0 {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "sequence '%s' has no steps", seqSym.Name))
		return nodeResult{}, false
	}
	firstItem := seqDecl.AllSequenceItem()[0]
	stepKey := "step_0"
	if stageDecl := firstItem.StageDeclaration(); stageDecl != nil {
		if id := stageDecl.IDENTIFIER(); id != nil {
			stepKey = id.GetText()
		}
	} else if nestedSeq := firstItem.SequenceDeclaration(); nestedSeq != nil {
		if id := nestedSeq.IDENTIFIER(); id != nil {
			stepKey = id.GetText()
		}
	}
	return newStageTransition(seqSym.Name, stepKey, kg), true
}

func analyzeStageRef(stageSym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	return newStageTransition(stageSym.Parent.Name, stageSym.Name, kg), true
}

func buildChannelReadNode(name string, sym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	nodeKey := kg.generate("on", name)
	chKey := uint32(sym.ID)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "on",
		Channels: types.NewChannels(),
		Config:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: sym.Type.Unwrap()}},
	}
	n.Channels.Read[chKey] = sym.Name
	return newNodeResult(n, "", ir.DefaultOutputParam), true
}

func buildChannelWriteNode(name string, sym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	nodeKey := kg.generate("write", name)
	chKey := uint32(sym.ID)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "write",
		Channels: types.NewChannels(),
		Config:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Inputs:   types.Params{{Name: ir.DefaultInputParam, Type: sym.Type.Unwrap()}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
	}
	n.Channels.Write[chKey] = sym.Name
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

func buildGlobalConstantNode(
	name string,
	sym *symbol.Scope,
	kg *keyGenerator,
) (nodeResult, bool) {
	key := kg.generate("const", name)
	n := ir.Node{
		Key:      key,
		Type:     "constant",
		Channels: types.NewChannels(),
		Config:   types.Params{{Name: "value", Type: sym.Type, Value: sym.DefaultValue}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: sym.Type}},
	}
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

func analyzeNextToken(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
) (nodeResult, bool) {
	if !kg.inSequence() {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "'next' used outside of a sequence"))
		return nodeResult{}, false
	}
	if kg.nextStepKey() == "" {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"'next' in last stage '%s' has no next stage",
			kg.stepKey(),
		))
		return nodeResult{}, false
	}
	entryKey := kg.entry(kg.seqName(), kg.nextStepKey())
	return emptyNodeResult(ir.Handle{Node: entryKey, Param: "activate"}), true
}

func analyzeFunctionNode(
	ctx acontext.Context[parser.IFunctionContext],
	kg *keyGenerator,
) (nodeResult, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	key := kg.generate(name, "")
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, false
	}
	if sym.Type.Kind != types.KindFunction {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"expected function type, got %s",
			sym.Type,
		))
		return nodeResult{}, false
	}
	freshType := types.Freshen(sym.Type, key)
	n := ir.Node{
		Key:      key,
		Type:     name,
		Channels: sym.Channels.Copy(),
		Config:   slices.Clone(freshType.Config),
		Outputs:  slices.Clone(freshType.Outputs),
		Inputs:   slices.Clone(freshType.Inputs),
	}
	var ok bool
	n.Config, ok = extractConfigValues(acontext.Child(ctx, ctx.AST.ConfigValues()), n.Config, n, sym)
	if !ok {
		return nodeResult{}, false
	}
	return newNodeResult(n, firstInputParam(n.Inputs), firstOutputParam(n.Outputs)), true
}

func analyzeExpression(
	ctx acontext.Context[parser.IExpressionContext],
	kg *keyGenerator,
) (nodeResult, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, false
	}

	if sym.Kind == symbol.KindConstant {
		outputType := ctx.Constraints.ApplySubstitutions(sym.Type.Outputs[0].Type)
		literalCtx := parser.GetLiteral(ctx.AST)
		parsedValue, err := literal.Parse(literalCtx, outputType)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return nodeResult{}, false
		}
		key := kg.generate("const", "")
		n := ir.Node{
			Key:      key,
			Type:     "constant",
			Channels: types.NewChannels(),
			Config:   types.Params{{Name: "value", Type: outputType, Value: parsedValue.Value}},
			Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
		}
		return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
	}

	key := kg.generate(sym.Name, "")
	freshType := types.Freshen(sym.Type, key)
	outputType := ctx.Constraints.ApplySubstitutions(freshType.Outputs[0].Type)
	n := ir.Node{
		Key:      key,
		Type:     sym.Name,
		Channels: sym.Channels.Copy(),
		Inputs:   freshType.Inputs,
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
	}
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

// Analyze performs semantic analysis on parsed Arc code and builds the IR.
// Returns a partially complete IR even on errors for LSP support.
func Analyze(
	ctx context.Context,
	t Text,
	resolver symbol.Resolver,
) (ir.IR, *diagnostics.Diagnostics) {
	var (
		aCtx = acontext.CreateRoot(ctx, t.AST, resolver)
		i    = ir.IR{Symbols: aCtx.Scope, TypeMap: aCtx.TypeMap}
	)

	analyzer.AnalyzeProgram(aCtx)
	i.Authorities = authority.Analyze(aCtx)
	if !aCtx.Diagnostics.Ok() {
		return i, aCtx.Diagnostics
	}

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
	for _, item := range t.AST.AllTopLevelItem() {
		if flow := item.FlowStatement(); flow != nil {
			nodes, edges, ok := analyzeFlow(acontext.Child(aCtx, flow), kg)
			if !ok {
				return i, aCtx.Diagnostics
			}
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		} else if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			seq, nodes, edges, ok := analyzeSequence(acontext.Child(aCtx, seqDecl), kg)
			if !ok {
				return i, aCtx.Diagnostics
			}
			i.Root.Sequences = append(i.Root.Sequences, seq)
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		} else if stageDecl := item.StageDeclaration(); stageDecl != nil {
			seq, nodes, edges, ok := analyzeTopLevelStage(acontext.Child(aCtx, stageDecl), kg)
			if !ok {
				return i, aCtx.Diagnostics
			}
			i.Root.Sequences = append(i.Root.Sequences, seq)
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		}
	}
	if len(i.Nodes) > 0 {
		if !analyzer.ResolveNodeTypes(i.Nodes, i.Edges, aCtx.Constraints, aCtx.Diagnostics) {
			return i, aCtx.Diagnostics
		}
		strata, diag := stratifier.Stratify(ctx, i.Nodes, i.Edges, i.Root.Sequences, aCtx.Diagnostics)
		if diag != nil && !diag.Ok() {
			aCtx.Diagnostics = diag
			return i, aCtx.Diagnostics
		}
		i.Root.Strata = strata
	}
	return i, aCtx.Diagnostics
}

type flowChainProcessor struct {
	kg                 *keyGenerator
	prevNode           *ir.Node
	ctx                acontext.Context[parser.IFlowStatementContext]
	prevOutput         ir.Handle
	nodes              []ir.Node
	edges              []ir.Edge
	additionalTriggers []nodeResult
	totalFlowNodes     int
	currentIndex       int
	lastOpIndex        int
}

func newFlowChainProcessor(
	ctx acontext.Context[parser.IFlowStatementContext],
	kg *keyGenerator,
) *flowChainProcessor {
	var total int
	for _, child := range ctx.AST.GetChildren() {
		if _, ok := child.(parser.IFlowNodeContext); ok {
			total++
		}
	}
	return &flowChainProcessor{ctx: ctx, kg: kg, totalFlowNodes: total}
}

func (p *flowChainProcessor) edgeKind() ir.EdgeKind {
	children := p.ctx.AST.GetChildren()
	if p.lastOpIndex < 0 || p.lastOpIndex >= len(children) {
		return ir.EdgeKindContinuous
	}
	if opCtx, ok := children[p.lastOpIndex].(parser.IFlowOperatorContext); ok && opCtx.TRANSITION() != nil {
		return ir.EdgeKindConditional
	}
	return ir.EdgeKindContinuous
}

// injectImplicitTriggers creates channel read nodes for all channels referenced
// in an expression when that expression is the first node in a flow statement.
// This enables the shorthand syntax: `ox_pt_1 > 20 => do_something{}`
// which expands to: `ox_pt_1 -> ox_pt_1 > 20 => do_something{}`
func (p *flowChainProcessor) injectImplicitTriggers(expr parser.IExpressionContext) bool {
	sym, err := p.ctx.Scope.Root().GetChildByParserRule(expr)
	if err != nil || sym.Kind == symbol.KindConstant {
		return true // Constants don't need triggers
	}

	if len(sym.Channels.Read) == 0 {
		return true // No channels referenced
	}

	// Create trigger node for each channel
	for _, chName := range sym.Channels.Read {
		chanSym, err := p.ctx.Scope.Resolve(p.ctx, chName)
		if err != nil {
			continue
		}
		result, ok := buildChannelReadNode(chName, chanSym, p.kg)
		if !ok {
			return false
		}
		p.nodes = append(p.nodes, result.node)

		if p.prevNode == nil {
			p.prevOutput = result.output
			p.prevNode = &result.node
		} else {
			p.additionalTriggers = append(p.additionalTriggers, result)
		}
	}
	return true
}

func (p *flowChainProcessor) processFlowNode(flowNode parser.IFlowNodeContext) bool {
	p.currentIndex++
	isLast := p.currentIndex == p.totalFlowNodes
	isSink := isLast && flowNode.Identifier() != nil

	// Inject implicit triggers for expression as first node
	if p.currentIndex == 1 && p.prevNode == nil {
		if expr := flowNode.Expression(); expr != nil {
			if !p.injectImplicitTriggers(expr) {
				return false
			}
		}
	}

	result, ok := analyzeFlowNode(acontext.Child(p.ctx, flowNode), p.kg, isSink)
	if !ok {
		return false
	}
	if p.prevNode != nil {
		if len(p.prevNode.Outputs) == 0 {
			p.ctx.Diagnostics.Add(diagnostics.Errorf(
				flowNode,
				"function '%s' has no output to connect in flow chain",
				p.prevNode.Type,
			))
			return false
		}
		p.edges = append(p.edges, ir.Edge{
			Source: p.prevOutput,
			Target: result.input,
			Kind:   p.edgeKind(),
		})
	}

	// Handle additional triggers (for expressions with multiple channel references)
	if len(p.additionalTriggers) > 0 {
		for _, trigger := range p.additionalTriggers {
			p.edges = append(p.edges, ir.Edge{
				Source: trigger.output,
				Target: result.input,
				Kind:   ir.EdgeKindContinuous,
			})
		}
		p.additionalTriggers = nil
	}

	if len(result.node.Outputs) > 0 {
		p.prevOutput = result.output
	}
	p.prevNode = &result.node
	if result.node.Key != "" {
		p.nodes = append(p.nodes, result.node)
	}
	return true
}

func (p *flowChainProcessor) processRoutingTable(rt parser.IRoutingTableContext) bool {
	if p.prevNode == nil {
		p.ctx.Diagnostics.Add(diagnostics.Errorf(
			p.ctx.AST,
			"input routing tables not yet implemented in text compiler",
		))
		return false
	}
	newNodes, newEdges, ok := analyzeOutputRoutingTable(
		acontext.Child(p.ctx, rt),
		*p.prevNode,
		p.prevOutput,
		p.kg,
	)
	if !ok {
		return false
	}
	p.nodes = append(p.nodes, newNodes...)
	p.edges = append(p.edges, newEdges...)
	p.prevNode = nil
	return true
}

func analyzeFlow(
	ctx acontext.Context[parser.IFlowStatementContext],
	kg *keyGenerator,
) ([]ir.Node, []ir.Edge, bool) {
	p := newFlowChainProcessor(ctx, kg)
	for i, child := range ctx.AST.GetChildren() {
		switch c := child.(type) {
		case parser.IFlowNodeContext:
			if !p.processFlowNode(c) {
				return nil, nil, false
			}
		case parser.IFlowOperatorContext:
			p.lastOpIndex = i
		case parser.IRoutingTableContext:
			if !p.processRoutingTable(c) {
				return nil, nil, false
			}
		}
	}
	if len(p.edges) < 1 {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"flow statement requires at least two nodes",
		))
		return nil, nil, false
	}
	return p.nodes, p.edges, true
}

func extractConfigValues(
	ctx acontext.Context[parser.IConfigValuesContext],
	config types.Params,
	node ir.Node,
	fnSym *symbol.Scope,
) (types.Params, bool) {
	if ctx.AST == nil {
		return config, true
	}

	parseConfigExpr := func(
		expr parser.IExpressionContext,
		paramType types.Type,
		paramName string,
	) (any, bool) {
		if paramType.Kind == types.KindChan {
			channelName := parser.GetExpressionText(expr)
			sym, err := ctx.Scope.Resolve(ctx, channelName)
			if err != nil {
				ctx.Diagnostics.Add(diagnostics.Error(err, expr))
				return nil, false
			}
			if err := paramType.ChanDirection.CheckCompatibility(sym.Type.ChanDirection); err != nil {
				ctx.Diagnostics.Add(diagnostics.Error(err, expr))
				return nil, false
			}
			channelKey := uint32(sym.ID)
			symbol.ResolveConfigChannel(&node.Channels, fnSym, paramName, channelKey, sym.Name)
			return channelKey, true
		}

		if primary := parser.GetPrimaryExpression(expr); primary != nil {
			if id := primary.IDENTIFIER(); id != nil {
				sym, err := ctx.Scope.Resolve(ctx, id.GetText())
				if err != nil {
					ctx.Diagnostics.Add(diagnostics.Error(err, expr))
					return nil, false
				}
				if sym.Kind == symbol.KindGlobalConstant {
					return sym.DefaultValue, true
				}
			}
		}

		if !parser.IsLiteral(expr) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				expr,
				"config value for '%s' must be a literal or global constant",
				paramName,
			))
			return nil, false
		}

		literalCtx := parser.GetLiteral(expr)
		parsedValue, err := literal.Parse(literalCtx, paramType)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, expr))
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

	for _, entry := range ctx.AST.AllRoutingEntry() {
		outputName := entry.IDENTIFIER(0).GetText()
		if !sourceNode.Outputs.Has(outputName) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry,
				"node '%s' does not have output '%s'",
				sourceNode.Key,
				outputName,
			))
			return nil, nil, false
		}

		flowNodes := entry.AllFlowNode()
		if len(flowNodes) == 0 {
			continue
		}

		var targetParamName string
		if len(entry.AllIDENTIFIER()) > 1 {
			targetParamName = entry.IDENTIFIER(1).GetText()
		}

		sourceOutput := ir.Handle{Node: sourceNode.Key, Param: outputName}
		prevOutputHandle := sourceOutput
		for i, flowNode := range flowNodes {
			isLast := i == len(flowNodes)-1
			isSink := isLast && flowNode.Identifier() != nil

			result, ok := analyzeFlowNode(acontext.Child(ctx, flowNode), kg, isSink)
			if !ok {
				return nil, nil, false
			}

			edges = append(edges, ir.Edge{
				Source: prevOutputHandle,
				Target: result.input,
				Kind:   ir.EdgeKindContinuous,
			})

			if isLast && targetParamName != "" {
				if !result.node.Inputs.Has(targetParamName) {
					ctx.Diagnostics.Add(diagnostics.Errorf(
						entry,
						"node '%s' does not have input '%s'",
						result.node.Key,
						targetParamName,
					))
					return nil, nil, false
				}
				edges[len(edges)-1].Target.Param = targetParamName
			}

			if len(result.node.Outputs) > 0 {
				prevOutputHandle = result.output
			}
			if result.node.Key != "" {
				nodes = append(nodes, result.node)
			}
		}
	}

	return nodes, edges, true
}

// stepInfo collects metadata about a step for computing next-step keys.
type stepInfo struct {
	key  string
	item parser.ISequenceItemContext
}

func analyzeSequence(
	ctx acontext.Context[parser.ISequenceDeclarationContext],
	kg *keyGenerator,
) (ir.Sequence, []ir.Node, []ir.Edge, bool) {
	var seqScope *symbol.Scope
	if id := ctx.AST.IDENTIFIER(); id != nil {
		resolved, err := ctx.Scope.Resolve(ctx, id.GetText())
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return ir.Sequence{}, nil, nil, false
		}
		seqScope = resolved
	} else {
		resolved, err := ctx.Scope.GetChildByParserRule(ctx.AST)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return ir.Sequence{}, nil, nil, false
		}
		seqScope = resolved
	}
	seqName := seqScope.Name
	seq := ir.Sequence{Key: seqName}

	// Pre-scan items to compute step keys and next-step keys.
	items := ctx.AST.AllSequenceItem()
	var steps []stepInfo
	for i, item := range items {
		key := fmt.Sprintf("step_%d", i)
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			if id := stageDecl.IDENTIFIER(); id != nil {
				key = id.GetText()
			}
		}
		if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
			if id := nestedSeq.IDENTIFIER(); id != nil {
				key = id.GetText()
			}
		}
		steps = append(steps, stepInfo{key: key, item: item})
	}

	var allNodes []ir.Node
	var allEdges []ir.Edge

	for i, si := range steps {
		nextStepKey := ""
		if i+1 < len(steps) {
			nextStepKey = steps[i+1].key
		}

		item := si.item
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			kg.push(seqName, si.key, nextStepKey)
			stg, nodes, edges, ok := analyzeStage(
				acontext.Child(ctx, stageDecl).WithScope(seqScope),
				seqName,
				kg,
			)
			kg.pop()
			if !ok {
				return ir.Sequence{}, nil, nil, false
			}
			seq.Steps = append(seq.Steps, ir.Step{Key: si.key, Stage: &stg})
			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)
			continue
		}

		if flowStmt := item.FlowStatement(); flowStmt != nil {
			kg.push(seqName, si.key, nextStepKey)
			nodes, edges, ok := analyzeFlow(
				acontext.Child(ctx, flowStmt).WithScope(seqScope),
				kg,
			)
			kg.pop()
			if !ok {
				return ir.Sequence{}, nil, nil, false
			}
			var nodeKeys []string
			for _, n := range nodes {
				nodeKeys = append(nodeKeys, n.Key)
			}
			// Create entry node for this flow step.
			entryNode := ir.Node{
				Key:      kg.entry(seqName, si.key),
				Type:     stage.EntryNodeName,
				Channels: types.NewChannels(),
				Inputs:   stage.EntryNodeInputs,
			}
			allNodes = append(allNodes, entryNode)
			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)

			// Auto-wire: the last node's output fires the next step's entry node
			// via a one-shot edge (if there is a next step).
			if nextStepKey != "" && len(nodes) > 0 {
				lastNode := nodes[len(nodes)-1]
				if len(lastNode.Outputs) > 0 {
					nextEntryKey := kg.entry(seqName, nextStepKey)
					allEdges = append(allEdges, ir.Edge{
						Source: ir.Handle{Node: lastNode.Key, Param: firstOutputParam(lastNode.Outputs)},
						Target: ir.Handle{Node: nextEntryKey, Param: stage.EntryActivationParam},
						Kind:   ir.EdgeKindConditional,
					})
				}
			}

			seq.Steps = append(seq.Steps, ir.Step{
				Key:  si.key,
				Flow: &ir.Flow{Nodes: nodeKeys},
			})
			continue
		}

		if single := item.SingleInvocation(); single != nil {
			kg.push(seqName, si.key, nextStepKey)
			node, ok := analyzeSingleInvocation(
				acontext.Child(ctx, single).WithScope(seqScope),
				kg,
			)
			kg.pop()
			if !ok {
				return ir.Sequence{}, nil, nil, false
			}
			// Create entry node for this flow step.
			entryNode := ir.Node{
				Key:      kg.entry(seqName, si.key),
				Type:     stage.EntryNodeName,
				Channels: types.NewChannels(),
				Inputs:   stage.EntryNodeInputs,
			}
			allNodes = append(allNodes, entryNode)
			allNodes = append(allNodes, node)

			// Auto-wire: the node's output fires the next step's entry node.
			if nextStepKey != "" && len(node.Outputs) > 0 {
				nextEntryKey := kg.entry(seqName, nextStepKey)
				allEdges = append(allEdges, ir.Edge{
					Source: ir.Handle{Node: node.Key, Param: firstOutputParam(node.Outputs)},
					Target: ir.Handle{Node: nextEntryKey, Param: stage.EntryActivationParam},
					Kind:   ir.EdgeKindConditional,
				})
			}

			seq.Steps = append(seq.Steps, ir.Step{
				Key:  si.key,
				Flow: &ir.Flow{Nodes: []string{node.Key}},
			})
			continue
		}

		if nestedSeqDecl := item.SequenceDeclaration(); nestedSeqDecl != nil {
			kg.push(seqName, si.key, nextStepKey)
			nestedSeq, nodes, edges, ok := analyzeSequence(
				acontext.Child(ctx, nestedSeqDecl).WithScope(seqScope),
				kg,
			)
			kg.pop()
			if !ok {
				return ir.Sequence{}, nil, nil, false
			}
			seq.Steps = append(seq.Steps, ir.Step{Key: si.key, Sequence: &nestedSeq})
			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)
		}
	}

	return seq, allNodes, allEdges, true
}

func analyzeTopLevelStage(
	ctx acontext.Context[parser.IStageDeclarationContext],
	kg *keyGenerator,
) (ir.Sequence, []ir.Node, []ir.Edge, bool) {
	id := ctx.AST.IDENTIFIER()
	if id == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "top-level stage must have a name"))
		return ir.Sequence{}, nil, nil, false
	}
	stageName := id.GetText()
	kg.push(stageName, stageName, "")
	stg, nodes, edges, ok := analyzeStage(ctx, stageName, kg)
	kg.pop()
	if !ok {
		return ir.Sequence{}, nil, nil, false
	}
	seq := ir.Sequence{
		Key: stageName,
		Steps: []ir.Step{{
			Key:   stageName,
			Stage: &stg,
		}},
	}
	return seq, nodes, edges, true
}

func analyzeStage(
	ctx acontext.Context[parser.IStageDeclarationContext],
	seqName string,
	kg *keyGenerator,
) (ir.Stage, []ir.Node, []ir.Edge, bool) {
	stageName := ""
	if id := ctx.AST.IDENTIFIER(); id != nil {
		stageName = id.GetText()
	}
	var (
		stg   = ir.Stage{Key: stageName}
		nodes []ir.Node
		edges []ir.Edge
	)

	entryNode := ir.Node{
		Key:      kg.entry(seqName, kg.stepKey()),
		Type:     stage.EntryNodeName,
		Channels: types.NewChannels(),
		Inputs:   stage.EntryNodeInputs,
	}
	nodes = append(nodes, entryNode)

	stageBody := ctx.AST.StageBody()
	if stageBody == nil {
		return stg, nodes, edges, true
	}

	for _, item := range stageBody.AllStageItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			itemNodes, itemEdges, ok := analyzeFlow(acontext.Child(ctx, flowStmt), kg)
			if !ok {
				return ir.Stage{}, nil, nil, false
			}
			nodes = append(nodes, itemNodes...)
			edges = append(edges, itemEdges...)

			for _, n := range itemNodes {
				stg.Nodes = append(stg.Nodes, n.Key)
			}
			continue
		}
		if single := item.SingleInvocation(); single != nil {
			node, ok := analyzeSingleInvocation(acontext.Child(ctx, single), kg)
			if !ok {
				return ir.Stage{}, nil, nil, false
			}
			nodes = append(nodes, node)
			stg.Nodes = append(stg.Nodes, node.Key)
			continue
		}
		if nestedSeqDecl := item.SequenceDeclaration(); nestedSeqDecl != nil {
			// The inline sequence is registered as a child of this stage's
			// scope, not the parent sequence's scope. Look up the stage's own
			// scope so analyzeSequence can resolve the nested seq via parser
			// rule. Top-level stages don't have their own scope entry; in
			// that case keep ctx.Scope.
			seqCtx := acontext.Child(ctx, nestedSeqDecl)
			if stageScope, err := ctx.Scope.GetChildByParserRule(ctx.AST); err == nil {
				seqCtx = seqCtx.WithScope(stageScope)
			}
			subSeq, subNodes, subEdges, ok := analyzeSequence(seqCtx, kg)
			if !ok {
				return ir.Stage{}, nil, nil, false
			}
			nodes = append(nodes, subNodes...)
			edges = append(edges, subEdges...)
			stg.Sequences = append(stg.Sequences, subSeq)
			continue
		}
	}

	return stg, nodes, edges, true
}

func analyzeSingleInvocation(
	ctx acontext.Context[parser.ISingleInvocationContext],
	kg *keyGenerator,
) (ir.Node, bool) {
	if fn := ctx.AST.Function(); fn != nil {
		result, ok := analyzeFunctionNode(acontext.Child(ctx, fn), kg)
		if !ok {
			return ir.Node{}, false
		}
		return result.node, true
	}
	if expr := ctx.AST.Expression(); expr != nil {
		result, ok := analyzeExpression(acontext.Child(ctx, expr), kg)
		if !ok {
			return ir.Node{}, false
		}
		return result.node, true
	}
	return ir.Node{}, false
}
