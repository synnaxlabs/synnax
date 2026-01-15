// Code generated from OracleParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // OracleParser
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by OracleParser.
type OracleParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by OracleParser#schema.
	VisitSchema(ctx *SchemaContext) interface{}

	// Visit a parse tree produced by OracleParser#nl.
	VisitNl(ctx *NlContext) interface{}

	// Visit a parse tree produced by OracleParser#importStmt.
	VisitImportStmt(ctx *ImportStmtContext) interface{}

	// Visit a parse tree produced by OracleParser#fileDomain.
	VisitFileDomain(ctx *FileDomainContext) interface{}

	// Visit a parse tree produced by OracleParser#definition.
	VisitDefinition(ctx *DefinitionContext) interface{}

	// Visit a parse tree produced by OracleParser#StructFull.
	VisitStructFull(ctx *StructFullContext) interface{}

	// Visit a parse tree produced by OracleParser#StructAlias.
	VisitStructAlias(ctx *StructAliasContext) interface{}

	// Visit a parse tree produced by OracleParser#typeRefList.
	VisitTypeRefList(ctx *TypeRefListContext) interface{}

	// Visit a parse tree produced by OracleParser#aliasBody.
	VisitAliasBody(ctx *AliasBodyContext) interface{}

	// Visit a parse tree produced by OracleParser#typeParams.
	VisitTypeParams(ctx *TypeParamsContext) interface{}

	// Visit a parse tree produced by OracleParser#typeParam.
	VisitTypeParam(ctx *TypeParamContext) interface{}

	// Visit a parse tree produced by OracleParser#structBody.
	VisitStructBody(ctx *StructBodyContext) interface{}

	// Visit a parse tree produced by OracleParser#fieldOmit.
	VisitFieldOmit(ctx *FieldOmitContext) interface{}

	// Visit a parse tree produced by OracleParser#fieldDef.
	VisitFieldDef(ctx *FieldDefContext) interface{}

	// Visit a parse tree produced by OracleParser#inlineDomain.
	VisitInlineDomain(ctx *InlineDomainContext) interface{}

	// Visit a parse tree produced by OracleParser#fieldBody.
	VisitFieldBody(ctx *FieldBodyContext) interface{}

	// Visit a parse tree produced by OracleParser#domain.
	VisitDomain(ctx *DomainContext) interface{}

	// Visit a parse tree produced by OracleParser#domainContent.
	VisitDomainContent(ctx *DomainContentContext) interface{}

	// Visit a parse tree produced by OracleParser#domainBlock.
	VisitDomainBlock(ctx *DomainBlockContext) interface{}

	// Visit a parse tree produced by OracleParser#TypeRefMap.
	VisitTypeRefMap(ctx *TypeRefMapContext) interface{}

	// Visit a parse tree produced by OracleParser#TypeRefNormal.
	VisitTypeRefNormal(ctx *TypeRefNormalContext) interface{}

	// Visit a parse tree produced by OracleParser#arrayModifier.
	VisitArrayModifier(ctx *ArrayModifierContext) interface{}

	// Visit a parse tree produced by OracleParser#mapType.
	VisitMapType(ctx *MapTypeContext) interface{}

	// Visit a parse tree produced by OracleParser#typeArgs.
	VisitTypeArgs(ctx *TypeArgsContext) interface{}

	// Visit a parse tree produced by OracleParser#typeModifiers.
	VisitTypeModifiers(ctx *TypeModifiersContext) interface{}

	// Visit a parse tree produced by OracleParser#qualifiedIdent.
	VisitQualifiedIdent(ctx *QualifiedIdentContext) interface{}

	// Visit a parse tree produced by OracleParser#expression.
	VisitExpression(ctx *ExpressionContext) interface{}

	// Visit a parse tree produced by OracleParser#expressionValue.
	VisitExpressionValue(ctx *ExpressionValueContext) interface{}

	// Visit a parse tree produced by OracleParser#enumDef.
	VisitEnumDef(ctx *EnumDefContext) interface{}

	// Visit a parse tree produced by OracleParser#enumBody.
	VisitEnumBody(ctx *EnumBodyContext) interface{}

	// Visit a parse tree produced by OracleParser#enumValue.
	VisitEnumValue(ctx *EnumValueContext) interface{}

	// Visit a parse tree produced by OracleParser#typeDefDef.
	VisitTypeDefDef(ctx *TypeDefDefContext) interface{}

	// Visit a parse tree produced by OracleParser#typeDefBody.
	VisitTypeDefBody(ctx *TypeDefBodyContext) interface{}
}
