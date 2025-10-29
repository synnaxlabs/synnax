// Code generated from ArcParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // ArcParser
import "github.com/antlr4-go/antlr/v4"

// ArcParserListener is a complete listener for a parse tree produced by ArcParser.
type ArcParserListener interface {
	antlr.ParseTreeListener

	// EnterProgram is called when entering the program production.
	EnterProgram(c *ProgramContext)

	// EnterTopLevelItem is called when entering the topLevelItem production.
	EnterTopLevelItem(c *TopLevelItemContext)

	// EnterFunctionDeclaration is called when entering the functionDeclaration production.
	EnterFunctionDeclaration(c *FunctionDeclarationContext)

	// EnterInputList is called when entering the inputList production.
	EnterInputList(c *InputListContext)

	// EnterInput is called when entering the input production.
	EnterInput(c *InputContext)

	// EnterOutputType is called when entering the outputType production.
	EnterOutputType(c *OutputTypeContext)

	// EnterMultiOutputBlock is called when entering the multiOutputBlock production.
	EnterMultiOutputBlock(c *MultiOutputBlockContext)

	// EnterNamedOutput is called when entering the namedOutput production.
	EnterNamedOutput(c *NamedOutputContext)

	// EnterConfigBlock is called when entering the configBlock production.
	EnterConfigBlock(c *ConfigBlockContext)

	// EnterConfig is called when entering the config production.
	EnterConfig(c *ConfigContext)

	// EnterFlowStatement is called when entering the flowStatement production.
	EnterFlowStatement(c *FlowStatementContext)

	// EnterRoutingTable is called when entering the routingTable production.
	EnterRoutingTable(c *RoutingTableContext)

	// EnterRoutingEntry is called when entering the routingEntry production.
	EnterRoutingEntry(c *RoutingEntryContext)

	// EnterFlowNode is called when entering the flowNode production.
	EnterFlowNode(c *FlowNodeContext)

	// EnterChannelIdentifier is called when entering the channelIdentifier production.
	EnterChannelIdentifier(c *ChannelIdentifierContext)

	// EnterFunction is called when entering the function production.
	EnterFunction(c *FunctionContext)

	// EnterConfigValues is called when entering the configValues production.
	EnterConfigValues(c *ConfigValuesContext)

	// EnterNamedConfigValues is called when entering the namedConfigValues production.
	EnterNamedConfigValues(c *NamedConfigValuesContext)

	// EnterNamedConfigValue is called when entering the namedConfigValue production.
	EnterNamedConfigValue(c *NamedConfigValueContext)

	// EnterAnonymousConfigValues is called when entering the anonymousConfigValues production.
	EnterAnonymousConfigValues(c *AnonymousConfigValuesContext)

	// EnterArguments is called when entering the arguments production.
	EnterArguments(c *ArgumentsContext)

	// EnterArgumentList is called when entering the argumentList production.
	EnterArgumentList(c *ArgumentListContext)

	// EnterBlock is called when entering the block production.
	EnterBlock(c *BlockContext)

	// EnterStatement is called when entering the statement production.
	EnterStatement(c *StatementContext)

	// EnterVariableDeclaration is called when entering the variableDeclaration production.
	EnterVariableDeclaration(c *VariableDeclarationContext)

	// EnterLocalVariable is called when entering the localVariable production.
	EnterLocalVariable(c *LocalVariableContext)

	// EnterStatefulVariable is called when entering the statefulVariable production.
	EnterStatefulVariable(c *StatefulVariableContext)

	// EnterAssignment is called when entering the assignment production.
	EnterAssignment(c *AssignmentContext)

	// EnterChannelOperation is called when entering the channelOperation production.
	EnterChannelOperation(c *ChannelOperationContext)

	// EnterChannelWrite is called when entering the channelWrite production.
	EnterChannelWrite(c *ChannelWriteContext)

	// EnterChannelRead is called when entering the channelRead production.
	EnterChannelRead(c *ChannelReadContext)

	// EnterBlockingRead is called when entering the blockingRead production.
	EnterBlockingRead(c *BlockingReadContext)

	// EnterNonBlockingRead is called when entering the nonBlockingRead production.
	EnterNonBlockingRead(c *NonBlockingReadContext)

	// EnterIfStatement is called when entering the ifStatement production.
	EnterIfStatement(c *IfStatementContext)

	// EnterElseIfClause is called when entering the elseIfClause production.
	EnterElseIfClause(c *ElseIfClauseContext)

	// EnterElseClause is called when entering the elseClause production.
	EnterElseClause(c *ElseClauseContext)

	// EnterReturnStatement is called when entering the returnStatement production.
	EnterReturnStatement(c *ReturnStatementContext)

	// EnterFunctionCall is called when entering the functionCall production.
	EnterFunctionCall(c *FunctionCallContext)

	// EnterType is called when entering the type production.
	EnterType(c *TypeContext)

	// EnterPrimitiveType is called when entering the primitiveType production.
	EnterPrimitiveType(c *PrimitiveTypeContext)

	// EnterNumericType is called when entering the numericType production.
	EnterNumericType(c *NumericTypeContext)

	// EnterIntegerType is called when entering the integerType production.
	EnterIntegerType(c *IntegerTypeContext)

	// EnterFloatType is called when entering the floatType production.
	EnterFloatType(c *FloatTypeContext)

	// EnterTemporalType is called when entering the temporalType production.
	EnterTemporalType(c *TemporalTypeContext)

	// EnterChannelType is called when entering the channelType production.
	EnterChannelType(c *ChannelTypeContext)

	// EnterSeriesType is called when entering the seriesType production.
	EnterSeriesType(c *SeriesTypeContext)

	// EnterExpression is called when entering the expression production.
	EnterExpression(c *ExpressionContext)

	// EnterLogicalOrExpression is called when entering the logicalOrExpression production.
	EnterLogicalOrExpression(c *LogicalOrExpressionContext)

	// EnterLogicalAndExpression is called when entering the logicalAndExpression production.
	EnterLogicalAndExpression(c *LogicalAndExpressionContext)

	// EnterEqualityExpression is called when entering the equalityExpression production.
	EnterEqualityExpression(c *EqualityExpressionContext)

	// EnterRelationalExpression is called when entering the relationalExpression production.
	EnterRelationalExpression(c *RelationalExpressionContext)

	// EnterAdditiveExpression is called when entering the additiveExpression production.
	EnterAdditiveExpression(c *AdditiveExpressionContext)

	// EnterMultiplicativeExpression is called when entering the multiplicativeExpression production.
	EnterMultiplicativeExpression(c *MultiplicativeExpressionContext)

	// EnterPowerExpression is called when entering the powerExpression production.
	EnterPowerExpression(c *PowerExpressionContext)

	// EnterUnaryExpression is called when entering the unaryExpression production.
	EnterUnaryExpression(c *UnaryExpressionContext)

	// EnterBlockingReadExpr is called when entering the blockingReadExpr production.
	EnterBlockingReadExpr(c *BlockingReadExprContext)

	// EnterPostfixExpression is called when entering the postfixExpression production.
	EnterPostfixExpression(c *PostfixExpressionContext)

	// EnterIndexOrSlice is called when entering the indexOrSlice production.
	EnterIndexOrSlice(c *IndexOrSliceContext)

	// EnterFunctionCallSuffix is called when entering the functionCallSuffix production.
	EnterFunctionCallSuffix(c *FunctionCallSuffixContext)

	// EnterPrimaryExpression is called when entering the primaryExpression production.
	EnterPrimaryExpression(c *PrimaryExpressionContext)

	// EnterTypeCast is called when entering the typeCast production.
	EnterTypeCast(c *TypeCastContext)

	// EnterBuiltinFunction is called when entering the builtinFunction production.
	EnterBuiltinFunction(c *BuiltinFunctionContext)

	// EnterLiteral is called when entering the literal production.
	EnterLiteral(c *LiteralContext)

	// EnterNumericLiteral is called when entering the numericLiteral production.
	EnterNumericLiteral(c *NumericLiteralContext)

	// EnterTemporalLiteral is called when entering the temporalLiteral production.
	EnterTemporalLiteral(c *TemporalLiteralContext)

	// EnterSeriesLiteral is called when entering the seriesLiteral production.
	EnterSeriesLiteral(c *SeriesLiteralContext)

	// EnterExpressionList is called when entering the expressionList production.
	EnterExpressionList(c *ExpressionListContext)

	// ExitProgram is called when exiting the program production.
	ExitProgram(c *ProgramContext)

	// ExitTopLevelItem is called when exiting the topLevelItem production.
	ExitTopLevelItem(c *TopLevelItemContext)

	// ExitFunctionDeclaration is called when exiting the functionDeclaration production.
	ExitFunctionDeclaration(c *FunctionDeclarationContext)

	// ExitInputList is called when exiting the inputList production.
	ExitInputList(c *InputListContext)

	// ExitInput is called when exiting the input production.
	ExitInput(c *InputContext)

	// ExitOutputType is called when exiting the outputType production.
	ExitOutputType(c *OutputTypeContext)

	// ExitMultiOutputBlock is called when exiting the multiOutputBlock production.
	ExitMultiOutputBlock(c *MultiOutputBlockContext)

	// ExitNamedOutput is called when exiting the namedOutput production.
	ExitNamedOutput(c *NamedOutputContext)

	// ExitConfigBlock is called when exiting the configBlock production.
	ExitConfigBlock(c *ConfigBlockContext)

	// ExitConfig is called when exiting the config production.
	ExitConfig(c *ConfigContext)

	// ExitFlowStatement is called when exiting the flowStatement production.
	ExitFlowStatement(c *FlowStatementContext)

	// ExitRoutingTable is called when exiting the routingTable production.
	ExitRoutingTable(c *RoutingTableContext)

	// ExitRoutingEntry is called when exiting the routingEntry production.
	ExitRoutingEntry(c *RoutingEntryContext)

	// ExitFlowNode is called when exiting the flowNode production.
	ExitFlowNode(c *FlowNodeContext)

	// ExitChannelIdentifier is called when exiting the channelIdentifier production.
	ExitChannelIdentifier(c *ChannelIdentifierContext)

	// ExitFunction is called when exiting the function production.
	ExitFunction(c *FunctionContext)

	// ExitConfigValues is called when exiting the configValues production.
	ExitConfigValues(c *ConfigValuesContext)

	// ExitNamedConfigValues is called when exiting the namedConfigValues production.
	ExitNamedConfigValues(c *NamedConfigValuesContext)

	// ExitNamedConfigValue is called when exiting the namedConfigValue production.
	ExitNamedConfigValue(c *NamedConfigValueContext)

	// ExitAnonymousConfigValues is called when exiting the anonymousConfigValues production.
	ExitAnonymousConfigValues(c *AnonymousConfigValuesContext)

	// ExitArguments is called when exiting the arguments production.
	ExitArguments(c *ArgumentsContext)

	// ExitArgumentList is called when exiting the argumentList production.
	ExitArgumentList(c *ArgumentListContext)

	// ExitBlock is called when exiting the block production.
	ExitBlock(c *BlockContext)

	// ExitStatement is called when exiting the statement production.
	ExitStatement(c *StatementContext)

	// ExitVariableDeclaration is called when exiting the variableDeclaration production.
	ExitVariableDeclaration(c *VariableDeclarationContext)

	// ExitLocalVariable is called when exiting the localVariable production.
	ExitLocalVariable(c *LocalVariableContext)

	// ExitStatefulVariable is called when exiting the statefulVariable production.
	ExitStatefulVariable(c *StatefulVariableContext)

	// ExitAssignment is called when exiting the assignment production.
	ExitAssignment(c *AssignmentContext)

	// ExitChannelOperation is called when exiting the channelOperation production.
	ExitChannelOperation(c *ChannelOperationContext)

	// ExitChannelWrite is called when exiting the channelWrite production.
	ExitChannelWrite(c *ChannelWriteContext)

	// ExitChannelRead is called when exiting the channelRead production.
	ExitChannelRead(c *ChannelReadContext)

	// ExitBlockingRead is called when exiting the blockingRead production.
	ExitBlockingRead(c *BlockingReadContext)

	// ExitNonBlockingRead is called when exiting the nonBlockingRead production.
	ExitNonBlockingRead(c *NonBlockingReadContext)

	// ExitIfStatement is called when exiting the ifStatement production.
	ExitIfStatement(c *IfStatementContext)

	// ExitElseIfClause is called when exiting the elseIfClause production.
	ExitElseIfClause(c *ElseIfClauseContext)

	// ExitElseClause is called when exiting the elseClause production.
	ExitElseClause(c *ElseClauseContext)

	// ExitReturnStatement is called when exiting the returnStatement production.
	ExitReturnStatement(c *ReturnStatementContext)

	// ExitFunctionCall is called when exiting the functionCall production.
	ExitFunctionCall(c *FunctionCallContext)

	// ExitType is called when exiting the type production.
	ExitType(c *TypeContext)

	// ExitPrimitiveType is called when exiting the primitiveType production.
	ExitPrimitiveType(c *PrimitiveTypeContext)

	// ExitNumericType is called when exiting the numericType production.
	ExitNumericType(c *NumericTypeContext)

	// ExitIntegerType is called when exiting the integerType production.
	ExitIntegerType(c *IntegerTypeContext)

	// ExitFloatType is called when exiting the floatType production.
	ExitFloatType(c *FloatTypeContext)

	// ExitTemporalType is called when exiting the temporalType production.
	ExitTemporalType(c *TemporalTypeContext)

	// ExitChannelType is called when exiting the channelType production.
	ExitChannelType(c *ChannelTypeContext)

	// ExitSeriesType is called when exiting the seriesType production.
	ExitSeriesType(c *SeriesTypeContext)

	// ExitExpression is called when exiting the expression production.
	ExitExpression(c *ExpressionContext)

	// ExitLogicalOrExpression is called when exiting the logicalOrExpression production.
	ExitLogicalOrExpression(c *LogicalOrExpressionContext)

	// ExitLogicalAndExpression is called when exiting the logicalAndExpression production.
	ExitLogicalAndExpression(c *LogicalAndExpressionContext)

	// ExitEqualityExpression is called when exiting the equalityExpression production.
	ExitEqualityExpression(c *EqualityExpressionContext)

	// ExitRelationalExpression is called when exiting the relationalExpression production.
	ExitRelationalExpression(c *RelationalExpressionContext)

	// ExitAdditiveExpression is called when exiting the additiveExpression production.
	ExitAdditiveExpression(c *AdditiveExpressionContext)

	// ExitMultiplicativeExpression is called when exiting the multiplicativeExpression production.
	ExitMultiplicativeExpression(c *MultiplicativeExpressionContext)

	// ExitPowerExpression is called when exiting the powerExpression production.
	ExitPowerExpression(c *PowerExpressionContext)

	// ExitUnaryExpression is called when exiting the unaryExpression production.
	ExitUnaryExpression(c *UnaryExpressionContext)

	// ExitBlockingReadExpr is called when exiting the blockingReadExpr production.
	ExitBlockingReadExpr(c *BlockingReadExprContext)

	// ExitPostfixExpression is called when exiting the postfixExpression production.
	ExitPostfixExpression(c *PostfixExpressionContext)

	// ExitIndexOrSlice is called when exiting the indexOrSlice production.
	ExitIndexOrSlice(c *IndexOrSliceContext)

	// ExitFunctionCallSuffix is called when exiting the functionCallSuffix production.
	ExitFunctionCallSuffix(c *FunctionCallSuffixContext)

	// ExitPrimaryExpression is called when exiting the primaryExpression production.
	ExitPrimaryExpression(c *PrimaryExpressionContext)

	// ExitTypeCast is called when exiting the typeCast production.
	ExitTypeCast(c *TypeCastContext)

	// ExitBuiltinFunction is called when exiting the builtinFunction production.
	ExitBuiltinFunction(c *BuiltinFunctionContext)

	// ExitLiteral is called when exiting the literal production.
	ExitLiteral(c *LiteralContext)

	// ExitNumericLiteral is called when exiting the numericLiteral production.
	ExitNumericLiteral(c *NumericLiteralContext)

	// ExitTemporalLiteral is called when exiting the temporalLiteral production.
	ExitTemporalLiteral(c *TemporalLiteralContext)

	// ExitSeriesLiteral is called when exiting the seriesLiteral production.
	ExitSeriesLiteral(c *SeriesLiteralContext)

	// ExitExpressionList is called when exiting the expressionList production.
	ExitExpressionList(c *ExpressionListContext)
}
