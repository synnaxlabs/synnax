// Copyright 2025 Synnax Labs, Inc.
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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"go.lsp.dev/protocol"
)

// Semantic token types (standard LSP types)
const (
	SemanticTokenTypeKeyword  uint32 = 0
	SemanticTokenTypeType     uint32 = 1
	SemanticTokenTypeOperator uint32 = 2
	SemanticTokenTypeVariable uint32 = 3
	SemanticTokenTypeFunction uint32 = 4
	SemanticTokenTypeString   uint32 = 5
	SemanticTokenTypeNumber   uint32 = 6
	SemanticTokenTypeComment  uint32 = 7
)

// Token type legend for LSP client
var SemanticTokenTypes = []string{
	"keyword",
	"type",
	"operator",
	"variable",
	"function",
	"string",
	"number",
	"comment",
}

// Token modifiers (none for now)
var SemanticTokenModifiers = []string{}

// SemanticTokensFull implements textDocument/semanticTokens/full
func (s *Server) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	uri := params.TextDocument.URI

	s.mu.RLock()
	doc, ok := s.documents[uri]
	s.mu.RUnlock()

	if !ok {
		return &protocol.SemanticTokens{Data: []uint32{}}, nil
	}

	// Parse the document to extract tokens
	tokens := extractSemanticTokens(doc.Content)

	return &protocol.SemanticTokens{Data: tokens}, nil
}

// Token represents a semantic token before encoding
type token struct {
	line      uint32
	startChar uint32
	length    uint32
	tokenType uint32
}

// extractSemanticTokens parses the document and extracts semantic tokens
func extractSemanticTokens(content string) []uint32 {
	// Set up ANTLR parser
	input := antlr.NewInputStream(content)
	lexer := parser.NewArcLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// Collect all tokens from the lexer
	stream.Fill()
	allTokens := stream.GetAllTokens()

	var tokens []token

	for _, t := range allTokens {
		// Skip EOF token
		if t.GetTokenType() == antlr.TokenEOF {
			continue
		}

		tokenType := mapTokenType(t.GetTokenType())
		if tokenType == nil {
			// Skip tokens we don't highlight (whitespace, etc.)
			continue
		}

		// ANTLR line numbers are 1-based, LSP is 0-based
		line := uint32(t.GetLine() - 1)
		// ANTLR column is 0-based, LSP is also 0-based
		startChar := uint32(t.GetColumn())
		length := uint32(len(t.GetText()))

		tokens = append(tokens, token{
			line:      line,
			startChar: startChar,
			length:    length,
			tokenType: *tokenType,
		})
	}

	// Encode tokens in LSP format
	return encodeSemanticTokens(tokens)
}

// mapTokenType maps ANTLR token type to semantic token type
func mapTokenType(antlrType int) *uint32 {
	var tokenType uint32

	switch antlrType {
	// Keywords
	case parser.ArcLexerFUNC, parser.ArcLexerIF,
		parser.ArcLexerELSE, parser.ArcLexerRETURN, parser.ArcLexerNOW,
		parser.ArcLexerLEN:
		tokenType = SemanticTokenTypeKeyword

	// Types
	case parser.ArcLexerI8, parser.ArcLexerI16, parser.ArcLexerI32, parser.ArcLexerI64,
		parser.ArcLexerU8, parser.ArcLexerU16, parser.ArcLexerU32, parser.ArcLexerU64,
		parser.ArcLexerF32, parser.ArcLexerF64, parser.ArcLexerSTRING,
		parser.ArcLexerTIMESTAMP, parser.ArcLexerTIMESPAN, parser.ArcLexerSERIES,
		parser.ArcLexerCHAN, parser.ArcLexerRECV_CHAN, parser.ArcLexerSEND_CHAN:
		tokenType = SemanticTokenTypeType

	// Operators
	case parser.ArcLexerARROW, parser.ArcLexerRECV, parser.ArcLexerDECLARE,
		parser.ArcLexerSTATE_DECLARE, parser.ArcLexerASSIGN,
		parser.ArcLexerPLUS, parser.ArcLexerMINUS, parser.ArcLexerSTAR,
		parser.ArcLexerSLASH, parser.ArcLexerPERCENT, parser.ArcLexerCARET,
		parser.ArcLexerEQ, parser.ArcLexerNEQ, parser.ArcLexerLT, parser.ArcLexerGT,
		parser.ArcLexerLEQ, parser.ArcLexerGEQ, parser.ArcLexerAND, parser.ArcLexerOR,
		parser.ArcLexerNOT:
		tokenType = SemanticTokenTypeOperator

	// Literals
	case parser.ArcLexerSTRING_LITERAL:
		tokenType = SemanticTokenTypeString

	case parser.ArcLexerINTEGER_LITERAL, parser.ArcLexerFLOAT_LITERAL,
		parser.ArcLexerTEMPORAL_LITERAL, parser.ArcLexerFREQUENCY_LITERAL:
		tokenType = SemanticTokenTypeNumber

	// Comments (these are typically skipped by ANTLR, but included for completeness)
	case parser.ArcLexerSINGLE_LINE_COMMENT, parser.ArcLexerMULTI_LINE_COMMENT:
		tokenType = SemanticTokenTypeComment

	// Identifiers - default to variable
	case parser.ArcLexerIDENTIFIER:
		tokenType = SemanticTokenTypeVariable

	default:
		// Skip other tokens (punctuation, etc.)
		return nil
	}

	return &tokenType
}

// encodeSemanticTokens encodes tokens in LSP format
// LSP format: [deltaLine, deltaStartChar, length, tokenType, tokenModifiers, ...]
func encodeSemanticTokens(tokens []token) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}

	encoded := make([]uint32, 0, len(tokens)*5)
	prevLine := uint32(0)
	prevChar := uint32(0)

	for _, t := range tokens {
		// Calculate deltas
		deltaLine := t.line - prevLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = t.startChar - prevChar
		} else {
			deltaChar = t.startChar
		}

		// Encode: [deltaLine, deltaStartChar, length, tokenType, tokenModifiers]
		encoded = append(encoded, deltaLine, deltaChar, t.length, t.tokenType, 0)

		// Update previous position
		prevLine = t.line
		prevChar = t.startChar
	}

	return encoded
}

// Helper to include comments in semantic tokens
// Since ANTLR skips comments, we need to extract them separately
func extractComments(content string) []token {
	var tokens []token
	lines := strings.Split(content, "\n")

	for lineIdx, line := range lines {
		// Single-line comments
		if idx := strings.Index(line, "//"); idx != -1 {
			tokens = append(tokens, token{
				line:      uint32(lineIdx),
				startChar: uint32(idx),
				length:    uint32(len(line) - idx),
				tokenType: SemanticTokenTypeComment,
			})
		}

		// Multi-line comments (simplified - doesn't handle multi-line properly)
		if idx := strings.Index(line, "/*"); idx != -1 {
			endIdx := strings.Index(line[idx:], "*/")
			if endIdx != -1 {
				tokens = append(tokens, token{
					line:      uint32(lineIdx),
					startChar: uint32(idx),
					length:    uint32(endIdx + 2),
					tokenType: SemanticTokenTypeComment,
				})
			}
		}
	}

	return tokens
}
