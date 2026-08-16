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
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
)

var _ = Describe("LSP", func() {
	Describe("TranslateDiagnostics", func() {
		const source = "test-analyzer"

		It("Should copy the diagnostic severity through unchanged", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  "m",
			})
			Expect(lsp.TranslateDiagnostics(d, source)[0].Severity).
				To(Equal(protocol.DiagnosticSeverityWarning))
		})

		It("Should return empty slice for empty diagnostics", func() {
			var d diagnostics.Diagnostics
			result := lsp.TranslateDiagnostics(d, source)
			Expect(result).To(BeEmpty())
		})

		It("Should copy the already-0-indexed range straight through", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 3},
					End:   protocol.Position{Line: 5, Character: 10},
				},
				Severity: protocol.DiagnosticSeverityError,
				Message:  "test error",
			})
			result := lsp.TranslateDiagnostics(d, source)
			Expect(result).To(HaveLen(1))
			Expect(
				result[0].Range.Start,
			).To(Equal(protocol.Position{Line: 5, Character: 3}))
			Expect(
				result[0].Range.End,
			).To(Equal(protocol.Position{Line: 5, Character: 10}))
		})

		It("Should set source from config", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "error"))
			result := lsp.TranslateDiagnostics(d, source)
			Expect(result[0].Source).To(Equal(protocol.NewOptional("test-analyzer")))
		})

		It("Should include error code when present", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "error").WithCode("TEST001"))
			result := lsp.TranslateDiagnostics(d, source)
			Expect(result[0].Code).To(Equal(protocol.String("TEST001")))
		})

		It("Should convert notes to related information", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 0},
					End:   protocol.Position{Line: 1, Character: 5},
				},
				Severity: protocol.DiagnosticSeverityError,
				Message:  "error",
				Notes: []protocol.DiagnosticRelatedInformation{{
					Message: "related note",
					Location: protocol.Location{Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 2},
					}},
				}},
			})
			result := lsp.TranslateDiagnostics(d, source)
			Expect(result[0].RelatedInformation).To(HaveLen(1))
			Expect(result[0].RelatedInformation[0].Message).To(Equal("related note"))
		})

		It("Should handle zero-line diagnostics safely", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
				},
				Severity: protocol.DiagnosticSeverityError,
				Message:  "zero line",
			})
			result := lsp.TranslateDiagnostics(d, source)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Range.Start.Line).To(Equal(uint32(0)))
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
