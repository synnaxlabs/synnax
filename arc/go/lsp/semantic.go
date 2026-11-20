// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"context"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/lsp"
	"github.com/synnaxlabs/x/set"
	"go.lsp.dev/protocol"
)

const (
	SemanticTokenTypeKeyword = iota
	SemanticTokenTypeType
	SemanticTokenTypeOperator
	SemanticTokenTypeVariable
	SemanticTokenTypeString
	SemanticTokenTypeNumber
	SemanticTokenTypeComment
	SemanticTokenTypeFunction
	semanticTokenTypeParameter
	SemanticTokenTypeChannel
	semanticTokenTypeSequence
	semanticTokenTypeStage
	semanticTokenTypeBlock
	SemanticTokenTypeStatefulVariable
	SemanticTokenTypeEdgeConditional
	SemanticTokenTypeEdgeContinuous
	SemanticTokenTypeConstant
	SemanticTokenTypeInput
	semanticTokenTypeOutput
	semanticTokenTypeNamespace
	semanticTokenTypeStringRaw
	semanticTokenTypeStringPlaceholder
	semanticTokenTypeChannelVariable
	semanticTokenTypeBoolean
)

var semanticTokenTypes = []string{
	"keyword",
	"type",
	"operator",
	"variable",
	"string",
	"number",
	"comment",
	"function",
	"parameter",
	"channel",
	"sequence",
	"stage",
	"block",
	"statefulVariable",
	"edgeConditional",
	"edgeContinuous",
	"constant",
	"input",
	"output",
	"namespace",
	"stringRaw",
	"stringPlaceholder",
	"channelVariable",
	"boolean",
}

func (s *Server) SemanticTokensFull(
	ctx context.Context,
	params *protocol.SemanticTokensParams,
) (*protocol.SemanticTokens, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return &protocol.SemanticTokens{Data: []uint32{}}, nil
	}
	tokens := extractSemanticTokens(ctx, doc.Content, doc.IR)
	return &protocol.SemanticTokens{Data: tokens}, nil
}

func extractSemanticTokens(ctx context.Context, content string, docIR ir.IR) []uint32 {
	allTokens := tokenizeContent(content)
	importIdents := importContextIdents(allTokens)
	var tokens []lsp.Token
	// Track prev/next token types so classifyToken can handle qualified names (e.g.,
	// control.set_authority).
	for i, t := range allTokens {
		if t.GetTokenType() == antlr.TokenEOF {
			continue
		}
		if t.GetTokenType() == parser.ArcLexerSTR_LITERAL ||
			t.GetTokenType() == parser.ArcLexerSTR_LITERAL_MULTI {
			tokens = append(tokens, expandStringToken(ctx, t, docIR)...)
			continue
		}
		if isRoutingKey(allTokens, i) {
			continue
		}
		var prevType, nextType int
		if i > 0 {
			prevType = allTokens[i-1].GetTokenType()
		}
		if i+1 < len(allTokens) {
			nextType = allTokens[i+1].GetTokenType()
		}
		qualifier := ""
		if prevType == parser.ArcLexerDOT && i >= 2 {
			qualifier = allTokens[i-2].GetText()
		}
		tokenType := classifyToken(
			ctx,
			t,
			prevType,
			nextType,
			importIdents.Contains(i),
			docIR,
			qualifier,
		)
		if tokenType == nil {
			continue
		}
		tokens = appendTokenPerLine(tokens, t, *tokenType)
	}
	return lsp.EncodeSemanticTokens(tokens)
}

// isRoutingKey reports whether the token at i is a TRUE or FALSE naming a
// routing-table branch (`true:`) rather than a bool literal.
func isRoutingKey(tokens []antlr.Token, i int) bool {
	switch tokens[i].GetTokenType() {
	case parser.ArcLexerTRUE, parser.ArcLexerFALSE:
	default:
		return false
	}
	for j := i + 1; j < len(tokens); j++ {
		if tokens[j].GetChannel() != antlr.TokenDefaultChannel {
			continue
		}
		return tokens[j].GetTokenType() == parser.ArcLexerCOLON
	}
	return false
}

