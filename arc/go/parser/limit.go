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
	// MaxSourceSize is the most bytes of source that a parse accepts.
	MaxSourceSize = 1 << 20
	// MaxDepth is the deepest that brackets, unary operators, and chains of '^' can
	// nest in parsed source.
	MaxDepth = 256
)

// limitError returns a diagnostic when source passes MaxSourceSize or its tokens pass
// MaxDepth, and nil otherwise. The parser recurses once for each level of nesting, and
// a Go stack overflow ends the process, so the check runs before the parse.
func limitError(
	source string,
	stream *antlr.CommonTokenStream,
) *diagnostics.Diagnostic {
	if len(source) > MaxSourceSize {
		return &diagnostics.Diagnostic{
			Severity: protocol.DiagnosticSeverityError,
			Message: fmt.Sprintf(
				"source is %d bytes, more than the limit of %d",
				len(source),
				MaxSourceSize,
			),
		}
	}
	stream.Fill()
	var (
		// depth counts the open brackets and the unary operators and '^' that the
		// parser is inside of. chain counts the operators since the last bracket.
		depth, chain int
		chains       []int
		operand      bool
	)
	for _, t := range stream.GetAllTokens() {
		if t.GetChannel() != antlr.TokenDefaultChannel {
			continue
		}
		switch t.GetTokenType() {
		case ArcLexerLPAREN, ArcLexerLBRACKET, ArcLexerLBRACE:
			chains = append(chains, chain)
			depth, chain, operand = depth+1, 0, false
		case ArcLexerRPAREN, ArcLexerRBRACKET, ArcLexerRBRACE:
			depth -= 1 + chain
			chain, operand = 0, true
			if n := len(chains); n > 0 {
				chain, chains = chains[n-1], chains[:n-1]
			}
		case ArcLexerCARET, ArcLexerNOT:
			depth, chain, operand = depth+1, chain+1, false
		case ArcLexerMINUS:
			if operand {
				depth, chain = depth-chain, 0
			} else {
				depth, chain = depth+1, chain+1
			}
			operand = false
		case ArcLexerIDENTIFIER, ArcLexerINTEGER_LITERAL, ArcLexerFLOAT_LITERAL,
			ArcLexerSTR_LITERAL, ArcLexerSTR_LITERAL_MULTI, ArcLexerTRUE,
			ArcLexerFALSE:
			operand = true
		case ArcLexerDOT:
			operand = false
		default:
			depth, chain, operand = depth-chain, 0, false
		}
		// An unmatched closing bracket must not hide the brackets that follow it.
		depth = max(depth, 0)
		if depth > MaxDepth {
			line, column := uint32(t.GetLine()-1), uint32(t.GetColumn())
			return &diagnostics.Diagnostic{
				Severity: protocol.DiagnosticSeverityError,
				Range: protocol.Range{
					Start: protocol.Position{Line: line, Character: column},
					End:   protocol.Position{Line: line, Character: column + 1},
				},
				Message: fmt.Sprintf(
					"nesting is deeper than the limit of %d",
					MaxDepth,
				),
			}
		}
	}
	return nil
}
