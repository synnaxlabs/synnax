// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package diagnostics provides error, warning, and hint reporting for language analysis.
package diagnostics

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"go.lsp.dev/protocol"
)

// Advance returns the position reached by walking off bytes of body from pos, resetting
// the character on each newline. Positions are 0-indexed, matching protocol.Position.
func Advance(pos protocol.Position, body string, off int) protocol.Position {
	for i := 0; i < off && i < len(body); i++ {
		if body[i] == '\n' {
			pos.Line++
			pos.Character = 0
		} else {
			pos.Character++
		}
	}
	return pos
}

func severityLabel(s protocol.DiagnosticSeverity) string {
	switch s {
	case protocol.DiagnosticSeverityError:
		return "error"
	case protocol.DiagnosticSeverityWarning:
		return "warning"
	case protocol.DiagnosticSeverityInformation:
		return "info"
	case protocol.DiagnosticSeverityHint:
		return "hint"
	default:
		return fmt.Sprintf("severity(%d)", s)
	}
}

// HintProvider is implemented by errors that include a hint for fixing the issue.
type HintProvider interface {
	GetHint() string
}

// Diagnostic represents a single compiler diagnostic message.
type Diagnostic struct {
	Code     ErrorCode                               `json:"code,omitempty"`
	Message  string                                  `json:"message"`
	Severity protocol.DiagnosticSeverity             `json:"severity"`
	Range    protocol.Range                          `json:"range"`
	Notes    []protocol.DiagnosticRelatedInformation `json:"notes,omitempty"`
}

// SetRange sets the Range from an ANTLR parser rule context.
func (d *Diagnostic) SetRange(ctx antlr.ParserRuleContext) {
	if ctx == nil {
		return
	}
	start := ctx.GetStart()
	stop := ctx.GetStop()
	// ANTLR lines are 1-indexed; store 0-indexed to match protocol.Position.
	d.Range.Start = protocol.Position{
		Line:      uint32(start.GetLine() - 1),
		Character: uint32(start.GetColumn()),
	}
	if stop != nil {
		d.Range.End = protocol.Position{
			Line:      uint32(stop.GetLine() - 1),
			Character: uint32(stop.GetColumn() + len(stop.GetText())),
		}
	} else {
		d.Range.End = protocol.Position{
			Line:      d.Range.Start.Line,
			Character: d.Range.Start.Character + uint32(len(start.GetText())),
		}
	}
}

// Error creates an error diagnostic from an existing error. If the error implements
// HintProvider, the hint is automatically extracted and added as a note.
func Error(err error, ctx antlr.ParserRuleContext) Diagnostic {
	d := Diagnostic{Severity: protocol.DiagnosticSeverityError, Message: err.Error()}
	d.SetRange(ctx)
	if hp, ok := err.(HintProvider); ok {
		if hint := hp.GetHint(); hint != "" {
			d.Notes = append(d.Notes, protocol.DiagnosticRelatedInformation{Message: hint})
		}
	}
	return d
}

// Errorf creates an error diagnostic with a formatted message.
func Errorf(ctx antlr.ParserRuleContext, format string, args ...any) Diagnostic {
	d := Diagnostic{Severity: protocol.DiagnosticSeverityError, Message: fmt.Sprintf(format, args...)}
	d.SetRange(ctx)
	return d
}

// Warningf creates a warning diagnostic with a formatted message.
func Warningf(ctx antlr.ParserRuleContext, format string, args ...any) Diagnostic {
	d := Diagnostic{Severity: protocol.DiagnosticSeverityWarning, Message: fmt.Sprintf(format, args...)}
	d.SetRange(ctx)
	return d
}

// Infof creates an info diagnostic with a formatted message.
func Infof(ctx antlr.ParserRuleContext, format string, args ...any) Diagnostic {
	d := Diagnostic{Severity: protocol.DiagnosticSeverityInformation, Message: fmt.Sprintf(format, args...)}
	d.SetRange(ctx)
	return d
}

// Hintf creates a hint diagnostic with a formatted message.
func Hintf(ctx antlr.ParserRuleContext, format string, args ...any) Diagnostic {
	d := Diagnostic{Severity: protocol.DiagnosticSeverityHint, Message: fmt.Sprintf(format, args...)}
	d.SetRange(ctx)
	return d
}

// WithCode returns a copy of the diagnostic with the given error code.
func (d Diagnostic) WithCode(code ErrorCode) Diagnostic {
	d.Code = code
	return d
}

