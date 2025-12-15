// Code generated from ArcParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // ArcParser

import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by ArcParser.
type ArcParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by ArcParser#program.
	VisitProgram(ctx *ProgramContext) interface{}

	// Visit a parse tree produced by ArcParser#topLevelItem.
	VisitTopLevelItem(ctx *TopLevelItemContext) interface{}

	// Visit a parse tree produced by ArcParser#functionDeclaration.
	VisitFunctionDeclaration(ctx *FunctionDeclarationContext) interface{}

	// Visit a parse tree produced by ArcParser#inputList.
	VisitInputList(ctx *InputListContext) interface{}

	// Visit a parse tree produced by ArcParser#input.
	VisitInput(ctx *InputContext) interface{}

	// Visit a parse tree produced by ArcParser#outputType.
	VisitOutputType(ctx *OutputTypeContext) interface{}

	// Visit a parse tree produced by ArcParser#multiOutputBlock.
	VisitMultiOutputBlock(ctx *MultiOutputBlockContext) interface{}

	// Visit a parse tree produced by ArcParser#namedOutput.
	VisitNamedOutput(ctx *NamedOutputContext) interface{}

	// Visit a parse tree produced by ArcParser#configBlock.
	VisitConfigBlock(ctx *ConfigBlockContext) interface{}

	// Visit a parse tree produced by ArcParser#config.
	VisitConfig(ctx *ConfigContext) interface{}

	// Visit a parse tree produced by ArcParser#sequenceDeclaration.
	VisitSequenceDeclaration(ctx *SequenceDeclarationContext) interface{}

	// Visit a parse tree produced by ArcParser#stageDeclaration.
	VisitStageDeclaration(ctx *StageDeclarationContext) interface{}

	// Visit a parse tree produced by ArcParser#stageBody.
	VisitStageBody(ctx *StageBodyContext) interface{}

	// Visit a parse tree produced by ArcParser#stageItem.
	VisitStageItem(ctx *StageItemContext) interface{}

	// Visit a parse tree produced by ArcParser#timerBuiltin.
	VisitTimerBuiltin(ctx *TimerBuiltinContext) interface{}

	// Visit a parse tree produced by ArcParser#logBuiltin.
	VisitLogBuiltin(ctx *LogBuiltinContext) interface{}

	// Visit a parse tree produced by ArcParser#matchBlock.
	VisitMatchBlock(ctx *MatchBlockContext) interface{}

	// Visit a parse tree produced by ArcParser#matchEntry.
	VisitMatchEntry(ctx *MatchEntryContext) interface{}

	// Visit a parse tree produced by ArcParser#flowStatement.
	VisitFlowStatement(ctx *FlowStatementContext) interface{}

	// Visit a parse tree produced by ArcParser#flowOperator.
	VisitFlowOperator(ctx *FlowOperatorContext) interface{}

	// Visit a parse tree produced by ArcParser#routingTable.
	VisitRoutingTable(ctx *RoutingTableContext) interface{}

	// Visit a parse tree produced by ArcParser#routingEntry.
	VisitRoutingEntry(ctx *RoutingEntryContext) interface{}

	// Visit a parse tree produced by ArcParser#flowNode.
	VisitFlowNode(ctx *FlowNodeContext) interface{}

	// Visit a parse tree produced by ArcParser#identifier.
	VisitIdentifier(ctx *IdentifierContext) interface{}

	// Visit a parse tree produced by ArcParser#function.
	VisitFunction(ctx *FunctionContext) interface{}

	// Visit a parse tree produced by ArcParser#configValues.
	VisitConfigValues(ctx *ConfigValuesContext) interface{}

	// Visit a parse tree produced by ArcParser#namedConfigValues.
	VisitNamedConfigValues(ctx *NamedConfigValuesContext) interface{}

	// Visit a parse tree produced by ArcParser#namedConfigValue.
	VisitNamedConfigValue(ctx *NamedConfigValueContext) interface{}

	// Visit a parse tree produced by ArcParser#anonymousConfigValues.
	VisitAnonymousConfigValues(ctx *AnonymousConfigValuesContext) interface{}

	// Visit a parse tree produced by ArcParser#arguments.
	VisitArguments(ctx *ArgumentsContext) interface{}

	// Visit a parse tree produced by ArcParser#argumentList.
	VisitArgumentList(ctx *ArgumentListContext) interface{}

	// Visit a parse tree produced by ArcParser#block.
	VisitBlock(ctx *BlockContext) interface{}

	// Visit a parse tree produced by ArcParser#statement.
	VisitStatement(ctx *StatementContext) interface{}

	// Visit a parse tree produced by ArcParser#variableDeclaration.
	VisitVariableDeclaration(ctx *VariableDeclarationContext) interface{}

	// Visit a parse tree produced by ArcParser#localVariable.
	VisitLocalVariable(ctx *LocalVariableContext) interface{}

	// Visit a parse tree produced by ArcParser#statefulVariable.
	VisitStatefulVariable(ctx *StatefulVariableContext) interface{}

	// Visit a parse tree produced by ArcParser#assignment.
	VisitAssignment(ctx *AssignmentContext) interface{}

	// Visit a parse tree produced by ArcParser#channelOperation.
	VisitChannelOperation(ctx *ChannelOperationContext) interface{}

	// Visit a parse tree produced by ArcParser#channelWrite.
	VisitChannelWrite(ctx *ChannelWriteContext) interface{}

	// Visit a parse tree produced by ArcParser#channelRead.
	VisitChannelRead(ctx *ChannelReadContext) interface{}

	// Visit a parse tree produced by ArcParser#nonBlockingRead.
	VisitNonBlockingRead(ctx *NonBlockingReadContext) interface{}

	// Visit a parse tree produced by ArcParser#ifStatement.
	VisitIfStatement(ctx *IfStatementContext) interface{}

	// Visit a parse tree produced by ArcParser#elseIfClause.
	VisitElseIfClause(ctx *ElseIfClauseContext) interface{}

	// Visit a parse tree produced by ArcParser#elseClause.
	VisitElseClause(ctx *ElseClauseContext) interface{}

	// Visit a parse tree produced by ArcParser#returnStatement.
	VisitReturnStatement(ctx *ReturnStatementContext) interface{}

	// Visit a parse tree produced by ArcParser#functionCall.
	VisitFunctionCall(ctx *FunctionCallContext) interface{}

	// Visit a parse tree produced by ArcParser#type.
	VisitType(ctx *TypeContext) interface{}

	// Visit a parse tree produced by ArcParser#primitiveType.
	VisitPrimitiveType(ctx *PrimitiveTypeContext) interface{}

	// Visit a parse tree produced by ArcParser#numericType.
	VisitNumericType(ctx *NumericTypeContext) interface{}

	// Visit a parse tree produced by ArcParser#integerType.
	VisitIntegerType(ctx *IntegerTypeContext) interface{}

	// Visit a parse tree produced by ArcParser#floatType.
	VisitFloatType(ctx *FloatTypeContext) interface{}

	// Visit a parse tree produced by ArcParser#temporalType.
	VisitTemporalType(ctx *TemporalTypeContext) interface{}

	// Visit a parse tree produced by ArcParser#channelType.
	VisitChannelType(ctx *ChannelTypeContext) interface{}

	// Visit a parse tree produced by ArcParser#seriesType.
	VisitSeriesType(ctx *SeriesTypeContext) interface{}

	// Visit a parse tree produced by ArcParser#expression.
	VisitExpression(ctx *ExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#logicalOrExpression.
	VisitLogicalOrExpression(ctx *LogicalOrExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#logicalAndExpression.
	VisitLogicalAndExpression(ctx *LogicalAndExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#equalityExpression.
	VisitEqualityExpression(ctx *EqualityExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#relationalExpression.
	VisitRelationalExpression(ctx *RelationalExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#additiveExpression.
	VisitAdditiveExpression(ctx *AdditiveExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#multiplicativeExpression.
	VisitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#powerExpression.
	VisitPowerExpression(ctx *PowerExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#unaryExpression.
	VisitUnaryExpression(ctx *UnaryExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#postfixExpression.
	VisitPostfixExpression(ctx *PostfixExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#indexOrSlice.
	VisitIndexOrSlice(ctx *IndexOrSliceContext) interface{}

	// Visit a parse tree produced by ArcParser#functionCallSuffix.
	VisitFunctionCallSuffix(ctx *FunctionCallSuffixContext) interface{}

	// Visit a parse tree produced by ArcParser#primaryExpression.
	VisitPrimaryExpression(ctx *PrimaryExpressionContext) interface{}

	// Visit a parse tree produced by ArcParser#typeCast.
	VisitTypeCast(ctx *TypeCastContext) interface{}

	// Visit a parse tree produced by ArcParser#builtinFunction.
	VisitBuiltinFunction(ctx *BuiltinFunctionContext) interface{}

	// Visit a parse tree produced by ArcParser#literal.
	VisitLiteral(ctx *LiteralContext) interface{}

	// Visit a parse tree produced by ArcParser#numericLiteral.
	VisitNumericLiteral(ctx *NumericLiteralContext) interface{}

	// Visit a parse tree produced by ArcParser#temporalLiteral.
	VisitTemporalLiteral(ctx *TemporalLiteralContext) interface{}

	// Visit a parse tree produced by ArcParser#seriesLiteral.
	VisitSeriesLiteral(ctx *SeriesLiteralContext) interface{}

	// Visit a parse tree produced by ArcParser#expressionList.
	VisitExpressionList(ctx *ExpressionListContext) interface{}
}
