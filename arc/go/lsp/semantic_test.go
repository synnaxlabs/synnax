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
	"github.com/synnaxlabs/x/lsp/protocol"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// Token type ids must mirror the iota constants in arc/go/lsp/semantic.go.
// Tests in this file pin both the legend ordering and the token-type routing
// for STR_LITERAL vs STR_LITERAL_RAW to those ids.
const (
	tokenTypeString    = uint32(4)
	tokenTypeStringRaw = uint32(21)
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

	Describe("Legend", func() {
		It("places stringRaw last in the semantic token types legend", func(ctx SpecContext) {
			result := MustSucceed(server.Initialize(ctx, &protocol.InitializeParams{
				ClientInfo: &protocol.ClientInfo{Name: "test"},
			}))
			provider, ok := result.Capabilities.SemanticTokensProvider.(map[string]any)
			Expect(ok).To(BeTrue())
			legend, ok := provider["legend"].(protocol.SemanticTokensLegend)
			Expect(ok).To(BeTrue())
			Expect(legend.TokenTypes).ToNot(BeEmpty())
			last := legend.TokenTypes[len(legend.TokenTypes)-1]
			Expect(string(last)).To(Equal("stringRaw"))
			Expect(uint32(len(legend.TokenTypes) - 1)).To(Equal(tokenTypeStringRaw))
		})
	})
})
