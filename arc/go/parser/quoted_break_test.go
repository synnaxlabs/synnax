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
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Quoted channel names in the parser (break)", func() {
	tokensOf := func(src string) []antlr.Token {
		l := parser.NewLexer(src, parser.Config{})
		var out []antlr.Token
		for _, t := range l.GetAllTokens() {
			if t.GetChannel() == antlr.TokenDefaultChannel {
				out = append(out, t)
			}
		}
		return out
	}
	DescribeTable("P lexes without a CHANNEL_NAME",
		func(src string) {
			for _, t := range tokensOf(src) {
				Expect(
					t.GetTokenType(),
				).ToNot(Equal(parser.ArcLexerCHANNEL_NAME), t.GetText())
			}
			MustSucceed(parser.Parse(src))
		},
		Entry("P1 double-quoted", `x := "it's"`),
		Entry("P2 raw", `x := r"it's"`),
		Entry("P3 f-string", `x := f"it's {1}"`),
		Entry("P4 multi", "x := `it's\n'a'`"),
		Entry("P5 line comment", "x := 1 // it's"),
		Entry("P6 block comment", "x := 1 /* it's\n 'a */"),
		Entry("P7 dashed off", `x := "a'b'c"`),
	)
	DescribeTable("P fails to parse",
		func(src string) {
			_, diag := parser.Parse(src)
			Expect(diag.String()).To(ContainSubstring("syntax error"))
		},
		Entry("P8 tab escape", `x := 'a\tb'`),
		Entry("P9 unicode escape", "x := 'a\\"+"u00e9'"),
		Entry("P10 CR", "x := 'a\rb'"),
		Entry("P11 trailing backslash", `x := 'a\'`),
		Entry("P12 index target", `'x'[0] = 1`),
		Entry("P13 declaration", `'x' := 1`),
		Entry("P14 function name", `func 'f'() {}`),
		Entry("P15 param", `func f() {'x' f64} {}`),
		Entry("P16 stateful", `'x' $= 1`),
		Entry("P17 sequence name", `sequence 'main' {}`),
		Entry("P18 stage name", `sequence main { stage 'a' {} }`),
	)
	It("P19 decodes an escaped quote at the edges", func() {
		primary := parser.GetPrimaryExpression(
			MustSucceed(parser.ParseExpression(`'\'a\''`)),
		)
		Expect(
			parser.PrimaryName(primary),
		).To(Equal(parser.Name{Head: "'a'", Quoted: true}))
	})
	It("P20 a quoted name that is empty", func() {
		Expect(parser.ParseExpression(`''`)).Error().To(MatchError(ContainSubstring(
			"1:0 error [E1000]: syntax error: token recognition error at: ''''",
		)))
	})
	It("P21 a quoted name with a tail is never split", func() {
		primary := parser.GetPrimaryExpression(
			MustSucceed(parser.ParseExpression(`'a.b'`)),
		)
		Expect(
			parser.PrimaryName(primary),
		).To(Equal(parser.Name{Head: "a.b", Quoted: true}))
	})
	It("P22b decomposed text against composed text", func() {
		decomposed := "A" + string(rune(0x308))
		composed := string(rune(0xc4))
		primary := parser.GetPrimaryExpression(
			MustSucceed(parser.ParseExpression("'" + decomposed + "'")),
		)
		Expect(parser.PrimaryName(primary).Head).To(Equal(composed))
	})
	It("P22 decomposed text", func() {
		primary := parser.GetPrimaryExpression(
			MustSucceed(parser.ParseExpression("'Ä'")),
		)
		Expect(parser.PrimaryName(primary).Head).To(Equal("Ä"))
	})
	It("P24 a quoted channel in an authority entry", func() {
		MustSucceed(parser.Parse("authority ('valve-cmd' 200)\n'pt' -> 'valve-cmd'"))
	})
	It("P25 a dashed channel in an authority entry with dashed names on", func() {
		MustSucceed(parser.Parse(
			"authority (valve-cmd 200)\npt -> valve-cmd",
			parser.Config{AllowDashedNames: true},
		))
	})
	It("P23 a quoted call in a flow", func() {
		MustSucceed(parser.Parse(`trig -> 'f'(1) -> out`))
	})
})
