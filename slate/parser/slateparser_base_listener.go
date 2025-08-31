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

// EnterTopLevelItem is called when production topLevelItem is entered.
func (s *BaseSlateParserListener) EnterTopLevelItem(ctx *TopLevelItemContext) {}

// ExitTopLevelItem is called when production topLevelItem is exited.
func (s *BaseSlateParserListener) ExitTopLevelItem(ctx *TopLevelItemContext) {}

// EnterFunctionDeclaration is called when production functionDeclaration is entered.
func (s *BaseSlateParserListener) EnterFunctionDeclaration(ctx *FunctionDeclarationContext) {}

// ExitFunctionDeclaration is called when production functionDeclaration is exited.
func (s *BaseSlateParserListener) ExitFunctionDeclaration(ctx *FunctionDeclarationContext) {}

// EnterParameterList is called when production parameterList is entered.
func (s *BaseSlateParserListener) EnterParameterList(ctx *ParameterListContext) {}

// ExitParameterList is called when production parameterList is exited.
func (s *BaseSlateParserListener) ExitParameterList(ctx *ParameterListContext) {}

// EnterParameter is called when production parameter is entered.
func (s *BaseSlateParserListener) EnterParameter(ctx *ParameterContext) {}

// ExitParameter is called when production parameter is exited.
func (s *BaseSlateParserListener) ExitParameter(ctx *ParameterContext) {}

// EnterReturnType is called when production returnType is entered.
func (s *BaseSlateParserListener) EnterReturnType(ctx *ReturnTypeContext) {}

// ExitReturnType is called when production returnType is exited.
func (s *BaseSlateParserListener) ExitReturnType(ctx *ReturnTypeContext) {}

// EnterTaskDeclaration is called when production taskDeclaration is entered.
func (s *BaseSlateParserListener) EnterTaskDeclaration(ctx *TaskDeclarationContext) {}

// ExitTaskDeclaration is called when production taskDeclaration is exited.
func (s *BaseSlateParserListener) ExitTaskDeclaration(ctx *TaskDeclarationContext) {}

// EnterConfigBlock is called when production configBlock is entered.
func (s *BaseSlateParserListener) EnterConfigBlock(ctx *ConfigBlockContext) {}

// ExitConfigBlock is called when production configBlock is exited.
func (s *BaseSlateParserListener) ExitConfigBlock(ctx *ConfigBlockContext) {}

// EnterConfigParameter is called when production configParameter is entered.
func (s *BaseSlateParserListener) EnterConfigParameter(ctx *ConfigParameterContext) {}

// ExitConfigParameter is called when production configParameter is exited.
func (s *BaseSlateParserListener) ExitConfigParameter(ctx *ConfigParameterContext) {}

// EnterFlowStatement is called when production flowStatement is entered.
func (s *BaseSlateParserListener) EnterFlowStatement(ctx *FlowStatementContext) {}

// ExitFlowStatement is called when production flowStatement is exited.
func (s *BaseSlateParserListener) ExitFlowStatement(ctx *FlowStatementContext) {}

// EnterFlowSource is called when production flowSource is entered.
func (s *BaseSlateParserListener) EnterFlowSource(ctx *FlowSourceContext) {}

// ExitFlowSource is called when production flowSource is exited.
func (s *BaseSlateParserListener) ExitFlowSource(ctx *FlowSourceContext) {}

// EnterFlowTarget is called when production flowTarget is entered.
func (s *BaseSlateParserListener) EnterFlowTarget(ctx *FlowTargetContext) {}

// ExitFlowTarget is called when production flowTarget is exited.
func (s *BaseSlateParserListener) ExitFlowTarget(ctx *FlowTargetContext) {}

// EnterChannelIdentifier is called when production channelIdentifier is entered.
func (s *BaseSlateParserListener) EnterChannelIdentifier(ctx *ChannelIdentifierContext) {}

// ExitChannelIdentifier is called when production channelIdentifier is exited.
func (s *BaseSlateParserListener) ExitChannelIdentifier(ctx *ChannelIdentifierContext) {}

// EnterTaskInvocation is called when production taskInvocation is entered.
func (s *BaseSlateParserListener) EnterTaskInvocation(ctx *TaskInvocationContext) {}

