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
)

var _ = Describe("ChannelText breaker", func() {
	never := func(string) bool { return false }

	DescribeTable("Should write one quoted token that holds the name",
		func(name string) {
			text := parser.ChannelText(name, never)
			tokens := parser.NewLexer(text, parser.Config{}).GetAllTokens()
			Expect(tokens).To(HaveLen(1), "text %q", text)
			Expect(tokens[0].GetText()).To(Equal(text))
			Expect(tokens[0].GetTokenType()).To(Equal(parser.ArcLexerCHANNEL_NAME))
			Expect(parser.DecodeChannelName(tokens[0].GetText())).To(Equal(name))
		},
		Entry("a newline", "a\nb"),
		Entry("a carriage return", "a\rb"),
	)
})
