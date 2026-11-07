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
	"strings"
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
	segments, err := literal.FmtStrParse(body)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	sym := strTerm.GetSymbol()
	base := protocol.Position{
		Line:      uint32(sym.GetLine() - 1),
		Character: uint32(sym.GetColumn() + bodyOffset(text)),
	}
	stored, analyzed := ctx.FmtStrings[strTerm]
	if !analyzed {
		stored = make([]parser.FmtSegment, len(segments))
		ctx.FmtStrings[strTerm] = stored
	}
	for i, seg := range segments {
		if seg.IsPlaceholder {
			analyzePlaceholder(ctx, body, base, seg, &stored[i], analyzed)
			continue
		}
		if analyzed {
			continue
		}
		stored[i].Text = seg.Text
		if flags.Raw {
			continue
		}
		if stored[i].Text, err = literal.UnescapeString(
			seg.Text,
			flags.Multi,
		); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
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
// expression. base is the source position of body[0].
func analyzePlaceholder[T antlr.ParserRuleContext](
	ctx context.Context[T],
	body string,
	base protocol.Position,
	seg literal.FmtStrSegment,
	out *parser.FmtSegment,
	analyzed bool,
) {
	segStart := diagnostics.Advance(base, body, seg.Start)
	segEnd := diagnostics.Advance(base, body, seg.End)
	emit := func(d diagnostics.Diagnostic) {
		ctx.Diagnostics.Add(d.WithRange(segStart, segEnd))
	}
	if seg.Text == "" {
		emit(diagnostics.Errorf(ctx.AST, "placeholder '{}' must contain an expression"))
		return
	}
	if !analyzed {
		line, column := runePosition(base, body[:seg.Start+1])
		expr, diags := parser.ParseExpressionAt(seg.Text, line, column, ctx.Config)
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
	Analyze(ctx.Child(out.Expr))
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

// runePosition returns the 1-based line and the rune column, as ANTLR counts them, of
// the source just past prefix, where base is the source position of prefix[0].
func runePosition(base protocol.Position, prefix string) (line, column int) {
	line = int(base.Line) + 1 + strings.Count(prefix, "\n")
	if i := strings.LastIndexByte(prefix, '\n'); i >= 0 {
		return line, utf8.RuneCountInString(prefix[i+1:])
	}
	return line, int(base.Character) + utf8.RuneCountInString(prefix)
}
