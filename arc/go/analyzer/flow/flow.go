// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// Analyze validates a flow statement's node chain and routing tables.
func Analyze(ctx context.Context[parser.IFlowStatementContext]) bool {
	nodes := ctx.AST.AllFlowNode()
	for i, node := range nodes {
		var prevNode parser.IFlowNodeContext
		if i != 0 {
			prevNode = nodes[i-1]
		}
		isLastNode := i == len(nodes)-1
		if !analyzeNode(context.Child(ctx, node), prevNode, isLastNode) {
			return false
		}
	}
	for _, routingTable := range ctx.AST.AllRoutingTable() {
		if !analyzeRoutingTable(context.Child(ctx, routingTable)) {
			return false
		}
	}
	return true
}

func analyzeNode(ctx context.Context[parser.IFlowNodeContext], prevNode parser.IFlowNodeContext, isLastNode bool) bool {
	if id := ctx.AST.Identifier(); id != nil {
		return analyzeIdentifier(context.Child(ctx, id), prevNode, isLastNode)
	}
	if fn := ctx.AST.Function(); fn != nil {
		return parseFunction(context.Child(ctx, fn), prevNode)
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return analyzeExpression(context.Child(ctx, expr))
	}
	if ctx.AST.NEXT() != nil {
		// NEXT is always valid - it will be resolved during sequence analysis
		return true
	}
	ctx.Diagnostics.AddError(errors.New("invalid flow source"), ctx.AST)
	return true
}

func parseFunction(ctx context.Context[parser.IFunctionContext], prevNode parser.IFlowNodeContext) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	funcType, ok := resolveFunc(ctx, name)
	if !ok {
		return false
	}

	if _, ok = validateFuncConfig(
		ctx,
		name,
		funcType.Type,
		ctx.AST.ConfigValues(),
		ctx.AST,
	); !ok {
		return false
	}
	if prevNode == nil {
		return true
	}

	if prevIDNode := prevNode.Identifier(); prevIDNode != nil {
		idName := prevIDNode.IDENTIFIER().GetText()
		idSym, err := ctx.Scope.Resolve(ctx, idName)
		if err != nil {
			ctx.Diagnostics.AddError(err, prevIDNode)
			return false
		}
		// When used as a source, identifier must be a channel
		if idSym.Kind != symbol.KindChannel {
			ctx.Diagnostics.AddError(
				errors.Newf("%s is not a channel", idName),
				prevIDNode,
			)
			return false
		}
		if len(funcType.Type.Inputs) > 0 {
			param := funcType.Type.Inputs[0]
			if idSym.Type.Kind != types.KindChan {
				ctx.Diagnostics.AddError(errors.Newf(
					"%s is not a valid channel",
					idName,
				), ctx.AST)
				return false
			}
			chanValueType := idSym.Type.Unwrap()
			if err = atypes.Check(
				ctx.Constraints,
				chanValueType,
				param.Type,
				ctx.AST,
				"channel to func parameter connection",
			); err != nil {
				ctx.Diagnostics.AddError(errors.Newf(
					"channel %s value type %s does not match func %s parameter type %s",
					idName,
					chanValueType,
					name,
					param,
				), ctx.AST)
				return false
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
				ctx.Diagnostics.AddError(errors.Newf(
					"expression type %s does not match func %s parameter type %s",
					exprType,
					name,
					param.Type,
				), ctx.AST)
				return false
			}
		}
	} else if prevFuncNode := prevNode.Function(); prevFuncNode != nil {
		prevFuncName := prevFuncNode.IDENTIFIER().GetText()
		prevFuncType, ok := resolveFunc(ctx, prevFuncName)
		if !ok {
			return false
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
			ctx.Diagnostics.AddError(
				errors.Newf("%s has more than one parameter", name),
				ctx.AST,
			)
			return false
		}
		if !hasRoutingTableBetween && len(funcType.Type.Inputs) > 0 {
			t := funcType.Type.Inputs[0].Type
			var prevOutputType types.Type
			if outputType, ok := prevFuncType.Type.Outputs.Get(ir.DefaultOutputParam); ok {
				prevOutputType = outputType.Type
			} else if len(prevFuncType.Type.Outputs) > 0 {
				ctx.Diagnostics.AddError(errors.Newf(
					"func '%s' has named outputs and requires a routing table",
					prevFuncName,
				), ctx.AST)
				return false
			}
			if err := atypes.Check(ctx.Constraints, prevOutputType, t, ctx.AST,
				"flow connection between fns"); err != nil {
				ctx.Diagnostics.AddError(errors.Newf(
					"return type %s of %s is not equal to argument type %s of %s",
					prevOutputType,
					prevFuncName,
					t,
					name,
				), ctx.AST)
				return false
			}
		}
	}
	return true
}

