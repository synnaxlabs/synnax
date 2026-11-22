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
	"regexp"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/synnaxlabs/arc/parser"
)

// identifierText matches the text of a token that scanCode reads as an identifier.
var identifierText = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// FuzzScanIdentifiers checks that scanCode finds each identifier outside strings that
// the live lexer finds with dashed names on.
func FuzzScanIdentifiers(f *testing.F) {
	for _, seed := range []string{
		"pt-01 -> out{}",
		"a-=1 a->b a-- a-",
		"authority ('valve-cmd' 200)",
		`f"{pt-01:.2f}" r"x-y" ` + "f`{a-b}`",
		"// a-b\n/* c-d */ e-f /* g",
		`"unclosed a-b`,
		`rf"{x}" fr"y" rfz-1 r-"q"`,
		"10ms-x 1.5e-3 .5a",
		`'a\'b' '' 'c`,
		"\u00E9a-b\r\nc",
		"!A0 $= b-c !=d-e $f-g",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		if !utf8.ValidString(src) {
			t.Skip()
		}
		var want, strs []span
		lexer := parser.NewLexer(src, parser.Config{AllowDashedNames: true})
		lexer.RemoveErrorListeners()
		for _, tok := range lexer.GetAllTokens() {
			s := span{
				start: byteOffset(src, tok.GetStart()),
				end:   byteOffset(src, tok.GetStop()+1),
			}
			switch {
			case tok.GetTokenType() == parser.ArcLexerSTR_LITERAL,
				tok.GetTokenType() == parser.ArcLexerSTR_LITERAL_MULTI:
				strs = append(strs, s)
			case identifierText.MatchString(tok.GetText()):
				// Keywords and type names lex as their own tokens.
				want = append(want, s)
			}
		}
		var got []span
		scanCode(src, nil, func(s span) {
			if !slices.ContainsFunc(strs, func(str span) bool {
				return str.start <= s.start && s.end <= str.end
			}) {
				got = append(got, s)
			}
		})
		if !slices.Equal(got, want) {
			t.Fatalf("source %q: scanned %v, lexed %v", src, got, want)
		}
	})
}

// byteOffset returns the byte offset in src of the code point at index i.
func byteOffset(src string, i int) int {
	for offset := range src {
		if i == 0 {
			return offset
		}
		i--
	}
	return len(src)
}
