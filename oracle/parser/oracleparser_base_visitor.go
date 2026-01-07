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

func (v *BaseOracleParserVisitor) VisitFileDomain(ctx *FileDomainContext) interface{} {
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

func (v *BaseOracleParserVisitor) VisitAliasBody(ctx *AliasBodyContext) interface{} {
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

func (v *BaseOracleParserVisitor) VisitFieldOmit(ctx *FieldOmitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitFieldDef(ctx *FieldDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitInlineDomain(ctx *InlineDomainContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitFieldBody(ctx *FieldBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitDomain(ctx *DomainContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitDomainContent(ctx *DomainContentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitDomainBlock(ctx *DomainBlockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitTypeRefMap(ctx *TypeRefMapContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitTypeRefNormal(ctx *TypeRefNormalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitMapType(ctx *MapTypeContext) interface{} {
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

func (v *BaseOracleParserVisitor) VisitEnumBody(ctx *EnumBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseOracleParserVisitor) VisitEnumValue(ctx *EnumValueContext) interface{} {
	return v.VisitChildren(ctx)
}
