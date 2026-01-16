// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package formatter provides formatting functionality for Oracle schema files.
// It reformats Oracle source code according to the canonical style defined in STYLE.md.
package formatter

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/oracle/parser"
)

const (
	indent      = "    " // 4 spaces
	maxLineLen  = 88
	hiddenChan  = 1 // ANTLR hidden channel
	commentLine = parser.OracleLexerLINE_COMMENT
	commentBlk  = parser.OracleLexerBLOCK_COMMENT
)

// Format formats Oracle source code according to the canonical style.
// Returns the formatted source code or an error if parsing fails.
func Format(source string) (string, error) {
	input := antlr.NewInputStream(source)
	lexer := parser.NewOracleLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewOracleParser(stream)

	// Track syntax errors
	errListener := &errorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errListener)
	p.RemoveErrorListeners()
	p.AddErrorListener(errListener)

	// Parse the schema
	tree := p.Schema()
	if errListener.hasErrors {
		// Return original source if parsing fails
		return source, nil
	}

	f := &formatter{
		tokens: stream,
		sb:     &strings.Builder{},
	}

	f.formatSchema(tree)
	return f.sb.String(), nil
}

// errorListener tracks whether any syntax errors occurred.
type errorListener struct {
	*antlr.DefaultErrorListener
	hasErrors bool
}

func (e *errorListener) SyntaxError(
	_ antlr.Recognizer,
	_ interface{},
	_, _ int,
	_ string,
	_ antlr.RecognitionException,
) {
	e.hasErrors = true
}

type formatter struct {
	tokens        *antlr.CommonTokenStream
	sb            *strings.Builder
	lastTokenIdx  int
	currentIndent int
}

func (f *formatter) write(s string) {
	f.sb.WriteString(s)
}

func (f *formatter) writeLine(s string) {
	f.sb.WriteString(s)
	f.sb.WriteString("\n")
}

func (f *formatter) writeIndent() {
	for i := 0; i < f.currentIndent; i++ {
		f.sb.WriteString(indent)
	}
}

func (f *formatter) newline() {
	f.sb.WriteString("\n")
}

// emitLeadingComments emits any comments at the very start of the file.
// Returns true if any comments were emitted.
func (f *formatter) emitLeadingComments() bool {
	// Get all tokens and look for leading comments
	f.tokens.Fill()
	allTokens := f.tokens.GetAllTokens()
	emitted := false
	for _, tok := range allTokens {
		if tok.GetChannel() == hiddenChan {
			if tok.GetTokenType() == commentLine || tok.GetTokenType() == commentBlk {
				f.writeLine(tok.GetText())
				f.lastTokenIdx = tok.GetTokenIndex()
				emitted = true
			}
		} else if tok.GetChannel() == antlr.TokenDefaultChannel {
			// Newlines are on default channel - skip them to continue reading comments
			if tok.GetTokenType() == parser.OracleLexerNEWLINE {
				continue
			}
			// Stop at first non-newline default channel token
			break
		}
	}
	return emitted
}

// emitCommentsBefore emits any comments that appear before the given token index.
func (f *formatter) emitCommentsBefore(tokenIdx int) {
	hiddenTokens := f.tokens.GetHiddenTokensToLeft(tokenIdx, hiddenChan)
	for _, tok := range hiddenTokens {
		if tok.GetTokenIndex() <= f.lastTokenIdx {
			continue
		}
		text := tok.GetText()
		f.writeIndent()
		f.writeLine(text)
		f.lastTokenIdx = tok.GetTokenIndex()
	}
}

