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
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
)

// lineState is the construct that a line starts in.
type lineState uint8

const (
	stCode lineState = iota
	// stString is inside a multi-line string.
	stString
	// stComment is inside a block comment.
	stComment
)

// orphan is a rune that no token holds, at its column.
type orphan struct {
	col int
	r   rune
}

// span is a quoted channel name or a one-line string of a line. end is exclusive, and
// prefix is the length of a string's r and f letters.
type span struct {
	kind               int
	start, end, prefix int
}

// quoteLine holds the quote facts of one line. Columns count code points. Facts are
// relative to their line, so an edit keeps the facts of the lines it does not change.
type quoteLine struct {
	// start is the construct that the line starts in. closes is the column after its
	// end, or -1 when it runs past the line.
	start  lineState
	closes int
	// names and orphans hold the spans and the runes that no token holds, after closes
	// and before opens, in order.
	names   []span
	orphans []orphan
	// opens is the column of a multi-line string or block comment that the line does
	// not close, or -1. end is the construct that the next line starts in.
	opens int
	end   lineState
}

// quoteScan holds the quote facts of each line of one text.
type quoteScan struct {
	lines []*quoteLine
}

// scanQuotes returns the quote facts of text, lexed by s.
func scanQuotes(text string, s *lineScanner) *quoteScan {
	q := &quoteScan{}
	state := stCode
	for line := range strings.SplitSeq(text, "\n") {
		ql := s.line(line, state)
		q.lines = append(q.lines, ql)
		state = ql.end
	}
	return q
}

// update returns the quote facts of text, which was old when q was scanned, lexed by
// scanner. It scans again from the first changed line until a line starts as it did
// before, and keeps the facts of the other lines.
func (q *quoteScan) update(old, text string, scanner *lineScanner) *quoteScan {
	p := 0
	for p < len(old) && p < len(text) && old[p] == text[p] {
		p++
	}
	s := 0
	for s < len(old)-p && s < len(text)-p && old[len(old)-1-s] == text[len(text)-1-s] {
		s++
	}
	first := strings.Count(text[:p], "\n")
	last := strings.Count(text[:len(text)-s], "\n")
	delta := last - strings.Count(old[:len(old)-s], "\n")
	out := make([]*quoteLine, first, len(q.lines)+max(delta, 0))
	copy(out, q.lines[:first])
	state := q.lines[first].start
	rest := text[strings.LastIndexByte(text[:p], '\n')+1:]
	for i := first; ; i++ {
		if i > last && i-delta < len(q.lines) && q.lines[i-delta].start == state {
			return &quoteScan{lines: append(out, q.lines[i-delta:]...)}
		}
		line, after, more := strings.Cut(rest, "\n")
		ql := scanner.line(line, state)
		out = append(out, ql)
		if !more {
			return &quoteScan{lines: out}
		}
		state, rest = ql.end, after
	}
}

// lineScanner lexes one line at a time with one lexer. It is not safe for concurrent
// use.
type lineScanner struct {
	lexer *parser.ArcLexer
}

func newLineScanner() *lineScanner {
	l := parser.NewArcLexer(antlr.NewInputStream(""))
	l.RemoveErrorListeners()
	return &lineScanner{lexer: l}
}

// line returns the quote facts of the text of a line that starts in start.
func (s *lineScanner) line(text string, start lineState) *quoteLine {
	runes := []rune(text)
	q := &quoteLine{start: start, opens: -1}
	from := 0
	if start != stCode {
		if from = closeOf(runes, start); from < 0 {
			q.closes, q.end = -1, start
			return q
		}
		q.closes = from
	}
	s.lexer.SetInputStream(antlr.NewInputStream(string(runes[from:])))
	next := from
	var prev antlr.Token
	for _, t := range s.lexer.GetAllTokens() {
		a, b := from+t.GetStart(), from+t.GetStop()+1
		if q.addOrphans(runes, next, a) {
			return q
		}
		next = b
		switch t.GetTokenType() {
		case parser.ArcLexerCHANNEL_NAME,
			parser.ArcLexerSTR_LITERAL,
			parser.ArcLexerSTR_LITERAL_MULTI:
			text := t.GetText()
			q.names = append(q.names, span{
				kind:   t.GetTokenType(),
				start:  a,
				end:    b,
				prefix: len(text) - len(trimStringPrefix(text)),
			})
		case parser.ArcLexerSTAR, parser.ArcLexerSTAR_ASSIGN:
			// A closed comment lexes as one token, so an adjacent "/*" opens a comment
			// that the line does not close.
			if prev != nil && prev.GetTokenType() == parser.ArcLexerSLASH &&
				prev.GetStop()+1 == t.GetStart() {
				q.opens, q.end = from+prev.GetStart(), stComment
				return q
			}
		}
		prev = t
	}
	q.addOrphans(runes, next, len(runes))
	return q
}

