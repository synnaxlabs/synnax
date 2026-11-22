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
	"strings"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/set"
	"go.lsp.dev/protocol"
)

// barePath is a dotted name in a token stream: an identifier, then one or more pairs of
// a dot and an identifier.
type barePath struct {
	// idents holds the index of each identifier of the path in the token stream.
	idents []int
	name   parser.Name
}

// barePaths returns the dotted names of tokens in order, skipping the identifiers in
// skip, which hold the paths of imports.
func barePaths(tokens []antlr.Token, skip set.Set[int]) []barePath {
	var (
		paths []barePath
		run   []int
	)
	end := func() {
		if len(run) > 1 {
			paths = append(paths, barePath{idents: run, name: pathName(tokens, run)})
		}
		run = nil
	}
	expectDot := false
	for i, t := range tokens {
		if t.GetChannel() != antlr.TokenDefaultChannel {
			continue
		}
		switch tt := t.GetTokenType(); {
		case expectDot && tt == parser.ArcLexerDOT:
			expectDot = false
		case !expectDot && tt == parser.ArcLexerIDENTIFIER && !skip.Contains(i):
			run, expectDot = append(run, i), true
		default:
			// A dot that no identifier follows ends the path before it.
			end()
			expectDot = false
			if tt == parser.ArcLexerIDENTIFIER && !skip.Contains(i) {
				run, expectDot = []int{i}, true
			}
		}
	}
	end()
	return paths
}

func pathName(tokens []antlr.Token, idents []int) parser.Name {
	var tail strings.Builder
	for i, idx := range idents[1:] {
		if i > 0 {
			tail.WriteByte('.')
		}
		tail.WriteString(tokens[idx].GetText())
	}
	return parser.Name{Head: tokens[idents[0]].GetText(), Tail: tail.String()}
}

// pathToken is a dotted name on one line and its range.
type pathToken struct {
	name parser.Name
	rng  protocol.Range
}

// linePaths returns the dotted names of tokens that sit on one line, in order.
func linePaths(tokens []antlr.Token) []pathToken {
	paths := barePaths(tokens, importContextIdents(tokens))
	out := make([]pathToken, 0, len(paths))
	for _, p := range paths {
		first, last := tokens[p.idents[0]], tokens[p.idents[len(p.idents)-1]]
		if first.GetLine() != last.GetLine() {
			continue
		}
		out = append(out, pathToken{
			name: p.name,
			rng: protocol.Range{
				Start: tokenRange(first).Start,
				End:   tokenRange(last).End,
			},
		})
	}
	return out
}

// holds reports whether the one-line range r holds pos.
func holds(r protocol.Range, pos protocol.Position) bool {
	return pos.Line == r.Start.Line && pos.Character >= r.Start.Character &&
		pos.Character < r.End.Character
}

// tokenRange returns the range of a token that spans one line.
func tokenRange(t antlr.Token) protocol.Range {
	start := protocol.Position{
		Line:      uint32(t.GetLine() - 1),
		Character: uint32(t.GetColumn()),
	}
	end := start
	end.Character += uint32(utf8.RuneCountInString(t.GetText()))
	return protocol.Range{Start: start, End: end}
}

// fmtStringTokens returns the string tokens of tokens that hold a placeholder.
func fmtStringTokens(tokens []antlr.Token) []antlr.Token {
	var out []antlr.Token
	for _, t := range tokens {
		switch t.GetTokenType() {
		case parser.ArcLexerSTR_LITERAL, parser.ArcLexerSTR_LITERAL_MULTI:
			if _, _, ok := fmtStrSegments(t.GetText()); ok {
				out = append(out, t)
			}
		}
	}
	return out
}

// placeholderPathAt returns the dotted name at pos inside a placeholder of the format
// string t.
func placeholderPathAt(
	t antlr.Token,
	pos protocol.Position,
	cfg parser.Config,
) (parser.Name, bool) {
	prefixLen, segs, ok := fmtStrSegments(t.GetText())
	if !ok {
		return parser.Name{}, false
	}
	at, bodyOff := tokenPositions(t), prefixLen+fmtStrDelimLen
	for _, seg := range segs {
		if !seg.IsPlaceholder {
			continue
		}
		inner := tokenizeContent(seg.Text, cfg)
		// A token counts runes, and the offsets of a segment count bytes.
		runeStart := make([]int, 0, len(seg.Text)+1)
		for i := range seg.Text {
			runeStart = append(runeStart, i)
		}
		runeStart = append(runeStart, len(seg.Text))
		offset := func(rune int) int { return bodyOff + seg.Offsets[runeStart[rune]] }
		for _, p := range barePaths(inner, nil) {
			first, last := inner[p.idents[0]], inner[p.idents[len(p.idents)-1]]
			r := protocol.Range{
				Start: at[offset(first.GetStart())],
				End:   at[offset(last.GetStop()+1)],
			}
			if holds(r, pos) {
				return p.name, true
			}
		}
	}
	return parser.Name{}, false
}
