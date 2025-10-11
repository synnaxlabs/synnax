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
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
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
	// Step 1: Check that a symbol for the stage exists and it has the right type
	name := ctx.AST.IDENTIFIER().GetText()
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	} else if sym.Kind != ir.KindStage {
		ctx.Diagnostics.AddError(errors.Newf("%s is not a stage", name), ctx.AST)
		return false
	}

	// Step 2: Validate configuration parameters
	var stageType ir.Stage
	if sym.Type != nil {
		stageType, _ = sym.Type.(ir.Stage)
	}
	configParams := make(map[string]bool)
	// Step 2A: Check for mismatch
	if configBlock := ctx.AST.ConfigValues(); configBlock != nil {
		if namedVals := configBlock.NamedConfigValues(); namedVals != nil {
			for _, configVal := range namedVals.AllNamedConfigValue() {
				key := configVal.IDENTIFIER().GetText()
				configParams[key] = true
				expectedType, exists := stageType.Config.Get(key)
				if !exists {
					ctx.Diagnostics.AddError(errors.Newf("unknown config parameter '%s' for stage '%s'", key, name), configVal)
					return false
				}
				if expr := configVal.Expression(); expr != nil {
					childCtx := context.Child(ctx, expr)
					if !expression.Analyze(childCtx) {
						return false
					}
					exprType := atypes.InferFromExpression(childCtx)
					if exprType != nil && expectedType != nil {
						// Use constraint-based checking for type variables
						if err := atypes.CheckEqual(ctx.Constraints, expectedType, exprType, configVal,
							"config parameter '"+key+"' for stage '"+name+"'"); err != nil {
							// Only report error if neither type is a type variable
							if !atypes.HasTypeVariables(expectedType) && !atypes.HasTypeVariables(exprType) {
								ctx.Diagnostics.AddError(
									errors.Newf(
										"type mismatch: config parameter '%s' expects %s but got %s",
										key,
										expectedType,
										exprType,
									),
									configVal,
								)
								return false
							}
						}
					}
				}
			}
		} else if anonVals := configBlock.AnonymousConfigValues(); anonVals != nil {
			ctx.Diagnostics.AddError(
				errors.Newf("anonymous configuration values are not supported"),
				configBlock.AnonymousConfigValues(),
			)
		}
	}
	// Step 2B: Check for missing required config parameters
	for paramName := range stageType.Config.Iter() {
		if !configParams[paramName] {
			ctx.Diagnostics.AddError(
				errors.Newf("missing required config parameter '%s' for stage '%s'", paramName, name),
				ctx.AST,
			)
			return false
		}
	}
	if prevNode == nil {
		return true
	}

	// Step 3: Validate that stage arguments are compatible with previous flow node.
	// Check if previous node is a channel
	if prevChannelNode := prevNode.ChannelIdentifier(); prevChannelNode != nil {
		channelName := prevChannelNode.IDENTIFIER().GetText()
		channelSym, err := ctx.Scope.Resolve(ctx, channelName)
		if err != nil {
			ctx.Diagnostics.AddError(err, prevChannelNode)
			return false
		}
		if channelSym.Kind != ir.KindChannel {
			ctx.Diagnostics.AddError(
				errors.Newf("%s is not a channel", channelName),
				prevChannelNode,
			)
			return false
		}
		// If this stage has parameters, the channel's value type should match the first param
		if stageType.Params.Count() > 0 {
			_, paramType := stageType.Params.At(0)
			chanType := channelSym.Type.(ir.Chan)
			// Create constraint between channel value type and parameter type
			if err := atypes.CheckEqual(ctx.Constraints, chanType.ValueType, paramType, ctx.AST,
				"channel to stage parameter connection"); err != nil {
				// Only report error if neither type is a type variable
				if !atypes.HasTypeVariables(chanType.ValueType) && !atypes.HasTypeVariables(paramType) {
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
		}
	} else if prevExpr := prevNode.Expression(); prevExpr != nil {
		// Handle expression -> stage connection
		// The expression creates a synthetic stage, so we need to get its return type
		exprType := atypes.InferFromExpression(context.Child(ctx, prevExpr))

		// If this stage has parameters, the expression's type should match the first param
		if stageType.Params.Count() > 0 {
			_, paramType := stageType.Params.At(0)
			// Create constraint between expression type and parameter type
			if err := atypes.CheckEqual(ctx.Constraints, exprType, paramType, ctx.AST,
				"expression to stage parameter connection"); err != nil {
				// Only report error if neither type is a type variable
				if !atypes.HasTypeVariables(exprType) && !atypes.HasTypeVariables(paramType) {
					ctx.Diagnostics.AddError(errors.Newf(
						"expression type %s does not match stage %s parameter type %s",
						exprType,
						name,
						paramType,
					), ctx.AST)
					return false
				}
			}
		}
	} else if prevTaskNode := prevNode.StageInvocation(); prevTaskNode != nil {
		prevTaskName := prevTaskNode.IDENTIFIER().GetText()
		// lookup stage in symbol table
		sym, err := ctx.Scope.Resolve(ctx, prevTaskName)
		if err != nil {
			ctx.Diagnostics.AddError(err, prevTaskNode)
			return false
		}
		var prevTaskType ir.Stage
		if sym.Kind != ir.KindStage {
			ctx.Diagnostics.AddError(
				errors.Newf("%s is not a stage", prevTaskNode),
				prevTaskNode,
			)
			prevTaskType, _ = sym.Type.(ir.Stage)
			return false
		}
		if stageType.Params.Count() > 1 {
			ctx.Diagnostics.AddError(
				errors.Newf("%s has more than one parameter", name),
				ctx.AST,
			)
			return false
		}
		// Validate that the return type of the previous stage matches the arg type
		// of the next stage.
		if stageType.Params.Count() > 0 {
			_, t := stageType.Params.At(0)
			// Use constraint-based checking for type variables
			if err := atypes.CheckEqual(ctx.Constraints, prevTaskType.Return, t, ctx.AST,
				"flow connection between stages"); err != nil {
				// Only report error if neither type is a type variable
				if !atypes.HasTypeVariables(prevTaskType.Return) && !atypes.HasTypeVariables(t) {
					ctx.Diagnostics.AddError(errors.Newf(
						"return type %s of %s is not equal to argument type %s of %s",
						prevTaskType.Return,
						prevTaskName,
						t,
						name,
					), ctx.AST)
					return false
				}
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
