// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package flow implements semantic analysis for Arc flow statements connecting channels
// and functions into reactive data pipelines.
package flow

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

func AnalyzeSingleFunction(ctx context.Context[parser.IFunctionContext]) {
	name := ctx.AST.IDENTIFIER().GetText()
	funcType := resolveFunc(ctx, name)
	if funcType == nil {
		return
	}
	validateFuncConfig(ctx, name, funcType.Type, ctx.AST.ConfigValues(), ctx.AST)
	for _, input := range funcType.Type.Inputs {
		if input.Value == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
				"standalone function '%s' has required input '%s' with no source; "+
					"use in a flow statement or provide a default value",
				name, input.Name))
		}
	}
}

// Analyze validates a flow statement's node chain and routing tables.
func Analyze(ctx context.Context[parser.IFlowStatementContext]) {
	nodes := ctx.AST.AllFlowNode()
	for i, node := range nodes {
		var prevNode parser.IFlowNodeContext
		if i != 0 {
			prevNode = nodes[i-1]
		}
		isLastNode := i == len(nodes)-1
		analyzeNode(context.Child(ctx, node), prevNode, isLastNode)
	}
	for _, routingTable := range ctx.AST.AllRoutingTable() {
		analyzeRoutingTable(context.Child(ctx, routingTable))
	}
}

func analyzeNode(ctx context.Context[parser.IFlowNodeContext], prevNode parser.IFlowNodeContext, isLastNode bool) {
	if id := ctx.AST.Identifier(); id != nil {
		analyzeIdentifier(context.Child(ctx, id), prevNode, isLastNode)
		return
	}
	if fn := ctx.AST.Function(); fn != nil {
		parseFunction(context.Child(ctx, fn), prevNode)
		return
	}
	if expr := ctx.AST.Expression(); expr != nil {
		AnalyzeSingleExpression(context.Child(ctx, expr))
		return
	}
	// NEXT is always valid - it will be resolved during sequence analysis.
	// The grammar guarantees flowNode is one of: identifier | function | expression | NEXT
}

func parseFunction(ctx context.Context[parser.IFunctionContext], prevNode parser.IFlowNodeContext) {
	name := ctx.AST.IDENTIFIER().GetText()
	funcType := resolveFunc(ctx, name)
	if funcType == nil {
		return
	}

	validateFuncConfig(
		ctx,
		name,
		funcType.Type,
		ctx.AST.ConfigValues(),
		ctx.AST,
	)
	if prevNode == nil {
		return
	}

	if prevIDNode := prevNode.Identifier(); prevIDNode != nil {
		idName := prevIDNode.IDENTIFIER().GetText()
		idSym, err := ctx.Scope.Resolve(ctx, idName)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, prevIDNode))
			return
		}
		// When used as a source, identifier must be a channel
		if idSym.Kind != symbol.KindChannel {
			ctx.Diagnostics.Add(diagnostics.Errorf(prevIDNode, "%s is not a channel", idName))
			return
		}
		if len(funcType.Type.Inputs) > 0 {
			param := funcType.Type.Inputs[0]
			if idSym.Type.Kind != types.KindChan {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"%s is not a valid channel",
					idName,
				))
				return
			}
			chanValueType := idSym.Type.Unwrap()
			if err = atypes.Check(
				ctx.Constraints,
				chanValueType,
				param.Type,
				ctx.AST,
				"channel to func parameter connection",
			); err != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"channel %s value type %s does not match func %s parameter type %s",
					idName,
					chanValueType,
					name,
					param,
				))
				return
			}
		}
	} else if prevExpr := prevNode.Expression(); prevExpr != nil {
		exprType := atypes.InferFromExpression(context.Child(ctx, prevExpr)).Unwrap()
		if len(funcType.Type.Inputs) > 0 {
			param := funcType.Type.Inputs[0]
			if err := atypes.Check(
				ctx.Constraints,
				exprType,
				param.Type,
				ctx.AST,
				"expression to func parameter connection",
			); err != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"expression type %s does not match func %s parameter type %s",
					exprType,
					name,
					param.Type,
				))
				return
			}
		}
	} else if prevFuncNode := prevNode.Function(); prevFuncNode != nil {
		prevFuncName := prevFuncNode.IDENTIFIER().GetText()
		prevFuncType := resolveFunc(ctx, prevFuncName)
		if prevFuncType == nil {
			return
		}
		hasRoutingTableBetween := false
		if parent := ctx.AST.GetParent(); parent != nil {
			if grandparent := parent.GetParent(); grandparent != nil {
				if flowStmt, ok := grandparent.(parser.IFlowStatementContext); ok {
					if len(flowStmt.AllRoutingTable()) > 0 {
						hasRoutingTableBetween = true
					}
				}
			}
		}

		if !hasRoutingTableBetween && len(funcType.Type.Inputs) > 1 {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "%s has more than one parameter", name))
			return
		}
		if !hasRoutingTableBetween && len(funcType.Type.Inputs) > 0 {
			t := funcType.Type.Inputs[0].Type
			var prevOutputType types.Type
			if outputType, ok := prevFuncType.Type.Outputs.Get(ir.DefaultOutputParam); ok {
				prevOutputType = outputType.Type
			} else if len(prevFuncType.Type.Outputs) > 0 {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"func '%s' has named outputs and requires a routing table",
					prevFuncName,
				))
				return
			} else {
				// Void function (no outputs) cannot feed into a function expecting input
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"func '%s' has no return value but '%s' expects an input parameter",
					prevFuncName,
					name,
				))
				return
			}
			if err := atypes.Check(ctx.Constraints, prevOutputType, t, ctx.AST,
				"flow connection between fns"); err != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"return type %s of %s is not equal to argument type %s of %s",
					prevOutputType,
					prevFuncName,
					t,
					name,
				))
				return
			}
		}
	}
}

