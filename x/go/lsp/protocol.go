// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package lsp provides shared LSP utilities for language servers.
package lsp

import (
	"context"

	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// Client extends protocol.Client with LSP 3.16+ methods missing from
// go.lsp.dev/protocol@v0.12.0.
type Client interface {
	protocol.Client
	SemanticTokensRefresh(ctx context.Context) error
}

// Severity converts an internal Severity to an LSP protocol DiagnosticSeverity.
func Severity(in diagnostics.Severity) protocol.DiagnosticSeverity {
	switch in {
	case diagnostics.SeverityWarning:
		return protocol.DiagnosticSeverityWarning
	case diagnostics.SeverityInfo:
		return protocol.DiagnosticSeverityInformation
	case diagnostics.SeverityHint:
		return protocol.DiagnosticSeverityHint
	case diagnostics.SeverityError:
		return protocol.DiagnosticSeverityError
	default:
		return protocol.DiagnosticSeverityError
	}
}

// TranslateConfig configures how diagnostics are translated to LSP format.
type TranslateConfig struct {
	// Source is the name shown in the LSP client's diagnostic source field
	// (e.g. "arc-analyzer", "oracle").
	Source string
}

// TranslateDiagnostics converts internal diagnostics to LSP protocol diagnostics.
// Line numbers are converted from 1-indexed (ANTLR) to 0-indexed (LSP).
func TranslateDiagnostics(
	analysisDiag diagnostics.Diagnostics,
	cfg TranslateConfig,
) []protocol.Diagnostic {
	oDiagnostics := make([]protocol.Diagnostic, 0, len(analysisDiag))
	for _, diag := range analysisDiag {
		end := diag.End
		if end.Line == 0 && end.Col == 0 {
			end.Line = diag.Start.Line
			end.Col = diag.Start.Col + 1
		}

		startLine := diag.Start.Line - 1
		if startLine < 0 {
			startLine = 0
		}
		endLine := end.Line - 1
		if endLine < 0 {
			endLine = 0
		}

		pDiag := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(startLine),
					Character: uint32(diag.Start.Col),
				},
				End: protocol.Position{
					Line:      uint32(endLine),
					Character: uint32(end.Col),
				},
			},
			Severity: Severity(diag.Severity),
			Source:   cfg.Source,
			Message:  diag.Message,
		}

		if diag.Code != "" {
			pDiag.Code = string(diag.Code)
		}

		if len(diag.Notes) > 0 {
			related := make([]protocol.DiagnosticRelatedInformation, 0, len(diag.Notes))
			for _, note := range diag.Notes {
				loc := protocol.Location{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      uint32(max(note.Start.Line-1, 0)),
							Character: uint32(note.Start.Col),
						},
						End: protocol.Position{
							Line:      uint32(max(note.Start.Line-1, 0)),
							Character: uint32(note.Start.Col + 1),
						},
					},
				}
				if note.Start.Line == 0 {
					loc.Range = pDiag.Range
				}
				related = append(related, protocol.DiagnosticRelatedInformation{
					Location: loc,
					Message:  note.Message,
				})
			}
			pDiag.RelatedInformation = related
		}

		oDiagnostics = append(oDiagnostics, pDiag)
	}
	return oDiagnostics
}

// ConvertToSemanticTokenTypes converts a string slice to protocol SemanticTokenTypes.
func ConvertToSemanticTokenTypes(types []string) []protocol.SemanticTokenTypes {
	result := make([]protocol.SemanticTokenTypes, len(types))
	for i, t := range types {
		result[i] = protocol.SemanticTokenTypes(t)
	}
	return result
}
