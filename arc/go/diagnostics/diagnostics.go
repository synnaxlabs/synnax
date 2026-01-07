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

//go:generate stringer -type=Severity
const (
	// Error indicates a critical issue that prevents compilation.
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
}

// Diagnostics is a collection of diagnostic messages.
type Diagnostics []Diagnostic

var _ error = (*Diagnostics)(nil)

// Ok returns true if there are no error-level diagnostics.
// Warnings, info, and hints are allowed.
func (d Diagnostics) Ok() bool {
	for _, diag := range d {
		if diag.Severity == Error {
			return false
		}
	}
	return true
}

// Error implements the error interface.
func (d Diagnostics) Error() string { return d.String() }

func (d *Diagnostics) Add(diag Diagnostic) {
	*d = append(*d, diag)
}

// AddError adds an error-level diagnostic with the given message and source location.
func (d *Diagnostics) AddError(
	err error,
	ctx antlr.ParserRuleContext,
) {
	diag := Diagnostic{Severity: Error, Message: err.Error()}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// AddWarning adds a warning-level diagnostic with the given message and source location.
func (d *Diagnostics) AddWarning(
	err error,
	ctx antlr.ParserRuleContext,
) {
	diag := Diagnostic{Severity: Warning, Message: err.Error()}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// AddInfo adds an info-level diagnostic with the given message and source location.
func (d *Diagnostics) AddInfo(
	err error,
	ctx antlr.ParserRuleContext,
) {
	diag := Diagnostic{Severity: Info, Message: err.Error()}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// AddHint adds a hint-level diagnostic with the given message and source location.
func (d *Diagnostics) AddHint(
	err error,
	ctx antlr.ParserRuleContext,
) {
	diag := Diagnostic{Severity: Hint, Message: err.Error()}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

// Errors returns only error-level diagnostics.
func (d Diagnostics) Errors() []Diagnostic {
	var errors []Diagnostic
	for _, diag := range d {
		if diag.Severity == Error {
			errors = append(errors, diag)
		}
	}
	return errors
}

// Warnings returns only warning-level diagnostics.
func (d Diagnostics) Warnings() []Diagnostic {
	var warnings []Diagnostic
	for _, diag := range d {
		if diag.Severity == Warning {
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
		sb.WriteString(fmt.Sprintf(
			"%d:%d %s: %s",
			diag.Line,
			diag.Column,
			diag.Severity.String(),
			diag.Message,
		))
	}
	return sb.String()
}
