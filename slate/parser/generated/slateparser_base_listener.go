// Code generated from SlateParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // SlateParser
import "github.com/antlr4-go/antlr/v4"

// BaseSlateParserListener is a complete listener for a parse tree produced by SlateParser.
type BaseSlateParserListener struct{}

var _ SlateParserListener = &BaseSlateParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseSlateParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseSlateParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseSlateParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseSlateParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterProgram is called when production program is entered.
func (s *BaseSlateParserListener) EnterProgram(ctx *ProgramContext) {}

// ExitProgram is called when production program is exited.
func (s *BaseSlateParserListener) ExitProgram(ctx *ProgramContext) {}

// EnterTopLevelStatement is called when production topLevelStatement is entered.
func (s *BaseSlateParserListener) EnterTopLevelStatement(ctx *TopLevelStatementContext) {}

// ExitTopLevelStatement is called when production topLevelStatement is exited.
func (s *BaseSlateParserListener) ExitTopLevelStatement(ctx *TopLevelStatementContext) {}

// EnterReactiveBinding is called when production reactiveBinding is entered.
func (s *BaseSlateParserListener) EnterReactiveBinding(ctx *ReactiveBindingContext) {}

// ExitReactiveBinding is called when production reactiveBinding is exited.
func (s *BaseSlateParserListener) ExitReactiveBinding(ctx *ReactiveBindingContext) {}

// EnterIntervalBinding is called when production intervalBinding is entered.
func (s *BaseSlateParserListener) EnterIntervalBinding(ctx *IntervalBindingContext) {}

// ExitIntervalBinding is called when production intervalBinding is exited.
func (s *BaseSlateParserListener) ExitIntervalBinding(ctx *IntervalBindingContext) {}

// EnterChannelList is called when production channelList is entered.
func (s *BaseSlateParserListener) EnterChannelList(ctx *ChannelListContext) {}

// ExitChannelList is called when production channelList is exited.
func (s *BaseSlateParserListener) ExitChannelList(ctx *ChannelListContext) {}

// EnterFunctionDecl is called when production functionDecl is entered.
func (s *BaseSlateParserListener) EnterFunctionDecl(ctx *FunctionDeclContext) {}

// ExitFunctionDecl is called when production functionDecl is exited.
func (s *BaseSlateParserListener) ExitFunctionDecl(ctx *FunctionDeclContext) {}

// EnterParameterList is called when production parameterList is entered.
func (s *BaseSlateParserListener) EnterParameterList(ctx *ParameterListContext) {}

// ExitParameterList is called when production parameterList is exited.
func (s *BaseSlateParserListener) ExitParameterList(ctx *ParameterListContext) {}

// EnterParameter is called when production parameter is entered.
func (s *BaseSlateParserListener) EnterParameter(ctx *ParameterContext) {}

// ExitParameter is called when production parameter is exited.
func (s *BaseSlateParserListener) ExitParameter(ctx *ParameterContext) {}

// EnterType is called when production type is entered.
func (s *BaseSlateParserListener) EnterType(ctx *TypeContext) {}

// ExitType is called when production type is exited.
func (s *BaseSlateParserListener) ExitType(ctx *TypeContext) {}

// EnterReturnType is called when production returnType is entered.
func (s *BaseSlateParserListener) EnterReturnType(ctx *ReturnTypeContext) {}

// ExitReturnType is called when production returnType is exited.
func (s *BaseSlateParserListener) ExitReturnType(ctx *ReturnTypeContext) {}

// EnterStatement is called when production statement is entered.
func (s *BaseSlateParserListener) EnterStatement(ctx *StatementContext) {}

// ExitStatement is called when production statement is exited.
func (s *BaseSlateParserListener) ExitStatement(ctx *StatementContext) {}

// EnterVariableDecl is called when production variableDecl is entered.
func (s *BaseSlateParserListener) EnterVariableDecl(ctx *VariableDeclContext) {}

// ExitVariableDecl is called when production variableDecl is exited.
func (s *BaseSlateParserListener) ExitVariableDecl(ctx *VariableDeclContext) {}

// EnterAssignment is called when production assignment is entered.
func (s *BaseSlateParserListener) EnterAssignment(ctx *AssignmentContext) {}

// ExitAssignment is called when production assignment is exited.
func (s *BaseSlateParserListener) ExitAssignment(ctx *AssignmentContext) {}

// EnterChannelWrite is called when production channelWrite is entered.
func (s *BaseSlateParserListener) EnterChannelWrite(ctx *ChannelWriteContext) {}

// ExitChannelWrite is called when production channelWrite is exited.
func (s *BaseSlateParserListener) ExitChannelWrite(ctx *ChannelWriteContext) {}

// EnterChannelRead is called when production channelRead is entered.
func (s *BaseSlateParserListener) EnterChannelRead(ctx *ChannelReadContext) {}

// ExitChannelRead is called when production channelRead is exited.
func (s *BaseSlateParserListener) ExitChannelRead(ctx *ChannelReadContext) {}