// ExitTaskInvocation is called when production taskInvocation is exited.
func (s *BaseSlateParserListener) ExitTaskInvocation(ctx *TaskInvocationContext) {}

// EnterConfigValues is called when production configValues is entered.
func (s *BaseSlateParserListener) EnterConfigValues(ctx *ConfigValuesContext) {}

// ExitConfigValues is called when production configValues is exited.
func (s *BaseSlateParserListener) ExitConfigValues(ctx *ConfigValuesContext) {}

// EnterNamedConfigValues is called when production namedConfigValues is entered.
func (s *BaseSlateParserListener) EnterNamedConfigValues(ctx *NamedConfigValuesContext) {}

// ExitNamedConfigValues is called when production namedConfigValues is exited.
func (s *BaseSlateParserListener) ExitNamedConfigValues(ctx *NamedConfigValuesContext) {}

// EnterNamedConfigValue is called when production namedConfigValue is entered.
func (s *BaseSlateParserListener) EnterNamedConfigValue(ctx *NamedConfigValueContext) {}

// ExitNamedConfigValue is called when production namedConfigValue is exited.
func (s *BaseSlateParserListener) ExitNamedConfigValue(ctx *NamedConfigValueContext) {}

// EnterAnonymousConfigValues is called when production anonymousConfigValues is entered.
func (s *BaseSlateParserListener) EnterAnonymousConfigValues(ctx *AnonymousConfigValuesContext) {}

// ExitAnonymousConfigValues is called when production anonymousConfigValues is exited.
func (s *BaseSlateParserListener) ExitAnonymousConfigValues(ctx *AnonymousConfigValuesContext) {}

// EnterArguments is called when production arguments is entered.
func (s *BaseSlateParserListener) EnterArguments(ctx *ArgumentsContext) {}

// ExitArguments is called when production arguments is exited.
func (s *BaseSlateParserListener) ExitArguments(ctx *ArgumentsContext) {}

// EnterArgumentList is called when production argumentList is entered.
func (s *BaseSlateParserListener) EnterArgumentList(ctx *ArgumentListContext) {}

// ExitArgumentList is called when production argumentList is exited.
func (s *BaseSlateParserListener) ExitArgumentList(ctx *ArgumentListContext) {}

// EnterBlock is called when production block is entered.
func (s *BaseSlateParserListener) EnterBlock(ctx *BlockContext) {}

// ExitBlock is called when production block is exited.
func (s *BaseSlateParserListener) ExitBlock(ctx *BlockContext) {}

// EnterStatement is called when production statement is entered.
func (s *BaseSlateParserListener) EnterStatement(ctx *StatementContext) {}

// ExitStatement is called when production statement is exited.
func (s *BaseSlateParserListener) ExitStatement(ctx *StatementContext) {}

// EnterVariableDeclaration is called when production variableDeclaration is entered.
func (s *BaseSlateParserListener) EnterVariableDeclaration(ctx *VariableDeclarationContext) {}

// ExitVariableDeclaration is called when production variableDeclaration is exited.
func (s *BaseSlateParserListener) ExitVariableDeclaration(ctx *VariableDeclarationContext) {}

// EnterLocalVariable is called when production localVariable is entered.
func (s *BaseSlateParserListener) EnterLocalVariable(ctx *LocalVariableContext) {}

// ExitLocalVariable is called when production localVariable is exited.
func (s *BaseSlateParserListener) ExitLocalVariable(ctx *LocalVariableContext) {}

// EnterStatefulVariable is called when production statefulVariable is entered.
func (s *BaseSlateParserListener) EnterStatefulVariable(ctx *StatefulVariableContext) {}

// ExitStatefulVariable is called when production statefulVariable is exited.
func (s *BaseSlateParserListener) ExitStatefulVariable(ctx *StatefulVariableContext) {}

// EnterAssignment is called when production assignment is entered.
func (s *BaseSlateParserListener) EnterAssignment(ctx *AssignmentContext) {}

// ExitAssignment is called when production assignment is exited.
func (s *BaseSlateParserListener) ExitAssignment(ctx *AssignmentContext) {}

// EnterChannelOperation is called when production channelOperation is entered.
func (s *BaseSlateParserListener) EnterChannelOperation(ctx *ChannelOperationContext) {}

