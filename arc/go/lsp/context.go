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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"go.lsp.dev/protocol"
)

type CompletionContext int

const (
	ContextUnknown CompletionContext = iota
	ContextComment
	ContextTypeAnnotation
	ContextExpression
	ContextStatementStart
)

func DetectCompletionContext(content string, pos protocol.Position) CompletionContext {
	if isPositionInComment(content, pos) {
		return ContextComment
	}

	tokens := TokenizeContent(content)
	tokensBeforeCursor := getTokensBeforeCursor(tokens, pos)

	if len(tokensBeforeCursor) == 0 {
		return ContextStatementStart
	}

	lastToken := tokensBeforeCursor[len(tokensBeforeCursor)-1]

	if isTypeAnnotationContext(tokensBeforeCursor, lastToken) {
		return ContextTypeAnnotation
	}

	if isExpressionContext(lastToken) {
		return ContextExpression
	}

	if isStatementStartContext(tokensBeforeCursor, lastToken, pos) {
		return ContextStatementStart
	}

	return ContextUnknown
}

func isPositionInComment(content string, pos protocol.Position) bool {
	input := antlr.NewInputStream(content)
	lexer := parser.NewArcLexer(input)
	lexer.RemoveErrorListeners()
	allTokens := lexer.GetAllTokens()
	line := int(pos.Line) + 1
	col := int(pos.Character)
	for _, t := range allTokens {
		tokenType := t.GetTokenType()
		if tokenType != parser.ArcLexerSINGLE_LINE_COMMENT && tokenType != parser.ArcLexerMULTI_LINE_COMMENT {
			continue
		}
		startLine := t.GetLine()
		startCol := t.GetColumn()
		text := t.GetText()
		if tokenType == parser.ArcLexerSINGLE_LINE_COMMENT {
			if line == startLine && col >= startCol {
				return true
			}
		} else {
			endLine, endCol := calculateEndPosition(startLine, startCol, text)
			if isPositionInRange(line, col, startLine, startCol, endLine, endCol) {
				return true
			}
		}
	}

	return false
}

func calculateEndPosition(startLine, startCol int, text string) (endLine, endCol int) {
	endLine = startLine
	endCol = startCol

	for _, ch := range text {
		if ch == '\n' {
			endLine++
			endCol = 0
		} else {
			endCol++
		}
	}

	return endLine, endCol
}

func isPositionInRange(line, col, startLine, startCol, endLine, endCol int) bool {
	if line < startLine || line > endLine {
		return false
	}
	if line == startLine && col < startCol {
		return false
	}
	if line == endLine && col > endCol {
		return false
	}
	return true
}

func getTokensBeforeCursor(tokens []antlr.Token, pos protocol.Position) []antlr.Token {
	line := int(pos.Line) + 1
	col := int(pos.Character)

	var result []antlr.Token
	for _, t := range tokens {
		tokenType := t.GetTokenType()
		if tokenType == antlr.TokenEOF ||
			tokenType == parser.ArcLexerWS ||
			tokenType == parser.ArcLexerSINGLE_LINE_COMMENT ||
			tokenType == parser.ArcLexerMULTI_LINE_COMMENT {
			continue
		}

		tokenLine := t.GetLine()
		tokenCol := t.GetColumn()
		tokenLen := len(t.GetText())

		if tokenLine > line || (tokenLine == line && tokenCol >= col) {
			break
		}

		if tokenLine < line || (tokenLine == line && tokenCol+tokenLen <= col) {
			result = append(result, t)
		}
	}

	return result
}

func isTypeAnnotationContext(tokens []antlr.Token, lastToken antlr.Token) bool {
	if lastToken.GetTokenType() != parser.ArcLexerIDENTIFIER {
		return false
	}

	if len(tokens) < 2 {
		return false
	}

	prevToken := tokens[len(tokens)-2]
	prevType := prevToken.GetTokenType()

	if prevType == parser.ArcLexerLPAREN || prevType == parser.ArcLexerCOMMA {
		return true
	}

	if prevType == parser.ArcLexerIDENTIFIER && len(tokens) >= 3 {
		prevPrevToken := tokens[len(tokens)-3]
		prevPrevType := prevPrevToken.GetTokenType()
		if prevPrevType == parser.ArcLexerLPAREN || prevPrevType == parser.ArcLexerCOMMA {
			return true
		}
	}

	return false
}

func isExpressionContext(lastToken antlr.Token) bool {
	tokenType := lastToken.GetTokenType()

	switch tokenType {
	case parser.ArcLexerDECLARE,
		parser.ArcLexerSTATE_DECLARE,
		parser.ArcLexerASSIGN,
		parser.ArcLexerPLUS_ASSIGN,
		parser.ArcLexerMINUS_ASSIGN,
		parser.ArcLexerSTAR_ASSIGN,
		parser.ArcLexerSLASH_ASSIGN,
		parser.ArcLexerPERCENT_ASSIGN,
		parser.ArcLexerPLUS,
		parser.ArcLexerMINUS,
		parser.ArcLexerSTAR,
		parser.ArcLexerSLASH,
		parser.ArcLexerPERCENT,
		parser.ArcLexerCARET,
		parser.ArcLexerEQ,
		parser.ArcLexerNEQ,
		parser.ArcLexerLT,
		parser.ArcLexerGT,
		parser.ArcLexerLEQ,
		parser.ArcLexerGEQ,
		parser.ArcLexerAND,
		parser.ArcLexerOR,
		parser.ArcLexerNOT,
		parser.ArcLexerLPAREN,
		parser.ArcLexerLBRACKET,
		parser.ArcLexerCOMMA,
		parser.ArcLexerRETURN:
		return true
	}

	return false
}

func isStatementStartContext(tokens []antlr.Token, lastToken antlr.Token, pos protocol.Position) bool {
	tokenType := lastToken.GetTokenType()

	if tokenType == parser.ArcLexerLBRACE {
		return true
	}

	if len(tokens) == 0 {
		return true
	}

	lastLine := lastToken.GetLine()
	cursorLine := int(pos.Line) + 1

	if cursorLine > lastLine {
		switch tokenType {
		case parser.ArcLexerRBRACE,
			parser.ArcLexerRPAREN,
			parser.ArcLexerRBRACKET,
			parser.ArcLexerIDENTIFIER,
			parser.ArcLexerINTEGER_LITERAL,
			parser.ArcLexerFLOAT_LITERAL,
			parser.ArcLexerSTR_LITERAL:
			return true
		}
	}

	return false
}
