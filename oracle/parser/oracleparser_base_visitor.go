// Code generated from OracleParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // OracleParser
import "github.com/antlr4-go/antlr/v4"

type BaseOracleParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseOracleParserVisitor) VisitSchema(ctx *SchemaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitNl(ctx *NlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitImportStmt(ctx *ImportStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitDefinition(ctx *DefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitStructFull(ctx *StructFullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitStructAlias(ctx *StructAliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitTypeParams(ctx *TypeParamsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitTypeParam(ctx *TypeParamContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitStructBody(ctx *StructBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitFieldDef(ctx *FieldDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitFieldBody(ctx *FieldBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitDomainDef(ctx *DomainDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitDomainBody(ctx *DomainBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitTypeRef(ctx *TypeRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitTypeArgs(ctx *TypeArgsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitTypeModifiers(ctx *TypeModifiersContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitQualifiedIdent(ctx *QualifiedIdentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitExpression(ctx *ExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitExpressionValue(ctx *ExpressionValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitEnumDef(ctx *EnumDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitEnumValue(ctx *EnumValueContext) interface{} {
	return v.VisitChildren(ctx)
}