func (f *formatter) formatSchema(ctx parser.ISchemaContext) {
	// Track what we've emitted for blank line logic
	hasImports := len(ctx.AllImportStmt()) > 0
	hasDomains := len(ctx.AllFileDomain()) > 0
	hasDefinitions := len(ctx.AllDefinition()) > 0

	// Emit comments at start of file (before any content)
	hadLeadingComments := f.emitLeadingComments()

	// Blank line after leading comments (e.g., copyright header)
	if hadLeadingComments && (hasImports || hasDomains || hasDefinitions) {
		f.newline()
	}

	// Format imports
	for _, imp := range ctx.AllImportStmt() {
		f.formatImport(imp)
	}

	// Blank line after imports
	if hasImports && (hasDomains || hasDefinitions) {
		f.newline()
	}

	// Format file-level domains with alignment
	f.formatFileDomains(ctx.AllFileDomain())

	// Blank line after file-level domains
	if hasDomains && hasDefinitions {
		f.newline()
	}

	// Format definitions with blank lines between
	defs := ctx.AllDefinition()
	for i, def := range defs {
		if i > 0 {
			f.newline()
		}
		f.emitCommentsBefore(def.GetStart().GetTokenIndex())
		f.formatDefinition(def)
	}

	// Trailing newline
	if f.sb.Len() > 0 && !strings.HasSuffix(f.sb.String(), "\n") {
		f.newline()
	}
}

func (f *formatter) formatImport(ctx parser.IImportStmtContext) {
	f.emitCommentsBefore(ctx.GetStart().GetTokenIndex())
	f.write("import ")
	f.write(ctx.STRING_LIT().GetText())
	f.newline()
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) formatFileDomains(domains []parser.IFileDomainContext) {
	if len(domains) == 0 {
		return
	}

	// Calculate alignment: max length of "@domain command"
	maxPrefixLen := 0
	for _, dom := range domains {
		prefixLen := 1 + len(dom.IDENT().GetText()) // "@" + domain name
		if dom.DomainContent() != nil && dom.DomainContent().Expression() != nil {
			expr := dom.DomainContent().Expression()
			prefixLen += 1 + len(expr.IDENT().GetText()) // " " + command
		}
		if prefixLen > maxPrefixLen {
			maxPrefixLen = prefixLen
		}
	}

	for _, dom := range domains {
		f.emitCommentsBefore(dom.GetStart().GetTokenIndex())
		f.write("@")
		f.write(dom.IDENT().GetText())
		if dom.DomainContent() != nil {
			f.write(" ")
			f.formatDomainContentAligned(dom.DomainContent(), false, maxPrefixLen, 1+len(dom.IDENT().GetText()))
		}
		f.newline()
		f.lastTokenIdx = dom.GetStop().GetTokenIndex()
	}
}

// formatDomainContentAligned formats domain content with alignment padding.
// currentPrefixLen is the length of "@domain" so far, maxPrefixLen is the target.
func (f *formatter) formatDomainContentAligned(ctx parser.IDomainContentContext, allowBlock bool, maxPrefixLen, currentPrefixLen int) {
	if ctx.Expression() != nil {
		f.formatExpressionAligned(ctx.Expression(), maxPrefixLen, currentPrefixLen)
	} else if ctx.DomainBlock() != nil {
		f.formatDomainBlock(ctx.DomainBlock())
	}
}

func (f *formatter) formatExpressionAligned(ctx parser.IExpressionContext, maxPrefixLen, currentPrefixLen int) {
	command := ctx.IDENT().GetText()
	f.write(command)

	values := ctx.AllExpressionValue()
	if len(values) > 0 {
		// Calculate padding needed to align values
		fullPrefixLen := currentPrefixLen + 1 + len(command) // +1 for space after @domain
		padding := maxPrefixLen - fullPrefixLen
		if padding < 0 {
			padding = 0
		}
		f.writePadding(padding)
		f.write(" ")
		for i, val := range values {
			if i > 0 {
				f.write(" ")
			}
			f.formatExpressionValue(val)
		}
	}
}

func (f *formatter) formatDefinition(ctx parser.IDefinitionContext) {
	if ctx.StructDef() != nil {
		f.formatStructDef(ctx.StructDef())
	} else if ctx.EnumDef() != nil {
		f.formatEnumDef(ctx.EnumDef())
	} else if ctx.TypeDefDef() != nil {
		f.formatTypeDefDef(ctx.TypeDefDef())
	}
}

func (f *formatter) formatStructDef(ctx parser.IStructDefContext) {
	switch v := ctx.(type) {
	case *parser.StructFullContext:
		f.formatStructFull(v)
	case *parser.StructAliasContext:
		f.formatStructAlias(v)
	}
}

