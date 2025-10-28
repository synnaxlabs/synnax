// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Code generated from ArcParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // ArcParser
import "github.com/antlr4-go/antlr/v4"

// BaseArcParserListener is a complete listener for a parse tree produced by ArcParser.
type BaseArcParserListener struct{}

var _ ArcParserListener = &BaseArcParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseArcParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseArcParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseArcParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseArcParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterProgram is called when production program is entered.
func (s *BaseArcParserListener) EnterProgram(ctx *ProgramContext) {}

// ExitProgram is called when production program is exited.
func (s *BaseArcParserListener) ExitProgram(ctx *ProgramContext) {}

// EnterTopLevelItem is called when production topLevelItem is entered.
func (s *BaseArcParserListener) EnterTopLevelItem(ctx *TopLevelItemContext) {}

// ExitTopLevelItem is called when production topLevelItem is exited.
func (s *BaseArcParserListener) ExitTopLevelItem(ctx *TopLevelItemContext) {}

// EnterFunctionDeclaration is called when production functionDeclaration is entered.
func (s *BaseArcParserListener) EnterFunctionDeclaration(ctx *FunctionDeclarationContext) {}

// ExitFunctionDeclaration is called when production functionDeclaration is exited.
func (s *BaseArcParserListener) ExitFunctionDeclaration(ctx *FunctionDeclarationContext) {}

// EnterInputList is called when production inputList is entered.
func (s *BaseArcParserListener) EnterInputList(ctx *InputListContext) {}

// ExitInputList is called when production inputList is exited.
func (s *BaseArcParserListener) ExitInputList(ctx *InputListContext) {}

// EnterInput is called when production input is entered.
func (s *BaseArcParserListener) EnterInput(ctx *InputContext) {}

// ExitInput is called when production input is exited.
func (s *BaseArcParserListener) ExitInput(ctx *InputContext) {}

// EnterOutputType is called when production outputType is entered.
func (s *BaseArcParserListener) EnterOutputType(ctx *OutputTypeContext) {}

// ExitOutputType is called when production outputType is exited.
func (s *BaseArcParserListener) ExitOutputType(ctx *OutputTypeContext) {}

// EnterMultiOutputBlock is called when production multiOutputBlock is entered.
func (s *BaseArcParserListener) EnterMultiOutputBlock(ctx *MultiOutputBlockContext) {}

// ExitMultiOutputBlock is called when production multiOutputBlock is exited.
func (s *BaseArcParserListener) ExitMultiOutputBlock(ctx *MultiOutputBlockContext) {}

// EnterNamedOutput is called when production namedOutput is entered.
func (s *BaseArcParserListener) EnterNamedOutput(ctx *NamedOutputContext) {}

// ExitNamedOutput is called when production namedOutput is exited.
func (s *BaseArcParserListener) ExitNamedOutput(ctx *NamedOutputContext) {}

// EnterConfigBlock is called when production configBlock is entered.
func (s *BaseArcParserListener) EnterConfigBlock(ctx *ConfigBlockContext) {}

// ExitConfigBlock is called when production configBlock is exited.
func (s *BaseArcParserListener) ExitConfigBlock(ctx *ConfigBlockContext) {}

// EnterConfig is called when production config is entered.
func (s *BaseArcParserListener) EnterConfig(ctx *ConfigContext) {}

// ExitConfig is called when production config is exited.
func (s *BaseArcParserListener) ExitConfig(ctx *ConfigContext) {}

// EnterFlowStatement is called when production flowStatement is entered.
func (s *BaseArcParserListener) EnterFlowStatement(ctx *FlowStatementContext) {}

// ExitFlowStatement is called when production flowStatement is exited.
func (s *BaseArcParserListener) ExitFlowStatement(ctx *FlowStatementContext) {}

// EnterRoutingTable is called when production routingTable is entered.
func (s *BaseArcParserListener) EnterRoutingTable(ctx *RoutingTableContext) {}

// ExitRoutingTable is called when production routingTable is exited.
func (s *BaseArcParserListener) ExitRoutingTable(ctx *RoutingTableContext) {}

// EnterRoutingEntry is called when production routingEntry is entered.
func (s *BaseArcParserListener) EnterRoutingEntry(ctx *RoutingEntryContext) {}

// ExitRoutingEntry is called when production routingEntry is exited.
func (s *BaseArcParserListener) ExitRoutingEntry(ctx *RoutingEntryContext) {}

