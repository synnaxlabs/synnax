// Copyright 2026 Synnax Labs, Inc.
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

	// EnterConfigList is called when entering the configList production.
	EnterConfigList(c *ConfigListContext)

	// EnterConfig is called when entering the config production.
	EnterConfig(c *ConfigContext)

	// EnterSequenceDeclaration is called when entering the sequenceDeclaration production.
	EnterSequenceDeclaration(c *SequenceDeclarationContext)

	// EnterStageDeclaration is called when entering the stageDeclaration production.
	EnterStageDeclaration(c *StageDeclarationContext)

	// EnterStageBody is called when entering the stageBody production.
	EnterStageBody(c *StageBodyContext)

	// EnterStageItem is called when entering the stageItem production.
	EnterStageItem(c *StageItemContext)

	// EnterSingleInvocation is called when entering the singleInvocation production.
	EnterSingleInvocation(c *SingleInvocationContext)

	// EnterFlowStatement is called when entering the flowStatement production.
	EnterFlowStatement(c *FlowStatementContext)

	// EnterFlowOperator is called when entering the flowOperator production.
	EnterFlowOperator(c *FlowOperatorContext)

	// EnterRoutingTable is called when entering the routingTable production.
	EnterRoutingTable(c *RoutingTableContext)

	// EnterRoutingEntry is called when entering the routingEntry production.
	EnterRoutingEntry(c *RoutingEntryContext)

	// EnterFlowNode is called when entering the flowNode production.
	EnterFlowNode(c *FlowNodeContext)

	// EnterIdentifier is called when entering the identifier production.
	EnterIdentifier(c *IdentifierContext)

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

	// EnterCompoundOp is called when entering the compoundOp production.
	EnterCompoundOp(c *CompoundOpContext)

	// EnterIfStatement is called when entering the ifStatement production.
	EnterIfStatement(c *IfStatementContext)

	// EnterElseIfClause is called when entering the elseIfClause production.
	EnterElseIfClause(c *ElseIfClauseContext)

	// EnterElseClause is called when entering the elseClause production.
	EnterElseClause(c *ElseClauseContext)

	// EnterReturnStatement is called when entering the returnStatement production.
	EnterReturnStatement(c *ReturnStatementContext)

	// EnterType is called when entering the type production.
	EnterType(c *TypeContext)

	// EnterUnitSuffix is called when entering the unitSuffix production.
	EnterUnitSuffix(c *UnitSuffixContext)

	// EnterPrimitiveType is called when entering the primitiveType production.
	EnterPrimitiveType(c *PrimitiveTypeContext)

	// EnterNumericType is called when entering the numericType production.
	EnterNumericType(c *NumericTypeContext)

	// EnterIntegerType is called when entering the integerType production.
	EnterIntegerType(c *IntegerTypeContext)

	// EnterFloatType is called when entering the floatType production.
	EnterFloatType(c *FloatTypeContext)

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

	// EnterLiteral is called when entering the literal production.
	EnterLiteral(c *LiteralContext)

	// EnterNumericLiteral is called when entering the numericLiteral production.
	EnterNumericLiteral(c *NumericLiteralContext)

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

	// ExitConfigList is called when exiting the configList production.
	ExitConfigList(c *ConfigListContext)

	// ExitConfig is called when exiting the config production.
	ExitConfig(c *ConfigContext)

	// ExitSequenceDeclaration is called when exiting the sequenceDeclaration production.
	ExitSequenceDeclaration(c *SequenceDeclarationContext)

	// ExitStageDeclaration is called when exiting the stageDeclaration production.
	ExitStageDeclaration(c *StageDeclarationContext)

	// ExitStageBody is called when exiting the stageBody production.
	ExitStageBody(c *StageBodyContext)

	// ExitStageItem is called when exiting the stageItem production.
	ExitStageItem(c *StageItemContext)

	// ExitSingleInvocation is called when exiting the singleInvocation production.
	ExitSingleInvocation(c *SingleInvocationContext)

	// ExitFlowStatement is called when exiting the flowStatement production.
	ExitFlowStatement(c *FlowStatementContext)

	// ExitFlowOperator is called when exiting the flowOperator production.
	ExitFlowOperator(c *FlowOperatorContext)

	// ExitRoutingTable is called when exiting the routingTable production.
	ExitRoutingTable(c *RoutingTableContext)

	// ExitRoutingEntry is called when exiting the routingEntry production.
	ExitRoutingEntry(c *RoutingEntryContext)

	// ExitFlowNode is called when exiting the flowNode production.
	ExitFlowNode(c *FlowNodeContext)

	// ExitIdentifier is called when exiting the identifier production.
	ExitIdentifier(c *IdentifierContext)

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

	// ExitCompoundOp is called when exiting the compoundOp production.
	ExitCompoundOp(c *CompoundOpContext)

	// ExitIfStatement is called when exiting the ifStatement production.
	ExitIfStatement(c *IfStatementContext)

	// ExitElseIfClause is called when exiting the elseIfClause production.
	ExitElseIfClause(c *ElseIfClauseContext)

	// ExitElseClause is called when exiting the elseClause production.
	ExitElseClause(c *ElseClauseContext)

	// ExitReturnStatement is called when exiting the returnStatement production.
	ExitReturnStatement(c *ReturnStatementContext)

	// ExitType is called when exiting the type production.
	ExitType(c *TypeContext)

	// ExitUnitSuffix is called when exiting the unitSuffix production.
	ExitUnitSuffix(c *UnitSuffixContext)

	// ExitPrimitiveType is called when exiting the primitiveType production.
	ExitPrimitiveType(c *PrimitiveTypeContext)

	// ExitNumericType is called when exiting the numericType production.
	ExitNumericType(c *NumericTypeContext)

	// ExitIntegerType is called when exiting the integerType production.
	ExitIntegerType(c *IntegerTypeContext)

	// ExitFloatType is called when exiting the floatType production.
	ExitFloatType(c *FloatTypeContext)

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

	// ExitLiteral is called when exiting the literal production.
	ExitLiteral(c *LiteralContext)

	// ExitNumericLiteral is called when exiting the numericLiteral production.
	ExitNumericLiteral(c *NumericLiteralContext)

	// ExitSeriesLiteral is called when exiting the seriesLiteral production.
	ExitSeriesLiteral(c *SeriesLiteralContext)

	// ExitExpressionList is called when exiting the expressionList production.
	ExitExpressionList(c *ExpressionListContext)
}
