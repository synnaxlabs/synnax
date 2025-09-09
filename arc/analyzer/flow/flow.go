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
	sym, err := ctx.Scope.Resolve(name)
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
					if !expression.Analyze(context.Child(ctx, expr)) {
						return false
					}
					exprType := atypes.InferFromExpression(ctx.Scope, expr, nil)
					if exprType != nil && expectedType != nil {
						if !atypes.Compatible(expectedType, exprType) {
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
	if prevTaskNode := prevNode.StageInvocation(); prevTaskNode != nil {
		prevTaskName := prevTaskNode.IDENTIFIER().GetText()
		// lookup stage in symbol table
		sym, err := ctx.Scope.Resolve(prevTaskName)
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
		// of the ntext stage.
		if stageType.Params.Count() > 0 {
			_, t := stageType.Params.At(0)
			if !ir.Equal(prevTaskType.Return, t) {
				ctx.Diagnostics.AddError(errors.Newf(
					"return type %s of %s is not equal to argument type %s of %s",
					prevTaskType.Return,
					prevTaskName,
					t,
					name,
				), ctx.AST)
			}
		}
	}
	return true
}

func analyzeChannel(ctx context.Context[parser.IChannelIdentifierContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	_, err := ctx.Scope.Resolve(name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return true
}
