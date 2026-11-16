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

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/set"
)

// SwapBrackets rewrites src from the grammar that wrote function inputs in braces and
// triggers in parens. A declaration becomes `func name(inputs) {triggers}`, with an
// empty `()` added when the old text had no input block. A flow call `name{...}`
// becomes `name(...)`. An expression call `name(...)` that starts a flow node or stage
// item becomes `(name(...))`, since the current grammar reads it as a flow call. One
// that starts a top-level item after a flow becomes `name (...)` instead, since the
// flow would read the wrapped call as a call on its last node. Comments, strings and
// layout are kept. A bracket with no match leaves its construct unchanged. Text already
// in the current grammar is not valid input. cfgs sets how names lex, as in
// parser.Parse.
func SwapBrackets(src string, cfgs ...parser.Config) string {
	s := newStream(src, parser.ConfigOf(cfgs...))
	e := editor{
		src:     []rune(src),
		replace: map[int]rune{},
		insert:  map[int]string{},
		drop:    make(set.Set[int]),
	}
	// open holds the matched brackets enclosing the current token.
	var open []bracket
	// afterFlow is set while the last top-level item read is a flow statement.
	afterFlow := false
	for i := 0; i < len(s.tokens); i++ {
		for len(open) > 0 && open[len(open)-1].end <= i {
			open = open[:len(open)-1]
		}
		t := s.tokens[i].GetTokenType()
		if len(open) == 0 {
			if isFlowOperator(t) {
				afterFlow = true
			} else if startsNonFlowItem(t) {
				afterFlow = false
			}
		}
		switch t {
		case parser.ArcLexerFUNC:
			i = e.rewriteDeclaration(s, i)
		case parser.ArcLexerLBRACE:
			end := s.match[i]
			if end == -1 {
				continue
			}
			call := isCallBrace(s.tokens, i)
			if call {
				e.swapPair(s.tokens, i, end, '(', ')')
				e.moveGap(s.tokens[i-1], s.tokens[i], s.tokens[end])
			}
			open = append(open, bracket{end: end, holdsItems: !call})
		case parser.ArcLexerLPAREN, parser.ArcLexerLBRACKET:
			if end := s.match[i]; end != -1 {
				open = append(open, bracket{end: end})
			}
		case parser.ArcLexerIDENTIFIER, parser.ArcLexerAUTHORITY:
			if n := len(open); n > 0 && !open[n-1].holdsItems {
				continue
			}
			paren := s.itemCallParen(i)
			if paren == -1 {
				continue
			}
			afterItem := i > 0 && endsItem(s.tokens[i-1].GetTokenType())
			if len(open) == 0 && afterItem && afterFlow {
				e.insert[s.tokens[paren].GetStart()] += " "
				continue
			}
			if len(open) > 0 && afterItem {
				e.insert[s.tokens[i-1].GetStop()+1] += ","
			}
			e.insert[s.tokens[i].GetStart()] += "("
			e.insert[s.tokens[s.match[paren]].GetStop()+1] += ")"
		}
	}
	return e.apply()
}

// bracket is a matched opening bracket. holdsItems is set for a brace that holds flow
// nodes or stage items: a stage or sequence body, or a routing table.
type bracket struct {
	end        int
	holdsItems bool
}

// stream is the visible token stream with every bracket resolved up front, so a
// program full of open brackets costs one pass rather than one scan per bracket.
type stream struct {
	tokens []antlr.Token
	// match holds the index of the bracket closing the one at i, or -1.
	match []int
	// nextBrace holds the index of the first `{` at or after i, or -1. It has one
	// extra slot so a lookup just past the last token is valid.
	nextBrace []int
}

func newStream(src string, cfg parser.Config) stream {
	tokens := visibleTokens(src, cfg)
	s := stream{
		tokens:    tokens,
		match:     make([]int, len(tokens)),
		nextBrace: make([]int, len(tokens)+1),
	}
	var braces, parens, squares []int
	for i, tok := range tokens {
		s.match[i] = -1
		switch tok.GetTokenType() {
		case parser.ArcLexerLBRACE:
			braces = append(braces, i)
		case parser.ArcLexerRBRACE:
			braces = closeBracket(braces, s.match, i)
		case parser.ArcLexerLPAREN:
			parens = append(parens, i)
		case parser.ArcLexerRPAREN:
			parens = closeBracket(parens, s.match, i)
		case parser.ArcLexerLBRACKET:
			squares = append(squares, i)
		case parser.ArcLexerRBRACKET:
			squares = closeBracket(squares, s.match, i)
		}
	}
	s.nextBrace[len(tokens)] = -1
	for i := len(tokens) - 1; i >= 0; i-- {
		s.nextBrace[i] = s.nextBrace[i+1]
		if tokens[i].GetTokenType() == parser.ArcLexerLBRACE {
			s.nextBrace[i] = i
		}
	}
	return s
}

// itemCallParen returns the index of the matched paren opening the expression call
// whose callee starts at i, or -1 when no such call starts a flow node or stage item
// there. The caller checks that i is not inside an expression bracket.
func (s stream) itemCallParen(i int) int {
	tokens := s.tokens
	if i > 0 && continuesExpression(tokens[i-1].GetTokenType()) {
		return -1
	}
	callee := i
	if i+2 < len(tokens) && tokens[i+1].GetTokenType() == parser.ArcLexerDOT {
		callee = i + 2
	} else if tokens[i].GetTokenType() == parser.ArcLexerAUTHORITY {
		return -1
	}
	paren := callee + 1
	if paren >= len(tokens) || tokens[paren].GetTokenType() != parser.ArcLexerLPAREN ||
		!adjacent(tokens[callee], tokens[paren]) || s.match[paren] == -1 {
		return -1
	}
	return paren
}

