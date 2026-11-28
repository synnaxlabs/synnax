// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"fmt"
	"regexp"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/synnaxlabs/arc/parser"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
)

// identifierText matches the text of a token that RenameInText reads as an
// identifier. Keywords and type names lex as their own tokens.
var identifierText = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// renamed is a new name as the name passes write one: an old name with a suffix.
const renamed = "Metrics_2"

type token struct {
	kind int
	text string
}

func (t token) String() string { return fmt.Sprintf("%d:%q", t.kind, t.text) }

// FuzzRenameInText checks that a rename of every name writes the new name in place of
// each identifier that the live lexer finds outside strings, with dashed names on or
// off, and that no other token changes.
func FuzzRenameInText(f *testing.F) {
	for _, seed := range []string{
		"pt-01 -> out{}",
		"a-=1 a->b a-- a-",
		"authority (valve-cmd 200)",
		`f"{pt-01:.2f}" r"x-y" ` + "f`{a-b}`",
		"// a-b\n/* c-d */ e-f /* g",
		`"unclosed a-b`,
		`rf"{x}" fr"y" rfz-1 r-"q"`,
		"10ms-x 1.5e-3 .5a 0x1",
		`'a\'b' '' 'c`,
		"éa-b\r\nc",
		"!A0 $= b-c !=d-e $f-g",
	} {
		f.Add(seed, true)
		f.Add(seed, false)
	}
	f.Fuzz(func(t *testing.T, src string, dashed bool) {
		if !utf8.ValidString(src) {
			t.Skip()
		}
		cfg := parser.Config{AllowDashedNames: dashed}
		want := lex(src, cfg)
		for i, tok := range want {
			if identifierText.MatchString(tok.text) {
				want[i] = token{kind: parser.ArcLexerIDENTIFIER, text: renamed}
			}
		}
		out := v1.RenameInText(
			src,
			cfg,
			func(string) (string, bool) { return renamed, true },
		)
		if got := lex(out, cfg); !slices.Equal(got, want) {
			t.Fatalf("source %q renamed to %q: lexed %v, want %v", src, out, got, want)
		}
	})
}

// lex returns the tokens of src that the live lexer finds with cfg. A string keeps only
// its kind, since a rename also writes into the placeholders of a format string.
func lex(src string, cfg parser.Config) []token {
	lexer := parser.NewLexer(src, cfg)
	lexer.RemoveErrorListeners()
	all := lexer.GetAllTokens()
	tokens := make([]token, len(all))
	for i, tok := range all {
		tokens[i] = token{kind: tok.GetTokenType(), text: tok.GetText()}
		if tokens[i].kind == parser.ArcLexerSTR_LITERAL ||
			tokens[i].kind == parser.ArcLexerSTR_LITERAL_MULTI {
			tokens[i].text = ""
		}
	}
	return tokens
}
