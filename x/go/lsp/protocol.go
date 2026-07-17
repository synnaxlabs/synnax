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
		if end == (protocol.Position{}) {
			end = protocol.Position{Line: diag.Start.Line, Character: diag.Start.Character + 1}
		}

		pDiag := protocol.Diagnostic{
			Range:    protocol.Range{Start: diag.Start, End: end},
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
				loc := protocol.Location{Range: protocol.Range{
					Start: note.Start,
					End:   protocol.Position{Line: note.Start.Line, Character: note.Start.Character + 1},
				}}
				// An unpositioned note points at the diagnostic's own range.
				if note.Start == (protocol.Position{}) {
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