func isFlowOperator(t int) bool {
	return t == parser.ArcLexerARROW || t == parser.ArcLexerTRANSITION
}

func startsNonFlowItem(t int) bool {
	switch t {
	case parser.ArcLexerFUNC, parser.ArcLexerSEQUENCE, parser.ArcLexerSTAGE,
		parser.ArcLexerDECLARE, parser.ArcLexerSTATE_DECLARE, parser.ArcLexerASSIGN,
		parser.ArcLexerPLUS_ASSIGN, parser.ArcLexerMINUS_ASSIGN,
		parser.ArcLexerSTAR_ASSIGN, parser.ArcLexerSLASH_ASSIGN,
		parser.ArcLexerPERCENT_ASSIGN:
		return true
	}
	return false
}

// continuesExpression reports whether a token of type t leaves an expression open, so
// a name after it is an operand and not the start of an item.
func continuesExpression(t int) bool {
	switch t {
	case parser.ArcLexerDOT, parser.ArcLexerASSIGN, parser.ArcLexerDECLARE,
		parser.ArcLexerSTATE_DECLARE, parser.ArcLexerPLUS_ASSIGN,
		parser.ArcLexerMINUS_ASSIGN, parser.ArcLexerSTAR_ASSIGN,
		parser.ArcLexerSLASH_ASSIGN, parser.ArcLexerPERCENT_ASSIGN,
		parser.ArcLexerPLUS, parser.ArcLexerMINUS, parser.ArcLexerSTAR,
		parser.ArcLexerSLASH, parser.ArcLexerPERCENT, parser.ArcLexerCARET,
		parser.ArcLexerEQ, parser.ArcLexerNEQ, parser.ArcLexerLT, parser.ArcLexerGT,
		parser.ArcLexerLEQ, parser.ArcLexerGEQ, parser.ArcLexerAND, parser.ArcLexerOR,
		parser.ArcLexerNOT:
		return true
	}
	return false
}

// endsItem reports whether a token of type t can end the item before a wrapped call.
// The parser would read the wrap's paren as a call on that item, so a comma must split
// them.
func endsItem(t int) bool {
	switch t {
	case parser.ArcLexerLBRACE, parser.ArcLexerCOMMA, parser.ArcLexerCOLON,
		parser.ArcLexerARROW, parser.ArcLexerTRANSITION:
		return false
	}
	return true
}

// closeBracket pairs the closer at i with the innermost open bracket on the stack. A
// closer with no opener is left unpaired.
func closeBracket(open, match []int, i int) []int {
	n := len(open)
	if n == 0 {
		return open
	}
	match[open[n-1]] = i
	return open[:n-1]
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
func (e *editor) rewriteDeclaration(s stream, i int) int {
	tokens := s.tokens
	name := i + 1
	if name >= len(tokens) ||
		tokens[name].GetTokenType() != parser.ArcLexerIDENTIFIER {
		return i
	}
	inputs, inputsEnd, triggers := -1, -1, name+1
	if triggers < len(tokens) &&
		tokens[triggers].GetTokenType() == parser.ArcLexerLBRACE {
		inputs = triggers
		if inputsEnd = s.match[inputs]; inputsEnd == -1 {
			return i
		}
		triggers = inputsEnd + 1
	}
	if triggers >= len(tokens) ||
		tokens[triggers].GetTokenType() != parser.ArcLexerLPAREN {
		return i
	}
	triggersEnd := s.match[triggers]
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
	body := s.nextBrace[triggersEnd+1]
	if body == -1 {
		return triggersEnd
	}
	if bodyEnd := s.match[body]; bodyEnd != -1 {
		return bodyEnd
	}
	return len(tokens)
}

func (e *editor) swapPair(tokens []antlr.Token, open, closing int, l, r rune) {
	e.replace[tokens[open].GetStart()] = l
	e.replace[tokens[closing].GetStart()] = r
}

// moveGap moves what sits between a call's name and its bracket to just after the
// closing bracket, since a flow call's paren must touch the name. Whitespace alone is
// dropped; a comment keeps its text and its line ending.
func (e *editor) moveGap(name, open, closing antlr.Token) {
	start, end := name.GetStop()+1, open.GetStart()
	gap := string(e.src[start:end])
	for i := start; i < end; i++ {
		e.drop.Add(i)
	}
	if strings.TrimSpace(gap) != "" {
		e.insert[closing.GetStop()+1] += gap
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
// follows a name that is not being declared as a function, stage or sequence.
// `stable.for` is the one dotted name whose last segment lexes as a keyword.
func isCallBrace(tokens []antlr.Token, i int) bool {
	if i < 1 {
		return false
	}
	switch tokens[i-1].GetTokenType() {
	case parser.ArcLexerIDENTIFIER:
		if i < 2 {
			return true
		}
		switch tokens[i-2].GetTokenType() {
		case parser.ArcLexerFUNC, parser.ArcLexerSTAGE, parser.ArcLexerSEQUENCE:
			return false
		}
		return true
	case parser.ArcLexerFOR:
		return i >= 2 && tokens[i-2].GetTokenType() == parser.ArcLexerDOT
	}
	return false
}

func adjacent(prev, next antlr.Token) bool {
	return prev.GetStop()+1 == next.GetStart()
}

// visibleTokens lexes src and drops whitespace, comments and the end marker. Lexer
// diagnostics are dropped too: a stored program that no longer lexes is rewritten as
// far as its tokens allow.
func visibleTokens(src string, cfg parser.Config) []antlr.Token {
	lexer := parser.NewLexer(src, cfg)
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