// EnterFlowNode is called when production flowNode is entered.
func (s *BaseArcParserListener) EnterFlowNode(ctx *FlowNodeContext) {}

// ExitFlowNode is called when production flowNode is exited.
func (s *BaseArcParserListener) ExitFlowNode(ctx *FlowNodeContext) {}

// EnterChannelIdentifier is called when production channelIdentifier is entered.
func (s *BaseArcParserListener) EnterChannelIdentifier(ctx *ChannelIdentifierContext) {}

// ExitChannelIdentifier is called when production channelIdentifier is exited.
func (s *BaseArcParserListener) ExitChannelIdentifier(ctx *ChannelIdentifierContext) {}

// EnterFunction is called when production function is entered.
func (s *BaseArcParserListener) EnterFunction(ctx *FunctionContext) {}

// ExitFunction is called when production function is exited.
func (s *BaseArcParserListener) ExitFunction(ctx *FunctionContext) {}

// EnterConfigValues is called when production configValues is entered.
func (s *BaseArcParserListener) EnterConfigValues(ctx *ConfigValuesContext) {}

// ExitConfigValues is called when production configValues is exited.
func (s *BaseArcParserListener) ExitConfigValues(ctx *ConfigValuesContext) {}

// EnterNamedConfigValues is called when production namedConfigValues is entered.
func (s *BaseArcParserListener) EnterNamedConfigValues(ctx *NamedConfigValuesContext) {}

// ExitNamedConfigValues is called when production namedConfigValues is exited.
func (s *BaseArcParserListener) ExitNamedConfigValues(ctx *NamedConfigValuesContext) {}

// EnterNamedConfigValue is called when production namedConfigValue is entered.
func (s *BaseArcParserListener) EnterNamedConfigValue(ctx *NamedConfigValueContext) {}

// ExitNamedConfigValue is called when production namedConfigValue is exited.
func (s *BaseArcParserListener) ExitNamedConfigValue(ctx *NamedConfigValueContext) {}

// EnterAnonymousConfigValues is called when production anonymousConfigValues is entered.
func (s *BaseArcParserListener) EnterAnonymousConfigValues(ctx *AnonymousConfigValuesContext) {}

// ExitAnonymousConfigValues is called when production anonymousConfigValues is exited.
func (s *BaseArcParserListener) ExitAnonymousConfigValues(ctx *AnonymousConfigValuesContext) {}

// EnterArguments is called when production arguments is entered.
func (s *BaseArcParserListener) EnterArguments(ctx *ArgumentsContext) {}

// ExitArguments is called when production arguments is exited.
func (s *BaseArcParserListener) ExitArguments(ctx *ArgumentsContext) {}

// EnterArgumentList is called when production argumentList is entered.
func (s *BaseArcParserListener) EnterArgumentList(ctx *ArgumentListContext) {}

// ExitArgumentList is called when production argumentList is exited.
func (s *BaseArcParserListener) ExitArgumentList(ctx *ArgumentListContext) {}

// EnterBlock is called when production block is entered.
func (s *BaseArcParserListener) EnterBlock(ctx *BlockContext) {}

// ExitBlock is called when production block is exited.
func (s *BaseArcParserListener) ExitBlock(ctx *BlockContext) {}

// EnterStatement is called when production statement is entered.
func (s *BaseArcParserListener) EnterStatement(ctx *StatementContext) {}

// ExitStatement is called when production statement is exited.
func (s *BaseArcParserListener) ExitStatement(ctx *StatementContext) {}

// EnterVariableDeclaration is called when production variableDeclaration is entered.
func (s *BaseArcParserListener) EnterVariableDeclaration(ctx *VariableDeclarationContext) {}

// ExitVariableDeclaration is called when production variableDeclaration is exited.
func (s *BaseArcParserListener) ExitVariableDeclaration(ctx *VariableDeclarationContext) {}

// EnterLocalVariable is called when production localVariable is entered.
func (s *BaseArcParserListener) EnterLocalVariable(ctx *LocalVariableContext) {}

// ExitLocalVariable is called when production localVariable is exited.
func (s *BaseArcParserListener) ExitLocalVariable(ctx *LocalVariableContext) {}

// EnterStatefulVariable is called when production statefulVariable is entered.
func (s *BaseArcParserListener) EnterStatefulVariable(ctx *StatefulVariableContext) {}

