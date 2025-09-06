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

	"github.com/antlr4-go/antlr/v4"
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
			if err := a.extractTask(taskDecl); err != nil {
				return nil, err
			}
		} else if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			a.extractFunction(funcDecl)
		}
	}

	// Second pass: process flow statements to build nodes and edges
	for _, item := range program.AllTopLevelItem() {
		if flow := item.FlowStatement(); flow != nil {
			if err := a.processFlow(flow); err != nil {
				return nil, err
			}
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

func (a *assembler) resolveTaskType(name string) (types.Task, error) {
	sym, err := a.scope.Resolve(name)
	if err != nil {
		return types.Task{}, err
	}
	return types.Assert[types.Task](sym.Type)
}

func (a *assembler) resolveTaskTypeByParserRule(rule antlr.ParserRuleContext) (types.Task, error) {
	sym, err := a.scope.Root().GetChildByParserRule(rule)
	if err != nil {
		return types.Task{}, err
	}
	return types.Assert[types.Task](sym.Type)
}

func (a *assembler) extractTask(taskDecl parser.ITaskDeclarationContext) error {
	name := taskDecl.IDENTIFIER().GetText()
	taskType, err := a.resolveTaskType(name)
	if err != nil {
		return err
	}
	task := Task{
		Key:      name,
		Config:   make(map[string]string),
		Params:   make(map[string]string),
		Stateful: make(map[string]StatefulVariable),
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
	return nil
}

func (a *assembler) extractFunction(funcDecl parser.IFunctionDeclarationContext) {
	name := funcDecl.IDENTIFIER().GetText()
	sym, _ := a.scope.Resolve(name)
	if sym == nil {
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

func (a *assembler) processFlow(flow parser.IFlowStatementContext) error {
	var prevHandle *Handle
	for i, flowNode := range flow.AllFlowNode() {
		handle, err := a.processFlowNode(flowNode)
		if err != nil {
			return err
		}
		if i > 0 && prevHandle != nil && handle != nil {
			a.edges = append(a.edges, Edge{
				Source: *prevHandle,
				Target: *handle,
			})
		}
		prevHandle = handle
	}
	return nil
}

func (a *assembler) processFlowNode(node parser.IFlowNodeContext) (*Handle, error) {
	if channel := node.ChannelIdentifier(); channel != nil {
		return a.processChannel(channel)
	}
	if task := node.TaskInvocation(); task != nil {
		return a.processTask(task)
	}
	if expr := node.Expression(); expr != nil {
		return a.processExpression(expr)
	}
	return nil, nil
}

func (a *assembler) processChannel(channel parser.IChannelIdentifierContext) (*Handle, error) {
	name := channel.IDENTIFIER().GetText()
	nodeKey := a.generateKey("on")
	a.nodes[nodeKey] = &Node{
		Key:    nodeKey,
		Type:   "on",
		Config: map[string]any{"channel": name},
	}
	return &Handle{Node: nodeKey, Param: "output"}, nil
}

func extractConfigValues(values parser.IConfigValuesContext, taskType types.Task) map[string]any {
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
			key, _ := taskType.Params.At(i)
			config[key] = getExpressionText(expr)
		}
	}
	return config
}

func (a *assembler) processTask(task parser.ITaskInvocationContext) (*Handle, error) {
	var (
		name    = task.IDENTIFIER().GetText()
		nodeKey = a.generateKey(name)
	)
	taskType, err := a.resolveTaskType(name)
	if err != nil {
		return nil, err
	}
	config := extractConfigValues(task.ConfigValues(), taskType)
	if args := task.Arguments(); args != nil {
		if argList := args.ArgumentList(); argList != nil {
			for i, expr := range argList.AllExpression() {
				config[fmt.Sprintf("_runtime%d", i)] = getExpressionText(expr)
			}
		}
	}
	a.nodes[nodeKey] = &Node{Key: nodeKey, Type: name, Config: config}
	return &Handle{Node: nodeKey, Param: "output"}, nil
}

func (a *assembler) processExpression(expr parser.IExpressionContext) (*Handle, error) {
	sym, err := a.scope.Root().GetChildByParserRule(expr)
	if err != nil {
		return nil, err
	}
	taskType, err := types.Assert[types.Task](sym.Type)
	if err != nil {
		return nil, err
	}
	task := Task{Key: sym.Name, Params: make(map[string]string)}
	for key, item := range taskType.Params.Iter() {
		task.Params[key] = item.String()
	}
	a.nodes[sym.Name] = &Node{
		Key:  sym.Name,
		Type: sym.Name,
		Config: map[string]any{
			"expression": getExpressionText(expr),
		},
	}
	a.tasks = append(a.tasks, task)
	return &Handle{Node: sym.Name, Param: "output"}, nil
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
