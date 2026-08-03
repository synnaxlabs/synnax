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

	// EnterImportStatement is called when entering the importStatement production.
	EnterImportStatement(c *ImportStatementContext)

	// EnterImportItem is called when entering the importItem production.
	EnterImportItem(c *ImportItemContext)

	// EnterImportPath is called when entering the importPath production.
	EnterImportPath(c *ImportPathContext)

	// EnterImportPathHead is called when entering the importPathHead production.
	EnterImportPathHead(c *ImportPathHeadContext)

	// EnterAuthorityBlock is called when entering the authorityBlock production.
	EnterAuthorityBlock(c *AuthorityBlockContext)

	// EnterAuthorityEntry is called when entering the authorityEntry production.
	EnterAuthorityEntry(c *AuthorityEntryContext)

	// EnterFunctionDeclaration is called when entering the functionDeclaration production.
	EnterFunctionDeclaration(c *FunctionDeclarationContext)

	// EnterTriggerList is called when entering the triggerList production.
	EnterTriggerList(c *TriggerListContext)

	// EnterTrigger is called when entering the trigger production.
	EnterTrigger(c *TriggerContext)

	// EnterOutputType is called when entering the outputType production.
	EnterOutputType(c *OutputTypeContext)

	// EnterMultiOutputBlock is called when entering the multiOutputBlock production.
	EnterMultiOutputBlock(c *MultiOutputBlockContext)

	// EnterNamedOutput is called when entering the namedOutput production.
	EnterNamedOutput(c *NamedOutputContext)

	// EnterInputBlock is called when entering the inputBlock production.
	EnterInputBlock(c *InputBlockContext)

	// EnterInputList is called when entering the inputList production.
	EnterInputList(c *InputListContext)

	// EnterInput is called when entering the input production.
	EnterInput(c *InputContext)

	// EnterSequenceDeclaration is called when entering the sequenceDeclaration production.
	EnterSequenceDeclaration(c *SequenceDeclarationContext)

	// EnterSequenceItem is called when entering the sequenceItem production.
	EnterSequenceItem(c *SequenceItemContext)

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

	// EnterQualifiedIdentifier is called when entering the qualifiedIdentifier production.
	EnterQualifiedIdentifier(c *QualifiedIdentifierContext)

	// EnterInputValues is called when entering the inputValues production.
	EnterInputValues(c *InputValuesContext)

	// EnterNamedInputValues is called when entering the namedInputValues production.
	EnterNamedInputValues(c *NamedInputValuesContext)

	// EnterNamedInputValue is called when entering the namedInputValue production.
	EnterNamedInputValue(c *NamedInputValueContext)

	// EnterAnonymousInputValues is called when entering the anonymousInputValues production.
	EnterAnonymousInputValues(c *AnonymousInputValuesContext)

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

	// EnterForStatement is called when entering the forStatement production.
	EnterForStatement(c *ForStatementContext)

	// EnterForClause is called when entering the forClause production.
	EnterForClause(c *ForClauseContext)

	// EnterBreakStatement is called when entering the breakStatement production.
	EnterBreakStatement(c *BreakStatementContext)

	// EnterContinueStatement is called when entering the continueStatement production.
	EnterContinueStatement(c *ContinueStatementContext)

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

	// ExitImportStatement is called when exiting the importStatement production.
	ExitImportStatement(c *ImportStatementContext)

	// ExitImportItem is called when exiting the importItem production.
	ExitImportItem(c *ImportItemContext)

	// ExitImportPath is called when exiting the importPath production.
	ExitImportPath(c *ImportPathContext)

	// ExitImportPathHead is called when exiting the importPathHead production.
	ExitImportPathHead(c *ImportPathHeadContext)

	// ExitAuthorityBlock is called when exiting the authorityBlock production.
	ExitAuthorityBlock(c *AuthorityBlockContext)

	// ExitAuthorityEntry is called when exiting the authorityEntry production.
	ExitAuthorityEntry(c *AuthorityEntryContext)

	// ExitFunctionDeclaration is called when exiting the functionDeclaration production.
	ExitFunctionDeclaration(c *FunctionDeclarationContext)

	// ExitTriggerList is called when exiting the triggerList production.
	ExitTriggerList(c *TriggerListContext)

	// ExitTrigger is called when exiting the trigger production.
	ExitTrigger(c *TriggerContext)

	// ExitOutputType is called when exiting the outputType production.
	ExitOutputType(c *OutputTypeContext)

	// ExitMultiOutputBlock is called when exiting the multiOutputBlock production.
	ExitMultiOutputBlock(c *MultiOutputBlockContext)

	// ExitNamedOutput is called when exiting the namedOutput production.
	ExitNamedOutput(c *NamedOutputContext)

	// ExitInputBlock is called when exiting the inputBlock production.
	ExitInputBlock(c *InputBlockContext)

	// ExitInputList is called when exiting the inputList production.
	ExitInputList(c *InputListContext)

	// ExitInput is called when exiting the input production.
	ExitInput(c *InputContext)

	// ExitSequenceDeclaration is called when exiting the sequenceDeclaration production.
	ExitSequenceDeclaration(c *SequenceDeclarationContext)

	// ExitSequenceItem is called when exiting the sequenceItem production.
	ExitSequenceItem(c *SequenceItemContext)

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

	// ExitQualifiedIdentifier is called when exiting the qualifiedIdentifier production.
	ExitQualifiedIdentifier(c *QualifiedIdentifierContext)

	// ExitInputValues is called when exiting the inputValues production.
	ExitInputValues(c *InputValuesContext)

	// ExitNamedInputValues is called when exiting the namedInputValues production.
	ExitNamedInputValues(c *NamedInputValuesContext)

	// ExitNamedInputValue is called when exiting the namedInputValue production.
	ExitNamedInputValue(c *NamedInputValueContext)

	// ExitAnonymousInputValues is called when exiting the anonymousInputValues production.
	ExitAnonymousInputValues(c *AnonymousInputValuesContext)

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

	// ExitForStatement is called when exiting the forStatement production.
	ExitForStatement(c *ForStatementContext)

	// ExitForClause is called when exiting the forClause production.
	ExitForClause(c *ForClauseContext)

	// ExitBreakStatement is called when exiting the breakStatement production.
	ExitBreakStatement(c *BreakStatementContext)

	// ExitContinueStatement is called when exiting the continueStatement production.
	ExitContinueStatement(c *ContinueStatementContext)

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
