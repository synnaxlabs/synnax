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
	"github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	types2 "github.com/synnaxlabs/arc/types"
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
	if taskInv := ctx.AST.StageInvocation(); taskInv != nil {
		return parseStageInvocation(context.Child(ctx, taskInv), prevNode)
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

func parseStageInvocation(ctx context.Context[parser.IStageInvocationContext], prevNode parser.IFlowNodeContext) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	stageType, ok := resolveStage(ctx, name)
	if !ok {
		return false
	}

	if _, ok := validateStageConfig(ctx, name, stageType.Type, ctx.AST.ConfigValues(), ctx.AST); !ok {
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
		if stageType.Type.Inputs.Count() > 0 {
			_, paramType := stageType.Type.Inputs.At(0)
			chanType := types2.Chan(channelSym.Type)
			// Create constraint between channel value type and parameter type
			if err := types.Check(
				ctx.Constraints,
				*chanType.ValueType,
				paramType,
				ctx.AST,
				"channel to stage parameter connection",
			); err != nil {
				ctx.Diagnostics.AddError(errors.Newf(
					"channel %s value type %s does not match stage %s parameter type %s",
					channelName,
					chanType.ValueType,
					name,
					paramType,
				), ctx.AST)
				return false
			}
		}
	} else if prevExpr := prevNode.Expression(); prevExpr != nil {
		exprType := types.InferFromExpression(context.Child(ctx, prevExpr))
		if stageType.Type.Inputs.Count() > 0 {
			_, paramType := stageType.Type.Inputs.At(0)
			if err := types.Check(
				ctx.Constraints,
				exprType,
				paramType,
				ctx.AST,
				"expression to stage parameter connection",
			); err != nil {
				ctx.Diagnostics.AddError(errors.Newf(
					"expression type %s does not match stage %s parameter type %s",
					exprType,
					name,
					paramType,
				), ctx.AST)
				return false
			}
		}
	} else if prevStageNode := prevNode.StageInvocation(); prevStageNode != nil {
		prevStageName := prevStageNode.IDENTIFIER().GetText()
		prevStageType, ok := resolveStage(ctx, prevStageName)
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

		if !hasRoutingTableBetween && stageType.Type.Inputs.Count() > 1 {
			ctx.Diagnostics.AddError(
				errors.Newf("%s has more than one parameter", name),
				ctx.AST,
			)
			return false
		}
		if !hasRoutingTableBetween && stageType.Type.Inputs.Count() > 0 {
			_, t := stageType.Type.Inputs.At(0)
			var prevOutputType types2.Type
			if outputType, ok := prevStageType.Type.Outputs.Get(ir.DefaultOutputParam); ok {
				prevOutputType = outputType
			} else if prevStageType.Type.Outputs.Count() > 0 {
				ctx.Diagnostics.AddError(errors.Newf(
					"stage '%s' has named outputs and requires a routing table",
					prevStageName,
				), ctx.AST)
				return false
			}
			if err := types.Check(ctx.Constraints, prevOutputType, t, ctx.AST,
				"flow connection between stages"); err != nil {
				ctx.Diagnostics.AddError(errors.Newf(
					"return type %s of %s is not equal to argument type %s of %s",
					prevOutputType,
					prevStageName,
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

func resolveStage[T antlr.ParserRuleContext](
	ctx context.Context[T],
	name string,
) (*symbol.Scope, bool) {
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return nil, false
	}
	if sym.Kind != symbol.KindFunction {
		ctx.Diagnostics.AddError(errors.Newf("%s is not a stage", name), ctx.AST)
		return nil, false
	}
	return sym, true
}

func validateStageConfig[T antlr.ParserRuleContext](
	ctx context.Context[T],
	stageName string,
	stageType types2.Type,
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
			expectedType, exists := stageType.Config.Get(key)
			if !exists {
				ctx.Diagnostics.AddError(
					errors.Newf("unknown config parameter '%s' for stage '%s'", key, stageName),
					configVal,
				)
				return nil, false
			}
			if expr := configVal.Expression(); expr != nil {
				childCtx := context.Child(ctx, expr)
				if !expression.Analyze(childCtx) {
					return nil, false
				}
				exprType := types.InferFromExpression(childCtx)
				if err := types.Check(ctx.Constraints, expectedType, exprType, configVal,
					"config parameter '"+key+"' for stage '"+stageName+"'"); err != nil {
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

	for paramName := range stageType.Config.Iter() {
		if !configParams[paramName] {
			ctx.Diagnostics.AddError(
				errors.Newf("missing required config parameter '%s' for stage '%s'", paramName, stageName),
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

	// Since there should only be one routing table per flow statement, check if this is it
	if len(tables) != 1 || tables[0] != ctx.AST {
		ctx.Diagnostics.AddError(errors.New("unexpected routing table configuration"), ctx.AST)
		return false
	}

	// Separate flow nodes into before and after the routing table
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

	// Determine if this is input or output routing
	if len(nodesBefore) == 0 && len(nodesAfter) > 0 {
		// Routing table comes first - this is INPUT routing
		return analyzeInputRoutingTable(ctx, nodesAfter)
	} else if len(nodesBefore) > 0 {
		// Routing table comes after - this is OUTPUT routing
		return analyzeOutputRoutingTable(ctx, nodesBefore, nodesAfter)
	} else {
		ctx.Diagnostics.AddError(errors.New("routing table must have associated flow nodes"), ctx.AST)
		return false
	}
}

func analyzeOutputRoutingTable(ctx context.Context[parser.IRoutingTableContext], nodesBefore []parser.IFlowNodeContext, nodesAfter []parser.IFlowNodeContext) bool {
	var prevStage parser.IStageInvocationContext
	for i := len(nodesBefore) - 1; i >= 0; i-- {
		if stageInv := nodesBefore[i].StageInvocation(); stageInv != nil {
			prevStage = stageInv
			break
		}
	}

	if prevStage == nil {
		ctx.Diagnostics.AddError(errors.New("output routing table must follow a stage invocation"), ctx.AST)
		return false
	}

	stageName := prevStage.IDENTIFIER().GetText()
	stageType, ok := resolveStage(ctx, stageName)
	if !ok {
		return false
	}

	_, hasDefaultOutput := stageType.Type.Outputs.Get(ir.DefaultOutputParam)
	hasNamedOutputs := stageType.Type.Outputs.Count() > 1 || (stageType.Type.Outputs.Count() == 1 && !hasDefaultOutput)
	if !hasNamedOutputs {
		ctx.Diagnostics.AddError(
			errors.Newf("stage '%s' does not have named outputs, cannot use routing table", stageName),
			ctx.AST,
		)
		return false
	}

	// Find the next stage after the routing table (if exists)
	var (
		nextStage     parser.IStageInvocationContext
		nextStageType types2.Type
	)
	for _, node := range nodesAfter {
		if stageInv := node.StageInvocation(); stageInv != nil {
			nextStage = stageInv
			nextStageName := nextStage.IDENTIFIER().GetText()
			nextStageScope, err := ctx.Scope.Resolve(ctx, nextStageName)
			if err == nil && nextStageScope.Kind == symbol.KindFunction {
				nextStageType = nextStageScope.Type
			}
			break
		}
	}

	// Analyze each routing entry
	for _, entry := range ctx.AST.AllRoutingEntry() {
		outputName := entry.IDENTIFIER(0).GetText()

		// Verify the output exists in the stage
		outputType, exists := stageType.Type.Outputs.Get(outputName)
		if !exists {
			ctx.Diagnostics.AddError(
				errors.Newf("stage '%s' does not have output '%s'", stageName, outputName),
				entry,
			)
			return false
		}

		// Check if there's a trailing parameter name
		var targetParamName string
		if len(entry.AllIDENTIFIER()) > 1 {
			targetParamName = entry.IDENTIFIER(1).GetText()

			// Verify the next stage exists and has this parameter
			if nextStage == nil {
				ctx.Diagnostics.AddError(
					errors.New("parameter mapping requires a stage after the routing table"),
					entry,
				)
				return false
			}

			if _, exists := nextStageType.Inputs.Get(targetParamName); !exists {
				ctx.Diagnostics.AddError(
					errors.Newf(
						"stage '%s' does not have parameter '%s'",
						nextStage.IDENTIFIER().GetText(),
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
			if !analyzeRoutingTargetWithParam(context.Child(ctx, flowNode), outputType, nextStageType, targetParam) {
				return false
			}
		}
	}

	return true
}

func analyzeInputRoutingTable(ctx context.Context[parser.IRoutingTableContext], nodes []parser.IFlowNodeContext) bool {
	// Find the first stage invocation in the flow nodes (the stage after routing table)
	var nextStage parser.IStageInvocationContext
	for i := 0; i < len(nodes); i++ {
		if stageInv := nodes[i].StageInvocation(); stageInv != nil {
			nextStage = stageInv
			break
		}
	}

	if nextStage == nil {
		ctx.Diagnostics.AddError(errors.New("input routing table must precede a stage invocation"), ctx.AST)
		return false
	}

	stageName := nextStage.IDENTIFIER().GetText()
	stageType, ok := resolveStage(ctx, stageName)
	if !ok {
		return false
	}

	// Analyze each routing entry
	// Format: source -> processing... -> parameterName
	for _, entry := range ctx.AST.AllRoutingEntry() {
		flowNodes := entry.AllFlowNode()
		if len(flowNodes) == 0 {
			ctx.Diagnostics.AddError(errors.New("routing entry must have at least one target"), entry)
			return false
		}

		// The LAST flow node should be a channel identifier representing the parameter name
		lastNode := flowNodes[len(flowNodes)-1]
		if lastNode.ChannelIdentifier() == nil {
			ctx.Diagnostics.AddError(
				errors.New("last element in input routing entry must be a parameter name (identifier)"),
				lastNode,
			)
			return false
		}

		paramName := lastNode.ChannelIdentifier().IDENTIFIER().GetText()

		// Verify the parameter exists in the stage
		paramType, exists := stageType.Type.Inputs.Get(paramName)
		if !exists {
			ctx.Diagnostics.AddError(
				errors.Newf("stage '%s' does not have parameter '%s'", stageName, paramName),
				lastNode,
			)
			return false
		}

		// Analyze the flow chain: source (entry.IDENTIFIER) -> processing nodes -> parameter
		// For type checking, we need to verify the output type of the chain matches paramType
		// TODO: Implement full type checking for the flow chain
		_ = paramType

		// Analyze intermediate processing nodes
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
	sourceType types2.Type,
	nextStageType types2.Type,
	targetParam *string,
) bool {
	// Handle stage invocation target
	if stageInv := ctx.AST.StageInvocation(); stageInv != nil {
		stageName := stageInv.IDENTIFIER().GetText()
		stageType, ok := resolveStage(ctx, stageName)
		if !ok {
			return false
		}

		if _, ok := validateStageConfig(ctx, stageName, stageType.Type, stageInv.ConfigValues(), stageInv); !ok {
			return false
		}

		if targetParam != nil {
			var outputType types2.Type
			if outType, ok := stageType.Type.Outputs.Get(ir.DefaultOutputParam); ok {
				outputType = outType
			} else if stageType.Type.Outputs.Count() > 0 {
				ctx.Diagnostics.AddError(errors.Newf(
					"stage '%s' has named outputs and requires explicit output selection",
					stageName,
				), ctx.AST)
				return false
			}

			if paramType, exists := nextStageType.Inputs.Get(*targetParam); exists {
				if err := types.Check(ctx.Constraints, outputType, paramType, ctx.AST,
					"routing table parameter mapping"); err != nil {
					ctx.Diagnostics.AddError(errors.Newf(
						"type mismatch: stage %s output type %s does not match target parameter %s type %s",
						stageName,
						outputType,
						*targetParam,
						paramType,
					), ctx.AST)
					return false
				}
			}
		} else {
			if stageType.Type.Inputs.Count() > 0 {
				_, paramType := stageType.Type.Inputs.At(0)
				if err := types.Check(ctx.Constraints, sourceType, paramType, ctx.AST,
					"routing table output to stage parameter"); err != nil {
					ctx.Diagnostics.AddError(errors.Newf(
						"type mismatch: output type %s does not match stage %s parameter type %s",
						sourceType,
						stageName,
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

		valueType := channelSym.Type.ValueType
		if err := types.Check(
			ctx.Constraints,
			sourceType,
			*valueType,
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
