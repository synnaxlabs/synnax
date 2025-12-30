// Code generated from OracleParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // OracleParser
import "github.com/antlr4-go/antlr/v4"

// BaseOracleParserListener is a complete listener for a parse tree produced by OracleParser.
type BaseOracleParserListener struct{}

var _ OracleParserListener = &BaseOracleParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseOracleParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseOracleParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseOracleParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseOracleParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterSchema is called when production schema is entered.
func (s *BaseOracleParserListener) EnterSchema(ctx *SchemaContext) {}

// ExitSchema is called when production schema is exited.
func (s *BaseOracleParserListener) ExitSchema(ctx *SchemaContext) {}

// EnterNl is called when production nl is entered.
func (s *BaseOracleParserListener) EnterNl(ctx *NlContext) {}

// ExitNl is called when production nl is exited.
func (s *BaseOracleParserListener) ExitNl(ctx *NlContext) {}

// EnterImportStmt is called when production importStmt is entered.
func (s *BaseOracleParserListener) EnterImportStmt(ctx *ImportStmtContext) {}

// ExitImportStmt is called when production importStmt is exited.
func (s *BaseOracleParserListener) ExitImportStmt(ctx *ImportStmtContext) {}

// EnterDefinition is called when production definition is entered.
func (s *BaseOracleParserListener) EnterDefinition(ctx *DefinitionContext) {}

// ExitDefinition is called when production definition is exited.
func (s *BaseOracleParserListener) ExitDefinition(ctx *DefinitionContext) {}

// EnterStructFull is called when production StructFull is entered.
func (s *BaseOracleParserListener) EnterStructFull(ctx *StructFullContext) {}

// ExitStructFull is called when production StructFull is exited.
func (s *BaseOracleParserListener) ExitStructFull(ctx *StructFullContext) {}

// EnterStructAlias is called when production StructAlias is entered.
func (s *BaseOracleParserListener) EnterStructAlias(ctx *StructAliasContext) {}

// ExitStructAlias is called when production StructAlias is exited.
func (s *BaseOracleParserListener) ExitStructAlias(ctx *StructAliasContext) {}

// EnterAliasBody is called when production aliasBody is entered.
func (s *BaseOracleParserListener) EnterAliasBody(ctx *AliasBodyContext) {}

// ExitAliasBody is called when production aliasBody is exited.
func (s *BaseOracleParserListener) ExitAliasBody(ctx *AliasBodyContext) {}

// EnterTypeParams is called when production typeParams is entered.
func (s *BaseOracleParserListener) EnterTypeParams(ctx *TypeParamsContext) {}

// ExitTypeParams is called when production typeParams is exited.
func (s *BaseOracleParserListener) ExitTypeParams(ctx *TypeParamsContext) {}

// EnterTypeParam is called when production typeParam is entered.
func (s *BaseOracleParserListener) EnterTypeParam(ctx *TypeParamContext) {}

// ExitTypeParam is called when production typeParam is exited.
func (s *BaseOracleParserListener) ExitTypeParam(ctx *TypeParamContext) {}

// EnterStructBody is called when production structBody is entered.
func (s *BaseOracleParserListener) EnterStructBody(ctx *StructBodyContext) {}

// ExitStructBody is called when production structBody is exited.
func (s *BaseOracleParserListener) ExitStructBody(ctx *StructBodyContext) {}

// EnterFieldDef is called when production fieldDef is entered.
func (s *BaseOracleParserListener) EnterFieldDef(ctx *FieldDefContext) {}

// ExitFieldDef is called when production fieldDef is exited.
func (s *BaseOracleParserListener) ExitFieldDef(ctx *FieldDefContext) {}

// EnterFieldBody is called when production fieldBody is entered.
func (s *BaseOracleParserListener) EnterFieldBody(ctx *FieldBodyContext) {}

// ExitFieldBody is called when production fieldBody is exited.
func (s *BaseOracleParserListener) ExitFieldBody(ctx *FieldBodyContext) {}

// EnterDomainDef is called when production domainDef is entered.
func (s *BaseOracleParserListener) EnterDomainDef(ctx *DomainDefContext) {}

// ExitDomainDef is called when production domainDef is exited.
func (s *BaseOracleParserListener) ExitDomainDef(ctx *DomainDefContext) {}

// EnterDomainBody is called when production domainBody is entered.
func (s *BaseOracleParserListener) EnterDomainBody(ctx *DomainBodyContext) {}

// ExitDomainBody is called when production domainBody is exited.
func (s *BaseOracleParserListener) ExitDomainBody(ctx *DomainBodyContext) {}

// EnterTypeRef is called when production typeRef is entered.
func (s *BaseOracleParserListener) EnterTypeRef(ctx *TypeRefContext) {}

// ExitTypeRef is called when production typeRef is exited.
func (s *BaseOracleParserListener) ExitTypeRef(ctx *TypeRefContext) {}

// EnterTypeArgs is called when production typeArgs is entered.
func (s *BaseOracleParserListener) EnterTypeArgs(ctx *TypeArgsContext) {}

// ExitTypeArgs is called when production typeArgs is exited.
func (s *BaseOracleParserListener) ExitTypeArgs(ctx *TypeArgsContext) {}

// EnterTypeModifiers is called when production typeModifiers is entered.
func (s *BaseOracleParserListener) EnterTypeModifiers(ctx *TypeModifiersContext) {}

// ExitTypeModifiers is called when production typeModifiers is exited.
func (s *BaseOracleParserListener) ExitTypeModifiers(ctx *TypeModifiersContext) {}

// EnterQualifiedIdent is called when production qualifiedIdent is entered.
func (s *BaseOracleParserListener) EnterQualifiedIdent(ctx *QualifiedIdentContext) {}

// ExitQualifiedIdent is called when production qualifiedIdent is exited.
func (s *BaseOracleParserListener) ExitQualifiedIdent(ctx *QualifiedIdentContext) {}

// EnterExpression is called when production expression is entered.
func (s *BaseOracleParserListener) EnterExpression(ctx *ExpressionContext) {}

// ExitExpression is called when production expression is exited.
func (s *BaseOracleParserListener) ExitExpression(ctx *ExpressionContext) {}

// EnterExpressionValue is called when production expressionValue is entered.
func (s *BaseOracleParserListener) EnterExpressionValue(ctx *ExpressionValueContext) {}

// ExitExpressionValue is called when production expressionValue is exited.
func (s *BaseOracleParserListener) ExitExpressionValue(ctx *ExpressionValueContext) {}

// EnterEnumDef is called when production enumDef is entered.
func (s *BaseOracleParserListener) EnterEnumDef(ctx *EnumDefContext) {}

// ExitEnumDef is called when production enumDef is exited.
func (s *BaseOracleParserListener) ExitEnumDef(ctx *EnumDefContext) {}

// EnterEnumValue is called when production enumValue is entered.
func (s *BaseOracleParserListener) EnterEnumValue(ctx *EnumValueContext) {}

// ExitEnumValue is called when production enumValue is exited.
func (s *BaseOracleParserListener) ExitEnumValue(ctx *EnumValueContext) {}
