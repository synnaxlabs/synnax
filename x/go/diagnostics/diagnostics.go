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
)

// Severity represents the importance level of a diagnostic message.
type Severity int

//go:generate stringer -type=Severity
const (
	// Error indicates a critical issue that prevents compilation/code generation.
	Error Severity = iota
	// Warning indicates a potential problem that doesn't prevent compilation.
	Warning
	// Info provides informational messages about analysis decisions.
	Info
	// Hint suggests code improvements or best practices.
	Hint
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Info:
		return "info"
	case Hint:
		return "hint"
	default:
		return fmt.Sprintf("Severity(%d)", s)
	}
}

// Diagnostic represents a single issue found during analysis.
type Diagnostic struct {
	Key      string   `json:"key"`
	Severity Severity `json:"severity"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Message  string   `json:"message"`
	File     string   `json:"file,omitempty"`
}

// Diagnostics is a collection of diagnostic messages.
type Diagnostics []Diagnostic

var _ error = (*Diagnostics)(nil)

// Ok returns true if there are no diagnostics. This is Arc's semantics - an empty
// collection is considered "ok".
func (d Diagnostics) Ok() bool {
	return len(d) == 0
}

// HasErrors returns true if there are any error-level diagnostics.
func (d Diagnostics) HasErrors() bool {
	for _, diag := range d {
		if diag.Severity == Error {
			return true
		}
	}
	return false
}

// Empty returns true if there are no diagnostics at all.
func (d Diagnostics) Empty() bool {
	return len(d) == 0
}

// Error implements the error interface.
func (d Diagnostics) Error() string { return d.String() }

// Add adds a diagnostic to the collection.
func (d *Diagnostics) Add(diag Diagnostic) {
	*d = append(*d, diag)
}

// AddError adds an error-level diagnostic with the given message and source location.
// The file parameter is optional for backwards compatibility with Arc.
func (d *Diagnostics) AddError(err error, ctx antlr.ParserRuleContext, file ...string) {
	diag := Diagnostic{Severity: Error, Message: err.Error()}
	if len(file) > 0 {
		diag.File = file[0]
	}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// AddErrorf adds an error-level diagnostic with a formatted message.
func (d *Diagnostics) AddErrorf(
	ctx antlr.ParserRuleContext,
	file string,
	format string,
	args ...interface{},
) {
	diag := Diagnostic{
		Severity: Error,
		Message:  fmt.Sprintf(format, args...),
		File:     file,
	}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// AddWarning adds a warning-level diagnostic with the given message and source location.
// The file parameter is optional for backwards compatibility with Arc.
func (d *Diagnostics) AddWarning(err error, ctx antlr.ParserRuleContext, file ...string) {
	diag := Diagnostic{Severity: Warning, Message: err.Error()}
	if len(file) > 0 {
		diag.File = file[0]
	}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// AddWarningf adds a warning-level diagnostic with a formatted message.
func (d *Diagnostics) AddWarningf(
	ctx antlr.ParserRuleContext,
	file string,
	format string,
	args ...interface{},
) {
	diag := Diagnostic{
		Severity: Warning,
		Message:  fmt.Sprintf(format, args...),
		File:     file,
	}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// AddInfo adds an info-level diagnostic with the given message and source location.
// The file parameter is optional for backwards compatibility with Arc.
func (d *Diagnostics) AddInfo(err error, ctx antlr.ParserRuleContext, file ...string) {
	diag := Diagnostic{Severity: Info, Message: err.Error()}
	if len(file) > 0 {
		diag.File = file[0]
	}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// AddHint adds a hint-level diagnostic with the given message and source location.
// The file parameter is optional for backwards compatibility with Arc.
func (d *Diagnostics) AddHint(err error, ctx antlr.ParserRuleContext, file ...string) {
	diag := Diagnostic{Severity: Hint, Message: err.Error()}
	if len(file) > 0 {
		diag.File = file[0]
	}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// Merge adds all diagnostics from another Diagnostics collection.
func (d *Diagnostics) Merge(other Diagnostics) {
	*d = append(*d, other...)
}

// Errors returns only the error-level diagnostics.
func (d Diagnostics) Errors() Diagnostics {
	var errors Diagnostics
	for _, diag := range d {
		if diag.Severity == Error {
			errors = append(errors, diag)
		}
	}
	return errors
}

// Warnings returns only the warning-level diagnostics.
func (d Diagnostics) Warnings() Diagnostics {
	var warnings Diagnostics
	for _, diag := range d {
		if diag.Severity == Warning {
			warnings = append(warnings, diag)
		}
	}
	return warnings
}

// FromError creates a Diagnostics with a single error from an error value.
func FromError(err error) *Diagnostics {
	d := &Diagnostics{}
	d.Add(Diagnostic{Severity: Error, Message: err.Error()})
	return d
}

// String formats all diagnostics as a human-readable string.
// Format: file:line:column severity: message (when File is set)
// Format: line:column severity: message (when File is empty)
func (d Diagnostics) String() string {
	if len(d) == 0 {
		return "analysis successful"
	}
	var sb strings.Builder
	for i, diag := range d {
		if i > 0 {
			sb.WriteString("\n")
		}
		if diag.File != "" {
			sb.WriteString(fmt.Sprintf(
				"%s:%d:%d %s: %s",
				diag.File,
				diag.Line,
				diag.Column,
				diag.Severity.String(),
				diag.Message,
			))
		} else {
			sb.WriteString(fmt.Sprintf(
				"%d:%d %s: %s",
				diag.Line,
				diag.Column,
				diag.Severity.String(),
				diag.Message,
			))
		}
	}
	return sb.String()
}