// ExitChannelOperation is called when production channelOperation is exited.
func (s *BaseSlateParserListener) ExitChannelOperation(ctx *ChannelOperationContext) {}

// EnterChannelWrite is called when production channelWrite is entered.
func (s *BaseSlateParserListener) EnterChannelWrite(ctx *ChannelWriteContext) {}

// ExitChannelWrite is called when production channelWrite is exited.
func (s *BaseSlateParserListener) ExitChannelWrite(ctx *ChannelWriteContext) {}

// EnterChannelRead is called when production channelRead is entered.
func (s *BaseSlateParserListener) EnterChannelRead(ctx *ChannelReadContext) {}

// ExitChannelRead is called when production channelRead is exited.
func (s *BaseSlateParserListener) ExitChannelRead(ctx *ChannelReadContext) {}

// EnterBlockingRead is called when production blockingRead is entered.
func (s *BaseSlateParserListener) EnterBlockingRead(ctx *BlockingReadContext) {}

// ExitBlockingRead is called when production blockingRead is exited.
func (s *BaseSlateParserListener) ExitBlockingRead(ctx *BlockingReadContext) {}

// EnterNonBlockingRead is called when production nonBlockingRead is entered.
func (s *BaseSlateParserListener) EnterNonBlockingRead(ctx *NonBlockingReadContext) {}

// ExitNonBlockingRead is called when production nonBlockingRead is exited.
func (s *BaseSlateParserListener) ExitNonBlockingRead(ctx *NonBlockingReadContext) {}

// EnterIfStatement is called when production ifStatement is entered.
func (s *BaseSlateParserListener) EnterIfStatement(ctx *IfStatementContext) {}

// ExitIfStatement is called when production ifStatement is exited.
func (s *BaseSlateParserListener) ExitIfStatement(ctx *IfStatementContext) {}

// EnterElseIfClause is called when production elseIfClause is entered.
func (s *BaseSlateParserListener) EnterElseIfClause(ctx *ElseIfClauseContext) {}

// ExitElseIfClause is called when production elseIfClause is exited.
func (s *BaseSlateParserListener) ExitElseIfClause(ctx *ElseIfClauseContext) {}

// EnterElseClause is called when production elseClause is entered.
func (s *BaseSlateParserListener) EnterElseClause(ctx *ElseClauseContext) {}

// ExitElseClause is called when production elseClause is exited.
func (s *BaseSlateParserListener) ExitElseClause(ctx *ElseClauseContext) {}

// EnterReturnStatement is called when production returnStatement is entered.
func (s *BaseSlateParserListener) EnterReturnStatement(ctx *ReturnStatementContext) {}

// ExitReturnStatement is called when production returnStatement is exited.
func (s *BaseSlateParserListener) ExitReturnStatement(ctx *ReturnStatementContext) {}

// EnterFunctionCall is called when production functionCall is entered.
func (s *BaseSlateParserListener) EnterFunctionCall(ctx *FunctionCallContext) {}

// ExitFunctionCall is called when production functionCall is exited.
func (s *BaseSlateParserListener) ExitFunctionCall(ctx *FunctionCallContext) {}

// EnterType is called when production type is entered.
func (s *BaseSlateParserListener) EnterType(ctx *TypeContext) {}

// ExitType is called when production type is exited.
func (s *BaseSlateParserListener) ExitType(ctx *TypeContext) {}

// EnterPrimitiveType is called when production primitiveType is entered.
func (s *BaseSlateParserListener) EnterPrimitiveType(ctx *PrimitiveTypeContext) {}

// ExitPrimitiveType is called when production primitiveType is exited.
func (s *BaseSlateParserListener) ExitPrimitiveType(ctx *PrimitiveTypeContext) {}

// EnterNumericType is called when production numericType is entered.
func (s *BaseSlateParserListener) EnterNumericType(ctx *NumericTypeContext) {}

// ExitNumericType is called when production numericType is exited.
func (s *BaseSlateParserListener) ExitNumericType(ctx *NumericTypeContext) {}

// EnterIntegerType is called when production integerType is entered.
func (s *BaseSlateParserListener) EnterIntegerType(ctx *IntegerTypeContext) {}

// ExitIntegerType is called when production integerType is exited.
func (s *BaseSlateParserListener) ExitIntegerType(ctx *IntegerTypeContext) {}

