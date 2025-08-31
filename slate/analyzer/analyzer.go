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
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

type Config struct {
	Program  parser.IProgramContext
	Resolver symbol.Resolver
}

func Analyze(cfg Config) result.Result {
	prog := cfg.Program
	rootScope := &symbol.Scope{GlobalResolver: cfg.Resolver}
	res := result.Result{}
	// First pass: collect declarations with empty type signatures
	for _, item := range prog.AllTopLevelItem() {
		if fn := item.FunctionDeclaration(); fn != nil {
			_, err := rootScope.AddSymbol(
				fn.IDENTIFIER().GetText(),
				symbol.KindFunction,
				types.Function{
					Params: make(map[string]symbol.Type),
				},
				fn,
			)
			if err != nil {
				res.AddError(err, fn)
				return res
			}
		} else if task := item.TaskDeclaration(); task != nil {
			_, err := rootScope.AddSymbol(
				task.IDENTIFIER().GetText(),
				symbol.KindTask,
				types.Task{
					Config: make(map[string]symbol.Type),
					Params: make(map[string]symbol.Type),
				},
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
	
	// Get the function type from the symbol
	fnType := fnScope.Symbol.Type.(types.Function)
	
	// Parse parameters and add them to the function's type signature
	if params := fn.ParameterList(); params != nil {
		for _, param := range params.AllParameter() {
			var paramType symbol.Type
			if typeCtx := param.Type_(); typeCtx != nil {
				paramType, _ = types.InferFromTypeContext(typeCtx)
			}
			paramName := param.IDENTIFIER().GetText()
			fnType.Params[paramName] = paramType
			
			// Also add to scope for use within function body
			if _, err := fnScope.AddSymbol(
				paramName,
				symbol.KindParam,
				paramType,
				param,
			); err != nil {
				result.AddError(err, param)
				return false
			}
		}
	}
	
	// Parse return type
	if retType := fn.ReturnType(); retType != nil {
		if typeCtx := retType.Type_(); typeCtx != nil {
			fnType.Return, _ = types.InferFromTypeContext(typeCtx)
		}
	}
	
	// Update the function's type in the symbol
	fnScope.Symbol.Type = fnType
	
	if block := fn.Block(); block != nil {
		visitBlock(fnScope, result, block)
	}
	return true
}

// visitTaskDeclaration analyzes a task declaration
func visitTaskDeclaration(
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
	
	// Get the task type from the symbol
	taskType := taskScope.Symbol.Type.(types.Task)
	
	// Parse config parameters and add them to the task's type signature
	if configBlock := task.ConfigBlock(); configBlock != nil {
		for _, param := range configBlock.AllConfigParameter() {
			paramName := param.IDENTIFIER().GetText()
			// Infer config parameter type
			var configType symbol.Type
			if typeCtx := param.Type_(); typeCtx != nil {
				configType, _ = types.InferFromTypeContext(typeCtx)
			}
			taskType.Config[paramName] = configType
			
			// Also add to scope for use within task body
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
	
	// Parse runtime parameters and add them to the task's type signature
	if params := task.ParameterList(); params != nil {
		for _, param := range params.AllParameter() {
			var paramType symbol.Type
			if typeCtx := param.Type_(); typeCtx != nil {
				paramType, _ = types.InferFromTypeContext(typeCtx)
			}
			paramName := param.IDENTIFIER().GetText()
			taskType.Params[paramName] = paramType
			
			// Also add to scope for use within task body
			if _, err := taskScope.AddSymbol(
				paramName,
				symbol.KindParam,
				paramType,
				param,
			); err != nil {
				result.AddError(err, param)
				return false
			}
		}
	}
	
	// Parse return type
	if retType := task.ReturnType(); retType != nil {
		if typeCtx := retType.Type_(); typeCtx != nil {
			taskType.Return, _ = types.InferFromTypeContext(typeCtx)
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
	parentScope *symbol.Scope,
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
	blockScope *symbol.Scope,
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
	blockScope *symbol.Scope,
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
	parentScope *symbol.Scope,
	result *result.Result,
	declaration antlr.ParserRuleContext,
	expression parser.IExpressionContext,
	typeCtx parser.ITypeContext,
) (symbol.Type, bool) {
	if typeCtx != nil {
		var varType symbol.Type
		var err error
		// Explicit type annotation
		varType, err = types.InferFromTypeContext(typeCtx)
		if err != nil {
			result.AddError(err, declaration)
			return nil, false
		}
		// If there's an initializer, check type compatibility
		if expression != nil {
			// Check if the expression is a literal
			exprType := types.InferFromExpression(parentScope, expression)
			if exprType != nil && varType != nil {
				isLiteral := isLiteralExpression(expression)
				// If it's a literal, we might allow some implicit conversions
				// For now, we still check compatibility the same way
				if (isLiteral && !types.LiteralCompatible(varType, exprType)) && !types.Compatible(varType, exprType) {
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
		return types.InferFromExpression(parentScope, expression), true
	}
	result.AddError(errors.Newf("no type declaration found for %s", declaration), declaration)
	return nil, false
}

// visitLocalVariable analyzes a local variable declaration
func visitLocalVariable(
	blockScope *symbol.Scope,
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
	_, err := blockScope.AddSymbol(name, symbol.KindVariable, varType, localVar)
	if err != nil {
		result.AddError(err, localVar)
		return false
	}

	return true
}

// visitStatefulVariable analyzes a stateful variable declaration
func visitStatefulVariable(
	blockScope *symbol.Scope,
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
	_, err := blockScope.AddSymbol(name, symbol.KindStatefulVariable, varType, statefulVar)
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
	parentScope *symbol.Scope,
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
	*symbol.Scope,
	*result.Result,
	parser.IReturnStatementContext,
) bool {
	return true
}

func visitChannelOperation(
	*symbol.Scope,
	*result.Result,
	parser.IChannelOperationContext,
) bool {
	return true
}

func visitAssignment(
	parentScope *symbol.Scope,
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
			exprType := types.InferFromExpression(parentScope, expr)
			if exprType != nil {
				varType := varScope.Symbol.Type.(symbol.Type)
				if !types.Compatible(varType, exprType) {
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