func (f *formatter) formatStructFull(ctx *parser.StructFullContext) {
	// Name struct<TypeParams> extends Parent {
	f.write(ctx.IDENT().GetText())
	f.write(" struct")

	// Type params come before extends
	if ctx.TypeParams() != nil {
		f.formatTypeParams(ctx.TypeParams())
	}

	// Handle extends clause (supports multiple inheritance)
	if ctx.EXTENDS() != nil && ctx.TypeRefList() != nil {
		f.write(" extends ")
		typeRefs := ctx.TypeRefList().AllTypeRef()
		for i, tr := range typeRefs {
			if i > 0 {
				f.write(", ")
			}
			f.formatTypeRef(tr)
		}
	}

	// Check if struct is empty
	body := ctx.StructBody()
	if isEmptyStructBody(body) {
		f.writeLine(" {}")
		f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
		return
	}

	f.writeLine(" {")
	f.currentIndent++
	f.formatStructBody(body)
	f.currentIndent--
	f.writeLine("}")
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func isEmptyStructBody(ctx parser.IStructBodyContext) bool {
	return len(ctx.AllFieldDef()) == 0 && len(ctx.AllDomain()) == 0 && len(ctx.AllFieldOmit()) == 0
}

func (f *formatter) formatStructAlias(ctx *parser.StructAliasContext) {
	f.write(ctx.IDENT().GetText())
	if ctx.TypeParams() != nil {
		f.formatTypeParams(ctx.TypeParams())
	}
	f.write(" = ")
	f.formatTypeRef(ctx.TypeRef())

	if ctx.AliasBody() != nil {
		f.formatAliasBody(ctx.AliasBody())
	} else {
		f.newline()
	}
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) formatAliasBody(ctx parser.IAliasBodyContext) {
	domains := ctx.AllDomain()
	if len(domains) == 0 {
		f.newline()
		return
	}

	f.writeLine(" {")
	f.currentIndent++
	f.formatDomains(domains)
	f.currentIndent--
	f.writeLine("}")
}

func (f *formatter) formatTypeParams(ctx parser.ITypeParamsContext) {
	// Try inline first
	inlineStr := f.formatTypeParamsToString(ctx)
	if f.currentLineLen()+len(inlineStr) <= maxLineLen {
		f.write(inlineStr)
		return
	}

	// Multi-line format
	f.writeLine("<")
	f.currentIndent++
	params := ctx.AllTypeParam()
	for i, param := range params {
		f.writeIndent()
		f.formatTypeParam(param)
		if i < len(params)-1 {
			f.write(",")
		}
		f.newline()
	}
	f.currentIndent--
	f.writeIndent()
	f.write(">")
}

func (f *formatter) formatTypeParamsToString(ctx parser.ITypeParamsContext) string {
	var sb strings.Builder
	sb.WriteString("<")
	params := ctx.AllTypeParam()
	for i, param := range params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(f.formatTypeParamToString(param))
	}
	sb.WriteString(">")
	return sb.String()
}

func (f *formatter) formatTypeParamToString(ctx parser.ITypeParamContext) string {
	var sb strings.Builder
	sb.WriteString(ctx.IDENT().GetText())
	if ctx.QUESTION() != nil {
		sb.WriteString("?")
	}
	typeRefs := ctx.AllTypeRef()
	hasExtends := ctx.EXTENDS() != nil
	if hasExtends && len(typeRefs) > 0 {
		sb.WriteString(" extends ")
		sb.WriteString(f.formatTypeRefToString(typeRefs[0]))
	}
	if ctx.EQUALS() != nil {
		idx := 0
		if hasExtends {
			idx = 1
		}
		if idx < len(typeRefs) {
			sb.WriteString(" = ")
			sb.WriteString(f.formatTypeRefToString(typeRefs[idx]))
		}
	}
	return sb.String()
}