// importContextIdents returns the token indexes that name modules inside an
// `import` statement (path heads, dotted path tails, and aliases — in both
// bare and parenthesized block form). The IMPORT keyword itself is excluded.
// Indexes are into the original tokens slice; the scan skips hidden-channel
// tokens (whitespace, comments) so it isn't fooled by the WS that separates
// `import` from the first module name.
func importContextIdents(tokens []antlr.Token) set.Set[int] {
	out := make(set.Set[int])
	visible := defaultChannelTokens(tokens)
	for vi := 0; vi < len(visible); vi++ {
		if visible[vi].tt != parser.ArcLexerIMPORT {
			continue
		}
		vi++
		if vi >= len(visible) {
			break
		}
		if visible[vi].tt == parser.ArcLexerLPAREN {
			vi = addImportBlockIdents(visible, vi+1, out) - 1
			continue
		}
		vi = addBareImportIdents(visible, vi, out) - 1
	}
	return out
}

// indexedToken is the type of a token and its index into the full token slice.
type indexedToken struct {
	idx int
	tt  int
}

func defaultChannelTokens(tokens []antlr.Token) []indexedToken {
	visible := make([]indexedToken, 0, len(tokens))
	for i, t := range tokens {
		if t.GetChannel() != antlr.TokenDefaultChannel {
			continue
		}
		visible = append(visible, indexedToken{i, t.GetTokenType()})
	}
	return visible
}

// addImportBlockIdents adds the module names of the parenthesized import block whose
// contents start at vi, and returns the index after its closing paren.
func addImportBlockIdents(visible []indexedToken, vi int, out set.Set[int]) int {
	depth := 1
	for vi < len(visible) && depth > 0 {
		switch visible[vi].tt {
		case parser.ArcLexerLPAREN:
			depth++
		case parser.ArcLexerRPAREN:
			depth--
		case parser.ArcLexerIDENTIFIER, parser.ArcLexerAUTHORITY:
			out.Add(visible[vi].idx)
		}
		vi++
	}
	return vi
}

// addBareImportIdents adds the module names of the bare import that starts at vi,
// IDENT|AUTHORITY (DOT IDENT)* (AS IDENT)?, and returns the index after it.
func addBareImportIdents(visible []indexedToken, vi int, out set.Set[int]) int {
	const (
		expectHead = iota
		afterHead
		expectTail
		expectAlias
	)
	state := expectHead
	for ; vi < len(visible); vi++ {
		tt := visible[vi].tt
		switch state {
		case expectHead:
			if tt != parser.ArcLexerIDENTIFIER && tt != parser.ArcLexerAUTHORITY {
				return vi
			}
			out.Add(visible[vi].idx)
			state = afterHead
		case afterHead:
			switch tt {
			case parser.ArcLexerDOT:
				state = expectTail
			case parser.ArcLexerAS:
				state = expectAlias
			default:
				return vi
			}
		case expectTail:
			if tt != parser.ArcLexerIDENTIFIER {
				return vi
			}
			out.Add(visible[vi].idx)
			state = afterHead
		case expectAlias:
			if tt != parser.ArcLexerIDENTIFIER {
				return vi
			}
			out.Add(visible[vi].idx)
			return vi + 1
		}
	}
	return vi
}

// isImportAlias reports whether name is bound as a module alias in the
// program root's scope tree.
func isImportAlias(docIR ir.IR, name string) bool {
	if docIR.Symbols == nil {
		return false
	}
	child := docIR.Symbols.FindChild(name)
	return child != nil && child.Kind == symbol.KindModuleAlias
}

