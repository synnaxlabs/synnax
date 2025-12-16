// Code generated from ArcParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // ArcParser
import "github.com/antlr4-go/antlr/v4"

type BaseArcParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseArcParserVisitor) VisitProgram(ctx *ProgramContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitTopLevelItem(ctx *TopLevelItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitFunctionDeclaration(ctx *FunctionDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitInputList(ctx *InputListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitInput(ctx *InputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitOutputType(ctx *OutputTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitMultiOutputBlock(ctx *MultiOutputBlockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitNamedOutput(ctx *NamedOutputContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitConfigBlock(ctx *ConfigBlockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitConfig(ctx *ConfigContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitSequenceDeclaration(ctx *SequenceDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitStageDeclaration(ctx *StageDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitStageBody(ctx *StageBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitStageItem(ctx *StageItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitFlowStatement(ctx *FlowStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitFlowOperator(ctx *FlowOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitRoutingTable(ctx *RoutingTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitRoutingEntry(ctx *RoutingEntryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitFlowNode(ctx *FlowNodeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitIdentifier(ctx *IdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitFunction(ctx *FunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitConfigValues(ctx *ConfigValuesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitNamedConfigValues(ctx *NamedConfigValuesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitNamedConfigValue(ctx *NamedConfigValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitAnonymousConfigValues(ctx *AnonymousConfigValuesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitArguments(ctx *ArgumentsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitArgumentList(ctx *ArgumentListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitBlock(ctx *BlockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitStatement(ctx *StatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitVariableDeclaration(ctx *VariableDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitLocalVariable(ctx *LocalVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitStatefulVariable(ctx *StatefulVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitAssignment(ctx *AssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitChannelOperation(ctx *ChannelOperationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitChannelWrite(ctx *ChannelWriteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitChannelRead(ctx *ChannelReadContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitNonBlockingRead(ctx *NonBlockingReadContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitIfStatement(ctx *IfStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitElseIfClause(ctx *ElseIfClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitElseClause(ctx *ElseClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitReturnStatement(ctx *ReturnStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitFunctionCall(ctx *FunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitType(ctx *TypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitPrimitiveType(ctx *PrimitiveTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitNumericType(ctx *NumericTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitIntegerType(ctx *IntegerTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitFloatType(ctx *FloatTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitTemporalType(ctx *TemporalTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitChannelType(ctx *ChannelTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitSeriesType(ctx *SeriesTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitExpression(ctx *ExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitLogicalOrExpression(ctx *LogicalOrExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitLogicalAndExpression(ctx *LogicalAndExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitEqualityExpression(ctx *EqualityExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitRelationalExpression(ctx *RelationalExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitAdditiveExpression(ctx *AdditiveExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitPowerExpression(ctx *PowerExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitUnaryExpression(ctx *UnaryExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitPostfixExpression(ctx *PostfixExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitIndexOrSlice(ctx *IndexOrSliceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitFunctionCallSuffix(ctx *FunctionCallSuffixContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitPrimaryExpression(ctx *PrimaryExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitTypeCast(ctx *TypeCastContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitLiteral(ctx *LiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitNumericLiteral(ctx *NumericLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitTemporalLiteral(ctx *TemporalLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitSeriesLiteral(ctx *SeriesLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseArcParserVisitor) VisitExpressionList(ctx *ExpressionListContext) interface{} {
	return v.VisitChildren(ctx)
}
