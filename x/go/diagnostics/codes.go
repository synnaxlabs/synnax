// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/antlr4-go/antlr/v4"
	"go.lsp.dev/protocol"
)

// ErrorCode identifies a diagnostic category. Each language defines its own
// constants (e.g. ARC2001 for Arc type-mismatch errors).
type ErrorCode string

// Kind is the declaration of a diagnostic.
type Kind struct {
	// Code identifies the kind.
	Code ErrorCode
	// Severity is the severity of each diagnostic of the kind.
	Severity protocol.DiagnosticSeverity
	// Message is the message format. A kind that reports with Wrap has none.
	Message string
}

// At returns a diagnostic of the kind at ctx. args fill the verbs of Message.
func (k Kind) At(ctx antlr.ParserRuleContext, args ...any) Diagnostic {
	d := Diagnostic{
		Code:     k.Code,
		Severity: k.Severity,
		Message:  fmt.Sprintf(k.Message, args...),
	}
	d.SetRange(ctx)
	return d
}

// Wrap returns a diagnostic of the kind at ctx with the message of err.
func (k Kind) Wrap(err error, ctx antlr.ParserRuleContext) Diagnostic {
	d := Error(err, ctx)
	d.Code, d.Severity = k.Code, k.Severity
	return d
}