// appendTokenPerLine emits one LSP semantic token per source line covered by t.
// Monaco does not render a single semantic token whose length crosses a newline,
// so multi-line ANTLR tokens (e.g. STR_LITERAL_MULTI spanning several lines)
// must be split into per-line entries to receive consistent coloring across the
// whole span. For single-line tokens this collapses to one append, matching the
// prior behavior.
func appendTokenPerLine(
	tokens []lsp.Token,
	t antlr.Token,
	tokenType uint32,
) []lsp.Token {
	return appendTextTokenPerLine(
		tokens,
		t.GetText(),
		uint32(t.GetLine()-1),
		uint32(t.GetColumn()),
		tokenType,
	)
}

func appendTextTokenPerLine(
	tokens []lsp.Token,
	text string,
	line, startChar, tokenType uint32,
) []lsp.Token {
	lineStart := 0
	for i := range len(text) {
		if text[i] != '\n' {
			continue
		}
		if i > lineStart {
			tokens = append(tokens, lsp.Token{
				Line:      line,
				StartChar: startChar,
				Length:    uint32(utf8.RuneCountInString(text[lineStart:i])),
				TokenType: tokenType,
			})
		}
		line++
		startChar = 0
		lineStart = i + 1
	}
	if lineStart < len(text) {
		tokens = append(tokens, lsp.Token{
			Line:      line,
			StartChar: startChar,
			Length:    uint32(utf8.RuneCountInString(text[lineStart:])),
			TokenType: tokenType,
		})
	}
	return tokens
}

func classifyToken(
	ctx context.Context,
	t antlr.Token,
	prevTokenType, nextTokenType int,
	inImport bool,
	docIR ir.IR,
	qualifier string,
) *uint32 {
	return classifyTokenAt(
		ctx,
		t,
		prevTokenType,
		nextTokenType,
		inImport,
		docIR,
		t.GetLine(),
		t.GetColumn(),
		qualifier,
	)
}

// Variant with explicit (line1, col0) for tokens lexed out of a sub-string.
// qualifier is the identifier preceding the DOT when t is a member access
// (e.g. "time" in "time.now"); it is empty otherwise.
func classifyTokenAt(
	ctx context.Context,
	t antlr.Token,
	prevTokenType, nextTokenType int,
	inImport bool,
	docIR ir.IR,
	line1, col0 int,
	qualifier string,
) *uint32 {
	antlrType := t.GetTokenType()
	// Identifiers (and AUTHORITY used as an importable name) inside an import
	// statement are module references, not variables.
	if inImport &&
		(antlrType == parser.ArcLexerIDENTIFIER || antlrType == parser.ArcLexerAUTHORITY) {
		tokenType := uint32(semanticTokenTypeNamespace)
		return &tokenType
	}
	// An IDENTIFIER immediately following a numeric literal is that literal's unit
	// suffix (e.g. "min" in "3min"), which the parser attaches to the literal via
	// an adjacency predicate rather than treating as a symbol reference. Emit no
	// token and defer to the grammar; otherwise a unit whose name collides with a
	// builtin (e.g. the "min" function) would be colored as that symbol, unlike
	// units such as "s" or "h" that resolve to nothing.
	if antlrType == parser.ArcLexerIDENTIFIER &&
		(prevTokenType == parser.ArcLexerINTEGER_LITERAL ||
			prevTokenType == parser.ArcLexerFLOAT_LITERAL) {
		return nil
	}
	// IDENTIFIER after DOT is the member part of a qualified name (e.g.,
	// "set_authority" in "control.set_authority"). Color it by its symbol
	// kind only when the qualifier resolves to an imported module that
	// actually has this member. An unimported or unknown qualifier yields no
	// token, so the highlight matches the analyzer's "undefined" view rather
	// than implying a valid reference.
	if antlrType == parser.ArcLexerIDENTIFIER && prevTokenType == parser.ArcLexerDOT {
		if docIR.Symbols == nil || qualifier == "" {
			return nil
		}
		scope := findScopeAtInternalPosition(
			docIR.Symbols,
			position{Line: line1, Col: col0},
		)
		mod, err := scope.Resolve(ctx, qualifier, symbol.WithoutUsageTracking)
		if err != nil {
			return nil
		}
		member, err := mod.Resolve(ctx, t.GetText(), symbol.WithoutUsageTracking)
		if err != nil {
			return nil
		}
		return mapSymbolKind(member.Kind)
	}
	// IDENTIFIER followed by DOT is a qualifier; if it names an imported
	// module (directly or via alias), color it as a namespace so callers can
	// visually distinguish module-qualified uses from local variables.
	if antlrType == parser.ArcLexerIDENTIFIER && nextTokenType == parser.ArcLexerDOT &&
		isImportAlias(docIR, t.GetText()) {
		tokenType := uint32(semanticTokenTypeNamespace)
		return &tokenType
	}
	if antlrType == parser.ArcLexerIDENTIFIER && docIR.Symbols != nil {
		return classifyIdentifierAt(ctx, t.GetText(), line1, col0, docIR.Symbols)
	}
	return mapLexerTokenType(antlrType)
}

