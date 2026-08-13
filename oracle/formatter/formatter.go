// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package formatter provides formatting functionality for Oracle schema files.
// It reformats Oracle source code according to the canonical style.
package formatter

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/oracle/parser"
)

const (
	indent      = "    " // 4 spaces
	maxLineLen  = 88
	tripleQuote = `"""`
	hiddenChan  = 1 // ANTLR hidden channel
	commentLine = parser.OracleLexerLINE_COMMENT
	commentBlk  = parser.OracleLexerBLOCK_COMMENT
)

// Format formats Oracle source code according to the canonical style. Returns the
// formatted source code or an error if parsing fails.
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
	_ any,
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
	// currentDomain is the name of the domain whose content is being emitted ("doc",
	// "go", ...); doc strings get prose layout rules.
	currentDomain string
}

func (f *formatter) write(s string) { f.sb.WriteString(s) }

func (f *formatter) writeLine(s string) { f.sb.WriteString(s); f.sb.WriteString("\n") }

func (f *formatter) writeIndent() {
	for i := 0; i < f.currentIndent; i++ {
		f.sb.WriteString(indent)
	}
}

func (f *formatter) newline() { f.sb.WriteString("\n") }

// emitLeadingComments emits any comments at the very start of the file. Returns true if
// any comments were emitted.
func (f *formatter) emitLeadingComments() bool {
	// Get all tokens and look for leading comments
	f.tokens.Fill()
	allTokens := f.tokens.GetAllTokens()
	var emitted bool
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

	// Format imports in alphabetical order. Comments in the import region are emitted
	// up front so reordering cannot drop or duplicate them.
	imports := ctx.AllImportStmt()
	if len(imports) > 0 {
		f.emitCommentsBefore(imports[len(imports)-1].GetStart().GetTokenIndex())
		sorted := make([]parser.IImportStmtContext, len(imports))
		copy(sorted, imports)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].STRING_LIT().GetText() < sorted[j].STRING_LIT().GetText()
		})
		maxStop := f.lastTokenIdx
		for _, imp := range sorted {
			f.formatImport(imp)
			if stop := imp.GetStop().GetTokenIndex(); stop > maxStop {
				maxStop = stop
			}
		}
		f.lastTokenIdx = maxStop
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
	domains = sortDomainLines(f, domains)

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
		f.currentDomain = dom.IDENT().GetText()
		if dom.DomainContent() != nil {
			f.write(" ")
			f.formatDomainContentAligned(
				dom.DomainContent(),
				maxPrefixLen,
				1+len(dom.IDENT().GetText()),
			)
		}
		f.newline()
		f.lastTokenIdx = dom.GetStop().GetTokenIndex()
	}
}

// formatDomainContentAligned formats domain content with alignment padding.
// currentPrefixLen is the length of "@domain" so far, maxPrefixLen is the target.
func (f *formatter) formatDomainContentAligned(
	ctx parser.IDomainContentContext,
	maxPrefixLen, currentPrefixLen int,
) {
	if ctx.Expression() != nil {
		f.formatExpressionAligned(ctx.Expression(), maxPrefixLen, currentPrefixLen)
	} else if ctx.DomainBlock() != nil {
		f.formatDomainBlock(ctx.DomainBlock())
	}
}

