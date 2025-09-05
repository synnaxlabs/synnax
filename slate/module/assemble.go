// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package module

import (
	"fmt"

	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
)

type assembler struct {
	tasks      []Task
	functions  []Function
	nodes      map[string]*Node
	edges      []Edge
	scope      *symbol.Scope
	counter    int
	wasmModule []byte
}

// Assemble builds a complete Module from the parsed program and analyzed symbol scope
func Assemble(
	program parser.IProgramContext,
	analysisResult analyzer.Result,
	wasmModule []byte,
) (*Module, error) {
	a := &assembler{
		tasks:      []Task{},
		functions:  []Function{},
		nodes:      make(map[string]*Node),
		edges:      []Edge{},
		scope:      analysisResult.Symbols,
		counter:    0,
		wasmModule: wasmModule,
	}

	// First pass: collect task and function definitions
	for _, item := range program.AllTopLevelItem() {
		if taskDecl := item.TaskDeclaration(); taskDecl != nil {
			a.extractTask(taskDecl)
		} else if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			a.extractFunction(funcDecl)
		}
	}

	// Second pass: process flow statements to build nodes and edges
	for _, item := range program.AllTopLevelItem() {
		if flow := item.FlowStatement(); flow != nil {
			a.processFlow(flow)
		}
	}

	return a.buildModule(), nil
}

func (a *assembler) buildModule() *Module {
	nodes := make([]Node, 0, len(a.nodes))
	for _, node := range a.nodes {
		nodes = append(nodes, *node)
	}
	return &Module{
		Tasks:     a.tasks,
		Functions: a.functions,
		Nodes:     nodes,
		Edges:     a.edges,
		Wasm:      a.wasmModule,
	}
}

func (a *assembler) extractTask(taskDecl parser.ITaskDeclarationContext) {
	name := taskDecl.IDENTIFIER().GetText()
	sym, _ := a.scope.Resolve(name)
	if sym == nil || sym.Symbol == nil {
		return
	}
	taskType, ok := sym.Type.(types.Task)
	if !ok {
		return
	}
	// Convert types.Task to graph.Task
	task := Task{
		Key:      name,
		Config:   make(map[string]string),
		Params:   make(map[string]string),
		Stateful: make(map[string]Variable),
	}
	for key, item := range taskType.Config.Iter() {
		task.Config[key] = item.String()
	}
	for key, item := range taskType.Params.Iter() {
		task.Params[key] = item.String()
	}
	if taskType.Return != nil {
		task.Returns = taskType.Return.String()
	}
	a.tasks = append(a.tasks, task)
}

func (a *assembler) extractFunction(funcDecl parser.IFunctionDeclarationContext) {
	name := funcDecl.IDENTIFIER().GetText()
	sym, _ := a.scope.Resolve(name)
	if sym == nil || sym.Symbol == nil {
		return
	}
	funcType, ok := sym.Type.(types.Function)
	if !ok {
		return
	}
	function := Function{Key: name, Params: make(map[string]string)}
	for key, item := range funcType.Params.Iter() {
		function.Params[key] = item.String()
	}
	if funcType.Return != nil {
		function.Returns = funcType.Return.String()
	}
	a.functions = append(a.functions, function)
}

func (a *assembler) processFlow(flow parser.IFlowStatementContext) {
	var prevHandle *Handle
	for i, flowNode := range flow.AllFlowNode() {
		handle := a.processFlowNode(flowNode)
		if i > 0 && prevHandle != nil && handle != nil {
			a.edges = append(a.edges, Edge{
				Source: *prevHandle,
				Target: *handle,
			})
		}
		prevHandle = handle
	}
}

func (a *assembler) processFlowNode(node parser.IFlowNodeContext) *Handle {
	if channel := node.ChannelIdentifier(); channel != nil {
		return a.processChannel(channel)
	}
	if task := node.TaskInvocation(); task != nil {
		return a.processTask(task)
	}
	if expr := node.Expression(); expr != nil {
		return a.processExpression(expr)
	}
	return nil
}

func (a *assembler) processChannel(channel parser.IChannelIdentifierContext) *Handle {
	name := channel.IDENTIFIER().GetText()
	nodeKey := a.generateKey("on")
	a.nodes[nodeKey] = &Node{
		Key:  nodeKey,
		Type: "on",
		Config: map[string]any{
			"channel": name,
		},
	}
	return &Handle{Node: nodeKey, Param: "output"}
}

func (a *assembler) processTask(task parser.ITaskInvocationContext) *Handle {
	name := task.IDENTIFIER().GetText()
	nodeKey := a.generateKey(name)
	config := make(map[string]any)
	if configValues := task.ConfigValues(); configValues != nil {
		if named := configValues.NamedConfigValues(); named != nil {
			for _, cv := range named.AllNamedConfigValue() {
				key := cv.IDENTIFIER().GetText()
				config[key] = getExpressionText(cv.Expression())
			}
		} else if anon := configValues.AnonymousConfigValues(); anon != nil {
			for i, expr := range anon.AllExpression() {
				config[fmt.Sprintf("_arg%d", i)] = getExpressionText(expr)
			}
		}
	}
	if args := task.Arguments(); args != nil {
		if argList := args.ArgumentList(); argList != nil {
			for i, expr := range argList.AllExpression() {
				config[fmt.Sprintf("_runtime%d", i)] = getExpressionText(expr)
			}
		}
	}
	a.nodes[nodeKey] = &Node{
		Key:    nodeKey,
		Type:   name,
		Config: config,
	}

	return &Handle{
		Node:  nodeKey,
		Param: "output",
	}
}

func (a *assembler) processExpression(expr parser.IExpressionContext) *Handle {
	// Create anonymous filter/guard task
	nodeKey := a.generateKey("filter")
	a.nodes[nodeKey] = &Node{
		Key:  nodeKey,
		Type: "filter", // Built-in expression filter
		Config: map[string]any{
			"expression": getExpressionText(expr),
		},
	}
	return &Handle{Node: nodeKey, Param: "output"}
}

func (a *assembler) generateKey(prefix string) string {
	a.counter++
	return fmt.Sprintf("%s_%d", prefix, a.counter)
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
