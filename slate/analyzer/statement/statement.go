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
	blockScope, err := parentScope.Add(symbol.Symbol{
		Kind:       symbol.KindBlock,
		ParserRule: block,
	})
	if err != nil {
		result.AddError(err, block)
		return false
	}
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
				if (isLiteral && !atypes.LiteralAssignmentCompatible(varType, exprType)) || (!isLiteral && !atypes.Compatible(varType, exprType)) {
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

	// Check if this is actually a non-blocking channel read
	// (i.e., the expression is a simple identifier that refers to a channel)
	if expr != nil && localVar.Type_() == nil {
		if isChannelIdentifier(blockScope, expr) {
			// This is a non-blocking channel read: varName := channelName
			chanType := getChannelType(blockScope, expr)
			if chanType != nil {
				// Add variable with the channel's value type
				_, err := blockScope.Add(symbol.Symbol{
					Name:       name,
					Kind:       symbol.KindChannel,
					Type:       chanType.ValueType,
					ParserRule: localVar,
				})
				if err != nil {
					result.AddError(err, localVar)
					return false
				}
				return true
			}
		}
	}

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
	_, err := blockScope.Add(symbol.Symbol{
		Name:       name,
		Type:       varType,
		ParserRule: localVar,
	})
	if err != nil {
		result.AddError(err, localVar)
		return false
	}
	return true
}

