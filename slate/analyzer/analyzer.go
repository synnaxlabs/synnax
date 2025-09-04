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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/slate/analyzer/expression"
	"github.com/synnaxlabs/slate/analyzer/flow"
	"github.com/synnaxlabs/slate/analyzer/result"
	atypes "github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	symbol2 "github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

type Config struct {
	Program  parser.IProgramContext
	Resolver symbol2.Resolver
}

type Result = result.Result

func Analyze(cfg Config) result.Result {
	prog := cfg.Program
	rootScope := &symbol2.Scope{GlobalResolver: cfg.Resolver}
	res := result.Result{Symbols: rootScope}
	// First pass: collect declarations with empty type signatures
	for _, item := range prog.AllTopLevelItem() {
		if fn := item.FunctionDeclaration(); fn != nil {
			name := fn.IDENTIFIER().GetText()
			_, err := rootScope.AddSymbol(name, symbol2.KindFunction, types.NewFunction(), fn)
			if err != nil {
				res.AddError(err, fn)
				return res
			}
		} else if task := item.TaskDeclaration(); task != nil {
			name := task.IDENTIFIER().GetText()
			_, err := rootScope.AddSymbol(
				name,
				symbol2.KindTask,
				types.NewTask(),
				task,
			)
			if err != nil {
				res.AddError(err, task)
				return res
			}
		}
	}

	for _, item := range prog.AllTopLevelItem() {
		if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			if !visitFunctionDeclaration(rootScope, &res, funcDecl) {
				return res
			}
		} else if taskDecl := item.TaskDeclaration(); taskDecl != nil {
			if !visitTaskDeclaration(rootScope, &res, taskDecl) {
				return res
			}
		} else if flowStmt := item.FlowStatement(); flowStmt != nil {
			if !flow.Visit(rootScope, &res, flowStmt) {
				return res
			}
		}
	}
	return res
}

// visitFunctionDeclaration analyzes a function declaration
func visitFunctionDeclaration(
	parentScope *symbol2.Scope,
	result *result.Result,
	fn parser.IFunctionDeclarationContext,
) bool {
	name := fn.IDENTIFIER().GetText()
	fnScope, err := parentScope.Get(name)
	if err != nil {
		result.AddError(err, fn)
		return false
	}

	// Get the function type from the symbol
	fnType := fnScope.Symbol.Type.(types.Function)
	if !visitParams(
		fnScope,
		result,
		fn.ParameterList(),
		&fnType.Params,
	) {
		return false
	}

	// Parse return type
	if retType := fn.ReturnType(); retType != nil {
		if typeCtx := retType.Type_(); typeCtx != nil {
			fnType.Return, _ = atypes.InferFromTypeContext(typeCtx)
		}
	}

	// Update the function's type in the symbol
	fnScope.Symbol.Type = fnType

	if block := fn.Block(); block != nil {
		visitBlock(fnScope, result, block)
	}
	return true
}

func visitParams(
	scope *symbol2.Scope,
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
			symbol2.KindParam,
			paramType,
			param,
		); err != nil {
			result.AddError(err, param)
			return false
		}
	}
	return true
}

