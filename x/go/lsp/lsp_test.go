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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
)

var _ = Describe("LSP", func() {
	Describe("Severity", func() {
		It("Should convert error severity", func() {
			Expect(lsp.Severity(diagnostics.SeverityError)).To(Equal(protocol.DiagnosticSeverityError))
		})

		It("Should convert warning severity", func() {
			Expect(lsp.Severity(diagnostics.SeverityWarning)).To(Equal(protocol.DiagnosticSeverityWarning))
		})

		It("Should convert info severity", func() {
			Expect(lsp.Severity(diagnostics.SeverityInfo)).To(Equal(protocol.DiagnosticSeverityInformation))
		})

		It("Should convert hint severity", func() {
			Expect(lsp.Severity(diagnostics.SeverityHint)).To(Equal(protocol.DiagnosticSeverityHint))
		})
	})

	Describe("TranslateDiagnostics", func() {
		cfg := lsp.TranslateConfig{Source: "test-analyzer"}

		It("Should return empty slice for empty diagnostics", func() {
			var d diagnostics.Diagnostics
			result := lsp.TranslateDiagnostics(d, cfg)
			Expect(result).To(BeEmpty())
		})

		It("Should convert 1-indexed lines to 0-indexed", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Start:    diagnostics.Position{Line: 5, Col: 3},
				End:      diagnostics.Position{Line: 5, Col: 10},
				Severity: diagnostics.SeverityError,
				Message:  "test error",
			})
			result := lsp.TranslateDiagnostics(d, cfg)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Range.Start.Line).To(Equal(uint32(4)))
			Expect(result[0].Range.Start.Character).To(Equal(uint32(3)))
			Expect(result[0].Range.End.Line).To(Equal(uint32(4)))
			Expect(result[0].Range.End.Character).To(Equal(uint32(10)))
		})

		It("Should set source from config", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "error"))
			result := lsp.TranslateDiagnostics(d, cfg)
			Expect(result[0].Source).To(Equal("test-analyzer"))
		})

		It("Should include error code when present", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "error").WithCode(diagnostics.ErrorCodeTypeMismatch))
			result := lsp.TranslateDiagnostics(d, cfg)
			Expect(result[0].Code).To(Equal("ARC2001"))
		})

		It("Should convert notes to related information", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Start:    diagnostics.Position{Line: 1, Col: 0},
				End:      diagnostics.Position{Line: 1, Col: 5},
				Severity: diagnostics.SeverityError,
				Message:  "error",
				Notes:    []diagnostics.Note{{Message: "related note", Start: diagnostics.Position{Line: 3, Col: 2}}},
			})
			result := lsp.TranslateDiagnostics(d, cfg)
			Expect(result[0].RelatedInformation).To(HaveLen(1))
			Expect(result[0].RelatedInformation[0].Message).To(Equal("related note"))
		})

		It("Should handle zero-line diagnostics safely", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Start:    diagnostics.Position{Line: 0, Col: 0},
				Severity: diagnostics.SeverityError,
				Message:  "zero line",
			})
			result := lsp.TranslateDiagnostics(d, cfg)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Range.Start.Line).To(Equal(uint32(0)))
		})
	})

	Describe("ConvertToSemanticTokenTypes", func() {
		It("Should convert string slice to protocol types", func() {
			types := []string{"keyword", "variable", "function"}
			result := lsp.ConvertToSemanticTokenTypes(types)
			Expect(result).To(HaveLen(3))
			Expect(result[0]).To(Equal(protocol.SemanticTokenTypes("keyword")))
			Expect(result[1]).To(Equal(protocol.SemanticTokenTypes("variable")))
			Expect(result[2]).To(Equal(protocol.SemanticTokenTypes("function")))
		})

		It("Should return empty slice for empty input", func() {
			result := lsp.ConvertToSemanticTokenTypes(nil)
			Expect(result).To(BeEmpty())
		})
	})

	Describe("RWCloser", func() {
		It("Should read and write", func() {
			var buf bytes.Buffer
			rwc := lsp.RWCloser{Reader: bytes.NewReader([]byte("hello")), Writer: &buf}
			b := make([]byte, 5)
			n, err := rwc.Read(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(5))
			Expect(string(b)).To(Equal("hello"))

			n, err = rwc.Write([]byte("world"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(5))
			Expect(buf.String()).To(Equal("world"))
		})

		It("Should close without error", func() {
			rwc := lsp.RWCloser{}
			Expect(rwc.Close()).To(Succeed())
		})
	})

	Describe("EncodeSemanticTokens", func() {
		It("Should return empty slice for no tokens", func() {
			result := lsp.EncodeSemanticTokens(nil)
			Expect(result).To(BeEmpty())
		})

		It("Should encode a single token", func() {
			tokens := []lsp.Token{{Line: 0, StartChar: 5, Length: 3, TokenType: 1}}
			result := lsp.EncodeSemanticTokens(tokens)
			Expect(result).To(Equal([]uint32{0, 5, 3, 1, 0}))
		})

		It("Should delta-encode multiple tokens on the same line", func() {
			tokens := []lsp.Token{
				{Line: 0, StartChar: 0, Length: 3, TokenType: 1},
				{Line: 0, StartChar: 5, Length: 4, TokenType: 2},
			}
			result := lsp.EncodeSemanticTokens(tokens)
			Expect(result).To(Equal([]uint32{
				0, 0, 3, 1, 0,
				0, 5, 4, 2, 0,
			}))
		})

		It("Should reset character delta on new lines", func() {
			tokens := []lsp.Token{
				{Line: 0, StartChar: 5, Length: 3, TokenType: 1},
				{Line: 2, StartChar: 3, Length: 4, TokenType: 2},
			}
			result := lsp.EncodeSemanticTokens(tokens)
			Expect(result).To(Equal([]uint32{
				0, 5, 3, 1, 0,
				2, 3, 4, 2, 0,
			}))
		})
	})
})