// isChannelIdentifier checks if an expression is a simple identifier that refers to a channel
func isChannelIdentifier(scope *symbol.Scope, expr parser.IExpressionContext) bool {
	// Navigate through the expression tree to find an identifier
	if logicalOr := expr.LogicalOrExpression(); logicalOr != nil {
		if ands := logicalOr.AllLogicalAndExpression(); len(ands) == 1 {
			if equalities := ands[0].AllEqualityExpression(); len(equalities) == 1 {
				if relationals := equalities[0].AllRelationalExpression(); len(relationals) == 1 {
					if additives := relationals[0].AllAdditiveExpression(); len(additives) == 1 {
						if multiplicatives := additives[0].AllMultiplicativeExpression(); len(multiplicatives) == 1 {
							if powers := multiplicatives[0].AllPowerExpression(); len(powers) == 1 {
								if unary := powers[0].UnaryExpression(); unary != nil {
									if postfix := unary.PostfixExpression(); postfix != nil {
										if primary := postfix.PrimaryExpression(); primary != nil {
											if id := primary.IDENTIFIER(); id != nil {
												// Check if this identifier refers to a channel
												if sym, err := scope.Resolve(id.GetText()); err == nil {
													if _, ok := sym.Type.(types.Chan); ok {
														return true
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
			}
		}
	}
	return false
}

// getChannelType retrieves the channel type from an expression that is a channel identifier
func getChannelType(scope *symbol.Scope, expr parser.IExpressionContext) *types.Chan {
	// Navigate through the expression tree to find the identifier and get its type
	if logicalOr := expr.LogicalOrExpression(); logicalOr != nil {
		if ands := logicalOr.AllLogicalAndExpression(); len(ands) == 1 {
			if equalities := ands[0].AllEqualityExpression(); len(equalities) == 1 {
				if relationals := equalities[0].AllRelationalExpression(); len(relationals) == 1 {
					if additives := relationals[0].AllAdditiveExpression(); len(additives) == 1 {
						if multiplicatives := additives[0].AllMultiplicativeExpression(); len(multiplicatives) == 1 {
							if powers := multiplicatives[0].AllPowerExpression(); len(powers) == 1 {
								if unary := powers[0].UnaryExpression(); unary != nil {
									if postfix := unary.PostfixExpression(); postfix != nil {
										if primary := postfix.PrimaryExpression(); primary != nil {
											if id := primary.IDENTIFIER(); id != nil {
												if sym, err := scope.Resolve(id.GetText()); err == nil {
													if chanType, ok := sym.Type.(types.Chan); ok {
														return &chanType
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
			}
		}
	}
	return nil
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
	_, err := blockScope.Add(symbol.Symbol{
		Name:       name,
		Kind:       symbol.KindStatefulVariable,
		Type:       varType,
		ParserRule: statefulVar,
	})
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
	scope *symbol.Scope,
	result *result.Result,
	returnStmt parser.IReturnStatementContext,
) bool {
	enclosingScope, err := scope.ClosestAncestorOfKind(symbol.KindFunction)
	if err != nil {
		enclosingScope, err = scope.ClosestAncestorOfKind(symbol.KindTask)
		if err != nil {
			result.AddError(
				errors.New("return statement not in function or task"),
				returnStmt,
			)
			return false
		}
	}
	var expectedReturnType types.Type
	if enclosingScope.Kind == symbol.KindFunction {
		fnType := enclosingScope.Type.(types.Function)
		expectedReturnType = fnType.Return
	} else if enclosingScope.Kind == symbol.KindTask {
		taskType := enclosingScope.Type.(types.Task)
		expectedReturnType = taskType.Return
	}
	returnExpr := returnStmt.Expression()
	if returnExpr != nil {
		if !expression.Analyze(scope, result, returnExpr) {
			return false
		}

		actualReturnType := atypes.InferFromExpression(scope, returnExpr, expectedReturnType)
		if expectedReturnType == nil {
			result.AddError(
				errors.New("unexpected return value in function/task with void return type"),
				returnStmt,
			)
			return false
		}
		if actualReturnType != nil {
			if !atypes.Compatible(expectedReturnType, actualReturnType) {
				result.AddError(
					errors.Newf(
						"type mismatch: cannot return %s, expected %s",
						actualReturnType,
						expectedReturnType,
					),
					returnStmt,
				)
				return false
			}
		}
	} else {
		if expectedReturnType != nil {
			result.AddError(
				errors.Newf(
					"return statement missing value of type %s",
					expectedReturnType,
				),
				returnStmt,
			)
			return false
		}
	}
	return true
}

func analyzeChannelOperation(
	scope *symbol.Scope,
	res *result.Result,
	ctx parser.IChannelOperationContext,
) bool {
	if write := ctx.ChannelWrite(); write != nil {
		return analyzeChannelWrite(scope, res, write)
	} else if read := ctx.ChannelRead(); read != nil {
		return analyzeChannelRead(scope, res, read)
	}
	return true
}

func analyzeChannelWrite(
	scope *symbol.Scope,
	res *result.Result,
	ctx parser.IChannelWriteContext,
) bool {
	// Get the channel name
	var channelName string
	if ctx.IDENTIFIER() != nil {
		channelName = ctx.IDENTIFIER().GetText()
	} else {
		return false
	}

	// Resolve the channel
	channelSym, err := scope.Resolve(channelName)
	if err != nil {
		res.AddError(err, ctx)
		return false
	}
	if taskScope, err := scope.ClosestAncestorOfKind(symbol.KindTask); err == nil {
		t := taskScope.Type.(types.Task)
		t.Channels.Write.Add(channelSym.Name)
	}

	// Check it's a channel type
	chanType, ok := channelSym.Type.(types.Chan)
	if !ok {
		res.AddError(errors.Newf("%s is not a channel", channelName), ctx)
		return false
	}

	// Analyze the expression being written
	expr := ctx.Expression()
	if expr == nil {
		return true
	}

	if !expression.Analyze(scope, res, expr) {
		return false
	}

	// Check type compatibility
	exprType := atypes.InferFromExpression(scope, expr, nil)
	if exprType != nil && chanType.ValueType != nil {
		if !atypes.Compatible(chanType.ValueType, exprType) {
			res.AddError(
				errors.Newf("type mismatch: cannot write %s to channel of type %s",
					exprType, chanType.ValueType),
				ctx,
			)
			return false
		}
	}

	return true
}

func analyzeChannelRead(
	scope *symbol.Scope,
	res *result.Result,
	ctx parser.IChannelReadContext,
) bool {
	if blocking := ctx.BlockingRead(); blocking != nil {
		return analyzeBlockingRead(scope, res, blocking)
	} else if nonBlocking := ctx.NonBlockingRead(); nonBlocking != nil {
		return analyzeNonBlockingRead(scope, res, nonBlocking)
	}
	return true
}

func analyzeBlockingRead(
	scope *symbol.Scope,
	res *result.Result,
	ctx parser.IBlockingReadContext,
) bool {
	// Format: varName := <-channelName
	ids := ctx.AllIDENTIFIER()
	if len(ids) != 2 {
		return false
	}

	varName := ids[0].GetText()
	channelName := ids[1].GetText()

	// Resolve the channel
	channelSym, err := scope.Resolve(channelName)
	if err != nil {
		res.AddError(errors.Wrapf(err, "undefined channel: %s", channelName), ctx)
		return false
	}

	// Check it's a channel type
	chanType, ok := channelSym.Type.(types.Chan)
	if !ok {
		res.AddError(errors.Newf("%s is not a channel", channelName), ctx)
		return false
	}

	// Add the variable with the channel's value type
	_, err = scope.Add(symbol.Symbol{
		Name:       varName,
		Kind:       symbol.KindVariable,
		Type:       chanType.ValueType,
		ParserRule: ctx,
	})
	if err != nil {
		res.AddError(err, ctx)
		return false
	}

	return true
}

func analyzeNonBlockingRead(
	scope *symbol.Scope,
	res *result.Result,
	ctx parser.INonBlockingReadContext,
) bool {
	// Format: varName := channelName
	ids := ctx.AllIDENTIFIER()
	if len(ids) != 2 {
		return false
	}

	varName := ids[0].GetText()
	channelName := ids[1].GetText()

	// Resolve the channel
	channelSym, err := scope.Resolve(channelName)
	if err != nil {
		res.AddError(errors.Wrapf(err, "undefined channel: %s", channelName), ctx)
		return false
	}

	// Check it's a channel type
	chanType, ok := channelSym.Type.(types.Chan)
	if !ok {
		res.AddError(errors.Newf("%s is not a channel", channelName), ctx)
		return false
	}

	// Add the variable with the channel's value type
	_, err = scope.Add(symbol.Symbol{
		Name:       varName,
		Kind:       symbol.KindVariable,
		Type:       chanType.ValueType,
		ParserRule: ctx,
	})
	if err != nil {
		res.AddError(err, ctx)
		return false
	}

	return true
}

func analyzeAssignment(
	parentScope *symbol.Scope,
	result *result.Result,
	assignment parser.IAssignmentContext,
) bool {
	name := assignment.IDENTIFIER().GetText()
	varScope, err := parentScope.Resolve(name)
	if err != nil {
		result.AddError(err, assignment)
		return false
	}
	expr := assignment.Expression()
	if expr == nil {
		return true
	}
	if !expression.Analyze(parentScope, result, expr) {
		return false
	}
	exprType := atypes.InferFromExpression(parentScope, expr, nil)
	if exprType == nil {
		return true
	}
	if varScope.Type == nil {
		return true
	}
	varType := varScope.Type.(types.Type)
	if atypes.Compatible(varType, exprType) {
		return true
	}
	result.AddError(
		errors.Newf("type mismatch: cannot assign %s to variable of type %s", exprType, varType),
		assignment,
	)
	return false
}
