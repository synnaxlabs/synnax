// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/slate/analyzer/expression"
	"github.com/synnaxlabs/slate/analyzer/result"
	atypes "github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

func AnalyzeBlock(
	parentScope *symbol.Scope,
	result *result.Result,
	block parser.IBlockContext,
) bool {
	blockScope := parentScope.AddBlock(block)
	for _, stmt := range block.AllStatement() {
		if !Analyze(blockScope, result, stmt) {
			return false
		}
	}
	return true
}

// Analyze analyzes a statement
func Analyze(
	blockScope *symbol.Scope,
	result *result.Result,
	ctx parser.IStatementContext,
) bool {
	if varDecl := ctx.VariableDeclaration(); varDecl != nil {
		return analyzeVariableDeclaration(blockScope, result, varDecl)
	} else if ifStmt := ctx.IfStatement(); ifStmt != nil {
		return analyzeIfStatement(blockScope, result, ifStmt)
	} else if returnStmt := ctx.ReturnStatement(); returnStmt != nil {
		return analyzeReturnStatement(blockScope, result, returnStmt)
	} else if channelOp := ctx.ChannelOperation(); channelOp != nil {
		return analyzeChannelOperation(blockScope, result, channelOp)
	} else if assignment := ctx.Assignment(); assignment != nil {
		return analyzeAssignment(blockScope, result, assignment)
	} else if expr := ctx.Expression(); expr != nil {
		return expression.Analyze(blockScope, result, expr)
	}
	return true
}

func analyzeVariableDeclaration(
	blockScope *symbol.Scope,
	result *result.Result,
	ctx parser.IVariableDeclarationContext,
) bool {
	if local := ctx.LocalVariable(); local != nil {
		return analyzeLocalVariable(blockScope, result, local)
	} else if stateful := ctx.StatefulVariable(); stateful != nil {
		return analyzeStatefulVariable(blockScope, result, stateful)
	}
	return true
}

func analyzeVariableDeclarationType(
	parentScope *symbol.Scope,
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
			exprType := atypes.InferFromExpression(parentScope, expression, nil)
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
		return atypes.InferFromExpression(parentScope, expression, nil), true
	}
	result.AddError(errors.Newf("no type declaration found for %s", declaration), declaration)
	return nil, false
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

// analyzeLocalVariable analyzes a local variable declaration
func analyzeLocalVariable(
	blockScope *symbol.Scope,
	result *result.Result,
	localVar parser.ILocalVariableContext,
) bool {
	name := localVar.IDENTIFIER().GetText()
	expr := localVar.Expression()
	// Also validate the expression for undefined variables
	if expr != nil {
		if !expression.Analyze(blockScope, result, expr) {
			return false
		}
	}
	varType, ok := analyzeVariableDeclarationType(
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

// analyzeStatefulVariable analyzes a stateful variable declaration
func analyzeStatefulVariable(
	blockScope *symbol.Scope,
	result *result.Result,
	statefulVar parser.IStatefulVariableContext,
) bool {
	name := statefulVar.IDENTIFIER().GetText()
	expr := statefulVar.Expression()
	varType, ok := analyzeVariableDeclarationType(
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
		return expression.Analyze(blockScope, result, expr)
	}
	return true
}

func analyzeIfStatement(
	parentScope *symbol.Scope,
	result *result.Result,
	ctx parser.IIfStatementContext,
) bool {
	// First analyze the condition expression
	if expr := ctx.Expression(); expr != nil {
		if !expression.Analyze(parentScope, result, expr) {
			return false
		}
	}

	// Analyze the main if block
	if block := ctx.Block(); block != nil {
		if !AnalyzeBlock(parentScope, result, block) {
			return false
		}
	}

	// Analyze all else-if clauses
	for _, elseIfClause := range ctx.AllElseIfClause() {
		// Analyze the else-if condition
		if expr := elseIfClause.Expression(); expr != nil {
			if !expression.Analyze(parentScope, result, expr) {
				return false
			}
		}
		// Analyze the else-if block
		if block := elseIfClause.Block(); block != nil {
			if !AnalyzeBlock(parentScope, result, block) {
				return false
			}
		}
	}

	// Analyze the else clause if present
	if elseClause := ctx.ElseClause(); elseClause != nil {
		if block := elseClause.Block(); block != nil {
			if !AnalyzeBlock(parentScope, result, block) {
				return false
			}
		}
	}

	return true
}

func analyzeReturnStatement(
	*symbol.Scope,
	*result.Result,
	parser.IReturnStatementContext,
) bool {
	return true
}

func analyzeChannelOperation(
	*symbol.Scope,
	*result.Result,
	parser.IChannelOperationContext,
) bool {
	return true
}

func analyzeAssignment(
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
		if !expression.Analyze(parentScope, result, expr) {
			return false
		}
		if varScope.Symbol != nil && varScope.Symbol.Type != nil {
			exprType := atypes.InferFromExpression(parentScope, expr, nil)
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