func (f *formatter) formatExpressionAligned(
	ctx parser.IExpressionContext,
	maxPrefixLen, currentPrefixLen int,
) {
	command := ctx.IDENT().GetText()
	f.write(command)

	values := ctx.AllExpressionValue()
	if len(values) > 0 {
		// Calculate padding needed to align values
		fullPrefixLen := currentPrefixLen + 1 + len(
			command,
		) // +1 for space after @domain
		padding := max(maxPrefixLen-fullPrefixLen, 0)
		f.writePadding(padding)
		f.write(" ")
		if len(values) == 1 {
			f.formatSoleExpressionValue(values[0])
			return
		}
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
	} else if ctx.UnionDef() != nil {
		f.formatUnionDef(ctx.UnionDef())
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
	return len(ctx.AllFieldDef()) == 0 &&
		len(ctx.AllDomain()) == 0 &&
		len(ctx.AllFieldOmit()) == 0 &&
		len(ctx.AllActionDef()) == 0
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
	actions := ctx.AllActionDef()
	domains := ctx.AllDomain()

	// Calculate alignment widths
	maxNameLen := 0
	maxTypeLen := 0
	for _, field := range fields {
		nameLen := len(fieldNameColumn(field))
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

	// Blank line before actions if there are fields or omissions
	if (len(fields) > 0 || len(fieldOmits) > 0) && len(actions) > 0 {
		f.newline()
	}

	// Format actions, each separated from neighbors by a blank line
	for i, action := range actions {
		f.emitCommentsBefore(action.GetStart().GetTokenIndex())
		f.formatActionDef(action)
		if i < len(actions)-1 {
			f.newline()
		}
	}

	// Blank line before struct-level domains if there are fields, omissions, or actions
	if (len(fields) > 0 || len(fieldOmits) > 0 || len(actions) > 0) &&
		len(domains) > 0 {
		f.newline()
	}

	// Format struct-level domains with alignment
	f.formatDomains(domains)
}

func (f *formatter) formatActionDef(ctx parser.IActionDefContext) {
	f.writeIndent()
	f.write("action ")
	f.write(ctx.IDENT().GetText())

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

	body := ctx.ActionBody()
	if body == nil || isEmptyActionBody(body) {
		f.writeLine(" {}")
		f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
		return
	}

	f.writeLine(" {")
	f.currentIndent++
	f.formatActionBody(body)
	f.currentIndent--
	f.writeIndent()
	f.writeLine("}")
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func isEmptyActionBody(ctx parser.IActionBodyContext) bool {
	return len(ctx.AllFieldDef()) == 0 && len(ctx.AllDomain()) == 0
}

func (f *formatter) formatActionBody(ctx parser.IActionBodyContext) {
	fields := ctx.AllFieldDef()
	domains := ctx.AllDomain()

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

	for _, field := range fields {
		f.emitCommentsBefore(field.GetStart().GetTokenIndex())
		f.formatFieldDefAligned(field, maxNameLen, maxTypeLen)
	}

	if len(fields) > 0 && len(domains) > 0 {
		f.newline()
	}

	f.formatDomains(domains)
}

func (f *formatter) formatFieldOmit(ctx parser.IFieldOmitContext) {
	f.writeIndent()
	f.write("-")
	f.write(ctx.IDENT().GetText())
	f.newline()
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) formatFieldDefAligned(
	ctx parser.IFieldDefContext,
	nameWidth, typeWidth int,
) {
	f.writeIndent()

	// The name column carries a standalone optionality marker (key?) when the field
	// omits its type to inherit it from the parent.
	nameCol := fieldNameColumn(ctx)
	typeStr := f.formatTypeRefToString(ctx.TypeRef())
	hasDefault := ctx.EQUALS() != nil && ctx.FieldDefault() != nil
	inlineDomains := ctx.AllInlineDomain()
	domainOmits := ctx.AllDomainOmit()
	hasDomains := len(inlineDomains) > 0 || len(domainOmits) > 0 ||
		ctx.FieldBody() != nil

	f.write(nameCol)

	// Only pad the name column when content follows, so a bare typeless override (key?)
	// does not trail whitespace.
	if typeStr != "" || hasDefault || hasDomains {
		f.writePadding(nameWidth - len(nameCol))
	}
	if typeStr != "" {
		f.write(" ")
		f.write(typeStr)
	}

	// Inline default value: name type = X. Struct and array literals that would
	// overflow the line are broken across multiple lines.
	if hasDefault {
		f.write(" = ")
		f.write(
			f.formatFieldDefaultPretty(
				ctx.FieldDefault(),
				f.currentLineLen(),
				f.currentIndent,
			),
		)
	}

	if hasDomains {
		// A typed field aligns domains in the type column; a default (= X) breaks that
		// alignment. A typeless override has no type column, so its content follows the
		// padded name column directly.
		if !hasDefault && typeStr != "" {
			f.writePadding(typeWidth - len(typeStr))
		}

		inlineStr := f.formatInlineDomainsToString(inlineDomains) +
			f.formatDomainOmitsToString(domainOmits)

		if ctx.FieldBody() != nil || hasTripleString(inlineDomains) ||
			f.currentLineLen()+len(inlineStr) > maxLineLen {
			f.formatFieldWithBraces(inlineDomains, domainOmits, ctx.FieldBody())
		} else {
			f.write(inlineStr)
			f.newline()
		}
	} else {
		f.newline()
	}

	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) writePadding(n int) {
	for range n {
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
			sb.WriteString("?")
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
			sb.WriteString("?")
		}
	}
	return sb.String()
}

// hasTripleString reports whether any inline domain carries a triple-quoted string
// value, which can never render on a single line.
func hasTripleString(domains []parser.IInlineDomainContext) bool {
	for _, dom := range domains {
		content := dom.DomainContent()
		if content == nil || content.Expression() == nil {
			continue
		}
		for _, val := range content.Expression().AllExpressionValue() {
			if val.TRIPLE_STRING_LIT() != nil {
				return true
			}
		}
	}
	return false
}

// standaloneFieldModifier returns the optionality marker of a typeless override (key?),
// or "" when the field declares a type or no marker.
func standaloneFieldModifier(ctx parser.IFieldDefContext) string {
	if ctx.TypeRef() != nil || ctx.TypeModifiers() == nil {
		return ""
	}
	return "?"
}

// fieldNameColumn is the text occupying a field's name column: its name plus any
// standalone optionality marker, which glues to the name when the type is omitted.
func fieldNameColumn(ctx parser.IFieldDefContext) string {
	return ctx.IDENT().GetText() + standaloneFieldModifier(ctx)
}

func (f *formatter) formatDomainOmitsToString(
	omits []parser.IDomainOmitContext,
) string {
	var sb strings.Builder
	for _, om := range omits {
		sb.WriteString(" -@")
		sb.WriteString(om.IDENT().GetText())
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

func (f *formatter) formatInlineDomainsToString(
	domains []parser.IInlineDomainContext,
) string {
	domains = sortDomainLines(f, domains)
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

func (f *formatter) formatDomainContentToString(
	ctx parser.IDomainContentContext,
) string {
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

func (f *formatter) formatDefaultValueToString(ctx parser.IDefaultValueContext) string {
	if ev := ctx.ExpressionValue(); ev != nil {
		return f.formatExpressionValueToString(ev)
	}
	if arr := ctx.ArrayDefault(); arr != nil {
		return f.formatArrayDefaultToString(arr)
	}
	if st := ctx.StructDefault(); st != nil {
		return f.formatStructDefaultToString(st)
	}
	return ""
}

func (f *formatter) formatArrayDefaultToString(arr parser.IArrayDefaultContext) string {
	parts := make([]string, 0, len(arr.AllDefaultValue()))
	for _, el := range arr.AllDefaultValue() {
		parts = append(parts, f.formatDefaultValueToString(el))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (f *formatter) formatStructDefaultToString(
	st parser.IStructDefaultContext,
) string {
	fields := st.AllStructFieldDefault()
	if len(fields) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(fields))
	for _, sf := range fields {
		parts = append(
			parts,
			sf.IDENT().GetText()+" = "+f.formatDefaultValueToString(sf.DefaultValue()),
		)
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// formatFieldDefaultPretty renders a field default starting at column col, where
// indentLevel is the indent of the field declaration. Struct and array literals whose
// single-line form would overflow maxLineLen are broken across lines, with contents
// indented one level deeper and the closing bracket aligned to indentLevel. Scalars and
// literals that fit stay on one line.
func (f *formatter) formatFieldDefaultPretty(
	ctx parser.IFieldDefaultContext,
	col, indentLevel int,
) string {
	if ev := ctx.ExpressionValue(); ev != nil {
		return f.formatExpressionValueToString(ev)
	}
	if arr := ctx.ArrayDefault(); arr != nil {
		return f.formatArrayDefaultPretty(arr, col, indentLevel)
	}
	if st := ctx.StructDefault(); st != nil {
		return f.formatStructDefaultPretty(st, col, indentLevel)
	}
	return ""
}

func (f *formatter) formatDefaultValuePretty(
	ctx parser.IDefaultValueContext,
	col, indentLevel int,
) string {
	if ev := ctx.ExpressionValue(); ev != nil {
		return f.formatExpressionValueToString(ev)
	}
	if arr := ctx.ArrayDefault(); arr != nil {
		return f.formatArrayDefaultPretty(arr, col, indentLevel)
	}
	if st := ctx.StructDefault(); st != nil {
		return f.formatStructDefaultPretty(st, col, indentLevel)
	}
	return ""
}

func (f *formatter) formatStructDefaultPretty(
	st parser.IStructDefaultContext,
	col, indentLevel int,
) string {
	single := f.formatStructDefaultToString(st)
	if col+len(single) <= maxLineLen {
		return single
	}
	fields := st.AllStructFieldDefault()
	inner := strings.Repeat(indent, indentLevel+1)
	var b strings.Builder
	b.WriteString("{\n")
	for i, sf := range fields {
		prefix := sf.IDENT().GetText() + " = "
		val := f.formatDefaultValuePretty(
			sf.DefaultValue(),
			len(inner)+len(prefix),
			indentLevel+1,
		)
		b.WriteString(inner)
		b.WriteString(prefix)
		b.WriteString(val)
		if i < len(fields)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat(indent, indentLevel))
	b.WriteString("}")
	return b.String()
}

func (f *formatter) formatArrayDefaultPretty(
	arr parser.IArrayDefaultContext,
	col, indentLevel int,
) string {
	single := f.formatArrayDefaultToString(arr)
	if col+len(single) <= maxLineLen {
		return single
	}
	elems := arr.AllDefaultValue()
	inner := strings.Repeat(indent, indentLevel+1)
	var b strings.Builder
	b.WriteString("[\n")
	for i, el := range elems {
		b.WriteString(inner)
		b.WriteString(f.formatDefaultValuePretty(el, len(inner), indentLevel+1))
		if i < len(elems)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat(indent, indentLevel))
	b.WriteString("]")
	return b.String()
}

func (f *formatter) formatExpressionValueToString(
	ctx parser.IExpressionValueContext,
) string {
	if ctx.TRIPLE_STRING_LIT() != nil {
		return ctx.TRIPLE_STRING_LIT().GetText()
	}
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

func (f *formatter) formatQualifiedIdentToString(
	ctx parser.IQualifiedIdentContext,
) string {
	idents := ctx.AllIDENT()
	if len(idents) == 1 {
		return idents[0].GetText()
	}
	return idents[0].GetText() + "." + idents[1].GetText()
}

func (f *formatter) formatFieldWithBraces(
	inlineDomains []parser.IInlineDomainContext,
	domainOmits []parser.IDomainOmitContext,
	fieldBody parser.IFieldBodyContext,
) {
	f.writeLine(" {")
	f.currentIndent++

	// Inline domains convert to body lines here, so both kinds sort as one group.
	// Body-sourced lines keep their comment and watermark handling; the max keeps the
	// watermark monotone when sorting emits them out of source order.
	type fieldDomain struct {
		domainLine
		body bool
		stop antlr.Token
	}
	var domains []fieldDomain
	for _, dom := range inlineDomains {
		domains = append(domains, fieldDomain{domainLine: dom})
	}
	if fieldBody != nil {
		for _, dom := range fieldBody.AllDomain() {
			domains = append(
				domains,
				fieldDomain{domainLine: dom, body: true, stop: dom.GetStop()},
			)
		}
	}
	domains = sortDomainLines(f, domains)

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
		if dom.body {
			f.emitCommentsBefore(dom.GetStart().GetTokenIndex())
		}
		f.writeIndent()
		f.write("@")
		f.write(dom.IDENT().GetText())
		f.currentDomain = dom.IDENT().GetText()
		if dom.DomainContent() != nil {
			f.write(" ")
			f.formatDomainContentAligned(
				dom.DomainContent(),
				maxPrefixLen,
				1+len(dom.IDENT().GetText()),
			)
		}
		f.newline()
		if dom.body {
			f.lastTokenIdx = max(f.lastTokenIdx, dom.stop.GetTokenIndex())
		}
	}

	// Domain removals (-@name): field-level markers first, then body markers.
	for _, om := range domainOmits {
		f.writeIndent()
		f.write("-@")
		f.write(om.IDENT().GetText())
		f.newline()
	}
	if fieldBody != nil {
		for _, om := range fieldBody.AllDomainOmit() {
			f.emitCommentsBefore(om.GetStart().GetTokenIndex())
			f.writeIndent()
			f.write("-@")
			f.write(om.IDENT().GetText())
			f.newline()
			f.lastTokenIdx = om.GetStop().GetTokenIndex()
		}
	}

	f.currentIndent--
	f.writeIndent()
	f.writeLine("}")
}

// domainLine is the surface shared by the domain context kinds the formatter sorts:
// file-level, block, and inline field domains.
type domainLine interface {
	IDENT() antlr.TerminalNode
	DomainContent() parser.IDomainContentContext
	GetStart() antlr.Token
}

// domainCommand returns the leading command of a domain's content: an expression's
// IDENT, a block's first expression IDENT, or "" for a bare domain like @key.
func domainCommand(content parser.IDomainContentContext) string {
	if content == nil {
		return ""
	}
	if expr := content.Expression(); expr != nil {
		return expr.IDENT().GetText()
	}
	if blk := content.DomainBlock(); blk != nil {
		if exprs := blk.AllExpression(); len(exprs) > 0 {
			return exprs[0].IDENT().GetText()
		}
	}
	return ""
}

// sortDomainLines returns the lines ordered by domain name, then leading command
// ("@go marshal" before "@go migrate"), with a stable sort. A group with an attached
// comment keeps source order: emission tracks a comment watermark, and reordering
// across a comment would drop it.
func sortDomainLines[T domainLine](f *formatter, domains []T) []T {
	if len(domains) < 2 {
		return domains
	}
	for _, dom := range domains {
		hidden := f.tokens.GetHiddenTokensToLeft(
			dom.GetStart().GetTokenIndex(), hiddenChan,
		)
		for _, tok := range hidden {
			if tok.GetTokenIndex() > f.lastTokenIdx {
				return domains
			}
		}
	}
	sorted := slices.Clone(domains)
	slices.SortStableFunc(sorted, func(a, b T) int {
		if c := strings.Compare(a.IDENT().GetText(), b.IDENT().GetText()); c != 0 {
			return c
		}
		return strings.Compare(
			domainCommand(a.DomainContent()), domainCommand(b.DomainContent()),
		)
	})
	return sorted
}

func (f *formatter) formatDomains(domains []parser.IDomainContext) {
	if len(domains) == 0 {
		return
	}
	domains = sortDomainLines(f, domains)

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
		f.currentDomain = dom.IDENT().GetText()
		if dom.DomainContent() != nil {
			f.write(" ")
			f.formatDomainContentAligned(
				dom.DomainContent(),
				maxPrefixLen,
				1+len(dom.IDENT().GetText()),
			)
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
	values := ctx.AllExpressionValue()
	if len(values) == 1 {
		f.write(" ")
		f.formatSoleExpressionValue(values[0])
		return
	}
	for _, val := range values {
		f.write(" ")
		f.formatExpressionValue(val)
	}
}

// formatSoleExpressionValue emits an expression's only value, applying the prose layout
// rules: triple-quoted strings keep their content off the quote lines, and a doc string
// too long for its line converts to triple-quoted wrapped form. Both re-layouts
// preserve the value the analyzer reads, so they never apply to a value sharing its
// expression with others, where a line break would detach the neighbors.
func (f *formatter) formatSoleExpressionValue(ctx parser.IExpressionValueContext) {
	if ts := ctx.TRIPLE_STRING_LIT(); ts != nil {
		f.writeTripleString(ts.GetText())
		return
	}
	if s := ctx.STRING_LIT(); s != nil && f.currentDomain == "doc" {
		text := s.GetText()
		if f.currentLineLen()+utf8.RuneCountInString(text) > maxLineLen {
			if value, err := strconv.Unquote(text); err == nil &&
				!strings.Contains(value, tripleQuote) {
				f.writeWrappedDoc(value)
				return
			}
		}
	}
	f.formatExpressionValue(ctx)
}

// writeTripleString lays a triple-quoted string out canonically: nothing follows the
// opening quotes on their line, content lines sit one indent level past the domain
// line, and the closing quotes sit alone at the current indent. Doc content re-fills to
// the line limit; every consumer treats a single line break as a space, so neither the
// re-indent nor the re-fill changes the dedented value's meaning.
func (f *formatter) writeTripleString(text string) {
	content := text[len(tripleQuote) : len(text)-len(tripleQuote)]
	lines := strings.Split(content, "\n")
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	lines = lines[start:end]
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		n := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent < 0 || n < minIndent {
			minIndent = n
		}
	}
	dedented := make([]string, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			continue
		}
		dedented[i] = trimmed[minIndent:]
	}
	contentIndent := strings.Repeat(indent, f.currentIndent+1)
	f.write(tripleQuote)
	f.newline()
	if f.currentDomain == "doc" {
		f.writeDocLines(dedented, contentIndent)
	} else {
		for _, line := range dedented {
			if line == "" {
				f.newline()
				continue
			}
			f.writeLine(contentIndent + line)
		}
	}
	f.writeIndent()
	f.write(tripleQuote)
}

// writeDocLines emits dedented doc content lines at the given indent, re-filling each
// paragraph to the line limit. A paragraph containing an indented line is structured
// content (an example block) and stays verbatim — re-wrapping it would destroy its
// layout.
func (f *formatter) writeDocLines(lines []string, contentIndent string) {
	width := maxLineLen - len(contentIndent)
	var para []string
	flush := func() {
		if len(para) == 0 {
			return
		}
		var structured bool
		for _, l := range para {
			if l[0] == ' ' || l[0] == '\t' {
				structured = true
				break
			}
		}
		if structured {
			for _, l := range para {
				f.writeLine(contentIndent + l)
			}
		} else {
			words := strings.Fields(strings.Join(para, " "))
			line := words[0]
			for _, word := range words[1:] {
				if utf8.RuneCountInString(line)+1+
					utf8.RuneCountInString(word) <= width {
					line += " " + word
					continue
				}
				f.writeLine(contentIndent + line)
				line = word
			}
			f.writeLine(contentIndent + line)
		}
		para = para[:0]
	}
	for _, line := range lines {
		if line == "" {
			flush()
			f.newline()
			continue
		}
		para = append(para, line)
	}
	flush()
}

// writeWrappedDoc converts an over-long single-quoted doc string to the triple-quoted
// layout, filling content lines to the line limit one indent level in. Blank lines
// survive as paragraph breaks; every doc consumer treats a single line break as a
// space, so re-wrapping cannot change generated output.
func (f *formatter) writeWrappedDoc(value string) {
	contentIndent := strings.Repeat(indent, f.currentIndent+1)
	width := maxLineLen - len(contentIndent)
	f.write(tripleQuote)
	f.newline()
	for pi, para := range strings.Split(value, "\n\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			continue
		}
		if pi > 0 {
			f.newline()
		}
		line := words[0]
		for _, word := range words[1:] {
			if utf8.RuneCountInString(line)+1+utf8.RuneCountInString(word) <= width {
				line += " " + word
				continue
			}
			f.writeLine(contentIndent + line)
			line = word
		}
		f.writeLine(contentIndent + line)
	}
	f.writeIndent()
	f.write(tripleQuote)
}

func (f *formatter) formatExpressionValue(ctx parser.IExpressionValueContext) {
	if ctx.TRIPLE_STRING_LIT() != nil {
		f.write(ctx.TRIPLE_STRING_LIT().GetText())
	} else if ctx.STRING_LIT() != nil {
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
	f.write("?")
}

func (f *formatter) formatEnumDef(ctx parser.IEnumDefContext) {
	f.write(ctx.IDENT().GetText())
	f.write(" enum")
	if ctx.EXTENDS() != nil && ctx.TypeRefList() != nil {
		f.write(" extends ")
		refs := ctx.TypeRefList().AllTypeRef()
		for i, ref := range refs {
			if i > 0 {
				f.write(", ")
			}
			f.write(ref.GetText())
		}
	}

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
	for range padding {
		f.write(" ")
	}

	f.write(" = ")
	if ctx.INT_LIT() != nil {
		f.write(ctx.INT_LIT().GetText())
	} else if ctx.STRING_LIT() != nil {
		f.write(ctx.STRING_LIT().GetText())
	}
	if body := ctx.EnumValueBody(); body != nil {
		f.writeLine(" {")
		f.currentIndent++
		domains := sortDomainLines(f, body.AllDomain())
		var maxPrefixLen int
		for _, dom := range domains {
			prefixLen := 1 + len(dom.IDENT().GetText())
			if dom.DomainContent() != nil && dom.DomainContent().Expression() != nil {
				expr := dom.DomainContent().Expression()
				prefixLen += 1 + len(expr.IDENT().GetText())
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
			f.currentDomain = dom.IDENT().GetText()
			if dom.DomainContent() != nil {
				f.write(" ")
				f.formatDomainContentAligned(
					dom.DomainContent(),
					maxPrefixLen,
					1+len(dom.IDENT().GetText()),
				)
			}
			f.newline()
			f.lastTokenIdx = dom.GetStop().GetTokenIndex()
		}
		f.currentIndent--
		f.writeIndent()
		f.write("}")
	}
	f.newline()
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func (f *formatter) formatUnionDef(ctx parser.IUnionDefContext) {
	idents := ctx.AllIDENT()
	if len(idents) < 3 {
		return
	}
	f.write(idents[0].GetText())
	f.write(" union on ")
	f.write(idents[2].GetText())

	if ctx.EXTENDS() != nil && ctx.TypeRefList() != nil {
		f.write(" extends ")
		for i, tr := range ctx.TypeRefList().AllTypeRef() {
			if i > 0 {
				f.write(", ")
			}
			f.formatTypeRef(tr)
		}
	}

	body := ctx.UnionBody()
	if isEmptyUnionBody(body) {
		f.writeLine(" {}")
		f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
		return
	}

	f.writeLine(" {")
	f.currentIndent++
	f.formatUnionBody(body)
	f.currentIndent--
	f.writeLine("}")
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

func isEmptyUnionBody(ctx parser.IUnionBodyContext) bool {
	return ctx == nil || (len(ctx.AllUnionVariant()) == 0 && len(ctx.AllDomain()) == 0)
}

func (f *formatter) formatUnionBody(ctx parser.IUnionBodyContext) {
	variants := ctx.AllUnionVariant()
	domains := ctx.AllDomain()

	maxNameLen := 0
	for _, v := range variants {
		if named, ok := v.(*parser.NamedVariantContext); ok {
			if nameLen := len(named.VariantName().GetText()); nameLen > maxNameLen {
				maxNameLen = nameLen
			}
		}
	}

	for _, v := range variants {
		f.emitCommentsBefore(v.GetStart().GetTokenIndex())
		f.formatUnionVariant(v, maxNameLen)
	}

	if len(variants) > 0 && len(domains) > 0 {
		f.newline()
	}
	f.formatDomains(domains)
}

func (f *formatter) formatUnionVariant(ctx parser.IUnionVariantContext, nameWidth int) {
	switch v := ctx.(type) {
	case *parser.NamedVariantContext:
		f.formatNamedVariant(v, nameWidth)
	case *parser.InlineVariantContext:
		f.formatInlineVariant(v)
	}
}

func (f *formatter) formatNamedVariant(ctx *parser.NamedVariantContext, nameWidth int) {
	f.writeIndent()
	name := ctx.VariantName().GetText()
	f.write(name)
	f.writePadding(nameWidth - len(name))
	f.write(" ")
	f.write(f.formatTypeRefToString(ctx.TypeRef()))

	body := ctx.UnionVariantBody()
	if body == nil || len(body.AllDomain()) == 0 {
		f.newline()
		f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
		return
	}

	f.writeLine(" {")
	f.currentIndent++
	f.formatDomains(body.AllDomain())
	f.currentIndent--
	f.writeIndent()
	f.writeLine("}")
	f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
}

// formatInlineVariant renders an inline variant body with struct-body formatting: name,
// optional extends clause, then fields and domains.
func (f *formatter) formatInlineVariant(ctx *parser.InlineVariantContext) {
	f.writeIndent()
	f.write(ctx.VariantName().GetText())
	if ctx.EXTENDS() != nil && ctx.TypeRefList() != nil {
		f.write(" extends ")
		for i, tr := range ctx.TypeRefList().AllTypeRef() {
			if i > 0 {
				f.write(", ")
			}
			f.formatTypeRef(tr)
		}
	}

	body := ctx.StructBody()
	if body == nil ||
		(len(body.AllFieldDef()) == 0 && len(body.AllDomain()) == 0 &&
			len(body.AllFieldOmit()) == 0 && len(body.AllActionDef()) == 0) {
		f.writeLine(" {}")
		f.lastTokenIdx = ctx.GetStop().GetTokenIndex()
		return
	}

	f.writeLine(" {")
	f.currentIndent++
	f.formatStructBody(body)
	f.currentIndent--
	f.writeIndent()
	f.writeLine("}")
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
