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

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
)

// orphan is a rune of a text that no token holds, at its code point offset.
type orphan struct {
	at int
	r  rune
}

// quoteScan holds the lexical facts of one text that a quoted channel name needs.
// Offsets count code points.
type quoteScan struct {
	// names holds the quoted channel name and string literal tokens, in order.
	names []antlr.Token
	// orphans holds the runes that no token holds, in order, such as the quote of an
	// unclosed name or string.
	orphans []orphan
	// open is the offset of an unclosed block comment or multi-line string, which runs
	// to the end of the text, or -1.
	open int
	// lines holds the offset of each line start.
	lines []int
}

// scanQuotes lexes text once into its quote facts.
func scanQuotes(text string) *quoteScan {
	runes := []rune(text)
	q := &quoteScan{open: -1, lines: []int{0}}
	for i, r := range runes {
		if r == '\n' {
			q.lines = append(q.lines, i+1)
		}
	}
	next := 0
	var prev antlr.Token
	for _, t := range tokenizeContent(text) {
		if t.GetTokenType() == antlr.TokenEOF {
			break
		}
		q.addOrphans(runes, next, t.GetStart())
		next = t.GetStop() + 1
		switch t.GetTokenType() {
		case parser.ArcLexerCHANNEL_NAME,
			parser.ArcLexerSTR_LITERAL,
			parser.ArcLexerSTR_LITERAL_MULTI:
			q.names = append(q.names, t)
		case parser.ArcLexerSTAR, parser.ArcLexerSTAR_ASSIGN:
			// A closed comment lexes as one token, so an adjacent "/*" opens a comment
			// that never closes.
			if q.open < 0 && prev != nil &&
				prev.GetTokenType() == parser.ArcLexerSLASH &&
				prev.GetStop()+1 == t.GetStart() {
				q.open = prev.GetStart()
			}
		}
		prev = t
	}
	q.addOrphans(runes, next, len(runes))
	return q
}

func (q *quoteScan) addOrphans(runes []rune, from, to int) {
	for i := from; i < to; i++ {
		q.orphans = append(q.orphans, orphan{at: i, r: runes[i]})
		if runes[i] == '`' && (q.open < 0 || i < q.open) {
			q.open = i
		}
	}
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

// at returns where the cursor at offset c sits. In a quoted name, it also returns the
// offset of the opening quote. In a complete string, it also returns its token.
func (q *quoteScan) at(c int) (cursorPlace, int, antlr.Token) {
	if q.open >= 0 && c > q.open {
		return inLiteral, 0, nil
	}
	i := sort.Search(len(q.names), func(i int) bool {
		return q.names[i].GetStart() >= c
	})
	if i > 0 && c <= q.names[i-1].GetStop() {
		t := q.names[i-1]
		if t.GetTokenType() == parser.ArcLexerCHANNEL_NAME {
			return inQuote, t.GetStart(), nil
		}
		// The prefix of a string precedes its opening delimiter.
		if c > t.GetStart()+len(t.GetText())-len(trimStringPrefix(t.GetText())) {
			return inLiteral, 0, t
		}
		return inCode, 0, nil
	}
	lineStart := q.lines[sort.SearchInts(q.lines, c+1)-1]
	quote, backslash := -1, -2
	j := sort.Search(len(q.orphans), func(j int) bool {
		return q.orphans[j].at >= lineStart
	})
	for ; j < len(q.orphans) && q.orphans[j].at < c; j++ {
		o := q.orphans[j]
		switch {
		case o.r == '\\':
			backslash = o.at
		case quote >= 0:
			if o.r == '\'' && backslash != o.at-1 {
				quote = -1
			}
		case o.r == '\'':
			quote = o.at
		case o.r == '"':
			// A string that never closes runs to the end of its line.
			return inLiteral, 0, nil
		}
	}
	if quote >= 0 {
		return inQuote, quote, nil
	}
	return inCode, 0, nil
}

// trimStringPrefix returns text without the r and f letters before its opening
// delimiter.
func trimStringPrefix(text string) string {
	for len(text) > 0 && (text[0] == 'r' || text[0] == 'f') {
		text = text[1:]
	}
	return text
}
