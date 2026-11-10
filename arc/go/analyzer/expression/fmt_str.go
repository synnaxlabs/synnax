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
	"unicode/utf8"

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

// AnalyzeFmtStrLiteral splits a format-string token (STR_LITERAL or STR_LITERAL_MULTI
// with f/rf prefix) into segments and analyzes its placeholders in ctx's scope. Each
// placeholder parses at its position in the source. The segments go in ctx.FmtStrings
// under strTerm, and a later call reuses them. Placeholder diagnostics anchor on the
// offending `{...}` span.
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
	segments, err := literal.FmtStrSplit(body, flags)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	sym := strTerm.GetSymbol()
	at := bodyPositions(protocol.Position{
		Line:      uint32(sym.GetLine() - 1),
		Character: uint32(sym.GetColumn() + bodyOffset(text)),
	}, body)
	stored, analyzed := ctx.FmtStrings[strTerm]
	if !analyzed {
		stored = make([]parser.FmtSegment, len(segments))
		ctx.FmtStrings[strTerm] = stored
	}
	for i, seg := range segments {
		if seg.IsPlaceholder {
			analyzePlaceholder(ctx, at, seg, &stored[i], analyzed)
			continue
		}
		if !analyzed {
			stored[i].Text = seg.Text
		}
	}
}

// bodyOffset returns the column offset from the start of a string token to the
// first byte of its body: the prefix length plus the one-character opening
// delimiter (either " or `).
func bodyOffset(text string) int {
	prefix := 0
	for prefix < 2 && prefix < len(text) && (text[prefix] == 'r' || text[prefix] == 'f') {
		prefix++
	}
	return prefix + 1
}

// analyzePlaceholder parses seg into out unless analyzed, then analyzes out's
// expression. at holds the source position of each byte of the body.
func analyzePlaceholder[T antlr.ParserRuleContext](
	ctx context.Context[T],
	at []protocol.Position,
	seg literal.FmtStrSegment,
	out *parser.FmtSegment,
	analyzed bool,
) {
	segStart, segEnd := at[seg.Start], at[seg.End]
	emit := func(d diagnostics.Diagnostic) {
		ctx.Diagnostics.Add(d.WithRange(segStart, segEnd))
	}
	if seg.Text == "" {
		emit(diagnostics.Errorf(ctx.AST, "placeholder '{}' must contain an expression"))
		return
	}
	if !analyzed {
		expr, diags := parser.ParseExpressionAt(
			seg.Text,
			textPosition(at, seg),
			ctx.Config,
		)
		if diags != nil && !diags.Ok() {
			emit(diagnostics.Errorf(ctx.AST,
				"invalid placeholder expression %q: %s", seg.Text, diags.String()))
			return
		}
		*out = parser.FmtSegment{Text: seg.Text, Spec: seg.Spec, Expr: expr}
	}
	if out.Expr == nil {
		return
	}
	errs := len(ctx.Diagnostics.Errors())
	Analyze(ctx.Child(out.Expr))
	if len(ctx.Diagnostics.Errors()) > errs || readsFailedImport(ctx, out.Expr) {
		return
	}
	t := types.InferFromExpression(ctx.Child(out.Expr)).UnwrapChan()
	if !t.IsNumeric() && t.Kind != basetypes.KindString &&
		t.Kind != basetypes.KindBool {
		emit(diagnostics.Errorf(
			ctx.AST,
			"placeholder %q has type %s; only numeric, string, and bool types are supported",
			seg.Text,
			t,
		))
		return
	}
	if seg.Spec == "" {
		return
	}
	if err := literal.FmtStrValidateSpec(seg.Spec, t); err != nil {
		emit(diagnostics.Error(err, ctx.AST))
	}
}

// textPosition maps each rune of a placeholder's text to its position in the source,
// through any escapes. at holds the position of each byte of the body.
func textPosition(at []protocol.Position, seg literal.FmtStrSegment) parser.Position {
	positions := make([]protocol.Position, 0, len(seg.Text)+1)
	for i := range seg.Text {
		positions = append(positions, at[seg.Offsets[i]])
	}
	positions = append(positions, at[seg.Offsets[len(seg.Text)]])
	return func(index int) (int, int) {
		p := positions[min(index, len(positions)-1)]
		return int(p.Line) + 1, int(p.Character)
	}
}

// bodyPositions returns the source position of each byte of body and of its end, with
// columns in runes as ANTLR counts them. base is the position of body[0].
func bodyPositions(base protocol.Position, body string) []protocol.Position {
	at := make([]protocol.Position, len(body)+1)
	p := base
	for i := 0; i < len(body); {
		r, size := utf8.DecodeRuneInString(body[i:])
		for j := range size {
			at[i+j] = p
		}
		i += size
		if r == '\n' {
			p = protocol.Position{Line: p.Line + 1}
		} else {
			p.Character++
		}
	}
	at[len(body)] = p
	return at
}

// readsFailedImport returns whether expr reads a name whose import failed. The import
// reports the error.
func readsFailedImport[T antlr.ParserRuleContext](
	ctx context.Context[T],
	expr parser.IExpressionContext,
) bool {
	for _, primary := range parser.CollectNamedPrimaries(expr) {
		head, tail := parser.PrimaryNameParts(primary)
		if _, err := ctx.ResolveQualified(head, tail); errors.Is(
			err,
			diagnostics.ErrReported,
		) {
			return true
		}
	}
	return false
}
