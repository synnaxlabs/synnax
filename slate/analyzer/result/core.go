// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package result

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/slate/symbol"
)

// Diagnostic represents a semantic analysis issue
type Diagnostic struct {
	Severity Severity
	Line     int
	Column   int
	Message  string
}

type Severity int

//go:generate stringer -type=Severity
const (
	Error Severity = iota
	Warning
	Info
	Hint
)

type Result struct {
	Diagnostics []Diagnostic
	Symbols     *symbol.Scope
}

func (r *Result) AddError(
	err error,
	ctx antlr.ParserRuleContext,
) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{
		Severity: Error,
		Line:     ctx.GetStart().GetLine(),
		Column:   ctx.GetStart().GetColumn(),
		Message:  err.Error(),
	})
}

func (r *Result) Ok() bool {
	return len(r.Diagnostics) == 0
}

func (r *Result) String() string {
	if len(r.Diagnostics) == 0 {
		return "analysis successful"
	}
	var sb strings.Builder
	for i, diag := range r.Diagnostics {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("%d:%d %s: %s", diag.Line, diag.Column, diag.Severity, diag.Message))
	}
	return sb.String()
}
