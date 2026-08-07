// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"go.lsp.dev/protocol"
)

// AnalyzeFmtStrLiteral parses a format-string token (STR_LITERAL or
// STR_LITERAL_MULTI with f/rf prefix) and analyzes its placeholders. Body
// offsets map to source bytes so per-placeholder diagnostics anchor on the
// offending span.
func AnalyzeFmtStrLiteral[T antlr.ParserRuleContext](
	ctx context.Context[T],
	strTerm antlr.TerminalNode,
) {
	text := strTerm.GetText()
	body, flags, ok := literal.StripQuotes(text)
	if !ok {
		ctx.Diagnostics.Add(diagnostics.Error(
			errors.Newf("invalid string literal: %s", text), ctx.AST,
		))
		return
	}
	if !flags.Format {
		return
	}
	sym := strTerm.GetSymbol()
	bodyOff := bodyOffset(text, flags)
	base := protocol.Position{
		Line:      uint32(sym.GetLine() - 1),
		Character: uint32(sym.GetColumn() + bodyOff),
	}
	AnalyzeFmtStrSegments(ctx, body, base, ctx.AST)
}

// bodyOffset returns the column offset from the start of a string token to the
// first byte of its body: the prefix length plus the one-character opening
// delimiter (either " or `).
func bodyOffset(text string, _ literal.StringFlags) int {
	prefix := 0
	for prefix < 2 && prefix < len(text) && (text[prefix] == 'r' || text[prefix] == 'f') {
		prefix++
	}
	return prefix + 1
}

// AnalyzeFmtStrSegments parses body and analyzes each placeholder expression
// in ctx's scope. base is the source position of body[0]; placeholder
// diagnostics anchor on the offending `{...}` span. Returns parsed segments,
// or nil if body is malformed.
func AnalyzeFmtStrSegments[T antlr.ParserRuleContext](
	ctx context.Context[T],
	body string,
	base protocol.Position,
	anchor antlr.ParserRuleContext,
) []literal.FmtStrSegment {
	segments, err := literal.FmtStrParse(body)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, anchor))
		return nil
	}
	for _, seg := range segments {
		if !seg.IsPlaceholder {
			continue
		}
		segStart := diagnostics.Advance(base, body, seg.Start)
		segEnd := diagnostics.Advance(base, body, seg.End)
		emit := func(d diagnostics.Diagnostic) {
			ctx.Diagnostics.Add(d.WithRange(segStart, segEnd))
		}
		if seg.Text == "" {
			emit(diagnostics.Errorf(anchor,
				"placeholder '{}' must contain an expression"))
			continue
		}
		expr, diags := parser.ParseExpression(seg.Text, ctx.Config)
		if diags != nil && !diags.Ok() {
			emit(diagnostics.Errorf(anchor,
				"invalid placeholder expression %q: %s", seg.Text, diags.String()))
			continue
		}
		Analyze(context.Child(ctx, expr))
		t := types.InferFromExpression(context.Child(ctx, expr)).UnwrapChan()
		if !t.IsNumeric() && t.Kind != basetypes.KindString {
			emit(diagnostics.Errorf(
				anchor,
				"placeholder %q has type %s; only numeric and string types are supported",
				seg.Text,
				t,
			))
			continue
		}
		if seg.Spec == "" {
			continue
		}
		if err := literal.FmtStrValidateSpec(seg.Spec, t); err != nil {
			emit(diagnostics.Error(err, anchor))
		}
	}
	return segments
}
