// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

	// EnterFileDomain is called when entering the fileDomain production.
	EnterFileDomain(c *FileDomainContext)

	// EnterDefinition is called when entering the definition production.
	EnterDefinition(c *DefinitionContext)

	// EnterStructFull is called when entering the StructFull production.
	EnterStructFull(c *StructFullContext)

	// EnterStructAlias is called when entering the StructAlias production.
	EnterStructAlias(c *StructAliasContext)

	// EnterTypeRefList is called when entering the typeRefList production.
	EnterTypeRefList(c *TypeRefListContext)

	// EnterAliasBody is called when entering the aliasBody production.
	EnterAliasBody(c *AliasBodyContext)

	// EnterTypeParams is called when entering the typeParams production.
	EnterTypeParams(c *TypeParamsContext)

	// EnterTypeParam is called when entering the typeParam production.
	EnterTypeParam(c *TypeParamContext)

	// EnterStructBody is called when entering the structBody production.
	EnterStructBody(c *StructBodyContext)

	// EnterFieldOmit is called when entering the fieldOmit production.
	EnterFieldOmit(c *FieldOmitContext)

	// EnterFieldDef is called when entering the fieldDef production.
	EnterFieldDef(c *FieldDefContext)

	// EnterInlineDomain is called when entering the inlineDomain production.
	EnterInlineDomain(c *InlineDomainContext)

	// EnterFieldBody is called when entering the fieldBody production.
	EnterFieldBody(c *FieldBodyContext)

	// EnterDomain is called when entering the domain production.
	EnterDomain(c *DomainContext)

	// EnterDomainContent is called when entering the domainContent production.
	EnterDomainContent(c *DomainContentContext)

	// EnterDomainBlock is called when entering the domainBlock production.
	EnterDomainBlock(c *DomainBlockContext)

	// EnterTypeRefMap is called when entering the TypeRefMap production.
	EnterTypeRefMap(c *TypeRefMapContext)

	// EnterTypeRefNormal is called when entering the TypeRefNormal production.
	EnterTypeRefNormal(c *TypeRefNormalContext)

	// EnterArrayModifier is called when entering the arrayModifier production.
	EnterArrayModifier(c *ArrayModifierContext)

	// EnterMapType is called when entering the mapType production.
	EnterMapType(c *MapTypeContext)

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

	// EnterEnumValueBody is called when entering the enumValueBody production.
	EnterEnumValueBody(c *EnumValueBodyContext)

	// EnterTypeDefDef is called when entering the typeDefDef production.
	EnterTypeDefDef(c *TypeDefDefContext)

	// EnterTypeDefBody is called when entering the typeDefBody production.
	EnterTypeDefBody(c *TypeDefBodyContext)

	// ExitSchema is called when exiting the schema production.
	ExitSchema(c *SchemaContext)

	// ExitNl is called when exiting the nl production.
	ExitNl(c *NlContext)

	// ExitImportStmt is called when exiting the importStmt production.
	ExitImportStmt(c *ImportStmtContext)

	// ExitFileDomain is called when exiting the fileDomain production.
	ExitFileDomain(c *FileDomainContext)

	// ExitDefinition is called when exiting the definition production.
	ExitDefinition(c *DefinitionContext)

	// ExitStructFull is called when exiting the StructFull production.
	ExitStructFull(c *StructFullContext)

	// ExitStructAlias is called when exiting the StructAlias production.
	ExitStructAlias(c *StructAliasContext)

	// ExitTypeRefList is called when exiting the typeRefList production.
	ExitTypeRefList(c *TypeRefListContext)

	// ExitAliasBody is called when exiting the aliasBody production.
	ExitAliasBody(c *AliasBodyContext)

	// ExitTypeParams is called when exiting the typeParams production.
	ExitTypeParams(c *TypeParamsContext)

	// ExitTypeParam is called when exiting the typeParam production.
	ExitTypeParam(c *TypeParamContext)

	// ExitStructBody is called when exiting the structBody production.
	ExitStructBody(c *StructBodyContext)

	// ExitFieldOmit is called when exiting the fieldOmit production.
	ExitFieldOmit(c *FieldOmitContext)

	// ExitFieldDef is called when exiting the fieldDef production.
	ExitFieldDef(c *FieldDefContext)

	// ExitInlineDomain is called when exiting the inlineDomain production.
	ExitInlineDomain(c *InlineDomainContext)

	// ExitFieldBody is called when exiting the fieldBody production.
	ExitFieldBody(c *FieldBodyContext)

	// ExitDomain is called when exiting the domain production.
	ExitDomain(c *DomainContext)

	// ExitDomainContent is called when exiting the domainContent production.
	ExitDomainContent(c *DomainContentContext)

	// ExitDomainBlock is called when exiting the domainBlock production.
	ExitDomainBlock(c *DomainBlockContext)

	// ExitTypeRefMap is called when exiting the TypeRefMap production.
	ExitTypeRefMap(c *TypeRefMapContext)

	// ExitTypeRefNormal is called when exiting the TypeRefNormal production.
	ExitTypeRefNormal(c *TypeRefNormalContext)

	// ExitArrayModifier is called when exiting the arrayModifier production.
	ExitArrayModifier(c *ArrayModifierContext)

	// ExitMapType is called when exiting the mapType production.
	ExitMapType(c *MapTypeContext)

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

	// ExitEnumValueBody is called when exiting the enumValueBody production.
	ExitEnumValueBody(c *EnumValueBodyContext)

	// ExitTypeDefDef is called when exiting the typeDefDef production.
	ExitTypeDefDef(c *TypeDefDefContext)

	// ExitTypeDefBody is called when exiting the typeDefBody production.
	ExitTypeDefBody(c *TypeDefBodyContext)
}
