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

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
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
)

var semanticTokenTypes = []string{
	"keyword",
	"type",
	"operator",
	"variable",
	"function",
	"string",
	"number",
	"comment",
}

func (s *Server) SemanticTokensFull(_ context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return &protocol.SemanticTokens{Data: []uint32{}}, nil
	}
	tokens := extractSemanticTokens(doc.Content)
	return &protocol.SemanticTokens{Data: tokens}, nil
}

type token struct {
	line      uint32
	startChar uint32
	length    uint32
	tokenType uint32
}

// extractSemanticTokens parses the document and extracts semantic tokens
func extractSemanticTokens(content string) []uint32 {
	var (
		input  = antlr.NewInputStream(content)
		lexer  = parser.NewArcLexer(input)
		stream = antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	)
	stream.Fill()
	allTokens := stream.GetAllTokens()
	var tokens []token
	for _, t := range allTokens {
		if t.GetTokenType() == antlr.TokenEOF {
			continue
		}
		tokenType := mapTokenType(t.GetTokenType())
		if tokenType == nil {
			continue
		}
		tokens = append(tokens, token{
			line:      uint32(t.GetLine() - 1),
			startChar: uint32(t.GetColumn()),
			length:    uint32(len(t.GetText())),
			tokenType: *tokenType,
		})
	}
	return encodeSemanticTokens(tokens)
}

func mapTokenType(antlrType int) *uint32 {
	var tokenType uint32
	switch antlrType {
	case parser.ArcLexerFUNC, parser.ArcLexerIF,
		parser.ArcLexerELSE, parser.ArcLexerRETURN, parser.ArcLexerNOW,
		parser.ArcLexerLEN,
		// Sequencing keywords
		parser.ArcLexerSTAGE, parser.ArcLexerMATCH, parser.ArcLexerNEXT:
		tokenType = SemanticTokenTypeKeyword
	case parser.ArcLexerI8, parser.ArcLexerI16, parser.ArcLexerI32, parser.ArcLexerI64,
		parser.ArcLexerU8, parser.ArcLexerU16, parser.ArcLexerU32, parser.ArcLexerU64,
		parser.ArcLexerF32, parser.ArcLexerF64, parser.ArcLexerSTR,
		parser.ArcLexerTIMESTAMP, parser.ArcLexerTIMESPAN, parser.ArcLexerSERIES,
		parser.ArcLexerCHAN, parser.ArcLexerRECV_CHAN, parser.ArcLexerSEND_CHAN:
		tokenType = SemanticTokenTypeType
	case parser.ArcLexerARROW, parser.ArcLexerRECV, parser.ArcLexerDECLARE,
		parser.ArcLexerSTATE_DECLARE, parser.ArcLexerTRANSITION, parser.ArcLexerASSIGN,
		parser.ArcLexerPLUS, parser.ArcLexerMINUS, parser.ArcLexerSTAR,
		parser.ArcLexerSLASH, parser.ArcLexerPERCENT, parser.ArcLexerCARET,
		parser.ArcLexerEQ, parser.ArcLexerNEQ, parser.ArcLexerLT, parser.ArcLexerGT,
		parser.ArcLexerLEQ, parser.ArcLexerGEQ, parser.ArcLexerAND, parser.ArcLexerOR,
		parser.ArcLexerNOT:
		tokenType = SemanticTokenTypeOperator
	case parser.ArcLexerSTR_LITERAL:
		tokenType = SemanticTokenTypeString
	case parser.ArcLexerINTEGER_LITERAL, parser.ArcLexerFLOAT_LITERAL,
		parser.ArcLexerTEMPORAL_LITERAL, parser.ArcLexerFREQUENCY_LITERAL:
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

func encodeSemanticTokens(tokens []token) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}
	encoded := make([]uint32, 0, len(tokens)*5)
	prevLine := uint32(0)
	prevChar := uint32(0)
	for _, t := range tokens {
		deltaLine := t.line - prevLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = t.startChar - prevChar
		} else {
			deltaChar = t.startChar
		}
		encoded = append(encoded, deltaLine, deltaChar, t.length, t.tokenType, 0)
		prevLine = t.line
		prevChar = t.startChar
	}

	return encoded
}