// ExitStatefulVariable is called when production statefulVariable is exited.
func (s *BaseArcParserListener) ExitStatefulVariable(ctx *StatefulVariableContext) {}

// EnterAssignment is called when production assignment is entered.
func (s *BaseArcParserListener) EnterAssignment(ctx *AssignmentContext) {}

// ExitAssignment is called when production assignment is exited.
func (s *BaseArcParserListener) ExitAssignment(ctx *AssignmentContext) {}

// EnterChannelOperation is called when production channelOperation is entered.
func (s *BaseArcParserListener) EnterChannelOperation(ctx *ChannelOperationContext) {}

// ExitChannelOperation is called when production channelOperation is exited.
func (s *BaseArcParserListener) ExitChannelOperation(ctx *ChannelOperationContext) {}

// EnterChannelWrite is called when production channelWrite is entered.
func (s *BaseArcParserListener) EnterChannelWrite(ctx *ChannelWriteContext) {}

// ExitChannelWrite is called when production channelWrite is exited.
func (s *BaseArcParserListener) ExitChannelWrite(ctx *ChannelWriteContext) {}

// EnterChannelRead is called when production channelRead is entered.
func (s *BaseArcParserListener) EnterChannelRead(ctx *ChannelReadContext) {}

// ExitChannelRead is called when production channelRead is exited.
func (s *BaseArcParserListener) ExitChannelRead(ctx *ChannelReadContext) {}

// EnterBlockingRead is called when production blockingRead is entered.
func (s *BaseArcParserListener) EnterBlockingRead(ctx *BlockingReadContext) {}

// ExitBlockingRead is called when production blockingRead is exited.
func (s *BaseArcParserListener) ExitBlockingRead(ctx *BlockingReadContext) {}

// EnterNonBlockingRead is called when production nonBlockingRead is entered.
func (s *BaseArcParserListener) EnterNonBlockingRead(ctx *NonBlockingReadContext) {}

// ExitNonBlockingRead is called when production nonBlockingRead is exited.
func (s *BaseArcParserListener) ExitNonBlockingRead(ctx *NonBlockingReadContext) {}

// EnterIfStatement is called when production ifStatement is entered.
func (s *BaseArcParserListener) EnterIfStatement(ctx *IfStatementContext) {}

// ExitIfStatement is called when production ifStatement is exited.
func (s *BaseArcParserListener) ExitIfStatement(ctx *IfStatementContext) {}

// EnterElseIfClause is called when production elseIfClause is entered.
func (s *BaseArcParserListener) EnterElseIfClause(ctx *ElseIfClauseContext) {}

// ExitElseIfClause is called when production elseIfClause is exited.
func (s *BaseArcParserListener) ExitElseIfClause(ctx *ElseIfClauseContext) {}

// EnterElseClause is called when production elseClause is entered.
func (s *BaseArcParserListener) EnterElseClause(ctx *ElseClauseContext) {}

// ExitElseClause is called when production elseClause is exited.
func (s *BaseArcParserListener) ExitElseClause(ctx *ElseClauseContext) {}

// EnterReturnStatement is called when production returnStatement is entered.
func (s *BaseArcParserListener) EnterReturnStatement(ctx *ReturnStatementContext) {}

// ExitReturnStatement is called when production returnStatement is exited.
func (s *BaseArcParserListener) ExitReturnStatement(ctx *ReturnStatementContext) {}

// EnterFunctionCall is called when production functionCall is entered.
func (s *BaseArcParserListener) EnterFunctionCall(ctx *FunctionCallContext) {}

// ExitFunctionCall is called when production functionCall is exited.
func (s *BaseArcParserListener) ExitFunctionCall(ctx *FunctionCallContext) {}

// EnterType is called when production type is entered.
func (s *BaseArcParserListener) EnterType(ctx *TypeContext) {}

// ExitType is called when production type is exited.
func (s *BaseArcParserListener) ExitType(ctx *TypeContext) {}

// EnterPrimitiveType is called when production primitiveType is entered.
func (s *BaseArcParserListener) EnterPrimitiveType(ctx *PrimitiveTypeContext) {}

// ExitPrimitiveType is called when production primitiveType is exited.
func (s *BaseArcParserListener) ExitPrimitiveType(ctx *PrimitiveTypeContext) {}

// EnterNumericType is called when production numericType is entered.
func (s *BaseArcParserListener) EnterNumericType(ctx *NumericTypeContext) {}