// EnterFloatType is called when production floatType is entered.
func (s *BaseSlateParserListener) EnterFloatType(ctx *FloatTypeContext) {}

// ExitFloatType is called when production floatType is exited.
func (s *BaseSlateParserListener) ExitFloatType(ctx *FloatTypeContext) {}

// EnterTemporalType is called when production temporalType is entered.
func (s *BaseSlateParserListener) EnterTemporalType(ctx *TemporalTypeContext) {}

// ExitTemporalType is called when production temporalType is exited.
func (s *BaseSlateParserListener) ExitTemporalType(ctx *TemporalTypeContext) {}

// EnterChannelType is called when production channelType is entered.
func (s *BaseSlateParserListener) EnterChannelType(ctx *ChannelTypeContext) {}

// ExitChannelType is called when production channelType is exited.
func (s *BaseSlateParserListener) ExitChannelType(ctx *ChannelTypeContext) {}

// EnterSeriesType is called when production seriesType is entered.
func (s *BaseSlateParserListener) EnterSeriesType(ctx *SeriesTypeContext) {}

// ExitSeriesType is called when production seriesType is exited.
func (s *BaseSlateParserListener) ExitSeriesType(ctx *SeriesTypeContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseSlateParserListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseSlateParserListener) ExitExpression(ctx *ExpressionContext) {}

// EnterLogicalOrExpression is called when production logicalOrExpression is entered.
func (s *BaseSlateParserListener) EnterLogicalOrExpression(ctx *LogicalOrExpressionContext) {}

// ExitLogicalOrExpression is called when production logicalOrExpression is exited.
func (s *BaseSlateParserListener) ExitLogicalOrExpression(ctx *LogicalOrExpressionContext) {}

// EnterLogicalAndExpression is called when production logicalAndExpression is entered.
func (s *BaseSlateParserListener) EnterLogicalAndExpression(ctx *LogicalAndExpressionContext) {}

// ExitLogicalAndExpression is called when production logicalAndExpression is exited.
func (s *BaseSlateParserListener) ExitLogicalAndExpression(ctx *LogicalAndExpressionContext) {}

// EnterEqualityExpression is called when production equalityExpression is entered.
func (s *BaseSlateParserListener) EnterEqualityExpression(ctx *EqualityExpressionContext) {}

// ExitEqualityExpression is called when production equalityExpression is exited.
func (s *BaseSlateParserListener) ExitEqualityExpression(ctx *EqualityExpressionContext) {}

// EnterRelationalExpression is called when production relationalExpression is entered.
func (s *BaseSlateParserListener) EnterRelationalExpression(ctx *RelationalExpressionContext) {}

// ExitRelationalExpression is called when production relationalExpression is exited.
func (s *BaseSlateParserListener) ExitRelationalExpression(ctx *RelationalExpressionContext) {}

// EnterAdditiveExpression is called when production additiveExpression is entered.
func (s *BaseSlateParserListener) EnterAdditiveExpression(ctx *AdditiveExpressionContext) {}

// ExitAdditiveExpression is called when production additiveExpression is exited.
func (s *BaseSlateParserListener) ExitAdditiveExpression(ctx *AdditiveExpressionContext) {}

// EnterMultiplicativeExpression is called when production multiplicativeExpression is entered.
func (s *BaseSlateParserListener) EnterMultiplicativeExpression(ctx *MultiplicativeExpressionContext) {
}

// ExitMultiplicativeExpression is called when production multiplicativeExpression is exited.
func (s *BaseSlateParserListener) ExitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) {
}

// EnterPowerExpression is called when production powerExpression is entered.
func (s *BaseSlateParserListener) EnterPowerExpression(ctx *PowerExpressionContext) {}

// ExitPowerExpression is called when production powerExpression is exited.
func (s *BaseSlateParserListener) ExitPowerExpression(ctx *PowerExpressionContext) {}

// EnterUnaryExpression is called when production unaryExpression is entered.
func (s *BaseSlateParserListener) EnterUnaryExpression(ctx *UnaryExpressionContext) {}

// ExitUnaryExpression is called when production unaryExpression is exited.
func (s *BaseSlateParserListener) ExitUnaryExpression(ctx *UnaryExpressionContext) {}

