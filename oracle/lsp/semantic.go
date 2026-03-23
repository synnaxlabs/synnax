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
	"github.com/synnaxlabs/oracle/parser"
	xlsp "github.com/synnaxlabs/x/lsp"
	"github.com/synnaxlabs/x/lsp/protocol"
	"github.com/synnaxlabs/x/set"
)

const (
	SemanticTokenTypeKeyword = iota
	SemanticTokenTypeType
	SemanticTokenTypeClass
	SemanticTokenTypeProperty
	SemanticTokenTypeDecorator
	SemanticTokenTypeString
	SemanticTokenTypeNumber
	SemanticTokenTypeComment
	SemanticTokenTypeFunction
)

var semanticTokenTypes = []string{
	"keyword",
	"type",
	"class",
	"property",
	"decorator",
	"string",
	"number",
	"comment",
	"function",
}

var primitiveTypes = set.New(
	"uuid", "string", "bool",
	"int8", "int16", "int32", "int64",
	"uint8", "uint16", "uint32", "uint64",
	"float32", "float64",
	"timestamp", "timespan", "time_range",
	"record", "bytes",
)

func (s *Server) SemanticTokensFull(_ context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return &protocol.SemanticTokens{Data: []uint32{}}, nil
	}
	tokens := extractSemanticTokens(doc.Content)
	return &protocol.SemanticTokens{Data: tokens}, nil
}

func extractSemanticTokens(content string) []uint32 {
	input := antlr.NewInputStream(content)
	lexer := parser.NewOracleLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	allTokens := stream.GetAllTokens()

	var tokens []xlsp.Token
	prevWasAt := false
	for _, t := range allTokens {
		if t.GetTokenType() == antlr.TokenEOF {
			continue
		}
		tokenType := mapTokenType(t.GetTokenType(), t.GetText(), prevWasAt)
		prevWasAt = t.GetTokenType() == parser.OracleLexerAT
		if tokenType == nil {
			continue
		}
		tokens = append(tokens, xlsp.Token{
			Line:      uint32(t.GetLine() - 1),
			StartChar: uint32(t.GetColumn()),
			Length:    uint32(len(t.GetText())),
			TokenType: *tokenType,
		})
	}
	return xlsp.EncodeSemanticTokens(tokens)
}

func mapTokenType(antlrType int, text string, prevWasAt bool) *uint32 {
	var tokenType uint32
	switch antlrType {
	case parser.OracleLexerSTRUCT, parser.OracleLexerENUM, parser.OracleLexerIMPORT,
		parser.OracleLexerEXTENDS, parser.OracleLexerMAP:
		tokenType = SemanticTokenTypeKeyword
	case parser.OracleLexerAT:
		tokenType = SemanticTokenTypeDecorator
	case parser.OracleLexerSTRING_LIT:
		tokenType = SemanticTokenTypeString
	case parser.OracleLexerINT_LIT, parser.OracleLexerFLOAT_LIT:
		tokenType = SemanticTokenTypeNumber
	case parser.OracleLexerBOOL_LIT:
		tokenType = SemanticTokenTypeKeyword
	case parser.OracleLexerLINE_COMMENT, parser.OracleLexerBLOCK_COMMENT:
		tokenType = SemanticTokenTypeComment
	case parser.OracleLexerIDENT:
		if prevWasAt {
			tokenType = SemanticTokenTypeFunction
		} else if primitiveTypes.Contains(text) {
			tokenType = SemanticTokenTypeType
		} else {
			tokenType = SemanticTokenTypeProperty
		}
	default:
		return nil
	}
	return &tokenType
}
