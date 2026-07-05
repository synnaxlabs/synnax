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

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
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
	SemanticTokenTypeParameter
	SemanticTokenTypeChannel
	SemanticTokenTypeSequence
	SemanticTokenTypeStage
	SemanticTokenTypeBlock
	SemanticTokenTypeStatefulVariable
	SemanticTokenTypeEdgeConditional
	SemanticTokenTypeEdgeContinuous
	SemanticTokenTypeConstant
	SemanticTokenTypeConfig
	SemanticTokenTypeInput
	SemanticTokenTypeOutput
	SemanticTokenTypeNamespace
	SemanticTokenTypeStringRaw
	SemanticTokenTypeStringPlaceholder
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
	"config",
	"input",
	"output",
	"namespace",
	"stringRaw",
	"stringPlaceholder",
}

func (s *Server) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
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
	// Track prev/next token types so classifyToken can handle qualified names (e.g., control.set_authority).
	for i, t := range allTokens {
		if t.GetTokenType() == antlr.TokenEOF {
			continue
		}
		if t.GetTokenType() == parser.ArcLexerSTR_LITERAL ||
			t.GetTokenType() == parser.ArcLexerSTR_LITERAL_MULTI {
			tokens = append(tokens, expandStringToken(ctx, t, docIR)...)
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
		tokenType := classifyToken(ctx, t, prevType, nextType, importIdents.Contains(i), docIR, qualifier)
		if tokenType == nil {
			continue
		}
		tokens = appendTokenPerLine(tokens, t, *tokenType)
	}
	return lsp.EncodeSemanticTokens(tokens)
}