func classifyIdentifierAt(
	ctx context.Context,
	name string,
	line1, col0 int,
	rootScope *symbol.Symbol,
) *uint32 {
	scope := findScopeAtInternalPosition(rootScope, position{Line: line1, Col: col0})
	sym, err := scope.Resolve(ctx, name, symbol.WithoutUsageTracking)
	if err != nil || sym == nil {
		return nil
	}
	if t := classifyVarKind(sym); t != nil {
		return t
	}
	return mapSymbolKind(sym.Kind)
}

func classifyVarKind(sym *symbol.Symbol) *uint32 {
	if !sym.IsChannelReadWrite() && !sym.IsReactive() {
		return nil
	}
	tokenType := uint32(semanticTokenTypeChannelVariable)
	return &tokenType
}

func mapSymbolKind(kind symbol.Kind) *uint32 {
	var tokenType uint32
	switch kind {
	case symbol.KindFunction:
		tokenType = SemanticTokenTypeFunction
	case symbol.KindVariable:
		tokenType = SemanticTokenTypeVariable
	case symbol.KindConstant:
		tokenType = SemanticTokenTypeConstant
	case symbol.KindStatefulVariable:
		tokenType = SemanticTokenTypeStatefulVariable
	case symbol.KindInput:
		tokenType = SemanticTokenTypeInput
	case symbol.KindOutput:
		tokenType = semanticTokenTypeOutput
	case symbol.KindChannel:
		tokenType = SemanticTokenTypeChannel
	case symbol.KindSequence, symbol.KindStage:
		// Sequence and stage names share the function highlight: they are
		// declarations of named, callable scopes, and using the function
		// color keeps them visually consistent with `func` declarations.
		tokenType = SemanticTokenTypeFunction
	case symbol.KindBlock, symbol.KindLoop:
		tokenType = semanticTokenTypeBlock
	case symbol.KindLoopVariable, symbol.KindModule, symbol.KindModuleAlias,
		symbol.KindAmbient:
		tokenType = SemanticTokenTypeVariable
	}
	return &tokenType
}

