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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/x/errors"
	"go.lsp.dev/protocol"
)

// ErrorCode identifies a diagnostic category. Each language defines its own
// constants (e.g. E2001 for Arc type-mismatch errors).
type ErrorCode string

// Kind is the declaration of a diagnostic.
type Kind struct {
	// Code identifies the kind.
	Code ErrorCode
	// Severity is the severity of each diagnostic of the kind.
	Severity protocol.DiagnosticSeverity
	// Message is the headline of the kind. The message of each diagnostic of the kind
	// is the headline, or the headline and a detail.
	Message string
	// Problem describes the kind for its documentation.
	Problem string
	// Example is source that reports the kind, for its documentation.
	Example string
}

// describe returns the message of a diagnostic of the kind with detail. The message
// starts with the headline.
func (k Kind) describe(detail string) string {
	head := k.Message + ": "
	if k.Message == "" || detail == k.Message || strings.HasPrefix(detail, head) {
		return detail
	}
	return head + detail
}

// At returns a diagnostic of the kind at ctx with the headline as its message.
func (k Kind) At(ctx antlr.ParserRuleContext) Diagnostic {
	return Diagnostic{
		Code:     k.Code,
		Severity: k.Severity,
		Message:  k.Message,
		Range:    rangeOf(ctx),
	}
}

// Atf returns a diagnostic of the kind at ctx. format and args give its detail.
func (k Kind) Atf(ctx antlr.ParserRuleContext, format string, args ...any) Diagnostic {
	return Diagnostic{
		Code:     k.Code,
		Severity: k.Severity,
		Message:  k.describe(fmt.Sprintf(format, args...)),
		Range:    rangeOf(ctx),
	}
}

// Wrap returns a diagnostic of the kind at ctx with the message of err as its detail.
func (k Kind) Wrap(err error, ctx antlr.ParserRuleContext) Diagnostic {
	d := Error(err, ctx)
	d.Code, d.Severity, d.Message = k.Code, k.Severity, k.describe(err.Error())
	return d
}

// kindError is an error that carries the kind of its diagnostic.
type kindError struct {
	error
	kind Kind
}

func (e *kindError) Unwrap() error { return e.error }

// Mark returns err as an error of the kind. Error gives the diagnostic of a marked
// error the code and severity of the kind, also when another error wraps it.
func (k Kind) Mark(err error) error { return &kindError{error: err, kind: k} }

// Errorf returns an error of the kind with the message of format and args.
func (k Kind) Errorf(format string, args ...any) error {
	return k.Mark(errors.Newf(format, args...))
}
