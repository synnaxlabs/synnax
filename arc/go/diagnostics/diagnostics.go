// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package diagnostics provides error, warning, and hint reporting for Arc language analysis.
package diagnostics

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

// Severity represents the importance level of a diagnostic message.
type Severity int

type Position struct {
	Line int
	Col  int
}

//go:generate stringer -type=Severity
const (
	// SeverityError indicates a critical issue that prevents compilation.
	SeverityError Severity = iota
	// SeverityWarning indicates a potential problem that doesn't prevent compilation.
	SeverityWarning
	// SeverityInfo provides informational messages about analysis decisions.
	SeverityInfo
	// SeverityHint suggests code improvements or best practices.
	SeverityHint
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	case SeverityHint:
		return "hint"
	default:
		return fmt.Sprintf("Severity(%d)", s)
	}
}

// Note provides supplementary context for a diagnostic, such as where a type was inferred.
type Note struct {
	Message string   `json:"message"`
	Start   Position `json:"start,omitempty"`
}

// HintProvider is implemented by errors that include a hint for fixing the issue.
type HintProvider interface {
	GetHint() string
}

// Diagnostic represents a single compiler diagnostic message.
type Diagnostic struct {
	Key      string    `json:"key"`
	Code     ErrorCode `json:"code,omitempty"`
	Message  string    `json:"message"`
	Severity Severity  `json:"severity"`
	Start    Position  `json:"start"`
	End      Position  `json:"end"`
	Notes    []Note    `json:"notes,omitempty"`
}

// SetRange sets the Start and End positions from an ANTLR parser rule context.
func (d *Diagnostic) SetRange(ctx antlr.ParserRuleContext) {
	if ctx == nil {
		return
	}
	start := ctx.GetStart()
	stop := ctx.GetStop()
	d.Start = Position{Line: start.GetLine(), Col: start.GetColumn()}
	if stop != nil {
		d.End = Position{Line: stop.GetLine(), Col: stop.GetColumn() + len(stop.GetText())}
	} else {
		d.End = Position{Line: d.Start.Line, Col: d.Start.Col + len(start.GetText())}
	}
}

// Diagnostics is a collection of diagnostic messages.
type Diagnostics []Diagnostic

var _ error = (*Diagnostics)(nil)

// Ok returns true if there are no error-level diagnostics.
// Warnings, info, and hints are allowed.
func (d Diagnostics) Ok() bool {
	for _, diag := range d {
		if diag.Severity == SeverityError {
			return false
		}
	}
	return true
}

// Error implements the error interface.
func (d Diagnostics) Error() string { return d.String() }