func analyzeIdentifier(
	ctx context.Context[parser.IIdentifierContext],
	prevNode parser.IFlowNodeContext,
	isLastNode bool,
) {
	name := ctx.AST.IDENTIFIER().GetText()
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}

	if prevNode != nil && prevNode.Function() != nil {
		validTarget := sym.Kind == symbol.KindChannel || sym.Kind == symbol.KindStage || sym.Kind == symbol.KindSequence
		if !validTarget {
			d := diagnostics.Errorf(ctx.AST, "%s is not a channel", name)
			if sym.Kind == symbol.KindFunction {
				d = d.WithNote("use " + name + "{} to instantiate the function")
			}
			ctx.Diagnostics.Add(d)
			return
		}
	}

	if isLastNode && prevNode != nil && sym.Kind == symbol.KindChannel {
		if prevExpr := prevNode.Expression(); prevExpr != nil {
			exprType := atypes.InferFromExpression(context.Child(ctx, prevExpr))
			chanValueType := sym.Type.Unwrap()
			if err = atypes.Check(
				ctx.Constraints,
				exprType,
				chanValueType,
				ctx.AST,
				"expression to channel sink",
			); err != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"expression type %s does not match channel %s value type %s",
					exprType, name, chanValueType,
				))
				return
			}
		}
	}
}

func resolveFunc[T antlr.ParserRuleContext](
	ctx context.Context[T],
	name string,
) *symbol.Scope {
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nil
	}
	if sym.Kind != symbol.KindFunction {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "%s is not a function", name))
		return nil
	}
	return sym
}

func validateFuncConfig[T antlr.ParserRuleContext](
	ctx context.Context[T],
	fnName string,
	fnType types.Type,
	configBlock parser.IConfigValuesContext,
	configNode antlr.ParserRuleContext,
) {
	if configBlock == nil {
		return
	}

	configParams := make(map[string]bool)
	if namedVals := configBlock.NamedConfigValues(); namedVals != nil {
		for _, configVal := range namedVals.AllNamedConfigValue() {
			key := configVal.IDENTIFIER().GetText()
			configParams[key] = true
			expectedType, exists := fnType.Config.Get(key)
			if !exists {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					configVal,
					"unknown config parameter '%s' for func '%s'",
					key,
					fnName,
				))
				continue
			}
			if expr := configVal.Expression(); expr != nil {
				childCtx := context.Child(ctx, expr)
				expression.Analyze(childCtx)
				exprType := atypes.InferFromExpression(childCtx)
				if err := atypes.Check(ctx.Constraints, expectedType.Type, exprType, configVal,
					"config parameter '"+key+"' for func '"+fnName+"'"); err != nil {
					ctx.Diagnostics.Add(diagnostics.Errorf(
						configVal,
						"type mismatch: config parameter '%s' expects %s but got %s",
						key,
						expectedType.Type,
						exprType,
					))
				}
			}
		}
	} else if anonVals := configBlock.AnonymousConfigValues(); anonVals != nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			anonVals,
			"anonymous configuration values are not supported",
		))
		return
	}

	for _, param := range fnType.Config {
		if !configParams[param.Name] {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				configNode,
				"missing required config parameter '%s' for func '%s'",
				param.Name,
				fnName,
			))
		}
	}
}

