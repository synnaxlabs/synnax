// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FindIdentifierToken", func() {
	It("Should return the IDENTIFIER token whose text matches in a function declaration", func() {
		prog := MustSucceed(parser.Parse(`func helper(x i32) i32 { return x }`))
		tok := parser.FindIdentifierToken(prog, "helper")
		Expect(tok).ToNot(BeNil())
		sym := tok.GetSymbol()
		Expect(sym.GetText()).To(Equal("helper"))
		Expect(sym.GetTokenType()).To(Equal(parser.ArcParserIDENTIFIER))
	})

	It("Should return nil when no IDENTIFIER matches the name", func() {
		prog := MustSucceed(parser.Parse(`func helper(x i32) i32 { return x }`))
		Expect(parser.FindIdentifierToken(prog, "missing")).To(BeNil())
	})

	It("Should return nil for an empty name", func() {
		prog := MustSucceed(parser.Parse(`func helper(x i32) i32 { return x }`))
		Expect(parser.FindIdentifierToken(prog, "")).To(BeNil())
	})

	It("Should return nil for a nil node", func() {
		Expect(parser.FindIdentifierToken(nil, "anything")).To(BeNil())
	})

	It("Should resolve to the first matching identifier when the name appears multiple times", func() {
		// `x` appears as the parameter and again as the return expression.
		// The walker is depth-first, so the parameter declaration is visited
		// first.
		prog := MustSucceed(parser.Parse(`func id(x i32) i32 { return x }`))
		tok := parser.FindIdentifierToken(prog, "x")
		Expect(tok).ToNot(BeNil())
		sym := tok.GetSymbol()
		Expect(sym.GetText()).To(Equal("x"))
		// The first occurrence is on line 1 of the input (1-indexed).
		Expect(sym.GetLine()).To(Equal(1))
	})

	It("Should resolve a name nested in a sequence stage", func() {
		prog := MustSucceed(parser.Parse(`
sequence main {
	stage entry { 1 -> valve }
	stage abort { 0 -> valve }
}
`))
		tok := parser.FindIdentifierToken(prog, "abort")
		Expect(tok).ToNot(BeNil())
		Expect(tok.GetSymbol().GetText()).To(Equal("abort"))
	})

	It("Should not match keywords that share text with non-identifier tokens", func() {
		// `sequence` is a keyword and tokenized as ArcParserSEQUENCE, not
		// ArcParserIDENTIFIER, so a search for it must miss.
		prog := MustSucceed(parser.Parse(`sequence main { stage s { 1 -> valve } }`))
		Expect(parser.FindIdentifierToken(prog, "sequence")).To(BeNil())
	})
})
