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
	ContextConfigParamName
	ContextConfigParamValue
)

type configContextInfo struct {
	functionName     string
	existingParams   []string
	currentParamName string
}

func DetectCompletionContext(content string, pos protocol.Position) CompletionContext {
	if isPositionInComment(content, pos) {
		return ContextComment
	}
	tokens := tokenizeContent(content)
	tokensBeforeCursor := getTokensBeforeCursor(tokens, pos)
	if len(tokensBeforeCursor) == 0 {
		return ContextStatementStart
	}
	lastToken := tokensBeforeCursor[len(tokensBeforeCursor)-1]
	if configCtx := detectConfigContext(tokensBeforeCursor); configCtx != ContextUnknown {
		return configCtx
	}
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
	allTokens := tokenizeContent(content)
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
	if !isInsideParentheses(tokens) {
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

func isInsideParentheses(tokens []antlr.Token) bool {
	depth := 0
	for _, t := range tokens {
		switch t.GetTokenType() {
		case parser.ArcLexerLPAREN:
			depth++
		case parser.ArcLexerRPAREN:
			depth--
		}
	}
	return depth > 0
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

func detectConfigContext(tokens []antlr.Token) CompletionContext {
	if len(tokens) == 0 {
		return ContextUnknown
	}
	braceDepth := 0
	bracketDepth := 0
	parenDepth := 0
	configBraceIndex := -1
	for i := len(tokens) - 1; i >= 0; i-- {
		tokenType := tokens[i].GetTokenType()
		switch tokenType {
		case parser.ArcLexerRBRACE:
			braceDepth++
		case parser.ArcLexerLBRACE:
			if braceDepth > 0 {
				braceDepth--
			} else {
				configBraceIndex = i
			}
		case parser.ArcLexerRBRACKET:
			bracketDepth++
		case parser.ArcLexerLBRACKET:
			if bracketDepth > 0 {
				bracketDepth--
			}
		case parser.ArcLexerRPAREN:
			parenDepth++
		case parser.ArcLexerLPAREN:
			if parenDepth > 0 {
				parenDepth--
			}
		}
		if configBraceIndex >= 0 {
			break
		}
	}
	if configBraceIndex < 1 {
		return ContextUnknown
	}
	prevToken := tokens[configBraceIndex-1]
	if prevToken.GetTokenType() != parser.ArcLexerIDENTIFIER {
		return ContextUnknown
	}
	if configBraceIndex >= 2 {
		prevPrevToken := tokens[configBraceIndex-2]
		prevPrevType := prevPrevToken.GetTokenType()
		if prevPrevType == parser.ArcLexerRPAREN ||
			prevPrevType == parser.ArcLexerSTAGE ||
			prevPrevType == parser.ArcLexerSEQUENCE {
			return ContextUnknown
		}
	}
	lastToken := tokens[len(tokens)-1]
	lastTokenType := lastToken.GetTokenType()
	if lastTokenType == parser.ArcLexerASSIGN {
		return ContextConfigParamValue
	}
	if lastTokenType == parser.ArcLexerLBRACE ||
		lastTokenType == parser.ArcLexerCOMMA ||
		lastTokenType == parser.ArcLexerIDENTIFIER {
		return ContextConfigParamName
	}
	return ContextUnknown
}

func extractConfigContext(content string, pos protocol.Position) *configContextInfo {
	tokens := tokenizeContent(content)
	tokensBeforeCursor := getTokensBeforeCursor(tokens, pos)
	if len(tokensBeforeCursor) == 0 {
		return nil
	}
	braceDepth := 0
	configBraceIndex := -1
	for i := len(tokensBeforeCursor) - 1; i >= 0; i-- {
		tokenType := tokensBeforeCursor[i].GetTokenType()
		switch tokenType {
		case parser.ArcLexerRBRACE:
			braceDepth++
		case parser.ArcLexerLBRACE:
			if braceDepth > 0 {
				braceDepth--
			} else {
				configBraceIndex = i
			}
		}
		if configBraceIndex >= 0 {
			break
		}
	}
	if configBraceIndex < 1 {
		return nil
	}
	prevToken := tokensBeforeCursor[configBraceIndex-1]
	if prevToken.GetTokenType() != parser.ArcLexerIDENTIFIER {
		return nil
	}
	if configBraceIndex >= 2 {
		prevPrevToken := tokensBeforeCursor[configBraceIndex-2]
		prevPrevType := prevPrevToken.GetTokenType()
		if prevPrevType == parser.ArcLexerRPAREN ||
			prevPrevType == parser.ArcLexerSTAGE ||
			prevPrevType == parser.ArcLexerSEQUENCE {
			return nil
		}
	}
	info := &configContextInfo{
		functionName:   prevToken.GetText(),
		existingParams: []string{},
	}
	for i := configBraceIndex + 1; i < len(tokensBeforeCursor); i++ {
		t := tokensBeforeCursor[i]
		if t.GetTokenType() == parser.ArcLexerIDENTIFIER {
			if i+1 < len(tokensBeforeCursor) && tokensBeforeCursor[i+1].GetTokenType() == parser.ArcLexerASSIGN {
				info.existingParams = append(info.existingParams, t.GetText())
			}
		}
	}
	lastToken := tokensBeforeCursor[len(tokensBeforeCursor)-1]
	if lastToken.GetTokenType() == parser.ArcLexerASSIGN && len(tokensBeforeCursor) >= 2 {
		prevToken := tokensBeforeCursor[len(tokensBeforeCursor)-2]
		if prevToken.GetTokenType() == parser.ArcLexerIDENTIFIER {
			info.currentParamName = prevToken.GetText()
		}
	}
	return info
}
