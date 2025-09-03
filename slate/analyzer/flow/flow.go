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
	"github.com/synnaxlabs/slate/analyzer/expression"
	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	atypes "github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// Visit processes a flow statement and returns true if successful
func Visit(scope *symbol.Scope, res *result.Result, ctx parser.IFlowStatementContext) bool {
	for i, flowNode := range ctx.AllFlowNode() {
		var prevFlowNode parser.IFlowNodeContext
		if i != 0 {
			prevFlowNode = ctx.FlowNode(i - 1)
		}
		if !visitSource(scope, res, prevFlowNode, flowNode) {
			return false
		}
	}
	return true
}

func visitSource(
	scope *symbol.Scope,
	res *result.Result,
	prevNode parser.IFlowNodeContext,
	currNode parser.IFlowNodeContext,
) bool {
	// Case 1: Source is a direct task invocation.
	if taskInv := currNode.TaskInvocation(); taskInv != nil {
		return parseTaskInvocation(
			scope,
			res,
			prevNode,
			taskInv,
		)
	}
	// Case 2: Source is a channel identifier
	if channelID := currNode.ChannelIdentifier(); channelID != nil {
		return visitChannel(scope, res, channelID)
	}
	// Case 3: Source is an expression
	if expr := currNode.Expression(); expr != nil {
		return expression.Visit(scope, res, expr)
	}

	res.AddError(errors.New("invalid flow source"), currNode)
	return true
}

func parseTaskInvocation(
	scope *symbol.Scope,
	res *result.Result,
	prevNode parser.IFlowNodeContext,
	task parser.ITaskInvocationContext,
) bool {
	// Step 1: Check that a symbol for the task exists and it has the right type
	name := task.IDENTIFIER().GetText()
	sym, err := scope.Get(name)
	if err != nil {
		res.AddError(err, task)
		return false
	} else if sym.Symbol != nil && sym.Symbol.Kind != symbol.KindTask {
		res.AddError(errors.Newf("%s is not a task", name), task)
		return false
	}

	// Step 2: Validate configuration parameters
	var taskType types.Task
	if sym.Symbol != nil && sym.Symbol.Type != nil {
		taskType, _ = sym.Symbol.Type.(types.Task)
	}
	configParams := make(map[string]bool)
	// Step 2A: Check for mismatch
	if configBlock := task.ConfigValues(); configBlock != nil {
		if namedVals := configBlock.NamedConfigValues(); namedVals != nil {
			for _, configVal := range namedVals.AllNamedConfigValue() {
				key := configVal.IDENTIFIER().GetText()
				configParams[key] = true
				expectedType, exists := taskType.Config.Get(key)
				if !exists {
					res.AddError(errors.Newf("unknown config parameter '%s' for task '%s'", key, name), configVal)
					return false
				}
				if expr := configVal.Expression(); expr != nil {
					if !expression.Visit(scope, res, expr) {
						return false
					}
					exprType := atypes.InferFromExpression(scope, expr)
					if exprType != nil && expectedType != nil {
						if !atypes.Compatible(expectedType, exprType) {
							res.AddError(
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
			res.AddError(
				errors.Newf("anonymous configuration values are not supported"),
				configBlock.AnonymousConfigValues(),
			)
		}
	}
	// Step 2B: Check for missing required config parameters
	for paramName := range taskType.Config.Iter() {
		if !configParams[paramName] {
			res.AddError(errors.Newf("missing required config parameter '%s' for task '%s'", paramName, name), task)
			return false
		}
	}
	if prevNode == nil {
		return true
	}

	// Step 3: Validate that task arguments are compatible with previous flow node.
	if prevTaskNode := prevNode.TaskInvocation(); prevTaskNode != nil {
		prevTaskName := prevTaskNode.IDENTIFIER().GetText()
		// lookup task in symbol table
		sym, err := scope.Get(prevTaskName)
		if err != nil {
			res.AddError(err, prevTaskNode)
			return false
		}
		var prevTaskType types.Task
		if sym.Symbol != nil && sym.Symbol.Kind != symbol.KindTask {
			res.AddError(errors.Newf("%s is not a task", prevTaskNode), prevTaskNode)
			prevTaskType, _ = sym.Symbol.Type.(types.Task)
			return false
		}
		if taskType.Params.Count() > 1 {
			res.AddError(errors.Newf("%s has more than one parameter", name), task)
			return false
		}
		// Validate that the return type of the previous task matches the arg type
		// of the ntext task.
		if taskType.Params.Count() > 0 && !types.Equal(prevTaskType.Return, taskType.Params.At(0)) {
			res.AddError(errors.Newf(
				"return type %s of %s is not equal to argument type %s of %s",
				prevTaskType.Return,
				prevTaskName,
				name,
				taskType.Params.At(0),
			), task)
		}
	}
	return true
}

func visitChannel(scope *symbol.Scope, res *result.Result, ch parser.IChannelIdentifierContext) bool {
	name := ch.IDENTIFIER().GetText()
	_, err := scope.Get(name)
	if err != nil {
		res.AddError(err, ch)
		return false
	}
	return true
}
