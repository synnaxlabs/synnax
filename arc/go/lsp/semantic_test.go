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
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/protocol"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// Token type ids must mirror the iota constants in arc/go/lsp/semantic.go.
// Tests in this file pin the legend ordering and the token-type routing
// to those ids.
const (
	tokenTypeKeyword           = uint32(0)
	tokenTypeOperator          = uint32(2)
	tokenTypeVariable          = uint32(3)
	tokenTypeString            = uint32(4)
	tokenTypeNumber            = uint32(5)
	tokenTypeFunction          = uint32(7)
	tokenTypeChannel           = uint32(9)
	tokenTypeStatefulVariable  = uint32(13)
	tokenTypeNamespace         = uint32(19)
	tokenTypeStringRaw         = uint32(20)
	tokenTypeStringPlaceholder = uint32(21)
	tokenTypeChannelVariable   = uint32(22)
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
			OpenArcDocument(server, ctx, uri, "x := \"hello\"")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeString)
			Expect(tokens).To(HaveLen(1))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(7)))
		})

		It("emits one token for an empty raw literal", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := \"\"")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeString)
			Expect(tokens).To(HaveLen(1))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(2)))
		})

		It("splits a multi-line literal with one mid-newline into two tokens", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `a\nb`")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeString)
			Expect(tokens).To(HaveLen(2))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(2)))
			Expect(tokens[1].Line).To(Equal(uint32(1)))
			Expect(tokens[1].StartChar).To(Equal(uint32(0)))
			Expect(tokens[1].Length).To(Equal(uint32(2)))
		})

		It("splits a three-line multi-line literal into three tokens", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `a\nb\nc`")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeString)
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
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeString)
			Expect(tokens).To(HaveLen(2))
			Expect(tokens[0].Line).To(Equal(uint32(0)))
			Expect(tokens[0].StartChar).To(Equal(uint32(5)))
			Expect(tokens[0].Length).To(Equal(uint32(4)))
			Expect(tokens[1].Line).To(Equal(uint32(1)))
			Expect(tokens[1].StartChar).To(Equal(uint32(0)))
			Expect(tokens[1].Length).To(Equal(uint32(1)))
		})

		It("skips empty lines in a multi-line literal with consecutive newlines", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := `a\n\nb`")
			tokens := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeString)
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
		It("routes single-quoted literals to the string token type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := "hi"`)
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			str := filterByType(tokens, tokenTypeString)
			Expect(str).To(HaveLen(1))
			Expect(str[0].Length).To(Equal(uint32(4)))
		})

		It("routes raw-prefixed literals to the string token type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := r"hi"`)
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			str := filterByType(tokens, tokenTypeString)
			Expect(str).To(HaveLen(1))
			Expect(str[0].Length).To(Equal(uint32(4)))
		})

		It("does not emit a string token when no string literal is present", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := 42")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(tokens, tokenTypeString)).To(BeEmpty())
		})

		It("routes raw triple-quoted literals to the string token type across newlines", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := r\"\"\"a\nb\"\"\"")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			str := filterByType(tokens, tokenTypeString)
			Expect(str).To(HaveLen(2))
			Expect(str[0].Line).To(Equal(uint32(0)))
			Expect(str[1].Line).To(Equal(uint32(1)))
		})
	})

	Describe("Numeric-literal unit suffixes", func() {
		// "3min" lexes as INTEGER_LITERAL "3" + IDENTIFIER "min"; the parser binds
		// the identifier to the literal as a unit suffix. Some unit names collide
		// with builtins (here the "min" function), so the suffix must not resolve
		// as a symbol — otherwise it is colored unlike units such as "s" or "h".
		// The suffix sits at line 1, char 7; the real call sits at line 2, char 8.
		const src = "func f() i64 {\n\tx := 3min\n\treturn min(1, 2)\n}"

		It("does not classify a unit suffix as the builtin it collides with", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, src)
			fn := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeFunction)
			Expect(fn).ToNot(ContainElement(
				decodedToken{Line: 1, StartChar: 7, Length: 3, TokenType: tokenTypeFunction},
			))
		})

		It("still classifies a real call to that builtin as a function", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, src)
			fn := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeFunction)
			Expect(fn).To(ContainElement(
				decodedToken{Line: 2, StartChar: 8, Length: 3, TokenType: tokenTypeFunction},
			))
		})
	})

	Describe("Format-string placeholders", func() {
		It("splits f\"val: {42}\" into prefix, string segments, placeholder braces, and a number", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := f"val: {42}"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			var inLit []decodedToken
			for _, t := range all {
				if t.Line == 0 && t.StartChar >= 5 {
					inLit = append(inLit, t)
				}
			}
			Expect(inLit).To(HaveLen(7))
			Expect(inLit[0]).To(Equal(decodedToken{Line: 0, StartChar: 5, Length: 1, TokenType: tokenTypeFunction}))
			Expect(inLit[1]).To(Equal(decodedToken{Line: 0, StartChar: 6, Length: 1, TokenType: tokenTypeString}))
			Expect(inLit[2]).To(Equal(decodedToken{Line: 0, StartChar: 7, Length: 5, TokenType: tokenTypeString}))
			Expect(inLit[3]).To(Equal(decodedToken{Line: 0, StartChar: 12, Length: 1, TokenType: tokenTypeStringPlaceholder}))
			Expect(inLit[4]).To(Equal(decodedToken{Line: 0, StartChar: 13, Length: 2, TokenType: tokenTypeNumber}))
			Expect(inLit[5]).To(Equal(decodedToken{Line: 0, StartChar: 15, Length: 1, TokenType: tokenTypeStringPlaceholder}))
			Expect(inLit[6]).To(Equal(decodedToken{Line: 0, StartChar: 16, Length: 1, TokenType: tokenTypeString}))
		})

		It("classifies a placeholder identifier through the global resolver", func(ctx SpecContext) {
			channels := []symbol.Symbol{
				{
					Name: "sensorData",
					Type: types.Chan(types.F64()),
					Kind: symbol.KindChannel,
				},
			}
			server, uri = SetupTestServer(lsp.Config{
				NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
			})
			OpenArcDocument(server, ctx, uri, `x := f"v: {sensorData}"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			ch := filterByType(all, tokenTypeChannel)
			Expect(ch).To(HaveLen(1))
			Expect(ch[0]).To(Equal(decodedToken{Line: 0, StartChar: 11, Length: 10, TokenType: tokenTypeChannel}))
		})

		It("treats {{ as a literal-brace escape and leaves bare } literal", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := f"a {{ b }} c"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(BeEmpty())
			str := filterByType(all, tokenTypeString)
			Expect(str).ToNot(BeEmpty())
		})

		It("recognizes a real placeholder while ignoring surrounding doubled braces", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := f"{{ {42} }}"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(HaveLen(2))
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
		})

		It("falls back to prefix + string tokens on a malformed placeholder", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := f"unterminated {x"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			str := filterByType(all, tokenTypeString)
			Expect(str).To(HaveLen(1))
			Expect(str[0]).To(Equal(decodedToken{Line: 0, StartChar: 6, Length: 17, TokenType: tokenTypeString}))
			fn := filterByType(all, tokenTypeFunction)
			Expect(fn).To(HaveLen(1))
			Expect(fn[0]).To(Equal(decodedToken{Line: 0, StartChar: 5, Length: 1, TokenType: tokenTypeFunction}))
			for _, op := range filterByType(all, tokenTypeOperator) {
				Expect(op.StartChar).To(BeNumerically("<", 5))
			}
		})

		It("emits a placeholder span for a numeric format spec after the expression", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := f"v={42:05d}"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			ph := filterByType(all, tokenTypeStringPlaceholder)
			Expect(ph).To(HaveLen(3))
			Expect(ph[0]).To(Equal(decodedToken{Line: 0, StartChar: 9, Length: 1, TokenType: tokenTypeStringPlaceholder}))
			Expect(ph[1]).To(Equal(decodedToken{Line: 0, StartChar: 12, Length: 4, TokenType: tokenTypeStringPlaceholder}))
			Expect(ph[2]).To(Equal(decodedToken{Line: 0, StartChar: 16, Length: 1, TokenType: tokenTypeStringPlaceholder}))
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
		})

		It("classifies multi-token placeholder expressions with prev/next context", func(ctx SpecContext) {
			channels := []symbol.Symbol{
				{
					Name: "sensor",
					Type: types.Chan(types.F64()),
					Kind: symbol.KindChannel,
				},
			}
			server, uri = SetupTestServer(lsp.Config{
				NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
			})
			OpenArcDocument(server, ctx, uri, `x := f"v={sensor + 1}"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeChannel)).To(HaveLen(1))
			plus := filterByType(all, tokenTypeOperator)
			Expect(plus).ToNot(BeEmpty())
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
		})

		It("skips inner placeholder tokens that classify to nil (parens)", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := f"v={(42)}"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(HaveLen(2))
		})

		It("classifies placeholders across newlines in a multi-line format string", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := f`a={1}\nb={2}`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(2))
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(HaveLen(4))
		})

		It("classifies placeholders inside an rf-prefixed format string", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := rf"v={42}"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(1))
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(HaveLen(2))
		})

		It("classifies placeholders inside an rf-prefixed multi-line format string", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "x := rf`a={1}\nb={2}`")
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeNumber)).To(HaveLen(2))
			Expect(filterByType(all, tokenTypeStringPlaceholder)).To(HaveLen(4))
		})

		DescribeTable("emits the r/f/rf/fr prefix as a function-typed token",
			func(ctx SpecContext, source string, prefixLen uint32) {
				OpenArcDocument(server, ctx, uri, source)
				all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
				fn := filterByType(all, tokenTypeFunction)
				Expect(fn).To(HaveLen(1))
				Expect(fn[0]).To(Equal(decodedToken{Line: 0, StartChar: 5, Length: prefixLen, TokenType: tokenTypeFunction}))
			},
			Entry("f-prefixed single-quoted", `x := f"hi {x}"`, uint32(1)),
			Entry("r-prefixed single-quoted", `x := r"path"`, uint32(1)),
			Entry("rf-prefixed single-quoted", `x := rf"hi {x}"`, uint32(2)),
			Entry("fr-prefixed single-quoted", `x := fr"hi {x}"`, uint32(2)),
			Entry("f-prefixed backtick", "x := f`hi`", uint32(1)),
			Entry("r-prefixed backtick", "x := r`hi`", uint32(1)),
			Entry("rf-prefixed backtick", "x := rf`hi {x}`", uint32(2)),
		)

		It("does not emit a function token for an unprefixed string", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `x := "plain"`)
			all := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(all, tokenTypeFunction)).To(BeEmpty())
		})
	})

	Describe("SemanticTokensFull", func() {
		It("returns an empty token stream for an unknown document URI", func(ctx SpecContext) {
			result := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///not-open.arc"},
			}))
			Expect(result.Data).To(BeEmpty())
		})

		It("routes module names in a bare import to the namespace token type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "import time\n")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			ns := filterByType(tokens, tokenTypeNamespace)
			Expect(ns).To(HaveLen(1))
			Expect(ns[0].Length).To(Equal(uint32(4))) // "time"
		})

		It("routes every module name in a block import to namespace", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "import (\n    time\n    math\n)\n")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			ns := filterByType(tokens, tokenTypeNamespace)
			Expect(ns).To(HaveLen(2))
		})

		It("routes both the module name and its alias to namespace", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "import time as t\n")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			ns := filterByType(tokens, tokenTypeNamespace)
			Expect(ns).To(HaveLen(2)) // "time" and "t"
		})

		It("routes import qualifiers at use sites to the namespace token type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri,
				"import time\n\nfunc cat() i64 { return time.now() }\n")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			ns := filterByType(tokens, tokenTypeNamespace)
			// One for the import declaration, one for the qualifier at the call site.
			Expect(ns).To(HaveLen(2))
			for _, t := range ns {
				Expect(t.Length).To(Equal(uint32(4)))
			}
		})

		It("does not route an unimported qualifier to namespace", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri,
				"func cat() i64 { return time.now() }\n")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			Expect(filterByType(tokens, tokenTypeNamespace)).To(BeEmpty())
		})

		It("does not route an unimported qualifier's member to function", func(ctx SpecContext) {
			// `now` at col 29 must not be function-colored: `time` is not
			// imported, so the analyzer treats `time.now` as undefined and the
			// highlight must match rather than imply a valid call.
			OpenArcDocument(server, ctx, uri,
				"func cat() i64 { return time.now() }\n")
			fns := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeFunction)
			for _, t := range fns {
				Expect(t.StartChar).ToNot(Equal(uint32(29)),
					"unimported member `now` must not be colored as a function")
			}
		})

		It("routes an imported qualifier's member to function", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri,
				"import time\n\nfunc cat() i64 { return time.now() }\n")
			fns := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeFunction)
			Expect(fns).To(ContainElement(
				decodedToken{Line: 2, StartChar: 29, Length: 3, TokenType: tokenTypeFunction},
			))
		})

		It("routes sequence and stage names to the function token type", func(ctx SpecContext) {
			// Sequence and stage names should share the same highlight
			// color as function names — they are declarations of named,
			// callable scopes.
			OpenArcDocument(server, ctx, uri, "sequence main {\n    stage first {\n    }\n}")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			var mainTok, firstTok *decodedToken
			for i := range tokens {
				t := tokens[i]
				if t.Line == 0 && t.StartChar == 9 && t.Length == 4 {
					mainTok = &t
				}
				if t.Line == 1 && t.StartChar == 10 && t.Length == 5 {
					firstTok = &t
				}
			}
			Expect(mainTok).ToNot(BeNil(), "sequence name token not emitted")
			Expect(firstTok).ToNot(BeNil(), "stage name token not emitted")
			Expect(mainTok.TokenType).To(Equal(tokenTypeFunction),
				"sequence name must be the function token type")
			Expect(firstTok.TokenType).To(Equal(tokenTypeFunction),
				"stage name must be the function token type")
		})

		It("routes import and as alongside func and authority to the keyword token type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, `import time as t

authority 255

func cat() {
    t.now()
}
`)
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			lengths := make([]uint32, 0)
			for _, tok := range filterByType(tokens, tokenTypeKeyword) {
				lengths = append(lengths, tok.Length)
			}
			Expect(lengths).To(ConsistOf(
				uint32(6), // import
				uint32(2), // as
				uint32(9), // authority
				uint32(4), // func
			))
		})
	})

	Describe("variable kinds", func() {
		It("classifies literal, stateful, channel read/write, and channel-read variables distinctly", func(ctx SpecContext) {
			channels := []symbol.Symbol{
				{
					Name: "sensorData",
					Type: types.Chan(types.F64()),
					Kind: symbol.KindChannel,
				},
			}
			server, uri = SetupTestServer(lsp.Config{
				NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
			})
			OpenArcDocument(server, ctx, uri, "stage s {\ncount := 0\ntotal $= 0\ncpu := sensorData\nrate := sensorData + 1.0\n}\n")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)

			value := filterByType(tokens, tokenTypeVariable)
			Expect(value).To(HaveLen(1))
			Expect(value[0].Line).To(Equal(uint32(1)))

			stateful := filterByType(tokens, tokenTypeStatefulVariable)
			Expect(stateful).To(HaveLen(1))
			Expect(stateful[0].Line).To(Equal(uint32(2)))

			channelVariable := filterByType(tokens, tokenTypeChannelVariable)
			Expect(channelVariable).To(HaveLen(2))
			Expect(channelVariable).To(ContainElements(
				decodedToken{Line: 3, StartChar: 0, Length: 3, TokenType: tokenTypeChannelVariable},
				decodedToken{Line: 4, StartChar: 0, Length: 4, TokenType: tokenTypeChannelVariable},
			))
		})

		It("colors the declaration of a reassigned channel read/write variable", func(ctx SpecContext) {
			channels := []symbol.Symbol{
				{Name: "crw_a", Type: types.Chan(types.F64()), Kind: symbol.KindChannel},
				{Name: "crw_b", Type: types.Chan(types.F64()), Kind: symbol.KindChannel},
				{Name: "out", Type: types.Chan(types.F64()), Kind: symbol.KindChannel},
			}
			server, uri = SetupTestServer(lsp.Config{
				NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
			})
			OpenArcDocument(server, ctx, uri, "sequence main {\n\tal := crw_a\n\tal -> out\n\tal = crw_b\n}")
			channelVariable := filterByType(decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data), tokenTypeChannelVariable)
			Expect(channelVariable).To(ContainElement(decodedToken{
				Line: 1, StartChar: 1, Length: 2, TokenType: tokenTypeChannelVariable,
			}))
			Expect(channelVariable).To(HaveLen(3))
		})

		It("colors a constant reference with the constant token type", func(ctx SpecContext) {
			constants := []symbol.Symbol{
				{Name: "MAX", Type: types.I64(), Kind: symbol.KindConstant},
			}
			server, uri = SetupTestServer(lsp.Config{
				NewRoot: func() *symbol.Symbol { return NewRoot(nil, constants...) },
			})
			OpenArcDocument(server, ctx, uri, "x := MAX\n")
			tokens := decodeSemanticTokens(SemanticTokens(server, ctx, uri).Data)
			constant := filterByType(tokens, uint32(lsp.SemanticTokenTypeConstant))
			Expect(constant).To(ContainElement(decodedToken{
				Line: 0, StartChar: 5, Length: 3,
				TokenType: uint32(lsp.SemanticTokenTypeConstant),
			}))
		})
	})

	Describe("Legend", func() {
		It("pins the tail of the semantic token types legend", func(ctx SpecContext) {
			result := MustSucceed(server.Initialize(ctx, &protocol.InitializeParams{
				ClientInfo: &protocol.ClientInfo{Name: "test"},
			}))
			provider, ok := result.Capabilities.SemanticTokensProvider.(map[string]any)
			Expect(ok).To(BeTrue())
			legend, ok := provider["legend"].(protocol.SemanticTokensLegend)
			Expect(ok).To(BeTrue())
			Expect(legend.TokenTypes).ToNot(BeEmpty())
			n := len(legend.TokenTypes)
			Expect(string(legend.TokenTypes[n-1])).To(Equal("channelVariable"))
			Expect(uint32(n - 1)).To(Equal(tokenTypeChannelVariable))
		})
	})
})
