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
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
)

var _ = Describe("LSP", func() {
	Describe("Severity", func() {
		DescribeTable("Should convert diagnostics severity to LSP severity",
			func(input diagnostics.Severity, expected protocol.DiagnosticSeverity) {
				Expect(lsp.Severity(input)).To(Equal(expected))
			},
			Entry("Error", diagnostics.Error, protocol.DiagnosticSeverityError),
			Entry("Warning", diagnostics.Warning, protocol.DiagnosticSeverityWarning),
			Entry("Info", diagnostics.Info, protocol.DiagnosticSeverityInformation),
			Entry("Hint", diagnostics.Hint, protocol.DiagnosticSeverityHint),
			Entry("Unknown defaults to error", diagnostics.Severity(99), protocol.DiagnosticSeverityError),
		)
	})

	Describe("TranslateDiagnostics", func() {
		It("Should translate empty diagnostics to empty slice", func() {
			d := diagnostics.Diagnostics{}
			cfg := lsp.TranslateConfig{Source: "test"}
			result := lsp.TranslateDiagnostics(d, cfg)
			Expect(result).To(BeEmpty())
		})

		It("Should translate diagnostics with correct line offset", func() {
			d := diagnostics.Diagnostics{
				{Severity: diagnostics.Error, Line: 10, Column: 5, Message: "error message"},
			}
			cfg := lsp.TranslateConfig{Source: "test-analyzer"}
			result := lsp.TranslateDiagnostics(d, cfg)

			Expect(result).To(HaveLen(1))
			Expect(result[0].Range.Start.Line).To(Equal(uint32(9)))
			Expect(result[0].Range.Start.Character).To(Equal(uint32(5)))
			Expect(result[0].Severity).To(Equal(protocol.DiagnosticSeverityError))
			Expect(result[0].Source).To(Equal("test-analyzer"))
			Expect(result[0].Message).To(Equal("error message"))
		})

		It("Should handle zero or negative line numbers", func() {
			d := diagnostics.Diagnostics{
				{Severity: diagnostics.Warning, Line: 0, Column: 0, Message: "at start"},
			}
			result := lsp.TranslateDiagnostics(d, lsp.TranslateConfig{})
			Expect(result[0].Range.Start.Line).To(Equal(uint32(0)))
		})

		It("Should translate multiple diagnostics", func() {
			d := diagnostics.Diagnostics{
				{Severity: diagnostics.Error, Line: 1, Column: 0, Message: "first"},
				{Severity: diagnostics.Warning, Line: 5, Column: 10, Message: "second"},
				{Severity: diagnostics.Hint, Line: 10, Column: 2, Message: "third"},
			}
			result := lsp.TranslateDiagnostics(d, lsp.TranslateConfig{Source: "src"})

			Expect(result).To(HaveLen(3))
			Expect(result[0].Severity).To(Equal(protocol.DiagnosticSeverityError))
			Expect(result[1].Severity).To(Equal(protocol.DiagnosticSeverityWarning))
			Expect(result[2].Severity).To(Equal(protocol.DiagnosticSeverityHint))
		})
	})

	Describe("RWCloser", func() {
		It("Should read from the reader", func() {
			input := "hello world"
			rw := &lsp.RWCloser{R: strings.NewReader(input), W: &bytes.Buffer{}}
			buf := make([]byte, len(input))
			n, err := rw.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(len(input)))
			Expect(string(buf)).To(Equal(input))
		})

		It("Should write to the writer", func() {
			var buf bytes.Buffer
			rw := &lsp.RWCloser{R: strings.NewReader(""), W: &buf}
			n, err := rw.Write([]byte("test output"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(11))
			Expect(buf.String()).To(Equal("test output"))
		})

		It("Should close without error", func() {
			rw := &lsp.RWCloser{R: strings.NewReader(""), W: &bytes.Buffer{}}
			Expect(rw.Close()).ToNot(HaveOccurred())
		})
	})

	Describe("ConvertToSemanticTokenTypes", func() {
		It("Should convert string slice to protocol types", func() {
			input := []string{"keyword", "type", "variable"}
			result := lsp.ConvertToSemanticTokenTypes(input)

			Expect(result).To(HaveLen(3))
			Expect(result[0]).To(Equal(protocol.SemanticTokenTypes("keyword")))
			Expect(result[1]).To(Equal(protocol.SemanticTokenTypes("type")))
			Expect(result[2]).To(Equal(protocol.SemanticTokenTypes("variable")))
		})

		It("Should return empty slice for empty input", func() {
			result := lsp.ConvertToSemanticTokenTypes([]string{})
			Expect(result).To(BeEmpty())
		})
	})

	Describe("EncodeSemanticTokens", func() {
		It("Should return empty slice for no tokens", func() {
			result := lsp.EncodeSemanticTokens([]lsp.Token{})
			Expect(result).To(Equal([]uint32{}))
		})

		It("Should encode a single token", func() {
			tokens := []lsp.Token{
				{Line: 0, StartChar: 5, Length: 3, TokenType: 1},
			}
			result := lsp.EncodeSemanticTokens(tokens)
			// deltaLine=0, deltaStartChar=5, length=3, tokenType=1, modifiers=0
			Expect(result).To(Equal([]uint32{0, 5, 3, 1, 0}))
		})

		It("Should delta-encode tokens on the same line", func() {
			tokens := []lsp.Token{
				{Line: 0, StartChar: 0, Length: 3, TokenType: 0},
				{Line: 0, StartChar: 10, Length: 5, TokenType: 1},
			}
			result := lsp.EncodeSemanticTokens(tokens)
			// First token: deltaLine=0, deltaStartChar=0, length=3, type=0, mods=0
			// Second token: deltaLine=0, deltaStartChar=10, length=5, type=1, mods=0
			Expect(result).To(Equal([]uint32{0, 0, 3, 0, 0, 0, 10, 5, 1, 0}))
		})

		It("Should delta-encode tokens on different lines", func() {
			tokens := []lsp.Token{
				{Line: 0, StartChar: 5, Length: 4, TokenType: 2},
				{Line: 3, StartChar: 2, Length: 6, TokenType: 1},
			}
			result := lsp.EncodeSemanticTokens(tokens)
			// First token: deltaLine=0, deltaStartChar=5, length=4, type=2, mods=0
			// Second token: deltaLine=3, deltaStartChar=2 (absolute when line changes), length=6, type=1, mods=0
			Expect(result).To(Equal([]uint32{0, 5, 4, 2, 0, 3, 2, 6, 1, 0}))
		})

		It("Should handle multiple tokens across lines", func() {
			tokens := []lsp.Token{
				{Line: 1, StartChar: 0, Length: 4, TokenType: 0},
				{Line: 1, StartChar: 5, Length: 3, TokenType: 1},
				{Line: 3, StartChar: 0, Length: 6, TokenType: 2},
				{Line: 3, StartChar: 8, Length: 2, TokenType: 0},
			}
			result := lsp.EncodeSemanticTokens(tokens)
			Expect(result).To(HaveLen(20))
			// First: 1, 0, 4, 0, 0
			Expect(result[0]).To(Equal(uint32(1)))
			Expect(result[1]).To(Equal(uint32(0)))
			// Second (same line): 0, 5, 3, 1, 0
			Expect(result[5]).To(Equal(uint32(0)))
			Expect(result[6]).To(Equal(uint32(5)))
			// Third (new line): 2, 0, 6, 2, 0
			Expect(result[10]).To(Equal(uint32(2)))
			Expect(result[11]).To(Equal(uint32(0)))
		})
	})
})
