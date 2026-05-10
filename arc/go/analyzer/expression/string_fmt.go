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
	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/parser"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
)

// AnalyzeStringFmtLiteral parses a STR_LITERAL_RAW token and analyzes its
// placeholders. Bypasses literal.ParseRawString so body offsets map to source
// bytes for per-placeholder diagnostic anchoring.
func AnalyzeStringFmtLiteral[T antlr.ParserRuleContext](
	ctx context.Context[T],
	rawStr antlr.TerminalNode,
) {
	body, ok := fmtstring.StripDelimiters(rawStr.GetText())
	if !ok {
		ctx.Diagnostics.Add(diagnostics.Error(
			errors.Newf("invalid raw string literal: %s", rawStr.GetText()), ctx.AST,
		))
		return
	}
	sym := rawStr.GetSymbol()
	base := diagnostics.Position{Line: sym.GetLine(), Col: sym.GetColumn() + 1}
	AnalyzeStringFmtSegments(ctx, body, base, ctx.AST)
}

// AnalyzeStringFmtSegments parses body and analyzes each placeholder expression
// in ctx's scope. base is the source position of body[0]; placeholder
// diagnostics anchor on the offending `{...}` span. Returns parsed segments,
// or nil if body is malformed.
func AnalyzeStringFmtSegments[T antlr.ParserRuleContext](
	ctx context.Context[T],
	body string,
	base diagnostics.Position,
	anchor antlr.ParserRuleContext,
) []fmtstring.Segment {
	segments, err := fmtstring.Parse(body)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, anchor))
		return nil
	}
	for _, seg := range segments {
		if !seg.IsPlaceholder {
			continue
		}
		segStart := base.Advance(body, seg.Start)
		segEnd := base.Advance(body, seg.End)
		emit := func(d diagnostics.Diagnostic) {
			ctx.Diagnostics.Add(d.WithRange(segStart, segEnd))
		}
		expr, diags := parser.ParseExpression(seg.Text)
		if diags != nil && !diags.Ok() {
			emit(diagnostics.Errorf(anchor,
				"invalid placeholder expression %q: %s", seg.Text, diags.String()))
			continue
		}
		Analyze(context.Child(ctx, expr))
		t := types.InferFromExpression(context.Child(ctx, expr)).UnwrapChan()
		if !t.IsNumeric() && t.Kind != basetypes.KindString {
			emit(diagnostics.Errorf(anchor,
				"placeholder %q has type %s; only numeric and string types are supported",
				seg.Text, t))
			continue
		}
		if seg.Spec == "" {
			continue
		}
		if err := fmtstring.ValidateSpec(seg.Spec, t); err != nil {
			emit(diagnostics.Error(err, anchor))
		}
	}
	return segments
}
