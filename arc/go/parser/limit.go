// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

const (
	// maxSourceSize is the most bytes of source that a parse accepts.
	maxSourceSize = 1 << 20
	// maxDepth is the deepest that grammar rules can nest in a parse. Each level of
	// parentheses in an expression nests about 11 rules.
	maxDepth = 2500
)

// depthExceeded is the panic value of depthLimit. The parse entry recovers it.
type depthExceeded struct{ token antlr.Token }

// depthLimit aborts a parse whose rules nest deeper than maxDepth. The parser recurses
// once for each rule, and a Go stack overflow ends the process, so the parse must stop
// before the stack does.
type depthLimit struct {
	*antlr.BaseParseTreeListener
	depth int
}

func (l *depthLimit) EnterEveryRule(ctx antlr.ParserRuleContext) {
	if l.depth++; l.depth > maxDepth {
		panic(depthExceeded{token: ctx.GetStart()})
	}
}

func (l *depthLimit) ExitEveryRule(antlr.ParserRuleContext) { l.depth-- }

func sizeDiagnostic(source string) diagnostics.Diagnostic {
	return diagnostics.Diagnostic{
		Severity: protocol.DiagnosticSeverityError,
		Message: fmt.Sprintf(
			"source is %d bytes, more than the limit of %d",
			len(source),
			maxSourceSize,
		),
	}
}

func (d depthExceeded) diagnostic() diagnostics.Diagnostic {
	line, column := uint32(d.token.GetLine()-1), uint32(d.token.GetColumn())
	return diagnostics.Diagnostic{
		Severity: protocol.DiagnosticSeverityError,
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: column},
			End:   protocol.Position{Line: line, Character: column + 1},
		},
		Message: "nesting is deeper than the parser allows",
	}
}