// EnterIfStatement is called when production ifStatement is entered.
func (s *BaseSlateParserListener) EnterIfStatement(ctx *IfStatementContext) {}

// ExitIfStatement is called when production ifStatement is exited.
func (s *BaseSlateParserListener) ExitIfStatement(ctx *IfStatementContext) {}

// EnterReturnStatement is called when production returnStatement is entered.
func (s *BaseSlateParserListener) EnterReturnStatement(ctx *ReturnStatementContext) {}

// ExitReturnStatement is called when production returnStatement is exited.
func (s *BaseSlateParserListener) ExitReturnStatement(ctx *ReturnStatementContext) {}

// EnterExpressionStatement is called when production expressionStatement is entered.
func (s *BaseSlateParserListener) EnterExpressionStatement(ctx *ExpressionStatementContext) {}

// ExitExpressionStatement is called when production expressionStatement is exited.
func (s *BaseSlateParserListener) ExitExpressionStatement(ctx *ExpressionStatementContext) {}

// EnterBlock is called when production block is entered.
func (s *BaseSlateParserListener) EnterBlock(ctx *BlockContext) {}

// ExitBlock is called when production block is exited.
func (s *BaseSlateParserListener) ExitBlock(ctx *BlockContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseSlateParserListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseSlateParserListener) ExitExpression(ctx *ExpressionContext) {}

// EnterLogicalOrExpr is called when production logicalOrExpr is entered.
func (s *BaseSlateParserListener) EnterLogicalOrExpr(ctx *LogicalOrExprContext) {}

// ExitLogicalOrExpr is called when production logicalOrExpr is exited.
func (s *BaseSlateParserListener) ExitLogicalOrExpr(ctx *LogicalOrExprContext) {}

// EnterLogicalAndExpr is called when production logicalAndExpr is entered.
func (s *BaseSlateParserListener) EnterLogicalAndExpr(ctx *LogicalAndExprContext) {}

// ExitLogicalAndExpr is called when production logicalAndExpr is exited.
func (s *BaseSlateParserListener) ExitLogicalAndExpr(ctx *LogicalAndExprContext) {}

// EnterEqualityExpr is called when production equalityExpr is entered.
func (s *BaseSlateParserListener) EnterEqualityExpr(ctx *EqualityExprContext) {}

// ExitEqualityExpr is called when production equalityExpr is exited.
func (s *BaseSlateParserListener) ExitEqualityExpr(ctx *EqualityExprContext) {}

// EnterRelationalExpr is called when production relationalExpr is entered.
func (s *BaseSlateParserListener) EnterRelationalExpr(ctx *RelationalExprContext) {}

// ExitRelationalExpr is called when production relationalExpr is exited.
func (s *BaseSlateParserListener) ExitRelationalExpr(ctx *RelationalExprContext) {}

// EnterAdditiveExpr is called when production additiveExpr is entered.
func (s *BaseSlateParserListener) EnterAdditiveExpr(ctx *AdditiveExprContext) {}

// ExitAdditiveExpr is called when production additiveExpr is exited.
func (s *BaseSlateParserListener) ExitAdditiveExpr(ctx *AdditiveExprContext) {}

// EnterMultiplicativeExpr is called when production multiplicativeExpr is entered.
func (s *BaseSlateParserListener) EnterMultiplicativeExpr(ctx *MultiplicativeExprContext) {}

// ExitMultiplicativeExpr is called when production multiplicativeExpr is exited.
func (s *BaseSlateParserListener) ExitMultiplicativeExpr(ctx *MultiplicativeExprContext) {}

// EnterUnaryExpr is called when production unaryExpr is entered.
func (s *BaseSlateParserListener) EnterUnaryExpr(ctx *UnaryExprContext) {}

// ExitUnaryExpr is called when production unaryExpr is exited.
func (s *BaseSlateParserListener) ExitUnaryExpr(ctx *UnaryExprContext) {}

// EnterPrimaryExpr is called when production primaryExpr is entered.
func (s *BaseSlateParserListener) EnterPrimaryExpr(ctx *PrimaryExprContext) {}

// ExitPrimaryExpr is called when production primaryExpr is exited.
func (s *BaseSlateParserListener) ExitPrimaryExpr(ctx *PrimaryExprContext) {}

// EnterFunctionCall is called when production functionCall is entered.
func (s *BaseSlateParserListener) EnterFunctionCall(ctx *FunctionCallContext) {}

// ExitFunctionCall is called when production functionCall is exited.
func (s *BaseSlateParserListener) ExitFunctionCall(ctx *FunctionCallContext) {}

// EnterArgumentList is called when production argumentList is entered.
func (s *BaseSlateParserListener) EnterArgumentList(ctx *ArgumentListContext) {}

// ExitArgumentList is called when production argumentList is exited.
func (s *BaseSlateParserListener) ExitArgumentList(ctx *ArgumentListContext) {}