func analyzeIdentifier(ctx context.Context[parser.IIdentifierContext], prevNode parser.IFlowNodeContext, isLastNode bool) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	// When used as a sink (last node in chain) with a previous expression,
	// add type constraint between the expression and channel value type
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
				ctx.Diagnostics.AddError(errors.Newf(
					"expression type %s does not match channel %s value type %s",
					exprType, name, chanValueType,
				), ctx.AST)
				return false
			}
		}
	}
	return true
}

func resolveFunc[T antlr.ParserRuleContext](
	ctx context.Context[T],
	name string,
) (*symbol.Scope, bool) {
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return nil, false
	}
	if sym.Kind != symbol.KindFunction {
		ctx.Diagnostics.AddError(errors.Newf("%s is not a function", name), ctx.AST)
		return nil, false
	}
	return sym, true
}

func validateFuncConfig[T antlr.ParserRuleContext](
	ctx context.Context[T],
	fnName string,
	fnType types.Type,
	configBlock parser.IConfigValuesContext,
	configNode antlr.ParserRuleContext,
) (map[string]bool, bool) {
	configParams := make(map[string]bool)
	if configBlock == nil {
		return configParams, true
	}

	if namedVals := configBlock.NamedConfigValues(); namedVals != nil {
		for _, configVal := range namedVals.AllNamedConfigValue() {
			key := configVal.IDENTIFIER().GetText()
			configParams[key] = true
			expectedType, exists := fnType.Config.Get(key)
			if !exists {
				ctx.Diagnostics.AddError(
					errors.Newf("unknown config parameter '%s' for func '%s'", key, fnName),
					configVal,
				)
				return nil, false
			}
			if expr := configVal.Expression(); expr != nil {
				childCtx := context.Child(ctx, expr)
				if !expression.Analyze(childCtx) {
					return nil, false
				}
				exprType := atypes.InferFromExpression(childCtx)
				if err := atypes.Check(ctx.Constraints, expectedType.Type, exprType, configVal,
					"config parameter '"+key+"' for func '"+fnName+"'"); err != nil {
					ctx.Diagnostics.AddError(
						errors.Newf(
							"type mismatch: config parameter '%s' expects %s but got %s",
							key,
							expectedType.Type,
							exprType,
						),
						configVal,
					)
					return nil, false
				}
			}
		}
	} else if anonVals := configBlock.AnonymousConfigValues(); anonVals != nil {
		ctx.Diagnostics.AddError(
			errors.Newf("anonymous configuration values are not supported"),
			anonVals,
		)
		return nil, false
	}

	for _, param := range fnType.Config {
		if !configParams[param.Name] {
			ctx.Diagnostics.AddError(
				errors.Newf("missing required config parameter '%s' for func '%s'", param.Name, fnName),
				configNode,
			)
			return nil, false
		}
	}

	return configParams, true
}

