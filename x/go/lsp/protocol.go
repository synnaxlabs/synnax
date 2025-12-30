// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"io"

	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// Severity converts a diagnostics.Severity to the corresponding LSP protocol severity.
func Severity(s diagnostics.Severity) protocol.DiagnosticSeverity {
	switch s {
	case diagnostics.Error:
		return protocol.DiagnosticSeverityError
	case diagnostics.Warning:
		return protocol.DiagnosticSeverityWarning
	case diagnostics.Info:
		return protocol.DiagnosticSeverityInformation
	case diagnostics.Hint:
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityError
	}
}

// TranslateConfig configures how diagnostics are translated to LSP format.
type TranslateConfig struct {
	// Source is the source name to include in the LSP diagnostic (e.g., "arc-analyzer").
	Source string
}

// TranslateDiagnostics converts internal Diagnostics to LSP protocol Diagnostics.
func TranslateDiagnostics(diags diagnostics.Diagnostics, cfg TranslateConfig) []protocol.Diagnostic {
	result := make([]protocol.Diagnostic, 0, len(diags))
	for _, d := range diags {
		// Lines are 1-indexed in diagnostics, but 0-indexed in LSP
		line := uint32(d.Line - 1)
		if d.Line <= 0 {
			line = 0
		}
		result = append(result, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      line,
					Character: uint32(d.Column),
				},
				End: protocol.Position{
					Line:      line,
					Character: uint32(d.Column + 10),
				},
			},
			Severity: Severity(d.Severity),
			Source:   cfg.Source,
			Message:  d.Message,
		})
	}
	return result
}

// RWCloser wraps an io.Reader and io.Writer to implement io.ReadWriteCloser.
// Useful for creating LSP streams from stdin/stdout.
type RWCloser struct {
	R io.Reader
	W io.Writer
}

func (rw *RWCloser) Read(p []byte) (int, error)  { return rw.R.Read(p) }
func (rw *RWCloser) Write(p []byte) (int, error) { return rw.W.Write(p) }
func (rw *RWCloser) Close() error                { return nil }

// ConvertToSemanticTokenTypes converts a slice of string token type names to LSP protocol types.
func ConvertToSemanticTokenTypes(types []string) []protocol.SemanticTokenTypes {
	result := make([]protocol.SemanticTokenTypes, len(types))
	for i, t := range types {
		result[i] = protocol.SemanticTokenTypes(t)
	}
	return result
}
