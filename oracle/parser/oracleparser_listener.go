// Code generated from OracleParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // OracleParser
import "github.com/antlr4-go/antlr/v4"

// OracleParserListener is a complete listener for a parse tree produced by OracleParser.
type OracleParserListener interface {
	antlr.ParseTreeListener

	// EnterSchema is called when entering the schema production.
	EnterSchema(c *SchemaContext)

	// EnterNl is called when entering the nl production.
	EnterNl(c *NlContext)

	// EnterImportStmt is called when entering the importStmt production.
	EnterImportStmt(c *ImportStmtContext)

	// EnterDefinition is called when entering the definition production.
	EnterDefinition(c *DefinitionContext)

	// EnterStructFull is called when entering the StructFull production.
	EnterStructFull(c *StructFullContext)

	// EnterStructAlias is called when entering the StructAlias production.
	EnterStructAlias(c *StructAliasContext)

	// EnterAliasBody is called when entering the aliasBody production.
	EnterAliasBody(c *AliasBodyContext)

	// EnterTypeParams is called when entering the typeParams production.
	EnterTypeParams(c *TypeParamsContext)

	// EnterTypeParam is called when entering the typeParam production.
	EnterTypeParam(c *TypeParamContext)

	// EnterStructBody is called when entering the structBody production.
	EnterStructBody(c *StructBodyContext)

	// EnterFieldDef is called when entering the fieldDef production.
	EnterFieldDef(c *FieldDefContext)

	// EnterFieldBody is called when entering the fieldBody production.
	EnterFieldBody(c *FieldBodyContext)

	// EnterDomainDef is called when entering the domainDef production.
	EnterDomainDef(c *DomainDefContext)

	// EnterDomainBody is called when entering the domainBody production.
	EnterDomainBody(c *DomainBodyContext)

	// EnterTypeRef is called when entering the typeRef production.
	EnterTypeRef(c *TypeRefContext)

	// EnterTypeArgs is called when entering the typeArgs production.
	EnterTypeArgs(c *TypeArgsContext)

	// EnterTypeModifiers is called when entering the typeModifiers production.
	EnterTypeModifiers(c *TypeModifiersContext)

	// EnterQualifiedIdent is called when entering the qualifiedIdent production.
	EnterQualifiedIdent(c *QualifiedIdentContext)

	// EnterExpression is called when entering the expression production.
	EnterExpression(c *ExpressionContext)

	// EnterExpressionValue is called when entering the expressionValue production.
	EnterExpressionValue(c *ExpressionValueContext)

	// EnterEnumDef is called when entering the enumDef production.
	EnterEnumDef(c *EnumDefContext)

	// EnterEnumBody is called when entering the enumBody production.
	EnterEnumBody(c *EnumBodyContext)

	// EnterEnumValue is called when entering the enumValue production.
	EnterEnumValue(c *EnumValueContext)

	// ExitSchema is called when exiting the schema production.
	ExitSchema(c *SchemaContext)

	// ExitNl is called when exiting the nl production.
	ExitNl(c *NlContext)

	// ExitImportStmt is called when exiting the importStmt production.
	ExitImportStmt(c *ImportStmtContext)

	// ExitDefinition is called when exiting the definition production.
	ExitDefinition(c *DefinitionContext)

	// ExitStructFull is called when exiting the StructFull production.
	ExitStructFull(c *StructFullContext)

	// ExitStructAlias is called when exiting the StructAlias production.
	ExitStructAlias(c *StructAliasContext)

	// ExitAliasBody is called when exiting the aliasBody production.
	ExitAliasBody(c *AliasBodyContext)

	// ExitTypeParams is called when exiting the typeParams production.
	ExitTypeParams(c *TypeParamsContext)

	// ExitTypeParam is called when exiting the typeParam production.
	ExitTypeParam(c *TypeParamContext)

	// ExitStructBody is called when exiting the structBody production.
	ExitStructBody(c *StructBodyContext)

	// ExitFieldDef is called when exiting the fieldDef production.
	ExitFieldDef(c *FieldDefContext)

	// ExitFieldBody is called when exiting the fieldBody production.
	ExitFieldBody(c *FieldBodyContext)

	// ExitDomainDef is called when exiting the domainDef production.
	ExitDomainDef(c *DomainDefContext)

	// ExitDomainBody is called when exiting the domainBody production.
	ExitDomainBody(c *DomainBodyContext)

	// ExitTypeRef is called when exiting the typeRef production.
	ExitTypeRef(c *TypeRefContext)

	// ExitTypeArgs is called when exiting the typeArgs production.
	ExitTypeArgs(c *TypeArgsContext)

	// ExitTypeModifiers is called when exiting the typeModifiers production.
	ExitTypeModifiers(c *TypeModifiersContext)

	// ExitQualifiedIdent is called when exiting the qualifiedIdent production.
	ExitQualifiedIdent(c *QualifiedIdentContext)

	// ExitExpression is called when exiting the expression production.
	ExitExpression(c *ExpressionContext)

	// ExitExpressionValue is called when exiting the expressionValue production.
	ExitExpressionValue(c *ExpressionValueContext)

	// ExitEnumDef is called when exiting the enumDef production.
	ExitEnumDef(c *EnumDefContext)

	// ExitEnumBody is called when exiting the enumBody production.
	ExitEnumBody(c *EnumBodyContext)

	// ExitEnumValue is called when exiting the enumValue production.
	ExitEnumValue(c *EnumValueContext)
}