// addOrphans adds the runes of [from, to) as orphans. An orphan backtick opens a
// multi-line string, which ends the line's facts, and then addOrphans returns true.
func (q *quoteLine) addOrphans(runes []rune, from, to int) bool {
	for i := from; i < to; i++ {
		if runes[i] == '`' {
			q.opens, q.end = i, stString
			return true
		}
		q.orphans = append(q.orphans, orphan{col: i, r: runes[i]})
	}
	return false
}

// closeOf returns the column after the end of the construct state that runes start in,
// or -1 when it runs past them.
func closeOf(runes []rune, state lineState) int {
	for i := 0; i < len(runes); i++ {
		switch {
		case state == stString && runes[i] == '\\':
			i++
		case state == stString && runes[i] == '`':
			return i + 1
		case state == stComment && runes[i] == '*' && i+1 < len(runes) &&
			runes[i+1] == '/':
			return i + 2
		}
	}
	return -1
}

// cursorPlace is where a cursor sits among the quotes and strings of a text.
type cursorPlace uint8

const (
	// inCode is outside any quoted name, string, or comment.
	inCode cursorPlace = iota
	// inQuote is in a quoted channel name, after its opening quote.
	inQuote
	// inLiteral is in the text of a string or a comment.
	inLiteral
)

// cursorAt is where a cursor sits. quote is the column of the opening quote of a quoted
// name. str is a one-line string that holds the cursor, and multi reports a cursor in a
// string that spans lines.
type cursorAt struct {
	place cursorPlace
	quote int
	str   *span
	multi bool
}

// at returns where the cursor at col of line sits.
func (q *quoteScan) at(line, col int) cursorAt {
	ql := q.lines[line]
	if ql.start != stCode && (ql.closes < 0 || col < ql.closes) {
		return cursorAt{place: inLiteral, multi: ql.start == stString}
	}
	if ql.opens >= 0 && col > ql.opens {
		return cursorAt{place: inLiteral, multi: ql.end == stString}
	}
	i := sort.Search(len(ql.names), func(i int) bool {
		return ql.names[i].start >= col
	})
	if i > 0 && col < ql.names[i-1].end {
		n := &ql.names[i-1]
		switch {
		case n.kind == parser.ArcLexerCHANNEL_NAME:
			return cursorAt{place: inQuote, quote: n.start}
		case col > n.start+n.prefix:
			return cursorAt{place: inLiteral, str: n}
		}
		return cursorAt{place: inCode}
	}
	quote, backslash := -1, -2
	for _, o := range ql.orphans {
		if o.col >= col {
			break
		}
		switch {
		case o.r == '\\':
			backslash = o.col
		case quote >= 0:
			if o.r == '\'' && backslash != o.col-1 {
				quote = -1
			}
		case o.r == '\'':
			quote = o.col
		case o.r == '"':
			// A string that never closes runs to the end of its line.
			return cursorAt{place: inLiteral}
		}
	}
	if quote >= 0 {
		return cursorAt{place: inQuote, quote: quote}
	}
	return cursorAt{place: inCode}
}

// multiString returns the line and column of the delimiter that opens the multi-line
// string that holds the cursor at col of line, and the line and column after its end.
// endCol is -1 when the string does not close.
func (q *quoteScan) multiString(
	line, col int,
) (startLine, startCol, endLine, endCol int) {
	startLine = line
	if ql := q.lines[line]; ql.opens < 0 || col <= ql.opens {
		for startLine--; q.lines[startLine].opens < 0; startLine-- {
		}
	}
	endLine, endCol = startLine+1, -1
	for ; endLine < len(q.lines); endLine++ {
		if endCol = q.lines[endLine].closes; endCol >= 0 {
			break
		}
	}
	return startLine, q.lines[startLine].opens, min(endLine, len(q.lines)-1), endCol
}

// trimStringPrefix returns text without the r and f letters before its opening
// delimiter.
func trimStringPrefix(text string) string {
	for len(text) > 0 && (text[0] == 'r' || text[0] == 'f') {
		text = text[1:]
	}
	return text
}
