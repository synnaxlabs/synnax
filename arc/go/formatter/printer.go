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
	cfg           Config
	output        strings.Builder
	indentLevel   int
	linePos       int
	lastTokenType int
	prevToken     antlr.Token
	needsSpace    bool
	atLineStart   bool
}

func newPrinter(cfg Config, _ string) *printer {
	return &printer{
		cfg:         cfg,
		atLineStart: true,
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

		p.emitLeadingComments(commentAttacher, tok)
		p.emitToken(tok, i, visibleTokens)
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

func (p *printer) emitLeadingComments(ca *commentAttacher, tok antlr.Token) {
	leading := ca.getLeadingComments(tok)
	for _, comment := range leading {
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

func (p *printer) emitToken(tok antlr.Token, idx int, tokens []antlr.Token) {
	tokType := tok.GetTokenType()

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
		p.handleComma(idx, tokens)
	case parser.ArcLexerCOLON:
		p.handleColon()
	default:
		p.handleDefault(tok, idx, tokens)
	}

	p.lastTokenType = tokType
}

func (p *printer) handleOpenBrace(idx int, tokens []antlr.Token) {
	p.writeSpace()
	p.output.WriteString("{")
	p.linePos++
	if p.isEmptyBlock(idx, tokens) {
		return
	}
	p.writeNewline()
	p.indentLevel++
}

func (p *printer) handleCloseBrace(idx int, tokens []antlr.Token) {
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
	p.output.WriteString("}")
	p.linePos++
	p.atLineStart = false
}

func (p *printer) handleOpenParen() {
	if p.needsSpaceBeforeParen() {
		p.writeSpace()
	}
	p.output.WriteString("(")
	p.linePos++
	p.atLineStart = false
}

func (p *printer) handleCloseParen() {
	p.output.WriteString(")")
	p.linePos++
	p.atLineStart = false
}

func (p *printer) handleOpenBracket() {
	p.output.WriteString("[")
	p.linePos++
	p.atLineStart = false
}

func (p *printer) handleCloseBracket() {
	p.output.WriteString("]")
	p.linePos++
	p.atLineStart = false
}

func (p *printer) handleComma(idx int, tokens []antlr.Token) {
	p.output.WriteString(",")
	p.linePos++
	if p.shouldBreakAfterComma(idx, tokens) {
		p.writeNewline()
	} else {
		p.needsSpace = true
	}
	p.atLineStart = false
}

func (p *printer) handleColon() {
	p.output.WriteString(":")
	p.linePos++
	p.needsSpace = true
	p.atLineStart = false
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
	} else if p.needsSpaceBefore(tok, idx, tokens) {
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

func (p *printer) needsSpaceBefore(tok antlr.Token, idx int, tokens []antlr.Token) bool {
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

	if prevType == parser.ArcLexerIDENTIFIER || p.isKeyword(prevType) || p.isType(prevType) ||
		p.isLiteral(prevType) {
		if tokType == parser.ArcLexerIDENTIFIER || p.isKeyword(tokType) || p.isType(tokType) ||
			p.isLiteral(tokType) {
			return true
		}
	}

	// Space after closing paren before type (return type)
	if prevType == parser.ArcLexerRPAREN && p.isType(tokType) {
		return true
	}

	// Space after type cast closing paren if followed by open brace
	if prevType == parser.ArcLexerRPAREN && tokType == parser.ArcLexerLBRACE {
		return true
	}

	// Space after closing brace before else
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

func (p *printer) shouldBreakAfterComma(idx int, tokens []antlr.Token) bool {
	depth := 0
	for i := idx - 1; i >= 0; i-- {
		tokType := tokens[i].GetTokenType()
		switch tokType {
		case parser.ArcLexerLBRACE:
			if depth == 0 {
				return true
			}
			depth--
		case parser.ArcLexerRBRACE:
			depth++
		case parser.ArcLexerLPAREN, parser.ArcLexerLBRACKET:
			if depth == 0 {
				return false
			}
		}
	}
	return false
}

func (p *printer) writeIndent() {
	indent := strings.Repeat(" ", p.indentLevel*p.cfg.IndentWidth)
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