func (f *formatter) formatTypeParam(ctx parser.ITypeParamContext) {
	f.write(ctx.IDENT().GetText())
	// Optional marker (?)
	if ctx.QUESTION() != nil {
		f.write("?")
	}
	typeRefs := ctx.AllTypeRef()
	hasExtends := ctx.EXTENDS() != nil
	if hasExtends && len(typeRefs) > 0 {
		f.write(" extends ")
		f.formatTypeRef(typeRefs[0])
	}
	// Default value (= X)
	if ctx.EQUALS() != nil {
		idx := 0
		if hasExtends {
			idx = 1
		}
		if idx < len(typeRefs) {
			f.write(" = ")
			f.formatTypeRef(typeRefs[idx])
		}
	}
}

func (f *formatter) formatStructBody(ctx parser.IStructBodyContext) {
	fields := ctx.AllFieldDef()
	fieldOmits := ctx.AllFieldOmit()
	domains := ctx.AllDomain()

	// Calculate alignment widths
	maxNameLen := 0
	maxTypeLen := 0
	for _, field := range fields {
		nameLen := len(field.IDENT().GetText())
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}
		typeLen := len(f.formatTypeRefToString(field.TypeRef()))
		if typeLen > maxTypeLen {
			maxTypeLen = typeLen
		}
	}

	// Format field omissions (-fieldName) first
	for _, omit := range fieldOmits {
		f.emitCommentsBefore(omit.GetStart().GetTokenIndex())
		f.formatFieldOmit(omit)
	}

	// Format fields with alignment
	for _, field := range fields {
		f.emitCommentsBefore(field.GetStart().GetTokenIndex())
		f.formatFieldDefAligned(field, maxNameLen, maxTypeLen)
	}

	// Blank line before struct-level domains if there are fields or omissions
	if (len(fields) > 0 || len(fieldOmits) > 0) && len(domains) > 0 {
		f.newline()
	}

	// Format struct-level domains with alignment
	f.formatDomains(domains)
}

func (f *formatter) formatFieldOmit(ctx parser.IFieldOmitContext) {
	f.writeIndent()
	f.write("-")
	f.write(ctx.IDENT().GetText())
	f.newline()
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) formatFieldDefAligned(ctx parser.IFieldDefContext, nameWidth, typeWidth int) {
	f.writeIndent()

	// Write name with padding
	name := ctx.IDENT().GetText()
	f.write(name)
	f.writePadding(nameWidth - len(name))
	f.write(" ")

	// Write type with padding (only if there are domains)
	typeStr := f.formatTypeRefToString(ctx.TypeRef())
	f.write(typeStr)

	inlineDomains := ctx.AllInlineDomain()
	hasDomains := len(inlineDomains) > 0 || ctx.FieldBody() != nil

	if hasDomains {
		f.writePadding(typeWidth - len(typeStr))

		// Try inline first
		inlineStr := f.formatInlineDomainsToString(inlineDomains)
		lineLen := f.currentLineLen() + len(inlineStr)

		if ctx.FieldBody() != nil || lineLen > maxLineLen {
			// Use brace form
			f.formatFieldWithBraces(inlineDomains, ctx.FieldBody())
		} else {
			// Use inline form
			f.write(inlineStr)
			f.newline()
		}
	} else {
		f.newline()
	}

	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) writePadding(n int) {
	for i := 0; i < n; i++ {
		f.write(" ")
	}
}

func (f *formatter) formatTypeRefToString(ctx parser.ITypeRefContext) string {
	var sb strings.Builder
	switch v := ctx.(type) {
	case *parser.TypeRefMapContext:
		sb.WriteString("map<")
		typeRefs := v.MapType().AllTypeRef()
		sb.WriteString(f.formatTypeRefToString(typeRefs[0]))
		sb.WriteString(", ")
		sb.WriteString(f.formatTypeRefToString(typeRefs[1]))
		sb.WriteString(">")
		if v.TypeModifiers() != nil {
			for range v.TypeModifiers().AllQUESTION() {
				sb.WriteString("?")
			}
		}
	case *parser.TypeRefNormalContext:
		sb.WriteString(f.formatQualifiedIdentToString(v.QualifiedIdent()))
		if v.TypeArgs() != nil {
			sb.WriteString("<")
			typeRefs := v.TypeArgs().AllTypeRef()
			for i, ref := range typeRefs {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(f.formatTypeRefToString(ref))
			}
			sb.WriteString(">")
		}
		if arrMod := v.ArrayModifier(); arrMod != nil {
			sb.WriteString("[")
			if intLit := arrMod.INT_LIT(); intLit != nil {
				sb.WriteString(intLit.GetText())
			}
			sb.WriteString("]")
		}
		if v.TypeModifiers() != nil {
			for range v.TypeModifiers().AllQUESTION() {
				sb.WriteString("?")
			}
		}
	}
	return sb.String()
}

