// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/protocol"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// Token type ids must mirror the iota constants in arc/go/lsp/semantic.go.
// Tests in this file pin both the legend ordering and the token-type routing
// for STR_LITERAL vs STR_LITERAL_RAW to those ids.
const (
	tokenTypeVariable          = uint32(3)
	tokenTypeOperator          = uint32(2)
	tokenTypeString            = uint32(4)
	tokenTypeNumber            = uint32(5)
	tokenTypeChannel           = uint32(9)
	tokenTypeStringRaw         = uint32(21)
	tokenTypeStringPlaceholder = uint32(22)
)

// decodeSemanticTokens turns the LSP delta-encoded uint32 stream from
// SemanticTokensFull back into absolute (line, startChar, length, tokenType)
// records so individual tokens can be asserted on directly.
type decodedToken struct {
	Line      uint32
	StartChar uint32
	Length    uint32
	TokenType uint32
}

func decodeSemanticTokens(data []uint32) []decodedToken {
	out := make([]decodedToken, 0, len(data)/5)
	var prevLine, prevChar uint32
	for i := 0; i < len(data); i += 5 {
		deltaLine := data[i]
		deltaChar := data[i+1]
		line := prevLine + deltaLine
		var char uint32
		if deltaLine == 0 {
			char = prevChar + deltaChar
		} else {
			char = deltaChar
		}
		out = append(out, decodedToken{
			Line:      line,
			StartChar: char,
			Length:    data[i+2],
			TokenType: data[i+3],
		})
		prevLine = line
		prevChar = char
	}
	return out
}

// filterByType returns only the tokens matching the given token type id.
func filterByType(tokens []decodedToken, tokenType uint32) []decodedToken {
	out := make([]decodedToken, 0, len(tokens))
	for _, t := range tokens {
		if t.TokenType == tokenType {
			out = append(out, t)
		}
	}
	return out
}

