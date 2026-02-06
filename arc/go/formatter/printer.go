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

type braceContext int

const (
	braceContextBlock braceContext = iota
	braceContextConfigBlock
	braceContextConfigValues
	braceContextStageBody
)

type parenContext int

const (
	parenContextDefault parenContext = iota
	parenContextInputList
	parenContextMultiOutput
)

type printer struct {
	output             strings.Builder
	indentLevel        int
	linePos            int
	lastTokenType      int
	prevToken          antlr.Token
	needsSpace         bool
	atLineStart        bool
	prevLine           int
	lastWasUnaryMinus  bool
	indentCache        []string
	delimiterStack     []int
	braceContextStack  []braceContext
	parenContextStack  []parenContext
	inlineConfigValues bool
	inlineConfigBlock  bool
	multilineParens    map[int]bool // tracks which paren depth levels are multiline
}

const maxCachedIndentLevel = 16

func newPrinter() *printer {
	cache := make([]string, maxCachedIndentLevel)
	for i := 0; i < maxCachedIndentLevel; i++ {
		cache[i] = strings.Repeat(" ", i*indentWidth)
	}
	return &printer{
		atLineStart:     true,
		indentCache:     cache,
		multilineParens: make(map[int]bool),
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
		p.handleOpenParen(idx, tokens)
	case parser.ArcLexerRPAREN:
		p.handleCloseParen(idx, tokens)
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

func (p *printer) detectBraceContext(idx int, tokens []antlr.Token) braceContext {
	if p.isFuncConfigBlock(idx, tokens) {
		return braceContextConfigBlock
	}
	if p.isStageBody(idx, tokens) {
		return braceContextStageBody
	}
	if p.isConfigValuesBlock(idx, tokens) {
		return braceContextConfigValues
	}
	return braceContextBlock
}

func (p *printer) isStageBody(idx int, tokens []antlr.Token) bool {
	if idx < 2 {
		return false
	}
	prevTok := tokens[idx-1]
	prevPrevTok := tokens[idx-2]
	return prevTok.GetTokenType() == parser.ArcLexerIDENTIFIER &&
		prevPrevTok.GetTokenType() == parser.ArcLexerSTAGE
}

func (p *printer) isFuncConfigBlock(idx int, tokens []antlr.Token) bool {
	if idx < 2 {
		return false
	}
	prevTok := tokens[idx-1]
	prevPrevTok := tokens[idx-2]
	return prevTok.GetTokenType() == parser.ArcLexerIDENTIFIER &&
		prevPrevTok.GetTokenType() == parser.ArcLexerFUNC
}

func (p *printer) isConfigValuesBlock(idx int, tokens []antlr.Token) bool {
	if idx < 1 {
		return false
	}
	prevTok := tokens[idx-1]
	if prevTok.GetTokenType() != parser.ArcLexerIDENTIFIER {
		return false
	}
	if idx >= 2 {
		prevPrevTok := tokens[idx-2]
		prevPrevType := prevPrevTok.GetTokenType()
		if prevPrevType == parser.ArcLexerFUNC ||
			prevPrevType == parser.ArcLexerSTAGE ||
			prevPrevType == parser.ArcLexerSEQUENCE {
			return false
		}
	}
	if p.isEmptyBlock(idx, tokens) {
		return true
	}
	return p.hasAssignInBraceBlock(idx, tokens)
}

func (p *printer) hasAssignInBraceBlock(idx int, tokens []antlr.Token) bool {
	braceDepth := 1
	for i := idx + 1; i < len(tokens); i++ {
		tokType := tokens[i].GetTokenType()
		switch tokType {
		case parser.ArcLexerLBRACE:
			braceDepth++
		case parser.ArcLexerRBRACE:
			braceDepth--
			if braceDepth == 0 {
				return false
			}
		case parser.ArcLexerASSIGN:
			if braceDepth == 1 {
				return true
			}
		}
	}
	return false
}

func (p *printer) shouldInlineConfigValues(idx int, tokens []antlr.Token) bool {
	length := p.calculateConfigValuesLength(idx, tokens)
	return length <= maxLineLength
}

func (p *printer) shouldInlineConfigBlock(idx int, tokens []antlr.Token) bool {
	length := p.calculateConfigBlockLength(idx, tokens)
	return length <= maxLineLength
}

func (p *printer) calculateConfigBlockLength(idx int, tokens []antlr.Token) int {
	length := p.linePos
	braceDepth := 1
	length++ // opening brace
	needsSpace := false
	for i := idx + 1; i < len(tokens); i++ {
		tok := tokens[i]
		tokType := tok.GetTokenType()
		switch tokType {
		case parser.ArcLexerLBRACE:
			braceDepth++
			length++
		case parser.ArcLexerRBRACE:
			braceDepth--
			length++
			if braceDepth == 0 {
				return length
			}
		case parser.ArcLexerCOMMA:
			length += 2 // comma + space
			needsSpace = false
		case parser.ArcLexerWS:
			continue
		default:
			if needsSpace {
				length++
			}
			length += len(tok.GetText())
			needsSpace = p.isWordToken(tokType)
		}
	}
	return length
}

func (p *printer) calculateConfigValuesLength(idx int, tokens []antlr.Token) int {
	length := p.linePos
	braceDepth := 1
	length++
	needsSpace := false
	for i := idx + 1; i < len(tokens); i++ {
		tok := tokens[i]
		tokType := tok.GetTokenType()
		switch tokType {
		case parser.ArcLexerLBRACE:
			braceDepth++
			length++
		case parser.ArcLexerRBRACE:
			braceDepth--
			length++
			if braceDepth == 0 {
				return length
			}
		case parser.ArcLexerCOMMA:
			length += 2
			needsSpace = false
		case parser.ArcLexerASSIGN:
			length++
			needsSpace = false
		case parser.ArcLexerWS:
			continue
		default:
			if needsSpace {
				length++
			}
			length += len(tok.GetText())
			needsSpace = p.isWordToken(tokType)
		}
	}
	return length
}

func (p *printer) inConfigValuesContext() bool {
	if len(p.braceContextStack) == 0 {
		return false
	}
	return p.braceContextStack[len(p.braceContextStack)-1] == braceContextConfigValues
}

func (p *printer) inConfigBlockContext() bool {
	if len(p.braceContextStack) == 0 {
		return false
	}
	return p.braceContextStack[len(p.braceContextStack)-1] == braceContextConfigBlock
}

func (p *printer) handleOpenBrace(idx int, tokens []antlr.Token) {
	ctx := p.detectBraceContext(idx, tokens)
	p.braceContextStack = append(p.braceContextStack, ctx)
	p.delimiterStack = append(p.delimiterStack, parser.ArcLexerLBRACE)

	if ctx == braceContextConfigValues {
		if p.shouldInlineConfigValues(idx, tokens) {
			p.inlineConfigValues = true
			p.emitChar("{")
			return
		}
		p.inlineConfigValues = false
	}

	if ctx == braceContextConfigBlock {
		if p.shouldInlineConfigBlock(idx, tokens) {
			p.inlineConfigBlock = true
			p.emitChar("{")
			return
		}
		p.inlineConfigBlock = false
	}

	// Add space before brace for non-config contexts (blocks, stage bodies)
	if ctx != braceContextConfigValues && ctx != braceContextConfigBlock {
		p.writeSpace()
	}
	p.emitChar("{")
	if p.isEmptyBlock(idx, tokens) {
		return
	}
	p.writeNewline()
	p.indentLevel++
}

func (p *printer) handleCloseBrace(idx int, tokens []antlr.Token) {
	p.popDelimiter()
	ctx := p.popBraceContext()

	if ctx == braceContextConfigValues && p.inlineConfigValues {
		p.emitChar("}")
		p.inlineConfigValues = false
		return
	}

	if ctx == braceContextConfigBlock && p.inlineConfigBlock {
		p.emitChar("}")
		p.inlineConfigBlock = false
		return
	}

	isEmptyBlock := idx > 0 && tokens[idx-1].GetTokenType() == parser.ArcLexerLBRACE
	if !isEmptyBlock {
		// Add trailing comma for multi-line config blocks and stage bodies
		if (ctx == braceContextConfigBlock || ctx == braceContextStageBody) &&
			p.lastTokenType != parser.ArcLexerCOMMA {
			p.emitChar(",")
		}
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

func (p *printer) popBraceContext() braceContext {
	if len(p.braceContextStack) == 0 {
		return braceContextBlock
	}
	ctx := p.braceContextStack[len(p.braceContextStack)-1]
	p.braceContextStack = p.braceContextStack[:len(p.braceContextStack)-1]
	return ctx
}

func (p *printer) handleOpenParen(idx int, tokens []antlr.Token) {
	ctx := p.detectParenContext(idx, tokens)
	p.parenContextStack = append(p.parenContextStack, ctx)

	if p.prevToken != nil && p.isBinaryOperator(p.lastTokenType) {
		p.writeSpace()
	} else if p.needsSpaceBeforeParen() {
		p.writeSpace()
	}

	// Check if this paren list should be multiline
	if ctx == parenContextInputList || ctx == parenContextMultiOutput {
		depth := len(p.parenContextStack)
		if p.shouldMultilineParenList(idx, tokens) {
			p.multilineParens[depth] = true
			p.emitChar("(")
			if !p.isEmptyParenList(idx, tokens) {
				p.writeNewline()
				p.indentLevel++
			}
			p.delimiterStack = append(p.delimiterStack, parser.ArcLexerLPAREN)
			return
		}
	}

	p.emitChar("(")
	p.delimiterStack = append(p.delimiterStack, parser.ArcLexerLPAREN)
}

func (p *printer) handleCloseParen(idx int, tokens []antlr.Token) {
	p.popDelimiter()
	ctx := p.popParenContext()
	depth := len(p.parenContextStack) + 1

	if p.multilineParens[depth] {
		delete(p.multilineParens, depth)
		isEmptyList := idx > 0 && tokens[idx-1].GetTokenType() == parser.ArcLexerLPAREN
		if !isEmptyList {
			// Add trailing comma for multiline paren lists
			if (ctx == parenContextInputList || ctx == parenContextMultiOutput) &&
				p.lastTokenType != parser.ArcLexerCOMMA {
				p.emitChar(",")
			}
			p.indentLevel--
			if p.indentLevel < 0 {
				p.indentLevel = 0
			}
			if !p.atLineStart {
				p.writeNewline()
			}
			p.writeIndent()
		}
	}
	p.emitChar(")")
}

func (p *printer) detectParenContext(idx int, tokens []antlr.Token) parenContext {
	if p.isInputListParen(idx, tokens) {
		return parenContextInputList
	}
	if p.isMultiOutputParen(idx, tokens) {
		return parenContextMultiOutput
	}
	return parenContextDefault
}

func (p *printer) isInputListParen(idx int, tokens []antlr.Token) bool {
	// Input list follows: func IDENTIFIER ( or func IDENTIFIER { ... } (
	for i := idx - 1; i >= 0; i-- {
		tokType := tokens[i].GetTokenType()
		switch tokType {
		case parser.ArcLexerIDENTIFIER:
			// Check if preceded by func
			if i > 0 && tokens[i-1].GetTokenType() == parser.ArcLexerFUNC {
				return true
			}
			continue
		case parser.ArcLexerRBRACE:
			// Skip over config block
			braceDepth := 1
			for j := i - 1; j >= 0 && braceDepth > 0; j-- {
				switch tokens[j].GetTokenType() {
				case parser.ArcLexerRBRACE:
					braceDepth++
				case parser.ArcLexerLBRACE:
					braceDepth--
					if braceDepth == 0 {
						i = j
					}
				}
			}
			continue
		default:
			return false
		}
	}
	return false
}

func (p *printer) isMultiOutputParen(idx int, tokens []antlr.Token) bool {
	// Multi-output follows: ) or ) type or ) IDENTIFIER type
	if idx < 1 {
		return false
	}
	prevType := tokens[idx-1].GetTokenType()

	// Direct: ) (
	if prevType == parser.ArcLexerRPAREN {
		// Check if the ) was from an input list
		return len(p.parenContextStack) > 0 &&
			p.parenContextStack[len(p.parenContextStack)-1] == parenContextInputList
	}

	// After type: ) type (
	if p.isType(prevType) {
		if idx >= 2 && tokens[idx-2].GetTokenType() == parser.ArcLexerRPAREN {
			return true
		}
		// After named output: ) IDENTIFIER type (
		if idx >= 3 &&
			tokens[idx-2].GetTokenType() == parser.ArcLexerIDENTIFIER &&
			tokens[idx-3].GetTokenType() == parser.ArcLexerRPAREN {
			return true
		}
	}

	return false
}

func (p *printer) popParenContext() parenContext {
	if len(p.parenContextStack) == 0 {
		return parenContextDefault
	}
	ctx := p.parenContextStack[len(p.parenContextStack)-1]
	p.parenContextStack = p.parenContextStack[:len(p.parenContextStack)-1]
	return ctx
}

func (p *printer) shouldMultilineParenList(idx int, tokens []antlr.Token) bool {
	length := p.calculateParenListLength(idx, tokens)
	return length > maxLineLength
}

func (p *printer) calculateParenListLength(idx int, tokens []antlr.Token) int {
	length := p.linePos
	parenDepth := 1
	length++ // opening paren
	needsSpace := false
	for i := idx + 1; i < len(tokens); i++ {
		tok := tokens[i]
		tokType := tok.GetTokenType()
		switch tokType {
		case parser.ArcLexerLPAREN:
			parenDepth++
			length++
		case parser.ArcLexerRPAREN:
			parenDepth--
			length++
			if parenDepth == 0 {
				return length
			}
		case parser.ArcLexerCOMMA:
			length += 2 // comma + space
			needsSpace = false
		case parser.ArcLexerWS:
			continue
		default:
			if needsSpace {
				length++
			}
			length += len(tok.GetText())
			needsSpace = p.isWordToken(tokType)
		}
	}
	return length
}

func (p *printer) isEmptyParenList(idx int, tokens []antlr.Token) bool {
	if idx+1 >= len(tokens) {
		return false
	}
	return tokens[idx+1].GetTokenType() == parser.ArcLexerRPAREN
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
	case parser.ArcLexerFUNC, parser.ArcLexerSEQUENCE, parser.ArcLexerAUTHORITY:
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
			nextType == parser.ArcLexerSTAGE || nextType == parser.ArcLexerAUTHORITY ||
			nextType == antlr.TokenEOF {
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

	if p.inConfigValuesContext() && p.inlineConfigValues {
		if tokType == parser.ArcLexerASSIGN || prevType == parser.ArcLexerASSIGN {
			return false
		}
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
		parser.ArcLexerNEXT, parser.ArcLexerNOT, parser.ArcLexerAUTHORITY:
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
	case parser.ArcLexerIF, parser.ArcLexerAUTHORITY:
		return true
	case parser.ArcLexerRBRACE:
		return true
	case parser.ArcLexerRPAREN:
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
	if p.inConfigValuesContext() && p.inlineConfigValues {
		return false
	}
	if p.inConfigBlockContext() && p.inlineConfigBlock {
		return false
	}
	// Break after comma in multiline paren lists
	if p.multilineParens[len(p.parenContextStack)] {
		return true
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
