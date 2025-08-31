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
	"fmt"

	"github.com/synnaxlabs/slate/analyzer/expression"
	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

// FlowStatement represents a parsed flow statement in the inter-task layer
type FlowStatement struct {
	Source Source
	Target Target
}

// Source represents the source of a flow (channel, task invocation, or expression)
type Source struct {
	ChannelID      string
	TaskInvocation *TaskInvocation
	Expression     parser.IExpressionContext
}

// Target represents the target of a flow (channel or task invocation)
type Target struct {
	ChannelID      string
	TaskInvocation *TaskInvocation
}

// TaskInvocation represents a task invocation with config and arguments
type TaskInvocation struct {
	TaskName   string
	ConfigVals map[string]interface{}
	Arguments  []parser.IExpressionContext
}

// Visit processes a flow statement and returns true if successful
func Visit(scope *symbol.Scope, res *result.Result, ctx parser.IFlowStatementContext) bool {
	source := parseSource(scope, res, ctx.FlowSource())
	if source == nil {
		return false
	}

	// Flow statements can have multiple targets chained with arrows
	// For now, just parse the first target
	if ctx.AllFlowTarget() == nil || len(ctx.AllFlowTarget()) == 0 {
		res.AddError(errors.New("flow statement must have at least one target"), ctx)
		return false
	}
	target := parseTarget(scope, res, ctx.FlowTarget(0))
	if target == nil {
		return false
	}

	// Store flow information in result context (to be implemented)
	// For now, just validate the flow
	return true
}

func parseSource(scope *symbol.Scope, res *result.Result, ctx parser.IFlowSourceContext) *Source {
	if ctx == nil {
		return nil
	}

	source := &Source{}

	if channelID := ctx.ChannelIdentifier(); channelID != nil {
		source.ChannelID = channelID.GetText()
		// Channels are external, so we don't validate them here
		return source
	}

	if taskInv := ctx.TaskInvocation(); taskInv != nil {
		source.TaskInvocation = parseTaskInvocation(scope, res, taskInv)
		if source.TaskInvocation == nil {
			return nil
		}
		return source
	}

	if expr := ctx.Expression(); expr != nil {
		// Validate expression
		if !expression.Visit(scope, res, expr) {
			return nil
		}
		source.Expression = expr
		return source
	}

	res.AddError(errors.New("invalid flow source"), ctx)
	return nil
}

func parseTarget(scope *symbol.Scope, res *result.Result, ctx parser.IFlowTargetContext) *Target {
	if ctx == nil {
		return nil
	}

	target := &Target{}

	if channelID := ctx.ChannelIdentifier(); channelID != nil {
		target.ChannelID = channelID.GetText()
		// Channels are external, so we don't validate them here
		return target
	}

	if taskInv := ctx.TaskInvocation(); taskInv != nil {
		target.TaskInvocation = parseTaskInvocation(scope, res, taskInv)
		if target.TaskInvocation == nil {
			return nil
		}
		return target
	}

	res.AddError(errors.New("invalid flow target"), ctx)
	return nil
}

func parseTaskInvocation(scope *symbol.Scope, res *result.Result, ctx parser.ITaskInvocationContext) *TaskInvocation {
	if ctx == nil {
		return nil
	}

	inv := &TaskInvocation{
		TaskName:   ctx.IDENTIFIER().GetText(),
		ConfigVals: make(map[string]interface{}),
		Arguments:  []parser.IExpressionContext{},
	}

	// Check if task exists
	sym, err := scope.Get(inv.TaskName)
	if err != nil {
		res.AddError(err, ctx)
		return nil
	} else if sym.Symbol != nil && sym.Symbol.Kind != symbol.KindTask {
		res.AddError(errors.Newf("%s is not a task", inv.TaskName), ctx)
		return nil
	}

	// Get the task type signature
	var taskType types.Task
	if sym.Symbol != nil && sym.Symbol.Type != nil {
		taskType, _ = sym.Symbol.Type.(types.Task)
	}

	// Parse config values if present
	providedParams := make(map[string]bool)
	if configBlock := ctx.ConfigValues(); configBlock != nil {
		if namedVals := configBlock.NamedConfigValues(); namedVals != nil {
			for _, configVal := range namedVals.AllNamedConfigValue() {
				key := configVal.IDENTIFIER().GetText()
				providedParams[key] = true
				
				// Check if this parameter exists in the task signature
				if taskType.Config != nil {
					expectedType, exists := taskType.Config[key]
					if !exists {
						res.AddError(errors.Newf("unknown config parameter '%s' for task '%s'", key, inv.TaskName), configVal)
						return nil
					}
					
					// Type check the expression
					if expr := configVal.Expression(); expr != nil {
						// Validate the expression
						if !expression.Visit(scope, res, expr) {
							return nil
						}
						
						// Check type compatibility
						exprType := types.InferFromExpression(scope, expr)
						if exprType != nil && expectedType != nil {
							if !types.Compatible(expectedType, exprType) {
								res.AddError(
									errors.Newf("type mismatch: config parameter '%s' expects %s but got %s", 
										key, expectedType, exprType),
									configVal,
								)
								return nil
							}
						}
						
						inv.ConfigVals[key] = expr
					}
				} else {
					// If no type info, just validate and store the expression
					if expr := configVal.Expression(); expr != nil {
						if !expression.Visit(scope, res, expr) {
							return nil
						}
						inv.ConfigVals[key] = expr
					}
				}
			}
		} else if anonVals := configBlock.AnonymousConfigValues(); anonVals != nil {
			// Anonymous config values are just expressions
			for i, expr := range anonVals.AllExpression() {
				// Store with numeric keys for anonymous values
				inv.ConfigVals[fmt.Sprintf("%d", i)] = expr
			}
		}
	}
	
	// Check for missing required config parameters
	if taskType.Config != nil {
		for paramName := range taskType.Config {
			if !providedParams[paramName] {
				res.AddError(errors.Newf("missing required config parameter '%s' for task '%s'", paramName, inv.TaskName), ctx)
				return nil
			}
		}
	}

	// Parse arguments if present
	if argList := ctx.Arguments(); argList != nil {
		if args := argList.ArgumentList(); args != nil {
			inv.Arguments = args.AllExpression()
		}
	}

	return inv
}
