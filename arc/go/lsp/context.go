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
	"slices"

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
	ContextInputParamName
	ContextInputParamValue
	ContextAuthorityEntry
	// ContextImportPath marks positions where the cursor sits in the
	// module-name slot of an `import` statement. Only module names should
	// be suggested there — not channels, functions, or other unrelated
	// symbols.
	ContextImportPath
	// ContextNone marks positions where the editor should not surface any
	// completions, such as the same line as the opening brace of a func,
	// sequence, or stage body. Completions resume once the cursor moves to
	// the next line.
	ContextNone
)

type inputContextInfo struct {
	functionName     string
	existingParams   []string
	currentParamName string
}

type inputParenResult struct {
	parenIndex   int
	functionName string
}

// findInputParen matches the unmatched `(` of a flow node's input values.
func findInputParen(tokens []antlr.Token) *inputParenResult {
	parenDepth := 0
	parenIndex := -1
	for i, token := range slices.Backward(tokens) {
		switch token.GetTokenType() {
		case parser.ArcLexerRPAREN:
			parenDepth++
		case parser.ArcLexerLPAREN:
			if parenDepth > 0 {
				parenDepth--
			} else {
				parenIndex = i
			}
		}
		if parenIndex >= 0 {
			break
		}
	}
	if parenIndex < 1 {
		return nil
	}
	prevToken := tokens[parenIndex-1]
	if prevToken.GetTokenType() != parser.ArcLexerIDENTIFIER ||
		!tokensAdjacent(prevToken, tokens[parenIndex]) {
		return nil
	}
	if parenIndex >= 2 && tokens[parenIndex-2].GetTokenType() == parser.ArcLexerFUNC {
		return nil
	}
	// A call inside a func body takes positional arguments.
	if detectNesting(tokens[:parenIndex]) == NestingFunction {
		return nil
	}
	return &inputParenResult{
		parenIndex:   parenIndex,
		functionName: prevToken.GetText(),
	}
}