func mapLexerTokenType(antlrType int) *uint32 {
	var tokenType uint32
	switch antlrType {
	case parser.ArcLexerFUNC, parser.ArcLexerIF,
		parser.ArcLexerELSE, parser.ArcLexerRETURN,
		parser.ArcLexerFOR, parser.ArcLexerBREAK, parser.ArcLexerCONTINUE,
		parser.ArcLexerSEQUENCE, parser.ArcLexerSTAGE,
		parser.ArcLexerNEXT, parser.ArcLexerAND, parser.ArcLexerOR,
		parser.ArcLexerNOT, parser.ArcLexerAUTHORITY,
		parser.ArcLexerIMPORT, parser.ArcLexerEXPORT, parser.ArcLexerAS:
		tokenType = SemanticTokenTypeKeyword
	case parser.ArcLexerI8, parser.ArcLexerI16, parser.ArcLexerI32, parser.ArcLexerI64,
		parser.ArcLexerU8, parser.ArcLexerU16, parser.ArcLexerU32, parser.ArcLexerU64,
		parser.ArcLexerF32, parser.ArcLexerF64, parser.ArcLexerSTR,
		parser.ArcLexerBOOL,
		parser.ArcLexerSERIES,
		parser.ArcLexerCHAN:
		tokenType = SemanticTokenTypeType
	case parser.ArcLexerTRANSITION:
		tokenType = SemanticTokenTypeEdgeConditional
	case parser.ArcLexerARROW:
		tokenType = SemanticTokenTypeEdgeContinuous
	case parser.ArcLexerDECLARE, parser.ArcLexerSTATE_DECLARE, parser.ArcLexerASSIGN,
		parser.ArcLexerPLUS, parser.ArcLexerMINUS, parser.ArcLexerSTAR,
		parser.ArcLexerSLASH, parser.ArcLexerPERCENT, parser.ArcLexerCARET,
		parser.ArcLexerEQ, parser.ArcLexerNEQ, parser.ArcLexerLT, parser.ArcLexerGT,
		parser.ArcLexerLEQ, parser.ArcLexerGEQ:
		tokenType = SemanticTokenTypeOperator
	case parser.ArcLexerSTR_LITERAL, parser.ArcLexerSTR_LITERAL_MULTI:
		tokenType = SemanticTokenTypeString
	case parser.ArcLexerINTEGER_LITERAL, parser.ArcLexerFLOAT_LITERAL:
		tokenType = SemanticTokenTypeNumber
	case parser.ArcLexerTRUE, parser.ArcLexerFALSE:
		tokenType = semanticTokenTypeBoolean
	case parser.ArcLexerSINGLE_LINE_COMMENT, parser.ArcLexerMULTI_LINE_COMMENT:
		tokenType = SemanticTokenTypeComment
	case parser.ArcLexerIDENTIFIER:
		tokenType = SemanticTokenTypeVariable
	default:
		return nil
	}
	return &tokenType
}

// expandStringToken emits LSP semantic tokens for a STR_LITERAL or
// STR_LITERAL_MULTI. Format strings (f/rf prefix) with placeholders have their
// {...} segments expanded with classified inner expression tokens; everything
// else collapses to a single string-colored token.
func expandStringToken(ctx context.Context, t antlr.Token, docIR ir.IR) []lsp.Token {
	text := t.GetText()
	body, flags, ok := literal.StripQuotes(text)
	prefixLen := 0
	for prefixLen < 2 && prefixLen < len(text) && (text[prefixLen] == 'r' || text[prefixLen] == 'f') {
		prefixLen++
	}
	if !ok || !flags.Format {
		return plainStringTokens(t, prefixLen)
	}
	segs, err := literal.FmtStrSplit(body, flags)
	if err != nil || !literal.FmtStrHasPlaceholder(segs) {
		return plainStringTokens(t, prefixLen)
	}
	const delimLen = 1
	e := fmtStrEmitter{
		ctx:     ctx,
		docIR:   docIR,
		text:    text,
		bodyOff: prefixLen + delimLen,
		at: textPositions(protocol.Position{
			Line:      uint32(t.GetLine() - 1),
			Character: uint32(t.GetColumn()),
		}, text),
	}
	e.emit(0, prefixLen, SemanticTokenTypeFunction)
	e.emit(prefixLen, e.bodyOff, SemanticTokenTypeString)
	for _, seg := range segs {
		if !seg.IsPlaceholder {
			e.emit(seg.Start+e.bodyOff, seg.End+e.bodyOff, SemanticTokenTypeString)
			continue
		}
		// A brace can be written as an escape, so the open brace runs up to the text,
		// and the spec and close brace run from the end of the text.
		textStart, textEnd := seg.Offsets[0], seg.Offsets[len(seg.Text)]
		e.emit(
			seg.Start+e.bodyOff,
			textStart+e.bodyOff,
			semanticTokenTypeStringPlaceholder,
		)
		e.emitInner(seg)
		e.emit(textEnd+e.bodyOff, seg.End+e.bodyOff, semanticTokenTypeStringPlaceholder)
	}
	e.emit(len(text)-delimLen, len(text), SemanticTokenTypeString)
	return e.tokens
}

