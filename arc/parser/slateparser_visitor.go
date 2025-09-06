// Code generated from SlateParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // SlateParser
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by SlateParser.
type SlateParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by SlateParser#program.
	VisitProgram(ctx *ProgramContext) interface{}

	// Visit a parse tree produced by SlateParser#topLevelItem.
	VisitTopLevelItem(ctx *TopLevelItemContext) interface{}

	// Visit a parse tree produced by SlateParser#functionDeclaration.
	VisitFunctionDeclaration(ctx *FunctionDeclarationContext) interface{}

	// Visit a parse tree produced by SlateParser#parameterList.
	VisitParameterList(ctx *ParameterListContext) interface{}

	// Visit a parse tree produced by SlateParser#parameter.
	VisitParameter(ctx *ParameterContext) interface{}

	// Visit a parse tree produced by SlateParser#returnType.
	VisitReturnType(ctx *ReturnTypeContext) interface{}

	// Visit a parse tree produced by SlateParser#taskDeclaration.
	VisitTaskDeclaration(ctx *TaskDeclarationContext) interface{}

	// Visit a parse tree produced by SlateParser#configBlock.
	VisitConfigBlock(ctx *ConfigBlockContext) interface{}

	// Visit a parse tree produced by SlateParser#configParameter.
	VisitConfigParameter(ctx *ConfigParameterContext) interface{}

	// Visit a parse tree produced by SlateParser#flowStatement.
	VisitFlowStatement(ctx *FlowStatementContext) interface{}

	// Visit a parse tree produced by SlateParser#flowNode.
	VisitFlowNode(ctx *FlowNodeContext) interface{}

	// Visit a parse tree produced by SlateParser#channelIdentifier.
	VisitChannelIdentifier(ctx *ChannelIdentifierContext) interface{}

	// Visit a parse tree produced by SlateParser#taskInvocation.
	VisitTaskInvocation(ctx *TaskInvocationContext) interface{}

	// Visit a parse tree produced by SlateParser#configValues.
	VisitConfigValues(ctx *ConfigValuesContext) interface{}

	// Visit a parse tree produced by SlateParser#namedConfigValues.
	VisitNamedConfigValues(ctx *NamedConfigValuesContext) interface{}

	// Visit a parse tree produced by SlateParser#namedConfigValue.
	VisitNamedConfigValue(ctx *NamedConfigValueContext) interface{}

	// Visit a parse tree produced by SlateParser#anonymousConfigValues.
	VisitAnonymousConfigValues(ctx *AnonymousConfigValuesContext) interface{}

	// Visit a parse tree produced by SlateParser#arguments.
	VisitArguments(ctx *ArgumentsContext) interface{}

	// Visit a parse tree produced by SlateParser#argumentList.
	VisitArgumentList(ctx *ArgumentListContext) interface{}

	// Visit a parse tree produced by SlateParser#block.
	VisitBlock(ctx *BlockContext) interface{}

	// Visit a parse tree produced by SlateParser#statement.
	VisitStatement(ctx *StatementContext) interface{}

	// Visit a parse tree produced by SlateParser#variableDeclaration.
	VisitVariableDeclaration(ctx *VariableDeclarationContext) interface{}

	// Visit a parse tree produced by SlateParser#localVariable.
	VisitLocalVariable(ctx *LocalVariableContext) interface{}

	// Visit a parse tree produced by SlateParser#statefulVariable.
	VisitStatefulVariable(ctx *StatefulVariableContext) interface{}

	// Visit a parse tree produced by SlateParser#assignment.
	VisitAssignment(ctx *AssignmentContext) interface{}

	// Visit a parse tree produced by SlateParser#channelOperation.
	VisitChannelOperation(ctx *ChannelOperationContext) interface{}

	// Visit a parse tree produced by SlateParser#channelWrite.
	VisitChannelWrite(ctx *ChannelWriteContext) interface{}

	// Visit a parse tree produced by SlateParser#channelRead.
	VisitChannelRead(ctx *ChannelReadContext) interface{}

	// Visit a parse tree produced by SlateParser#blockingRead.
	VisitBlockingRead(ctx *BlockingReadContext) interface{}

	// Visit a parse tree produced by SlateParser#nonBlockingRead.
	VisitNonBlockingRead(ctx *NonBlockingReadContext) interface{}

	// Visit a parse tree produced by SlateParser#ifStatement.
	VisitIfStatement(ctx *IfStatementContext) interface{}

	// Visit a parse tree produced by SlateParser#elseIfClause.
	VisitElseIfClause(ctx *ElseIfClauseContext) interface{}

	// Visit a parse tree produced by SlateParser#elseClause.
	VisitElseClause(ctx *ElseClauseContext) interface{}

	// Visit a parse tree produced by SlateParser#returnStatement.
	VisitReturnStatement(ctx *ReturnStatementContext) interface{}

	// Visit a parse tree produced by SlateParser#functionCall.
	VisitFunctionCall(ctx *FunctionCallContext) interface{}

	// Visit a parse tree produced by SlateParser#type.
	VisitType(ctx *TypeContext) interface{}

	// Visit a parse tree produced by SlateParser#primitiveType.
	VisitPrimitiveType(ctx *PrimitiveTypeContext) interface{}

	// Visit a parse tree produced by SlateParser#numericType.
	VisitNumericType(ctx *NumericTypeContext) interface{}

	// Visit a parse tree produced by SlateParser#integerType.
	VisitIntegerType(ctx *IntegerTypeContext) interface{}

	// Visit a parse tree produced by SlateParser#floatType.
	VisitFloatType(ctx *FloatTypeContext) interface{}

	// Visit a parse tree produced by SlateParser#temporalType.
	VisitTemporalType(ctx *TemporalTypeContext) interface{}

	// Visit a parse tree produced by SlateParser#channelType.
	VisitChannelType(ctx *ChannelTypeContext) interface{}

	// Visit a parse tree produced by SlateParser#seriesType.
	VisitSeriesType(ctx *SeriesTypeContext) interface{}

	// Visit a parse tree produced by SlateParser#expression.
	VisitExpression(ctx *ExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#logicalOrExpression.
	VisitLogicalOrExpression(ctx *LogicalOrExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#logicalAndExpression.
	VisitLogicalAndExpression(ctx *LogicalAndExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#equalityExpression.
	VisitEqualityExpression(ctx *EqualityExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#relationalExpression.
	VisitRelationalExpression(ctx *RelationalExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#additiveExpression.
	VisitAdditiveExpression(ctx *AdditiveExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#multiplicativeExpression.
	VisitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#powerExpression.
	VisitPowerExpression(ctx *PowerExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#unaryExpression.
	VisitUnaryExpression(ctx *UnaryExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#blockingReadExpr.
	VisitBlockingReadExpr(ctx *BlockingReadExprContext) interface{}

	// Visit a parse tree produced by SlateParser#postfixExpression.
	VisitPostfixExpression(ctx *PostfixExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#indexOrSlice.
	VisitIndexOrSlice(ctx *IndexOrSliceContext) interface{}

	// Visit a parse tree produced by SlateParser#functionCallSuffix.
	VisitFunctionCallSuffix(ctx *FunctionCallSuffixContext) interface{}

	// Visit a parse tree produced by SlateParser#primaryExpression.
	VisitPrimaryExpression(ctx *PrimaryExpressionContext) interface{}

	// Visit a parse tree produced by SlateParser#typeCast.
	VisitTypeCast(ctx *TypeCastContext) interface{}

	// Visit a parse tree produced by SlateParser#builtinFunction.
	VisitBuiltinFunction(ctx *BuiltinFunctionContext) interface{}

	// Visit a parse tree produced by SlateParser#literal.
	VisitLiteral(ctx *LiteralContext) interface{}

	// Visit a parse tree produced by SlateParser#numericLiteral.
	VisitNumericLiteral(ctx *NumericLiteralContext) interface{}

	// Visit a parse tree produced by SlateParser#temporalLiteral.
	VisitTemporalLiteral(ctx *TemporalLiteralContext) interface{}

	// Visit a parse tree produced by SlateParser#seriesLiteral.
	VisitSeriesLiteral(ctx *SeriesLiteralContext) interface{}

	// Visit a parse tree produced by SlateParser#expressionList.
	VisitExpressionList(ctx *ExpressionListContext) interface{}
}
