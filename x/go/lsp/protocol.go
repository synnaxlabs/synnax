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
// Positions are passed through unchanged; diagnostics store 0-indexed positions.
func TranslateDiagnostics(
	analysisDiag diagnostics.Diagnostics,
	source string,
) []protocol.Diagnostic {
	oDiagnostics := make([]protocol.Diagnostic, 0, len(analysisDiag))
	for _, diag := range analysisDiag {
		rng := diag.Range
		if rng.End == (protocol.Position{}) {
			rng.End = protocol.Position{
				Line:      rng.Start.Line,
				Character: rng.Start.Character + 1,
			}
		}

		pDiag := protocol.Diagnostic{
			Range:    rng,
			Severity: diag.Severity,
			Source:   protocol.NewOptional(source),
			Message:  protocol.String(diag.Message),
		}

		if diag.Code != "" {
			pDiag.Code = protocol.String(diag.Code)
		}

		if len(diag.Notes) > 0 {
			related := make([]protocol.DiagnosticRelatedInformation, len(diag.Notes))
			for i, note := range diag.Notes {
				// An unpositioned note points at the diagnostic's own range.
				if note.Location.Range == (protocol.Range{}) {
					note.Location.Range = pDiag.Range
				}
				related[i] = note
			}
			pDiag.RelatedInformation = related
		}

		oDiagnostics = append(oDiagnostics, pDiag)
	}
	return oDiagnostics
}
