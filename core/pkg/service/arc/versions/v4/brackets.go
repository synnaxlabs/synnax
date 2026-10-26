// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/set"
)

// SwapBrackets rewrites src from the grammar that wrote function inputs in braces and
// triggers in parens. A declaration becomes `func name(inputs) {triggers}`, with an
// empty `()` added when the old text had no input block, and a flow call `name{...}`
// becomes `name(...)` with no gap before the paren. Comments, strings and layout are
// kept. A bracket with no match leaves its construct unchanged. Text already in the
// current grammar is not valid input.
func SwapBrackets(src string) string {
	tokens := visibleTokens(src)
	e := editor{
		src:     []rune(src),
		replace: map[int]rune{},
		insert:  map[int]string{},
		drop:    make(set.Set[int]),
	}
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].GetTokenType() {
		case parser.ArcLexerFUNC:
			i = e.rewriteDeclaration(tokens, i)
		case parser.ArcLexerLBRACE:
			if end := matchIdx(tokens, i); end != -1 && isCallBrace(tokens, i) {
				e.swapPair(tokens, i, end, '(', ')')
				e.dropGap(tokens[i-1], tokens[i])
			}
		}
	}
	return e.apply()
}

// editor accumulates single-rune replacements, insertions and drops keyed by rune
// offset.
type editor struct {
	src     []rune
	replace map[int]rune
	insert  map[int]string
	drop    set.Set[int]
}

// rewriteDeclaration swaps the brackets of the declaration whose `func` is at i and
// returns the index to resume from. Braces inside a function body are never a call, so
// the body is skipped whole. A header that does not reach its trigger parens is left
// unchanged.
func (e *editor) rewriteDeclaration(tokens []antlr.Token, i int) int {
	name := i + 1
	if name >= len(tokens) ||
		tokens[name].GetTokenType() != parser.ArcLexerIDENTIFIER {
		return i
	}
	inputs, inputsEnd, triggers := -1, -1, name+1
	if triggers < len(tokens) &&
		tokens[triggers].GetTokenType() == parser.ArcLexerLBRACE {
		inputs = triggers
		if inputsEnd = matchIdx(tokens, inputs); inputsEnd == -1 {
			return i
		}
		triggers = inputsEnd + 1
	}
	if triggers >= len(tokens) ||
		tokens[triggers].GetTokenType() != parser.ArcLexerLPAREN {
		return i
	}
	triggersEnd := matchIdx(tokens, triggers)
	if triggersEnd == -1 {
		return i
	}
	if inputs != -1 {
		e.swapPair(tokens, inputs, inputsEnd, '(', ')')
	} else {
		insertion := "()"
		if adjacent(tokens[name], tokens[triggers]) {
			insertion = "() "
		}
		e.insert[tokens[name].GetStop()+1] = insertion
	}
	e.swapPair(tokens, triggers, triggersEnd, '{', '}')
	body := indexOf(tokens, triggersEnd+1, parser.ArcLexerLBRACE)
	if body == -1 {
		return triggersEnd
	}
	if bodyEnd := matchIdx(tokens, body); bodyEnd != -1 {
		return bodyEnd
	}
	return len(tokens)
}

func (e *editor) swapPair(tokens []antlr.Token, open, closing int, l, r rune) {
	e.replace[tokens[open].GetStart()] = l
	e.replace[tokens[closing].GetStart()] = r
}

// dropGap drops the whitespace between a call's name and its bracket: a flow call's
// paren must touch the name.
func (e *editor) dropGap(name, bracket antlr.Token) {
	for i := name.GetStop() + 1; i < bracket.GetStart(); i++ {
		if unicode.IsSpace(e.src[i]) {
			e.drop.Add(i)
		}
	}
}

func (e *editor) apply() string {
	var b strings.Builder
	b.Grow(len(e.src))
	for i, ch := range e.src {
		b.WriteString(e.insert[i])
		if e.drop.Contains(i) {
			continue
		}
		if repl, ok := e.replace[i]; ok {
			ch = repl
		}
		b.WriteRune(ch)
	}
	b.WriteString(e.insert[len(e.src)])
	return b.String()
}

// isCallBrace reports whether the brace at i opens the input values of a flow call: it
// follows a name that is not a stage or sequence name. `stable.for` is the one dotted
// name whose last segment lexes as a keyword.
func isCallBrace(tokens []antlr.Token, i int) bool {
	if i < 1 {
		return false
	}
	switch tokens[i-1].GetTokenType() {
	case parser.ArcLexerIDENTIFIER:
		if i < 2 {
			return true
		}
		before := tokens[i-2].GetTokenType()
		return before != parser.ArcLexerSTAGE && before != parser.ArcLexerSEQUENCE
	case parser.ArcLexerFOR:
		return i >= 2 && tokens[i-2].GetTokenType() == parser.ArcLexerDOT
	}
	return false
}

// matchIdx returns the index of the bracket closing the one at open, or -1.
func matchIdx(tokens []antlr.Token, open int) int {
	l, r := parser.ArcLexerLBRACE, parser.ArcLexerRBRACE
	if tokens[open].GetTokenType() == parser.ArcLexerLPAREN {
		l, r = parser.ArcLexerLPAREN, parser.ArcLexerRPAREN
	}
	depth := 0
	for i := open; i < len(tokens); i++ {
		switch tokens[i].GetTokenType() {
		case l:
			depth++
		case r:
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}

func indexOf(tokens []antlr.Token, from, tokType int) int {
	for i := from; i < len(tokens); i++ {
		if tokens[i].GetTokenType() == tokType {
			return i
		}
	}
	return -1
}

func adjacent(prev, next antlr.Token) bool {
	return prev.GetStop()+1 == next.GetStart()
}

// visibleTokens lexes src and drops whitespace, comments and the end marker. Lexer
// diagnostics are dropped too: a stored program that no longer lexes is rewritten as
// far as its tokens allow.
func visibleTokens(src string) []antlr.Token {
	lexer := parser.NewArcLexer(antlr.NewInputStream(src))
	lexer.RemoveErrorListeners()
	all := lexer.GetAllTokens()
	visible := make([]antlr.Token, 0, len(all))
	for _, tok := range all {
		switch tok.GetTokenType() {
		case parser.ArcLexerWS, parser.ArcLexerSINGLE_LINE_COMMENT,
			parser.ArcLexerMULTI_LINE_COMMENT, antlr.TokenEOF:
			continue
		}
		visible = append(visible, tok)
	}
	return visible
}