// ExitNumericType is called when production numericType is exited.
func (s *BaseArcParserListener) ExitNumericType(ctx *NumericTypeContext) {}

// EnterIntegerType is called when production integerType is entered.
func (s *BaseArcParserListener) EnterIntegerType(ctx *IntegerTypeContext) {}

// ExitIntegerType is called when production integerType is exited.
func (s *BaseArcParserListener) ExitIntegerType(ctx *IntegerTypeContext) {}

// EnterFloatType is called when production floatType is entered.
func (s *BaseArcParserListener) EnterFloatType(ctx *FloatTypeContext) {}

// ExitFloatType is called when production floatType is exited.
func (s *BaseArcParserListener) ExitFloatType(ctx *FloatTypeContext) {}

// EnterTemporalType is called when production temporalType is entered.
func (s *BaseArcParserListener) EnterTemporalType(ctx *TemporalTypeContext) {}

// ExitTemporalType is called when production temporalType is exited.
func (s *BaseArcParserListener) ExitTemporalType(ctx *TemporalTypeContext) {}

// EnterChannelType is called when production channelType is entered.
func (s *BaseArcParserListener) EnterChannelType(ctx *ChannelTypeContext) {}

// ExitChannelType is called when production channelType is exited.
func (s *BaseArcParserListener) ExitChannelType(ctx *ChannelTypeContext) {}

// EnterSeriesType is called when production seriesType is entered.
func (s *BaseArcParserListener) EnterSeriesType(ctx *SeriesTypeContext) {}

// ExitSeriesType is called when production seriesType is exited.
func (s *BaseArcParserListener) ExitSeriesType(ctx *SeriesTypeContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseArcParserListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseArcParserListener) ExitExpression(ctx *ExpressionContext) {}

// EnterLogicalOrExpression is called when production logicalOrExpression is entered.
func (s *BaseArcParserListener) EnterLogicalOrExpression(ctx *LogicalOrExpressionContext) {}

// ExitLogicalOrExpression is called when production logicalOrExpression is exited.
func (s *BaseArcParserListener) ExitLogicalOrExpression(ctx *LogicalOrExpressionContext) {}

// EnterLogicalAndExpression is called when production logicalAndExpression is entered.
func (s *BaseArcParserListener) EnterLogicalAndExpression(ctx *LogicalAndExpressionContext) {}

// ExitLogicalAndExpression is called when production logicalAndExpression is exited.
func (s *BaseArcParserListener) ExitLogicalAndExpression(ctx *LogicalAndExpressionContext) {}

// EnterEqualityExpression is called when production equalityExpression is entered.
func (s *BaseArcParserListener) EnterEqualityExpression(ctx *EqualityExpressionContext) {}

// ExitEqualityExpression is called when production equalityExpression is exited.
func (s *BaseArcParserListener) ExitEqualityExpression(ctx *EqualityExpressionContext) {}

// EnterRelationalExpression is called when production relationalExpression is entered.
func (s *BaseArcParserListener) EnterRelationalExpression(ctx *RelationalExpressionContext) {}

// ExitRelationalExpression is called when production relationalExpression is exited.
func (s *BaseArcParserListener) ExitRelationalExpression(ctx *RelationalExpressionContext) {}

// EnterAdditiveExpression is called when production additiveExpression is entered.
func (s *BaseArcParserListener) EnterAdditiveExpression(ctx *AdditiveExpressionContext) {}

// ExitAdditiveExpression is called when production additiveExpression is exited.
func (s *BaseArcParserListener) ExitAdditiveExpression(ctx *AdditiveExpressionContext) {}

// EnterMultiplicativeExpression is called when production multiplicativeExpression is entered.
func (s *BaseArcParserListener) EnterMultiplicativeExpression(ctx *MultiplicativeExpressionContext) {}

// ExitMultiplicativeExpression is called when production multiplicativeExpression is exited.
func (s *BaseArcParserListener) ExitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) {}

// EnterPowerExpression is called when production powerExpression is entered.
func (s *BaseArcParserListener) EnterPowerExpression(ctx *PowerExpressionContext) {}

// ExitPowerExpression is called when production powerExpression is exited.
func (s *BaseArcParserListener) ExitPowerExpression(ctx *PowerExpressionContext) {}

// EnterUnaryExpression is called when production unaryExpression is entered.
func (s *BaseArcParserListener) EnterUnaryExpression(ctx *UnaryExpressionContext) {}

