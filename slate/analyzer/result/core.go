// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package result

import "github.com/antlr4-go/antlr/v4"

// Diagnostic represents a semantic analysis issue
type Diagnostic struct {
	Severity DiagnosticSeverity
	Line     int
	Column   int
	Message  string
}

type DiagnosticSeverity int

const (
	SeverityError DiagnosticSeverity = iota
	SeverityWarning
	SeverityInfo
	SeverityHint
)

type Result struct {
	Diagnostics []Diagnostic
}

func (r *Result) AddError(
	err error,
	ctx antlr.ParserRuleContext,
) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{
		Severity: SeverityError,
		Line:     ctx.GetStart().GetLine(),
		Column:   ctx.GetStart().GetColumn(),
		Message:  err.Error(),
	})
}
