// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
)

type printer struct {
	output            strings.Builder
	indentLevel       int
	linePos           int
	lastTokenType     int
	prevToken         antlr.Token
	needsSpace        bool
	atLineStart       bool
	prevLine          int
	lastWasUnaryMinus bool
	indentCache       []string
	delimiterStack    []int
}

const maxCachedIndentLevel = 16

func newPrinter() *printer {
	cache := make([]string, maxCachedIndentLevel)
	for i := 0; i < maxCachedIndentLevel; i++ {
		cache[i] = strings.Repeat(" ", i*indentWidth)
	}
	return &printer{
		atLineStart: true,
		indentCache: cache,
	}
}

func (p *printer) print(tokens []antlr.Token) string {
	visibleTokens, comments := p.separateTokens(tokens)
	commentAttacher := newCommentAttacher(comments)

	for i, tok := range visibleTokens {
		if tok.GetTokenType() == antlr.TokenEOF {
			p.emitTrailingComments(commentAttacher)
			continue
		}

		p.emitToken(tok, i, visibleTokens, commentAttacher)
		if p.isLastTokenOnLine(tok, i, visibleTokens) {
			p.emitTrailingComment(commentAttacher, tok)
		}
		p.prevToken = tok
	}

	return strings.TrimRight(p.output.String(), " \t") + "\n"
}

func (p *printer) isLastTokenOnLine(tok antlr.Token, idx int, tokens []antlr.Token) bool {
	tokLine := tok.GetLine()
	if idx+1 >= len(tokens) {
		return true
	}
	nextTok := tokens[idx+1]
	if nextTok.GetTokenType() == antlr.TokenEOF {
		return true
	}
	return nextTok.GetLine() != tokLine
}

func (p *printer) separateTokens(tokens []antlr.Token) (visible []antlr.Token, comments []antlr.Token) {
	for _, tok := range tokens {
		switch tok.GetTokenType() {
		case parser.ArcLexerSINGLE_LINE_COMMENT, parser.ArcLexerMULTI_LINE_COMMENT:
			comments = append(comments, tok)
		case parser.ArcLexerWS:
			continue
		default:
			visible = append(visible, tok)
		}
	}
	return
}

func (p *printer) emitLeadingCommentsPreFetched(comments []antlr.Token) {
	for _, comment := range comments {
		p.writeIndent()
		p.output.WriteString(strings.TrimSpace(comment.GetText()))
		p.writeNewline()
	}
}

func (p *printer) emitTrailingComment(ca *commentAttacher, tok antlr.Token) {
	trailing := ca.getTrailingComment(tok)
	if trailing != nil {
		p.output.WriteString(" ")
		p.output.WriteString(strings.TrimSpace(trailing.GetText()))
	}
}

func (p *printer) emitTrailingComments(ca *commentAttacher) {
	trailing := ca.getRemainingComments()
	for _, comment := range trailing {
		if !p.atLineStart {
			p.writeNewline()
		}
		p.writeIndent()
		p.output.WriteString(strings.TrimSpace(comment.GetText()))
		p.writeNewline()
	}
}

