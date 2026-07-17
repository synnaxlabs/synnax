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
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// TranslateDiagnostics converts internal diagnostics to LSP protocol diagnostics,
// tagging each with source as its LSP diagnostic source (e.g. "arc-analyzer").
// Line numbers are converted from 1-indexed (ANTLR) to 0-indexed (LSP).
func TranslateDiagnostics(
	analysisDiag diagnostics.Diagnostics,
	source string,
) []protocol.Diagnostic {
	oDiagnostics := make([]protocol.Diagnostic, 0, len(analysisDiag))
	for _, diag := range analysisDiag {
		end := diag.End
		if end.Line == 0 && end.Col == 0 {
			end.Line = diag.Start.Line
			end.Col = diag.Start.Col + 1
		}

		startLine := max(diag.Start.Line-1, 0)
		endLine := max(end.Line-1, 0)

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
			Severity: diag.Severity,
			Source:   protocol.NewOptional(source),
			Message:  protocol.String(diag.Message),
		}

		if diag.Code != "" {
			pDiag.Code = protocol.String(diag.Code)
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