func (f *formatter) currentLineLen() int {
	s := f.sb.String()
	lastNewline := strings.LastIndex(s, "\n")
	if lastNewline == -1 {
		return len(s)
	}
	return len(s) - lastNewline - 1
}

func (f *formatter) formatInlineDomainsToString(domains []parser.IInlineDomainContext) string {
	var sb strings.Builder
	for _, dom := range domains {
		sb.WriteString(" @")
		sb.WriteString(dom.IDENT().GetText())
		if dom.DomainContent() != nil {
			content := f.formatDomainContentToString(dom.DomainContent())
			if content != "" {
				sb.WriteString(" ")
				sb.WriteString(content)
			}
		}
	}
	return sb.String()
}

func (f *formatter) formatDomainContentToString(ctx parser.IDomainContentContext) string {
	if ctx.Expression() != nil {
		return f.formatExpressionToString(ctx.Expression())
	}
	// Block form - can't inline
	return ""
}

func (f *formatter) formatExpressionToString(ctx parser.IExpressionContext) string {
	var sb strings.Builder
	sb.WriteString(ctx.IDENT().GetText())
	for _, val := range ctx.AllExpressionValue() {
		sb.WriteString(" ")
		sb.WriteString(f.formatExpressionValueToString(val))
	}
	return sb.String()
}

func (f *formatter) formatExpressionValueToString(ctx parser.IExpressionValueContext) string {
	if ctx.STRING_LIT() != nil {
		return ctx.STRING_LIT().GetText()
	}
	if ctx.INT_LIT() != nil {
		return ctx.INT_LIT().GetText()
	}
	if ctx.FLOAT_LIT() != nil {
		return ctx.FLOAT_LIT().GetText()
	}
	if ctx.BOOL_LIT() != nil {
		return ctx.BOOL_LIT().GetText()
	}
	if ctx.QualifiedIdent() != nil {
		return f.formatQualifiedIdentToString(ctx.QualifiedIdent())
	}
	return ""
}

func (f *formatter) formatQualifiedIdentToString(ctx parser.IQualifiedIdentContext) string {
	idents := ctx.AllIDENT()
	if len(idents) == 1 {
		return idents[0].GetText()
	}
	return idents[0].GetText() + "." + idents[1].GetText()
}

