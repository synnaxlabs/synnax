// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package diagnostics

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

type Severity int

//go:generate stringer -type=Severity
const (
	Error Severity = iota
	Warning
	Info
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

// Diagnostic represents a semantic analysis issue
type Diagnostic struct {
	Key      string   `json:"key"`
	Severity Severity `json:"severity"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Message  string   `json:"message"`
}

var _ error = (*Diagnostics)(nil)

type Diagnostics []Diagnostic

func (d Diagnostics) Ok() bool {
	return len(d) == 0
}

func (d Diagnostics) Error() string {
	return d.String()
}

func (d *Diagnostics) Add(diag Diagnostic) {
	*d = append(*d, diag)
}

func (d *Diagnostics) AddError(
	err error,
	ctx antlr.ParserRuleContext,
) {
	diag := Diagnostic{
		Severity: Error,
		Message:  err.Error(),
	}
	if ctx != nil {
		diag.Line = ctx.GetStart().GetLine()
		diag.Column = ctx.GetStart().GetColumn()
	}
	*d = append(*d, diag)
}

func (d Diagnostics) String() string {
	if len(d) == 0 {
		return "analysis successful"
	}
	var sb strings.Builder
	for i, diag := range d {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("%d:%d %s: %s", diag.Line, diag.Column, diag.Severity.String(), diag.Message))
	}
	return sb.String()
}