// importContextIdents returns the token indexes that name modules inside an
// `import` statement (path heads, dotted path tails, and aliases — in both
// bare and parenthesized block form). The IMPORT keyword itself is excluded.
// Indexes are into the original tokens slice; the scan skips hidden-channel
// tokens (whitespace, comments) so it isn't fooled by the WS that separates
// `import` from the first module name.
func importContextIdents(tokens []antlr.Token) set.Set[int] {
	out := make(set.Set[int])
	type idxTok struct {
		idx int
		tt  int
	}
	visible := make([]idxTok, 0, len(tokens))
	for i, t := range tokens {
		if t.GetChannel() != antlr.TokenDefaultChannel {
			continue
		}
		visible = append(visible, idxTok{i, t.GetTokenType()})
	}
	for vi := 0; vi < len(visible); vi++ {
		if visible[vi].tt != parser.ArcLexerIMPORT {
			continue
		}
		vi++
		if vi >= len(visible) {
			break
		}
		if visible[vi].tt == parser.ArcLexerLPAREN {
			depth := 1
			vi++
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
			vi--
			continue
		}
		// Bare form: IDENT|AUTHORITY (DOT IDENT)* (AS IDENT)?
		const (
			expectHead = iota
			afterHead
			expectTail
			expectAlias
		)
		state := expectHead
		for vi < len(visible) {
			tt := visible[vi].tt
			done := false
			switch state {
			case expectHead:
				if tt == parser.ArcLexerIDENTIFIER || tt == parser.ArcLexerAUTHORITY {
					out.Add(visible[vi].idx)
					state = afterHead
					vi++
				} else {
					done = true
				}
			case afterHead:
				switch tt {
				case parser.ArcLexerDOT:
					state = expectTail
					vi++
				case parser.ArcLexerAS:
					state = expectAlias
					vi++
				default:
					done = true
				}
			case expectTail:
				if tt == parser.ArcLexerIDENTIFIER {
					out.Add(visible[vi].idx)
					state = afterHead
					vi++
				} else {
					done = true
				}
			case expectAlias:
				if tt == parser.ArcLexerIDENTIFIER {
					out.Add(visible[vi].idx)
					vi++
				}
				done = true
			}
			if done {
				break
			}
		}
		vi--
	}
	return out
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
func appendTokenPerLine(tokens []lsp.Token, t antlr.Token, tokenType uint32) []lsp.Token {
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
	for i := 0; i < len(text); i++ {
		if text[i] != '\n' {
			continue
		}
		if i > lineStart {
			tokens = append(tokens, lsp.Token{
				Line:      line,
				StartChar: startChar,
				Length:    uint32(i - lineStart),
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
			Length:    uint32(len(text) - lineStart),
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
	return classifyTokenAt(ctx, t, prevTokenType, nextTokenType, inImport, docIR, t.GetLine(), t.GetColumn(), qualifier)
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
	if inImport && (antlrType == parser.ArcLexerIDENTIFIER || antlrType == parser.ArcLexerAUTHORITY) {
		tokenType := uint32(SemanticTokenTypeNamespace)
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
		scope := findScopeAtInternalPosition(docIR.Symbols, position{Line: line1, Col: col0})
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
		tokenType := uint32(SemanticTokenTypeNamespace)
		return &tokenType
	}
	if antlrType == parser.ArcLexerIDENTIFIER && docIR.Symbols != nil {
		return classifyIdentifierAt(ctx, t.GetText(), line1, col0, docIR.Symbols)
	}
	return mapLexerTokenType(antlrType)
}

func classifyIdentifierAt(ctx context.Context, name string, line1, col0 int, rootScope *symbol.Symbol) *uint32 {
	scope := findScopeAtInternalPosition(rootScope, position{Line: line1, Col: col0})
	sym, err := scope.Resolve(ctx, name, symbol.WithoutUsageTracking)
	if err != nil || sym == nil {
		return nil
	}
	return mapSymbolKind(sym.Kind)
}

func mapSymbolKind(kind symbol.Kind) *uint32 {
	var tokenType uint32
	switch kind {
	case symbol.KindFunction:
		tokenType = SemanticTokenTypeFunction
	case symbol.KindVariable:
		tokenType = SemanticTokenTypeVariable
	case symbol.KindConstant, symbol.KindGlobalConstant:
		tokenType = SemanticTokenTypeConstant
	case symbol.KindStatefulVariable:
		tokenType = SemanticTokenTypeStatefulVariable
	case symbol.KindConfig:
		tokenType = SemanticTokenTypeConfig
	case symbol.KindInput:
		tokenType = SemanticTokenTypeInput
	case symbol.KindOutput:
		tokenType = SemanticTokenTypeOutput
	case symbol.KindChannel:
		tokenType = SemanticTokenTypeChannel
	case symbol.KindSequence, symbol.KindStage:
		// Sequence and stage names share the function highlight: they are
		// declarations of named, callable scopes, and using the function
		// color keeps them visually consistent with `func` declarations.
		tokenType = SemanticTokenTypeFunction
	case symbol.KindBlock, symbol.KindLoop:
		tokenType = SemanticTokenTypeBlock
	case symbol.KindLoopVariable:
		tokenType = SemanticTokenTypeVariable
	default:
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
		parser.ArcLexerIMPORT, parser.ArcLexerAS:
		tokenType = SemanticTokenTypeKeyword
	case parser.ArcLexerI8, parser.ArcLexerI16, parser.ArcLexerI32, parser.ArcLexerI64,
		parser.ArcLexerU8, parser.ArcLexerU16, parser.ArcLexerU32, parser.ArcLexerU64,
		parser.ArcLexerF32, parser.ArcLexerF64, parser.ArcLexerSTR,
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
	fallback := func() []lsp.Token {
		var out []lsp.Token
		line, col := uint32(t.GetLine()-1), uint32(t.GetColumn())
		if prefixLen > 0 {
			out = appendTextTokenPerLine(out, text[:prefixLen], line, col, SemanticTokenTypeFunction)
			col += uint32(prefixLen)
		}
		out = appendTextTokenPerLine(out, text[prefixLen:], line, col, SemanticTokenTypeString)
		return out
	}
	if !ok || !flags.Format {
		return fallback()
	}
	segs, err := literal.FmtStrParse(body)
	if err != nil {
		return fallback()
	}
	if !literal.FmtStrHasPlaceholder(segs) {
		return fallback()
	}
	const delimLen = 1
	bodyOff := prefixLen + delimLen
	cursor := diagnostics.Position{Line: t.GetLine() - 1, Col: t.GetColumn()}
	prevOff := 0
	posOf := func(off int) (uint32, uint32) {
		cursor = cursor.Advance(text[prevOff:], off-prevOff)
		prevOff = off
		return uint32(cursor.Line), uint32(cursor.Col)
	}
	var tokens []lsp.Token
	emit := func(a, b int, tt uint32) {
		if a >= b {
			return
		}
		line, col := posOf(a)
		tokens = appendTextTokenPerLine(tokens, text[a:b], line, col, tt)
	}
	emitInner := func(a, b int) {
		inner := tokenizeContent(text[a:b])
		baseLine, baseCol := posOf(a)
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
			relLine, relCol := uint32(it.GetLine()-1), uint32(it.GetColumn())
			absLine, absCol := baseLine+relLine, relCol
			if relLine == 0 {
				absCol = baseCol + relCol
			}
			tt := classifyTokenAt(ctx, it, prev, next, false, docIR, int(absLine)+1, int(absCol), qualifier)
			if tt == nil {
				continue
			}
			tokens = appendTextTokenPerLine(tokens, it.GetText(), absLine, absCol, *tt)
		}
	}
	emit(0, prefixLen, SemanticTokenTypeFunction)
	emit(prefixLen, bodyOff, SemanticTokenTypeString)
	for _, seg := range segs {
		if !seg.IsPlaceholder {
			emit(seg.Start+bodyOff, seg.End+bodyOff, SemanticTokenTypeString)
			continue
		}
		emit(seg.Start+bodyOff, seg.Start+bodyOff+1, SemanticTokenTypeStringPlaceholder)
		exprEnd := seg.End - 1
		if seg.SpecOffset >= 0 {
			exprEnd = seg.SpecOffset
		}
		emitInner(seg.Start+bodyOff+1, exprEnd+bodyOff)
		if seg.SpecOffset >= 0 {
			emit(seg.SpecOffset+bodyOff, seg.End-1+bodyOff, SemanticTokenTypeStringPlaceholder)
		}
		emit(seg.End-1+bodyOff, seg.End+bodyOff, SemanticTokenTypeStringPlaceholder)
	}
	emit(len(text)-delimLen, len(text), SemanticTokenTypeString)
	return tokens
}