func (f *formatter) formatFieldWithBraces(
	inlineDomains []parser.IInlineDomainContext,
	fieldBody parser.IFieldBodyContext,
) {
	f.writeLine(" {")
	f.currentIndent++

	// Calculate alignment: max length of "@domain command" across both inline and body domains
	maxPrefixLen := 0
	for _, dom := range inlineDomains {
		prefixLen := 1 + len(dom.IDENT().GetText()) // "@" + domain name
		if dom.DomainContent() != nil && dom.DomainContent().Expression() != nil {
			expr := dom.DomainContent().Expression()
			prefixLen += 1 + len(expr.IDENT().GetText()) // " " + command
		}
		if prefixLen > maxPrefixLen {
			maxPrefixLen = prefixLen
		}
	}
	if fieldBody != nil {
		for _, dom := range fieldBody.AllDomain() {
			prefixLen := 1 + len(dom.IDENT().GetText()) // "@" + domain name
			if dom.DomainContent() != nil && dom.DomainContent().Expression() != nil {
				expr := dom.DomainContent().Expression()
				prefixLen += 1 + len(expr.IDENT().GetText()) // " " + command
			}
			if prefixLen > maxPrefixLen {
				maxPrefixLen = prefixLen
			}
		}
	}

	// Convert inline domains to regular domains with alignment
	for _, dom := range inlineDomains {
		f.writeIndent()
		f.write("@")
		f.write(dom.IDENT().GetText())
		if dom.DomainContent() != nil {
			f.write(" ")
			f.formatDomainContentAligned(dom.DomainContent(), true, maxPrefixLen, 1+len(dom.IDENT().GetText()))
		}
		f.newline()
	}

	// Format field body domains with alignment
	if fieldBody != nil {
		for _, dom := range fieldBody.AllDomain() {
			f.emitCommentsBefore(dom.GetStart().GetTokenIndex())
			f.writeIndent()
			f.write("@")
			f.write(dom.IDENT().GetText())
			if dom.DomainContent() != nil {
				f.write(" ")
				f.formatDomainContentAligned(dom.DomainContent(), true, maxPrefixLen, 1+len(dom.IDENT().GetText()))
			}
			f.newline()
			f.lastTokenIdx = dom.GetStop().GetTokenIndex()
		}
	}

	f.currentIndent--
	f.writeIndent()
	f.writeLine("}")
}

func (f *formatter) formatDomains(domains []parser.IDomainContext) {
	if len(domains) == 0 {
		return
	}

	// Calculate alignment: max length of "@domain command"
	maxPrefixLen := 0
	for _, dom := range domains {
		prefixLen := 1 + len(dom.IDENT().GetText()) // "@" + domain name
		if dom.DomainContent() != nil && dom.DomainContent().Expression() != nil {
			expr := dom.DomainContent().Expression()
			prefixLen += 1 + len(expr.IDENT().GetText()) // " " + command
		}
		if prefixLen > maxPrefixLen {
			maxPrefixLen = prefixLen
		}
	}

	for _, dom := range domains {
		f.emitCommentsBefore(dom.GetStart().GetTokenIndex())
		f.writeIndent()
		f.write("@")
		f.write(dom.IDENT().GetText())
		if dom.DomainContent() != nil {
			f.write(" ")
			f.formatDomainContentAligned(dom.DomainContent(), true, maxPrefixLen, 1+len(dom.IDENT().GetText()))
		}
		f.newline()
		f.lastTokenIdx = dom.GetStop().GetTokenIndex()
	}
}

func (f *formatter) formatDomainBlock(ctx parser.IDomainBlockContext) {
	exprs := ctx.AllExpression()

	// Single expression - convert to inline (remove braces)
	if len(exprs) == 1 {
		f.formatExpression(exprs[0])
		return
	}

	// Multiple expressions - use block form
	f.writeLine("{")
	f.currentIndent++
	for _, expr := range exprs {
		f.writeIndent()
		f.formatExpression(expr)
		f.newline()
	}
	f.currentIndent--
	f.writeIndent()
	f.write("}")
}

func (f *formatter) formatExpression(ctx parser.IExpressionContext) {
	f.write(ctx.IDENT().GetText())
	for _, val := range ctx.AllExpressionValue() {
		f.write(" ")
		f.formatExpressionValue(val)
	}
}

func (f *formatter) formatExpressionValue(ctx parser.IExpressionValueContext) {
	if ctx.STRING_LIT() != nil {
		f.write(ctx.STRING_LIT().GetText())
	} else if ctx.INT_LIT() != nil {
		f.write(ctx.INT_LIT().GetText())
	} else if ctx.FLOAT_LIT() != nil {
		f.write(ctx.FLOAT_LIT().GetText())
	} else if ctx.BOOL_LIT() != nil {
		f.write(ctx.BOOL_LIT().GetText())
	} else if ctx.QualifiedIdent() != nil {
		f.formatQualifiedIdent(ctx.QualifiedIdent())
	}
}

func (f *formatter) formatQualifiedIdent(ctx parser.IQualifiedIdentContext) {
	idents := ctx.AllIDENT()
	f.write(idents[0].GetText())
	if len(idents) > 1 {
		f.write(".")
		f.write(idents[1].GetText())
	}
}

