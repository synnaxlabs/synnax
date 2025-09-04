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
	"github.com/synnaxlabs/slate/analyzer/flow"
	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/analyzer/statement"
	atypes "github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

type Options struct {
	Resolver symbol.Resolver
}

type Result = result.Result

func newRootScope(opts Options) *symbol.Scope {
	return &symbol.Scope{GlobalResolver: opts.Resolver, Counter: new(int)}
}

func Analyze(
	prog parser.IProgramContext,
	opts Options,
) Result {
	rootScope := newRootScope(opts)
	res := result.Result{Symbols: rootScope}
	// First pass: collect declarations with empty type signatures
	for _, item := range prog.AllTopLevelItem() {
		if fn := item.FunctionDeclaration(); fn != nil {
			name := fn.IDENTIFIER().GetText()
			_, err := rootScope.AddSymbol(name, symbol.KindFunction, types.NewFunction(), fn)
			if err != nil {
				res.AddError(err, fn)
				return res
			}
		} else if task := item.TaskDeclaration(); task != nil {
			name := task.IDENTIFIER().GetText()
			_, err := rootScope.AddSymbol(
				name,
				symbol.KindTask,
				types.NewTask(),
				task,
			)
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
	scope := newRootScope(opts)
	res := result.Result{Symbols: scope}
	statement.Analyze(scope, &res, stmt)
	return res
}

func AnalyzeBlock(
	block parser.IBlockContext,
	opts Options,
) Result {
	scope := newRootScope(opts)
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
	fnScope, err := parentScope.Get(name)
	if err != nil {
		result.AddError(err, fn)
		return false
	}
	fnType := fnScope.Symbol.Type.(types.Function)
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
	fnScope.Symbol.Type = fnType
	if block := fn.Block(); block != nil {
		statement.AnalyzeBlock(fnScope, result, block)
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
		if _, err := scope.AddSymbol(
			paramName,
			symbol.KindParam,
			paramType,
			param,
		); err != nil {
			result.AddError(err, param)
			return false
		}
	}
	return true
}

func analyzeTaskDeclaration(
	parentScope *symbol.Scope,
	result *result.Result,
	task parser.ITaskDeclarationContext,
) bool {
	name := task.IDENTIFIER().GetText()
	taskScope, err := parentScope.Get(name)
	if err != nil {
		result.AddError(err, task)
		return false
	}

	taskType := taskScope.Symbol.Type.(types.Task)
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
			_, err := taskScope.AddSymbol(
				paramName,
				symbol.KindConfigParam,
				configType,
				param,
			)
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
	taskScope.Symbol.Type = taskType
	if block := task.Block(); block != nil {
		if !statement.AnalyzeBlock(taskScope, result, block) {
			return false
		}
	}
	return true
}