func (p *printer) emitToken(tok antlr.Token, idx int, tokens []antlr.Token, ca *commentAttacher) {
	tokType := tok.GetTokenType()
	tokLine := tok.GetLine()

	leadingComments := ca.getLeadingComments(tok)
	hasLeadingComments := len(leadingComments) > 0

	if p.prevLine > 0 && tokLine > p.prevLine && !p.atLineStart {
		if tokType != parser.ArcLexerRBRACE || hasLeadingComments {
			p.writeNewline()
			if !hasLeadingComments {
				lineDiff := tokLine - p.prevLine
				blankLines := lineDiff - 1
				if blankLines > maxBlankLines {
					blankLines = maxBlankLines
				}
				for i := 0; i < blankLines; i++ {
					p.writeNewline()
				}
			}
		}
	}

	p.emitLeadingCommentsPreFetched(leadingComments)

	switch tokType {
	case parser.ArcLexerLBRACE:
		p.handleOpenBrace(idx, tokens)
	case parser.ArcLexerRBRACE:
		p.handleCloseBrace(idx, tokens)
	case parser.ArcLexerLPAREN:
		p.handleOpenParen()
	case parser.ArcLexerRPAREN:
		p.handleCloseParen()
	case parser.ArcLexerLBRACKET:
		p.handleOpenBracket()
	case parser.ArcLexerRBRACKET:
		p.handleCloseBracket()
	case parser.ArcLexerCOMMA:
		p.handleComma()
	case parser.ArcLexerCOLON:
		p.handleColon()
	default:
		p.handleDefault(tok, idx, tokens)
	}

	p.lastWasUnaryMinus = p.isUnaryMinus(tokType)
	p.lastTokenType = tokType
	p.prevLine = tokLine
}

func (p *printer) handleOpenBrace(idx int, tokens []antlr.Token) {
	p.writeSpace()
	p.emitChar("{")
	p.delimiterStack = append(p.delimiterStack, parser.ArcLexerLBRACE)
	if p.isEmptyBlock(idx, tokens) {
		return
	}
	p.writeNewline()
	p.indentLevel++
}

func (p *printer) handleCloseBrace(idx int, tokens []antlr.Token) {
	p.popDelimiter()
	isEmptyBlock := idx > 0 && tokens[idx-1].GetTokenType() == parser.ArcLexerLBRACE
	if !isEmptyBlock {
		p.indentLevel--
		if p.indentLevel < 0 {
			p.indentLevel = 0
		}
		if !p.atLineStart {
			p.writeNewline()
		}
		p.writeIndent()
	}
	p.emitChar("}")
}

func (p *printer) handleOpenParen() {
	if p.prevToken != nil && p.isBinaryOperator(p.lastTokenType) {
		p.writeSpace()
	} else if p.needsSpaceBeforeParen() {
		p.writeSpace()
	}
	p.emitChar("(")
	p.delimiterStack = append(p.delimiterStack, parser.ArcLexerLPAREN)
}

func (p *printer) handleCloseParen() {
	p.popDelimiter()
	p.emitChar(")")
}

func (p *printer) handleOpenBracket() {
	if p.prevToken != nil && p.isBinaryOperator(p.lastTokenType) {
		p.writeSpace()
	}
	p.emitChar("[")
	p.delimiterStack = append(p.delimiterStack, parser.ArcLexerLBRACKET)
}

func (p *printer) handleCloseBracket() {
	p.popDelimiter()
	p.emitChar("]")
}

func (p *printer) handleComma() {
	p.emitChar(",")
	if p.shouldBreakAfterComma() {
		p.writeNewline()
	} else {
		p.needsSpace = true
	}
}

func (p *printer) handleColon() {
	p.emitChar(":")
	p.needsSpace = true
}

func (p *printer) handleDefault(tok antlr.Token, idx int, tokens []antlr.Token) {
	tokType := tok.GetTokenType()
	tokText := tok.GetText()

	if p.needsNewlineBefore(tokType) {
		if !p.atLineStart {
			p.writeNewline()
		}
	}

	if p.atLineStart {
		p.writeIndent()
	} else if p.needsSpaceBefore(tok) {
		p.writeSpace()
	}

	p.output.WriteString(tokText)
	p.linePos += len(tokText)
	p.atLineStart = false
	p.needsSpace = false

	if p.needsNewlineAfter(tokType, idx, tokens) {
		p.writeNewline()
	}
}

func (p *printer) needsNewlineBefore(tokType int) bool {
	switch tokType {
	case parser.ArcLexerFUNC, parser.ArcLexerSEQUENCE:
		return p.prevToken != nil && p.lastTokenType != parser.ArcLexerRBRACE
	case parser.ArcLexerSTAGE:
		return p.prevToken != nil
	}
	return false
}