// plainStringTokens colors the prefix of t as a function and the rest as a string.
func plainStringTokens(t antlr.Token, prefixLen int) []lsp.Token {
	text := t.GetText()
	var out []lsp.Token
	line, col := uint32(t.GetLine()-1), uint32(t.GetColumn())
	if prefixLen > 0 {
		out = appendTextTokenPerLine(
			out,
			text[:prefixLen],
			line,
			col,
			SemanticTokenTypeFunction,
		)
		col += uint32(prefixLen)
	}
	return appendTextTokenPerLine(
		out,
		text[prefixLen:],
		line,
		col,
		SemanticTokenTypeString,
	)
}

// fmtStrEmitter emits the semantic tokens of a format string literal. at holds the
// source position of each byte of text, and bodyOff is the offset of the body in text.
type fmtStrEmitter struct {
	ctx     context.Context
	docIR   ir.IR
	text    string
	at      []protocol.Position
	bodyOff int
	tokens  []lsp.Token
}

func (e *fmtStrEmitter) emit(a, b int, tt uint32) {
	if a >= b {
		return
	}
	e.tokens = appendTextTokenPerLine(
		e.tokens,
		e.text[a:b],
		e.at[a].Line,
		e.at[a].Character,
		tt,
	)
}

// emitInner emits the classified tokens of a placeholder. The placeholder's text has
// its escapes applied, as the compiler parses it, and Offsets maps each of its bytes
// back to the source.
func (e *fmtStrEmitter) emitInner(seg literal.FmtStrSegment) {
	inner := tokenizeContent(seg.Text)
	runeStart := make([]int, 0, len(seg.Text)+1)
	for i := range seg.Text {
		runeStart = append(runeStart, i)
	}
	runeStart = append(runeStart, len(seg.Text))
	for i, it := range inner {
		if it.GetTokenType() == antlr.TokenEOF {
			continue
		}
		var prev, next int
		if i > 0 {
			prev = inner[i-1].GetTokenType()
		}
		if i+1 < len(inner) {
			next = inner[i+1].GetTokenType()
		}
		qualifier := ""
		if prev == parser.ArcLexerDOT && i >= 2 {
			qualifier = inner[i-2].GetText()
		}
		a := e.bodyOff + seg.Offsets[runeStart[it.GetStart()]]
		b := e.bodyOff + seg.Offsets[runeStart[it.GetStop()+1]]
		tt := classifyTokenAt(
			e.ctx,
			it,
			prev,
			next,
			false,
			e.docIR,
			int(e.at[a].Line)+1,
			int(e.at[a].Character),
			qualifier,
		)
		if tt != nil {
			e.emit(a, b, *tt)
		}
	}
}

// textPositions returns the source position of each byte of text and of its end, with
// columns in runes as ANTLR counts them. base is the position of text[0].
func textPositions(base protocol.Position, text string) []protocol.Position {
	at := make([]protocol.Position, len(text)+1)
	p := base
	for i, r := range text {
		for j := i; j < i+utf8.RuneLen(r); j++ {
			at[j] = p
		}
		if r == '\n' {
			p = protocol.Position{Line: p.Line + 1}
		} else {
			p.Character++
		}
	}
	at[len(text)] = p
	return at
}
