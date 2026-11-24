// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"fmt"
	"regexp"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/synnaxlabs/arc/parser"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
)

// identifierText matches the text of a token that RenameChannels reads as an
// identifier. Keywords and type names lex as their own tokens.
var identifierText = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

const (
	// name holds each character that a channel token or a string literal escapes.
	name = "x'\\\"`"
	// quoted is the channel token of name.
	quoted = "'x\\'\\\\\"`'"
)

type token struct {
	kind int
	text string
}

func (t token) String() string { return fmt.Sprintf("%d:%q", t.kind, t.text) }

// FuzzRenameChannels checks that a rename of every name writes the channel token of the
// new name in place of each identifier that the live lexer finds outside strings with
// dashed names on, and that no other token changes.
func FuzzRenameChannels(f *testing.F) {
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
		"éa-b\r\nc",
		"!A0 $= b-c !=d-e $f-g",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, src string) {
		if !utf8.ValidString(src) {
			t.Skip()
		}
		want := lex(src)
		for i, tok := range want {
			if identifierText.MatchString(tok.text) {
				want[i] = token{kind: parser.ArcLexerCHANNEL_NAME, text: quoted}
			}
		}
		out := v4.RenameChannels(src, func(string) (string, bool) { return name, true })
		if got := lex(out); !slices.Equal(got, want) {
			t.Fatalf("source %q renamed to %q: lexed %v, want %v", src, out, got, want)
		}
	})
}

// lex returns the tokens of src with dashed names on. A string keeps only its kind,
// since a rename also writes into the placeholders of a format string.
func lex(src string) []token {
	lexer := parser.NewLexer(src, parser.Config{AllowDashedNames: true})
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