func (p *printer) needsNewlineAfter(tokType int, idx int, tokens []antlr.Token) bool {
	if idx+1 >= len(tokens) {
		return false
	}
	nextTok := tokens[idx+1]
	nextType := nextTok.GetTokenType()

	switch tokType {
	case parser.ArcLexerRBRACE:
		if nextType == parser.ArcLexerFUNC || nextType == parser.ArcLexerSEQUENCE ||
			nextType == parser.ArcLexerSTAGE || nextType == antlr.TokenEOF {
			return true
		}
	}
	return false
}

func (p *printer) needsSpaceBefore(tok antlr.Token) bool {
	if p.needsSpace {
		return true
	}
	if p.prevToken == nil {
		return false
	}

	tokType := tok.GetTokenType()
	prevType := p.lastTokenType

	if p.isUnitSuffix(tok) {
		return false
	}

	if p.lastWasUnaryMinus {
		return false
	}

	if p.isBinaryOperator(tokType) || p.isBinaryOperator(prevType) {
		return true
	}

	if prevType == parser.ArcLexerLPAREN || prevType == parser.ArcLexerLBRACKET ||
		prevType == parser.ArcLexerLBRACE {
		return false
	}

	if tokType == parser.ArcLexerRPAREN || tokType == parser.ArcLexerRBRACKET {
		return false
	}

	if p.isWordToken(prevType) && p.isWordToken(tokType) {
		return true
	}

	if prevType == parser.ArcLexerRPAREN && p.isType(tokType) {
		return true
	}

	if prevType == parser.ArcLexerRPAREN && tokType == parser.ArcLexerLBRACE {
		return true
	}

	if prevType == parser.ArcLexerRBRACE && tokType == parser.ArcLexerELSE {
		return true
	}

	return false
}

func (p *printer) isUnitSuffix(tok antlr.Token) bool {
	if tok.GetTokenType() != parser.ArcLexerIDENTIFIER {
		return false
	}
	if p.prevToken == nil {
		return false
	}
	prevType := p.prevToken.GetTokenType()
	if prevType != parser.ArcLexerINTEGER_LITERAL && prevType != parser.ArcLexerFLOAT_LITERAL {
		return false
	}
	return p.tokensAdjacent(p.prevToken, tok)
}

func (p *printer) tokensAdjacent(prev, curr antlr.Token) bool {
	return prev.GetStop()+1 == curr.GetStart()
}

func (p *printer) isBinaryOperator(tokType int) bool {
	switch tokType {
	case parser.ArcLexerPLUS, parser.ArcLexerMINUS, parser.ArcLexerSTAR,
		parser.ArcLexerSLASH, parser.ArcLexerPERCENT, parser.ArcLexerCARET,
		parser.ArcLexerEQ, parser.ArcLexerNEQ, parser.ArcLexerLT, parser.ArcLexerGT,
		parser.ArcLexerLEQ, parser.ArcLexerGEQ, parser.ArcLexerAND, parser.ArcLexerOR,
		parser.ArcLexerDECLARE, parser.ArcLexerSTATE_DECLARE, parser.ArcLexerASSIGN,
		parser.ArcLexerARROW, parser.ArcLexerTRANSITION,
		parser.ArcLexerPLUS_ASSIGN, parser.ArcLexerMINUS_ASSIGN,
		parser.ArcLexerSTAR_ASSIGN, parser.ArcLexerSLASH_ASSIGN, parser.ArcLexerPERCENT_ASSIGN:
		return true
	}
	return false
}