// visitTaskDeclaration analyzes a task declaration
func visitTaskDeclaration(
	parentScope *symbol2.Scope,
	result *result.Result,
	task parser.ITaskDeclarationContext,
) bool {
	name := task.IDENTIFIER().GetText()
	taskScope, err := parentScope.Get(name)
	if err != nil {
		result.AddError(err, task)
		return false
	}

	// Get the task type from the symbol
	taskType := taskScope.Symbol.Type.(types.Task)

	// Parse config parameters and add them to the task's type signature
	if configBlock := task.ConfigBlock(); configBlock != nil {
		for _, param := range configBlock.AllConfigParameter() {
			paramName := param.IDENTIFIER().GetText()
			// Infer config parameter type
			var configType types.Type
			if typeCtx := param.Type_(); typeCtx != nil {
				configType, _ = atypes.InferFromTypeContext(typeCtx)
			}
			if !taskType.Config.Put(paramName, configType) {
				result.AddError(errors.Newf("duplicate configuration parameter %s", param), task)
			}
			// Also add to scope for use within task body
			_, err := taskScope.AddSymbol(
				paramName,
				symbol2.KindConfigParam,
				configType,
				param,
			)
			if err != nil {
				result.AddError(err, param)
				return false
			}
		}
	}

	if !visitParams(
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

	// Update the task's type in the symbol
	taskScope.Symbol.Type = taskType

	if block := task.Block(); block != nil {
		if !visitBlock(taskScope, result, block) {
			return false
		}
	}
	return true
}

// visitBlock analyzes a block of statements
func visitBlock(
	parentScope *symbol2.Scope,
	result *result.Result,
	block parser.IBlockContext,
) bool {
	blockScope := parentScope.AddBlock()
	for _, stmt := range block.AllStatement() {
		if !visitStatement(blockScope, result, stmt) {
			return false
		}
	}
	return true
}

// visitStatement analyzes a statement
func visitStatement(
	blockScope *symbol2.Scope,
	result *result.Result,
	ctx parser.IStatementContext,
) bool {
	if varDecl := ctx.VariableDeclaration(); varDecl != nil {
		return visitVariableDeclaration(blockScope, result, varDecl)
	} else if ifStmt := ctx.IfStatement(); ifStmt != nil {
		return visitIfStatement(blockScope, result, ifStmt)
	} else if returnStmt := ctx.ReturnStatement(); returnStmt != nil {
		return visitReturnStatement(blockScope, result, returnStmt)
	} else if channelOp := ctx.ChannelOperation(); channelOp != nil {
		return visitChannelOperation(blockScope, result, channelOp)
	} else if assignment := ctx.Assignment(); assignment != nil {
		return visitAssignment(blockScope, result, assignment)
	} else if expr := ctx.Expression(); expr != nil {
		return expression.Visit(blockScope, result, expr)
	}
	return true
}

func visitVariableDeclaration(
	blockScope *symbol2.Scope,
	result *result.Result,
	ctx parser.IVariableDeclarationContext,
) bool {
	if local := ctx.LocalVariable(); local != nil {
		return visitLocalVariable(blockScope, result, local)
	} else if stateful := ctx.StatefulVariable(); stateful != nil {
		return visitStatefulVariable(blockScope, result, stateful)
	}
	return true
}

// isLiteralExpression checks if an expression is a literal value (number, string, bool)
func isLiteralExpression(expr parser.IExpressionContext) bool {
	if expr == nil {
		return false
	}

	// Check if the expression is a simple literal
	if logicalOr := expr.LogicalOrExpression(); logicalOr != nil {
		// Navigate down through the expression hierarchy to find literals
		if ands := logicalOr.AllLogicalAndExpression(); len(ands) == 1 {
			if equalities := ands[0].AllEqualityExpression(); len(equalities) == 1 {
				if relationals := equalities[0].AllRelationalExpression(); len(relationals) == 1 {
					if additives := relationals[0].AllAdditiveExpression(); len(additives) == 1 {
						if multiplicatives := additives[0].AllMultiplicativeExpression(); len(multiplicatives) == 1 {
							if powers := multiplicatives[0].AllPowerExpression(); len(powers) == 1 {
								if unary := powers[0].UnaryExpression(); unary != nil {
									if postfix := unary.PostfixExpression(); postfix != nil {
										if primary := postfix.PrimaryExpression(); primary != nil {
											// Check if it's a literal
											return primary.Literal() != nil
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return false
}

func visitVariableDeclarationType(
	parentScope *symbol2.Scope,
	result *result.Result,
	declaration antlr.ParserRuleContext,
	expression parser.IExpressionContext,
	typeCtx parser.ITypeContext,
) (types.Type, bool) {
	if typeCtx != nil {
		var varType types.Type
		var err error
		// Explicit type annotation
		varType, err = atypes.InferFromTypeContext(typeCtx)
		if err != nil {
			result.AddError(err, declaration)
			return nil, false
		}
		// If there's an initializer, check type compatibility
		if expression != nil {
			// Check if the expression is a literal
			exprType := atypes.InferFromExpression(parentScope, expression)
			if exprType != nil && varType != nil {
				isLiteral := isLiteralExpression(expression)
				// If it's a literal, we might allow some implicit conversions
				// For now, we still check compatibility the same way
				if (isLiteral && !atypes.LiteralAssignmentCompatible(varType, exprType)) && !atypes.Compatible(varType, exprType) {
					result.AddError(
						errors.Newf("type mismatch: cannot assign %s to %s", exprType, varType),
						declaration,
					)
					return nil, false
				}
			}
		}
		return varType, true
	}
	if expression != nil {
		return atypes.InferFromExpression(parentScope, expression), true
	}
	result.AddError(errors.Newf("no type declaration found for %s", declaration), declaration)
	return nil, false
}

// visitLocalVariable analyzes a local variable declaration
func visitLocalVariable(
	blockScope *symbol2.Scope,
	result *result.Result,
	localVar parser.ILocalVariableContext,
) bool {
	name := localVar.IDENTIFIER().GetText()
	expr := localVar.Expression()
	// Also validate the expression for undefined variables
	if expr != nil {
		if !expression.Visit(blockScope, result, expr) {
			return false
		}
	}
	varType, ok := visitVariableDeclarationType(
		blockScope,
		result,
		localVar,
		expr,
		localVar.Type_(),
	)
	if !ok {
		return false
	}
	_, err := blockScope.AddSymbol(name, symbol2.KindVariable, varType, localVar)
	if err != nil {
		result.AddError(err, localVar)
		return false
	}

	return true
}

// visitStatefulVariable analyzes a stateful variable declaration
func visitStatefulVariable(
	blockScope *symbol2.Scope,
	result *result.Result,
	statefulVar parser.IStatefulVariableContext,
) bool {
	name := statefulVar.IDENTIFIER().GetText()
	expr := statefulVar.Expression()
	varType, ok := visitVariableDeclarationType(
		blockScope,
		result,
		statefulVar,
		expr,
		statefulVar.Type_(),
	)
	if !ok {
		return false
	}
	_, err := blockScope.AddSymbol(name, symbol2.KindStatefulVariable, varType, statefulVar)
	if err != nil {
		result.AddError(err, statefulVar)
		return false
	}
	if expr != nil {
		return expression.Visit(blockScope, result, expr)
	}
	return true
}

func visitIfStatement(
	parentScope *symbol2.Scope,
	result *result.Result,
	ctx parser.IIfStatementContext,
) bool {
	if block := ctx.Block(); block != nil {
		if !visitBlock(parentScope, result, block) {
			return false
		}
	}
	return true
}

func visitReturnStatement(
	*symbol2.Scope,
	*result.Result,
	parser.IReturnStatementContext,
) bool {
	return true
}

func visitChannelOperation(
	*symbol2.Scope,
	*result.Result,
	parser.IChannelOperationContext,
) bool {
	return true
}

func visitAssignment(
	parentScope *symbol2.Scope,
	result *result.Result,
	assignment parser.IAssignmentContext,
) bool {
	name := assignment.IDENTIFIER().GetText()
	varScope, err := parentScope.Get(name)
	if err != nil {
		result.AddError(err, assignment)
		return false
	}
	if expr := assignment.Expression(); expr != nil {
		if !expression.Visit(parentScope, result, expr) {
			return false
		}
		if varScope.Symbol != nil && varScope.Symbol.Type != nil {
			exprType := atypes.InferFromExpression(parentScope, expr)
			if exprType != nil {
				varType := varScope.Symbol.Type.(types.Type)
				if !atypes.Compatible(varType, exprType) {
					result.AddError(
						errors.Newf("type mismatch: cannot assign %s to variable of type %s", exprType, varType),
						assignment,
					)
					return false
				}
			}
		}
	}
	return true
}