// WithRange returns a copy of the diagnostic with an explicit range, overriding
// any range set by SetRange.
func (d Diagnostic) WithRange(start, end protocol.Position) Diagnostic {
	d.Range = protocol.Range{Start: start, End: end}
	return d
}

// WithNote returns a copy of the diagnostic with an additional note.
func (d Diagnostic) WithNote(note string) Diagnostic {
	if note != "" {
		d.Notes = append(d.Notes, protocol.DiagnosticRelatedInformation{Message: note})
	}
	return d
}

// WithNoteAt returns a copy of the diagnostic with an additional note at the given position.
func (d Diagnostic) WithNoteAt(note string, pos protocol.Position) Diagnostic {
	if note != "" {
		d.Notes = append(d.Notes, protocol.DiagnosticRelatedInformation{
			Message: note,
			Location: protocol.Location{Range: protocol.Range{
				Start: pos,
				End:   protocol.Position{Line: pos.Line, Character: pos.Character + 1},
			}},
		})
	}
	return d
}

// Diagnostics is a collection of diagnostic messages.
type Diagnostics []Diagnostic

var _ error = (*Diagnostics)(nil)

// Ok returns true if there are no error-level diagnostics.
// Warnings, info, and hints are allowed.
func (d Diagnostics) Ok() bool {
	for _, diag := range d {
		if diag.Severity == protocol.DiagnosticSeverityError {
			return false
		}
	}
	return true
}

// Empty returns true if there are no diagnostics at all.
func (d Diagnostics) Empty() bool { return len(d) == 0 }

// Error implements the error interface.
func (d Diagnostics) Error() string { return d.String() }

func (d *Diagnostics) Add(diag Diagnostic) {
	for _, idx := range d.AtLocation(diag.Range.Start) {
		existing := (*d)[idx]
		if existing.Message == diag.Message {
			if diag.Severity < existing.Severity {
				(*d)[idx] = diag
			}
			return
		}
	}
	*d = append(*d, diag)
}

// Merge appends all diagnostics from other into d, using Add to preserve
// deduplication semantics.
func (d *Diagnostics) Merge(other Diagnostics) {
	for _, diag := range other {
		d.Add(diag)
	}
}

// AtLocation returns the indices of all diagnostics at the given position.
func (d *Diagnostics) AtLocation(start protocol.Position) []int {
	var indices []int
	for i, diag := range *d {
		if diag.Range.Start == start {
			indices = append(indices, i)
		}
	}
	return indices
}

// Errors returns only error-level diagnostics.
func (d Diagnostics) Errors() []Diagnostic {
	var errors []Diagnostic
	for _, diag := range d {
		if diag.Severity == protocol.DiagnosticSeverityError {
			errors = append(errors, diag)
		}
	}
	return errors
}

// Warnings returns only warning-level diagnostics.
func (d Diagnostics) Warnings() []Diagnostic {
	var warnings []Diagnostic
	for _, diag := range d {
		if diag.Severity == protocol.DiagnosticSeverityWarning {
			warnings = append(warnings, diag)
		}
	}
	return warnings
}

// String formats all diagnostics as a human-readable string with line:column severity: message format.
func (d Diagnostics) String() string {
	if len(d) == 0 {
		return "analysis successful"
	}
	var sb strings.Builder
	for i, diag := range d {
		if i > 0 {
			sb.WriteString("\n")
		}
		// Positions are stored 0-indexed; display lines 1-indexed for humans.
		if diag.Code != "" {
			_, _ = fmt.Fprintf(&sb,
				"%d:%d %s [%s]: %s",
				diag.Range.Start.Line+1,
				diag.Range.Start.Character,
				severityLabel(diag.Severity),
				diag.Code,
				diag.Message,
			)
		} else {
			_, _ = fmt.Fprintf(&sb,
				"%d:%d %s: %s",
				diag.Range.Start.Line+1,
				diag.Range.Start.Character,
				severityLabel(diag.Severity),
				diag.Message,
			)
		}
		for _, note := range diag.Notes {
			sb.WriteString("\n")
			start := note.Location.Range.Start
			if start != (protocol.Position{}) {
				_, _ = fmt.Fprintf(&sb, "  %d:%d note: %s", start.Line+1, start.Character, note.Message)
			} else {
				_, _ = fmt.Fprintf(&sb, "  note: %s", note.Message)
			}
		}
	}
	return sb.String()
}
