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
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
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
	SemanticTokenTypeEdgeOneShot
	SemanticTokenTypeEdgeContinuous
	SemanticTokenTypeConstant
	SemanticTokenTypeConfig
	SemanticTokenTypeInput
	SemanticTokenTypeOutput
	SemanticTokenTypeUnit
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
	"edgeOneShot",
	"edgeContinuous",
	"constant",
	"config",
	"input",
	"output",
	"unit",
}

func (s *Server) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return &protocol.SemanticTokens{Data: []uint32{}}, nil
	}
	tokens := extractSemanticTokens(ctx, doc.Content, doc.IR)
	return &protocol.SemanticTokens{Data: tokens}, nil
}

type token struct {
	line      uint32
	startChar uint32
	length    uint32
	tokenType uint32
}

func extractSemanticTokens(ctx context.Context, content string, docIR ir.IR) []uint32 {
	allTokens := tokenizeContent(content)
	var tokens []token
	for _, t := range allTokens {
		if t.GetTokenType() == antlr.TokenEOF {
			continue
		}
		tokenType := classifyToken(ctx, t, docIR)
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

func classifyToken(ctx context.Context, t antlr.Token, docIR ir.IR) *uint32 {
	antlrType := t.GetTokenType()
	if antlrType == parser.ArcLexerIDENTIFIER && docIR.Symbols != nil {
		return classifyIdentifier(ctx, t, docIR.Symbols)
	}
	return mapLexerTokenType(antlrType)
}

func classifyIdentifier(ctx context.Context, t antlr.Token, rootScope *symbol.Scope) *uint32 {
	var (
		name  = t.GetText()
		pos   = position{Line: t.GetLine(), Col: t.GetColumn()}
		scope = findScopeAtInternalPosition(rootScope, pos)
	)
	sym, err := scope.Resolve(ctx, name)
	if err != nil || sym == nil {
		tokenType := uint32(SemanticTokenTypeVariable)
		return &tokenType
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
	case symbol.KindConstant:
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
	case symbol.KindBlock:
		tokenType = SemanticTokenTypeBlock
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
		parser.ArcLexerSEQUENCE, parser.ArcLexerSTAGE,
		parser.ArcLexerNEXT, parser.ArcLexerAND, parser.ArcLexerOR,
		parser.ArcLexerNOT:
		tokenType = SemanticTokenTypeKeyword
	case parser.ArcLexerI8, parser.ArcLexerI16, parser.ArcLexerI32, parser.ArcLexerI64,
		parser.ArcLexerU8, parser.ArcLexerU16, parser.ArcLexerU32, parser.ArcLexerU64,
		parser.ArcLexerF32, parser.ArcLexerF64, parser.ArcLexerSTR,
		parser.ArcLexerSERIES,
		parser.ArcLexerCHAN:
		tokenType = SemanticTokenTypeType
	case parser.ArcLexerTRANSITION:
		tokenType = SemanticTokenTypeEdgeOneShot
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