func analyzeRoutingTable(ctx context.Context[parser.IRoutingTableContext]) {
	flowStmt, ok := ctx.AST.GetParent().(parser.IFlowStatementContext)
	if !ok {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"routing table must be part of a flow statement",
		))
		return
	}

	tables := flowStmt.AllRoutingTable()

	if len(tables) != 1 || tables[0] != ctx.AST {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"unexpected routing table configuration",
		))
		return
	}

	var nodesBefore, nodesAfter []parser.IFlowNodeContext
	foundRoutingTable := false

	for _, child := range flowStmt.GetChildren() {
		if rt, ok := child.(parser.IRoutingTableContext); ok && rt == ctx.AST {
			foundRoutingTable = true
			continue
		}
		if flowNode, ok := child.(parser.IFlowNodeContext); ok {
			if foundRoutingTable {
				nodesAfter = append(nodesAfter, flowNode)
			} else {
				nodesBefore = append(nodesBefore, flowNode)
			}
		}
	}

	if len(nodesBefore) == 0 && len(nodesAfter) > 0 {
		analyzeInputRoutingTable(ctx, nodesAfter)
	} else if len(nodesBefore) > 0 {
		analyzeOutputRoutingTable(ctx, nodesBefore, nodesAfter)
	} else {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"routing table must have associated flow nodes",
		))
	}
}

func analyzeOutputRoutingTable(
	ctx context.Context[parser.IRoutingTableContext],
	nodesBefore []parser.IFlowNodeContext,
	nodesAfter []parser.IFlowNodeContext,
) {
	var PrevFunc parser.IFunctionContext
	for i := len(nodesBefore) - 1; i >= 0; i-- {
		if fn := nodesBefore[i].Function(); fn != nil {
			PrevFunc = fn
			break
		}
	}

	if PrevFunc == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"output routing table must follow a func invocation",
		))
		return
	}

	fnName := PrevFunc.IDENTIFIER().GetText()
	fnType := resolveFunc(ctx, fnName)
	if fnType == nil {
		return
	}

	_, hasDefaultOutput := fnType.Type.Outputs.Get(ir.DefaultOutputParam)
	hasNamedOutputs := len(fnType.Type.Outputs) > 1 || (len(fnType.Type.Outputs) == 1 && !hasDefaultOutput)
	if !hasNamedOutputs {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"func '%s' does not have named outputs, cannot use routing table",
			fnName,
		))
		return
	}

	var (
		nextFunc     parser.IFunctionContext
		nextFuncType types.Type
	)
	for _, node := range nodesAfter {
		if fn := node.Function(); fn != nil {
			nextFunc = fn
			nextFuncName := nextFunc.IDENTIFIER().GetText()
			nextFuncScope, err := ctx.Scope.Resolve(ctx, nextFuncName)
			if err == nil && nextFuncScope.Kind == symbol.KindFunction {
				nextFuncType = nextFuncScope.Type
			}
			break
		}
	}

	// Analyze each routing entry
	for _, entry := range ctx.AST.AllRoutingEntry() {
		outputName := entry.IDENTIFIER(0).GetText()

		outputType, exists := fnType.Type.Outputs.Get(outputName)
		if !exists {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry,
				"func '%s' does not have output '%s'",
				fnName,
				outputName,
			))
			continue
		}

		var targetParamName string
		if len(entry.AllIDENTIFIER()) > 1 {
			targetParamName = entry.IDENTIFIER(1).GetText()

			if nextFunc == nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					entry,
					"parameter mapping requires a func after the routing table",
				))
				continue
			}

			if _, exists := nextFuncType.Inputs.Get(targetParamName); !exists {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					entry,
					"func '%s' does not have parameter '%s'",
					nextFunc.IDENTIFIER().GetText(),
					targetParamName,
				))
				continue
			}
		}

		// Analyze each flow node in the routing entry chain
		flowNodes := entry.AllFlowNode()
		for i, flowNode := range flowNodes {
			isLastNode := i == len(flowNodes)-1
			var targetParam *string
			if isLastNode && targetParamName != "" {
				targetParam = &targetParamName
			}
			analyzeRoutingTargetWithParam(
				context.Child(ctx, flowNode),
				outputType.Type,
				nextFuncType,
				targetParam,
			)
		}
	}
}