var _ = Describe("Semantic Tokens", func() {
	var (
		server *lsp.Server
		uri    protocol.DocumentURI
	)

	BeforeEach(func() {
		server, uri = SetupTestServer()
	})

	Describe("appendTokenPerLine — raw string spans", func() {
		It("emits one token for a single-line raw literal", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `hello`")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeStringRaw)
			Expect(tokens).To(HaveLen(1))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(7)))
		})

		It("emits one token for an empty raw literal", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := ``")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeStringRaw)
			Expect(tokens).To(HaveLen(1))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(2)))
		})

		It("splits a raw literal with one mid-newline into two tokens", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `a\nb`")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeStringRaw)
			Expect(tokens).To(HaveLen(2))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(2)))
			Expect(tokens[1].Line).To(Equal(uint32(1)))
			Expect(tokens[1].StartChar).To(Equal(uint32(0)))
			Expect(tokens[1].Length).To(Equal(uint32(2)))
		})

		It("splits a three-line raw literal into three tokens", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `a\nb\nc`")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeStringRaw)
			Expect(tokens).To(HaveLen(3))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(2)))
			Expect(tokens[1].Line).To(Equal(uint32(1)))
			Expect(tokens[1].StartChar).To(Equal(uint32(0)))
			Expect(tokens[1].Length).To(Equal(uint32(1)))
			Expect(tokens[2].Line).To(Equal(uint32(2)))
			Expect(tokens[2].StartChar).To(Equal(uint32(0)))
			Expect(tokens[2].Length).To(Equal(uint32(2)))
		})

		It("emits a final token for a closing backtick on its own line", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `abc\n`")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeStringRaw)
			Expect(tokens).To(HaveLen(2))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(4)))
			Expect(tokens[1].Line).To(Equal(uint32(1)))
			Expect(tokens[1].StartChar).To(Equal(uint32(0)))
			Expect(tokens[1].Length).To(Equal(uint32(1)))
		})

		It("emits one token for a raw literal with escaped backticks", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `say \\`hi\\``")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeStringRaw)
			Expect(tokens).To(HaveLen(1))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(12)))
		})

		It("skips empty lines in a raw literal with consecutive newlines", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `a\n\nb`")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeStringRaw)
			Expect(tokens).To(HaveLen(2))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(2)))
			Expect(tokens[1].Line).To(Equal(uint32(2)))
			Expect(tokens[1].StartChar).To(Equal(uint32(0)))
			Expect(tokens[1].Length).To(Equal(uint32(2)))
		})
	})

	Describe("Token Type Routing", func() {
		It("routes backtick literals to the stringRaw token type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `hi`")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			raw := filterByType(tokens, tokenTypeStringRaw)
			Expect(raw).To(HaveLen(1))
			Expect(raw[0].Length).To(Equal(uint32(4)))
		})

		It("routes regular double-quoted strings to the string token type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := "hi"`)
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			str := filterByType(tokens, tokenTypeString)
			Expect(str).To(HaveLen(1))
			Expect(str[0].Length).To(Equal(uint32(4)))
			Expect(filterByType(tokens, tokenTypeStringRaw)).To(BeEmpty())
		})

		It("does not emit a stringRaw token when no backtick literal is present", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := 42")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(tokens, tokenTypeStringRaw)).To(BeEmpty())
		})
	})

	Describe("Raw-string placeholders", func() {
		It("splits `{42}` into stringRaw segments, placeholder braces, and a number", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `val: {42}`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			var inLit []decodedToken
			for _, t := range all {
				if t.Line == 0 && t.StartChar >= 5 {
					inLit = append(inLit, t)
				}
			}
			Expect(inLit).To(HaveLen(6))
			Expect(inLit[0]).To(Equal(decodedToken{Line: 0, StartChar: 5, Length: 1, TokenType: tokenTypeStringRaw}))
			Expect(inLit[1]).To(Equal(decodedToken{Line: 0, StartChar: 6, Length: 5, TokenType: tokenTypeStringRaw}))
			Expect(inLit[2]).To(Equal(decodedToken{Line: 0, StartChar: 11, Length: 1, TokenType: tokenTypeStringPlaceholder}))
			Expect(inLit[3]).To(Equal(decodedToken{Line: 0, StartChar: 12, Length: 2, TokenType: tokenTypeNumber}))
			Expect(inLit[4]).To(Equal(decodedToken{Line: 0, StartChar: 14, Length: 1, TokenType: tokenTypeStringPlaceholder}))
			Expect(inLit[5]).To(Equal(decodedToken{Line: 0, StartChar: 15, Length: 1, TokenType: tokenTypeStringRaw}))
		})

		It("classifies a placeholder identifier through the global resolver", func(ctx SpecContext) {
			globalResolver := symbol.MapResolver{
				"sensorData": symbol.Symbol{
					Name: "sensorData",
					Type: types.Chan(types.F64()),
					Kind: symbol.KindChannel,
				},
			}
			server, uri = SetupTestServer(lsp.Config{GlobalResolver: globalResolver})
			OpenArcDocument(server, ctx, uri, "x := `v: {sensorData}`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			ch := filterByType(all, tokenTypeChannel)
			Expect(ch).To(HaveLen(1))
			Expect(ch[0]).To(Equal(decodedToken{Line: 0, StartChar: 10, Length: 10, TokenType: tokenTypeChannel}))
		})

		It("does not treat `\\{` or `\\}` as placeholder bounds", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `a \\{ b \\} c`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(BeEmpty())
			raw := filterByType(all, tokenTypeStringRaw)
			Expect(raw).ToNot(BeEmpty())
		})

		It("recognizes a real placeholder while ignoring surrounding escapes", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `\\{ {42} \\}`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(HaveLen(2))
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
		})

		It("falls back to a single stringRaw on a malformed placeholder", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `unterminated {x`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			raw := filterByType(all, tokenTypeStringRaw)
			Expect(raw).To(HaveLen(1))
			Expect(raw[0].Length).To(Equal(uint32(17)))
			for _, op := range filterByType(all, tokenTypeOperator) {
				Expect(op.StartChar < 5).To(BeTrue())
			}
		})

		It("emits a placeholder span for a numeric format spec after the expression", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `v={42%05d}`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			ph := filterByType(all, tokenTypeStringPlaceholder)
			Expect(ph).To(HaveLen(3))
			Expect(ph[0]).To(Equal(decodedToken{Line: 0, StartChar: 8, Length: 1, TokenType: tokenTypeStringPlaceholder}))
			Expect(ph[1]).To(Equal(decodedToken{Line: 0, StartChar: 11, Length: 4, TokenType: tokenTypeStringPlaceholder}))
			Expect(ph[2]).To(Equal(decodedToken{Line: 0, StartChar: 15, Length: 1, TokenType: tokenTypeStringPlaceholder}))
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
		})

		It("classifies multi-token placeholder expressions with prev/next context", func(ctx SpecContext) {
			globalResolver := symbol.MapResolver{
				"sensor": symbol.Symbol{
					Name: "sensor",
					Type: types.Chan(types.F64()),
					Kind: symbol.KindChannel,
				},
			}
			server, uri = SetupTestServer(lsp.Config{GlobalResolver: globalResolver})
			OpenArcDocument(server, ctx, uri, "x := `v={sensor + 1}`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeChannel)).To(HaveLen(1))
			plus := filterByType(all, tokenTypeOperator)
			Expect(plus).ToNot(BeEmpty())
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
		})

		It("skips inner placeholder tokens that classify to nil (parens)", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `v={(42)}`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(HaveLen(2))
		})
	})

	Describe("SemanticTokensFull", func() {
		It("returns an empty token stream for an unknown document URI", func(ctx SpecContext) {
			result := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///not-open.arc"},
			}))
			Expect(result.Data).To(BeEmpty())
		})
	})

	Describe("Legend", func() {
		It("registers stringRaw and stringPlaceholder at the end of the semantic token types legend", func(ctx SpecContext) {
			result := MustSucceed(server.Initialize(ctx, &protocol.InitializeParams{
				ClientInfo: &protocol.ClientInfo{Name: "test"},
			}))
			provider, ok := result.Capabilities.SemanticTokensProvider.(map[string]any)
			Expect(ok).To(BeTrue())
			legend, ok := provider["legend"].(protocol.SemanticTokensLegend)
			Expect(ok).To(BeTrue())
			Expect(legend.TokenTypes).ToNot(BeEmpty())
			n := len(legend.TokenTypes)
			Expect(string(legend.TokenTypes[n-2])).To(Equal("stringRaw"))
			Expect(string(legend.TokenTypes[n-1])).To(Equal("stringPlaceholder"))
			Expect(uint32(n - 2)).To(Equal(tokenTypeStringRaw))
			Expect(uint32(n - 1)).To(Equal(tokenTypeStringPlaceholder))
		})
	})
})