// ExitUnaryExpression is called when production unaryExpression is exited.
func (s *BaseArcParserListener) ExitUnaryExpression(ctx *UnaryExpressionContext) {}

// EnterBlockingReadExpr is called when production blockingReadExpr is entered.
func (s *BaseArcParserListener) EnterBlockingReadExpr(ctx *BlockingReadExprContext) {}

// ExitBlockingReadExpr is called when production blockingReadExpr is exited.
func (s *BaseArcParserListener) ExitBlockingReadExpr(ctx *BlockingReadExprContext) {}

// EnterPostfixExpression is called when production postfixExpression is entered.
func (s *BaseArcParserListener) EnterPostfixExpression(ctx *PostfixExpressionContext) {}

// ExitPostfixExpression is called when production postfixExpression is exited.
func (s *BaseArcParserListener) ExitPostfixExpression(ctx *PostfixExpressionContext) {}

// EnterIndexOrSlice is called when production indexOrSlice is entered.
func (s *BaseArcParserListener) EnterIndexOrSlice(ctx *IndexOrSliceContext) {}

// ExitIndexOrSlice is called when production indexOrSlice is exited.
func (s *BaseArcParserListener) ExitIndexOrSlice(ctx *IndexOrSliceContext) {}

// EnterFunctionCallSuffix is called when production functionCallSuffix is entered.
func (s *BaseArcParserListener) EnterFunctionCallSuffix(ctx *FunctionCallSuffixContext) {}

// ExitFunctionCallSuffix is called when production functionCallSuffix is exited.
func (s *BaseArcParserListener) ExitFunctionCallSuffix(ctx *FunctionCallSuffixContext) {}

// EnterPrimaryExpression is called when production primaryExpression is entered.
func (s *BaseArcParserListener) EnterPrimaryExpression(ctx *PrimaryExpressionContext) {}

// ExitPrimaryExpression is called when production primaryExpression is exited.
func (s *BaseArcParserListener) ExitPrimaryExpression(ctx *PrimaryExpressionContext) {}

// EnterTypeCast is called when production typeCast is entered.
func (s *BaseArcParserListener) EnterTypeCast(ctx *TypeCastContext) {}

// ExitTypeCast is called when production typeCast is exited.
func (s *BaseArcParserListener) ExitTypeCast(ctx *TypeCastContext) {}

// EnterBuiltinFunction is called when production builtinFunction is entered.
func (s *BaseArcParserListener) EnterBuiltinFunction(ctx *BuiltinFunctionContext) {}

// ExitBuiltinFunction is called when production builtinFunction is exited.
func (s *BaseArcParserListener) ExitBuiltinFunction(ctx *BuiltinFunctionContext) {}

// EnterLiteral is called when production literal is entered.
func (s *BaseArcParserListener) EnterLiteral(ctx *LiteralContext) {}

// ExitLiteral is called when production literal is exited.
func (s *BaseArcParserListener) ExitLiteral(ctx *LiteralContext) {}

// EnterNumericLiteral is called when production numericLiteral is entered.
func (s *BaseArcParserListener) EnterNumericLiteral(ctx *NumericLiteralContext) {}

// ExitNumericLiteral is called when production numericLiteral is exited.
func (s *BaseArcParserListener) ExitNumericLiteral(ctx *NumericLiteralContext) {}

// EnterTemporalLiteral is called when production temporalLiteral is entered.
func (s *BaseArcParserListener) EnterTemporalLiteral(ctx *TemporalLiteralContext) {}

// ExitTemporalLiteral is called when production temporalLiteral is exited.
func (s *BaseArcParserListener) ExitTemporalLiteral(ctx *TemporalLiteralContext) {}

// EnterSeriesLiteral is called when production seriesLiteral is entered.
func (s *BaseArcParserListener) EnterSeriesLiteral(ctx *SeriesLiteralContext) {}

// ExitSeriesLiteral is called when production seriesLiteral is exited.
func (s *BaseArcParserListener) ExitSeriesLiteral(ctx *SeriesLiteralContext) {}

// EnterExpressionList is called when production expressionList is entered.
func (s *BaseArcParserListener) EnterExpressionList(ctx *ExpressionListContext) {}

// ExitExpressionList is called when production expressionList is exited.
func (s *BaseArcParserListener) ExitExpressionList(ctx *ExpressionListContext) {}
