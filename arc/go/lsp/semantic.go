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
	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/lsp"
	"github.com/synnaxlabs/x/lsp/protocol"
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
	SemanticTokenTypeUnit
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
	"unit",
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
	var tokens []lsp.Token
	// Track prev/next token types so classifyToken can handle qualified names (e.g., authority.set).
	for i, t := range allTokens {
		if t.GetTokenType() == antlr.TokenEOF {
			continue
		}
		if t.GetTokenType() == parser.ArcLexerSTR_LITERAL_RAW {
			tokens = append(tokens, expandRawStringPlaceholders(ctx, t, docIR)...)
			continue
		}
		var prevType, nextType int
		if i > 0 {
			prevType = allTokens[i-1].GetTokenType()
		}
		if i+1 < len(allTokens) {
			nextType = allTokens[i+1].GetTokenType()
		}
		tokenType := classifyToken(ctx, t, prevType, nextType, docIR)
		if tokenType == nil {
			continue
		}
		tokens = appendTokenPerLine(tokens, t, *tokenType)
	}
	return lsp.EncodeSemanticTokens(tokens)
}

// appendTokenPerLine emits one LSP semantic token per source line covered by t.
// Monaco does not render a single semantic token whose length crosses a newline,
// so multi-line ANTLR tokens (e.g. STR_LITERAL_RAW spanning several lines) must
// be split into per-line entries to receive consistent coloring across the whole
// span. For single-line tokens this collapses to one append, matching the prior
// behavior.
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

func classifyToken(ctx context.Context, t antlr.Token, prevTokenType, nextTokenType int, docIR ir.IR) *uint32 {
	return classifyTokenAt(ctx, t, prevTokenType, nextTokenType, docIR, t.GetLine(), t.GetColumn())
}

// Variant with explicit (line1, col0) for tokens lexed out of a sub-string.
func classifyTokenAt(
	ctx context.Context,
	t antlr.Token,
	prevTokenType, nextTokenType int,
	docIR ir.IR,
	line1, col0 int,
) *uint32 {
	antlrType := t.GetTokenType()
	// IDENTIFIER after DOT is the member part of a qualified name
	// (e.g., "set" in "authority.set"). Color it as a function.
	if antlrType == parser.ArcLexerIDENTIFIER && prevTokenType == parser.ArcLexerDOT {
		tokenType := uint32(SemanticTokenTypeFunction)
		return &tokenType
	}
	if antlrType == parser.ArcLexerIDENTIFIER && docIR.Symbols != nil {
		return classifyIdentifierAt(ctx, t.GetText(), line1, col0, docIR.Symbols)
	}
	// AUTHORITY followed by DOT is a module prefix (authority.set), not the
	// authority keyword. Color it as a namespace/variable instead of a keyword.
	if antlrType == parser.ArcLexerAUTHORITY && nextTokenType == parser.ArcLexerDOT {
		tokenType := uint32(SemanticTokenTypeVariable)
		return &tokenType
	}
	return mapLexerTokenType(antlrType)
}

func classifyIdentifierAt(ctx context.Context, name string, line1, col0 int, rootScope *symbol.Scope) *uint32 {
	scope := findScopeAtInternalPosition(rootScope, position{Line: line1, Col: col0})
	sym, err := scope.Resolve(ctx, name)
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
	case symbol.KindSequence:
		tokenType = SemanticTokenTypeSequence
	case symbol.KindStage:
		tokenType = SemanticTokenTypeStage
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
		parser.ArcLexerNOT, parser.ArcLexerAUTHORITY:
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
	case parser.ArcLexerSTR_LITERAL:
		tokenType = SemanticTokenTypeString
	case parser.ArcLexerSTR_LITERAL_RAW:
		tokenType = SemanticTokenTypeStringRaw
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

// expandRawStringPlaceholders tokenizes a STR_LITERAL_RAW with `{...}` placeholders.
// All parsing is delegated to fmtstring.Parse; this function only translates
// segment offsets into LSP semantic tokens with line/column bookkeeping.
func expandRawStringPlaceholders(ctx context.Context, t antlr.Token, docIR ir.IR) []lsp.Token {
	text := t.GetText()
	fallback := func() []lsp.Token { return appendTokenPerLine(nil, t, SemanticTokenTypeStringRaw) }
	body, ok := fmtstring.StripDelimiters(text)
	if !ok {
		return fallback()
	}
	segs, err := fmtstring.Parse(body)
	if err != nil {
		return fallback()
	}
	if !fmtstring.HasPlaceholder(segs) {
		return fallback()
	}
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
			relLine, relCol := uint32(it.GetLine()-1), uint32(it.GetColumn())
			absLine, absCol := baseLine+relLine, relCol
			if relLine == 0 {
				absCol = baseCol + relCol
			}
			tt := classifyTokenAt(ctx, it, prev, next, docIR, int(absLine)+1, int(absCol))
			if tt == nil {
				continue
			}
			tokens = appendTextTokenPerLine(tokens, it.GetText(), absLine, absCol, *tt)
		}
	}
	// bodyOff converts body offsets to text offsets (skip the leading backtick).
	const bodyOff = 1
	emit(0, 1, SemanticTokenTypeStringRaw)
	for _, seg := range segs {
		if !seg.IsPlaceholder {
			emit(seg.Start+bodyOff, seg.End+bodyOff, SemanticTokenTypeStringRaw)
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
	emit(len(text)-1, len(text), SemanticTokenTypeStringRaw)
	return tokens
}