func (p *printer) isUnaryMinus(tokType int) bool {
	if tokType != parser.ArcLexerMINUS {
		return false
	}
	if p.prevToken == nil {
		return true
	}
	switch p.lastTokenType {
	case parser.ArcLexerLPAREN, parser.ArcLexerLBRACKET, parser.ArcLexerCOMMA,
		parser.ArcLexerCOLON, parser.ArcLexerRETURN, parser.ArcLexerDECLARE,
		parser.ArcLexerSTATE_DECLARE, parser.ArcLexerASSIGN,
		parser.ArcLexerPLUS, parser.ArcLexerSTAR, parser.ArcLexerSLASH, parser.ArcLexerPERCENT,
		parser.ArcLexerEQ, parser.ArcLexerNEQ, parser.ArcLexerLT, parser.ArcLexerGT,
		parser.ArcLexerLEQ, parser.ArcLexerGEQ, parser.ArcLexerAND, parser.ArcLexerOR,
		parser.ArcLexerARROW, parser.ArcLexerTRANSITION:
		return true
	}
	return false
}

func (p *printer) isKeyword(tokType int) bool {
	switch tokType {
	case parser.ArcLexerFUNC, parser.ArcLexerIF, parser.ArcLexerELSE,
		parser.ArcLexerRETURN, parser.ArcLexerSEQUENCE, parser.ArcLexerSTAGE,
		parser.ArcLexerNEXT, parser.ArcLexerNOT:
		return true
	}
	return false
}

func (p *printer) isType(tokType int) bool {
	switch tokType {
	case parser.ArcLexerI8, parser.ArcLexerI16, parser.ArcLexerI32, parser.ArcLexerI64,
		parser.ArcLexerU8, parser.ArcLexerU16, parser.ArcLexerU32, parser.ArcLexerU64,
		parser.ArcLexerF32, parser.ArcLexerF64, parser.ArcLexerSTR,
		parser.ArcLexerSERIES, parser.ArcLexerCHAN:
		return true
	}
	return false
}

func (p *printer) isLiteral(tokType int) bool {
	switch tokType {
	case parser.ArcLexerINTEGER_LITERAL, parser.ArcLexerFLOAT_LITERAL, parser.ArcLexerSTR_LITERAL:
		return true
	}
	return false
}

func (p *printer) isWordToken(tokType int) bool {
	return tokType == parser.ArcLexerIDENTIFIER ||
		p.isKeyword(tokType) || p.isType(tokType) || p.isLiteral(tokType)
}

func (p *printer) needsSpaceBeforeParen() bool {
	if p.prevToken == nil {
		return false
	}
	switch p.lastTokenType {
	case parser.ArcLexerIDENTIFIER:
		return false
	case parser.ArcLexerIF:
		return true
	case parser.ArcLexerRBRACE:
		return true
	}
	return false
}

func (p *printer) isEmptyBlock(idx int, tokens []antlr.Token) bool {
	if idx+1 >= len(tokens) {
		return false
	}
	return tokens[idx+1].GetTokenType() == parser.ArcLexerRBRACE
}

func (p *printer) shouldBreakAfterComma() bool {
	if len(p.delimiterStack) == 0 {
		return false
	}
	return p.delimiterStack[len(p.delimiterStack)-1] == parser.ArcLexerLBRACE
}

func (p *printer) emitChar(char string) {
	p.output.WriteString(char)
	p.linePos++
	p.atLineStart = false
}

func (p *printer) popDelimiter() {
	if len(p.delimiterStack) > 0 {
		p.delimiterStack = p.delimiterStack[:len(p.delimiterStack)-1]
	}
}

func (p *printer) writeIndent() {
	var indent string
	if p.indentLevel < maxCachedIndentLevel {
		indent = p.indentCache[p.indentLevel]
	} else {
		indent = strings.Repeat(" ", p.indentLevel*indentWidth)
	}
	p.output.WriteString(indent)
	p.linePos = len(indent)
	p.atLineStart = false
}

func (p *printer) writeSpace() {
	p.output.WriteString(" ")
	p.linePos++
}

func (p *printer) writeNewline() {
	p.output.WriteString("\n")
	p.linePos = 0
	p.atLineStart = true
	p.needsSpace = false
}
