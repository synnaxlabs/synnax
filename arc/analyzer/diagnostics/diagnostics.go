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
	"github.com/synnaxlabs/x/errors"
)

type Severity int

//go:generate stringer -type=Severity
const (
	Error Severity = iota
	Warning
	Info
	Hint
)

// Diagnostic represents a semantic analysis issue
type Diagnostic struct {
	Key      string   `json:"key"`
	Severity Severity `json:"severity"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Message  string   `json:"message"`
}

type Diagnostics []Diagnostic

func (d Diagnostics) Ok() bool {
	return len(d) == 0
}

func (d Diagnostics) Error() error {
	return errors.Newf(d.String())
}

func (d *Diagnostics) AddError(
	err error,
	ctx antlr.ParserRuleContext,
) {
	*d = append(*d, Diagnostic{
		Severity: Error,
		Line:     ctx.GetStart().GetLine(),
		Column:   ctx.GetStart().GetColumn(),
		Message:  err.Error(),
	})
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
		sb.WriteString(fmt.Sprintf("%d:%d %s: %s", diag.Line, diag.Column, diag.Severity, diag.Message))
	}
	return sb.String()
}