func analyzeRoutingTable(ctx context.Context[parser.IRoutingTableContext]) bool {
	// Find the parent flow statement
	flowStmt, ok := ctx.AST.GetParent().(parser.IFlowStatementContext)
	if !ok {
		ctx.Diagnostics.AddError(errors.New("routing table must be part of a flow statement"), ctx.AST)
		return false
	}

	tables := flowStmt.AllRoutingTable()

	if len(tables) != 1 || tables[0] != ctx.AST {
		ctx.Diagnostics.AddError(errors.New("unexpected routing table configuration"), ctx.AST)
		return false
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
		return analyzeInputRoutingTable(ctx, nodesAfter)
	} else if len(nodesBefore) > 0 {
		return analyzeOutputRoutingTable(ctx, nodesBefore, nodesAfter)
	} else {
		ctx.Diagnostics.AddError(errors.New("routing table must have associated flow nodes"), ctx.AST)
		return false
	}
}

func analyzeOutputRoutingTable(
	ctx context.Context[parser.IRoutingTableContext],
	nodesBefore []parser.IFlowNodeContext,
	nodesAfter []parser.IFlowNodeContext,
) bool {
	var PrevFunc parser.IFunctionContext
	for i := len(nodesBefore) - 1; i >= 0; i-- {
		if fn := nodesBefore[i].Function(); fn != nil {
			PrevFunc = fn
			break
		}
	}

	if PrevFunc == nil {
		ctx.Diagnostics.AddError(errors.New("output routing table must follow a func invocation"), ctx.AST)
		return false
	}

	fnName := PrevFunc.IDENTIFIER().GetText()
	fnType, ok := resolveFunc(ctx, fnName)
	if !ok {
		return false
	}

	_, hasDefaultOutput := fnType.Type.Outputs.Get(ir.DefaultOutputParam)
	hasNamedOutputs := len(fnType.Type.Outputs) > 1 || (len(fnType.Type.Outputs) == 1 && !hasDefaultOutput)
	if !hasNamedOutputs {
		ctx.Diagnostics.AddError(
			errors.Newf("func '%s' does not have named outputs, cannot use routing table", fnName),
			ctx.AST,
		)
		return false
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
			ctx.Diagnostics.AddError(
				errors.Newf("func '%s' does not have output '%s'", fnName, outputName),
				entry,
			)
			return false
		}

		var targetParamName string
		if len(entry.AllIDENTIFIER()) > 1 {
			targetParamName = entry.IDENTIFIER(1).GetText()

			if nextFunc == nil {
				ctx.Diagnostics.AddError(
					errors.New("parameter mapping requires a func after the routing table"),
					entry,
				)
				return false
			}

			if _, exists := nextFuncType.Inputs.Get(targetParamName); !exists {
				ctx.Diagnostics.AddError(
					errors.Newf(
						"func '%s' does not have parameter '%s'",
						nextFunc.IDENTIFIER().GetText(),
						targetParamName,
					),
					entry,
				)
				return false
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
			if !analyzeRoutingTargetWithParam(context.Child(ctx, flowNode), outputType.Type, nextFuncType, targetParam) {
				return false
			}
		}
	}

	return true
}

