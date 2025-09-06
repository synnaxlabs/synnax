// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/result"
	"github.com/synnaxlabs/arc/analyzer/statement"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

type Options struct {
	Resolver symbol.Resolver
}

type Result = result.Result

func Analyze(
	prog parser.IProgramContext,
	opts Options,
) Result {
	rootScope := symbol.CreateRoot(opts.Resolver)
	res := result.Result{Symbols: rootScope}
	// First pass: collect declarations with empty type signatures
	for _, item := range prog.AllTopLevelItem() {
		if fn := item.FunctionDeclaration(); fn != nil {
			name := fn.IDENTIFIER().GetText()
			_, err := rootScope.Add(symbol.Symbol{
				Name:       name,
				Kind:       symbol.KindFunction,
				Type:       types.NewFunction(),
				ParserRule: fn,
			})
			if err != nil {
				res.AddError(err, fn)
				return res
			}
		} else if task := item.TaskDeclaration(); task != nil {
			name := task.IDENTIFIER().GetText()
			_, err := rootScope.Add(symbol.Symbol{
				Name:       name,
				Kind:       symbol.KindTask,
				Type:       types.NewTask(),
				ParserRule: task,
			})
			if err != nil {
				res.AddError(err, task)
				return res
			}
		}
	}

	// Second pass: analyze tree
	for _, item := range prog.AllTopLevelItem() {
		if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			if !analyzeFunctionDeclaration(rootScope, &res, funcDecl) {
				return res
			}
		} else if taskDecl := item.TaskDeclaration(); taskDecl != nil {
			if !analyzeTaskDeclaration(rootScope, &res, taskDecl) {
				return res
			}
		} else if flowStmt := item.FlowStatement(); flowStmt != nil {
			if !flow.Analyze(rootScope, &res, flowStmt) {
				return res
			}
		}
	}
	return res
}

func AnalyzeStatement(
	stmt parser.IStatementContext,
	opts Options,
) Result {
	scope := symbol.CreateRoot(opts.Resolver)
	res := result.Result{Symbols: scope}
	statement.Analyze(scope, &res, stmt)
	return res
}

func AnalyzeBlock(
	block parser.IBlockContext,
	opts Options,
) Result {
	scope := symbol.CreateRoot(opts.Resolver)
	res := result.Result{Symbols: scope}
	statement.AnalyzeBlock(scope, &res, block)
	return res
}

// analyzeFunctionDeclaration analyzes a function declaration
func analyzeFunctionDeclaration(
	parentScope *symbol.Scope,
	result *result.Result,
	fn parser.IFunctionDeclarationContext,
) bool {
	name := fn.IDENTIFIER().GetText()
	fnScope, err := parentScope.Resolve(name)
	if err != nil {
		result.AddError(err, fn)
		return false
	}
	fnType := fnScope.Type.(types.Function)
	if !analyzeParams(
		fnScope,
		result,
		fn.ParameterList(),
		&fnType.Params,
	) {
		return false
	}
	if retType := fn.ReturnType(); retType != nil {
		if typeCtx := retType.Type_(); typeCtx != nil {
			fnType.Return, _ = atypes.InferFromTypeContext(typeCtx)
		}
	}
	fnScope.Type = fnType
	if block := fn.Block(); block != nil {
		if !statement.AnalyzeBlock(fnScope, result, block) {
			return false
		}

		// Check if the function has a return type and if all paths return
		if fnType.Return != nil && !blockAlwaysReturns(block) {
			result.AddError(
				errors.Newf("function '%s' must return a value of type %s on all paths", name, fnType.Return),
				fn,
			)
			return false
		}
	}
	return true
}