func analyzeInputRoutingTable(
	ctx context.Context[parser.IRoutingTableContext],
	nodes []parser.IFlowNodeContext,
) {
	var nextFunc parser.IFunctionContext
	for i := 0; i < len(nodes); i++ {
		if fn := nodes[i].Function(); fn != nil {
			nextFunc = fn
			break
		}
	}

	if nextFunc == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "input routing table must precede a func invocation"))
		return
	}

	fnName := nextFunc.IDENTIFIER().GetText()
	fnType := resolveFunc(ctx, fnName)
	if fnType == nil {
		return
	}

	for _, entry := range ctx.AST.AllRoutingEntry() {
		flowNodes := entry.AllFlowNode()
		if len(flowNodes) == 0 {
			ctx.Diagnostics.Add(diagnostics.Errorf(entry, "routing entry must have at least one target"))
			continue
		}

		lastNode := flowNodes[len(flowNodes)-1]
		if lastNode.Identifier() == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				lastNode,
				"last element in input routing entry must be a parameter name (identifier)",
			))
			continue
		}

		paramName := lastNode.Identifier().IDENTIFIER().GetText()

		paramType, exists := fnType.Type.Inputs.Get(paramName)
		if !exists {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				lastNode,
				"func '%s' does not have parameter '%s'",
				fnName,
				paramName,
			))
			continue
		}

		// Analyze the flow chain: source (entry.IDENTIFIER) -> processing nodes -> parameter
		// For type checking, we need to verify the output type of the chain matches paramType
		// TODO: Implement full type checking for the flow chain
		// See https://linear.app/synnax/issue/SY-3176/implement-full-type-checking-for-arc-flow-statements
		_ = paramType

		for i := 0; i < len(flowNodes)-1; i++ {
			analyzeNode(context.Child(ctx, flowNodes[i]), nil, false)
		}
	}
}

func analyzeRoutingTargetWithParam(
	ctx context.Context[parser.IFlowNodeContext],
	sourceType types.Type,
	nextFuncType types.Type,
	targetParam *string,
) {
	if fn := ctx.AST.Function(); fn != nil {
		fnName := fn.IDENTIFIER().GetText()
		fnType := resolveFunc(ctx, fnName)
		if fnType == nil {
			return
		}

		validateFuncConfig(
			ctx,
			fnName,
			fnType.Type,
			fn.ConfigValues(),
			fn,
		)

		if targetParam != nil {
			var outputType types.Type
			if outType, ok := fnType.Type.Outputs.Get(ir.DefaultOutputParam); ok {
				outputType = outType.Type
			} else if len(fnType.Type.Outputs) > 0 {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"func '%s' has named outputs and requires explicit output selection",
					fnName,
				))
				return
			}

			if param, exists := nextFuncType.Inputs.Get(*targetParam); exists {
				if err := atypes.Check(ctx.Constraints, outputType, param.Type, ctx.AST,
					"routing table parameter mapping"); err != nil {
					ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
						"type mismatch: func %s output type %s does not match target parameter %s type %s",
						fnName,
						outputType,
						*targetParam,
						param,
					))
					return
				}
			}
		} else {
			if len(fnType.Type.Inputs) > 0 {
				param := fnType.Type.Inputs[0]
				if err := atypes.Check(ctx.Constraints, sourceType, param.Type, ctx.AST,
					"routing table output to func parameter"); err != nil {
					ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
						"type mismatch: output type %s does not match func %s parameter type %s",
						sourceType,
						fnName,
						param.Type,
					))
					return
				}
			}
		}
	} else if idNode := ctx.AST.Identifier(); idNode != nil {
		idName := idNode.IDENTIFIER().GetText()
		idSym, err := ctx.Scope.Resolve(ctx, idName)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return
		}

		// Allow both channels and sequences as routing targets
		if idSym.Kind != symbol.KindChannel && idSym.Kind != symbol.KindSequence {
			ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "%s is not a channel or sequence", idName))
			return
		}

		// Only do type checking for channels (sequences accept any input for activation)
		if idSym.Kind == symbol.KindChannel {
			valueType := idSym.Type.Unwrap()
			if err = atypes.Check(
				ctx.Constraints,
				sourceType,
				valueType,
				ctx.AST,
				"routing table output to channel",
			); err != nil {
				ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
					"type mismatch: output type %s does not match channel %s value type %s",
					sourceType,
					idName,
					valueType,
				))
				return
			}
		}
	} else if expr := ctx.AST.Expression(); expr != nil {
		AnalyzeSingleExpression(context.Child(ctx, expr))
	}
}
