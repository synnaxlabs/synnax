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

	// Visit a parse tree produced by OracleParser#definition.
	VisitDefinition(ctx *DefinitionContext) interface{}

	// Visit a parse tree produced by OracleParser#structDef.
	VisitStructDef(ctx *StructDefContext) interface{}

	// Visit a parse tree produced by OracleParser#structBody.
	VisitStructBody(ctx *StructBodyContext) interface{}

	// Visit a parse tree produced by OracleParser#fieldDef.
	VisitFieldDef(ctx *FieldDefContext) interface{}

	// Visit a parse tree produced by OracleParser#fieldBody.
	VisitFieldBody(ctx *FieldBodyContext) interface{}

	// Visit a parse tree produced by OracleParser#domainDef.
	VisitDomainDef(ctx *DomainDefContext) interface{}

	// Visit a parse tree produced by OracleParser#domainBody.
	VisitDomainBody(ctx *DomainBodyContext) interface{}

	// Visit a parse tree produced by OracleParser#typeRef.
	VisitTypeRef(ctx *TypeRefContext) interface{}

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

	// Visit a parse tree produced by OracleParser#enumValue.
	VisitEnumValue(ctx *EnumValueContext) interface{}
}