func analyzeParams(
	scope *symbol.Scope,
	result *result.Result,
	params parser.IParameterListContext,
	paramTypes *types.OrderedMap[string, types.Type],
) bool {
	if params == nil {
		return true
	}
	for _, param := range params.AllParameter() {
		var paramType types.Type
		if typeCtx := param.Type_(); typeCtx != nil {
			paramType, _ = atypes.InferFromTypeContext(typeCtx)
		}
		paramName := param.IDENTIFIER().GetText()
		if !paramTypes.Put(paramName, paramType) {
			result.AddError(
				errors.Newf("duplicate parameter %s", paramName),
				param,
			)
			return false
		}

		// Also add to scope for use within task body
		if _, err := scope.Add(symbol.Symbol{
			Name:       paramName,
			Kind:       symbol.KindParam,
			Type:       paramType,
			ParserRule: param,
		}); err != nil {
			result.AddError(err, param)
			return false
		}
	}
	return true
}

// blockAlwaysReturns checks if a block always returns a value on all execution paths
func blockAlwaysReturns(block parser.IBlockContext) bool {
	if block == nil {
		return false
	}
	statements := block.AllStatement()
	if len(statements) == 0 {
		return false
	}
	// Check statements from last to first
	for i := len(statements) - 1; i >= 0; i-- {
		stmt := statements[i]
		// Check if it's a return statement
		if stmt.ReturnStatement() != nil {
			return true
		}
		// Check if it's an if statement that covers all paths
		if ifStmt := stmt.IfStatement(); ifStmt != nil {
			// An if statement guarantees a return only if:
			// 1. It has an else clause
			// 2. All branches (if, else-ifs, else) return
			if ifStmt.ElseClause() != nil {
				// Check if the main if block returns
				if !blockAlwaysReturns(ifStmt.Block()) {
					continue // This if doesn't guarantee return
				}
				// Check all else-if branches
				allElseIfsReturn := true
				for _, elseIf := range ifStmt.AllElseIfClause() {
					if !blockAlwaysReturns(elseIf.Block()) {
						allElseIfsReturn = false
						break
					}
				}
				if !allElseIfsReturn {
					continue
				}
				if blockAlwaysReturns(ifStmt.ElseClause().Block()) {
					return true //
				}
			}
			// No else or not all branches return, continue checking previous statements
		}
		// For other statement types, continue checking previous statements
		// (assignments, expressions, etc. don't affect return paths)
	}
	return false
}

func analyzeTaskDeclaration(
	parentScope *symbol.Scope,
	result *result.Result,
	task parser.ITaskDeclarationContext,
) bool {
	name := task.IDENTIFIER().GetText()
	taskScope, err := parentScope.Resolve(name)
	if err != nil {
		result.AddError(err, task)
		return false
	}
	taskType := taskScope.Type.(types.Task)
	if configBlock := task.ConfigBlock(); configBlock != nil {
		for _, param := range configBlock.AllConfigParameter() {
			paramName := param.IDENTIFIER().GetText()
			var configType types.Type
			if typeCtx := param.Type_(); typeCtx != nil {
				configType, _ = atypes.InferFromTypeContext(typeCtx)
			}
			if !taskType.Config.Put(paramName, configType) {
				result.AddError(errors.Newf("duplicate configuration parameter %s", param), task)
			}
			_, err := taskScope.Add(symbol.Symbol{
				Name:       paramName,
				Kind:       symbol.KindConfigParam,
				Type:       configType,
				ParserRule: param,
			})
			if err != nil {
				result.AddError(err, param)
				return false
			}
		}
	}

	if !analyzeParams(
		taskScope,
		result,
		task.ParameterList(),
		&taskType.Params,
	) {
		return false
	}

	// Parse return type
	if retType := task.ReturnType(); retType != nil {
		if typeCtx := retType.Type_(); typeCtx != nil {
			taskType.Return, _ = atypes.InferFromTypeContext(typeCtx)
		}
	}
	taskScope.Type = taskType
	if block := task.Block(); block != nil {
		if !statement.AnalyzeBlock(taskScope, result, block) {
			return false
		}

		// Check if the task has a return type and if all paths return
		if taskType.Return != nil && !blockAlwaysReturns(block) {
			result.AddError(
				errors.Newf("task '%s' must return a value of type %s on all paths", name, taskType.Return),
				task,
			)
			return false
		}
	}
	return true
}