func tokensAdjacent(a, b antlr.Token) bool {
	return a.GetLine() == b.GetLine() &&
		a.GetColumn()+len(a.GetText()) == b.GetColumn()
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
	if isAuthorityEntryContext(tokensBeforeCursor) {
		return ContextAuthorityEntry
	}
	if inputCtx := detectInputContext(tokensBeforeCursor); inputCtx != ContextUnknown {
		return inputCtx
	}
	if isSameLineAfterOpenBrace(lastToken, pos) {
		return ContextNone
	}
	if isDeclarationNameContext(tokensBeforeCursor) {
		return ContextNone
	}
	if isImportPathContext(tokensBeforeCursor) {
		return ContextImportPath
	}
	if isTypeAnnotationContext(tokensBeforeCursor, visibleTokens(tokens), lastToken) {
		return ContextTypeAnnotation
	}
	if isExpressionContext(tokensBeforeCursor, lastToken) {
		return ContextExpression
	}
	// When the last token is a partial identifier the user is typing, strip it
	// and re-evaluate context against the preceding token. This lets all
	// existing context rules apply correctly to the real syntactic position.
	if lastToken.GetTokenType() == parser.ArcLexerIDENTIFIER &&
		len(tokensBeforeCursor) >= 2 {
		stripped := tokensBeforeCursor[:len(tokensBeforeCursor)-1]
		strippedLast := stripped[len(stripped)-1]
		if isExpressionContext(stripped, strippedLast) {
			return ContextExpression
		}
		if isStatementStartContext(stripped, strippedLast, pos) {
			return ContextStatementStart
		}
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
		if tokenType != parser.ArcLexerSINGLE_LINE_COMMENT &&
			tokenType != parser.ArcLexerMULTI_LINE_COMMENT {
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
	for _, t := range visibleTokens(tokens) {
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

// isDeclarationNameContext reports whether the cursor sits at the slot
// that introduces a new identifier following a `sequence`, `stage`, or
// `func` keyword (with or without a partial name already typed). Those
// slots cannot legally reference any existing symbol, so no completions
// should be offered. `import` is intentionally excluded: the slot after
// `import` references an existing module name and should still surface
// module suggestions.
func isDeclarationNameContext(tokens []antlr.Token) bool {
	if len(tokens) == 0 {
		return false
	}
	last := tokens[len(tokens)-1]
	if isDeclarationKeyword(last.GetTokenType()) {
		return true
	}
	if last.GetTokenType() == parser.ArcLexerIDENTIFIER && len(tokens) >= 2 {
		prev := tokens[len(tokens)-2]
		if isDeclarationKeyword(prev.GetTokenType()) {
			return true
		}
	}
	return false
}

// isImportPathContext reports whether the cursor sits in the module-name
// slot of an `import` statement (loose form `import name` or block form
// `import ( name ... )`). The slot legally references only module names,
// so the completion list should be filtered to modules.
//
// Detection has two paths:
//   - Loose form: walking back must match `IDENT (DOT IDENT)* IMPORT`
//     entirely on the cursor's current source line. The same-line
//     constraint is what prevents a bare `m` on a new line after a
//     previous `import math` from being misread as another import path.
//   - Block form: there must be an unmatched LPAREN somewhere behind the
//     cursor whose preceding token is IMPORT. Newlines inside the block
//     are fine.
func isImportPathContext(tokens []antlr.Token) bool {
	if len(tokens) == 0 {
		return false
	}
	anchorLine := tokens[len(tokens)-1].GetLine()
	end := len(tokens) - 1
	if tokens[end].GetTokenType() == parser.ArcLexerIDENTIFIER {
		end--
	}
	j := end
	if j >= 0 &&
		tokens[j].GetLine() == anchorLine &&
		tokens[j].GetTokenType() == parser.ArcLexerDOT {
		j--
	}
	for j >= 0 && tokens[j].GetLine() == anchorLine {
		if tokens[j].GetTokenType() == parser.ArcLexerIMPORT {
			return true
		}
		if tokens[j].GetTokenType() != parser.ArcLexerIDENTIFIER {
			break
		}
		j--
		if j < 0 || tokens[j].GetLine() != anchorLine {
			break
		}
		if tokens[j].GetTokenType() == parser.ArcLexerIMPORT {
			return true
		}
		if tokens[j].GetTokenType() != parser.ArcLexerDOT {
			break
		}
		j--
	}
	parenDepth := 0
	for k := end; k >= 0; k-- {
		switch tokens[k].GetTokenType() {
		case parser.ArcLexerRPAREN:
			parenDepth++
		case parser.ArcLexerLPAREN:
			if parenDepth > 0 {
				parenDepth--
				continue
			}
			return k >= 1 && tokens[k-1].GetTokenType() == parser.ArcLexerIMPORT
		}
	}
	return false
}

func isDeclarationKeyword(t int) bool {
	switch t {
	case parser.ArcLexerSEQUENCE,
		parser.ArcLexerSTAGE,
		parser.ArcLexerFUNC:
		return true
	}
	return false
}

func visibleTokens(tokens []antlr.Token) []antlr.Token {
	var out []antlr.Token
	for _, t := range tokens {
		switch t.GetTokenType() {
		case antlr.TokenEOF, parser.ArcLexerWS,
			parser.ArcLexerSINGLE_LINE_COMMENT, parser.ArcLexerMULTI_LINE_COMMENT:
			continue
		}
		out = append(out, t)
	}
	return out
}

func isTypeAnnotationContext(tokens, all []antlr.Token, lastToken antlr.Token) bool {
	if lastToken.GetTokenType() != parser.ArcLexerIDENTIFIER {
		return false
	}
	if len(tokens) < 2 {
		return false
	}
	if !isFuncParamList(tokens, all) {
		return false
	}
	prevType := tokens[len(tokens)-2].GetTokenType()
	if startsParam(prevType) {
		return true
	}
	if prevType == parser.ArcLexerIDENTIFIER && len(tokens) >= 3 {
		return startsParam(tokens[len(tokens)-3].GetTokenType())
	}
	return false
}

func startsParam(tokenType int) bool {
	return tokenType == parser.ArcLexerLPAREN ||
		tokenType == parser.ArcLexerLBRACE ||
		tokenType == parser.ArcLexerCOMMA
}

// isFuncParamList reports whether the innermost unmatched `(` or `{` opens a func
// declaration's input list or trigger list.
func isFuncParamList(tokens, all []antlr.Token) bool {
	parenDepth, braceDepth := 0, 0
	for i, token := range slices.Backward(tokens) {
		switch token.GetTokenType() {
		case parser.ArcLexerRPAREN:
			parenDepth++
		case parser.ArcLexerLPAREN:
			if parenDepth > 0 {
				parenDepth--
			} else {
				return isInputListParen(tokens, i)
			}
		case parser.ArcLexerRBRACE:
			braceDepth++
		case parser.ArcLexerLBRACE:
			if braceDepth > 0 {
				braceDepth--
			} else {
				return isTriggerListBrace(tokens, all, i)
			}
		}
	}
	return false
}

func isInputListParen(tokens []antlr.Token, i int) bool {
	return i >= 2 &&
		tokens[i-1].GetTokenType() == parser.ArcLexerIDENTIFIER &&
		tokens[i-2].GetTokenType() == parser.ArcLexerFUNC
}

// isTriggerListBrace matches the `{` after a func declaration's input list. On the
// cursor's line it is a trigger list. On an earlier line it is one only when a body
// follows its closing brace, so an unfinished body is not a param list.
func isTriggerListBrace(tokens, all []antlr.Token, i int) bool {
	if i < 1 || tokens[i-1].GetTokenType() != parser.ArcLexerRPAREN {
		return false
	}
	depth := 0
	for j := i - 1; j >= 0; j-- {
		switch tokens[j].GetTokenType() {
		case parser.ArcLexerRPAREN:
			depth++
		case parser.ArcLexerLPAREN:
			depth--
			if depth == 0 {
				if !isInputListParen(tokens, j) {
					return false
				}
				return tokens[i].GetLine() == tokens[len(tokens)-1].GetLine() ||
					bodyFollowsBlock(all, i)
			}
		}
	}
	return false
}

// bodyFollowsBlock reports whether the brace block opened at all[i] closes and a body
// follows it: a `{`, an output type then `{`, or a multi-output list then `{`.
func bodyFollowsBlock(all []antlr.Token, i int) bool {
	j := matchingClose(all, i, parser.ArcLexerLBRACE, parser.ArcLexerRBRACE)
	if j < 0 {
		return false
	}
	j++
	if j < len(all) && all[j].GetTokenType() == parser.ArcLexerLPAREN {
		j = matchingClose(all, j, parser.ArcLexerLPAREN, parser.ArcLexerRPAREN)
		if j < 0 {
			return false
		}
		j++
	} else {
		for j < len(all) && isTypeToken(all[j].GetTokenType()) {
			j++
		}
	}
	return j < len(all) && all[j].GetTokenType() == parser.ArcLexerLBRACE
}

func matchingClose(tokens []antlr.Token, i, open, closing int) int {
	depth := 0
	for j := i; j < len(tokens); j++ {
		switch tokens[j].GetTokenType() {
		case open:
			depth++
		case closing:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

func isTypeToken(tokenType int) bool {
	switch tokenType {
	case parser.ArcLexerI8, parser.ArcLexerI16, parser.ArcLexerI32, parser.ArcLexerI64,
		parser.ArcLexerU8, parser.ArcLexerU16, parser.ArcLexerU32, parser.ArcLexerU64,
		parser.ArcLexerF32, parser.ArcLexerF64, parser.ArcLexerSTR,
		parser.ArcLexerBOOL, parser.ArcLexerSERIES, parser.ArcLexerCHAN:
		return true
	}
	return false
}

func isExpressionContext(tokens []antlr.Token, lastToken antlr.Token) bool {
	tokenType := lastToken.GetTokenType()
	if tokenType == parser.ArcLexerCOMMA {
		return isExpressionLevelComma(tokens)
	}
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
		parser.ArcLexerRETURN:
		return true
	}
	return false
}

// isExpressionLevelComma reports whether the innermost unmatched opening
// delimiter walking back from the end of tokens is `(` or `[` — i.e. the
// trailing comma is separating function arguments, function parameters, or
// array elements. Returns false when the innermost unmatched opening is `{`
// (a body block) or when there is no unmatched opening at all, in which case
// the comma is acting as a statement separator and the next position is
// statement-start, not expression.
func isExpressionLevelComma(tokens []antlr.Token) bool {
	parenDepth, bracketDepth, braceDepth := 0, 0, 0
	for _, token := range slices.Backward(tokens) {
		switch token.GetTokenType() {
		case parser.ArcLexerRPAREN:
			parenDepth++
		case parser.ArcLexerLPAREN:
			if parenDepth > 0 {
				parenDepth--
			} else {
				return braceDepth == 0
			}
		case parser.ArcLexerRBRACKET:
			bracketDepth++
		case parser.ArcLexerLBRACKET:
			if bracketDepth > 0 {
				bracketDepth--
			} else {
				return braceDepth == 0
			}
		case parser.ArcLexerRBRACE:
			braceDepth++
		case parser.ArcLexerLBRACE:
			if braceDepth > 0 {
				braceDepth--
			} else {
				return false
			}
		}
	}
	return false
}

// isSameLineAfterOpenBrace reports whether the cursor sits on the same line as
// the most recent token, when that token is an opening brace. The check is used
// to suppress completions immediately after typing the `{` of a func, sequence,
// or stage body so suggestions only appear once the cursor moves to the next
// line.
func isSameLineAfterOpenBrace(lastToken antlr.Token, pos protocol.Position) bool {
	if lastToken.GetTokenType() != parser.ArcLexerLBRACE {
		return false
	}
	cursorLine := int(pos.Line) + 1
	return cursorLine == lastToken.GetLine()
}

func isStatementStartContext(
	tokens []antlr.Token,
	lastToken antlr.Token,
	pos protocol.Position,
) bool {
	tokenType := lastToken.GetTokenType()
	lastLine := lastToken.GetLine()
	cursorLine := int(pos.Line) + 1
	if tokenType == parser.ArcLexerLBRACE {
		return cursorLine > lastLine
	}
	// A comma at statement level (not inside a function call, parameter list,
	// or array literal) terminates the preceding statement. The next position
	// is the start of a new statement regardless of whether the cursor has
	// moved to a new line — unlike `{`, a comma is a closer, not an opener,
	// so there is no need to suppress completions until the cursor wraps.
	if tokenType == parser.ArcLexerCOMMA {
		return !isExpressionLevelComma(tokens)
	}
	if len(tokens) == 0 {
		return true
	}
	if cursorLine > lastLine {
		switch tokenType {
		case parser.ArcLexerRBRACE,
			parser.ArcLexerRPAREN,
			parser.ArcLexerRBRACKET,
			parser.ArcLexerIDENTIFIER,
			parser.ArcLexerINTEGER_LITERAL,
			parser.ArcLexerFLOAT_LITERAL,
			parser.ArcLexerSTR_LITERAL,
			parser.ArcLexerSTR_LITERAL_MULTI:
			return true
		}
	}
	return false
}

// isAuthorityEntryContext checks if the cursor is inside an authority(...) block.
func isAuthorityEntryContext(tokens []antlr.Token) bool {
	parenDepth := 0
	for i, token := range slices.Backward(tokens) {
		tokenType := token.GetTokenType()
		switch tokenType {
		case parser.ArcLexerRPAREN:
			parenDepth++
		case parser.ArcLexerLPAREN:
			if parenDepth > 0 {
				parenDepth--
			} else {
				return i > 0 && tokens[i-1].GetTokenType() == parser.ArcLexerAUTHORITY
			}
		}
	}
	return false
}

// extractAuthorityExistingChannels returns channel names already listed in the
// authority block before the cursor position.
func extractAuthorityExistingChannels(content string, pos protocol.Position) []string {
	tokens := tokenizeContent(content)
	tokensBeforeCursor := getTokensBeforeCursor(tokens, pos)
	// Find the opening paren of the authority block.
	parenDepth := 0
	parenIdx := -1
	for i, t := range slices.Backward(tokensBeforeCursor) {
		tokenType := t.GetTokenType()
		switch tokenType {
		case parser.ArcLexerRPAREN:
			parenDepth++
		case parser.ArcLexerLPAREN:
			if parenDepth > 0 {
				parenDepth--
			} else {
				parenIdx = i
			}
		}
		if parenIdx >= 0 {
			break
		}
	}
	if parenIdx < 0 {
		return nil
	}
	// Collect identifiers inside the block — these are channel names in entries
	// like `valve 100`.
	var existing []string
	for i := parenIdx + 1; i < len(tokensBeforeCursor); i++ {
		if tokensBeforeCursor[i].GetTokenType() == parser.ArcLexerIDENTIFIER {
			existing = append(existing, tokensBeforeCursor[i].GetText())
		}
	}
	return existing
}

func detectInputContext(tokens []antlr.Token) CompletionContext {
	result := findInputParen(tokens)
	if result == nil {
		return ContextUnknown
	}
	lastToken := tokens[len(tokens)-1]
	lastTokenType := lastToken.GetTokenType()
	if lastTokenType == parser.ArcLexerASSIGN {
		return ContextInputParamValue
	}
	if lastTokenType == parser.ArcLexerLPAREN ||
		lastTokenType == parser.ArcLexerCOMMA ||
		lastTokenType == parser.ArcLexerIDENTIFIER {
		return ContextInputParamName
	}
	return ContextUnknown
}

// NestingKind represents the kind of nesting context the cursor is in.
type NestingKind int

const (
	NestingTopLevel     NestingKind = iota
	NestingFunction                 // inside func body, if block, etc.
	NestingSequenceBody             // inside sequence body (only stage declarations)
	NestingStageBody                // inside stage body (flow statements)
)

// detectNesting walks the tokens before the cursor and determines the
// innermost nesting context. It tracks SEQUENCE/STAGE/FUNC keywords
// followed by brace patterns, and any other braces (if blocks, etc.)
// are treated as function-level nesting.
func detectNesting(tokens []antlr.Token) NestingKind {
	type frame struct{ kind NestingKind }
	var stack []frame
	for i := 0; i < len(tokens); i++ {
		tt := tokens[i].GetTokenType()
		switch tt {
		case parser.ArcLexerSEQUENCE, parser.ArcLexerSTAGE:
			kind := NestingSequenceBody
			if tt == parser.ArcLexerSTAGE {
				kind = NestingStageBody
			}
			if i+2 < len(tokens) &&
				tokens[i+1].GetTokenType() == parser.ArcLexerIDENTIFIER &&
				tokens[i+2].GetTokenType() == parser.ArcLexerLBRACE {
				stack = append(stack, frame{kind: kind})
				i += 2
			} else if i+1 < len(tokens) &&
				tokens[i+1].GetTokenType() == parser.ArcLexerLBRACE {
				stack = append(stack, frame{kind: kind})
				i++
			}
		case parser.ArcLexerLBRACE:
			// A routing table's entries are flow statements of the enclosing body.
			kind := NestingFunction
			if i > 0 && isArrow(tokens[i-1]) {
				kind = NestingTopLevel
				if len(stack) > 0 {
					kind = stack[len(stack)-1].kind
				}
			}
			stack = append(stack, frame{kind: kind})
		case parser.ArcLexerRBRACE:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(stack) == 0 {
		return NestingTopLevel
	}
	return stack[len(stack)-1].kind
}

func isArrow(t antlr.Token) bool {
	tt := t.GetTokenType()
	return tt == parser.ArcLexerARROW || tt == parser.ArcLexerTRANSITION
}

func extractInputContext(content string, pos protocol.Position) *inputContextInfo {
	tokens := tokenizeContent(content)
	tokensBeforeCursor := getTokensBeforeCursor(tokens, pos)
	result := findInputParen(tokensBeforeCursor)
	if result == nil {
		return nil
	}
	info := &inputContextInfo{
		functionName:   result.functionName,
		existingParams: []string{},
	}
	for i := result.parenIndex + 1; i < len(tokensBeforeCursor); i++ {
		t := tokensBeforeCursor[i]
		if t.GetTokenType() == parser.ArcLexerIDENTIFIER {
			if i+1 < len(tokensBeforeCursor) &&
				tokensBeforeCursor[i+1].GetTokenType() == parser.ArcLexerASSIGN {
				info.existingParams = append(info.existingParams, t.GetText())
			}
		}
	}
	lastToken := tokensBeforeCursor[len(tokensBeforeCursor)-1]
	if lastToken.GetTokenType() == parser.ArcLexerASSIGN &&
		len(tokensBeforeCursor) >= 2 {
		prevToken := tokensBeforeCursor[len(tokensBeforeCursor)-2]
		if prevToken.GetTokenType() == parser.ArcLexerIDENTIFIER {
			info.currentParamName = prevToken.GetText()
		}
	}
	return info
}
