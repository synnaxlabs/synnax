// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check

import (
	"context"
	"time"

	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// AnalyzeGate surfaces every analyzer diagnostic the pipeline collected.
// Errors fail the gate. Warnings, info, and hints are reported but only
// fail the gate when WarningsAsErrors is set.
//
// The previous `oracle check` implementation silently dropped warnings.
// That made it possible for "unresolved type" or other soft analyzer
// signals to land on disk and survive review. This gate forces them to
// the surface.
type AnalyzeGate struct {
	// WarningsAsErrors promotes SeverityWarning findings to SeverityError
	// for the purpose of this gate's pass/fail decision.
	WarningsAsErrors bool
}

// NewAnalyzeGate constructs an analyze gate.
func NewAnalyzeGate(warningsAsErrors bool) *AnalyzeGate {
	return &AnalyzeGate{WarningsAsErrors: warningsAsErrors}
}

func (AnalyzeGate) Name() string { return "analyze" }

func (g AnalyzeGate) Run(_ context.Context, p *pipeline.Result, _ Env) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	if p.Diagnostics != nil {
		p.Diagnostics.Each(func(file string, d diagnostics.Diagnostic) {
			// Positions are 0-indexed; render 1-indexed, with 0 meaning no location.
			var line int
			if d.Range.Start != (protocol.Position{}) {
				line = int(d.Range.Start.Line) + 1
			}
			f := Finding{
				Path:     file,
				Line:     line,
				Col:      int(d.Range.Start.Character),
				Message:  d.Message,
				Severity: severityFromDiagnostic(d.Severity, g.WarningsAsErrors),
				FixHint:  hintFromNotes(d.Notes),
			}
			r.Findings = append(r.Findings, f)
			if f.Severity == SeverityError {
				r.Status = StatusFail
			}
		})
	}
	r.Elapsed = time.Since(start)
	return r
}

func severityFromDiagnostic(s protocol.DiagnosticSeverity, warningsAsErrors bool) Severity {
	switch s {
	case protocol.DiagnosticSeverityError:
		return SeverityError
	case protocol.DiagnosticSeverityWarning:
		if warningsAsErrors {
			return SeverityError
		}
		return SeverityWarning
	default:
		return SeverityInfo
	}
}

func hintFromNotes(notes []protocol.DiagnosticRelatedInformation) string {
	if len(notes) == 0 {
		return ""
	}
	return notes[0].Message
}
