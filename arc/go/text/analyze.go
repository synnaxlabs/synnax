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
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
)

type keyGenerator struct {
	occurrences   map[string]int
	seqName       string
	stageName     string
	nextStageName string
}

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

func (kg *keyGenerator) generate(role, name string) string {
	base := role
	if name != "" {
		base = role + "_" + name
	}
	count := kg.occurrences[base]
	kg.occurrences[base]++
	return fmt.Sprintf("%s_%d", base, count)
}

func (kg *keyGenerator) entry(seqName, stageName string) string {
	return fmt.Sprintf("entry_%s_%s", seqName, stageName)
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

func analyzeStageRef(stageSym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	entryKey := kg.entry(stageSym.Parent.Name, stageSym.Name)
	return emptyNodeResult(ir.Handle{Node: entryKey, Param: "activate"}), true
}

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

func analyzeNextToken(
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

func analyzeFunctionNode(
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
		Channels: sym.Channels,
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

func analyzeExpression(
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
		Channels: sym.Channels.Copy(),
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
	}
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

// Analyze performs semantic analysis on parsed Arc code and builds the IR.
// Returns a partially complete IR even on errors for LSP support.
func Analyze(
	ctx_ context.Context,
	t Text,
	resolver symbol.Resolver,
) (ir.IR, *diagnostics.Diagnostics) {
	var (
		ctx = acontext.CreateRoot(ctx_, t.AST, resolver)
		i   = ir.IR{Symbols: ctx.Scope, TypeMap: ctx.TypeMap}
	)
	if !analyzer.AnalyzeProgram(ctx) {
		return i, ctx.Diagnostics
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

type flowChainProcessor struct {
	ctx                acontext.Context[parser.IFlowStatementContext]
	kg                 *keyGenerator
	totalFlowNodes     int
	currentIndex       int
	prevOutput         ir.Handle
	prevNode           *ir.Node
	lastOpIndex        int
	nodes              []ir.Node
	edges              []ir.Edge
	additionalTriggers []nodeResult
}

func newFlowChainProcessor(ctx acontext.Context[parser.IFlowStatementContext], kg *keyGenerator) *flowChainProcessor {
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
		return ir.Continuous
	}
	if opCtx, ok := children[p.lastOpIndex].(parser.IFlowOperatorContext); ok && opCtx.TRANSITION() != nil {
		return ir.OneShot
	}
	return ir.Continuous
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
		p.edges = append(p.edges, ir.Edge{Source: p.prevOutput, Target: result.input, Kind: p.edgeKind()})
	}

	// Handle additional triggers (for expressions with multiple channel references)
	if len(p.additionalTriggers) > 0 {
		for _, trigger := range p.additionalTriggers {
			p.edges = append(p.edges, ir.Edge{
				Source: trigger.output,
				Target: result.input,
				Kind:   ir.Continuous,
			})
		}
		p.additionalTriggers = nil
	}

	p.prevOutput = result.output
	p.prevNode = &result.node
	if result.node.Key != "" {
		p.nodes = append(p.nodes, result.node)
	}
	return true
}

func (p *flowChainProcessor) processRoutingTable(rt parser.IRoutingTableContext) bool {
	if p.prevNode == nil {
		p.ctx.Diagnostics.AddError(
			errors.Newf("input routing tables not yet implemented in text compiler"),
			p.ctx.AST,
		)
		return false
	}
	newNodes, newEdges, ok := analyzeOutputRoutingTable(acontext.Child(p.ctx, rt), *p.prevNode, p.prevOutput, p.kg)
	if !ok {
		return false
	}
	p.nodes = append(p.nodes, newNodes...)
	p.edges = append(p.edges, newEdges...)
	p.prevNode = nil
	return true
}

func analyzeFlow(ctx acontext.Context[parser.IFlowStatementContext], kg *keyGenerator) ([]ir.Node, []ir.Edge, bool) {
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
		ctx.Diagnostics.AddError(errors.Newf("flow statement requires at least two nodes"), ctx.AST)
		return nil, nil, false
	}
	return p.nodes, p.edges, true
}

func extractConfigValues(
	ctx acontext.Context[parser.IConfigValuesContext],
	config types.Params,
	node ir.Node,
) (types.Params, bool) {
	if ctx.AST == nil {
		return config, true
	}

	parseConfigExpr := func(expr parser.IExpressionContext, paramType types.Type, paramName string) (any, bool) {
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

		if !expression.IsLiteral(expr) {
			ctx.Diagnostics.AddError(
				fmt.Errorf("config value for '%s' must be a literal", paramName),
				expr,
			)
			return nil, false
		}

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
			ctx.Diagnostics.AddError(
				errors.Newf("node '%s' does not have output '%s'", sourceNode.Key, outputName),
				entry,
			)
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

func analyzeSequence(
	ctx acontext.Context[parser.ISequenceDeclarationContext],
	kg *keyGenerator,
) (ir.Sequence, []ir.Node, []ir.Edge, bool) {
	seqName := ctx.AST.IDENTIFIER().GetText()
	seq := ir.Sequence{Key: seqName}

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

func analyzeStage(
	ctx acontext.Context[parser.IStageDeclarationContext],
	seqName string,
	kg *keyGenerator,
) (ir.Stage, []ir.Node, []ir.Edge, bool) {
	stageName := ctx.AST.IDENTIFIER().GetText()
	stage := ir.Stage{Key: stageName}

	var nodes []ir.Node
	var edges []ir.Edge

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

		for _, n := range itemNodes {
			stage.Nodes = append(stage.Nodes, n.Key)
		}
	}

	return stage, nodes, edges, true
}