func analyzeInputRoutingTable(
	ctx context.Context[parser.IRoutingTableContext],
	nodes []parser.IFlowNodeContext,
) bool {
	var nextFunc parser.IFunctionContext
	for i := 0; i < len(nodes); i++ {
		if fn := nodes[i].Function(); fn != nil {
			nextFunc = fn
			break
		}
	}

	if nextFunc == nil {
		ctx.Diagnostics.AddError(errors.New("input routing table must precede a func invocation"), ctx.AST)
		return false
	}

	fnName := nextFunc.IDENTIFIER().GetText()
	fnType, ok := resolveFunc(ctx, fnName)
	if !ok {
		return false
	}

	for _, entry := range ctx.AST.AllRoutingEntry() {
		flowNodes := entry.AllFlowNode()
		if len(flowNodes) == 0 {
			ctx.Diagnostics.AddError(errors.New("routing entry must have at least one target"), entry)
			return false
		}

		lastNode := flowNodes[len(flowNodes)-1]
		if lastNode.Identifier() == nil {
			ctx.Diagnostics.AddError(
				errors.New("last element in input routing entry must be a parameter name (identifier)"),
				lastNode,
			)
			return false
		}

		paramName := lastNode.Identifier().IDENTIFIER().GetText()

		paramType, exists := fnType.Type.Inputs.Get(paramName)
		if !exists {
			ctx.Diagnostics.AddError(
				errors.Newf("func '%s' does not have parameter '%s'", fnName, paramName),
				lastNode,
			)
			return false
		}

		// Analyze the flow chain: source (entry.IDENTIFIER) -> processing nodes -> parameter
		// For type checking, we need to verify the output type of the chain matches paramType
		// TODO: Implement full type checking for the flow chain
		// See https://linear.app/synnax/issue/SY-3176/implement-full-type-checking-for-arc-flow-statements
		_ = paramType

		for i := 0; i < len(flowNodes)-1; i++ {
			if !analyzeNode(context.Child(ctx, flowNodes[i]), nil, false) {
				return false
			}
		}
	}

	return true
}

func analyzeRoutingTargetWithParam(
	ctx context.Context[parser.IFlowNodeContext],
	sourceType types.Type,
	nextFuncType types.Type,
	targetParam *string,
) bool {
	if fn := ctx.AST.Function(); fn != nil {
		fnName := fn.IDENTIFIER().GetText()
		fnType, ok := resolveFunc(ctx, fnName)
		if !ok {
			return false
		}

		if _, ok = validateFuncConfig(
			ctx,
			fnName,
			fnType.Type,
			fn.ConfigValues(),
			fn,
		); !ok {
			return false
		}

		if targetParam != nil {
			var outputType types.Type
			if outType, ok := fnType.Type.Outputs.Get(ir.DefaultOutputParam); ok {
				outputType = outType.Type
			} else if len(fnType.Type.Outputs) > 0 {
				ctx.Diagnostics.AddError(errors.Newf(
					"func '%s' has named outputs and requires explicit output selection",
					fnName,
				), ctx.AST)
				return false
			}

			if param, exists := nextFuncType.Inputs.Get(*targetParam); exists {
				if err := atypes.Check(ctx.Constraints, outputType, param.Type, ctx.AST,
					"routing table parameter mapping"); err != nil {
					ctx.Diagnostics.AddError(errors.Newf(
						"type mismatch: func %s output type %s does not match target parameter %s type %s",
						fnName,
						outputType,
						*targetParam,
						param,
					), ctx.AST)
					return false
				}
			}
		} else {
			if len(fnType.Type.Inputs) > 0 {
				param := fnType.Type.Inputs[0]
				if err := atypes.Check(ctx.Constraints, sourceType, param.Type, ctx.AST,
					"routing table output to func parameter"); err != nil {
					ctx.Diagnostics.AddError(errors.Newf(
						"type mismatch: output type %s does not match func %s parameter type %s",
						sourceType,
						fnName,
						param.Type,
					), ctx.AST)
					return false
				}
			}
		}
	} else if idNode := ctx.AST.Identifier(); idNode != nil {
		idName := idNode.IDENTIFIER().GetText()
		idSym, err := ctx.Scope.Resolve(ctx, idName)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}

		// Allow both channels and sequences as routing targets
		if idSym.Kind != symbol.KindChannel && idSym.Kind != symbol.KindSequence {
			ctx.Diagnostics.AddError(
				errors.Newf("%s is not a channel or sequence", idName),
				ctx.AST,
			)
			return false
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
				ctx.Diagnostics.AddError(errors.Newf(
					"type mismatch: output type %s does not match channel %s value type %s",
					sourceType,
					idName,
					valueType,
				), ctx.AST)
				return false
			}
		}
	} else if expr := ctx.AST.Expression(); expr != nil {
		if !analyzeExpression(context.Child(ctx, expr)) {
			return false
		}
	}
	return true
}