// EnterBlockingReadExpr is called when production blockingReadExpr is entered.
func (s *BaseSlateParserListener) EnterBlockingReadExpr(ctx *BlockingReadExprContext) {}

// ExitBlockingReadExpr is called when production blockingReadExpr is exited.
func (s *BaseSlateParserListener) ExitBlockingReadExpr(ctx *BlockingReadExprContext) {}

// EnterPostfixExpression is called when production postfixExpression is entered.
func (s *BaseSlateParserListener) EnterPostfixExpression(ctx *PostfixExpressionContext) {}

// ExitPostfixExpression is called when production postfixExpression is exited.
func (s *BaseSlateParserListener) ExitPostfixExpression(ctx *PostfixExpressionContext) {}

// EnterIndexOrSlice is called when production indexOrSlice is entered.
func (s *BaseSlateParserListener) EnterIndexOrSlice(ctx *IndexOrSliceContext) {}

// ExitIndexOrSlice is called when production indexOrSlice is exited.
func (s *BaseSlateParserListener) ExitIndexOrSlice(ctx *IndexOrSliceContext) {}

// EnterFunctionCallSuffix is called when production functionCallSuffix is entered.
func (s *BaseSlateParserListener) EnterFunctionCallSuffix(ctx *FunctionCallSuffixContext) {}

// ExitFunctionCallSuffix is called when production functionCallSuffix is exited.
func (s *BaseSlateParserListener) ExitFunctionCallSuffix(ctx *FunctionCallSuffixContext) {}

// EnterPrimaryExpression is called when production primaryExpression is entered.
func (s *BaseSlateParserListener) EnterPrimaryExpression(ctx *PrimaryExpressionContext) {}

// ExitPrimaryExpression is called when production primaryExpression is exited.
func (s *BaseSlateParserListener) ExitPrimaryExpression(ctx *PrimaryExpressionContext) {}

// EnterTypeCast is called when production typeCast is entered.
func (s *BaseSlateParserListener) EnterTypeCast(ctx *TypeCastContext) {}

// ExitTypeCast is called when production typeCast is exited.
func (s *BaseSlateParserListener) ExitTypeCast(ctx *TypeCastContext) {}

// EnterBuiltinFunction is called when production builtinFunction is entered.
func (s *BaseSlateParserListener) EnterBuiltinFunction(ctx *BuiltinFunctionContext) {}

// ExitBuiltinFunction is called when production builtinFunction is exited.
func (s *BaseSlateParserListener) ExitBuiltinFunction(ctx *BuiltinFunctionContext) {}

// EnterLiteral is called when production literal is entered.
func (s *BaseSlateParserListener) EnterLiteral(ctx *LiteralContext) {}

// ExitLiteral is called when production literal is exited.
func (s *BaseSlateParserListener) ExitLiteral(ctx *LiteralContext) {}

// EnterNumericLiteral is called when production numericLiteral is entered.
func (s *BaseSlateParserListener) EnterNumericLiteral(ctx *NumericLiteralContext) {}

// ExitNumericLiteral is called when production numericLiteral is exited.
func (s *BaseSlateParserListener) ExitNumericLiteral(ctx *NumericLiteralContext) {}

// EnterTemporalLiteral is called when production temporalLiteral is entered.
func (s *BaseSlateParserListener) EnterTemporalLiteral(ctx *TemporalLiteralContext) {}

// ExitTemporalLiteral is called when production temporalLiteral is exited.
func (s *BaseSlateParserListener) ExitTemporalLiteral(ctx *TemporalLiteralContext) {}

// EnterSeriesLiteral is called when production seriesLiteral is entered.
func (s *BaseSlateParserListener) EnterSeriesLiteral(ctx *SeriesLiteralContext) {}

// ExitSeriesLiteral is called when production seriesLiteral is exited.
func (s *BaseSlateParserListener) ExitSeriesLiteral(ctx *SeriesLiteralContext) {}

// EnterExpressionList is called when production expressionList is entered.
func (s *BaseSlateParserListener) EnterExpressionList(ctx *ExpressionListContext) {}

// ExitExpressionList is called when production expressionList is exited.
func (s *BaseSlateParserListener) ExitExpressionList(ctx *ExpressionListContext) {}
