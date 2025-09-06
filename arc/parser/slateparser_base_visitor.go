// Code generated from SlateParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // SlateParser
import "github.com/antlr4-go/antlr/v4"

type BaseSlateParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseSlateParserVisitor) VisitProgram(ctx *ProgramContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitTopLevelItem(ctx *TopLevelItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitFunctionDeclaration(ctx *FunctionDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitParameterList(ctx *ParameterListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitParameter(ctx *ParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitReturnType(ctx *ReturnTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitTaskDeclaration(ctx *TaskDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitConfigBlock(ctx *ConfigBlockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitConfigParameter(ctx *ConfigParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitFlowStatement(ctx *FlowStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitFlowNode(ctx *FlowNodeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitChannelIdentifier(ctx *ChannelIdentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitTaskInvocation(ctx *TaskInvocationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitConfigValues(ctx *ConfigValuesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitNamedConfigValues(ctx *NamedConfigValuesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitNamedConfigValue(ctx *NamedConfigValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitAnonymousConfigValues(ctx *AnonymousConfigValuesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitArguments(ctx *ArgumentsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitArgumentList(ctx *ArgumentListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitBlock(ctx *BlockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitStatement(ctx *StatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitVariableDeclaration(ctx *VariableDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitLocalVariable(ctx *LocalVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitStatefulVariable(ctx *StatefulVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitAssignment(ctx *AssignmentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitChannelOperation(ctx *ChannelOperationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitChannelWrite(ctx *ChannelWriteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitChannelRead(ctx *ChannelReadContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitBlockingRead(ctx *BlockingReadContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitNonBlockingRead(ctx *NonBlockingReadContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitIfStatement(ctx *IfStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitElseIfClause(ctx *ElseIfClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitElseClause(ctx *ElseClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitReturnStatement(ctx *ReturnStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitFunctionCall(ctx *FunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitType(ctx *TypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitPrimitiveType(ctx *PrimitiveTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitNumericType(ctx *NumericTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitIntegerType(ctx *IntegerTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitFloatType(ctx *FloatTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitTemporalType(ctx *TemporalTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitChannelType(ctx *ChannelTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitSeriesType(ctx *SeriesTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitExpression(ctx *ExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitLogicalOrExpression(ctx *LogicalOrExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitLogicalAndExpression(ctx *LogicalAndExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitEqualityExpression(ctx *EqualityExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitRelationalExpression(ctx *RelationalExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitAdditiveExpression(ctx *AdditiveExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitMultiplicativeExpression(ctx *MultiplicativeExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitPowerExpression(ctx *PowerExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitUnaryExpression(ctx *UnaryExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitBlockingReadExpr(ctx *BlockingReadExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitPostfixExpression(ctx *PostfixExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitIndexOrSlice(ctx *IndexOrSliceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitFunctionCallSuffix(ctx *FunctionCallSuffixContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitPrimaryExpression(ctx *PrimaryExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitTypeCast(ctx *TypeCastContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitBuiltinFunction(ctx *BuiltinFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitLiteral(ctx *LiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitNumericLiteral(ctx *NumericLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitTemporalLiteral(ctx *TemporalLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitSeriesLiteral(ctx *SeriesLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSlateParserVisitor) VisitExpressionList(ctx *ExpressionListContext) interface{} {
	return v.VisitChildren(ctx)
}