func (f *formatter) formatTypeRef(ctx parser.ITypeRefContext) {
	switch v := ctx.(type) {
	case *parser.TypeRefMapContext:
		f.formatMapType(v.MapType())
		if v.TypeModifiers() != nil {
			f.formatTypeModifiers(v.TypeModifiers())
		}
	case *parser.TypeRefNormalContext:
		f.formatQualifiedIdent(v.QualifiedIdent())
		if v.TypeArgs() != nil {
			f.formatTypeArgs(v.TypeArgs())
		}
		if arrMod := v.ArrayModifier(); arrMod != nil {
			f.write("[")
			if intLit := arrMod.INT_LIT(); intLit != nil {
				f.write(intLit.GetText())
			}
			f.write("]")
		}
		if v.TypeModifiers() != nil {
			f.formatTypeModifiers(v.TypeModifiers())
		}
	}
}

func (f *formatter) formatMapType(ctx parser.IMapTypeContext) {
	f.write("map<")
	typeRefs := ctx.AllTypeRef()
	f.formatTypeRef(typeRefs[0])
	f.write(", ")
	f.formatTypeRef(typeRefs[1])
	f.write(">")
}

func (f *formatter) formatTypeArgs(ctx parser.ITypeArgsContext) {
	f.write("<")
	typeRefs := ctx.AllTypeRef()
	for i, ref := range typeRefs {
		if i > 0 {
			f.write(", ")
		}
		f.formatTypeRef(ref)
	}
	f.write(">")
}

func (f *formatter) formatTypeModifiers(ctx parser.ITypeModifiersContext) {
	questions := ctx.AllQUESTION()
	for range questions {
		f.write("?")
	}
}

func (f *formatter) formatEnumDef(ctx parser.IEnumDefContext) {
	f.write(ctx.IDENT().GetText())
	f.write(" enum")

	body := ctx.EnumBody()
	values := body.AllEnumValue()
	domains := body.AllDomain()

	if len(values) == 0 && len(domains) == 0 {
		f.writeLine(" {}")
		f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
		return
	}

	f.writeLine(" {")
	f.currentIndent++

	// Calculate alignment for enum values
	maxNameLen := 0
	for _, val := range values {
		nameLen := len(val.IDENT().GetText())
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}
	}

	// Format enum values with alignment
	for _, val := range values {
		f.emitCommentsBefore(val.GetStart().GetTokenIndex())
		f.formatEnumValue(val, maxNameLen)
	}

	// Blank line before enum-level domains if there are values
	if len(values) > 0 && len(domains) > 0 {
		f.newline()
	}

	// Format enum-level domains with alignment
	f.formatDomains(domains)

	f.currentIndent--
	f.writeLine("}")
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) formatEnumValue(ctx parser.IEnumValueContext, alignTo int) {
	f.writeIndent()
	name := ctx.IDENT().GetText()
	f.write(name)

	// Pad for alignment
	padding := alignTo - len(name)
	for i := 0; i < padding; i++ {
		f.write(" ")
	}

	f.write(" = ")
	if ctx.INT_LIT() != nil {
		f.write(ctx.INT_LIT().GetText())
	} else if ctx.STRING_LIT() != nil {
		f.write(ctx.STRING_LIT().GetText())
	}
	f.newline()
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) formatTypeDefDef(ctx parser.ITypeDefDefContext) {
	// Format: Name<TypeParams> baseType { domains }
	f.write(ctx.IDENT().GetText())
	if ctx.TypeParams() != nil {
		f.formatTypeParams(ctx.TypeParams())
	}
	f.write(" ")
	f.formatTypeRef(ctx.TypeRef())

	body := ctx.TypeDefBody()
	if body == nil || len(body.AllDomain()) == 0 {
		f.newline()
		f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
		return
	}

	// Has domains - use brace form
	f.writeLine(" {")
	f.currentIndent++
	f.formatDomains(body.AllDomain())
	f.currentIndent--
	f.writeLine("}")
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}