func (d *Diagnostics) Add(diag Diagnostic) {
	for _, idx := range d.AtLocation(diag.Start) {
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

// AtLocation returns the indices of all diagnostics at the given position.
func (d *Diagnostics) AtLocation(start Position) []int {
	var indices []int
	for i, diag := range *d {
		if diag.Start == start {
			indices = append(indices, i)
		}
	}
	return indices
}

func (d *Diagnostics) AddError(err error, ctx antlr.ParserRuleContext) {
	diag := Diagnostic{Severity: SeverityError, Message: err.Error()}
	if ctx != nil {
		start := ctx.GetStart()
		stop := ctx.GetStop()
		diag.Start = Position{Line: start.GetLine(), Col: start.GetColumn()}
		if stop != nil {
			diag.End = Position{Line: stop.GetLine(), Col: stop.GetColumn() + len(stop.GetText())}
		} else {
			diag.End = Position{Line: diag.Start.Line, Col: diag.Start.Col + len(start.GetText())}
		}
	}
	if hp, ok := err.(HintProvider); ok {
		if hint := hp.GetHint(); hint != "" {
			diag.Notes = append(diag.Notes, Note{Message: hint})
		}
	}
	d.Add(diag)
}

func (d *Diagnostics) AddWarning(err error, ctx antlr.ParserRuleContext) {
	diag := Diagnostic{Severity: SeverityWarning, Message: err.Error()}
	if ctx != nil {
		start := ctx.GetStart()
		stop := ctx.GetStop()
		diag.Start = Position{Line: start.GetLine(), Col: start.GetColumn()}
		if stop != nil {
			diag.End = Position{Line: stop.GetLine(), Col: stop.GetColumn() + len(stop.GetText())}
		} else {
			diag.End = Position{Line: diag.Start.Line, Col: diag.Start.Col + len(start.GetText())}
		}
	}
	d.Add(diag)
}

func (d *Diagnostics) AddInfo(err error, ctx antlr.ParserRuleContext) {
	diag := Diagnostic{Severity: SeverityInfo, Message: err.Error()}
	if ctx != nil {
		start := ctx.GetStart()
		stop := ctx.GetStop()
		diag.Start = Position{Line: start.GetLine(), Col: start.GetColumn()}
		if stop != nil {
			diag.End = Position{Line: stop.GetLine(), Col: stop.GetColumn() + len(stop.GetText())}
		} else {
			diag.End = Position{Line: diag.Start.Line, Col: diag.Start.Col + len(start.GetText())}
		}
	}
	d.Add(diag)
}

func (d *Diagnostics) AddHint(err error, ctx antlr.ParserRuleContext) {
	diag := Diagnostic{Severity: SeverityHint, Message: err.Error()}
	if ctx != nil {
		start := ctx.GetStart()
		stop := ctx.GetStop()
		diag.Start = Position{Line: start.GetLine(), Col: start.GetColumn()}
		if stop != nil {
			diag.End = Position{Line: stop.GetLine(), Col: stop.GetColumn() + len(stop.GetText())}
		} else {
			diag.End = Position{Line: diag.Start.Line, Col: diag.Start.Col + len(start.GetText())}
		}
	}
	d.Add(diag)
}

// AddErrorWithCode adds an error with a specific error code.
func (d *Diagnostics) AddErrorWithCode(code ErrorCode, msg string, ctx antlr.ParserRuleContext) {
	diag := Diagnostic{Severity: SeverityError, Code: code, Message: msg}
	if ctx != nil {
		start := ctx.GetStart()
		stop := ctx.GetStop()
		diag.Start = Position{Line: start.GetLine(), Col: start.GetColumn()}
		if stop != nil {
			diag.End = Position{Line: stop.GetLine(), Col: stop.GetColumn() + len(stop.GetText())}
		} else {
			diag.End = Position{Line: diag.Start.Line, Col: diag.Start.Col + len(start.GetText())}
		}
	}
	d.Add(diag)
}

// AddErrorWithNote adds an error with an accompanying note for additional context.
func (d *Diagnostics) AddErrorWithNote(err error, ctx antlr.ParserRuleContext, note string) {
	diag := Diagnostic{Severity: SeverityError, Message: err.Error()}
	if ctx != nil {
		start := ctx.GetStart()
		stop := ctx.GetStop()
		diag.Start = Position{Line: start.GetLine(), Col: start.GetColumn()}
		if stop != nil {
			diag.End = Position{Line: stop.GetLine(), Col: stop.GetColumn() + len(stop.GetText())}
		} else {
			diag.End = Position{Line: diag.Start.Line, Col: diag.Start.Col + len(start.GetText())}
		}
	}
	if hp, ok := err.(HintProvider); ok {
		if hint := hp.GetHint(); hint != "" {
			diag.Notes = append(diag.Notes, Note{Message: hint})
		}
	}
	if note != "" {
		diag.Notes = append(diag.Notes, Note{Message: note})
	}
	d.Add(diag)
}

// AddErrorWithCodeAndNote adds an error with a code and an accompanying note.
func (d *Diagnostics) AddErrorWithCodeAndNote(
	code ErrorCode,
	msg string,
	ctx antlr.ParserRuleContext,
	note string,
) {
	diag := Diagnostic{Severity: SeverityError, Code: code, Message: msg}
	if ctx != nil {
		start := ctx.GetStart()
		stop := ctx.GetStop()
		diag.Start = Position{Line: start.GetLine(), Col: start.GetColumn()}
		if stop != nil {
			diag.End = Position{Line: stop.GetLine(), Col: stop.GetColumn() + len(stop.GetText())}
		} else {
			diag.End = Position{Line: diag.Start.Line, Col: diag.Start.Col + len(start.GetText())}
		}
	}
	if note != "" {
		diag.Notes = append(diag.Notes, Note{Message: note})
	}
	d.Add(diag)
}

// Errors returns only error-level diagnostics.
func (d Diagnostics) Errors() []Diagnostic {
	var errors []Diagnostic
	for _, diag := range d {
		if diag.Severity == SeverityError {
			errors = append(errors, diag)
		}
	}
	return errors
}

// Warnings returns only warning-level diagnostics.
func (d Diagnostics) Warnings() []Diagnostic {
	var warnings []Diagnostic
	for _, diag := range d {
		if diag.Severity == SeverityWarning {
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
		if diag.Code != "" {
			sb.WriteString(fmt.Sprintf(
				"%d:%d %s [%s]: %s",
				diag.Start.Line,
				diag.Start.Col,
				diag.Severity.String(),
				diag.Code,
				diag.Message,
			))
		} else {
			sb.WriteString(fmt.Sprintf(
				"%d:%d %s: %s",
				diag.Start.Line,
				diag.Start.Col,
				diag.Severity.String(),
				diag.Message,
			))
		}
		for _, note := range diag.Notes {
			sb.WriteString("\n")
			if note.Start.Line > 0 {
				sb.WriteString(fmt.Sprintf("  %d:%d note: %s", note.Start.Line, note.Start.Col, note.Message))
			} else {
				sb.WriteString(fmt.Sprintf("  note: %s", note.Message))
			}
		}
	}
	return sb.String()
}
