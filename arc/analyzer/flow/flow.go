// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

// Analyze processes a flow statement and returns true if successful
func Analyze(ctx context.Context[parser.IFlowStatementContext]) bool {
	for i, node := range ctx.AST.AllFlowNode() {
		var prevNode parser.IFlowNodeContext
		if i != 0 {
			prevNode = ctx.AST.FlowNode(i - 1)
		}
		if !analyzeNode(context.Child(ctx, node), prevNode) {
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

func analyzeNode(ctx context.Context[parser.IFlowNodeContext], prevNode parser.IFlowNodeContext) bool {
	if fn := ctx.AST.Function(); fn != nil {
		return parseFunction(context.Child(ctx, fn), prevNode)
	}
	if channelID := ctx.AST.ChannelIdentifier(); channelID != nil {
		return analyzeChannel(context.Child(ctx, channelID))
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return analyzeExpression(context.Child(ctx, expr))
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

	if prevChannelNode := prevNode.ChannelIdentifier(); prevChannelNode != nil {
		channelName := prevChannelNode.IDENTIFIER().GetText()
		channelSym, err := ctx.Scope.Resolve(ctx, channelName)
		if err != nil {
			ctx.Diagnostics.AddError(err, prevChannelNode)
			return false
		}
		if channelSym.Kind != symbol.KindChannel {
			ctx.Diagnostics.AddError(
				errors.Newf("%s is not a channel", channelName),
				prevChannelNode,
			)
			return false
		}
		if funcType.Type.Inputs.Count() > 0 {
			_, paramType := funcType.Type.Inputs.At(0)
			if channelSym.Type.Kind != types.KindChan {
				ctx.Diagnostics.AddError(errors.Newf(
					"%s is not a valid channel",
					channelName,
				), ctx.AST)
				return false
			}
			chanValueType := channelSym.Type.Unwrap()
			if err = atypes.Check(
				ctx.Constraints,
				chanValueType,
				paramType,
				ctx.AST,
				"channel to func parameter connection",
			); err != nil {
				ctx.Diagnostics.AddError(errors.Newf(
					"channel %s value type %s does not match func %s parameter type %s",
					channelName,
					chanValueType,
					name,
					paramType,
				), ctx.AST)
				return false
			}
		}
	} else if prevExpr := prevNode.Expression(); prevExpr != nil {
		exprType := atypes.InferFromExpression(context.Child(ctx, prevExpr)).Unwrap()
		if funcType.Type.Inputs.Count() > 0 {
			_, paramType := funcType.Type.Inputs.At(0)
			if err := atypes.Check(
				ctx.Constraints,
				exprType,
				paramType,
				ctx.AST,
				"expression to func parameter connection",
			); err != nil {
				ctx.Diagnostics.AddError(errors.Newf(
					"expression type %s does not match func %s parameter type %s",
					exprType,
					name,
					paramType,
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

		if !hasRoutingTableBetween && funcType.Type.Inputs.Count() > 1 {
			ctx.Diagnostics.AddError(
				errors.Newf("%s has more than one parameter", name),
				ctx.AST,
			)
			return false
		}
		if !hasRoutingTableBetween && funcType.Type.Inputs.Count() > 0 {
			_, t := funcType.Type.Inputs.At(0)
			var prevOutputType types.Type
			if outputType, ok := prevFuncType.Type.Outputs.Get(ir.DefaultOutputParam); ok {
				prevOutputType = outputType
			} else if prevFuncType.Type.Outputs.Count() > 0 {
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

func analyzeChannel(ctx context.Context[parser.IChannelIdentifierContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	_, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
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
				if err := atypes.Check(ctx.Constraints, expectedType, exprType, configVal,
					"config parameter '"+key+"' for func '"+fnName+"'"); err != nil {
					ctx.Diagnostics.AddError(
						errors.Newf(
							"type mismatch: config parameter '%s' expects %s but got %s",
							key,
							expectedType,
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

	for paramName := range fnType.Config.Iter() {
		if !configParams[paramName] {
			ctx.Diagnostics.AddError(
				errors.Newf("missing required config parameter '%s' for func '%s'", paramName, fnName),
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
	hasNamedOutputs := fnType.Type.Outputs.Count() > 1 || (fnType.Type.Outputs.Count() == 1 && !hasDefaultOutput)
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
			if !analyzeRoutingTargetWithParam(context.Child(ctx, flowNode), outputType, nextFuncType, targetParam) {
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
		if lastNode.ChannelIdentifier() == nil {
			ctx.Diagnostics.AddError(
				errors.New("last element in input routing entry must be a parameter name (identifier)"),
				lastNode,
			)
			return false
		}

		paramName := lastNode.ChannelIdentifier().IDENTIFIER().GetText()

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
			if !analyzeNode(context.Child(ctx, flowNodes[i]), nil) {
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
				outputType = outType
			} else if fnType.Type.Outputs.Count() > 0 {
				ctx.Diagnostics.AddError(errors.Newf(
					"func '%s' has named outputs and requires explicit output selection",
					fnName,
				), ctx.AST)
				return false
			}

			if paramType, exists := nextFuncType.Inputs.Get(*targetParam); exists {
				if err := atypes.Check(ctx.Constraints, outputType, paramType, ctx.AST,
					"routing table parameter mapping"); err != nil {
					ctx.Diagnostics.AddError(errors.Newf(
						"type mismatch: func %s output type %s does not match target parameter %s type %s",
						fnName,
						outputType,
						*targetParam,
						paramType,
					), ctx.AST)
					return false
				}
			}
		} else {
			if fnType.Type.Inputs.Count() > 0 {
				_, paramType := fnType.Type.Inputs.At(0)
				if err := atypes.Check(ctx.Constraints, sourceType, paramType, ctx.AST,
					"routing table output to func parameter"); err != nil {
					ctx.Diagnostics.AddError(errors.Newf(
						"type mismatch: output type %s does not match func %s parameter type %s",
						sourceType,
						fnName,
						paramType,
					), ctx.AST)
					return false
				}
			}
		}
	} else if channelID := ctx.AST.ChannelIdentifier(); channelID != nil {
		channelName := channelID.IDENTIFIER().GetText()
		channelSym, err := ctx.Scope.Resolve(ctx, channelName)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}

		if channelSym.Kind != symbol.KindChannel {
			ctx.Diagnostics.AddError(
				errors.Newf("%s is not a channel", channelName),
				ctx.AST,
			)
			return false
		}

		valueType := channelSym.Type.Unwrap()
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
				channelName,
				valueType,
			), ctx.AST)
			return false
		}
	} else if expr := ctx.AST.Expression(); expr != nil {
		if !analyzeExpression(context.Child(ctx, expr)) {
			return false
		}
	}
	return true
}
