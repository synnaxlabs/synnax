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
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
)

func AnalyzeBlock(ctx context.Context[parser.IBlockContext]) bool {
	blockScope, err := ctx.Scope.Add(ctx, ir.Symbol{
		Kind:       ir.KindBlock,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	for _, stmt := range ctx.AST.AllStatement() {
		if !Analyze(context.Child(ctx, stmt).WithScope(blockScope)) {
			return false
		}
	}
	return true
}

// Analyze analyzes a statement
func Analyze(ctx context.Context[parser.IStatementContext]) bool {
	if varDecl := ctx.AST.VariableDeclaration(); varDecl != nil {
		return analyzeVariableDeclaration(context.Child(ctx, varDecl))
	} else if ifStmt := ctx.AST.IfStatement(); ifStmt != nil {
		return analyzeIfStatement(context.Child(ctx, ifStmt))
	} else if returnStmt := ctx.AST.ReturnStatement(); returnStmt != nil {
		return analyzeReturnStatement(context.Child(ctx, returnStmt))
	} else if channelOp := ctx.AST.ChannelOperation(); channelOp != nil {
		return analyzeChannelOperation(context.Child(ctx, channelOp))
	} else if assignment := ctx.AST.Assignment(); assignment != nil {
		return analyzeAssignment(context.Child(ctx, assignment))
	} else if expr := ctx.AST.Expression(); expr != nil {
		return expression.Analyze(context.Child(ctx, expr))
	}
	return true
}

func analyzeVariableDeclaration(ctx context.Context[parser.IVariableDeclarationContext]) bool {
	if local := ctx.AST.LocalVariable(); local != nil {
		return analyzeLocalVariable(context.Child(ctx, local))
	} else if stateful := ctx.AST.StatefulVariable(); stateful != nil {
		return analyzeStatefulVariable(context.Child(ctx, stateful))
	}
	return true
}

func analyzeVariableDeclarationType[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	expression parser.IExpressionContext,
	typeCtx parser.ITypeContext,
) (ir.Type, bool) {
	if typeCtx != nil {
		var varType ir.Type
		var err error
		// Explicit type annotation
		varType, err = atypes.InferFromTypeContext(typeCtx)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return nil, false
		}
		// If there's an initializer, check type compatibility
		if expression != nil {
			// Check if the expression is a literal
			exprType := atypes.InferFromExpression(context.Child(ctx, expression))
			if exprType != nil && varType != nil {
				isLiteral := isLiteralExpression(context.Child(ctx, expression))
				// If it's a literal, we might allow some implicit conversions
				// For now, we still check compatibility the same way
				if (isLiteral && !atypes.LiteralAssignmentCompatible(varType, exprType)) || (!isLiteral && !atypes.Compatible(varType, exprType)) {
					ctx.Diagnostics.AddError(
						errors.Newf("type mismatch: cannot assign %s to %s", exprType, varType),
						ctx.AST,
					)
					return nil, false
				}
			}
		}
		return varType, true
	}
	if expression != nil {
		return atypes.InferFromExpression(context.Child(ctx, expression)), true
	}
	ctx.Diagnostics.AddError(
		errors.Newf("no type declaration found for %s", ctx.AST), ctx.AST,
	)
	return nil, false
}

// isLiteralExpression checks if an expression is a literal value (number, string, bool)
func isLiteralExpression(ctx context.Context[parser.IExpressionContext]) bool {
	if ctx.AST == nil {
		return false
	}

	// Check if the expression is a simple literal
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
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
func analyzeLocalVariable(ctx context.Context[parser.ILocalVariableContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()

	// Check if this is actually a non-blocking channel read
	// (i.e., the expression is a simple identifier that refers to a channel)
	if expr != nil && ctx.AST.Type_() == nil {
		childCtx := context.Child(ctx, expr)
		if isChannelIdentifier(childCtx) {
			// This is a non-blocking channel read: varName := channelName
			chanType := getChannelType(childCtx)
			if chanType != nil {
				// Add variable with the channel's value type
				_, err := childCtx.Scope.Add(ctx, ir.Symbol{
					Name:       name,
					Kind:       ir.KindChannel,
					Type:       chanType.ValueType,
					ParserRule: ctx.AST,
				})
				if err != nil {
					ctx.Diagnostics.AddError(err, ctx.AST)
					return false
				}
				return true
			}
		}
	}

	// Also validate the expression for undefined variables
	if expr != nil {
		if !expression.Analyze(context.Child(ctx, expr)) {
			return false
		}
	}
	varType, ok := analyzeVariableDeclarationType(
		ctx,
		expr,
		ctx.AST.Type_(),
	)
	if !ok {
		return false
	}
	_, err := ctx.Scope.Add(ctx, ir.Symbol{
		Name:       name,
		Type:       varType,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return true
}

// isChannelIdentifier checks if an expression is a simple identifier that refers to a channel
func isChannelIdentifier(ctx context.Context[parser.IExpressionContext]) bool {
	// Navigate through the expression tree to find an identifier
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
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
												if sym, err := ctx.Scope.Resolve(ctx, id.GetText()); err == nil {
													if _, ok := sym.Type.(ir.Chan); ok {
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
func getChannelType(ctx context.Context[parser.IExpressionContext]) *ir.Chan {
	// Navigate through the expression tree to find the identifier and get its type
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
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
												if sym, err := ctx.Scope.Resolve(ctx, id.GetText()); err == nil {
													if chanType, ok := sym.Type.(ir.Chan); ok {
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
func analyzeStatefulVariable(ctx context.Context[parser.IStatefulVariableContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()
	varType, ok := analyzeVariableDeclarationType(
		ctx,
		expr,
		ctx.AST.Type_(),
	)
	if !ok {
		return false
	}
	_, err := ctx.Scope.Add(ctx, ir.Symbol{
		Name:       name,
		Kind:       ir.KindStatefulVariable,
		Type:       varType,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	if expr != nil {
		return expression.Analyze(context.Child(ctx, expr))
	}
	return true
}

func analyzeIfStatement(ctx context.Context[parser.IIfStatementContext]) bool {
	// First analyze the condition expression
	if expr := ctx.AST.Expression(); expr != nil {
		if !expression.Analyze(context.Child(ctx, expr)) {
			return false
		}
	}

	// Analyze the main if block
	if block := ctx.AST.Block(); block != nil {
		if !AnalyzeBlock(context.Child(ctx, block)) {
			return false
		}
	}

	// Analyze all else-if clauses
	for _, elseIfClause := range ctx.AST.AllElseIfClause() {
		// Analyze the else-if condition
		if expr := elseIfClause.Expression(); expr != nil {
			if !expression.Analyze(context.Child(ctx, expr)) {
				return false
			}
		}
		// Analyze the else-if block
		if block := elseIfClause.Block(); block != nil {
			if !AnalyzeBlock(context.Child(ctx, block)) {
				return false
			}
		}
	}

	// Analyze the else clause if present
	if elseClause := ctx.AST.ElseClause(); elseClause != nil {
		if block := elseClause.Block(); block != nil {
			if !AnalyzeBlock(context.Child(ctx, block)) {
				return false
			}
		}
	}

	return true
}

func analyzeReturnStatement(ctx context.Context[parser.IReturnStatementContext]) bool {
	enclosingScope, err := ctx.Scope.ClosestAncestorOfKind(ir.KindFunction)
	if err != nil {
		enclosingScope, err = ctx.Scope.ClosestAncestorOfKind(ir.KindStage)
		if err != nil {
			ctx.Diagnostics.AddError(
				errors.New("return statement not in function or stage"),
				ctx.AST,
			)
			return false
		}
	}
	var expectedReturnType ir.Type
	switch enclosingScope.Kind {
	case ir.KindFunction:
		fnType := enclosingScope.Type.(ir.Function)
		expectedReturnType = fnType.Return
	case ir.KindStage:
		stageType := enclosingScope.Type.(ir.Stage)
		expectedReturnType = stageType.Return
	}
	returnExpr := ctx.AST.Expression()
	if returnExpr != nil {
		if !expression.Analyze(context.Child(ctx, returnExpr)) {
			return false
		}

		actualReturnType := atypes.InferFromExpression(context.Child(ctx, returnExpr).WithTypeHint(expectedReturnType))
		if expectedReturnType == nil {
			ctx.Diagnostics.AddError(
				errors.New("unexpected return value in function/stage with void return type"),
				ctx.AST,
			)
			return false
		}
		if actualReturnType != nil {
			if !atypes.Compatible(expectedReturnType, actualReturnType) {
				ctx.Diagnostics.AddError(
					errors.Newf(
						"type mismatch: cannot return %s, expected %s",
						actualReturnType,
						expectedReturnType,
					),
					ctx.AST,
				)
				return false
			}
		}
	} else {
		if expectedReturnType != nil {
			ctx.Diagnostics.AddError(
				errors.Newf(
					"return statement missing value of type %s",
					expectedReturnType,
				),
				ctx.AST,
			)
			return false
		}
	}
	return true
}

func analyzeChannelOperation(ctx context.Context[parser.IChannelOperationContext]) bool {
	if write := ctx.AST.ChannelWrite(); write != nil {
		return analyzeChannelWrite(context.Child(ctx, write))
	} else if read := ctx.AST.ChannelRead(); read != nil {
		return analyzeChannelRead(context.Child(ctx, read))
	}
	return true
}

func analyzeChannelWrite(ctx context.Context[parser.IChannelWriteContext]) bool {
	// Get the channel name
	var channelName string
	if ctx.AST.IDENTIFIER() != nil {
		channelName = ctx.AST.IDENTIFIER().GetText()
	} else {
		return false
	}

	// Resolve the channel
	channelSym, err := ctx.Scope.Resolve(ctx, channelName)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	// Check it's a channel type
	chanType, ok := channelSym.Type.(ir.Chan)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Newf("%s is not a channel", channelName),
			ctx.AST,
		)
		return false
	}

	// Analyze the expression being written
	expr := ctx.AST.Expression()
	if expr == nil {
		return true
	}

	if !expression.Analyze(context.Child(ctx, expr)) {
		return false
	}

	// Check type compatibility
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if exprType != nil && chanType.ValueType != nil {
		if !atypes.Compatible(chanType.ValueType, exprType) {
			ctx.Diagnostics.AddError(
				errors.Newf("type mismatch: cannot write %s to channel of type %s",
					exprType, chanType.ValueType),
				ctx.AST,
			)
			return false
		}
	}

	return true
}

func analyzeChannelRead(ctx context.Context[parser.IChannelReadContext]) bool {
	if blocking := ctx.AST.BlockingRead(); blocking != nil {
		return analyzeBlockingRead(context.Child(ctx, blocking))
	} else if nonBlocking := ctx.AST.NonBlockingRead(); nonBlocking != nil {
		return analyzeNonBlockingRead(context.Child(ctx, nonBlocking))
	}
	return true
}

func analyzeBlockingRead(ctx context.Context[parser.IBlockingReadContext]) bool {
	// Format: varName := <-channelName
	ids := ctx.AST.AllIDENTIFIER()
	if len(ids) != 2 {
		return false
	}

	varName := ids[0].GetText()
	channelName := ids[1].GetText()

	// Resolve the channel
	channelSym, err := ctx.Scope.Resolve(ctx, channelName)
	if err != nil {
		ctx.Diagnostics.AddError(
			errors.Wrapf(err, "undefined channel: %s", channelName),
			ctx.AST,
		)
		return false
	}

	// Check it's a channel type
	chanType, ok := channelSym.Type.(ir.Chan)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Newf("%s is not a channel", channelName),
			ctx.AST,
		)
		return false
	}

	// Add the variable with the channel's value type
	_, err = ctx.Scope.Add(ctx, ir.Symbol{
		Name:       varName,
		Kind:       ir.KindVariable,
		Type:       chanType.ValueType,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	return true
}

func analyzeNonBlockingRead(ctx context.Context[parser.INonBlockingReadContext]) bool {
	// Format: varName := channelName
	ids := ctx.AST.AllIDENTIFIER()
	if len(ids) != 2 {
		return false
	}

	varName := ids[0].GetText()
	channelName := ids[1].GetText()

	// Resolve the channel
	channelSym, err := ctx.Scope.Resolve(ctx, channelName)
	if err != nil {
		ctx.Diagnostics.AddError(
			errors.Wrapf(err, "undefined channel: %s", channelName),
			ctx.AST,
		)
		return false
	}

	// Check it's a channel type
	chanType, ok := channelSym.Type.(ir.Chan)
	if !ok {
		ctx.Diagnostics.AddError(
			errors.Newf("%s is not a channel", channelName),
			ctx.AST,
		)
		return false
	}

	// Add the variable with the channel's value type
	_, err = ctx.Scope.Add(ctx, ir.Symbol{
		Name:       varName,
		Kind:       ir.KindVariable,
		Type:       chanType.ValueType,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	return true
}

func analyzeAssignment(ctx context.Context[parser.IAssignmentContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	varScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	expr := ctx.AST.Expression()
	if expr == nil {
		return true
	}
	if !expression.Analyze(context.Child(ctx, expr)) {
		return false
	}
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if exprType == nil {
		return true
	}
	if varScope.Type == nil {
		return true
	}
	varType := varScope.Type
	if atypes.Compatible(varType, exprType) {
		return true
	}
	ctx.Diagnostics.AddError(
		errors.Newf("type mismatch: cannot assign %s to variable of type %s", exprType, varType),
		ctx.AST,
	)
	return false
}
